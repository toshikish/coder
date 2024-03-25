package spice_test

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/moby/moby/pkg/namesgenerator"
	"github.com/stretchr/testify/require"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest"
	"github.com/coder/coder/v2/coderd/database"
	"github.com/coder/coder/v2/coderd/database/dbauthz"
	"github.com/coder/coder/v2/coderd/database/dbgen"
	"github.com/coder/coder/v2/coderd/database/dbmem"
	"github.com/coder/coder/v2/coderd/database/dbtestutil"
	"github.com/coder/coder/v2/coderd/database/dbtime"
	"github.com/coder/coder/v2/coderd/database/spice"
	"github.com/coder/coder/v2/coderd/database/spice/policy"
	"github.com/coder/coder/v2/coderd/rbac"
	"github.com/coder/coder/v2/testutil"
)

func TestInNestedTX(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t, testutil.WaitLong)
	db, _ := dbtestutil.NewDB(t)
	s, err := spice.New(ctx, &spice.SpiceServerOpts{
		PostgresURI: "",
		Logger:      slog.Logger{},
		Store:       db,
	})
	require.NoError(t, err, "must not error")

	u := dbgen.User(t, db, database.User{})
	actor := rbac.Subject{
		ID:     u.ID.String(),
		Roles:  rbac.RoleNames{rbac.RoleOwner()},
		Groups: []string{},
		Scope:  rbac.ScopeAll,
	}

	ctx = dbauthz.As(ctx, actor)
	// A double nested tx
	err = s.InTx(func(tx database.Store) error {
		return tx.InTx(func(tx database.Store) error {
			_, err := tx.GetUserByID(ctx, u.ID)
			return err
		}, nil)
	}, nil)
	require.NoError(t, err, "must not error")
}

func TestExampleDatabaseLayer(t *testing.T) {
	t.Parallel()

	logger := slogtest.Make(t, &slogtest.Options{
		IgnoreErrors:   false,
		SkipCleanup:    false,
		IgnoredErrorIs: nil,
	}).Leveled(slog.LevelDebug)

	ctx := testutil.Context(t, testutil.WaitLong)

	db, err := spice.New(ctx, &spice.SpiceServerOpts{
		PostgresURI: "",
		Logger:      logger,
		Store:       dbmem.New(),
		Debug:       false,
	})
	require.NoError(t, err)
	err = db.Run(ctx)
	require.NoError(t, err)

	god := spice.AsGod(ctx)
	def, err := db.GetDefaultOrganization(god)
	require.NoError(t, err)

	owner := dbgen.User(t, db, database.User{
		RBACRoles: []string{rbac.RoleOwner()},
	})
	member := dbgen.User(t, db, database.User{})
	otherMember := dbgen.User(t, db, database.User{})

	dbgen.OrganizationMember(t, db, database.OrganizationMember{
		UserID:         owner.ID,
		OrganizationID: def.ID,
	})
	dbgen.OrganizationMember(t, db, database.OrganizationMember{
		UserID:         member.ID,
		OrganizationID: def.ID,
	})
	dbgen.OrganizationMember(t, db, database.OrganizationMember{
		UserID:         otherMember.ID,
		OrganizationID: def.ID,
	})

	memberCtx := spice.AsUser(ctx, member.ID)
	otherMemberCtx := spice.AsUser(ctx, otherMember.ID)
	ownerCtx := spice.AsUser(ctx, owner.ID)

	// Workspace owner by member
	workspace, err := db.InsertWorkspace(memberCtx, database.InsertWorkspaceParams{
		ID:                uuid.New(),
		CreatedAt:         dbtime.Now(),
		UpdatedAt:         dbtime.Now(),
		OwnerID:           member.ID,
		OrganizationID:    def.ID,
		TemplateID:        uuid.New(),
		Name:              namesgenerator.GetRandomName(1),
		AutostartSchedule: sql.NullString{},
		Ttl:               sql.NullInt64{},
		LastUsedAt:        time.Time{},
		AutomaticUpdates:  database.AutomaticUpdatesNever,
	})
	require.NoError(t, err)

	// Member can view
	_, err = db.GetWorkspaceByID(memberCtx, workspace.ID)
	require.NoErrorf(t, err, "Member: %s", member.ID.String())

	// Owner can view
	_, err = db.GetWorkspaceByID(ownerCtx, workspace.ID)
	require.NoErrorf(t, err, "Owner: %s", owner.ID.String())

	// Other member cannot
	_, err = db.GetWorkspaceByID(otherMemberCtx, workspace.ID)
	require.Errorf(t, err, "OtherMember: %s", otherMember.ID.String())
}

