package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/internal/answers"
	"github.com/RomanAgaltsev/keel/internal/lock"
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
