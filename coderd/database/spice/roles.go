package spice

import (
	"context"
	"fmt"

	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/google/uuid"

	"github.com/coder/coder/v2/coderd/database/spice/policy"
)

func (s *SpiceDB) UpsertCustomOrganizationRole(ctx context.Context, roleName string, orgID uuid.UUID, assignPerms func(role *policy.ObjOrg_role, organization *policy.ObjOrganization)) error {
	// TODO: Do a perm check if the user can make a custom org role?

	builder := policy.New()
	org := builder.Organization(orgID)
	newRole := builder.Org_role(policy.String(roleName))
	// Associate to the specific org
	newRole.Organization(org)
	// Let the caller assign permissions.
	assignPerms(newRole, org)

	// Remove all existing permissions for this role for the upsert.
	// This ensures a clean slate, so all perms must be provided in the
	// function call.
	resp, err := s.permCli.DeleteRelationships(ctx, &v1.DeleteRelationshipsRequest{
		RelationshipFilter: &v1.RelationshipFilter{
			ResourceType:             newRole.Object().ObjectType,
			OptionalResourceId:       newRole.Object().ObjectId,
			OptionalResourceIdPrefix: "",
			OptionalRelation:         "",
			// Remove all relations from the org to this role.
			// This will strip it of all it's permissions, and leave a clean slate.
			// Any assigned users will remain though!
			OptionalSubjectFilter: &v1.SubjectFilter{
				SubjectType:       org.Object().ObjectType,
				OptionalSubjectId: org.Object().ObjectId,
				OptionalRelation:  nil,
			},
		},
		OptionalPreconditions:         nil,
		OptionalLimit:                 0,
		OptionalAllowPartialDeletions: false,
	})
	if err != nil {
		return fmt.Errorf("failed to delete existing relationships: %w", err)
	}
	s.zedToken.Store(resp.DeletedAt)

	// Add the new permissions.
	_, err = s.WriteRelationships(ctx, builder.Relationships...)
	if err != nil {
		return fmt.Errorf("failed to write relationships: %w", err)
	}

	// TODO: Write to coderd database with the new roles?

	return nil
}
