package schemacommon

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// StringUnknownWhenStringSiblingChanges plans a string attribute as unknown
// while it is unset and the named string attribute is changing.
//
// SecObserve derives the issue tracker base URL from issue_tracker_type:
// whenever the type changes, the server recomputes the base URL -- filling it
// in from a per-type default, or leaving it as submitted.
//
// Left alone, the framework carries the prior state value into the plan for
// an unset Optional+Computed attribute, and Terraform then rejects the
// server's recomputed answer with "provider produced inconsistent result".
// Planning unknown unconditionally would be worse: every plan would show a
// diff for an attribute nobody touched. Restricting it to a changing sibling
// keeps steady-state plans empty and still lets the server decide when it
// matters.
func StringUnknownWhenStringSiblingChanges(sibling string) planmodifier.String {
	return stringUnknownWhenStringSiblingChanges{sibling: sibling}
}

type stringUnknownWhenStringSiblingChanges struct {
	sibling string
}

func (m stringUnknownWhenStringSiblingChanges) Description(context.Context) string {
	return "planned as unknown while unset and " + m.sibling + " changes, because SecObserve recomputes it"
}

func (m stringUnknownWhenStringSiblingChanges) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m stringUnknownWhenStringSiblingChanges) PlanModifyString(
	ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse,
) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() || !req.ConfigValue.IsNull() {
		return
	}

	var fromState, fromPlan types.String
	if diags := req.State.GetAttribute(ctx, path.Root(m.sibling), &fromState); diags.HasError() {
		return
	}
	if diags := req.Plan.GetAttribute(ctx, path.Root(m.sibling), &fromPlan); diags.HasError() {
		return
	}

	if !fromState.Equal(fromPlan) {
		resp.PlanValue = types.StringUnknown()
	}
}
