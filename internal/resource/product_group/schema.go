package product_group

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/schemacommon"
)

func (r *productGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Computed:            true,
			PlanModifiers:       []planmodifier.Int64{schemacommon.Int64UseStateForUnknown()},
			MarkdownDescription: "Numeric id of the product group.",
		},
		"name": schema.StringAttribute{
			Required:   true,
			Validators: []validator.String{stringvalidator.LengthBetween(1, 255)},
			MarkdownDescription: "Name of the product group. Must be unique across **both** products and product " +
				"groups, because SecObserve keeps them in one table.",
		},
		"description": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
			Validators:          []validator.String{stringvalidator.LengthAtMost(2048)},
			MarkdownDescription: "Free-text description.",
		},
	}

	// Blocks shared with secobserve_product.
	schemacommon.AddSecurityGate(attributes)
	schemacommon.AddBranchHousekeeping(attributes)
	schemacommon.AddNotifications(attributes)
	schemacommon.AddApprovals(attributes)
	schemacommon.AddApprovers(attributes, "product group")
	schemacommon.AddRiskAcceptance(attributes)
	schemacommon.AddLicensePolicy(attributes)
	schemacommon.AddBranchPropagation(attributes)

	resp.Schema = schema.Schema{
		// v0 -> v1: security_gate_threshold_* and
		// repository_branch_housekeeping_{keep_inactive_days,exempt_branches}
		// moved into the nested security_gate/repository_branch_housekeeping
		// objects. See UpgradeState.
		Version: 1,
		MarkdownDescription: "A product group bundles related products and supplies the defaults they inherit: " +
			"security gate thresholds, branch housekeeping, notification targets, approval requirements and the " +
			"license policy.\n\n" +
			"The attributes that model inheritance are tri-state. Leaving one unset means *inherit from the " +
			"instance settings*, which is not the same as setting it to `false`.\n\n" +
			"~> Deleting a product group **cascades to every product in it**, together with their branches, " +
			"services and observations.",
		Attributes: attributes,
	}
}
