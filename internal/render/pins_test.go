package render_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2/internal/render"
)

// recipePlan renders a builtin recipe with its declared archetype, so the pin
// checks see exactly what a scaffolded repo receives.
func recipePlan(t *testing.T, name string) render.Plan {
	t.Helper()
	return planForRecipe(t, name)
}

func TestNoStaleActionPins(t *testing.T) {
	for _, rec := range []string{"go-service", "rust-service", "go-library"} {
		plan := recipePlan(t, rec)
		for path, content := range plan.Files {
			if !strings.HasPrefix(path, ".github/workflows/") {
				continue
			}
			require.NotContains(t, content, "actions/checkout@v5", "%s/%s", rec, path)
			require.NotContains(t, content, "actions/checkout@v6", "%s/%s", rec, path)
			require.NotContains(t, content, "actions/setup-go@v5", "%s/%s", rec, path)
		}
	}
}

func TestWorkflowsDoNotPinToolVersions(t *testing.T) {
	// The Taskfile owns tool versions. A workflow that pins golangci-lint too
	// is a second source of truth, and the two drifted by nine minor versions
	// before this test existed.
	for _, rec := range []string{"go-service", "rust-service", "go-library"} {
		plan := recipePlan(t, rec)
		for path, content := range plan.Files {
			if !strings.HasPrefix(path, ".github/workflows/") {
				continue
			}
			require.NotContains(t, content, "golangci-lint@", "%s/%s pins a tool the Taskfile owns", rec, path)
			require.NotContains(t, content, "GOLANGCI", "%s/%s pins a tool the Taskfile owns", rec, path)
		}
	}
}

func TestTaskfileAndLintAgreeOnGolangciVersion(t *testing.T) {
	plan := recipePlan(t, "go-service")
	require.Contains(t, plan.Files["Taskfile.yml"], "v2.12.2")
}

// TestTemplatePinsMatchKeelsOwnWorkflows replaces a denylist with a comparison.
//
// The old guard (TestNoStaleActionPins) named three specific bad pins, so it
// could only catch drift someone had already noticed and written down. Measured
// at the 2026-08-15 review, five pins had drifted past it: the templates lagged
// keel's own workflows on setup-go, goreleaser-action and release-please-action,
// and the Go and Rust modules disagreed with each other on codecov-action and
// dependency-review-action.
//
// keel dogfoods the actions it emits, so its own workflows are the reference.
// This needs no maintenance: bump keel's workflow and the templates must follow.
func TestTemplatePinsMatchKeelsOwnWorkflows(t *testing.T) {
	own := actionPins(t, ownWorkflowSources(t))

	for _, rec := range []string{"go-service", "rust-service", "go-library"} {
		rendered := map[string]string{}
		for path, content := range recipePlan(t, rec).Files {
			if strings.HasPrefix(path, ".github/workflows/") {
				rendered[path] = content
			}
		}
		for action, version := range actionPins(t, rendered) {
			ownVersion, shared := own[action]
			if !shared {
				continue // an action keel does not use itself; nothing to compare against
			}
			require.Equal(t, ownVersion, version,
				"%s pins %s@%s but keel's own workflows use @%s — bump the template",
				rec, action, version, ownVersion)
		}
	}
}

// TestTemplatePinsAgreeAcrossLanguages catches the other half: two modules
// emitting the same action at different versions, which no comparison against
// keel's own workflows would notice if keel does not use that action at all.
func TestTemplatePinsAgreeAcrossLanguages(t *testing.T) {
	seen := map[string]map[string]string{} // action -> version -> first recipe that used it

	for _, rec := range []string{"go-service", "rust-service"} {
		rendered := map[string]string{}
		for path, content := range recipePlan(t, rec).Files {
			if strings.HasPrefix(path, ".github/workflows/") {
				rendered[path] = content
			}
		}
		for action, version := range actionPins(t, rendered) {
			if seen[action] == nil {
				seen[action] = map[string]string{}
			}
			seen[action][version] = rec
		}
	}

	for action, versions := range seen {
		require.Len(t, versions, 1,
			"%s is pinned at %v by different recipes; a scaffold's toolchain must not depend on its language",
			action, versions)
	}
}

// actionPinRE matches a workflow's `uses: owner/repo@vN` pin, ignoring any
// trailing subpath (github/codeql-action/init and .../analyze share a version).
var actionPinRE = regexp.MustCompile(`uses:\s+([a-zA-Z0-9._-]+/[a-zA-Z0-9._-]+)(?:/[a-zA-Z0-9._/-]+)?@(v[0-9]+)`)

// actionPins maps action repo -> pinned major across a set of workflow sources.
func actionPins(t *testing.T, sources map[string]string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, content := range sources {
		for _, m := range actionPinRE.FindAllStringSubmatch(content, -1) {
			out[m[1]] = m[2]
		}
	}
	return out
}

// ownWorkflowSources reads keel's own workflows from disk.
func ownWorkflowSources(t *testing.T) map[string]string {
	t.Helper()
	dir := filepath.Join("..", "..", ".github", "workflows")
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	out := map[string]string{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yml") {
			continue
		}
		b, rerr := os.ReadFile(filepath.Join(dir, e.Name()))
		require.NoError(t, rerr)
		out[e.Name()] = string(b)
	}
	require.NotEmpty(t, out, "keel's own workflows are the reference; none were read")
	return out
}
