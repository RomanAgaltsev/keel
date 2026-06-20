package render

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/internal/answers"
	"github.com/RomanAgaltsev/keel/internal/manifest"
)

func TestSafeDest(t *testing.T) {
	require.NoError(t, safeDest("README.md"))
	require.NoError(t, safeDest("pkg/main.go"))
	require.Error(t, safeDest("/etc/passwd"))
	require.Error(t, safeDest("../escape"))
	require.Error(t, safeDest("a/../../escape"))
	require.Error(t, safeDest(""))
}

func TestRenderModuleRejectsEscapingDest(t *testing.T) {
	tfs := fstest.MapFS{"x.tmpl": {Data: []byte("y")}}
	m := manifest.Manifest{Name: "evil", Files: []manifest.FileRule{{Src: "*", Dest: "../out"}}}
	_, err := renderModule(m, tfs, answers.Answers{})
	require.ErrorContains(t, err, "unsafe destination")
}
