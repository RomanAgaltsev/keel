package render

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2/internal/answers"
	"github.com/RomanAgaltsev/keel/v2/internal/manifest"
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

// TestBuildPlanAggregatesEmits is the seam the whole contract rests on: the
// union has to exist as answers before any template runs, because templates are
// parsed with missingkey=error.
func TestBuildPlanAggregatesEmits(t *testing.T) {
	mods := []moduleFS{
		{
			Manifest: manifest.Manifest{
				Name:  "lint-ish",
				Emits: manifest.Emits{Checks: []string{"lint"}, Actions: []string{"arduino/setup-task"}},
			},
			FS: fstest.MapFS{},
		},
		{
			Manifest: manifest.Manifest{
				Name: "release-ish",
				Emits: manifest.Emits{
					Actions: []string{"googleapis/release-please-action"},
					Needs:   []string{manifest.NeedCanApprovePullRequestReviews},
				},
			},
			FS: fstest.MapFS{},
		},
		{
			// Declares a check the first module already declares: the union
			// deduplicates, so a shared context is required once.
			Manifest: manifest.Manifest{
				Name:  "also-lint",
				Emits: manifest.Emits{Checks: []string{"lint", "typos"}},
			},
			FS: fstest.MapFS{},
		},
	}

	p, err := BuildPlan(mods, answers.Answers{"repo_name": "demo"})
	require.NoError(t, err)

	// Sorted and deduplicated, so a recipe renders byte-identically whatever
	// order its modules resolve in -- the golden trees depend on that.
	require.Equal(t, []string{"lint", "typos"}, p.Answers["emitted_checks"])
	require.Equal(t, []string{"arduino/setup-task", "googleapis/release-please-action"}, p.Answers["emitted_actions"])
	require.Equal(t, []string{manifest.NeedCanApprovePullRequestReviews}, p.Answers["emitted_needs"])
}

// TestBuildPlanEmitsKeysExistWhenNothingIsDeclared covers the missingkey=error
// hazard: a template referencing .emitted_checks must not explode just because
// no module in the recipe declares any.
func TestBuildPlanEmitsKeysExistWhenNothingIsDeclared(t *testing.T) {
	mods := []moduleFS{{Manifest: manifest.Manifest{Name: "bare"}, FS: fstest.MapFS{}}}

	p, err := BuildPlan(mods, answers.Answers{"repo_name": "demo"})
	require.NoError(t, err)

	// Empty slices rather than nil: both range to nothing in a template, but an
	// explicit empty slice keeps the assertion unambiguous.
	require.Equal(t, []string{}, p.Answers["emitted_checks"])
	require.Equal(t, []string{}, p.Answers["emitted_actions"])
	require.Equal(t, []string{}, p.Answers["emitted_needs"])
}
