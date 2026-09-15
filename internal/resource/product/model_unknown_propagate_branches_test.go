package product

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/schemacommon"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tftest"
)

// Reported: a config expression whose result depends on a not-yet-known
// upstream value -- e.g.
//
//	propagate_branches = each.value.propagate_branches == null ? null : [
//	  for b in each.value.propagate_branches : { propagate_to = b }
//	]
//
// with each.value coming from a for_each over something not fully known at
// plan time -- makes the whole conditional's result unknown, not merely
// null. Before this fix, PropagateBranches was a plain Go slice
// ([]PropagateBranchModel), and the framework's reflection-based Get cannot
// populate a plain slice field from an unknown list: "Received unknown
// value, however the target type cannot handle unknown values... Suggested
// Type: basetypes.ListValue". Get must succeed regardless of whether
// propagate_branches ends up null, unknown or populated.
func TestModelHandlesUnknownPropagateBranches(t *testing.T) {
	ctx := context.Background()
	resourceSchema := tftest.ResourceSchema(t, New())

	state := tfsdk.State{
		Raw:    tftypes.NewValue(resourceSchema.Type().TerraformType(ctx), nil),
		Schema: resourceSchema,
	}
	if diags := state.SetAttribute(
		ctx, path.Root("propagate_branches"), types.ListUnknown(schemacommon.PropagateBranchObjectType),
	); diags.HasError() {
		t.Fatalf("setting propagate_branches to unknown: %v", diags.Errors())
	}

	var got model
	if diags := state.Get(ctx, &got); diags.HasError() {
		t.Fatalf("Get with an unknown propagate_branches: %v", diags.Errors())
	}
	if !got.PropagateBranches.IsUnknown() {
		t.Errorf("PropagateBranches = %#v, want unknown", got.PropagateBranches)
	}
}
