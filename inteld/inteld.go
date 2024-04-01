package inteld

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/elastic/go-sysinfo"
	"github.com/hashicorp/yamux"
	"github.com/spf13/afero"
	"golang.org/x/xerrors"

	"cdr.dev/slog"
	"github.com/coder/coder/v2/codersdk"
	"github.com/coder/coder/v2/inteld/pathman"
	"github.com/coder/coder/v2/inteld/proto"
	"github.com/coder/retry"
)

type Dialer func(ctx context.Context) (proto.DRPCIntelDaemonClient, error)

type Options struct {
	// Dialer connects the daemon to a client.
	Dialer Dialer

	Filesystem afero.Fs

	// InvokeBinary is the path to the binary that will be
	// associated with aliased commands.
	InvokeBinary string

	// InvokeDirectory is the directory where binaries are aliased
	// to and overridden in the $PATH so they can be man-in-the-middled.
	InvokeDirectory string

	Logger slog.Logger
}

func New(opts Options) *API {
	closeContext, closeCancel := context.WithCancel(context.Background())
	api := &API{
		clientDialer:    opts.Dialer,
		clientChan:      make(chan proto.DRPCIntelDaemonClient),
		closeContext:    closeContext,
		closeCancel:     closeCancel,
		filesystem:      opts.Filesystem,
		logger:          opts.Logger,
		invokeDirectory: opts.InvokeDirectory,
		invokeBinary:    opts.InvokeBinary,
	}
	api.closeWaitGroup.Add(2)
	go api.connectLoop()
	go api.registerLoop()
	return api
}

type API struct {
	filesystem      afero.Fs
	invokeBinary    string
	invokeDirectory string

	clientDialer   Dialer
	clientChan     chan proto.DRPCIntelDaemonClient
	closeContext   context.Context
	closeCancel    context.CancelFunc
	closed         bool
	closeMutex     sync.Mutex
	closeWaitGroup sync.WaitGroup
	logger         slog.Logger
}

func (a *API) registerLoop() {
	defer a.logger.Debug(a.closeContext, "system loop exited")
	defer a.closeWaitGroup.Done()
	for {
		client, ok := a.client()
		if !ok {
			a.logger.Debug(a.closeContext, "shut down before client (re) connected")
			return
		}
		err := pathman.Prepend(a.closeContext, a.filesystem, a.invokeDirectory)
		if err != nil {
			a.logger.Error(a.closeContext, "unable to prepend invoke directory to PATH", slog.Error(err))
		}
		userEmail, err := fetchFromGitConfig("user.email")
		if err != nil {
			a.logger.Warn(a.closeContext, "unable to fetch user.email from git config", slog.Error(err))
		}
		userName, err := fetchFromGitConfig("user.name")
		if err != nil {
			a.logger.Warn(a.closeContext, "unable to fetch user.name from git config", slog.Error(err))
		}
		var (
			machineID   string
			hostname    string
			osVersion   string
			memoryTotal uint64
		)
		sysInfoHost, err := sysinfo.Host()
		if err == nil {
			info := sysInfoHost.Info()
			machineID = info.UniqueID
			osVersion = info.OS.Version
			hostname = info.Hostname
			mem, err := sysInfoHost.Memory()
			if err == nil {
				memoryTotal = mem.Total
			}
		} else {
			a.logger.Warn(a.closeContext, "unable to fetch machine information", slog.Error(err))
		}
		system, err := client.Register(a.closeContext, &proto.RegisterRequest{
			MachineId:       machineID,
			Hostname:        hostname,
			OperatingSystem: runtime.GOOS,
			Architecture:    runtime.GOARCH,
			// TODO: This could be nil
			OperatingSystemVersion: osVersion,
			CpuCores:               uint32(runtime.NumCPU()),
			MemoryTotal:            memoryTotal,
			GitConfigEmail:         userEmail,
			GitConfigName:          userName,
		})
		if err != nil {
			if errors.Is(err, context.Canceled) ||
				errors.Is(err, yamux.ErrSessionShutdown) {
				continue
			}
		}
		a.systemLoop(system)
	}
}

