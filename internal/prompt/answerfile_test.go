package prompt_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/internal/prompt"
)

func TestLoadAnswersFile(t *testing.T) {
	a, err := prompt.LoadAnswersFile(filepath.Join("testdata", "answers.yaml"))
	require.NoError(t, err)
	require.Equal(t, "demo", a["repo_name"])
	require.Equal(t, true, a["create_remote"])
	require.Equal(t, true, a["enable_codeql"])
}

func TestLoadAnswersFileMissing(t *testing.T) {
	_, err := prompt.LoadAnswersFile(filepath.Join("testdata", "nope.yaml"))
	require.Error(t, err)
}
