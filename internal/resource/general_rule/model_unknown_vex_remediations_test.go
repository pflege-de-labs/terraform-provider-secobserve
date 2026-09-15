package general_rule

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

// Same bug class as product's TestModelHandlesUnknownPropagateBranches:
// new_vex_remediations was a plain Go slice ([]VEXRemediationModel), and the
// framework's reflection-based Get cannot populate a plain slice field from
// an unknown list -- a config expression whose result depends on a
// not-yet-known upstream value plans this attribute as unknown, not merely
// null. Get must succeed regardless.
func TestModelHandlesUnknownNewVEXRemediations(t *testing.T) {
	ctx := context.Background()
	resourceSchema := tftest.ResourceSchema(t, New())

	state := tfsdk.State{
		Raw:    tftypes.NewValue(resourceSchema.Type().TerraformType(ctx), nil),
		Schema: resourceSchema,
	}
	if diags := state.SetAttribute(
		ctx, path.Root("new_vex_remediations"), types.ListUnknown(schemacommon.VEXRemediationObjectType),
	); diags.HasError() {
		t.Fatalf("setting new_vex_remediations to unknown: %v", diags.Errors())
	}

	var got model
	if diags := state.Get(ctx, &got); diags.HasError() {
		t.Fatalf("Get with an unknown new_vex_remediations: %v", diags.Errors())
	}
	if !got.NewVEXRemediations.IsUnknown() {
		t.Errorf("NewVEXRemediations = %#v, want unknown", got.NewVEXRemediations)
	}
}
