// Package prompt collects answers from a file and/or an interactive wizard,
// merges module-contributed questions, and validates the result.
package prompt

import "github.com/RomanAgaltsev/keel/internal/manifest"

// CoreQuestions are the built-in questions asked for every recipe (§6.3).
func CoreQuestions() []manifest.Question {
	return []manifest.Question{
		{ID: "repo_name", Prompt: "Repository name", Type: "string", Required: true},
		{ID: "description", Prompt: "Short description", Type: "string"},
		{ID: "module_path", Prompt: "Module path / namespace", Type: "string", Required: true},
		{ID: "author_name", Prompt: "Author name", Type: "string", Required: true},
		{ID: "author_email", Prompt: "Author email", Type: "string", Required: true},
		{ID: "license", Prompt: "License", Type: "select", Options: []string{"MIT", "Apache-2.0", "BSD-3-Clause", "none"}, Default: "MIT"},
		{ID: "visibility", Prompt: "Visibility", Type: "select", Options: []string{"public", "private"}, Default: "public"},
		{ID: "provider", Prompt: "Remote provider", Type: "select", Options: []string{"github", "gitlab", "bitbucket", "sourcecraft", "none"}, Default: "github"},
		{ID: "create_remote", Prompt: "Create the remote repository?", Type: "bool", Default: true},
	}
}
