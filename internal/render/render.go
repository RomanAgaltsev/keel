package render

import (
	"bytes"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"text/template"

	"github.com/RomanAgaltsev/keel/internal/answers"
	"github.com/RomanAgaltsev/keel/internal/manifest"
)

// renderModule renders one module's files into a dest-path -> content map.
func renderModule(m manifest.Manifest, tfs fs.FS, a answers.Answers) (map[string]string, error) {
	out := map[string]string{}
	for _, rule := range m.Files {
		ok, err := evalWhen(rule.When, a)
		if err != nil {
			return nil, fmt.Errorf("module %q: when %q: %w", m.Name, rule.When, err)
		}
		if !ok {
			continue
		}
		matches, err := fs.Glob(tfs, rule.Src)
		if err != nil {
			return nil, fmt.Errorf("module %q: glob %q: %w", m.Name, rule.Src, err)
		}
		for _, src := range matches {
			info, err := fs.Stat(tfs, src)
			if err != nil {
				return nil, err
			}
			if info.IsDir() {
				continue
			}
			rel, err := renderString(strings.TrimSuffix(src, ".tmpl"), a)
			if err != nil {
				return nil, fmt.Errorf("module %q: path %q: %w", m.Name, src, err)
			}
			dest := path.Join(rule.Dest, rel)

			raw, err := fs.ReadFile(tfs, src)
			if err != nil {
				return nil, err
			}
			content, err := renderString(string(raw), a)
			if err != nil {
				return nil, fmt.Errorf("module %q: render %q: %w", m.Name, src, err)
			}
			out[dest] = content
		}
	}
	return out, nil
}

func renderString(tmpl string, a answers.Answers) (string, error) {
	t, err := template.New("t").Option("missingkey=error").Parse(tmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, map[string]any(a)); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// evalWhen returns true when the (optional) condition renders to "true".
func evalWhen(cond string, a answers.Answers) (bool, error) {
	if strings.TrimSpace(cond) == "" {
		return true, nil
	}
	s, err := renderString(cond, a)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(s) == "true", nil
}
