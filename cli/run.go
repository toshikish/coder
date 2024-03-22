package cli

import (
	"fmt"
	"math/rand"
	"time"

	"golang.org/x/xerrors"

	"github.com/coder/coder/v2/cli/cliui"
	"github.com/coder/coder/v2/codersdk"
	"github.com/coder/serpent"
)

func randomRunName() string {
	t := time.Now()
	return fmt.Sprintf("run-%02d%02d%02d-%d", t.Year()%100, t.Month(), t.Day(), rand.Intn(100000))
}

func (r *RootCmd) run() *serpent.Command {
	var (
		client         = new(codersdk.Client)
		parameterFlags workspaceParameterFlags
	)
	return &serpent.Command{
		Use:   "run <template> -- <command>",
		Short: "Run a command using an ephemeral workspace",
		// Hidden because this command is not well tested nor does
		// it have a stable interface or known future.
		Hidden: true,
		Long: `Starts a new workspace using the specified template and runs the given command in it.
Once the command completes, any desired artifacts are copied out of the workspace and it is destroyed.`,
		Middleware: serpent.Chain(r.InitClient(client)),
		Handler: func(inv *serpent.Invocation) error {
			organization, err := CurrentOrganization(r, inv, client)
			if err != nil {
				return err
			}
			if len(inv.Args) == 0 {
				return xerrors.New("missing template name")
			}

			templateName := inv.Args[0]

			template, err := client.TemplateByName(inv.Context(), organization.ID, templateName)
			if err != nil {
				return xerrors.Errorf("get template by name: %w", err)
			}

			var (
				ctx               = inv.Context()
				templateVersionID = template.ActiveVersionID
				workspaceName     = randomRunName()
			)

			cliBuildParameters, err := asWorkspaceBuildParameters(parameterFlags.richParameters)
			if err != nil {
				return xerrors.Errorf("can't parse given parameter values: %w", err)
			}

			richParameters, err := prepWorkspaceBuild(inv, client, prepWorkspaceBuildArgs{
				Action:            WorkspaceCreate,
				TemplateVersionID: templateVersionID,
				NewWorkspaceName:  workspaceName,

				RichParameterFile: parameterFlags.richParameterFile,
				RichParameters:    cliBuildParameters,
			})
			if err != nil {
				return xerrors.Errorf("prepare build: %w", err)
			}

			workspace, err := client.CreateWorkspace(ctx, organization.ID, codersdk.Me,
				codersdk.CreateWorkspaceRequest{
					TemplateVersionID:   templateVersionID,
					Name:                workspaceName,
					RichParameterValues: richParameters,
				},
			)
			if err != nil {
				return xerrors.Errorf("create workspace: %w", err)
			}

			err = cliui.WorkspaceBuild(inv.Context(), inv.Stdout, client, workspace.LatestBuild.ID)
			if err != nil {
				return xerrors.Errorf("watch build: %w", err)
			}

			defer func() {
				var state []byte
				_, err := client.CreateWorkspaceBuild(inv.Context(), workspace.ID, codersdk.CreateWorkspaceBuildRequest{
					Transition:       codersdk.WorkspaceTransitionDelete,
					ProvisionerState: state,
				})
				if err != nil {
					cliui.Errorf(inv.Stderr, "failed to delete workspace: %v", err)
				}
			}()

			err = cliui.WorkspaceBuild(inv.Context(), inv.Stdout, client, workspace.LatestBuild.ID)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(
				inv.Stdout,
				"\n%s has been deleted at %s!\n", cliui.Keyword(workspace.FullName()),
				cliui.Timestamp(time.Now()),
			)

			return nil
		},
		Options: append(
			serpent.OptionSet{},
			parameterFlags.allOptions()...,
		),
	}
}
