package product_group

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
func newModelV0(name string) modelV0 {
	var m modelV0
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
	prior := newModelV0("group")
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

	if upgraded.SecurityGate.Block == nil {
		t.Fatal("SecurityGate.Block = nil, want non-nil")
	}
	if got := upgraded.SecurityGate.Block.Critical.ValueInt64(); got != 5 {
		t.Errorf("Block.Critical = %d, want 5", got)
	}
	if !upgraded.SecurityGate.Block.Active.ValueBool() {
		t.Error("Block.Active not carried forward")
	}
	if upgraded.BranchHousekeeping.Block == nil {
		t.Fatal("BranchHousekeeping.Block = nil, want non-nil")
	}
	if got := upgraded.BranchHousekeeping.Block.KeepInactiveDays.ValueInt64(); got != 60 {
		t.Errorf("Block.KeepInactiveDays = %d, want 60", got)
	}
	if got := upgraded.BranchHousekeeping.Block.ExemptBranches.ValueString(); got != "^main$" {
		t.Errorf("Block.ExemptBranches = %q, want \"^main$\"", got)
	}
	if !upgraded.BranchHousekeeping.Block.Active.ValueBool() {
		t.Error("BranchHousekeeping.Block.Active not carried forward")
	}
}

// A schema-version-0 state that never configured any threshold/housekeeping
// field -- the common case -- must not synthesize a nested block out of
// nulls.
func TestUpgradeStateV0LeavesUnconfiguredBlocksNil(t *testing.T) {
	prior := newModelV0("group")

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

	if upgraded.SecurityGate.Block != nil {
		t.Errorf("SecurityGate.Block = %+v, want nil", upgraded.SecurityGate.Block)
	}
	if upgraded.BranchHousekeeping.Block != nil {
		t.Errorf("BranchHousekeeping.Block = %+v, want nil", upgraded.BranchHousekeeping.Block)
	}
}

// newModelV1 is the v1 equivalent of newModelV0.
func newModelV1(name string) modelV1 {
	var m modelV1
	m.Name = types.StringValue(name)
	m.AssessmentApprovers = types.SetNull(types.Int64Type)
	m.AssessmentApproverAuthorizationGroups = types.SetNull(types.Int64Type)
	m.ObservationNotificationStatusList = types.SetNull(types.StringType)
	return m
}

func priorStateV1(t *testing.T, prior modelV1) tfsdk.State {
	t.Helper()

	ctx := context.Background()
	priorSchema := priorSchemaV1()

	state := tfsdk.State{
		Raw:    tftypes.NewValue(priorSchema.Type().TerraformType(ctx), nil),
		Schema: *priorSchema,
	}
	if diags := state.Set(ctx, prior); diags.HasError() {
		t.Fatalf("setting prior state: %v", diags.Errors())
	}
	return state
}

// A schema-version-1 state (security_gate_active top-level, security_gate a
// thresholds-only nested attribute) folds active into the new block's active
// field alongside the thresholds.
func TestUpgradeStateV1MovesExplicitValues(t *testing.T) {
	prior := newModelV1("group")
	prior.SecurityGateActive = types.BoolValue(true)
	prior.Thresholds = &securityGateThresholdsV1{
		Critical: types.Int64Value(5),
		High:     types.Int64Null(),
		Medium:   types.Int64Null(),
		Low:      types.Int64Null(),
		None:     types.Int64Null(),
		Unknown:  types.Int64Null(),
	}
	prior.RepositoryBranchHousekeepingActive = types.BoolValue(true)
	prior.branchHousekeepingV1.Settings = &branchHousekeepingSettingsV1{
		KeepInactiveDays: types.Int64Value(60),
		ExemptBranches:   types.StringValue("^main$"),
	}

	req := resource.UpgradeStateRequest{State: statePtr(priorStateV1(t, prior))}
	resp := &resource.UpgradeStateResponse{}
	upgradeStateV1(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("upgrade: %v", resp.Diagnostics.Errors())
	}

	var upgraded model
	if diags := resp.State.Get(context.Background(), &upgraded); diags.HasError() {
		t.Fatalf("reading upgraded state: %v", diags.Errors())
	}

	if upgraded.SecurityGate.Block == nil {
		t.Fatal("SecurityGate.Block = nil, want non-nil")
	}
	if got := upgraded.SecurityGate.Block.Critical.ValueInt64(); got != 5 {
		t.Errorf("Block.Critical = %d, want 5", got)
	}
	if !upgraded.SecurityGate.Block.Active.ValueBool() {
		t.Error("Block.Active not carried forward")
	}
	if upgraded.BranchHousekeeping.Block == nil {
		t.Fatal("BranchHousekeeping.Block = nil, want non-nil")
	}
	if got := upgraded.BranchHousekeeping.Block.KeepInactiveDays.ValueInt64(); got != 60 {
		t.Errorf("Block.KeepInactiveDays = %d, want 60", got)
	}
	if !upgraded.BranchHousekeeping.Block.Active.ValueBool() {
		t.Error("BranchHousekeeping.Block.Active not carried forward")
	}
}

// A schema-version-1 state that never configured either block must not
// synthesize one out of nulls.
func TestUpgradeStateV1LeavesUnconfiguredBlocksNil(t *testing.T) {
	prior := newModelV1("group")

	req := resource.UpgradeStateRequest{State: statePtr(priorStateV1(t, prior))}
	resp := &resource.UpgradeStateResponse{}
	upgradeStateV1(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("upgrade: %v", resp.Diagnostics.Errors())
	}

	var upgraded model
	if diags := resp.State.Get(context.Background(), &upgraded); diags.HasError() {
		t.Fatalf("reading upgraded state: %v", diags.Errors())
	}

	if upgraded.SecurityGate.Block != nil {
		t.Errorf("SecurityGate.Block = %+v, want nil", upgraded.SecurityGate.Block)
	}
	if upgraded.BranchHousekeeping.Block != nil {
		t.Errorf("BranchHousekeeping.Block = %+v, want nil", upgraded.BranchHousekeeping.Block)
	}
}

func statePtr(s tfsdk.State) *tfsdk.State { return &s }
