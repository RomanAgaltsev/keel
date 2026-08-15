package update_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2/internal/lock"
	"github.com/RomanAgaltsev/keel/v2/internal/update"
)

// embedded is the version each module resolves to in this fake binary.
func prov(embedded map[string]string) update.Provenance {
	return func(name string) (string, string, error) {
		v, ok := embedded[name]
		if !ok {
			return "", "", fmt.Errorf("module %q not found", name)
		}
		return "builtin", v, nil
	}
}

func lk(mods ...lock.Module) lock.Lock { return lock.Lock{Modules: mods} }

func mod(name, version string) lock.Module {
	return lock.Module{Name: name, Source: "builtin", Version: version}
}

func TestResolveStates(t *testing.T) {
	got, err := update.Resolve(
		lk(mod("base-layout", "2.0.0"), mod("lint-go", "1.0.0"), mod("spell", "1.0.0")),
		[]string{"base-layout", "lint-go", "license"},
		prov(map[string]string{"base-layout": "2.0.0", "lint-go": "2.0.0", "license": "1.0.0"}),
		update.Options{},
	)
	require.NoError(t, err)
	require.Equal(t, update.Unchanged, got.State["base-layout"], "same version")
	require.Equal(t, update.Behind, got.State["lint-go"], "embedded is newer")
	require.Equal(t, update.Added, got.State["license"], "in recipe, not in lock")
	require.Equal(t, update.Orphaned, got.State["spell"], "in lock, not in recipe")
}

func TestResolveCandidatesAreBehindAndAdded(t *testing.T) {
	got, err := update.Resolve(
		lk(mod("base-layout", "1.0.0"), mod("spell", "1.0.0"), mod("lint-go", "2.0.0")),
		[]string{"base-layout", "lint-go", "license"},
		prov(map[string]string{"base-layout": "2.0.0", "lint-go": "2.0.0", "license": "1.0.0"}),
		update.Options{},
	)
	require.NoError(t, err)
	require.Equal(t,
		map[string]bool{"base-layout": true, "license": true},
		got.Candidates())
}

func TestResolveAddedIsVersionChanged(t *testing.T) {
	// classifyRendered treats "no baseline + version unchanged" as "the current
	// render is the baseline", which would misclassify a freshly added module.
	got, err := update.Resolve(
		lk(), []string{"license"},
		prov(map[string]string{"license": "1.0.0"}), update.Options{},
	)
	require.NoError(t, err)
	require.True(t, got.VersionChanged["license"])
	require.Equal(t, "1.0.0", got.Refreshed["license"])
	require.Equal(t, "builtin", got.Source["license"])
}

func TestResolveNeverDowngrades(t *testing.T) {
	// A lock ahead of the binary (user downgraded keel) must not rewrite files.
	got, err := update.Resolve(
		lk(mod("lint-go", "3.0.0")), []string{"lint-go"},
		prov(map[string]string{"lint-go": "2.0.0"}), update.Options{},
	)
	require.NoError(t, err)
	require.Equal(t, update.Unchanged, got.State["lint-go"])
	require.Empty(t, got.Candidates())
}

func TestResolveUnparseableVersionIsUnchanged(t *testing.T) {
	got, err := update.Resolve(
		lk(mod("lint-go", "not-semver")), []string{"lint-go"},
		prov(map[string]string{"lint-go": "2.0.0"}), update.Options{},
	)
	require.NoError(t, err)
	require.Equal(t, update.Unchanged, got.State["lint-go"])
}

func TestResolveReconfigurePromotesUnchanged(t *testing.T) {
	got, err := update.Resolve(
		lk(mod("base-layout", "2.0.0"), mod("spell", "1.0.0")),
		[]string{"base-layout"},
		prov(map[string]string{"base-layout": "2.0.0"}),
		update.Options{Reconfigure: true},
	)
	require.NoError(t, err)
	require.Equal(t, update.Behind, got.State["base-layout"], "reconfigure re-renders everything")
	require.Equal(t, update.Orphaned, got.State["spell"], "reconfigure does not resurrect a dropped module")
}

func TestResolveOnlyFilterNarrowsEverything(t *testing.T) {
	got, err := update.Resolve(
		lk(mod("base-layout", "1.0.0"), mod("spell", "1.0.0")),
		[]string{"base-layout", "license"},
		prov(map[string]string{"base-layout": "2.0.0", "license": "1.0.0"}),
		update.Options{Only: []string{"base-layout"}},
	)
	require.NoError(t, err)
	require.Equal(t, update.Behind, got.State["base-layout"])
	// --modules scopes additions and orphaning too, or scoping an update could
	// still delete an unrelated module's files.
	require.Equal(t, update.Unchanged, got.State["license"])
	require.Equal(t, update.Unchanged, got.State["spell"])
}

func TestResolveUnknownRecipeModuleErrors(t *testing.T) {
	_, err := update.Resolve(
		lk(), []string{"ghost"}, prov(map[string]string{}), update.Options{},
	)
	require.ErrorContains(t, err, "ghost")
}

func TestResolveOrderedAccessors(t *testing.T) {
	got, err := update.Resolve(
		lk(mod("spell", "1.0.0"), mod("typos-old", "1.0.0")),
		[]string{"license", "governance"},
		prov(map[string]string{"license": "1.0.0", "governance": "1.0.0"}),
		update.Options{},
	)
	require.NoError(t, err)
	// Added follows recipe order; orphaned is sorted. Neither may range a map.
	require.Equal(t, []string{"license", "governance"}, got.AddedModules())
	require.Equal(t, []string{"spell", "typos-old"}, got.OrphanedModules())
}
