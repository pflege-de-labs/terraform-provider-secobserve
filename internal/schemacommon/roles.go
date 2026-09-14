package schemacommon

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
)

// AddRole contributes a role attribute.
//
// The role is exposed by name rather than by the wire integer: SecObserve only
// range-checks 1..5 and has no choices on the column, so a typo'd number would
// be stored silently.
func AddRole(attributes map[string]schema.Attribute, subject string) {
	attributes["role"] = schema.StringAttribute{
		Required:   true,
		Validators: []validator.String{stringvalidator.OneOf(client.RoleNameList...)},
		MarkdownDescription: fmt.Sprintf(
			"Role granted to the %s, in ascending order of privilege: %s.\n\n"+
				"~> Granting or changing `Owner` requires the provider's own identity to be an owner of the "+
				"product or a superuser.",
			subject, MarkdownList(client.RoleNameList)),
	}
}

// RoleValue converts a role name to the numeric value the API expects. The
// schema validator guarantees the name is known.
func RoleValue(name string) int64 {
	return client.RoleNames[name]
}

// RoleName converts a numeric role from an API response back to its name.
func RoleName(role int64) string {
	return client.RoleName(role)
}
