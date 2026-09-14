// Package authorization_group implements the secobserve_authorization_group
// data source.
package authorization_group //nolint:revive // the package name mirrors the Terraform data source name

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
	_ datasource.DataSourceWithConfigure        = (*groupDataSource)(nil)
	_ datasource.DataSourceWithConfigValidators = (*groupDataSource)(nil)
)

// New returns the secobserve_authorization_group data source.
func New() datasource.DataSource {
	return &groupDataSource{}
}

type groupDataSource struct {
	client *client.Client
}

type model struct {
	ID        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	OIDCGroup types.String `tfsdk:"oidc_group"`

	HasProductGroupMembers types.Bool `tfsdk:"has_product_group_members"`
	HasProductMembers      types.Bool `tfsdk:"has_product_members"`
	HasUsers               types.Bool `tfsdk:"has_users"`
	IsManager              types.Bool `tfsdk:"is_manager"`
}

func (d *groupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_authorization_group"
}

func (d *groupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = tfutil.DataSourceClient(req, resp)
}

func (d *groupDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("name")),
	}
}

func (d *groupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up an authorization group by id or exact name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Numeric id of the authorization group. Supply either this or `name`.",
			},
			"name": schema.StringAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "Exact name of the authorization group. Supply either this or `id`. " +
					"Matching is exact and case-sensitive, even though SecObserve's own filter is a " +
					"substring match.",
			},
			"oidc_group": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "IdP group claim this group is mapped to, if any.",
			},
			"has_product_group_members": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the group has a role on any product group.",
			},
			"has_product_members": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the group has a role on any product.",
			},
			"has_users": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the group has any members.",
			},
			"is_manager": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the provider's own identity manages this group.",
			},
		},
	}
}

func (d *groupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var (
		found client.AuthorizationGroup
		err   error
	)
	if !config.ID.IsNull() {
		found, err = d.client.AuthorizationGroup(ctx, config.ID.ValueInt64())
	} else {
		found, err = d.client.AuthorizationGroupByName(ctx, config.Name.ValueString())
	}
	if err != nil {
		resp.Diagnostics.AddError("Could not read SecObserve authorization group", err.Error())
		return
	}

	state := model{
		ID:                     types.Int64Value(found.ID),
		Name:                   types.StringValue(found.Name),
		OIDCGroup:              types.StringValue(found.OIDCGroup),
		HasProductGroupMembers: types.BoolValue(found.HasProductGroupMembers),
		HasProductMembers:      types.BoolValue(found.HasProductMembers),
		HasUsers:               types.BoolValue(found.HasUsers),
		IsManager:              types.BoolValue(found.IsManager),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
