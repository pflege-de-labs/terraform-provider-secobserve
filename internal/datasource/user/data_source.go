// Package user implements the secobserve_user data source.
package user

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jabbrwcky/terraform-provider-secobserve/internal/client"
	"github.com/jabbrwcky/terraform-provider-secobserve/internal/tfutil"
)

var (
	_ datasource.DataSourceWithConfigure        = (*userDataSource)(nil)
	_ datasource.DataSourceWithConfigValidators = (*userDataSource)(nil)
)

// New returns the secobserve_user data source.
func New() datasource.DataSource {
	return &userDataSource{}
}

type userDataSource struct {
	client *client.Client
}

type model struct {
	ID          types.Int64  `tfsdk:"id"`
	Username    types.String `tfsdk:"username"`
	FirstName   types.String `tfsdk:"first_name"`
	LastName    types.String `tfsdk:"last_name"`
	FullName    types.String `tfsdk:"full_name"`
	Email       types.String `tfsdk:"email"`
	IsActive    types.Bool   `tfsdk:"is_active"`
	IsSuperuser types.Bool   `tfsdk:"is_superuser"`
	IsExternal  types.Bool   `tfsdk:"is_external"`
	IsOIDCUser  types.Bool   `tfsdk:"is_oidc_user"`
	HasPassword types.Bool   `tfsdk:"has_password"`
}

func (d *userDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *userDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = tfutil.DataSourceClient(req, resp)
}

func (d *userDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("username")),
	}
}

func (d *userDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a user by id or exact username.\n\n" +
			"On an OIDC-enabled instance this is the documented way to *reference* a user provisioned " +
			"just-in-time at login -- for `secobserve_product_member`, `assessment_approvers`, and so on -- " +
			"without Terraform claiming ownership of a record SecObserve itself overwrites on every login.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Numeric id of the user. Supply either this or `username`.",
			},
			"username": schema.StringAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "Exact username. Supply either this or `id`. Matching is exact and " +
					"case-sensitive, even though SecObserve's own filter is a substring match.",
			},
			"first_name":   schema.StringAttribute{Computed: true, MarkdownDescription: "First name."},
			"last_name":    schema.StringAttribute{Computed: true, MarkdownDescription: "Last name."},
			"full_name":    schema.StringAttribute{Computed: true, MarkdownDescription: "Display name."},
			"email":        schema.StringAttribute{Computed: true, MarkdownDescription: "Email address."},
			"is_active":    schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether the user can authenticate."},
			"is_superuser": schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether the user has superuser privileges."},
			"is_external":  schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether the user is external."},
			"is_oidc_user": schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether the user is OIDC-provisioned."},
			"has_password": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the user has a usable password set.",
			},
		},
	}
}

func (d *userDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var (
		found client.User
		err   error
	)
	if !config.ID.IsNull() {
		found, err = d.client.User(ctx, config.ID.ValueInt64())
	} else {
		found, err = d.client.UserByUsername(ctx, config.Username.ValueString())
	}
	if err != nil {
		resp.Diagnostics.AddError("Could not read SecObserve user", err.Error())
		return
	}

	state := model{
		ID:          types.Int64Value(found.ID),
		Username:    types.StringValue(found.Username),
		FirstName:   types.StringValue(found.FirstName),
		LastName:    types.StringValue(found.LastName),
		FullName:    types.StringValue(found.FullName),
		Email:       types.StringValue(found.Email),
		IsActive:    types.BoolValue(found.IsActive),
		IsSuperuser: types.BoolValue(found.IsSuperuser),
		IsExternal:  types.BoolValue(found.IsExternal),
		IsOIDCUser:  types.BoolValue(found.IsOIDCUser),
		HasPassword: types.BoolValue(found.HasPassword),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
