package product_group

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/schemacommon"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

// model mirrors the resource schema. The embedded blocks are flattened by the
// framework, so their fields are addressed as if declared here.
type model struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`

	schemacommon.SecurityGate
	schemacommon.BranchHousekeeping
	schemacommon.Notifications
	schemacommon.Approvals
	schemacommon.Approvers
	schemacommon.RiskAcceptance
	schemacommon.LicensePolicy
	schemacommon.BranchPropagation
}

func (m model) toRequest(ctx context.Context, diags *diag.Diagnostics) client.ProductGroupRequest {
	return client.ProductGroupRequest{
		Name:                     m.Name.ValueString(),
		Description:              tfutil.StringValue(m.Description),
		SecurityGateFields:       m.SecurityGate.ToAPI(),
		BranchHousekeepingFields: m.BranchHousekeeping.ToAPI(),
		NotificationFields:       m.Notifications.ToAPI(ctx, diags),
		ApprovalFields:           m.Approvals.ToAPI(),
		ApproverFields:           m.Approvers.ToAPI(ctx, diags),
		RiskAcceptanceFields:     m.RiskAcceptance.ToAPI(),
		LicensePolicyFields:      m.LicensePolicy.ToAPI(),
		BranchPropagationFields:  m.BranchPropagation.ToAPI(ctx, diags),
	}
}

func (m *model) fromAPI(ctx context.Context, group client.Product, diags *diag.Diagnostics) {
	m.ID = types.Int64Value(group.ID)
	m.Name = types.StringValue(group.Name)
	m.Description = types.StringValue(group.Description)

	m.SecurityGate.FromAPI(group.SecurityGateFields)
	m.BranchHousekeeping.FromAPI(group.BranchHousekeepingFields)
	m.Notifications.FromAPI(ctx, group.NotificationFields, diags)
	m.Approvals.FromAPI(group.ApprovalFields)
	m.Approvers.FromAPI(ctx, group.ApproverFields, diags)
	m.RiskAcceptance.FromAPI(group.RiskAcceptanceFields)
	m.LicensePolicy.FromAPI(group.LicensePolicyFields)
	m.BranchPropagation.FromAPI(ctx, group.BranchPropagationFields, diags)
}
