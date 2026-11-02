package tfutil

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// IDIdentitySchema is the identity every listed resource shares: SecObserve's
// numeric primary key. See docs/design/list-resources.md.
func IDIdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.Int64Attribute{
				RequiredForImport: true,
				Description:       "Numeric id of the object in SecObserve.",
			},
		},
	}
}

// SetIdentityID writes the id into a resource identity.
func SetIdentityID(ctx context.Context, identity *tfsdk.ResourceIdentity, id types.Int64) diag.Diagnostics {
	if identity == nil {
		return nil
	}
	return identity.SetAttribute(ctx, path.Root("id"), id)
}

// ImportByIdentity handles an `import { identity = { id = ... } }` block and
// reports whether it did. String import ids are left to the caller.
//
// resource.ImportStatePassthroughWithIdentity cannot be used: it only
// handles string-typed identity and state attributes.
func ImportByIdentity(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) bool {
	if req.ID != "" || req.Identity == nil {
		return false
	}

	var id types.Int64
	resp.Diagnostics.Append(req.Identity.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return true
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
	return true
}
