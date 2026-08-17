package answers_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2/internal/answers"
)

func TestDeriveDefaultsToService(t *testing.T) {
	got := answers.Derive(answers.Answers{"repo_name": "demo"})
	require.Equal(t, "service", got["archetype"])
	require.Equal(t, false, got["is_library"])
	require.Equal(t, "demo", got["package_name"])
}

func TestDeriveLibrarySetsIsLibrary(t *testing.T) {
	got := answers.Derive(answers.Answers{"repo_name": "demo", "archetype": "library"})
	require.Equal(t, "library", got["archetype"])
	require.Equal(t, true, got["is_library"])
}

// TestDeriveIgnoresSuppliedIsLibrary pins the single-source-of-truth rule: the
// archetype comes from the recipe, so a stale is_library persisted in a
// lockfile's answers must never win.
func TestDeriveIgnoresSuppliedIsLibrary(t *testing.T) {
	got := answers.Derive(answers.Answers{
		"repo_name": "demo", "archetype": "service", "is_library": true,
	})
	require.Equal(t, false, got["is_library"])
}

func TestDeriveDoesNotMutateInput(t *testing.T) {
	in := answers.Answers{"repo_name": "demo"}
	answers.Derive(in)
	require.NotContains(t, in, "archetype")
	require.NotContains(t, in, "is_library")
	require.NotContains(t, in, "package_name")
}

func TestPackageName(t *testing.T) {
	for _, tc := range []struct{ name, in, want string }{
		{"plain", "demo", "demo"},
		{"hyphenated", "go-thing", "gothing"},
		{"mixed case and underscore", "Foo_Bar", "foobar"},
		{"leading digit", "2fa", "pkg2fa"},
		{"empty", "", "pkg"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, answers.PackageName(tc.in))
		})
	}
}

func TestDeriveInitialVersionFromArchetype(t *testing.T) {
	lib := answers.Derive(answers.Answers{"archetype": answers.ArchetypeLibrary, "repo_name": "demo"})
	require.Equal(t, "0.1.0", lib["initial_version"])
	require.Equal(t, true, lib["pre_major"])

	svc := answers.Derive(answers.Answers{"archetype": answers.ArchetypeService, "repo_name": "demo"})
	require.Equal(t, "1.0.0", svc["initial_version"])
	require.Equal(t, false, svc["pre_major"])

	// The default archetype is service, so an unset archetype must not
	// accidentally put a service into 0.x.
	def := answers.Derive(answers.Answers{"repo_name": "demo"})
	require.Equal(t, "1.0.0", def["initial_version"])
}
