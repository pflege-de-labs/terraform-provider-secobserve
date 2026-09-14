package client

// The API enums, transcribed from the backend's types modules. They are used
// both for schema validation and to build attribute descriptions, so the
// documentation can never drift from what is actually accepted.
//
// Sources: core/types.py, issue_tracker/types.py.
var (
	// Severities are the values of Severity.SEVERITY_CHOICES. Note that
	// "None" is a severity level, not an absence of one.
	Severities = []string{"Unknown", "None", "Low", "Medium", "High", "Critical"}

	// Statuses are the values of Status.STATUS_LIST. Several contain a space.
	Statuses = []string{
		"Open",
		"Affected",
		"Resolved",
		"Duplicate",
		"False positive",
		"In review",
		"Not affected",
		"Not security",
		"Risk accepted",
	}

	// IssueTrackerTypes are the supported issue trackers.
	IssueTrackerTypes = []string{"GitHub", "GitLab", "Jira"}

	// OSVLinuxDistributions are the distributions OSV can be queried for.
	OSVLinuxDistributions = []string{
		"AlmaLinux",
		"Alpine",
		"Chainguard",
		"Debian",
		"Mageia",
		"openSUSE",
		"Photon OS",
		"Red Hat",
		"Rocky Linux",
		"SUSE",
		"Ubuntu",
		"Wolfi",
	}
)

// IssueTrackerJira is the one issue tracker type with extra required fields.
const IssueTrackerJira = "Jira"

// IssueTrackerGitHub gets its base URL filled in server-side when omitted.
const IssueTrackerGitHub = "GitHub"

// GitHubDefaultBaseURL is what the backend substitutes for an empty
// issue_tracker_base_url when the type is GitHub
// (core/api/serializers_product.py:561-564).
const GitHubDefaultBaseURL = "https://api.github.com"

// RoleNameList is RoleNames in ascending order of privilege, for schema
// descriptions and validators.
var RoleNameList = []string{"Reader", "Upload", "Writer", "Maintainer", "Owner"}

// ParserTypes are the values of Parser_Type.TYPE_CHOICES
// (import_observations/types.py), used by secobserve_settings'
// observation_title_notification_parser_type.
var ParserTypes = []string{"SCA", "SAST", "DAST", "IAST", "Secrets", "Infrastructure", "Other", "Manual"}

// VEXJustificationStyles are the values of
// VEX_Justification_Styles.STYLE_CHOICES (commons/types.py). Unlike most
// enums here this one has no blank choice: the model field has a default but
// no blank=True.
var VEXJustificationStyles = []string{"CSAF/OpenVEX", "CycloneDX"}

// RuleTypes are the values of Rule_Type.RULE_TYPE_CHOICES (rules/types.py).
var RuleTypes = []string{"Fields", "Rego"}

// RuleTypeRego is the one rule type that requires rego_module to be set.
const RuleTypeRego = "Rego"

// VEXJustifications are the values of VEX_Justification.VEX_JUSTIFICATION_CHOICES
// (core/types.py), used by rule.new_vex_justification.
var VEXJustifications = []string{
	"component_not_present",
	"vulnerable_code_not_present",
	"vulnerable_code_cannot_be_controlled_by_adversary",
	"vulnerable_code_not_in_execute_path",
	"inline_mitigations_already_exist",
	"code_not_present",
	"code_not_reachable",
	"requires_configuration",
	"requires_dependency",
	"requires_environment",
	"protected_by_compiler",
	"protected_at_runtime",
	"protected_at_perimeter",
	"protected_by_mitigating_control",
}

// LicensePolicyEvaluationResults are the values of
// License_Policy_Evaluation_Result.RESULT_CHOICES (licenses/types.py). Note
// the space in "Review required".
var LicensePolicyEvaluationResults = []string{"Allowed", "Forbidden", "Ignored", "Review required", "Unknown"}

// PurlTypes are the keys of PURL_Type.PURL_TYPE_CHOICES (core/types.py), used
// by license_policy.ignore_component_type_list.
var PurlTypes = []string{
	"alpm", "apk", "bitbucket", "bitnami", "cargo", "cocoapods", "composer", "conan",
	"conda", "cpan", "cran", "deb", "docker", "gem", "generic", "github", "golang",
	"hackage", "hex", "huggingface", "luarocks", "maven", "mlflow", "npm", "nuget",
	"oci", "pub", "pypi", "rpm", "qpkg", "swid", "swift",
}
