package schemacommon

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
)

// Regression test: rego_module coming from an unknown config value (e.g. a
// Terraform variable, always unknown during `terraform validate`) must not be
// reported as missing.
func TestValidateRuleFieldsDoesNotFlagUnknownRegoModuleAsMissing(t *testing.T) {
	r := RuleFields{
		Type:       types.StringValue(client.RuleTypeRego),
		RegoModule: types.StringUnknown(),
	}

	var diags diag.Diagnostics
	r.ValidateRuleFields(&diags)

	if diags.HasError() {
		t.Fatalf("expected no error for an unknown rego_module, got: %v", diags)
	}
}

func TestValidateRuleFieldsStillRejectsGenuinelyMissingRegoModule(t *testing.T) {
	r := RuleFields{
		Type:       types.StringValue(client.RuleTypeRego),
		RegoModule: types.StringValue(""),
	}

	var diags diag.Diagnostics
	r.ValidateRuleFields(&diags)

	if !diags.HasError() {
		t.Fatal("expected an error for a genuinely empty rego_module")
	}
}
