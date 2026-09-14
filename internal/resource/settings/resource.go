// Package settings implements the secobserve_settings resource: the
// instance-wide configuration singleton.
package settings

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/jabbrwcky/terraform-provider-secobserve/internal/client"
	"github.com/jabbrwcky/terraform-provider-secobserve/internal/tfutil"
)

var (
	_ resource.ResourceWithConfigure      = (*settingsResource)(nil)
	_ resource.ResourceWithValidateConfig = (*settingsResource)(nil)
)

// New returns the secobserve_settings resource.
func New() resource.Resource {
	return &settingsResource{}
}

type settingsResource struct {
	client *client.Client
}

func (r *settingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_settings"
}

func (r *settingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = tfutil.ResourceClient(req, resp)
}

// Create writes the settings singleton, which always already exists
// server-side (Settings.load() returns a default instance even before one
// has ever been saved) -- there is no POST on this endpoint, so Create and
// Update both PATCH.
func (r *settingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := plan.toRequest(ctx, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateSettings(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("Could not update SecObserve settings", err.Error())
		return
	}

	var state model
	state.fromAPI(ctx, updated, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *settingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	settings, err := r.client.GetSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Could not read SecObserve settings", err.Error())
		return
	}

	var state model
	state.fromAPI(ctx, settings, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *settingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := plan.toRequest(ctx, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateSettings(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("Could not update SecObserve settings", err.Error())
		return
	}

	var refreshed model
	refreshed.fromAPI(ctx, updated, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

// Delete only removes the resource from Terraform state. The singleton
// itself is never deleted -- there is no DELETE on this endpoint, and
// resetting every setting to its factory default on a `terraform destroy`
// would be a surprising, destructive side effect for something that reads as
// "stop managing this."
func (r *settingsResource) Delete(_ context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning(
		"SecObserve settings were not reset",
		"secobserve_settings has no delete on the API: the instance settings singleton always exists and "+
			"cannot be removed. This only stops Terraform from managing it; every value stays exactly as "+
			"it was.",
	)
}

// ImportState always targets the singleton at id 1, regardless of the id
// given -- there being only one settings record, any id is accepted.
func (r *settingsResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	settings, err := r.client.GetSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Could not import SecObserve settings", err.Error())
		return
	}

	var state model
	state.fromAPI(ctx, settings, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
