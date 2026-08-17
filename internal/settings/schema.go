package settings

// Desired is a partial desired state for a remote repository. Every field is
// optional so that an undeclared key is distinguishable from one declared false:
// reconciliation converges only what the file names, and leaves the rest alone.
type Desired struct {
	Version    int         `yaml:"version"`
	Repository *Repository `yaml:"repository,omitempty"`
	Security   *Security   `yaml:"security,omitempty"`
	Actions    *Actions    `yaml:"actions,omitempty"`
	Rulesets   []Ruleset   `yaml:"rulesets,omitempty"`
}

// Repository holds general repository options: merge policy and feature toggles.
type Repository struct {
	AllowSquashMerge    *bool `yaml:"allow_squash_merge,omitempty"`
	AllowMergeCommit    *bool `yaml:"allow_merge_commit,omitempty"`
	AllowRebaseMerge    *bool `yaml:"allow_rebase_merge,omitempty"`
	AllowAutoMerge      *bool `yaml:"allow_auto_merge,omitempty"`
	DeleteBranchOnMerge *bool `yaml:"delete_branch_on_merge,omitempty"`
	HasWiki             *bool `yaml:"has_wiki,omitempty"`
	HasProjects         *bool `yaml:"has_projects,omitempty"`
	// Topics is replace-all: the provider API has no per-topic write, so declaring
	// it hands keel ownership of the whole list. Omit it to keep hand-managed topics.
	Topics *[]string `yaml:"topics,omitempty"`
}

// Security holds the code-security and analysis toggles.
type Security struct {
	DependencyGraph               *bool `yaml:"dependency_graph,omitempty"`
	DependabotAlerts              *bool `yaml:"dependabot_alerts,omitempty"`
	DependabotSecurityUpdates     *bool `yaml:"dependabot_security_updates,omitempty"`
	SecretScanning                *bool `yaml:"secret_scanning,omitempty"`
	SecretScanningPushProtection  *bool `yaml:"secret_scanning_push_protection,omitempty"`
	PrivateVulnerabilityReporting *bool `yaml:"private_vulnerability_reporting,omitempty"`
}

// Actions holds the CI policy: which actions may run and what the default job
// token may do.
type Actions struct {
	// Allowed is one of AllowedAll, AllowedLocalOnly, AllowedLocalAndVerified.
	Allowed *string `yaml:"allowed,omitempty"`

	// AllowedPatterns extends AllowedLocalAndVerified with named third-party
	// actions: "GitHub-owned, verified, and also these". Each entry is a host
	// allow-list pattern, e.g. "arduino/setup-task@*".
	//
	// It is legal only with AllowedLocalAndVerified. "all" needs no list and
	// "local_only" admits none, so a pattern under either would be discarded
	// without complaint.
	AllowedPatterns []string `yaml:"allowed_patterns,omitempty"`
	// DefaultWorkflowPermissions is "read" or "write".
	DefaultWorkflowPermissions   *string `yaml:"default_workflow_permissions,omitempty"`
	CanApprovePullRequestReviews *bool   `yaml:"can_approve_pull_request_reviews,omitempty"`
}

// Allowed-actions policy values. LocalAndVerified is keel's own vocabulary; the
// driver translates it, since no provider has a single enum value for it.
const (
	AllowedAll              = "all"
	AllowedLocalOnly        = "local_only"
	AllowedLocalAndVerified = "local_and_verified"
)

// Ruleset is one branch-protection ruleset. Name is the identity key: keel
// matches an existing ruleset by name and updates it in place, so a ruleset
// created by hand under a different name is never touched.
type Ruleset struct {
	Name                     string   `yaml:"name"`
	Target                   string   `yaml:"target"` // only "branch" is supported
	Ref                      string   `yaml:"ref"`    // branch name, e.g. "main"
	RequiredStatusChecks     []string `yaml:"required_status_checks,omitempty"`
	RequiredApprovingReviews *int     `yaml:"required_approving_reviews,omitempty"`
	RequiredLinearHistory    *bool    `yaml:"required_linear_history,omitempty"`
	BlockForcePush           *bool    `yaml:"block_force_push,omitempty"`
	BlockDeletion            *bool    `yaml:"block_deletion,omitempty"`
	// Bypass names roles that may bypass the ruleset; only "repo_admin" is supported.
	Bypass []string `yaml:"bypass,omitempty"`
}

// BypassRepoAdmin is the only supported bypass role.
const BypassRepoAdmin = "repo_admin"
