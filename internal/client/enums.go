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
