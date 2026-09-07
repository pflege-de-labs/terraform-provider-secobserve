// Package branch implements the secobserve_branch resource.
package branch

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jabbrwcky/terraform-provider-secobserve/internal/client"
	"github.com/jabbrwcky/terraform-provider-secobserve/internal/schemacommon"
	"github.com/jabbrwcky/terraform-provider-secobserve/internal/tfutil"
)

var (
	_ resource.ResourceWithConfigure      = (*branchResource)(nil)
	_ resource.ResourceWithImportState    = (*branchResource)(nil)
	_ resource.ResourceWithValidateConfig = (*branchResource)(nil)
)

// New returns the secobserve_branch resource.
func New() resource.Resource {
	return &branchResource{}
}

type branchResource struct {
	client *client.Client
}

type model struct {
	ID              types.Int64  `tfsdk:"id"`
	Product         types.Int64  `tfsdk:"product"`
	Name            types.String `tfsdk:"name"`
	NameWithProduct types.String `tfsdk:"name_with_product"`

	IsDefaultBranch     types.Bool   `tfsdk:"is_default_branch"`
	HousekeepingProtect types.Bool   `tfsdk:"housekeeping_protect"`
	LastImport          types.String `tfsdk:"last_import"`

	Purl                 types.String `tfsdk:"purl"`
	CPE23                types.String `tfsdk:"cpe23"`
	OSVLinuxDistribution types.String `tfsdk:"osv_linux_distribution"`
	OSVLinuxRelease      types.String `tfsdk:"osv_linux_release"`
}

func (r *branchResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_branch"
}

func (r *branchResource) Configure(
	_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse,
) {
	r.client = tfutil.ResourceClient(req, resp)
}

func (r *branchResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A branch or version of a product. Observations and license components are " +
			"attributed to one.\n\n" +
			"~> `is_default_branch` is effectively a per-product singleton: setting it here clears the flag " +
			"on the product's other branches, which Terraform cannot observe. Set it on exactly one " +
			"`secobserve_branch` per product.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{schemacommon.Int64UseStateForUnknown()},
				MarkdownDescription: "Numeric id of the branch.",
			},
			"product": schema.Int64Attribute{
				Required:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
				MarkdownDescription: "Id of the product this branch belongs to. Immutable: SecObserve rejects " +
					"moving a branch between products, so a change replaces the branch.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 255)},
				MarkdownDescription: "Name of the branch or version. Unique per product, not globally.",
			},
			"name_with_product": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Branch name qualified with its product, as SecObserve displays it.",
			},
			"is_default_branch": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				MarkdownDescription: "Whether this is the product's default branch.\n\n" +
					"Setting it also updates the product's read-only `repository_default_branch` and clears " +
					"the flag on the product's other branches.\n\n" +
					"~> SecObserve refuses to delete the default branch. Move the flag to another branch " +
					"before destroying this one.",
			},
			"housekeeping_protect": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Exempt this branch from automatic deletion by branch housekeeping.",
			},
			"last_import": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When observations were last imported for this branch.",
			},
			"purl": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				Validators:          []validator.String{stringvalidator.LengthAtMost(255)},
				MarkdownDescription: "Package URL identifying this branch. Validated server-side.",
			},
			"cpe23": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				Validators:          []validator.String{stringvalidator.LengthAtMost(255)},
				MarkdownDescription: "CPE 2.3 name identifying this branch. Validated server-side.",
			},
			"osv_linux_distribution": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.OneOf(schemacommon.WithEmpty(client.OSVLinuxDistributions)...),
				},
				MarkdownDescription: "Linux distribution to consider when matching OSV advisories for this " +
					"branch, overriding the product's setting. One of " +
					schemacommon.MarkdownList(client.OSVLinuxDistributions) + ".",
			},
			"osv_linux_release": schema.StringAttribute{
				Optional:   true,
				Computed:   true,
				Default:    stringdefault.StaticString(""),
				Validators: []validator.String{stringvalidator.LengthAtMost(255)},
				MarkdownDescription: "Release of the Linux distribution. Requires `osv_linux_distribution` " +
					"to be set.",
			},
		},
	}
}

func (r *branchResource) ValidateConfig(
	ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse,
) {
	var config model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if tfutil.StringValue(config.OSVLinuxRelease) != "" && tfutil.StringValue(config.OSVLinuxDistribution) == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("osv_linux_release"),
			"osv_linux_release needs a distribution",
			"SecObserve rejects osv_linux_release unless osv_linux_distribution is also set.",
		)
	}
}

