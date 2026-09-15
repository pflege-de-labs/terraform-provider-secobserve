// Package product_rule implements the secobserve_product_rule resource.
package product_rule //nolint:revive // the package name mirrors the Terraform resource name

import (
	"context"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/schemacommon"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

var (
	_ resource.ResourceWithConfigure      = (*productRuleResource)(nil)
	_ resource.ResourceWithImportState    = (*productRuleResource)(nil)
	_ resource.ResourceWithValidateConfig = (*productRuleResource)(nil)
)

// New returns the secobserve_product_rule resource.
func New() resource.Resource {
	return &productRuleResource{}
}

type productRuleResource struct {
	client *client.Client
}

type model struct {
	ID      types.Int64 `tfsdk:"id"`
	Product types.Int64 `tfsdk:"product"`

	schemacommon.RuleFields
	schemacommon.RuleApproval
}

func (r *productRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_product_rule"
}

func (r *productRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = tfutil.ResourceClient(req, resp)
}

func (r *productRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Computed:            true,
			PlanModifiers:       []planmodifier.Int64{schemacommon.Int64UseStateForUnknown()},
			MarkdownDescription: "Numeric id of the rule.",
		},
		"product": schema.Int64Attribute{
			Required:      true,
			Validators:    []validator.Int64{int64validator.AtLeast(1)},
			PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			MarkdownDescription: "Id of the product this rule applies to. Immutable: SecObserve rejects " +
				"moving a rule to a different product, so a change replaces it.",
		},
	}
	schemacommon.AddRuleFields(attributes)
	schemacommon.AddRuleApproval(attributes)

	resp.Schema = schema.Schema{
		MarkdownDescription: "A rule applied to observations of one product.\n\n" +
			"~> **Approval is out of band.** A rule created while `product_rules_need_approval` is enabled " +
			"on the product or its product group starts in `approval_status = \"Needs approval\"`, and " +
			"every apply that touches this resource resets it back to that state. SecObserve also rejects " +
			"self-approval, so the identity Terraform authenticates as can never approve a rule it just " +
			"wrote -- somebody else has to do it through the UI or API.",
		Attributes: attributes,
	}
}

func (r *productRuleResource) ValidateConfig(
	ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse,
) {
	var config model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	config.RuleFields.ValidateRuleFields(&resp.Diagnostics)
}

func (m *model) fromAPI(ctx context.Context, rule client.Rule, diags *diag.Diagnostics) {
	m.ID = types.Int64Value(rule.ID)
	m.Product = tfutil.Int64(rule.Product)
	m.RuleFields.FromAPI(ctx, rule.RuleFields, diags)
	m.RuleApproval.FromAPI(rule)
}

func (r *productRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateProductRule(ctx, plan.Product.ValueInt64(), plan.RuleFields.ToAPI(ctx, &resp.Diagnostics))
	if err != nil {
		resp.Diagnostics.AddError("Could not create SecObserve product rule", err.Error())
		return
	}

	var state model
	state.fromAPI(ctx, created, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *productRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := r.client.ProductRule(ctx, state.ID.ValueInt64())
	if err != nil {
		if client.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Could not read SecObserve product rule", err.Error())
		return
	}

	var refreshed model
	refreshed.fromAPI(ctx, rule, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *productRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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

	updated, err := r.client.UpdateProductRule(
		ctx, state.ID.ValueInt64(), plan.Product.ValueInt64(), plan.RuleFields.ToAPI(ctx, &resp.Diagnostics))
	if err != nil {
		resp.Diagnostics.AddError("Could not update SecObserve product rule", err.Error())
		return
	}

	var refreshed model
	refreshed.fromAPI(ctx, updated, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *productRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteProductRule(ctx, state.ID.ValueInt64()); err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError("Could not delete SecObserve product rule", err.Error())
	}
}

// ImportState takes "<product id>/<rule name>": product rule names are
// genuinely unique per product, unlike general rule names.
func (r *productRuleResource) ImportState(
	ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	productID, name, found := strings.Cut(req.ID, "/")
	if !found {
		resp.Diagnostics.AddError(
			"Invalid import id",
			"Import a product rule as \"<product id>/<rule name>\", for example \"12/Ignore dev dependencies\".",
		)
		return
	}

	parsedProduct, err := strconv.ParseInt(productID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid import id",
			"The product part of \"<product id>/<rule name>\" must be numeric: "+err.Error(),
		)
		return
	}

	rule, err := r.client.ProductRuleByName(ctx, parsedProduct, name)
	if err != nil {
		resp.Diagnostics.AddError("Could not import SecObserve product rule", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), rule.ID)...)
}
