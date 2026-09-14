// Package license_policy implements the secobserve_license_policy data source.
package license_policy //nolint:revive // the package name mirrors the Terraform data source name

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

var (
	_ datasource.DataSourceWithConfigure        = (*licensePolicyDataSource)(nil)
	_ datasource.DataSourceWithConfigValidators = (*licensePolicyDataSource)(nil)
)

// New returns the secobserve_license_policy data source.
func New() datasource.DataSource {
	return &licensePolicyDataSource{}
}

type licensePolicyDataSource struct {
	client *client.Client
}

type model struct {
	ID                      types.Int64  `tfsdk:"id"`
	Name                    types.String `tfsdk:"name"`
	Parent                  types.Int64  `tfsdk:"parent"`
	ParentName              types.String `tfsdk:"parent_name"`
	Description             types.String `tfsdk:"description"`
	IsPublic                types.Bool   `tfsdk:"is_public"`
	IgnoreComponentTypeList types.Set    `tfsdk:"ignore_component_type_list"`

	IsParent               types.Bool `tfsdk:"is_parent"`
	IsManager              types.Bool `tfsdk:"is_manager"`
	HasProducts            types.Bool `tfsdk:"has_products"`
	HasProductGroups       types.Bool `tfsdk:"has_product_groups"`
	HasItems               types.Bool `tfsdk:"has_items"`
	HasUsers               types.Bool `tfsdk:"has_users"`
	HasAuthorizationGroups types.Bool `tfsdk:"has_authorization_groups"`
}

func (d *licensePolicyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_license_policy"
}

func (d *licensePolicyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = tfutil.DataSourceClient(req, resp)
}

func (d *licensePolicyDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("name")),
	}
}

func (d *licensePolicyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a license policy by id or exact name.\n\n" +
			"This is the documented way to reference the seeded `Standard` policy without adopting it into " +
			"Terraform, and to reference a policy for `secobserve_product.license_policy` / " +
			"`secobserve_product_group.license_policy`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Numeric id of the license policy. Supply either this or `name`.",
			},
			"name": schema.StringAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "Exact name of the license policy. Supply either this or `id`. Matching " +
					"is exact and case-sensitive, even though SecObserve's own filter is a substring match.",
			},
			"parent":      schema.Int64Attribute{Computed: true, MarkdownDescription: "Id of the parent policy, if any."},
			"parent_name": schema.StringAttribute{Computed: true, MarkdownDescription: "Name of the parent policy, or empty."},
			"description": schema.StringAttribute{Computed: true, MarkdownDescription: "Free-text description."},
			"is_public": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether every authenticated user can see this policy, not just members.",
			},
			"ignore_component_type_list": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Package URL types this policy skips entirely.",
			},
			"is_parent": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether at least one other policy has this one as its parent.",
			},
			"is_manager": schema.BoolAttribute{
				Computed: true,
				MarkdownDescription: "Whether the provider's identity is an explicit manager member of this " +
					"policy. A superuser can manage the policy through the resource regardless of this flag.",
			},
			"has_products":       schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether at least one product uses this policy."},
			"has_product_groups": schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether at least one product group uses this policy."},
			"has_items":          schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether this policy has at least one item."},
			"has_users":          schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether this policy has at least one user member."},
			"has_authorization_groups": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this policy has at least one authorization group member.",
			},
		},
	}
}

func (d *licensePolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var (
		found client.LicensePolicy
		err   error
	)
	if !config.ID.IsNull() {
		found, err = d.client.LicensePolicy(ctx, config.ID.ValueInt64())
	} else {
		found, err = d.client.LicensePolicyByName(ctx, config.Name.ValueString())
		if err == nil {
			// The name lookup goes through the list endpoint, whose serializer
			// omits ignore_component_type_list and every is_*/has_* field;
			// re-read the detail representation.
			found, err = d.client.LicensePolicy(ctx, found.ID)
		}
	}
	if err != nil {
		resp.Diagnostics.AddError("Could not read SecObserve license policy", err.Error())
		return
	}

	ignoreTypes, setDiags := types.SetValueFrom(ctx, types.StringType, found.IgnoreComponentTypeList)
	resp.Diagnostics.Append(setDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := model{
		ID:                      types.Int64Value(found.ID),
		Name:                    types.StringValue(found.Name),
		Parent:                  tfutil.Int64(found.Parent),
		ParentName:              types.StringValue(found.ParentName),
		Description:             types.StringValue(found.Description),
		IsPublic:                types.BoolValue(found.IsPublic),
		IgnoreComponentTypeList: ignoreTypes,
		IsParent:                types.BoolValue(found.IsParent),
		IsManager:               types.BoolValue(found.IsManager),
		HasProducts:             types.BoolValue(found.HasProducts),
		HasProductGroups:        types.BoolValue(found.HasProductGroups),
		HasItems:                types.BoolValue(found.HasItems),
		HasUsers:                types.BoolValue(found.HasUsers),
		HasAuthorizationGroups:  types.BoolValue(found.HasAuthorizationGroups),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
