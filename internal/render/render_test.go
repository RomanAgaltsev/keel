package render

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/internal/answers"
	"github.com/RomanAgaltsev/keel/internal/manifest"
)

func TestRenderModule(t *testing.T) {
	tfs := fstest.MapFS{
		"README.md.tmpl": {Data: []byte("# {{ .repo_name }}\n")},
		"name.txt.tmpl":  {Data: []byte("{{ .repo_name }}")},
	}
	m := manifest.Manifest{
		Name:  "demo",
		Files: []manifest.FileRule{{Src: "*", Dest: "."}},
	}
	a := answers.Answers{"repo_name": "foo"}

	files, err := renderModule(m, tfs, a)
	require.NoError(t, err)
	require.Equal(t, "# foo\n", files["README.md"])
	require.Equal(t, "foo", files["name.txt"])
}

func TestRenderModuleWhenFalseSkips(t *testing.T) {
	tfs := fstest.MapFS{"x.tmpl": {Data: []byte("y")}}
	m := manifest.Manifest{
		Name:  "demo",
		Files: []manifest.FileRule{{Src: "*", Dest: ".", When: "{{ .on }}"}},
	}
	files, err := renderModule(m, tfs, answers.Answers{"on": false})
	require.NoError(t, err)
	require.Empty(t, files)
}

func TestRenderModuleVerbatimNonTmpl(t *testing.T) {
	tfs := fstest.MapFS{
		"ci.yml":         {Data: []byte("token: ${{ secrets.GITHUB_TOKEN }}\n")}, // must NOT be templated
		"README.md.tmpl": {Data: []byte("# {{ .repo_name }}\n")},                 // must be templated
	}
	m := manifest.Manifest{Name: "demo", Files: []manifest.FileRule{{Src: "*", Dest: "."}}}
	files, err := renderModule(m, tfs, answers.Answers{"repo_name": "foo"})
	require.NoError(t, err)
	require.Equal(t, "token: ${{ secrets.GITHUB_TOKEN }}\n", files["ci.yml"]) // unchanged
	require.Equal(t, "# foo\n", files["README.md"])
}

func TestRenderModuleMissingLiteralSrcErrors(t *testing.T) {
	tfs := fstest.MapFS{"present.yml": {Data: []byte("ok")}}
	m := manifest.Manifest{Name: "demo", Files: []manifest.FileRule{{Src: "absent.yml", Dest: "."}}}
	_, err := renderModule(m, tfs, answers.Answers{})
	require.ErrorContains(t, err, "absent.yml") // a non-glob src that matches nothing is an authoring bug
}
