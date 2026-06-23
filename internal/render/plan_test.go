package render

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/internal/answers"
	"github.com/RomanAgaltsev/keel/internal/manifest"
)

func TestBuildPlanCollision(t *testing.T) {
	a := answers.Answers{}
	mods := []moduleFS{
		{
			Manifest: manifest.Manifest{Name: "a", Files: []manifest.FileRule{{Src: "*", Dest: "."}}},
			FS:       fstest.MapFS{"x.tmpl": {Data: []byte("1")}},
		},
		{
			Manifest: manifest.Manifest{Name: "b", Files: []manifest.FileRule{{Src: "*", Dest: "."}}},
			FS:       fstest.MapFS{"x.tmpl": {Data: []byte("2")}},
		},
	}
	_, err := BuildPlan(mods, a)
	require.ErrorContains(t, err, "collision")
	require.ErrorContains(t, err, "x")
}

func TestBuildPlanMerges(t *testing.T) {
	a := answers.Answers{}
	mods := []moduleFS{
		{
			Manifest: manifest.Manifest{Name: "a", Files: []manifest.FileRule{{Src: "*", Dest: "."}}},
			FS:       fstest.MapFS{"a.tmpl": {Data: []byte("A")}},
		},
		{
			Manifest: manifest.Manifest{Name: "b", Files: []manifest.FileRule{{Src: "*", Dest: "."}}},
			FS:       fstest.MapFS{"b.tmpl": {Data: []byte("B")}},
		},
	}
	plan, err := BuildPlan(mods, a)
	require.NoError(t, err)
	require.Equal(t, "A", plan.Files["a"])
	require.Equal(t, "B", plan.Files["b"])
}

func TestPlanOwnerMapsDestToModule(t *testing.T) {
	mods := []moduleFS{
		{
			Manifest: manifest.Manifest{Name: "base", Files: []manifest.FileRule{{Src: "*", Dest: "."}}},
			FS: fstest.MapFS{
				"README.md.tmpl": &fstest.MapFile{Data: []byte("hi")},
			},
		},
	}
	// renderModule writes README.md (the .tmpl suffix is stripped).
	p, err := BuildPlan(mods, answers.Answers{})

	require.NoError(t, err)
	require.Equal(t, "base", p.Owner()["README.md"])

	// Owner is a copy: mutating it doesn't change the plan.
	p.Owner()["README.md"] = "tampered"
	require.Equal(t, "base", p.Owner()["README.md"])
}
