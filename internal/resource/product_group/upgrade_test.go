package product_group

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/schemacommon"
)

// testSettingsClient stubs GET /api/settings/1/ with the given defaults, for
// upgradeStateV0's dropIfDefault comparison. Defaults chosen far from any
// value a test sets explicitly, so "kept" and "dropped" are unambiguous.
func testSettingsClient(t *testing.T, settings client.Settings) *client.Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(settings)
	}))
	t.Cleanup(server.Close)

	c, err := client.New(client.Config{BaseURL: server.URL, APIToken: "test"})
	if err != nil {
		t.Fatalf("building test client: %v", err)
	}
	return c
}

func defaultTestSettings() client.Settings {
	return client.Settings{
		SecurityGateThresholdCritical:      1000,
		SecurityGateThresholdHigh:          1000,
		SecurityGateThresholdMedium:        1000,
		SecurityGateThresholdLow:           1000,
		SecurityGateThresholdNone:          1000,
		SecurityGateThresholdUnknown:       1000,
		BranchHousekeepingKeepInactiveDays: 1000,
		BranchHousekeepingExemptBranches:   "^instance-default-pattern$",
	}
}

// newModelV0 builds a minimal but schema-valid modelV0: the Set-typed
// attributes need an explicitly typed null, not the Go zero value, or
// tfsdk.State.Set rejects it.
func newModelV0(name string) modelV0 {
	var m modelV0
	m.Name = types.StringValue(name)
	m.AssessmentApprovers = types.SetNull(types.Int64Type)
	m.AssessmentApproverAuthorizationGroups = types.SetNull(types.Int64Type)
	m.ObservationNotificationStatusList = types.SetNull(types.StringType)
	m.PropagateBranches = types.ListNull(schemacommon.PropagateBranchObjectType)
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

	r := &productGroupResource{client: testSettingsClient(t, defaultTestSettings())}
	req := resource.UpgradeStateRequest{State: statePtr(priorState(t, prior))}
	resp := &resource.UpgradeStateResponse{}
	r.upgradeStateV0(context.Background(), req, resp)
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

	r := &productGroupResource{client: testSettingsClient(t, defaultTestSettings())}
	req := resource.UpgradeStateRequest{State: statePtr(priorState(t, prior))}
	resp := &resource.UpgradeStateResponse{}
	r.upgradeStateV0(context.Background(), req, resp)
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

// The exact scenario reported: a v0 state where every field was
// server-filled to the instance-wide default (active=true, every threshold
// equal to the current default, keep_inactive_days/exempt_branches equal to
// the current default) must upgrade to no block at all -- not a block full
// of values nobody configured. Values that differ from the current default
// are still carried forward untouched.
func TestUpgradeStateV0DropsValuesMatchingCurrentDefaults(t *testing.T) {
	settings := defaultTestSettings()

	prior := newModelV0("group")
	prior.SecurityGateActive = types.BoolValue(true)
	prior.SecurityGateThresholdCritical = types.Int64Value(2) // differs -- kept
	prior.SecurityGateThresholdHigh = types.Int64Value(settings.SecurityGateThresholdHigh)
	prior.SecurityGateThresholdMedium = types.Int64Value(settings.SecurityGateThresholdMedium)
	prior.SecurityGateThresholdLow = types.Int64Value(settings.SecurityGateThresholdLow)
	prior.SecurityGateThresholdNone = types.Int64Value(settings.SecurityGateThresholdNone)
	prior.SecurityGateThresholdUnknown = types.Int64Value(settings.SecurityGateThresholdUnknown)
	prior.RepositoryBranchHousekeepingActive = types.BoolValue(true)
	prior.RepositoryBranchHousekeepingKeepInactiveDays = types.Int64Value(settings.BranchHousekeepingKeepInactiveDays)
	prior.RepositoryBranchHousekeepingExemptBranches = types.StringValue(settings.BranchHousekeepingExemptBranches)

	r := &productGroupResource{client: testSettingsClient(t, settings)}
	req := resource.UpgradeStateRequest{State: statePtr(priorState(t, prior))}
	resp := &resource.UpgradeStateResponse{}
	r.upgradeStateV0(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("upgrade: %v", resp.Diagnostics.Errors())
	}

	var upgraded model
	if diags := resp.State.Get(context.Background(), &upgraded); diags.HasError() {
		t.Fatalf("reading upgraded state: %v", diags.Errors())
	}

	if upgraded.SecurityGate.Block == nil {
		t.Fatal("SecurityGate.Block = nil, want non-nil (active + a non-default threshold survive)")
	}
	if got := upgraded.SecurityGate.Block.Critical.ValueInt64(); got != 2 {
		t.Errorf("Critical = %d, want 2 (non-default, must survive)", got)
	}
	if !upgraded.SecurityGate.Block.High.IsNull() {
		t.Errorf("High = %v, want null (matched current default, must be dropped)", upgraded.SecurityGate.Block.High)
	}
	if !upgraded.SecurityGate.Block.Medium.IsNull() || !upgraded.SecurityGate.Block.Low.IsNull() ||
		!upgraded.SecurityGate.Block.None.IsNull() || !upgraded.SecurityGate.Block.Unknown.IsNull() {
		t.Error("default-matching thresholds were not dropped")
	}
	if upgraded.BranchHousekeeping.Block == nil {
		t.Fatal("BranchHousekeeping.Block = nil, want non-nil (active survives)")
	}
	if !upgraded.BranchHousekeeping.Block.KeepInactiveDays.IsNull() {
		t.Errorf("KeepInactiveDays = %v, want null (matched current default, must be dropped)",
			upgraded.BranchHousekeeping.Block.KeepInactiveDays)
	}
	if got := upgraded.BranchHousekeeping.Block.ExemptBranches.ValueString(); got != "" {
		t.Errorf("ExemptBranches = %q, want \"\" (matched current default, must be dropped)", got)
	}
}

// newModelV1 is the v1 equivalent of newModelV0.
func newModelV1(name string) modelV1 {
	var m modelV1
	m.Name = types.StringValue(name)
	m.AssessmentApprovers = types.SetNull(types.Int64Type)
	m.AssessmentApproverAuthorizationGroups = types.SetNull(types.Int64Type)
	m.ObservationNotificationStatusList = types.SetNull(types.StringType)
	m.PropagateBranches = types.ListNull(schemacommon.PropagateBranchObjectType)
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
