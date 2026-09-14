package product

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jabbrwcky/terraform-provider-secobserve/internal/client"
	"github.com/jabbrwcky/terraform-provider-secobserve/internal/schemacommon"
	"github.com/jabbrwcky/terraform-provider-secobserve/internal/tfutil"
)

// scanners is the block for the scanners SecObserve runs itself, as opposed to
// the ones that push results in through the import API. Products only.
type scanners struct {
	OSVEnabled                  types.Bool   `tfsdk:"osv_enabled"`
	OSVLinuxDistribution        types.String `tfsdk:"osv_linux_distribution"`
	OSVLinuxRelease             types.String `tfsdk:"osv_linux_release"`
	AutomaticOSVScanningEnabled types.Bool   `tfsdk:"automatic_osv_scanning_enabled"`

	VulnerableCodeEnabled                  types.Bool `tfsdk:"vulnerablecode_enabled"`
	AutomaticVulnerableCodeScanningEnabled types.Bool `tfsdk:"automatic_vulnerablecode_scanning_enabled"`
}

func addScanners(attributes map[string]schema.Attribute) {
	attributes["osv_enabled"] = schema.BoolAttribute{
		Optional:            true,
		Computed:            true,
		Default:             booldefault.StaticBool(true),
		MarkdownDescription: "Whether this product may be scanned against the OSV database.",
	}
	attributes["osv_linux_distribution"] = schema.StringAttribute{
		Optional:   true,
		Computed:   true,
		Default:    stringdefault.StaticString(""),
		Validators: []validator.String{stringvalidator.OneOf(schemacommon.WithEmpty(client.OSVLinuxDistributions)...)},
		MarkdownDescription: "Linux distribution to consider when matching OSV advisories. One of " +
			schemacommon.MarkdownList(client.OSVLinuxDistributions) + ".",
	}
	attributes["osv_linux_release"] = schema.StringAttribute{
		Optional:   true,
		Computed:   true,
		Default:    stringdefault.StaticString(""),
		Validators: []validator.String{stringvalidator.LengthAtMost(255)},
		MarkdownDescription: "Release of the Linux distribution, for example `22.04`. Requires " +
			"`osv_linux_distribution` to be set.",
	}
	attributes["automatic_osv_scanning_enabled"] = schema.BoolAttribute{
		Optional:            true,
		Computed:            true,
		Default:             booldefault.StaticBool(false),
		MarkdownDescription: "Whether SecObserve scans this product against OSV on its own schedule.",
	}
	attributes["vulnerablecode_enabled"] = schema.BoolAttribute{
		Optional:            true,
		Computed:            true,
		Default:             booldefault.StaticBool(true),
		MarkdownDescription: "Whether this product may be scanned against VulnerableCode.",
	}
	attributes["automatic_vulnerablecode_scanning_enabled"] = schema.BoolAttribute{
		Optional: true,
		Computed: true,
		Default:  booldefault.StaticBool(false),
		MarkdownDescription: "Whether SecObserve scans this product against VulnerableCode on its own " +
			"schedule. Requires `vulnerablecode_base_url` in the instance settings.",
	}
}

func (s scanners) toAPI() client.ScannerFields {
	return client.ScannerFields{
		OSVEnabled:                             tfutil.BoolValue(s.OSVEnabled),
		OSVLinuxDistribution:                   tfutil.StringValue(s.OSVLinuxDistribution),
		OSVLinuxRelease:                        tfutil.StringValue(s.OSVLinuxRelease),
		AutomaticOSVScanningEnabled:            tfutil.BoolValue(s.AutomaticOSVScanningEnabled),
		VulnerableCodeEnabled:                  tfutil.BoolValue(s.VulnerableCodeEnabled),
		AutomaticVulnerableCodeScanningEnabled: tfutil.BoolValue(s.AutomaticVulnerableCodeScanningEnabled),
	}
}

func (s *scanners) fromAPI(fields client.ScannerFields) {
	s.OSVEnabled = types.BoolValue(fields.OSVEnabled)
	s.OSVLinuxDistribution = types.StringValue(fields.OSVLinuxDistribution)
	s.OSVLinuxRelease = types.StringValue(fields.OSVLinuxRelease)
	s.AutomaticOSVScanningEnabled = types.BoolValue(fields.AutomaticOSVScanningEnabled)
	s.VulnerableCodeEnabled = types.BoolValue(fields.VulnerableCodeEnabled)
	s.AutomaticVulnerableCodeScanningEnabled = types.BoolValue(fields.AutomaticVulnerableCodeScanningEnabled)
}
