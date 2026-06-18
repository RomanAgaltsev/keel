package prompt_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/internal/answers"
	"github.com/RomanAgaltsev/keel/internal/manifest"
	"github.com/RomanAgaltsev/keel/internal/prompt"
)

func coreAndModule() []manifest.Question {
	core := prompt.CoreQuestions()
	mod := []manifest.Question{{ID: "enable_codeql", Prompt: "CodeQL?", Type: "bool", Default: true}}
	merged, err := prompt.MergeQuestions(core, mod)
	if err != nil {
		panic(err)
	}
	return merged
}

func fullPreset() answers.Answers {
	return answers.Answers{
		"repo_name": "demo", "description": "d", "module_path": "github.com/x/demo",
		"author_name": "Roman", "author_email": "roman-agalcev@yandex.ru",
		"license": "MIT", "visibility": "public", "provider": "github", "create_remote": true,
	}
}

func TestCollectNonInteractiveFillsDefaults(t *testing.T) {
	got, err := prompt.Collect(coreAndModule(), fullPreset(), nil)
	require.NoError(t, err)
	require.Equal(t, "demo", got["repo_name"])
	require.Equal(t, true, got["enable_codeql"]) // default applied
}

func TestCollectNonInteractiveMissingRequiredErrors(t *testing.T) {
	preset := fullPreset()
	delete(preset, "repo_name")
	_, err := prompt.Collect(coreAndModule(), preset, nil)
	require.ErrorContains(t, err, "repo_name")
}

func TestCollectValidatesRules(t *testing.T) {
	preset := fullPreset()
	preset["module_path"] = "nopath"
	_, err := prompt.Collect(coreAndModule(), preset, nil)
	require.ErrorContains(t, err, "module path")
}

func TestMergeConflictErrors(t *testing.T) {
	core := prompt.CoreQuestions()
	bad := []manifest.Question{{ID: "repo_name", Prompt: "different", Type: "bool"}}
	_, err := prompt.MergeQuestions(core, bad)
	require.ErrorContains(t, err, "repo_name")
}

func TestCollectInteractiveAsksMissing(t *testing.T) {
	preset := fullPreset()
	delete(preset, "description") // optional, but asker can fill
	asker := prompt.AskerFunc(func(qs []manifest.Question, into answers.Answers) error {
		into["description"] = "from wizard"
		return nil
	})
	got, err := prompt.Collect(coreAndModule(), preset, asker)
	require.NoError(t, err)
	require.Equal(t, "from wizard", got["description"])
}
