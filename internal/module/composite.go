package module

import (
	"fmt"
	"io/fs"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/RomanAgaltsev/keel/internal/manifest"
)

// Provenancer reports where a module came from, for the lockfile.
type Provenancer interface {
	Provenance(name string) (source, version string)
}

// External is a resolved external module handed to a Composite. FS is rooted at
// the module directory (module.yaml + templates/ at its root).
type External struct {
	Name    string
	FS      fs.FS
	Source  string
	Version string
}

// Composite overlays builtin modules with resolved external modules behind the
// Loader interface and reports provenance.
type Composite struct {
	builtin   *FSLoader
	ext       map[string]External
	manifests map[string]manifest.Manifest
}

// NewComposite builds a loader over the builtin FS plus the given externals. It
// fails fast on a name collision (external vs builtin, or external vs external).
func NewComposite(builtin fs.FS, externals []External) (*Composite, error) {
	c := &Composite{
		builtin:   NewFSLoader(builtin),
		ext:       make(map[string]External, len(externals)),
		manifests: make(map[string]manifest.Manifest, len(externals)),
	}
	builtinNames, err := c.builtin.ModuleNames()
	if err != nil {
		return nil, err
	}
	bset := make(map[string]bool, len(builtinNames))
	for _, n := range builtinNames {
		bset[n] = true
	}
	for _, e := range externals {
		if bset[e.Name] {
			return nil, fmt.Errorf("external module %q collides with a builtin module", e.Name)
		}
		if _, dup := c.ext[e.Name]; dup {
			return nil, fmt.Errorf("external module %q listed more than once", e.Name)
		}
		m, err := loadManifestFS(e.FS)
		if err != nil {
			return nil, fmt.Errorf("external module %q: %w", e.Name, err)
		}
		if m.Name != "" && m.Name != e.Name {
			return nil, fmt.Errorf("external module %q: its module.yaml declares name %q", e.Name, m.Name)
		}
		m.Name = e.Name
		c.ext[e.Name] = e
		c.manifests[e.Name] = m
	}
	return c, nil
}

// Load returns an external manifest if present, else the builtin one.
func (c *Composite) Load(name string) (manifest.Manifest, error) {
	if m, ok := c.manifests[name]; ok {
		return m, nil
	}
	return c.builtin.Load(name)
}

// TemplateFS returns the template FS for a module (external: its templates/ subdir).
func (c *Composite) TemplateFS(name string) (fs.FS, error) {
	if e, ok := c.ext[name]; ok {
		sub, err := fs.Sub(e.FS, "templates")
		if err != nil {
			return nil, fmt.Errorf("template fs for %q: %w", name, err)
		}
		return sub, nil
	}
	return c.builtin.TemplateFS(name)
}

// ModuleNames returns the sorted union of builtin and external module names.
func (c *Composite) ModuleNames() ([]string, error) {
	names, err := c.builtin.ModuleNames()
	if err != nil {
		return nil, err
	}
	for n := range c.ext {
		names = append(names, n)
	}
	sort.Strings(names)
	return names, nil
}

// Provenance returns the lock source/version for a module.
func (c *Composite) Provenance(name string) (string, string) {
	if e, ok := c.ext[name]; ok {
		return e.Source, e.Version
	}
	m, err := c.builtin.Load(name)
	if err != nil {
		return "builtin", ""
	}
	return "builtin", m.Version
}

func loadManifestFS(fsys fs.FS) (manifest.Manifest, error) {
	var m manifest.Manifest
	b, err := fs.ReadFile(fsys, "module.yaml")
	if err != nil {
		return m, fmt.Errorf("module.yaml not found: %w", err)
	}
	if err := yaml.Unmarshal(b, &m); err != nil {
		return m, err
	}
	return m, nil
}