func (a *API) systemLoop(client proto.DRPCIntelDaemon_RegisterClient) {
	ctx := a.closeContext
	for {
		resp, err := client.Recv()
		if err != nil {
			if errors.Is(err, context.Canceled) ||
				errors.Is(err, yamux.ErrSessionShutdown) {
				return
			}

			a.logger.Warn(ctx, "unable to receive a message", slog.Error(err))
			return
		}

		switch m := resp.Message.(type) {
		case *proto.SystemResponse_TrackExecutables:
			err = a.trackExecutables(m.TrackExecutables.BinaryName)
			if err != nil {
				// TODO: send an error back to the server
				a.logger.Warn(ctx, "unable to track executables", slog.Error(err))
			}
			a.logger.Info(ctx, "tracked executables", slog.F("binary_names", m.TrackExecutables.BinaryName))
		}
	}
}

// trackExecutables creates symlinks in the invoke directory for the
// given binary names.
func (a *API) trackExecutables(binaryNames []string) error {
	// Clear out any existing symlinks so we're only tracking the
	// executables we're told to track.
	files, err := afero.ReadDir(a.filesystem, a.invokeDirectory)
	if errors.Is(err, os.ErrNotExist) {
		err = a.filesystem.MkdirAll(a.invokeDirectory, 0755)
		if err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}
	for _, file := range files {
		// Clear out the directory to remove old filenames.
		// Don't do this for the global dir because it makes
		// debugging harder.
		err = a.filesystem.Remove(filepath.Join(a.invokeDirectory, file.Name()))
		if err != nil {
			return err
		}
	}
	err = a.filesystem.MkdirAll(a.invokeDirectory, 0755)
	if err != nil {
		return err
	}
	linker, ok := a.filesystem.(afero.Linker)
	if !ok {
		return xerrors.New("filesystem does not support symlinks")
	}
	for _, binaryName := range binaryNames {
		err = linker.SymlinkIfPossible(a.invokeBinary, filepath.Join(a.invokeDirectory, binaryName))
		if err != nil {
			return err
		}
	}
	return nil
}

// connectLoop starts a loop that attempts to connect to coderd.
func (a *API) connectLoop() {
	defer a.logger.Debug(a.closeContext, "connect loop exited")
	defer a.closeWaitGroup.Done()
connectLoop:
	for retrier := retry.New(50*time.Millisecond, 10*time.Second); retrier.Wait(a.closeContext); {
		a.logger.Debug(a.closeContext, "dialing coderd")
		client, err := a.clientDialer(a.closeContext)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			var sdkErr *codersdk.Error
			// If something is wrong with our auth, stop trying to connect.
			if errors.As(err, &sdkErr) && sdkErr.StatusCode() == http.StatusForbidden {
				a.logger.Error(a.closeContext, "not authorized to dial coderd", slog.Error(err))
				return
			}
			if a.isClosed() {
				return
			}
			a.logger.Warn(a.closeContext, "coderd client failed to dial", slog.Error(err))
			continue
		}
		a.logger.Info(a.closeContext, "successfully connected to coderd")
		retrier.Reset()

		// serve the client until we are closed or it disconnects
		for {
			select {
			case <-a.closeContext.Done():
				client.DRPCConn().Close()
				return
			case <-client.DRPCConn().Closed():
				a.logger.Info(a.closeContext, "connection to coderd closed")
				continue connectLoop
			case a.clientChan <- client:
				continue
			}
		}
	}
}

// client returns the current client or nil if the API is closed
func (a *API) client() (proto.DRPCIntelDaemonClient, bool) {
	select {
	case <-a.closeContext.Done():
		return nil, false
	case client := <-a.clientChan:
		return client, true
	}
}

// isClosed returns whether the API is closed or not.
func (a *API) isClosed() bool {
	select {
	case <-a.closeContext.Done():
		return true
	default:
		return false
	}
}

func (a *API) Close() error {
	a.closeMutex.Lock()
	defer a.closeMutex.Unlock()
	if a.closed {
		return nil
	}
	a.closed = true
	a.closeCancel()
	a.closeWaitGroup.Wait()
	return nil
}

// fetchFromGitConfig returns the value of a property from the git config.
// If the property is not found, it returns an empty string.
// If git is not installed, it returns an empty string.
func fetchFromGitConfig(property string) (string, error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return "", nil
	}
	cmd := exec.Command(gitPath, "config", "--get", property)
	output, err := cmd.Output()
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(string(output)), nil
}
