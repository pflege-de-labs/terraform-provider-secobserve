// Package license_group implements the secobserve_license_group data source.
package license_group //nolint:revive // the package name mirrors the Terraform data source name

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
	_ datasource.DataSourceWithConfigure        = (*licenseGroupDataSource)(nil)
	_ datasource.DataSourceWithConfigValidators = (*licenseGroupDataSource)(nil)
)

// New returns the secobserve_license_group data source.
func New() datasource.DataSource {
	return &licenseGroupDataSource{}
}

type licenseGroupDataSource struct {
	client *client.Client
}

type model struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	IsPublic    types.Bool   `tfsdk:"is_public"`

	IsManager              types.Bool `tfsdk:"is_manager"`
	IsInLicensePolicy      types.Bool `tfsdk:"is_in_license_policy"`
	HasLicenses            types.Bool `tfsdk:"has_licenses"`
	HasUsers               types.Bool `tfsdk:"has_users"`
	HasAuthorizationGroups types.Bool `tfsdk:"has_authorization_groups"`

	Licenses types.Set `tfsdk:"licenses"`
}

func (d *licenseGroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_license_group"
}

func (d *licenseGroupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = tfutil.DataSourceClient(req, resp)
}

func (d *licenseGroupDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("name")),
	}
}

func (d *licenseGroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a license group by id or exact name.\n\n" +
			"This is the documented way to reference the system-managed `(ScanCode LicenseDB)` groups, which " +
			"are reimported nightly and should not be owned by `secobserve_license_group`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Numeric id of the license group. Supply either this or `name`.",
			},
			"name": schema.StringAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "Exact name of the license group. Supply either this or `id`. Matching " +
					"is exact and case-sensitive, even though SecObserve's own filter is a substring match.",
			},
			"description": schema.StringAttribute{Computed: true, MarkdownDescription: "Free-text description."},
			"is_public": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether every authenticated user can see this group, not just members.",
			},
			"is_manager": schema.BoolAttribute{
				Computed: true,
				MarkdownDescription: "Whether the provider's identity is an explicit manager member of this " +
					"group. A superuser can manage the group through the resource regardless of this flag.",
			},
			"is_in_license_policy": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether any license policy item references this group.",
			},
			"has_licenses": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this group contains at least one license.",
			},
			"has_users": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this group has at least one user member.",
			},
			"has_authorization_groups": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this group has at least one authorization group member.",
			},
			"licenses": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.Int64Type,
				MarkdownDescription: "Ids of the licenses currently in this group.",
			},
		},
	}
}

func (d *licenseGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var (
		found client.LicenseGroup
		err   error
	)
	if !config.ID.IsNull() {
		found, err = d.client.LicenseGroup(ctx, config.ID.ValueInt64())
	} else {
		found, err = d.client.LicenseGroupByName(ctx, config.Name.ValueString())
		if err == nil {
			// The name lookup goes through the list endpoint, whose serializer
			// excludes the is_*/has_* fields entirely; re-read the detail
			// representation.
			found, err = d.client.LicenseGroup(ctx, found.ID)
		}
	}
	if err != nil {
		resp.Diagnostics.AddError("Could not read SecObserve license group", err.Error())
		return
	}

	licenses, err := d.client.ListLicensesInGroup(ctx, found.ID)
	if err != nil {
		resp.Diagnostics.AddError("Could not list licenses in SecObserve license group", err.Error())
		return
	}
	ids := make([]int64, 0, len(licenses))
	for _, license := range licenses {
		ids = append(ids, license.ID)
	}
	licenseSet, setDiags := types.SetValueFrom(ctx, types.Int64Type, ids)
	resp.Diagnostics.Append(setDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := model{
		ID:                     types.Int64Value(found.ID),
		Name:                   types.StringValue(found.Name),
		Description:            types.StringValue(found.Description),
		IsPublic:               types.BoolValue(found.IsPublic),
		IsManager:              types.BoolValue(found.IsManager),
		IsInLicensePolicy:      types.BoolValue(found.IsInLicensePolicy),
		HasLicenses:            types.BoolValue(found.HasLicenses),
		HasUsers:               types.BoolValue(found.HasUsers),
		HasAuthorizationGroups: types.BoolValue(found.HasAuthorizationGroups),
		Licenses:               licenseSet,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
