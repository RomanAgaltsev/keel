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
