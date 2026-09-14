package tfutil

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The conversions below are deliberately explicit about null.
//
// SecObserve distinguishes three states for many attributes: absent (inherit
// from the product group or the global settings), null (explicitly unset) and
// a value. Requests therefore always carry every managed field, with Go nil
// mapping to JSON null, rather than relying on omitempty.

// StringPtr converts a Terraform string to *string, mapping null and unknown
// to nil.
func StringPtr(value types.String) *string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	converted := value.ValueString()
	return &converted
}

// StringValue converts a Terraform string to a plain string, mapping null and
// unknown to "". Used for the many SecObserve CharFields that are
// blank=True without null=True and therefore reject a JSON null.
func StringValue(value types.String) string {
	if value.IsNull() || value.IsUnknown() {
		return ""
	}
	return value.ValueString()
}

// BoolPtr converts a Terraform bool to *bool, mapping null and unknown to nil.
// Used for the tri-state flags where null means "inherit".
func BoolPtr(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	converted := value.ValueBool()
	return &converted
}

// BoolValue converts a Terraform bool to a plain bool, mapping null and
// unknown to false.
func BoolValue(value types.Bool) bool {
	if value.IsNull() || value.IsUnknown() {
		return false
	}
	return value.ValueBool()
}

// Int64Ptr converts a Terraform int64 to *int64, mapping null and unknown to
// nil.
func Int64Ptr(value types.Int64) *int64 {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	converted := value.ValueInt64()
	return &converted
}

// String converts a *string from an API response into a Terraform string,
// mapping nil to null.
func String(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

// Bool converts a *bool from an API response into a Terraform bool, mapping
// nil to null.
func Bool(value *bool) types.Bool {
	if value == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*value)
}

// Int64 converts a *int64 from an API response into a Terraform int64,
// mapping nil to null.
func Int64(value *int64) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*value)
}

// StringSet converts a Terraform set of strings into a []string, mapping null
// and unknown to nil.
//
// Sets rather than lists are used for SecObserve's many-to-many attributes:
// the API does not guarantee an order, so a list would produce spurious diffs.
func StringSet(ctx context.Context, value types.Set, diags *diag.Diagnostics) []string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	var converted []string
	diags.Append(value.ElementsAs(ctx, &converted, false)...)
	return converted
}

// StringSetValue converts a []string from an API response into a Terraform
// set. A nil slice becomes an empty set, not null: SecObserve returns [] for
// unset collection attributes, and mapping that to null would show as a diff.
func StringSetValue(ctx context.Context, values []string, diags *diag.Diagnostics) types.Set {
	if values == nil {
		values = []string{}
	}
	converted, conversionDiags := types.SetValueFrom(ctx, types.StringType, values)
	diags.Append(conversionDiags...)
	return converted
}

// Int64Set converts a Terraform set of int64 into a []int64, mapping null and
// unknown to nil.
func Int64Set(ctx context.Context, value types.Set, diags *diag.Diagnostics) []int64 {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	var converted []int64
	diags.Append(value.ElementsAs(ctx, &converted, false)...)
	return converted
}

// Int64SetValue converts a []int64 from an API response into a Terraform set.
func Int64SetValue(ctx context.Context, values []int64, diags *diag.Diagnostics) types.Set {
	if values == nil {
		values = []int64{}
	}
	converted, conversionDiags := types.SetValueFrom(ctx, types.Int64Type, values)
	diags.Append(conversionDiags...)
	return converted
}
