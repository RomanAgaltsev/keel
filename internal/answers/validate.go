package answers

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	reRepoName = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	reEmail    = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

// ValidateRepoName checks a repository name (no spaces or slashes).
func ValidateRepoName(s string) error {
	if s == "" {
		return fmt.Errorf("repo name is required")
	}
	if !reRepoName.MatchString(s) {
		return fmt.Errorf("repo name %q may contain only letters, digits, '.', '_', '-'", s)
	}
	return nil
}

// ValidateModulePath checks a Go-style module path (host/path, no spaces).
func ValidateModulePath(s string) error {
	if !strings.Contains(s, "/") {
		return fmt.Errorf("module path %q must contain a '/' (e.g. github.com/you/repo)", s)
	}
	if strings.ContainsAny(s, " \t") {
		return fmt.Errorf("module path %q must not contain whitespace", s)
	}
	return nil
}

// ValidateEmail checks a basic email shape.
func ValidateEmail(s string) error {
	if !reEmail.MatchString(s) {
		return fmt.Errorf("invalid email %q", s)
	}
	return nil
}
