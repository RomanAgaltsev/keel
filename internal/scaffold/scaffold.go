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
	"github.com/RomanAgaltsev/keel/internal/manifest"
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

	// 1. Resolve the module graph once, then build the render plan from it
	//    (cross-module collisions fail here). The resolved manifests are reused for
	//    the lockfile, so the graph is walked only once per run.
	manifests, err := module.Resolve(opts.Loader, opts.ModuleNames)
	if err != nil {
		return res, err
	}
	plan, err := render.BuildFromManifests(opts.Loader, manifests, opts.Answers)
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
		Name:        opts.Answers.String("repo_name"),
		Description: opts.Answers.String("description"),
		Private:     opts.Answers.String("visibility") == "private",
	}

	// 2b. Dry-run: report what would be written/skipped, then stop. It must touch
	// neither disk nor network, so remote presence is resolved only from inputs that
	// require no network call (an explicit --remote-url); the provider is never asked.
	if opts.DryRun {
		res.State.RemotePresent = opts.RemoteURL != ""
		return dryRunResult(plan, opts, res)
	}

	remotePresent, remote, err := detectRemote(ctx, opts, spec)
	if err != nil {
		return res, err
	}
	res.State.RemotePresent = remotePresent

	// 3. Materialize the working tree.
	res.Written, res.Skipped, err = materialize(ctx, opts, plan, res.State, remote)
	if err != nil {
		return res, err
	}

	// 4-6. git init + identity, write lock, stage + commit.
	repo := git.New(opts.Target)
	committed, err := commitStep(ctx, repo, opts, manifests)
	if err != nil {
		return res, err
	}
	res.Committed = committed

	// 7. Remote step.
	if err := remoteStep(ctx, repo, opts, spec, remote, &res); err != nil {
		return res, err
	}
	return res, nil
}

// detectRemote resolves whether a remote already exists and how to reach it.
func detectRemote(ctx context.Context, opts Options, spec provider.RepoSpec) (bool, provider.RemoteRepo, error) {
	if opts.RemoteURL != "" {
		return true, provider.RemoteRepo{CloneURL: opts.RemoteURL}, nil
	}
	if opts.CreateRemote && opts.Provider != nil {
		exists, r, err := opts.Provider.RepoExists(ctx, spec)
		if err != nil {
			return false, provider.RemoteRepo{}, fmt.Errorf("check remote: %w", err)
		}
		return exists, r, nil
	}
	return false, provider.RemoteRepo{}, nil
}

// dryRunResult reports what would be written/skipped without touching disk or network.
func dryRunResult(plan render.Plan, opts Options, res Result) (Result, error) {
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

	// Remote existence is not probed under --dry-run (no network). If a provider
	// remote would have been created/reused, the real run may clone-then-overlay and
	// skip files already present on the remote — so the counts above are an upper bound.
	if opts.CreateRemote && opts.RemoteURL == "" {
		res.NextSteps = append(res.NextSteps,
			"remote existence not checked in --dry-run; clone-then-overlay may skip files already on the remote")
	}
	return res, nil
}

// materialize writes the plan into the working tree, branching on repo-state.
func materialize(ctx context.Context, opts Options, plan render.Plan, state State, remote provider.RemoteRepo) (written, skipped []string, err error) {
	targetExists, err := pathExists(opts.Target)
	if err != nil {
		return nil, nil, err
	}

	switch {
	case !state.LocalPresent && state.RemotePresent:
		// clone-then-overlay. Only clean up a target we created (an absent or empty
		// dir), never one the user populated — keel never destroys existing work.
		createdByUs := !targetExists
		if _, err := git.Clone(ctx, remote.CloneURL, opts.Target); err != nil {
			if createdByUs {
				_ = os.RemoveAll(opts.Target) //nolint:gosec // best-effort cleanup of the partial clone
			}
			return nil, nil, err
		}
		w, err := render.OverlayPlan(plan, opts.Target, opts.Overwrite)
		if err != nil {
			return nil, nil, err
		}
		return w.Written, w.Skipped, nil
	case state.LocalPresent:
		// overlay into existing dir
		w, err := render.OverlayPlan(plan, opts.Target, opts.Overwrite)
		if err != nil {
			return nil, nil, err
		}
		return w.Written, w.Skipped, nil
	case targetExists:
		// local absent, remote absent, but an existing (empty) dir takes the overlay
		// path (WritePlan refuses any existing target, even an empty one).
		w, err := render.OverlayPlan(plan, opts.Target, opts.Overwrite)
		if err != nil {
			return nil, nil, err
		}
		return w.Written, w.Skipped, nil
	default:
		// non-existent target takes the fresh atomic write.
		if err := render.WritePlan(plan, opts.Target); err != nil {
			return nil, nil, err
		}
		return keysSorted(plan.Files), nil, nil
	}
}

