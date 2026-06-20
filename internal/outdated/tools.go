package outdated

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/RomanAgaltsev/keel/internal/modver"
)

// ToolUpdate reports a pin that is behind its latest upstream release.
type ToolUpdate struct {
	Repo    string
	Current string
	Latest  string
}

// Report is the full outdated result across both axes.
type Report struct {
	Tools   []ToolUpdate
	Modules []ModuleUpdate
}

// Empty reports whether nothing is outdated.
func (r Report) Empty() bool { return len(r.Tools) == 0 && len(r.Modules) == 0 }

// toolConcurrency bounds the in-flight release lookups, to stay gentle on
// GitHub's rate limiter while still overlapping the network latency.
const toolConcurrency = 6

// ToolUpdates resolves the latest release for each pin and returns those that are
// behind, plus a count of pins skipped (no release, or a comparison that could not
// be made). It is best-effort: per-pin failures are skipped, not fatal. Lookups run
// concurrently (bounded); the result is sorted, so output stays deterministic.
func ToolUpdates(ctx context.Context, pins []Pin, rc ReleaseClient) (ups []ToolUpdate, skipped int) {
	var (
		mu  sync.Mutex
		wg  sync.WaitGroup
		sem = make(chan struct{}, toolConcurrency) // bound in-flight lookups
	)
	for _, p := range pins {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			latest, err := rc.LatestTag(ctx, p.Repo())
			if err == nil {
				if behind, berr := isBehind(p, latest); berr != nil {
					err = berr
				} else if behind {
					mu.Lock()
					ups = append(ups, ToolUpdate{Repo: p.Repo(), Current: p.Current, Latest: latest})
					mu.Unlock()
				}
			}
			if err != nil { // best-effort: a per-pin failure is skipped, never fatal
				mu.Lock()
				skipped++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	sort.Slice(ups, func(i, j int) bool { return ups[i].Repo < ups[j].Repo })
	return ups, skipped
}

// isBehind compares a pin's current value to the latest release tag. For floating
// major tags it compares majors; for full pins it compares semver.
func isBehind(p Pin, latest string) (bool, error) {
	if p.Major {
		cur, err := majorOf(p.Current)
		if err != nil {
			return false, err
		}
		lat, err := majorOf(latest)
		if err != nil {
			return false, err
		}
		return lat > cur, nil
	}
	cmp, err := modver.Compare(p.Current, latest)
	if err != nil {
		return false, err
	}
	return cmp < 0, nil
}

// majorOf extracts the major number from a tag like "v5" or "v5.2.1".
func majorOf(tag string) (int, error) {
	s := strings.TrimPrefix(tag, "v")
	if i := strings.IndexByte(s, '.'); i >= 0 {
		s = s[:i]
	}
	return strconv.Atoi(s)
}
