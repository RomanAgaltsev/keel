package render_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel"
	"github.com/RomanAgaltsev/keel/internal/answers"
	"github.com/RomanAgaltsev/keel/internal/module"
	"github.com/RomanAgaltsev/keel/internal/render"
)

func taskfileGo(t *testing.T) string {
	t.Helper()
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"base-layout", "taskfile-go"}, answers.Answers{
		"repo_name": "demo", "description": "d", "module_path": "github.com/acme/demo", "provider": "github",
	})
	require.NoError(t, err)
	return plan.Files["Taskfile.yml"]
}

func TestTaskfileGoHasTheFullTaskSet(t *testing.T) {
	got := taskfileGo(t)
	for _, task := range []string{
		"setup:", "formatters:install:", "golangci-lint:install:", "format:",
		"lint:", "vet:", "test:", "cover:", "deps:update:", "build:", "ci:",
	} {
		require.Contains(t, got, task)
	}
}

func TestTaskfileGoBuildsTheRightBinary(t *testing.T) {
	got := taskfileGo(t)
	require.Contains(t, got, "./cmd/demo")
	require.Contains(t, got, "-X main.version=")
}

func TestTaskfileGoPreservesTaskTemplating(t *testing.T) {
	// The whole file is a Go template now; Task's own {{ }} must survive verbatim
	// or the emitted Taskfile is inert.
	got := taskfileGo(t)
	require.Contains(t, got, "{{.BIN_DIR}}")
	require.Contains(t, got, `{{if eq OS "windows"}}.exe{{end}}`)
	require.Contains(t, got, `{{.ROOT_DIR | replace "\\" "/"}}`)
	require.NotContains(t, got, `{{"`) // no un-rendered escape hatches left behind
}

func TestTaskfileGoPinsMatchLintModule(t *testing.T) {
	require.Contains(t, taskfileGo(t), "GOLANGCI_LINT_VERSION: \"v2.12.2\"")
	require.False(t, strings.Contains(taskfileGo(t), "v2.3.0"))
}

// TestTaskfileGoInstallGuardsUseTestF pins a guard that has to be `test -f`.
// Under Task's embedded shell on Windows an `ls <missing>` status command
// reports success, so an `ls`-based guard marks the install task up to date and
// `task setup` silently installs nothing — the tools are then missing when
// `task lint` runs. `test -f` is evaluated correctly on both platforms.
func TestTaskfileGoInstallGuardsUseTestF(t *testing.T) {
	got := taskfileGo(t)
	require.Contains(t, got, "test -f {{.GOFUMPT}}")
	require.Contains(t, got, "test -f {{.GCI}}")
	require.Contains(t, got, "test -f {{.GOLANGCI_LINT}}")
	require.NotContains(t, got, "- ls {{")
}

// TestTaskfileGoExposesKeelTasks pins the scaffold's only affordance pointing
// back at the tool that made it. .scaffold.lock is inert otherwise: nothing in
// the repo tells its owner that `keel outdated` and `keel update` exist.
func TestTaskfileGoExposesKeelTasks(t *testing.T) {
	got := taskfileGo(t)
	require.Contains(t, got, "keel:outdated:")
	require.Contains(t, got, "keel:update:")
}

// TestTaskfileGoKeelTasksAreNotInCI is the load-bearing half. meta-scaffold.yml
// runs `task ci` inside a fresh scaffold in keel's own CI, so a keel task in the
// gate would make that build self-referential — and would force every
// contributor's CI to install keel to run the project's own tests.
func TestTaskfileGoKeelTasksAreNotInCI(t *testing.T) {
	require.NotContains(t, taskBlock(t, taskfileGo(t), "ci"), "keel",
		"the ci gate must never depend on keel")
}

// taskBlock returns the body of one task: everything from its "  name:" line up
// to the next task at the same indent. Slicing to the end of the file instead
// would make the assertion depend on task order in the template.
func taskBlock(t *testing.T, taskfile, name string) string {
	t.Helper()
	_, rest, ok := strings.Cut(taskfile, "\n  "+name+":")
	require.True(t, ok, "task %q not found", name)
	for i, line := range strings.Split(rest, "\n") {
		if i > 0 && strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "   ") {
			return strings.Join(strings.Split(rest, "\n")[:i], "\n")
		}
	}
	return rest
}
