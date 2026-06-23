package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/internal/answers"
	"github.com/RomanAgaltsev/keel/internal/lock"
	"github.com/RomanAgaltsev/keel/internal/modver"
)

func TestUpdateDryRunReportsBehindModule(t *testing.T) {
	target := t.TempDir()

	// A lock whose "lint-go" module is behind the embedded version (0.0.1 < current),
	// with a recorded hash for a file the user has since edited. (lint-go renders
	// .golangci.yml — a real dest in the go-service tree — post the v1.4.0 rename.)
	lk := lock.Lock{
		KeelVersion: "0.0.0", Recipe: "go-service",
		Modules: []lock.Module{{
			Name: "lint-go", Source: "builtin", Version: "0.0.1",
			Files: []lock.File{{Path: ".golangci.yml", SHA256: "deadbeef"}}, // != on-disk ⇒ Conflict
		}},
		Answers: fullAnswers(), // a helper returning a complete answers map (see new_test.go)
	}
	require.NoError(t, lock.Write(filepath.Join(target, ".scaffold.lock"), lk))
	require.NoError(t, os.WriteFile(filepath.Join(target, ".golangci.yml"), []byte("user-edited"), 0o644))

	var buf bytes.Buffer
	root := newRootCmd()
	root.SetOut(&buf)
	root.SetArgs([]string{"update", "--path", target, "--dry-run"})
	require.NoError(t, root.Execute())

	out := buf.String()
	require.Contains(t, out, "dry-run")
	require.Contains(t, out, ".golangci.yml")
	require.Contains(t, out, "conflict")

	// Dry-run writes nothing: the lock is untouched and no .keel-new appears.
	got, err := lock.Read(filepath.Join(target, ".scaffold.lock"))
	require.NoError(t, err)
	require.Equal(t, "0.0.1", got.Modules[0].Version)
	_, statErr := os.Stat(filepath.Join(target, ".golangci.yml.keel-new"))
	require.True(t, os.IsNotExist(statErr))
}

func fullAnswers() answers.Answers {
	return answers.Answers{
		"repo_name":          "demo",
		"description":        "d",
		"module_path":        "github.com/x/demo",
		"author_name":        "Roman Agaltsev",
		"author_email":       "roman-agalcev@yandex.ru",
		"license":            "MIT",
		"visibility":         "public",
		"provider":           "none",
		"create_remote":      false,
		"enable_codeql":      true,
		"enable_govulncheck": true,
		"enable_codecov":     false,
		"dep_bot":            "dependabot",
	}
}

func TestUpdateAppliesAndRewritesLock(t *testing.T) {
	target := t.TempDir()
	// lint-go behind embedded version; one recorded file the user has NOT edited
	// (recorded hash matches on-disk) ⇒ Clean → overwritten with the real render.
	// .golangci.yml is a real lint-go dest in the go-service tree.
	clean := "version: v2.0.0\n"
	lk := lock.Lock{
		KeelVersion: "0.0.0", Recipe: "go-service",
		Modules: []lock.Module{{
			Name: "lint-go", Source: "builtin", Version: "0.0.1",
			Files: []lock.File{{Path: ".golangci.yml", SHA256: lock.HashBytes([]byte(clean))}},
		}},
		Answers: fullAnswers(),
	}
	require.NoError(t, lock.Write(filepath.Join(target, ".scaffold.lock"), lk))
	require.NoError(t, os.WriteFile(filepath.Join(target, ".golangci.yml"), []byte(clean), 0o644))

	root := newRootCmd()
	root.SetArgs([]string{"update", "--path", target, "--modules", "lint-go"})
	require.NoError(t, root.Execute())

	// Lock rewritten: lint version bumped to the current embedded version (> 0.0.1).
	got, err := lock.Read(filepath.Join(target, ".scaffold.lock"))
	require.NoError(t, err)
	require.Equal(t, 2, got.LockVersion)
	cmp, err := modverCompare(t, "0.0.1", got.Modules[0].Version)
	require.NoError(t, err)
	require.Equal(t, -1, cmp) // strictly newer than the old locked version
}

