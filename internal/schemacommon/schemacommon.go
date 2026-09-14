// Package schemacommon holds the attribute blocks shared by more than one
// resource -- products and product groups, and general rules and product
// rules. Each block pairs a schema contribution with the model struct that
// resources embed by value, so the two halves cannot drift apart.
//
// Embedding relies on the framework flattening value-embedded structs into the
// object and addressing their promoted tfsdk fields as if declared directly.
// The embedded field itself must carry no tfsdk tag.
package schemacommon

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Int64UseStateForUnknown keeps a computed id stable across updates.
func Int64UseStateForUnknown() planmodifier.Int64 {
	return int64planmodifier.UseStateForUnknown()
}

// StringUseStateForUnknown keeps a computed string stable across updates.
func StringUseStateForUnknown() planmodifier.String {
	return stringplanmodifier.UseStateForUnknown()
}

// EmptyInt64Set is the default for the many-to-many id attributes: SecObserve
// answers with [] rather than null, so null would read as a diff.
func EmptyInt64Set() types.Set {
	return types.SetValueMust(types.Int64Type, []attr.Value{})
}

// EmptyStringSet is EmptyInt64Set for string collections.
func EmptyStringSet() types.Set {
	return types.SetValueMust(types.StringType, []attr.Value{})
}
