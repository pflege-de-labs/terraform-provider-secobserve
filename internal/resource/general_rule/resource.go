// Package general_rule implements the secobserve_general_rule resource.
package general_rule //nolint:revive // the package name mirrors the Terraform resource name

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jabbrwcky/terraform-provider-secobserve/internal/client"
	"github.com/jabbrwcky/terraform-provider-secobserve/internal/schemacommon"
	"github.com/jabbrwcky/terraform-provider-secobserve/internal/tfutil"
)

var (
	_ resource.ResourceWithConfigure      = (*generalRuleResource)(nil)
	_ resource.ResourceWithImportState    = (*generalRuleResource)(nil)
	_ resource.ResourceWithValidateConfig = (*generalRuleResource)(nil)
)

// New returns the secobserve_general_rule resource.
func New() resource.Resource {
	return &generalRuleResource{}
}

type generalRuleResource struct {
	client *client.Client
}

type model struct {
	ID types.Int64 `tfsdk:"id"`

	schemacommon.RuleFields
	schemacommon.RuleApproval
}

func (r *generalRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_general_rule"
}

func (r *generalRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = tfutil.ResourceClient(req, resp)
}

func (r *generalRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Computed:            true,
			PlanModifiers:       []planmodifier.Int64{schemacommon.Int64UseStateForUnknown()},
			MarkdownDescription: "Numeric id of the rule.",
		},
	}
	schemacommon.AddRuleFields(attributes)
	schemacommon.AddRuleApproval(attributes)

	resp.Schema = schema.Schema{
		MarkdownDescription: "A rule applied to observations across every product on the instance.\n\n" +
			"~> Writing this resource requires the provider's token to belong to a **superuser** -- " +
			"SecObserve rejects a general rule write from anyone else outright.\n\n" +
			"~> **Names are not reliably unique.** SecObserve's uniqueness constraint on a rule is " +
			"`(product, name)`, and a general rule always has a null product; PostgreSQL treats two NULLs " +
			"as distinct, so nothing stops two general rules sharing a name. Importing by name fails with " +
			"an ambiguity error rather than guessing which one was meant -- import by id instead if that " +
			"happens.\n\n" +
			"~> **Approval is out of band.** A rule created while general rule approval is required in the " +
			"instance settings starts in `approval_status = \"Needs approval\"`, and every apply that " +
			"touches this resource resets it back to that state. SecObserve also rejects self-approval, so " +
			"the identity Terraform authenticates as can never approve a rule it just wrote -- somebody " +
			"else has to do it through the UI or API.",
		Attributes: attributes,
	}
}

func (r *generalRuleResource) ValidateConfig(
	ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse,
) {
	var config model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	config.RuleFields.ValidateRuleFields(&resp.Diagnostics)
}

func (m *model) fromAPI(rule client.Rule) {
	m.ID = types.Int64Value(rule.ID)
	m.RuleFields.FromAPI(rule.RuleFields)
	m.RuleApproval.FromAPI(rule)
}

func (r *generalRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateGeneralRule(ctx, plan.RuleFields.ToAPI())
	if err != nil {
		resp.Diagnostics.AddError("Could not create SecObserve general rule", err.Error())
		return
	}

	var state model
	state.fromAPI(created)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *generalRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := r.client.GeneralRule(ctx, state.ID.ValueInt64())
	if err != nil {
		if client.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Could not read SecObserve general rule", err.Error())
		return
	}

	var refreshed model
	refreshed.fromAPI(rule)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *generalRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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

	updated, err := r.client.UpdateGeneralRule(ctx, state.ID.ValueInt64(), plan.RuleFields.ToAPI())
	if err != nil {
		resp.Diagnostics.AddError("Could not update SecObserve general rule", err.Error())
		return
	}

	var refreshed model
	refreshed.fromAPI(updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *generalRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteGeneralRule(ctx, state.ID.ValueInt64()); err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError("Could not delete SecObserve general rule", err.Error())
	}
}

// ImportState accepts either the numeric id or the rule's name. Name import
// fails with an ambiguity error rather than an arbitrary pick when more than
// one general rule shares the name.
func (r *generalRuleResource) ImportState(
	ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	if id, err := strconv.ParseInt(req.ID, 10, 64); err == nil {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
		return
	}

	rule, err := r.client.GeneralRuleByName(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Could not import SecObserve general rule",
			"Pass either the numeric id or the exact name. General rule names are not guaranteed unique; "+
				"if this reports ambiguity, import by id instead.\n\nUnderlying error: "+err.Error(),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), rule.ID)...)
}
