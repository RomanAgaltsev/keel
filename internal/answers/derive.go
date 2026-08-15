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
	out := make(Answers, len(a)+3)
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
