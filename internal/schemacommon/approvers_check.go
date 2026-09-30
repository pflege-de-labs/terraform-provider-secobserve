package schemacommon

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

// CheckApproverGroups mirrors SecObserve's rule that every designated approver
// group already holds Writer or above on one of productIDs -- the product or
// product group itself and, for a product, its product group
// (core/api/serializers_product.py:153-193).
//
// An empty productIDs means the create is certain to be rejected, so that is an
// error. A missing membership is only a warning: the provider cannot see a
// membership planned in the same run, which the create would then find.
//
// A nil client, e.g. while the provider configuration is still unknown, skips
// the API lookup.
func (a Approvers) CheckApproverGroups(
	ctx context.Context, c *client.Client, productIDs []int64, diags *diag.Diagnostics,
) {
	set := a.AssessmentApproverAuthorizationGroups
	if set.IsNull() || set.IsUnknown() || len(set.Elements()) == 0 {
		return
	}
	for _, element := range set.Elements() {
		if element.IsUnknown() {
			return
		}
	}

	attribute := path.Root("assessment_approver_authorization_groups")
	if len(productIDs) == 0 {
		diags.AddAttributeError(attribute,
			"Approver groups cannot be set on create",
			"SecObserve only accepts approver groups that already hold the Writer role or above on the "+
				"product or its product group. A new product group, or a new product without a "+
				"product_group, has no memberships yet, so SecObserve always rejects this create.\n\n"+
				"Create it without assessment_approver_authorization_groups, add a "+
				"secobserve_product_authorization_group_member with role Writer or above, then set the approvers.",
		)
		return
	}
	if c == nil {
		return
	}

	groups := tfutil.Int64Set(ctx, set, diags)
	for _, group := range groups {
		members, err := c.ProductAuthorizationGroupMembersOf(ctx, group)
		if err != nil {
			diags.AddAttributeWarning(attribute,
				"Could not verify approver group role",
				fmt.Sprintf("Looking up the memberships of authorization group %d failed, so the plan cannot "+
					"confirm it holds Writer or above: %s", group, err),
			)
			continue
		}

		qualified := slices.ContainsFunc(members, func(m client.ProductAuthorizationGroupMember) bool {
			return slices.Contains(productIDs, m.Product) && m.Role >= client.RoleWriter
		})
		if !qualified {
			diags.AddAttributeWarning(attribute,
				"Approver group lacks the Writer role",
				fmt.Sprintf("Authorization group %d holds no Writer role or above on %v. SecObserve will reject "+
					"the apply with \"Designated approver groups must have at least the Writer role.\"\n\n"+
					"Ignore this if a secobserve_product_authorization_group_member granting that role is "+
					"created earlier in the same apply; the provider cannot see other planned resources.",
					group, productIDs),
			)
		}
	}
}