func TestUpdateConflictWritesKeelNewAndBlocksCommit(t *testing.T) {
	target := t.TempDir()
	require.NoError(t, gitInit(t, target)) // helper: `git init -b main` + identity
	edited := "user edits\n"
	lk := lock.Lock{
		KeelVersion: "0.0.0", Recipe: "go-service",
		Modules: []lock.Module{{
			Name: "lint-go", Source: "builtin", Version: "0.0.1",
			Files: []lock.File{{Path: ".golangci.yml", SHA256: "does-not-match"}}, // ⇒ Conflict
		}},
		Answers: fullAnswers(),
	}
	require.NoError(t, lock.Write(filepath.Join(target, ".scaffold.lock"), lk))
	require.NoError(t, os.WriteFile(filepath.Join(target, ".golangci.yml"), []byte(edited), 0o644))

	root := newRootCmd()
	root.SetArgs([]string{"update", "--path", target, "--modules", "lint-go", "--commit"})
	require.NoError(t, root.Execute())

	// User's file preserved; new render beside it.
	b, _ := os.ReadFile(filepath.Join(target, ".golangci.yml"))
	require.Equal(t, edited, string(b))
	_, statErr := os.Stat(filepath.Join(target, ".golangci.yml.keel-new"))
	require.NoError(t, statErr)
	// --commit is suppressed when conflicts exist: no "keel update" commit.
	require.NotContains(t, gitLog(t, target), "keel update")
}

// TestUpdateFillsDefaultsForNewerTemplateKeys covers H1: a lock written by an older
// keel (missing an answer a newer template references) must render via the question's
// default instead of failing with a raw missingkey error.
func TestUpdateFillsDefaultsForNewerTemplateKeys(t *testing.T) {
	target := t.TempDir()
	ans := fullAnswers()
	delete(ans, "enable_codeql") // security-go gained this question after this repo was scaffolded
	lk := lock.Lock{
		KeelVersion: "0.0.0", Recipe: "go-service",
		Modules: []lock.Module{{Name: "security-go", Source: "builtin", Version: "0.0.1"}},
		Answers: ans,
	}
	require.NoError(t, lock.Write(filepath.Join(target, ".scaffold.lock"), lk))

	root := newRootCmd()
	root.SetArgs([]string{"update", "--path", target, "--dry-run"})
	// Before the fix this failed with `map has no entry for key "enable_codeql"`.
	require.NoError(t, root.Execute())
}

// TestUpdateRequiresReconfigureForMissingRequiredAnswer covers H1's other half: a
// missing *required* answer with no default is a clear, actionable error.
func TestUpdateRequiresReconfigureForMissingRequiredAnswer(t *testing.T) {
	target := t.TempDir()
	ans := fullAnswers()
	delete(ans, "repo_name") // required core question, no default
	lk := lock.Lock{
		KeelVersion: "0.0.0", Recipe: "go-service",
		Modules: []lock.Module{{Name: "base-go", Source: "builtin", Version: "0.0.1"}},
		Answers: ans,
	}
	require.NoError(t, lock.Write(filepath.Join(target, ".scaffold.lock"), lk))

	root := newRootCmd()
	root.SetArgs([]string{"update", "--path", target, "--dry-run"})
	err := root.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "--reconfigure")
}

// TestUpdateNoCandidatesLeavesLockUntouched covers M1(a): when nothing is behind,
// the lock is not rewritten just to bump keel_version.
func TestUpdateNoCandidatesLeavesLockUntouched(t *testing.T) {
	target := t.TempDir()
	lk := lock.Lock{
		KeelVersion: "0.0.0", Recipe: "go-service",
		Modules: []lock.Module{{Name: "lint-go", Source: "builtin", Version: "999.0.0"}}, // ahead ⇒ not a candidate
		Answers: fullAnswers(),
	}
	require.NoError(t, lock.Write(filepath.Join(target, ".scaffold.lock"), lk))
	before, err := os.ReadFile(filepath.Join(target, ".scaffold.lock"))
	require.NoError(t, err)

	var buf bytes.Buffer
	root := newRootCmd()
	root.SetOut(&buf)
	root.SetArgs([]string{"update", "--path", target})
	require.NoError(t, root.Execute())

	require.Contains(t, buf.String(), "up to date")
	after, err := os.ReadFile(filepath.Join(target, ".scaffold.lock"))
	require.NoError(t, err)
	require.Equal(t, before, after) // lock byte-for-byte unchanged
}

