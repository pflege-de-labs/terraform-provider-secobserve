package product

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// newModelV0 builds a minimal but schema-valid modelV0: the Set-typed
// attributes need an explicitly typed null, not the Go zero value, or
// tfsdk.State.Set rejects it.
func newModelV0(id int64, name string) modelV0 {
	var m modelV0
	m.ID = types.Int64Value(id)
	m.Name = types.StringValue(name)
	m.AssessmentApprovers = types.SetNull(types.Int64Type)
	m.AssessmentApproverAuthorizationGroups = types.SetNull(types.Int64Type)
	m.ObservationNotificationStatusList = types.SetNull(types.StringType)
	return m
}

func priorState(t *testing.T, prior modelV0) tfsdk.State {
	t.Helper()

	ctx := context.Background()
	priorSchema := priorSchemaV0()

	state := tfsdk.State{
		Raw:    tftypes.NewValue(priorSchema.Type().TerraformType(ctx), nil),
		Schema: *priorSchema,
	}
	if diags := state.Set(ctx, prior); diags.HasError() {
		t.Fatalf("setting prior state: %v", diags.Errors())
	}
	return state
}

// A schema-version-0 state with explicit thresholds and housekeeping days
// moves them into the new nested blocks, unchanged.
func TestUpgradeStateV0MovesExplicitValues(t *testing.T) {
	prior := newModelV0(1, "product")
	prior.SecurityGateActive = types.BoolValue(true)
	prior.SecurityGateThresholdCritical = types.Int64Value(5)
	prior.SecurityGateThresholdHigh = types.Int64Null()
	prior.SecurityGateThresholdMedium = types.Int64Null()
	prior.SecurityGateThresholdLow = types.Int64Null()
	prior.SecurityGateThresholdNone = types.Int64Null()
	prior.SecurityGateThresholdUnknown = types.Int64Null()
	prior.RepositoryBranchHousekeepingActive = types.BoolValue(true)
	prior.RepositoryBranchHousekeepingKeepInactiveDays = types.Int64Value(60)
	prior.RepositoryBranchHousekeepingExemptBranches = types.StringValue("^main$")

	req := resource.UpgradeStateRequest{State: statePtr(priorState(t, prior))}
	resp := &resource.UpgradeStateResponse{}
	upgradeStateV0(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("upgrade: %v", resp.Diagnostics.Errors())
	}

	var upgraded model
	if diags := resp.State.Get(context.Background(), &upgraded); diags.HasError() {
		t.Fatalf("reading upgraded state: %v", diags.Errors())
	}

	if upgraded.SecurityGate.Thresholds == nil {
		t.Fatal("Thresholds = nil, want non-nil")
	}
	if got := upgraded.SecurityGate.Thresholds.Critical.ValueInt64(); got != 5 {
		t.Errorf("Thresholds.Critical = %d, want 5", got)
	}
	if upgraded.BranchHousekeeping.Settings == nil {
		t.Fatal("Settings = nil, want non-nil")
	}
	if got := upgraded.BranchHousekeeping.Settings.KeepInactiveDays.ValueInt64(); got != 60 {
		t.Errorf("Settings.KeepInactiveDays = %d, want 60", got)
	}
	if got := upgraded.BranchHousekeeping.Settings.ExemptBranches.ValueString(); got != "^main$" {
		t.Errorf("Settings.ExemptBranches = %q, want \"^main$\"", got)
	}
	if !upgraded.SecurityGateActive.ValueBool() {
		t.Error("SecurityGateActive not carried forward")
	}
}

// A schema-version-0 state that never configured any threshold/housekeeping
// field -- the common case -- must not synthesize a nested block out of
// nulls.
func TestUpgradeStateV0LeavesUnconfiguredBlocksNil(t *testing.T) {
	prior := newModelV0(2, "product")

	req := resource.UpgradeStateRequest{State: statePtr(priorState(t, prior))}
	resp := &resource.UpgradeStateResponse{}
	upgradeStateV0(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("upgrade: %v", resp.Diagnostics.Errors())
	}

	var upgraded model
	if diags := resp.State.Get(context.Background(), &upgraded); diags.HasError() {
		t.Fatalf("reading upgraded state: %v", diags.Errors())
	}

	if upgraded.SecurityGate.Thresholds != nil {
		t.Errorf("Thresholds = %+v, want nil", upgraded.SecurityGate.Thresholds)
	}
	if upgraded.BranchHousekeeping.Settings != nil {
		t.Errorf("Settings = %+v, want nil", upgraded.BranchHousekeeping.Settings)
	}
}

func statePtr(s tfsdk.State) *tfsdk.State { return &s }
