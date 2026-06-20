package outdated

import (
	"regexp"
	"sort"
	"strings"
)

// Pin is a discoverable version reference in a repo.
type Pin struct {
	Ref     string // "actions/checkout", "github/codeql-action/init", "golangci/golangci-lint"
	Current string // "v5" or "v2.3.0"
	Major   bool   // true: floating major tag (compare majors); false: full semver pin
}

// Repo returns the owner/repo to query for releases (the first two path segments).
func (p Pin) Repo() string {
	parts := strings.SplitN(p.Ref, "/", 3)
	if len(parts) >= 2 {
		return parts[0] + "/" + parts[1]
	}
	return p.Ref
}

var (
	reUses       = regexp.MustCompile(`uses:\s*([A-Za-z0-9._/-]+)@(v[0-9][^\s]*)`)
	reGolangci   = regexp.MustCompile(`(?:GOLANGCI_VERSION|GOLANGCI_LINT_VERSION):\s*["']?(v[0-9.]+)`)
	reGolangciWF = regexp.MustCompile(`version:\s*["']?(v[0-9.]+)["']?\s*#\s*golangci-lint`)
)

// ParsePins extracts action and golangci-lint pins from the given file contents
// (path -> bytes). Docker (`uses: docker://…`) and local (`uses: ./…`) refs are
// ignored. Results are deduped by (Ref, Current) and sorted by Ref.
func ParsePins(files map[string][]byte) []Pin {
	seen := map[string]Pin{}
	add := func(p Pin) { seen[p.Ref+"@"+p.Current] = p }

	for _, b := range files {
		s := string(b)
		for _, m := range reUses.FindAllStringSubmatch(s, -1) {
			ref, ver := m[1], m[2]
			if strings.HasPrefix(ref, ".") || strings.Contains(ref, "://") {
				continue // local or docker — skip
			}
			add(Pin{Ref: ref, Current: ver, Major: isMajorOnly(ver)})
		}
		for _, m := range reGolangci.FindAllStringSubmatch(s, -1) {
			add(Pin{Ref: "golangci/golangci-lint", Current: m[1], Major: false})
		}
		for _, m := range reGolangciWF.FindAllStringSubmatch(s, -1) {
			add(Pin{Ref: "golangci/golangci-lint", Current: m[1], Major: false})
		}
	}

	out := make([]Pin, 0, len(seen))
	for _, p := range seen {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Ref != out[j].Ref {
			return out[i].Ref < out[j].Ref
		}
		return out[i].Current < out[j].Current
	})
	return out
}

// isMajorOnly reports whether ver is a bare major tag like "v5" (vs "v5.2.1").
func isMajorOnly(ver string) bool {
	return !strings.Contains(strings.TrimPrefix(ver, "v"), ".")
}
