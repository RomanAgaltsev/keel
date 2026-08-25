// Command depcheck fails CI when a module.yaml dep trails the Go proxy.
//
// module.yaml is not a manifest format dependabot understands, and ROADMAP §G
// records a renovate custom manager over modules/ that never ran once. So this
// guard is keel's own.
package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/RomanAgaltsev/keel/v2/internal/godeps"
	"github.com/RomanAgaltsev/keel/v2/internal/manifest"
	"github.com/RomanAgaltsev/keel/v2/internal/module"
	"github.com/RomanAgaltsev/keel/v2/internal/modver"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "depcheck:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	declared, err := declaredDeps()
	if err != nil {
		return err
	}
	latest, err := resolveAll(ctx, godeps.NewProxy(), declared)
	if err != nil {
		return err
	}
	lines := stale(declared, latest)
	if len(lines) == 0 {
		return nil
	}
	for _, l := range lines {
		fmt.Fprintln(os.Stderr, l)
	}
	return fmt.Errorf("%d dependency pin(s) trail the proxy", len(lines))
}

// declaredDeps reads every module's deps block, keyed by module name.
func declaredDeps() (map[string][]manifest.Dep, error) {
	l := module.NewFSLoader(os.DirFS("."))
	names, err := l.ModuleNames()
	if err != nil {
		return nil, err
	}
	out := map[string][]manifest.Dep{}
	for _, name := range names {
		m, err := l.Load(name)
		if err != nil {
			return nil, err
		}
		if len(m.Deps) > 0 {
			out[name] = m.Deps
		}
	}
	return out, nil
}

// resolveAll resolves each distinct module path once, however many modules
// declare it.
func resolveAll(ctx context.Context, r godeps.Resolver, declared map[string][]manifest.Dep) (map[string]string, error) {
	paths := map[string]bool{}
	for _, deps := range declared {
		for _, d := range deps {
			paths[d.Path] = true
		}
	}
	out := make(map[string]string, len(paths))
	for p := range paths {
		v, err := r.LatestVersion(ctx, p)
		if err != nil {
			return nil, err
		}
		out[p] = v
	}
	return out, nil
}

// stale returns one human-readable line per module.yaml dep that trails the
// proxy's latest. A pin ahead of the proxy is not drift: a version can lead the
// proxy briefly right after a release.
func stale(declared map[string][]manifest.Dep, latest map[string]string) []string {
	var out []string
	mods := make([]string, 0, len(declared))
	for m := range declared {
		mods = append(mods, m)
	}
	sort.Strings(mods)
	for _, mod := range mods {
		for _, d := range declared[mod] {
			newest, ok := latest[d.Path]
			if !ok {
				continue
			}
			c, err := modver.Compare(
				strings.TrimPrefix(d.Version, "v"), strings.TrimPrefix(newest, "v"),
			)
			if err != nil || c >= 0 {
				continue
			}
			out = append(out, fmt.Sprintf(
				"modules/%s/module.yaml: %s is pinned at %s, latest is %s",
				mod, d.Path, d.Version, newest,
			))
		}
	}
	return out
}
