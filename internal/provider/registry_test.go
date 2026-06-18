package provider_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/internal/provider"
)

func TestForGitHubNeedsToken(t *testing.T) {
	_, err := provider.For("github", provider.Env{}) // no token
	require.ErrorContains(t, err, "KEEL_GITHUB_TOKEN")
}

func TestForGitHubOK(t *testing.T) {
	p, err := provider.For("github", provider.Env{Token: "tok", Owner: "me"})
	require.NoError(t, err)
	require.Equal(t, "github", p.Name())
}

func TestForNoneIsNil(t *testing.T) {
	p, err := provider.For("none", provider.Env{})
	require.NoError(t, err)
	require.Nil(t, p)
}
