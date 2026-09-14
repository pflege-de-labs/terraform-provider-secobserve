package product

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/schemacommon"
	sovalidators "github.com/pflege-de-labs/terraform-provider-secobserve/internal/validators"
)

func (r *productResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := map[string]schema.Attribute{
		// Identity.
		"id": schema.Int64Attribute{
			Computed:            true,
			PlanModifiers:       []planmodifier.Int64{schemacommon.Int64UseStateForUnknown()},
			MarkdownDescription: "Numeric id of the product.",
		},
		"name": schema.StringAttribute{
			Required:   true,
			Validators: []validator.String{stringvalidator.LengthBetween(1, 255)},
			MarkdownDescription: "Name of the product. Must be unique across **both** products and product " +
				"groups, because SecObserve keeps them in one table.",
		},
		"description": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
			Validators:          []validator.String{stringvalidator.LengthAtMost(2048)},
			MarkdownDescription: "Free-text description.",
		},
		"product_group": schema.Int64Attribute{
			Optional:   true,
			Validators: []validator.Int64{int64validator.AtLeast(1)},
			MarkdownDescription: "Id of the product group this product belongs to. The product inherits the " +
				"group's security gate, housekeeping, notification and approval settings wherever its own " +
				"are left unset.\n\n" +
				"~> Deleting the product group **deletes this product too**.",
		},
		"product_group_name": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Name of the product group, for convenience.",
		},

		// Repository.
		"repository_prefix": schema.StringAttribute{
			Optional: true,
			Computed: true,
			Default:  stringdefault.StaticString(""),
			Validators: []validator.String{
				stringvalidator.LengthAtMost(255),
				sovalidators.HTTPURL(),
			},
			MarkdownDescription: "Base URL of the source repository, used to build links to source files.",
		},
		"repository_default_branch": schema.Int64Attribute{
			Computed: true,
			MarkdownDescription: "Id of the product's default branch.\n\n" +
				"Read-only: SecObserve sets it as a side effect of `is_default_branch` on a " +
				"`secobserve_branch`, so set the flag there rather than here.",
		},
		"repository_default_branch_name": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Name of the product's default branch.",
		},

		// Identifiers.
		"purl": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
			Validators:          []validator.String{stringvalidator.LengthAtMost(255)},
			MarkdownDescription: "Package URL identifying the product. Validated server-side.",
		},
		"cpe23": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
			Validators:          []validator.String{stringvalidator.LengthAtMost(255)},
			MarkdownDescription: "CPE 2.3 name identifying the product. Validated server-side.",
		},

		// Rules.
		"apply_general_rules": schema.BoolAttribute{
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(true),
			MarkdownDescription: "Whether the instance-wide general rules are applied to this product.",
		},

		// Status.
		"security_gate_passed": schema.BoolAttribute{
			Computed: true,
			MarkdownDescription: "Whether the product currently passes its security gate. Computed by " +
				"SecObserve from the observations that have been imported.",
		},
	}

	// Blocks shared with secobserve_product_group.
	schemacommon.AddSecurityGate(attributes)
	schemacommon.AddBranchHousekeeping(attributes)
	schemacommon.AddNotifications(attributes)
	schemacommon.AddApprovals(attributes)
	schemacommon.AddApprovers(attributes, "product or its product group")
	schemacommon.AddRiskAcceptance(attributes)
	schemacommon.AddLicensePolicy(attributes)
	schemacommon.AddBranchPropagation(attributes)

	// Product-only blocks.
	addIssueTracker(attributes)
	addScanners(attributes)

	resp.Schema = schema.Schema{
		MarkdownDescription: "A product is the unit vulnerabilities and licenses are tracked against: a " +
			"repository, a service, a container image, or whatever else you scan.\n\n" +
			"The attributes that model inheritance are tri-state. Leaving one unset means *inherit from the " +
			"product group, or failing that from the instance settings*, which is not the same as setting it " +
			"to `false`.\n\n" +
			"~> Deleting a product **cascades** to its branches, services, observations and API tokens.\n\n" +
			"~> Whoever creates a product becomes an `Owner` member of it automatically. Since Terraform " +
			"creates products as the provider's own identity, that membership exists outside Terraform and a " +
			"`secobserve_product_member` for the same user would fail as a duplicate.",
		Attributes: attributes,
	}
}
