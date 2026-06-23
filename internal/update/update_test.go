package update_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/internal/lock"
	"github.com/RomanAgaltsev/keel/internal/update"
)

// diskHash builds a HashOf seam from an in-memory "disk" map.
func diskHash(disk map[string]string) func(string) (string, bool, error) {
	return func(path string) (string, bool, error) {
		c, ok := disk[path]
		if !ok {
			return "", false, nil
		}
		return lock.HashBytes([]byte(c)), true, nil
	}
}

func TestClassify(t *testing.T) {
	orig := "old"
	newC := "new"
	origHash := lock.HashBytes([]byte(orig))

	in := update.Input{
		Candidates:     map[string]bool{"lint": true},
		VersionChanged: map[string]bool{"lint": true},
		Owner: map[string]string{
			"clean.yml": "lint", "edited.yml": "lint", "added.yml": "lint",
		},
		Render: map[string]string{ // freshly rendered new content
			"clean.yml": newC, "edited.yml": newC, "added.yml": newC,
		},
		Original: map[string]map[string]string{"lint": {
			"clean.yml":   origHash,
			"edited.yml":  origHash,
			"removed.yml": origHash, // in lock, absent from new render ⇒ Removed
		}},
		HashOf: diskHash(map[string]string{
			"clean.yml":   orig,   // untouched ⇒ Clean
			"edited.yml":  "mine", // user-edited ⇒ Conflict
			"removed.yml": orig,
			// added.yml absent on disk ⇒ New
		}),
	}

	p, err := update.Classify(in)
	require.NoError(t, err)

	got := map[string]update.Class{}
	for _, c := range p.Changes {
		got[c.Path] = c.Class
	}
	require.Equal(t, update.Clean, got["clean.yml"])
	require.Equal(t, update.Conflict, got["edited.yml"])
	require.Equal(t, update.New, got["added.yml"])
	require.Equal(t, update.Removed, got["removed.yml"])

	// Changes are sorted by Path (deterministic output).
	require.True(t, len(p.Changes) == 4)
	require.Equal(t, "added.yml", p.Changes[0].Path)
}

func TestClassifySkipsNoOpAndNonCandidates(t *testing.T) {
	same := "same"
	in := update.Input{
		Candidates:     map[string]bool{"lint": true},
		VersionChanged: map[string]bool{"lint": true},
		Owner:          map[string]string{"a.yml": "lint", "b.yml": "other"},
		Render:         map[string]string{"a.yml": same, "b.yml": "x"},
		Original:       map[string]map[string]string{"lint": {"a.yml": lock.HashBytes([]byte(same))}},
		HashOf:         diskHash(map[string]string{"a.yml": same}), // on-disk == new render
	}
	p, err := update.Classify(in)
	require.NoError(t, err)
	require.Empty(t, p.Changes) // a.yml is a no-op; b.yml's module isn't a candidate
}

func TestClassifyV1ConservativeButPreciseWhenVersionUnchanged(t *testing.T) {
	newC := "new"
	// v1 lock ⇒ no Original hashes at all.
	base := update.Input{
		Candidates: map[string]bool{"lint": true},
		Owner:      map[string]string{"f.yml": "lint"},
		Render:     map[string]string{"f.yml": newC},
		Original:   map[string]map[string]string{}, // v1: nothing recorded
		HashOf:     diskHash(map[string]string{"f.yml": "mine"}),
	}

	// Version changed ⇒ no baseline ⇒ conservative Conflict.
	base.VersionChanged = map[string]bool{"lint": true}
	p, err := update.Classify(base)
	require.NoError(t, err)
	require.Equal(t, update.Conflict, p.Changes[0].Class)

	// Version unchanged (only reachable under --reconfigure) ⇒ the current render IS
	// the original baseline, so an unedited file is Clean.
	base.VersionChanged = map[string]bool{"lint": false}
	base.HashOf = diskHash(map[string]string{"f.yml": newC}) // matches the reconstructed baseline
	p, err = update.Classify(base)
	require.NoError(t, err)
	require.Empty(t, p.Changes) // reconstructed baseline == on disk == new ⇒ no-op
}