func (m model) toRequest() client.BranchRequest {
	return client.BranchRequest{
		Product:              m.Product.ValueInt64(),
		Name:                 m.Name.ValueString(),
		IsDefaultBranch:      tfutil.BoolValue(m.IsDefaultBranch),
		HousekeepingProtect:  tfutil.BoolValue(m.HousekeepingProtect),
		Purl:                 tfutil.StringValue(m.Purl),
		CPE23:                tfutil.StringValue(m.CPE23),
		OSVLinuxDistribution: tfutil.StringValue(m.OSVLinuxDistribution),
		OSVLinuxRelease:      tfutil.StringValue(m.OSVLinuxRelease),
	}
}

func (m *model) fromAPI(branch client.Branch) {
	m.ID = types.Int64Value(branch.ID)
	m.Product = types.Int64Value(branch.Product)
	m.Name = types.StringValue(branch.Name)
	m.NameWithProduct = types.StringValue(branch.NameWithProduct)
	m.IsDefaultBranch = types.BoolValue(branch.IsDefaultBranch)
	m.HousekeepingProtect = types.BoolValue(branch.HousekeepingProtect)
	m.LastImport = tfutil.String(branch.LastImport)
	m.Purl = types.StringValue(branch.Purl)
	m.CPE23 = types.StringValue(branch.CPE23)
	m.OSVLinuxDistribution = types.StringValue(branch.OSVLinuxDistribution)
	m.OSVLinuxRelease = types.StringValue(branch.OSVLinuxRelease)
}

func (r *branchResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateBranch(ctx, plan.toRequest())
	if err != nil {
		resp.Diagnostics.AddError("Could not create SecObserve branch", err.Error())
		return
	}

	var state model
	state.fromAPI(created)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *branchResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	branch, err := r.client.Branch(ctx, state.ID.ValueInt64())
	if err != nil {
		if client.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Could not read SecObserve branch", err.Error())
		return
	}

	var refreshed model
	refreshed.fromAPI(branch)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *branchResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateBranch(ctx, state.ID.ValueInt64(), plan.toRequest())
	if err != nil {
		resp.Diagnostics.AddError("Could not update SecObserve branch", err.Error())
		return
	}

	var refreshed model
	refreshed.fromAPI(updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *branchResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteBranch(ctx, state.ID.ValueInt64())
	if err != nil {
		if client.NotFound(err) {
			return
		}
		r.explainDeleteFailure(err, state, &resp.Diagnostics)
	}
}

// explainDeleteFailure turns the two failures SecObserve reports here into
// something actionable.
func (r *branchResource) explainDeleteFailure(err error, state model, diags *diag.Diagnostics) {
	if state.IsDefaultBranch.ValueBool() {
		diags.AddError(
			"SecObserve refused to delete the default branch",
			fmt.Sprintf("Branch %q is the default branch of product %d, and SecObserve does not allow "+
				"deleting it.\n\nMove is_default_branch to another branch of the product first, then destroy "+
				"this one.\n\nUnderlying error: %s",
				state.Name.ValueString(), state.Product.ValueInt64(), err),
		)
		return
	}
	if client.Conflict(err) {
		diags.AddError(
			"SecObserve refused to delete the branch",
			"Something still references it.\n\nUnderlying error: "+err.Error(),
		)
		return
	}
	diags.AddError("Could not delete SecObserve branch", err.Error())
}

// ImportState takes "<product id>/<branch name>", because branch names are
// only unique within a product.
func (r *branchResource) ImportState(
	ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	productID, name, found := strings.Cut(req.ID, "/")
	if !found {
		resp.Diagnostics.AddError(
			"Invalid import id",
			"Import a branch as \"<product id>/<branch name>\", for example \"12/main\". Branch names are "+
				"unique per product, not globally.",
		)
		return
	}

	parsedProduct, err := strconv.ParseInt(productID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid import id",
			"The product part of \"<product id>/<branch name>\" must be numeric: "+err.Error(),
		)
		return
	}

	branch, err := r.client.BranchByName(ctx, parsedProduct, name)
	if err != nil {
		resp.Diagnostics.AddError("Could not import SecObserve branch", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), branch.ID)...)
}
