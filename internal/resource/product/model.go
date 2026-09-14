package product

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jabbrwcky/terraform-provider-secobserve/internal/client"
	"github.com/jabbrwcky/terraform-provider-secobserve/internal/schemacommon"
	"github.com/jabbrwcky/terraform-provider-secobserve/internal/tfutil"
)

// model mirrors the resource schema. The embedded blocks are flattened by the
// framework, so their fields are addressed as if declared here.
type model struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`

	ProductGroup     types.Int64  `tfsdk:"product_group"`
	ProductGroupName types.String `tfsdk:"product_group_name"`

	RepositoryPrefix            types.String `tfsdk:"repository_prefix"`
	RepositoryDefaultBranch     types.Int64  `tfsdk:"repository_default_branch"`
	RepositoryDefaultBranchName types.String `tfsdk:"repository_default_branch_name"`

	Purl  types.String `tfsdk:"purl"`
	CPE23 types.String `tfsdk:"cpe23"`

	ApplyGeneralRules  types.Bool `tfsdk:"apply_general_rules"`
	SecurityGatePassed types.Bool `tfsdk:"security_gate_passed"`

	schemacommon.SecurityGate
	schemacommon.BranchHousekeeping
	schemacommon.Notifications
	schemacommon.Approvals
	schemacommon.Approvers
	schemacommon.RiskAcceptance
	schemacommon.LicensePolicy
	schemacommon.BranchPropagation

	issueTracker
	scanners
}

func (m model) toRequest(ctx context.Context, diags *diag.Diagnostics) client.ProductRequest {
	return client.ProductRequest{
		Name:              m.Name.ValueString(),
		Description:       tfutil.StringValue(m.Description),
		ProductGroup:      tfutil.Int64Ptr(m.ProductGroup),
		Purl:              tfutil.StringValue(m.Purl),
		CPE23:             tfutil.StringValue(m.CPE23),
		RepositoryPrefix:  tfutil.StringValue(m.RepositoryPrefix),
		ApplyGeneralRules: tfutil.BoolValue(m.ApplyGeneralRules),

		SecurityGateFields:       m.SecurityGate.ToAPI(),
		BranchHousekeepingFields: m.BranchHousekeeping.ToAPI(),
		NotificationFields:       m.Notifications.ToAPI(ctx, diags),
		ApprovalFields:           m.Approvals.ToAPI(),
		ApproverFields:           m.Approvers.ToAPI(ctx, diags),
		RiskAcceptanceFields:     m.RiskAcceptance.ToAPI(),
		LicensePolicyFields:      m.LicensePolicy.ToAPI(),
		BranchPropagationFields:  m.BranchPropagation.ToAPI(),
		IssueTrackerFields:       m.issueTracker.toAPI(),
		ScannerFields:            m.scanners.toAPI(),
	}
}

func (m *model) fromAPI(ctx context.Context, product client.Product, diags *diag.Diagnostics) {
	m.ID = types.Int64Value(product.ID)
	m.Name = types.StringValue(product.Name)
	m.Description = types.StringValue(product.Description)

	m.ProductGroup = tfutil.Int64(product.ProductGroup)
	m.ProductGroupName = tfutil.String(product.ProductGroupName)

	m.RepositoryPrefix = types.StringValue(product.RepositoryPrefix)
	m.RepositoryDefaultBranch = tfutil.Int64(product.RepositoryDefaultBranch)
	m.RepositoryDefaultBranchName = tfutil.String(product.RepositoryDefaultBranchName)

	m.Purl = types.StringValue(product.Purl)
	m.CPE23 = types.StringValue(product.CPE23)

	m.ApplyGeneralRules = types.BoolValue(product.ApplyGeneralRules)
	m.SecurityGatePassed = tfutil.Bool(product.SecurityGatePassed)

	m.SecurityGate.FromAPI(product.SecurityGateFields)
	m.BranchHousekeeping.FromAPI(product.BranchHousekeepingFields)
	m.Notifications.FromAPI(ctx, product.NotificationFields, diags)
	m.Approvals.FromAPI(product.ApprovalFields)
	m.Approvers.FromAPI(ctx, product.ApproverFields, diags)
	m.RiskAcceptance.FromAPI(product.RiskAcceptanceFields)
	m.LicensePolicy.FromAPI(product.LicensePolicyFields)
	m.BranchPropagation.FromAPI(ctx, product.BranchPropagationFields)
	m.issueTracker.fromAPI(product.IssueTrackerFields)
	m.scanners.fromAPI(product.ScannerFields)
}
