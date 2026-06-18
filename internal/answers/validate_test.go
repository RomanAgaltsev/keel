package answers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateRepoName(t *testing.T) {
	require.NoError(t, ValidateRepoName("my-repo_1.0"))
	require.Error(t, ValidateRepoName(""))
	require.Error(t, ValidateRepoName("has space"))
	require.Error(t, ValidateRepoName("bad/slash"))
}

func TestValidateModulePath(t *testing.T) {
	require.NoError(t, ValidateModulePath("github.com/RomanAgaltsev/keel"))
	require.Error(t, ValidateModulePath("nopath"))
	require.Error(t, ValidateModulePath("has space/x"))
}

func TestValidateEmail(t *testing.T) {
	require.NoError(t, ValidateEmail("roman-agalcev@yandex.ru"))
	require.Error(t, ValidateEmail("nope"))
}
