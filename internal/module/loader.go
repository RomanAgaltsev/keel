package module

import (
	"fmt"
	"io/fs"
	"path"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/RomanAgaltsev/keel/internal/manifest"
)

// Loader loads module manifest and template file trees by module name.
type Loader interface {
	Load(name string) (manifest.Manifest, error)
	ModuleNames() ([]string, error)
	// FS returns the filesystem rooted at a module's template dir.
	TemplateFS(name string) (fs.FS, error)
}

// FSLoader loads modules from an fs.FS laid out as modules/<name>/module.yaml.
type FSLoader struct {
	fsys fs.FS
}

// NewFSLoader returns a loader over the given filesystem (e.g. the embedded BuiltFS).
func NewFSLoader(fsys fs.FS) *FSLoader {
	return &FSLoader{fsys: fsys}
}

// Load reads and parses modules/<name>/module.yaml.
func (l *FSLoader) Load(name string) (manifest.Manifest, error) {
	var m manifest.Manifest
	b, err := fs.ReadFile(l.fsys, path.Join("modules", name, "module.yaml"))
	if err != nil {
		return m, fmt.Errorf("load module %q: %w", name, err)
	}
	if err := yaml.Unmarshal(b, &m); err != nil {
		return m, fmt.Errorf("parse module %q: %w", name, err)
	}
	return m, nil
}

// ModuleNames lists every module directory under modules/.
func (l *FSLoader) ModuleNames() ([]string, error) {
	entries, err := fs.ReadDir(l.fsys, "modules")
	if err != nil {
		return nil, fmt.Errorf("list modules: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// TemplateFS returns the filesystem rooted at modules/<name>/templates.
func (l *FSLoader) TemplateFS(name string) (fs.FS, error) {
	sub, err := fs.Sub(l.fsys, path.Join("modules", name, "templates"))
	if err != nil {
		return nil, fmt.Errorf("template fs for %q: %w", name, err)
	}
	return sub, nil
}

// RecipeQuestions returns the concatenated questions of the named modules
// (resolved, dependencies first), in resolved order.
func RecipeQuestions(l Loader, names []string) ([]manifest.Question, error) {
	manifests, err := Resolve(l, names)
	if err != nil {
		return nil, err
	}
	var out []manifest.Question
	for _, m := range manifests {
		out = append(out, m.Questions...)
	}
	return out, nil
}
