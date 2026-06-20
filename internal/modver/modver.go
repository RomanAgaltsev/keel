// Package modver provides module-version change-gate logic: which modules a diff
// touched, whether their versions were bumped, and semver increments. It is pure
// (no git, no filesystem) so it is fully unit-testable.
package modver

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ModulesTouched returns the sorted, unique module names referenced by changed
// file path of the form "modules/<name>/...". Paths outside modules/ are ignored.
func ModulesTouched(changed []string) []string {
	seen := map[string]struct{}{}
	for _, p := range changed {
		p = strings.ReplaceAll(p, "\\", "/")
		parts := strings.Split(p, "/")
		if len(parts) >= 2 && parts[0] == "modules" && parts[1] != "" {
			seen[parts[1]] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// Offenders returns the touched modules whose version did not change between the
// base and head revisions. Modules absent from base (newly added) are not
// offenders. Order matches the input.
func Offenders(touched []string, base, head map[string]string) []string {
	var out []string
	for _, m := range touched {
		b, existed := base[m]
		if !existed {
			continue // new module - no prior version to bump from
		}
		if b == head[m] {
			out = append(out, m)
		}
	}
	return out
}

// Bump increments a semver string ("MAJOR.MINOR.PATCH", optionally "v"-prefixed)
// at level "patch", "minor", or "major".
func Bump(version, level string) (string, error) {
	nums, err := parseSemver(version)
	if err != nil {
		return "", err
	}
	switch level {
	case "major":
		nums[0], nums[1], nums[2] = nums[0]+1, 0, 0
	case "minor":
		nums[1], nums[2] = nums[1]+1, 0
	case "patch":
		nums[2]++
	default:
		return "", fmt.Errorf("unknown bump level %q", level)
	}
	out := fmt.Sprintf("%d.%d.%d", nums[0], nums[1], nums[2])
	if strings.HasPrefix(version, "v") {
		out = "v" + out
	}
	return out, nil
}

// Compare returns -1, 0, or 1 as a is less than, equal to, or greater than b.
// Both must be valid semver, optionally "v"-prefixed.
func Compare(a, b string) (int, error) {
	an, err := parseSemver(a)
	if err != nil {
		return 0, err
	}
	bn, err := parseSemver(b)
	if err != nil {
		return 0, err
	}
	for i := range 3 {
		switch {
		case an[i] < bn[i]:
			return -1, nil
		case an[i] > bn[i]:
			return 1, nil
		}
	}
	return 0, nil
}

func parseSemver(v string) ([3]int, error) {
	var out [3]int
	parts := strings.Split(strings.TrimPrefix(v, "v"), ".")
	if len(parts) != 3 {
		return out, fmt.Errorf("not a semver version: %q", v)
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, fmt.Errorf("not a semver version: %q", v)
		}
		out[i] = n
	}
	return out, nil
}
