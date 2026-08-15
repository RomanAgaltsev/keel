package settings_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2/internal/settings"
)

// write drops content into a temp file and returns its path.
func write(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "keel-settings.yml")
	require.NoError(t, os.WriteFile(p, []byte(content), 0o600))
	return p
}

const full = `
version: 1
repository:
  allow_squash_merge: true
  allow_merge_commit: false
  delete_branch_on_merge: true
  has_wiki: false
  topics: [go, cli]
security:
  dependency_graph: true
  secret_scanning: false
actions:
  allowed: local_and_verified
  default_workflow_permissions: read
  can_approve_pull_request_reviews: false
rulesets:
  - name: "keel: main"
    target: branch
    ref: main
    required_status_checks: [lint, test]
    required_approving_reviews: 0
    block_force_push: true
    bypass: [repo_admin]
`

func TestLoadFull(t *testing.T) {
	d, err := settings.Load(write(t, full))
	require.NoError(t, err)

	require.NotNil(t, d.Repository)
	require.True(t, *d.Repository.AllowSquashMerge)
	require.False(t, *d.Repository.AllowMergeCommit)
	require.Equal(t, []string{"go", "cli"}, *d.Repository.Topics)

	// Undeclared must be distinguishable from declared-false: allow_rebase_merge
	// is absent, secret_scanning is present and false.
	require.Nil(t, d.Repository.AllowRebaseMerge)
	require.NotNil(t, d.Security.SecretScanning)
	require.False(t, *d.Security.SecretScanning)

	require.Len(t, d.Rulesets, 1)
	require.Equal(t, "keel: main", d.Rulesets[0].Name)
	require.Equal(t, 0, *d.Rulesets[0].RequiredApprovingReviews)
	require.Equal(t, []string{"repo_admin"}, d.Rulesets[0].Bypass)
}

func TestLoadMissingFileIsSentinel(t *testing.T) {
	_, err := settings.Load(filepath.Join(t.TempDir(), "nope.yml"))
	require.ErrorIs(t, err, settings.ErrNotFound)
}

func TestLoadRejectsUnknownKey(t *testing.T) {
	_, err := settings.Load(write(t, "version: 1\nrepository:\n  has_wikki: false\n"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "has_wikki")
}

func TestLoadRejectsWrongVersion(t *testing.T) {
	_, err := settings.Load(write(t, "version: 2\n"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "version 2")
}

func TestValidateRejectsBadEnums(t *testing.T) {
	_, err := settings.Load(write(t, "version: 1\nactions:\n  allowed: everything\n"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "allowed")

	_, err = settings.Load(write(t, "version: 1\nactions:\n  default_workflow_permissions: admin\n"))
	require.Error(t, err)

	_, err = settings.Load(write(t, "version: 1\nrulesets:\n  - target: branch\n    ref: main\n"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "name")
}

func TestLoadEmptyOptionalSections(t *testing.T) {
	d, err := settings.Load(write(t, "version: 1\n"))
	require.NoError(t, err)
	require.Nil(t, d.Repository)
	require.Nil(t, d.Security)
	require.Nil(t, d.Actions)
	require.Empty(t, d.Rulesets)
}
