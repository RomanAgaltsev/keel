// Command modulebump patch-bumps the version of every changed module that is
// missing a bump, used by the `task modules:bump` workflow.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/RomanAgaltsev/keel/internal/modver"
)

var versionLine = regexp.MustCompile(`(?m)^(version:\s*)\S+([^\S\n]*)$`)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "modulebump:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	switch {
	case len(args) >= 1 && args[0] == "--auto":
		level := "patch"
		if len(args) >= 2 {
			level = args[1]
		}
		return auto("origin/main", level)
	case len(args) == 2:
		_, err := bumpModule(args[0], args[1])
		return err
	default:
		return errors.New("usage: modulebump <module> <patch|minor|major> | modulebump --auto [level]")
	}
}

// auto patch-bumps every module changed since base whose version was not bumped.
func auto(base, level string) error {
	changed, err := changedFiles(base)
	if err != nil {
		return err
	}
	touched := modver.ModulesTouched(changed)
	prevVers, headVers := map[string]string{}, map[string]string{}
	for _, m := range touched {
		nv, err := versionAt("", m)
		if err != nil {
			return err
		}
		headVers[m] = nv
		if ov, err := versionAt(base, m); err == nil {
			prevVers[m] = ov
		}
	}
	for _, m := range modver.Offenders(touched, prevVers, headVers) {
		bumped, err := bumpModule(m, level)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "bumped %s -> %s\n", m, bumped)
	}
	return nil
}

// bumpModule rewrites modules/<m>/module.yaml's version line and returns the new version.
func bumpModule(m, level string) (string, error) {
	path := "modules/" + m + "/module.yaml"
	b, err := os.ReadFile(path) //nolint:gosec // dev tool; path built from a local module name
	if err != nil {
		return "", err
	}
	cur, err := versionFrom(b)
	if err != nil {
		return "", err
	}
	next, err := modver.Bump(cur, level)
	if err != nil {
		return "", err
	}
	out := versionLine.ReplaceAll(b, []byte("${1}"+next+"${2}"))
	if err := os.WriteFile(path, out, 0o644); err != nil { //nolint:gosec // module.yaml is a source file
		return "", err
	}
	return next, nil
}

func versionFrom(b []byte) (string, error) {
	var doc struct {
		Version string `yaml:"version"`
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return "", err
	}
	if doc.Version == "" {
		return "", errors.New("no version field")
	}
	return doc.Version, nil
}

func changedFiles(base string) ([]string, error) {
	out, err := exec.CommandContext(context.Background(), "git", "diff", "--name-only", base+"...HEAD").Output() //nolint:gosec // dev tool; base is a local git ref
	if err != nil {
		return nil, fmt.Errorf("git diff against %q: %w", base, err)
	}
	var files []string
	for _, l := range strings.Split(string(out), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			files = append(files, l)
		}
	}
	return files, nil
}

func versionAt(ref, m string) (string, error) {
	path := "modules/" + m + "/module.yaml"
	var (
		b   []byte
		err error
	)
	if ref == "" {
		b, err = os.ReadFile(path)
	} else {
		b, err = exec.CommandContext(context.Background(), "git", "show", ref+":"+path).Output() //nolint:gosec // dev tool; ref and path are local
	}
	if err != nil {
		return "", err
	}
	return versionFrom(b)
}
