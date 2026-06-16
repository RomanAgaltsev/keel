package module

import (
	"fmt"

	"github.com/RomanAgaltsev/keel/internal/manifest"
)

// Resolve expands the given module names with transitive requires and returns
// the manifests in topological order (dependencies before dependents).
// It errors on a missing module or a dependency cycle.
func Resolve(l Loader, names []string) ([]manifest.Manifest, error) {
	const (
		unvisited = iota
		visiting
		done
	)
	state := map[string]int{}
	out := make([]manifest.Manifest, 0, len(names))

	var visit func(name string) error
	visit = func(name string) error {
		switch state[name] {
		case done:
			return nil
		case visiting:
			return fmt.Errorf("dependency cycle through module %q", name)
		}
		state[name] = visiting

		m, err := l.Load(name)
		if err != nil {
			return err
		}

		for _, dep := range m.Requires {
			if err := visit(dep); err != nil {
				return err
			}
		}
		state[name] = done
		out = append(out, m)
		return nil
	}

	for _, name := range names {
		if err := visit(name); err != nil {
			return nil, err
		}
	}

	return out, nil
}
