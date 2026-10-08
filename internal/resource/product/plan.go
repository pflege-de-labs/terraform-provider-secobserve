package product

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// ModifyPlan surfaces SecObserve's approver group role check at plan time.
func (r *productResource) ModifyPlan(
	ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var productIDs []int64
	if !req.State.Raw.IsNull() {
		var state model
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		// SecObserve only validates submitted approvers; skip the lookups on unrelated changes.
		if plan.AssessmentApproverAuthorizationGroups.Equal(state.AssessmentApproverAuthorizationGroups) &&
			plan.ProductGroup.Equal(state.ProductGroup) {
			return
		}
		productIDs = append(productIDs, state.ID.ValueInt64())
	}

	if plan.ProductGroup.IsUnknown() {
		return
	}
	if !plan.ProductGroup.IsNull() {
		productIDs = append(productIDs, plan.ProductGroup.ValueInt64())
	}

	plan.Approvers.CheckApproverGroups(ctx, r.client, productIDs, &resp.Diagnostics)
}