// TestUpdateCommitCleanScopesToKeelFiles covers M1(b) + M2: a clean --commit makes
// exactly one commit containing keel's files but not the user's unrelated WIP.
func TestUpdateCommitCleanScopesToKeelFiles(t *testing.T) {
	target := t.TempDir()
	require.NoError(t, gitInit(t, target))
	clean := "version: v2.0.0\n"
	lk := lock.Lock{
		KeelVersion: "0.0.0", Recipe: "go-service",
		Modules: []lock.Module{{
			Name: "lint-go", Source: "builtin", Version: "0.0.1",
			Files: []lock.File{{Path: ".golangci.yml", SHA256: lock.HashBytes([]byte(clean))}}, // == on-disk ⇒ Clean
		}},
		Answers: fullAnswers(),
	}
	require.NoError(t, lock.Write(filepath.Join(target, ".scaffold.lock"), lk))
	require.NoError(t, os.WriteFile(filepath.Join(target, ".golangci.yml"), []byte(clean), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(target, "UNRELATED.txt"), []byte("wip"), 0o644))

	root := newRootCmd()
	root.SetArgs([]string{"update", "--path", target, "--modules", "lint-go", "--commit"})
	require.NoError(t, root.Execute())

	require.Contains(t, gitLog(t, target), "keel update")
	out, err := exec.Command("git", "-C", target, "show", "--name-only", "--format=", "HEAD").CombinedOutput()
	require.NoError(t, err, string(out))
	require.Contains(t, string(out), ".golangci.yml")
	require.Contains(t, string(out), ".scaffold.lock")
	require.NotContains(t, string(out), "UNRELATED.txt") // the user's WIP is not bundled in
}

// TestUpdateDryRunOverwriteRelabelsConflict covers M3: --dry-run must reflect what
// --overwrite would actually do (replace in place, not write .keel-new).
func TestUpdateDryRunOverwriteRelabelsConflict(t *testing.T) {
	target := t.TempDir()
	lk := lock.Lock{
		KeelVersion: "0.0.0", Recipe: "go-service",
		Modules: []lock.Module{{
			Name: "lint-go", Source: "builtin", Version: "0.0.1",
			Files: []lock.File{{Path: ".golangci.yml", SHA256: "deadbeef"}}, // != on-disk ⇒ Conflict
		}},
		Answers: fullAnswers(),
	}
	require.NoError(t, lock.Write(filepath.Join(target, ".scaffold.lock"), lk))
	require.NoError(t, os.WriteFile(filepath.Join(target, ".golangci.yml"), []byte("user-edited"), 0o644))

	var buf bytes.Buffer
	root := newRootCmd()
	root.SetOut(&buf)
	root.SetArgs([]string{"update", "--path", target, "--dry-run", "--overwrite"})
	require.NoError(t, root.Execute())

	out := buf.String()
	require.Contains(t, out, "overwrite")
	require.NotContains(t, out, "conflict")
}

// TestUpdateWarnsOnUnknownModule covers L3: a --modules typo is surfaced, not
// silently swallowed by the candidate intersection.
func TestUpdateWarnsOnUnknownModule(t *testing.T) {
	target := t.TempDir()
	lk := lock.Lock{
		KeelVersion: "0.0.0", Recipe: "go-service",
		Modules: []lock.Module{{Name: "lint-go", Source: "builtin", Version: "0.0.1"}},
		Answers: fullAnswers(),
	}
	require.NoError(t, lock.Write(filepath.Join(target, ".scaffold.lock"), lk))

	var buf bytes.Buffer
	root := newRootCmd()
	root.SetOut(&buf)
	root.SetArgs([]string{"update", "--path", target, "--dry-run", "--modules", "lnt-go"})
	require.NoError(t, root.Execute())
	require.Contains(t, buf.String(), `--modules "lnt-go" is not a module`)
}

// modverCompare wraps modver.Compare for the apply-path tests.
func modverCompare(t *testing.T, a, b string) (int, error) {
	t.Helper()
	return modver.Compare(a, b)
}

// gitInit initialises a git repo at dir on the "main" branch with a throwaway
// commit identity, so commit-related assertions have a usable repo.
func gitInit(t *testing.T, dir string) error {
	t.Helper()
	for _, args := range [][]string{
		{"init", "-b", "main"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Keel Test"},
	} {
		out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("git %v: %w: %s", args, err, out)
		}
	}
	return nil
}

// gitLog returns `git log --oneline` for dir, or "" when the repo has no commits
// yet (git exits non-zero on an unborn branch).
func gitLog(t *testing.T, dir string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "log", "--oneline").CombinedOutput()
	if err != nil {
		return ""
	}
	return string(out)
}
