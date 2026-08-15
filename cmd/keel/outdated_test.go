package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2"
	"github.com/RomanAgaltsev/keel/v2/internal/lock"
	"github.com/RomanAgaltsev/keel/v2/internal/recipe"
)

func TestOutdatedModulesOnly(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, lock.Write(filepath.Join(dir, ".scaffold.lock"), lock.Lock{
		Recipe:  "go-service",
		Modules: []lock.Module{{Name: "lint-go", Source: "builtin", Version: "0.9.0"}},
	}))

	cmd := newOutdatedCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--path", dir, "--modules-only"})
	err := cmd.Execute()

	require.ErrorContains(t, err, "updates available") // non-zero exit signal
	s := out.String()
	require.Contains(t, s, "lint-go")
	require.Contains(t, s, "0.9.0")
}

// TestOutdatedModulesOnlyClean uses a lock with no recipe, so version checking
// finds nothing outdated but the composition half cannot run at all. Since the
// 2026-08-15 review that is exit 2 ("could not check"), not exit 0 — the version
// result is still printed, and the unchecked half is named.
func TestOutdatedModulesOnlyClean(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, lock.Write(filepath.Join(dir, ".scaffold.lock"), lock.Lock{
		Modules: []lock.Module{{Name: "ext", Source: "git", Version: "0.1.0"}}, // non-builtin → skipped
	}))
	cmd := newOutdatedCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--path", dir, "--modules-only"})

	err := cmd.Execute()

	var ec exitCodeError
	require.ErrorAs(t, err, &ec)
	require.Equal(t, 2, ec.code, "a check that could not run is not the same as being up to date")
	require.Contains(t, out.String(), "not checked (composition)")
	require.NotContains(t, out.String(), "Everything is up to date.")
}

// TestOutdatedFullyCleanExitsZero is the control: when every check runs and finds
// nothing, the command must still say so plainly and exit 0.
func TestOutdatedFullyCleanExitsZero(t *testing.T) {
	dir := t.TempDir()
	rec, err := recipe.Load(keel.BuiltinFS, "go-service")
	require.NoError(t, err)
	mods := make([]lock.Module, 0, len(rec.Modules))
	for _, name := range rec.ModuleNames() {
		mods = append(mods, lock.Module{Name: name, Source: "builtin", Version: "999.0.0"})
	}
	require.NoError(t, lock.Write(filepath.Join(dir, ".scaffold.lock"), lock.Lock{
		Recipe: "go-service", Modules: mods,
	}))

	cmd := newOutdatedCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--path", dir, "--modules-only"})

	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "Everything is up to date.")
}

func TestReadPinFiles(t *testing.T) {
	dir := t.TempDir()
	wfDir := filepath.Join(dir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(wfDir, 0o750))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte("version: '3'\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(wfDir, "ci.yml"), []byte("name: ci\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(wfDir, "release.yaml"), []byte("name: release\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(wfDir, "notes.txt"), []byte("ignored\n"), 0o644)) // non-yaml skipped
	require.NoError(t, os.MkdirAll(filepath.Join(wfDir, "sub"), 0o750))                             // dir skipped

	files, err := readPinFiles(dir)
	require.NoError(t, err)

	require.Contains(t, files, "Taskfile.yml")
	require.Contains(t, files, ".github/workflows/ci.yml")
	require.Contains(t, files, ".github/workflows/release.yaml")
	require.NotContains(t, files, ".github/workflows/notes.txt")
	require.Len(t, files, 3)
}

func TestReadPinFilesNoWorkflows(t *testing.T) {
	dir := t.TempDir() // no Taskfile, no .github/workflows
	files, err := readPinFiles(dir)
	require.NoError(t, err)
	require.Empty(t, files)
}

func TestOutdatedReportsRecipeGainedModules(t *testing.T) {
	repo := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".scaffold.lock"), []byte(`lock_version: 2
keel_version: 1.7.1
recipe: go-service
modules:
    - name: base-layout
      source: builtin
      version: 1.0.0
answers: {}
`), 0o600))

	cmd := newOutdatedCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--path", repo, "--modules-only"})
	_ = cmd.Execute() // non-zero exit when updates exist is expected

	require.Contains(t, buf.String(), "license", "a recipe-gained module must be reported")
}

// TestOutdatedWithoutLockDoesNotClaimUpToDate is the H1 guard from the
// 2026-08-15 review. `keel outdated` exists to answer "am I behind?", and it
// used to answer "Everything is up to date." in four situations where it had
// checked nothing at all — the loudest being a directory that is not a keel repo
// (a mistyped --path, or the wrong working directory). "I could not check" and
// "there is nothing to fix" must not print identically.
func TestOutdatedWithoutLockDoesNotClaimUpToDate(t *testing.T) {
	dir := t.TempDir()

	cmd := newOutdatedCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--path", dir, "--modules-only"})
	_ = cmd.Execute()

	out := buf.String()
	require.NotContains(t, out, "Everything is up to date",
		"a directory with no .scaffold.lock was never compared against anything")
	require.Contains(t, out, ".scaffold.lock")
}

// TestOutdatedReportsDegradedCompositionCheck covers the subtler H1 paths: the
// lock exists, so version checking works, but the composition half cannot run.
// `outdated` builds a builtin-only loader, so a recipe naming an external module
// makes update.Resolve fail — and that used to be swallowed into a clean report.
// The version half is still worth printing; the command just has to admit which
// half it skipped.
func TestOutdatedReportsDegradedCompositionCheck(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "recipe.yaml"), []byte(
		"name: custom\nlanguage: go\nmodules: [base-layout, my-external]\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".scaffold.lock"), []byte(`lock_version: 2
keel_version: 2.4.0
recipe: custom
recipe_source: recipe.yaml
modules:
    - name: base-layout
      source: builtin
      version: 1.0.0
answers: {}
`), 0o600))

	cmd := newOutdatedCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--path", dir, "--modules-only"})
	_ = cmd.Execute()

	out := buf.String()
	require.NotContains(t, out, "Everything is up to date")
	require.Contains(t, out, "composition")
}