// commitStep initializes the repo, sets identity, writes the lock and commits.
// It reports whether a commit was actually created (an idempotent re-run with no
// staged changes produces no commit). The lock is written *before* committing so
// it lands in the same commit (and re-runs with identical answers produce no diff).
func commitStep(ctx context.Context, repo *git.Repo, opts Options, manifests []manifest.Manifest) (bool, error) {
	if !repo.IsRepo() {
		if err := repo.Init(ctx, "main"); err != nil {
			return false, err
		}
	}
	if err := repo.SetIdentity(ctx, opts.Answers.String("author_name"), opts.Answers.String("author_email")); err != nil {
		return false, err
	}
	if err := writeLock(opts, manifests); err != nil {
		return false, err
	}

	// Stage + commit. Skip an empty commit on idempotent re-runs / all-skipped
	// overlays — nothing staged means nothing to commit.
	if err := repo.AddAll(ctx); err != nil {
		return false, err
	}
	changed, err := repo.HasChanges(ctx)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	if err := repo.Commit(ctx, "chore: scaffold with keel"); err != nil {
		return false, err
	}
	return true, nil
}

// writeLock writes .scaffold.lock recording what produced the repo, from the
// already-resolved manifests (no second graph walk).
func writeLock(opts Options, manifests []manifest.Manifest) error {
	lmods := make([]lock.Module, len(manifests))
	for i, m := range manifests {
		lmods[i] = lock.Module{Name: m.Name, Source: "builtin", Version: m.Version}
	}
	return lock.Write(filepath.Join(opts.Target, ".scaffold.lock"), lock.Lock{
		KeelVersion: opts.KeelVersion, Recipe: opts.Recipe, Modules: lmods, Answers: opts.Answers,
	})
}

// remoteStep creates and/or wires the remote and records follow-up steps.
func remoteStep(ctx context.Context, repo *git.Repo, opts Options, spec provider.RepoSpec, remote provider.RemoteRepo, res *Result) error {
	// Create a remote only when a provider is present and none exists yet. Wiring and
	// pushing, by contrast, need only a remote URL (which --remote-url also supplies),
	// so they are gated separately — not on the provider.
	if opts.CreateRemote && opts.Provider != nil && !res.State.RemotePresent {
		r, err := opts.Provider.CreateRepo(ctx, spec)
		if err != nil {
			return fmt.Errorf("create remote: %w", err)
		}
		remote, res.Created = r, true
	}
	if remote.CloneURL == "" {
		return nil // no remote to wire (provider "none" and no --remote-url)
	}
	if err := repo.AddRemote(ctx, "origin", remote.CloneURL); err != nil {
		return err
	}
	if res.State.LocalPresent && res.State.RemotePresent {
		// both-exist: never force-push; hand reconciliation to the user.
		res.NextSteps = []string{
			"git fetch origin",
			"git rebase origin/main   # or merge, resolve any conflicts",
			"git push -u origin main",
		}
		return nil
	}
	if err := repo.Push(ctx, "origin", "main"); err != nil {
		// Push failures are reported, not fatal to local work.
		res.NextSteps = []string{fmt.Sprintf("git push failed; run: git -C %s push -u origin main", opts.Target)}
	} else {
		res.Pushed = true
	}
	return nil
}

func keysSorted(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
