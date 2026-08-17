package render

import (
	"fmt"
	"io/fs"
	"slices"
	"sort"
	"strings"

	"github.com/RomanAgaltsev/keel/v2/internal/answers"
	"github.com/RomanAgaltsev/keel/v2/internal/manifest"
	"github.com/RomanAgaltsev/keel/v2/internal/module"
)

// moduleFS pairs a resolved manifest with its template filesystem.
type moduleFS struct {
	Manifest manifest.Manifest
	FS       fs.FS
}

// Plan is the merged set of files to write (dest path -> rendered content).
type Plan struct {
	Files map[string]string

	// Answers is the answer set the templates were rendered with, including the
	// derived keys. Exposed so tests and callers can assert on what the recipe
	// actually computed rather than re-deriving it.
	Answers answers.Answers

	owner map[string]string // dest -> module name, for collision messages
}

// Owner returns a copy of the dest → module-name map, so callers can group a
// plan's files by the module that produced each one without mutating the plan.
func (p Plan) Owner() map[string]string {
	out := make(map[string]string, len(p.owner))
	for dest, mod := range p.owner {
		out[dest] = mod
	}
	return out
}

// BuildPlan renders every module in order and merges the results, failing fast
// on any cross-module destination collision.
func BuildPlan(mods []moduleFS, a answers.Answers) (Plan, error) {
	// Templates are parsed with missingkey=error, so the derived keys must exist
	// before any template runs. Derive covers the answer-only keys; the emits
	// union needs the module list, which only this function has.
	a = answers.Derive(a)
	checks, actions, needs := unionEmits(mods)
	a["emitted_checks"], a["emitted_actions"], a["emitted_needs"] = checks, actions, needs
	a["emitted_action_patterns"] = actionPatterns(actions)

	p := Plan{
		Files:   map[string]string{},
		Answers: a,
		owner:   map[string]string{},
	}
	for _, mf := range mods {
		files, err := renderModule(mf.Manifest, mf.FS, a)
		if err != nil {
			return Plan{}, err
		}
		dests := make([]string, 0, len(files))
		for d := range files {
			dests = append(dests, d)
		}
		sort.Strings(dests) // deterministic order
		for _, dest := range dests {
			if prev, ok := p.owner[dest]; ok {
				return Plan{}, fmt.Errorf("file collision at %q: modules %q and %q both write it", dest, prev, mf.Manifest.Name)
			}
			p.owner[dest] = mf.Manifest.Name
			p.Files[dest] = files[dest]
		}
	}
	return p, nil
}

// unionEmits collects every module's declared contract.
//
// Checks and actions are sorted, deduplicated lists because templates range
// over them, and sorted so a recipe renders byte-identically whatever order its
// modules resolve in -- the golden trees depend on that.
//
// Needs is a set rather than a list because templates ask about membership, and
// keel registers no template functions: `index .emitted_needs "x"` is a builtin
// and yields false for an absent key, where a list would need a `has` function
// that does not exist.
func unionEmits(mods []moduleFS) (checks, actions []string, needs map[string]bool) {
	var c, ac []string
	needs = map[string]bool{}
	for _, mf := range mods {
		c = append(c, mf.Manifest.Emits.Checks...)
		ac = append(ac, mf.Manifest.Emits.Actions...)
		for _, n := range mf.Manifest.Emits.Needs {
			needs[n] = true
		}
	}
	return sortedSet(c), sortedSet(ac), needs
}

// actionPatterns turns declared action names into host allow-list patterns.
//
// A normal action is versioned with "@", so "owner/repo" becomes
// "owner/repo@*". A Docker action is not: it is referenced as
// "docker://image:tag", so "@*" would be meaningless there and the reference is
// left bare, which allows the image at any tag.
//
// Computed here rather than in the template because keel registers no template
// functions, so a template cannot test for the prefix.
func actionPatterns(actions []string) []string {
	out := make([]string, 0, len(actions))
	for _, a := range actions {
		if strings.HasPrefix(a, "docker://") {
			out = append(out, a)
			continue
		}
		out = append(out, a+"@*")
	}
	return out
}

// sortedSet returns vals sorted with duplicates removed, never nil.
func sortedSet(vals []string) []string {
	slices.Sort(vals)
	out := slices.Compact(vals)
	if out == nil {
		return []string{}
	}
	return out
}

// BuildRecipe resolves module names through the loader and builds the plan.
func BuildRecipe(l module.Loader, names []string, a answers.Answers) (Plan, error) {
	manifests, err := module.Resolve(l, names)
	if err != nil {
		return Plan{}, err
	}
	return BuildFromManifests(l, manifests, a)
}

// BuildFromManifests builds a plan from already-resolved manifests (dependency
// order preserved), loading each module's template FS through l. Callers that have
// already resolved the graph use this to avoid walking it twice.
func BuildFromManifests(l module.Loader, manifests []manifest.Manifest, a answers.Answers) (Plan, error) {
	mods := make([]moduleFS, len(manifests))
	for i, m := range manifests {
		tfs, err := l.TemplateFS(m.Name)
		if err != nil {
			return Plan{}, err
		}
		mods[i] = moduleFS{Manifest: m, FS: tfs}
	}
	return BuildPlan(mods, a)
}
