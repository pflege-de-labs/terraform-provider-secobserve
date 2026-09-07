package schemacommon

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The two modifiers below exist for the same reason.
//
// SecObserve derives several attributes from a sibling: the security gate
// thresholds from security_gate_active, the housekeeping retention from
// repository_branch_housekeeping_active, the issue tracker base URL from
// issue_tracker_type. Whenever that sibling changes, the server recomputes the
// dependent value -- filling it in from the instance settings, or clearing it.
//
// Left alone, the framework carries the prior state value into the plan for an
// unset Optional+Computed attribute, and Terraform then rejects the server's
// recomputed answer with "provider produced inconsistent result". Planning
// unknown unconditionally would be worse: every plan would show a diff for
// attributes nobody touched. Restricting it to a changing sibling keeps
// steady-state plans empty and still lets the server decide when it matters.

// Int64UnknownWhenBoolSiblingChanges plans an int64 attribute as unknown while
// it is unset and the named bool attribute is changing.
func Int64UnknownWhenBoolSiblingChanges(sibling string) planmodifier.Int64 {
	return int64UnknownWhenBoolSiblingChanges{sibling: sibling}
}

type int64UnknownWhenBoolSiblingChanges struct {
	sibling string
}

func (m int64UnknownWhenBoolSiblingChanges) Description(context.Context) string {
	return "planned as unknown while unset and " + m.sibling + " changes, because SecObserve recomputes it"
}

func (m int64UnknownWhenBoolSiblingChanges) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m int64UnknownWhenBoolSiblingChanges) PlanModifyInt64(
	ctx context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response,
) {
	// A destroy plan is null throughout; on create the attribute is unknown
	// already; a configured value must be honoured as written.
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() || !req.ConfigValue.IsNull() {
		return
	}

	var fromState, fromPlan types.Bool
	if diags := req.State.GetAttribute(ctx, path.Root(m.sibling), &fromState); diags.HasError() {
		return
	}
	if diags := req.Plan.GetAttribute(ctx, path.Root(m.sibling), &fromPlan); diags.HasError() {
		return
	}

	if !fromState.Equal(fromPlan) {
		resp.PlanValue = types.Int64Unknown()
	}
}

// StringUnknownWhenStringSiblingChanges plans a string attribute as unknown
// while it is unset and the named string attribute is changing.
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
