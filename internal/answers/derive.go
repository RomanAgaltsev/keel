package answers

import "strings"

// Archetype values a recipe may declare. A CLI produces a binary, so it behaves
// exactly like a service everywhere in the templates; only a library differs.
const (
	ArchetypeService = "service"
	ArchetypeLibrary = "library"
	ArchetypeCLI     = "cli"
)

// Derive returns a copy of a with the archetype-derived keys filled in.
//
// It exists because templates are parsed with missingkey=error: a template that
// names .is_library fails to render for any caller that did not set it, and most
// callers — including every render test — supply a deliberately minimal answer
// map. Deriving here means no call site has to remember.
//
// is_library and package_name are always recomputed rather than read from a, so
// the recipe stays the single source of truth for the archetype. Working on a
// copy keeps both keys out of the lockfile's answers map.
func Derive(a Answers) Answers {
	out := make(Answers, len(a)+5)
	for k, v := range a {
		out[k] = v
	}
	arch := out.String("archetype")
	if arch == "" {
		arch = ArchetypeService
	}
	out["archetype"] = arch
	out["is_library"] = arch == ArchetypeLibrary
	out["package_name"] = PackageName(out.String("repo_name"))

	// A library that ships 1.0.0 has made a public API-stability commitment on
	// its first commit, and in Go escaping it costs a /v2 import-path move that
	// every consumer must follow. The 0.x range exists for exactly this, and
	// archetype: library is keel's own signal that it applies. Nobody imports a
	// service, so 1.0.0 stays right there.
	out["initial_version"] = "1.0.0"
	out["pre_major"] = false
	if arch == ArchetypeLibrary {
		out["initial_version"] = "0.1.0"
		// Without this, release-please promotes the first breaking change from
		// 0.x straight to 1.0.0, undoing the choice at the first feat!.
		out["pre_major"] = true
	}
	return out
}

// PackageName converts a repository name into a legal Go package name: every
// character that is not a letter or digit is dropped and the rest is lowercased,
// so "go-thing" becomes "gothing". A result that is empty or starts with a digit
// is prefixed with "pkg", because neither is a legal Go identifier.
func PackageName(repo string) string {
	var b strings.Builder
	for _, r := range repo {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		}
	}
	s := b.String()
	if s == "" || (s[0] >= '0' && s[0] <= '9') {
		s = "pkg" + s
	}
	return s
}
