package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/RomanAgaltsev/keel/v2/internal/lock"
	"github.com/RomanAgaltsev/keel/v2/internal/provider"
	"github.com/RomanAgaltsev/keel/v2/internal/settings"
)

// resolveProvider is a seam so tests can drive the command without a token or a
// network. Production code never reassigns it.
var resolveProvider = provider.Resolve

// settingsFlags holds the parsed flags for the settings command.
type settingsFlags struct {
	dir      string
	check    bool
	repo     string // owner/name, overriding .scaffold.lock
	provider string // provider name, overriding .scaffold.lock
}

func newSettingsCmd() *cobra.Command {
	var f settingsFlags
	cmd := &cobra.Command{
		Use:   "settings",
		Short: "Reconcile the remote repository's settings",
	}
	apply := &cobra.Command{
		Use:   "apply",
		Short: "Apply .github/keel-settings.yml to the remote (or report drift with --check)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSettingsApply(cmd, &f)
		},
	}
	apply.Flags().StringVarP(&f.dir, "directory", "C", ".", "repository directory")
	apply.Flags().BoolVar(&f.check, "check", false, "report drift without changing anything (exit 1 on drift, 2 on error)")
	apply.Flags().StringVar(&f.repo, "repo", "", "owner/name, when there is no .scaffold.lock")
	apply.Flags().StringVar(&f.provider, "provider", "", "provider name, when there is no .scaffold.lock")
	cmd.AddCommand(apply)
	return cmd
}

func runSettingsApply(cmd *cobra.Command, f *settingsFlags) error {
	d, err := settings.Load(filepath.Join(f.dir, filepath.FromSlash(settings.DefaultPath)))
	if errors.Is(err, settings.ErrNotFound) {
		return exitCodeError{code: 2, err: fmt.Errorf("no %s in %q; scaffold with a repo-settings module or write one by hand", settings.DefaultPath, f.dir)}
	}
	if err != nil {
		return exitCodeError{code: 2, err: err}
	}

	applier, name, err := resolveApplier(f)
	if err != nil {
		return exitCodeError{code: 2, err: err}
	}

	rep := settings.Reconcile(cmd.Context(), applier.SettingsGroups(provider.RepoSpec{Name: name}), d, !f.check)
	rep.Render(cmd.OutOrStdout())

	switch {
	case len(rep.Failed) > 0 && f.check:
		return exitCodeError{code: 2, err: errors.New("one or more setting groups could not be read")}
	case len(rep.Failed) > 0:
		return exitCodeError{code: 1, err: errors.New("one or more setting groups failed")}
	case f.check && !rep.InSync():
		return exitCodeError{code: 1} // drift is not an error; the report already said what differs
	default:
		return nil
	}
}

// resolveApplier turns the flags and lockfile into a settings-capable provider.
func resolveApplier(f *settingsFlags) (provider.SettingsApplier, string, error) {
	name, owner, provName, err := resolveTarget(f)
	if err != nil {
		return nil, "", err
	}
	p, err := resolveProvider(provName, "github.com/"+owner+"/"+name)
	if err != nil {
		return nil, "", err
	}
	applier, ok := p.(provider.SettingsApplier)
	if !ok {
		return nil, "", fmt.Errorf("settings are not supported for provider %s", provName)
	}
	return applier, name, nil
}

// resolveTarget determines repo name, owner and provider from .scaffold.lock,
// with --repo and --provider overriding. A repo keel never created has no lock,
// which is the retrofit case the flags exist for.
func resolveTarget(f *settingsFlags) (name, owner, provName string, err error) {
	if f.repo != "" {
		parts := strings.SplitN(f.repo, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", "", fmt.Errorf("--repo %q: want owner/name", f.repo)
		}
		owner, name = parts[0], parts[1]
	}
	provName = f.provider

	if l, lockErr := lock.Read(filepath.Join(f.dir, ".scaffold.lock")); lockErr == nil {
		name, owner, provName = fillFromLock(l, name, owner, provName)
	}
	if name == "" || owner == "" {
		return "", "", "", fmt.Errorf("cannot determine the remote repo: no .scaffold.lock in %q — pass --repo owner/name", f.dir)
	}
	if provName == "" {
		provName = "github"
	}
	return name, owner, provName, nil
}

// fillFromLock supplies whatever the flags did not, from the lock's answers.
func fillFromLock(l lock.Lock, name, owner, provName string) (string, string, string) {
	if name == "" {
		if v, ok := l.Answers["repo_name"].(string); ok {
			name = v
		}
		if v, ok := l.Answers["module_path"].(string); ok {
			if seg := strings.Split(v, "/"); len(seg) >= 2 {
				owner = seg[1]
			}
		}
	}
	if provName == "" {
		if v, ok := l.Answers["provider"].(string); ok {
			provName = v
		}
	}
	return name, owner, provName
}
