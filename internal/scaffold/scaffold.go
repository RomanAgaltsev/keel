package scaffold

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/RomanAgaltsev/keel/internal/answers"
	"github.com/RomanAgaltsev/keel/internal/git"
	"github.com/RomanAgaltsev/keel/internal/lock"
	"github.com/RomanAgaltsev/keel/internal/module"
	"github.com/RomanAgaltsev/keel/internal/provider"
	"github.com/RomanAgaltsev/keel/internal/render"
)

// Options configures a keel new run.
type Options struct {
	Target       string
	Recipe       string
	ModuleNames  []string
	Loader       module.Loader
	Provider     provider.Provider // may be nil when CreateRemote is false
	Answers      answers.Answers
	CreateRemote bool
	RemoteURL    string // explicit remote (forces RemotePresent); else discovered via Provider
	Overwrite    bool
	DryRun       bool // run steps and report, touch neither disk nor network
	KeelVersion  string
}

// Result reports what a run did (or, under DryRun, would do).
type Result struct {
	State     State
	Written   []string
	Skipped   []string
	Created   bool
	Committed bool
	Pushed    bool
	DryRun    bool
	NextSteps []string
}

// Run executes the lifecycle, branching on the repo-state.
func Run(ctx context.Context, opts Options) (Result, error) {
	var res Result

	// 1. Build the render plan (cross-module collisions fail here).
	plan, err := render.BuildRecipe(opts.Loader, opts.ModuleNames, opts.Answers)
	if err != nil {
		return res, err
	}

	// 2. Detect state.
	lp, err := localPresent(opts.Target)
	if err != nil {
		return res, err
	}
	res.State.LocalPresent = lp

	spec := provider.RepoSpec{
		Name:        str(opts.Answers, "repo_name"),
		Description: str(opts.Answers, "description"),
		Private:     str(opts.Answers, "visibility") == "private",
	}
	var remote provider.RemoteRepo
	if opts.RemoteURL != "" {
		res.State.RemotePresent = true
		remote = provider.RemoteRepo{CloneURL: opts.RemoteURL}
	} else if opts.CreateRemote && opts.Provider != nil {
		exists, r, err := opts.Provider.RepoExists(ctx, spec)
		if err != nil {
			return res, fmt.Errorf("check remote: %w", err)
		}
		res.State.RemotePresent = exists
		remote = r
	}

	targetExists, err := pathExists(opts.Target)
	if err != nil {
		return res, err
	}

	// 2b. Dry-run: report what would be written/skipped, then stop (no disk/network).
	if opts.DryRun {
		res.DryRun = true
		for dest := range plan.Files {
			full := filepath.Join(opts.Target, filepath.FromSlash(dest))
			if _, statErr := os.Stat(full); statErr == nil && !opts.Overwrite {
				res.Skipped = append(res.Skipped, dest)
			} else {
				res.Written = append(res.Written, dest)
			}
		}
		sort.Strings(res.Written)
		sort.Strings(res.Skipped)
		return res, nil
	}

	// 3. Materialize the working tree.
	repo := git.New(opts.Target)
	switch {
	case !res.State.LocalPresent && res.State.RemotePresent:
		// clone-then-overlay
		if _, err := git.Clone(remote.CloneURL, opts.Target); err != nil {
			return res, err
		}
		w, err := render.OverlayPlan(plan, opts.Target, opts.Overwrite)
		if err != nil {
			return res, err
		}
		res.Written, res.Skipped = w.Written, w.Skipped
	case res.State.LocalPresent:
		// overlay into existing dir
		w, err := render.OverlayPlan(plan, opts.Target, opts.Overwrite)
		if err != nil {
			return res, err
		}
		res.Written, res.Skipped = w.Written, w.Skipped
	default:
		// local absent, remote absent. A non-existent target takes the fresh
		// atomic write; an existing *empty* dir takes the overlay path (WritePlan
		// refuses any existing target, even an empty one).
		if targetExists {
			w, err := render.OverlayPlan(plan, opts.Target, opts.Overwrite)
			if err != nil {
				return res, err
			}
			res.Written, res.Skipped = w.Written, w.Skipped
		} else {
			if err := render.WritePlan(plan, opts.Target); err != nil {
				return res, err
			}
			res.Written = keysSorted(plan.Files)
		}
	}

	// 4. git init (if needed) + identity.
	if !repo.IsRepo() {
		if err := repo.Init("main"); err != nil {
			return res, err
		}
	}
	if err := repo.SetIdentity(str(opts.Answers, "author_name"), str(opts.Answers, "author_email")); err != nil {
		return res, err
	}

	// 5. Write .scaffold.lock *before* committing so it lands in the same commit
	// (and re-runs with identical answers produce no diff → no empty commit).
	manifests, err := module.Resolve(opts.Loader, opts.ModuleNames)
	if err != nil {
		return res, err
	}
	lmods := make([]lock.Module, len(manifests))
	for i, m := range manifests {
		lmods[i] = lock.Module{Name: m.Name, Source: "builtin", Version: m.Version}
	}
	if err := lock.Write(filepath.Join(opts.Target, ".scaffold.lock"), lock.Lock{
		KeelVersion: opts.KeelVersion, Recipe: opts.Recipe, Modules: lmods, Answers: opts.Answers,
	}); err != nil {
		return res, err
	}

	// 6. Stage + commit. Skip an empty commit on idempotent re-runs / all-skipped
	// overlays — nothing staged means nothing to commit.
	if err := repo.AddAll(); err != nil {
		return res, err
	}
	changed, err := repo.HasChanges()
	if err != nil {
		return res, err
	}
	if changed {
		if err := repo.Commit("chore: scaffold with keel"); err != nil {
			return res, err
		}
		res.Committed = true
	}

	// 7. Remote step.
	if opts.CreateRemote && opts.Provider != nil {
		if !res.State.RemotePresent {
			r, err := opts.Provider.CreateRepo(ctx, spec)
			if err != nil {
				return res, fmt.Errorf("create remote: %w", err)
			}
			remote, res.Created = r, true
		}
		if remote.CloneURL != "" {
			if err := repo.AddRemote("origin", remote.CloneURL); err != nil {
				return res, err
			}
		}
		if res.State.LocalPresent && res.State.RemotePresent {
			// both-exist: never force-push; hand reconciliation to the user.
			res.NextSteps = []string{
				"git fetch origin",
				"git rebase origin/main   # or merge, resolve any conflicts",
				"git push -u origin main",
			}
		} else if remote.CloneURL != "" {
			if err := repo.Push("origin", "main"); err != nil {
				// Push failures are reported, not fatal to local work.
				res.NextSteps = []string{fmt.Sprintf("git push failed; run: git -C %s push -u origin main", opts.Target)}
			} else {
				res.Pushed = true
			}
		}
	}
	return res, nil
}

func str(a answers.Answers, k string) string {
	s, _ := a[k].(string)
	return s
}

func keysSorted(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