func TestExampleList(t *testing.T) {
	t.Parallel()

	logger := slogtest.Make(t, &slogtest.Options{
		IgnoreErrors:   false,
		SkipCleanup:    false,
		IgnoredErrorIs: nil,
	}).Leveled(slog.LevelDebug)

	ctx := testutil.Context(t, testutil.WaitLong)

	db, err := spice.New(ctx, &spice.SpiceServerOpts{
		PostgresURI: "",
		Logger:      logger,
		Store:       dbmem.New(),
		Debug:       false,
	})
	require.NoError(t, err)
	err = db.Run(ctx)
	require.NoError(t, err)

	god := spice.AsGod(ctx)
	def, err := db.GetDefaultOrganization(god)
	require.NoError(t, err)

	owner := dbgen.User(t, db, database.User{
		RBACRoles: []string{rbac.RoleOwner()},
	})
	dbgen.OrganizationMember(t, db, database.OrganizationMember{
		UserID:         owner.ID,
		OrganizationID: def.ID,
	})

	users := make([]database.User, 5)
	for i := range users {
		users[i] = dbgen.User(t, db, database.User{})
		dbgen.OrganizationMember(t, db, database.OrganizationMember{
			UserID:         users[i].ID,
			OrganizationID: def.ID,
		})

		memberCtx := spice.AsUser(ctx, users[i].ID)
		// Workspace owner by member
		_, err := db.InsertWorkspace(memberCtx, database.InsertWorkspaceParams{
			ID:                uuid.New(),
			CreatedAt:         dbtime.Now(),
			UpdatedAt:         dbtime.Now(),
			OwnerID:           users[i].ID,
			OrganizationID:    def.ID,
			TemplateID:        uuid.New(),
			Name:              namesgenerator.GetRandomName(1),
			AutostartSchedule: sql.NullString{},
			Ttl:               sql.NullInt64{},
			LastUsedAt:        time.Time{},
			AutomaticUpdates:  database.AutomaticUpdatesNever,
		})
		require.NoError(t, err)
	}

	db.Debugging(true)
	memberCtx := spice.AsUser(ctx, users[0].ID)
	workspaces, err := db.GetWorkspaces(memberCtx, database.GetWorkspacesParams{})
	require.NoError(t, err)
	require.Len(t, workspaces, 1)

	ownerCtx := spice.AsUser(ctx, owner.ID)
	workspaces, err = db.GetWorkspaces(ownerCtx, database.GetWorkspacesParams{})
	require.NoError(t, err)
	require.Len(t, workspaces, len(users))
}

func TestCustomOrganizationRoles(t *testing.T) {
	t.Parallel()

	logger := slogtest.Make(t, &slogtest.Options{
		IgnoreErrors:   false,
		SkipCleanup:    false,
		IgnoredErrorIs: nil,
	}).Leveled(slog.LevelDebug)

	ctx := testutil.Context(t, testutil.WaitLong)

	db, err := spice.New(ctx, &spice.SpiceServerOpts{
		PostgresURI: "",
		Logger:      logger,
		Store:       dbmem.New(),
		Debug:       false,
	})
	require.NoError(t, err)
	err = db.Run(ctx)
	require.NoError(t, err)

	god := spice.AsGod(ctx)
	def, err := db.GetDefaultOrganization(god)
	require.NoError(t, err)

	// Create a normal user in the organization.
	user := dbgen.User(t, db, database.User{
		RBACRoles: []string{},
	})
	dbgen.OrganizationMember(t, db, database.OrganizationMember{
		UserID:         user.ID,
		OrganizationID: def.ID,
	})

	userCtx := spice.AsUser(ctx, user.ID)

	// By default, users cannot create templates.
	err = db.Check(policy.New().Organization(def.ID).CanCreate_template(userCtx))
	require.Error(t, err)

	// Create the new custom role
	const roleName = "template_creator"
	err = db.UpsertCustomOrganizationRole(spice.WithDebugging(ctx), roleName, def.ID, func(role *policy.ObjOrg_role, organization *policy.ObjOrganization) {
		organization.
			Template_creatorOrg_role(role).        // Create
			Template_editorOrg_role(role).         // Edit
			Template_insights_viewerOrg_role(role) // See insights
	})
	require.NoError(t, err)

	// Now the custom role exists, let's assign it to the user and try it out.
	// TODO: Probably a helper function for this on the spicedb or something.
	// 	Doing this manually requires knowing the rolename and relations.
	builder := policy.New()
	builder.Org_role(policy.String(roleName)).MemberUser(builder.User(user.ID))
	_, err = db.WriteRelationships(ctx, builder.Relationships...)
	require.NoError(t, err)

	// The user is a member of the org and has the role, so they should be able to
	// create a template
	err = db.Check(policy.New().Organization(def.ID).CanCreate_template(userCtx))
	require.NoError(t, err)

	// An example of updating the role to also grant create_user permissions.
	err = db.Check(policy.New().Organization(def.ID).CanCreate_org_member(userCtx))
	require.Error(t, err) // The custom role does not grant this yet

	err = db.UpsertCustomOrganizationRole(spice.WithDebugging(ctx), roleName, def.ID, func(role *policy.ObjOrg_role, organization *policy.ObjOrganization) {
		organization.
			// Intentionally omit create_template to check it was removed.
			Template_editorOrg_role(role).          // Edit
			Template_insights_viewerOrg_role(role). // See insights
			Member_creatorOrg_role(role)            // Create org members
	})
	require.NoError(t, err)

	// Now it will work!
	err = db.Check(policy.New().Organization(def.ID).CanCreate_org_member(userCtx))
	require.NoError(t, err)

	// We removed this perm
	err = db.Check(policy.New().Organization(def.ID).CanCreate_template(userCtx))
	require.Error(t, err)

	// And this perm still exists
	err = db.Check(policy.New().Organization(def.ID).CanEdit_templates(userCtx))
	require.NoError(t, err)

	// You can see what perms the role has.
	perms, err := db.OrganizationRolePermissions(ctx, roleName, def.ID)
	require.NoError(t, err)
	// Role template_creator has permissions: member_creator, template_editor, template_insights_viewer
	t.Logf("Role %s has permissions: %s", roleName, strings.Join(perms, ", "))

	// Who has the role?
	assigned, err := db.OrganizationRoleAssignedActors(ctx, roleName)
	require.NoError(t, err)
	// Role template_creator is assigned to: e902b0fa-58d1-4979-9102-16047cb56173
	t.Logf("Role %s is assigned to: %s", roleName, strings.Join(assigned, ", "))
}
