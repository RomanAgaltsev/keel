package provider_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/internal/provider"
)

func TestResolveNoneIsNil(t *testing.T) {
	p, err := provider.Resolve("none", "github.com/me/repo")
	require.NoError(t, err)
	require.Nil(t, p)
}

func TestResolveUnknown(t *testing.T) {
	_, err := provider.Resolve("frobnicator", "github.com/me/repo")
	require.ErrorContains(t, err, "unknown provider")
}

func TestResolveGitHubFromEnv(t *testing.T) {
	t.Setenv("KEEL_GITHUB_TOKEN", "tok")
	p, err := provider.Resolve("github", "github.com/octocat/demo")
	require.NoError(t, err)
	require.Equal(t, "github", p.Name())
}

func TestResolveGitHubMissingToken(t *testing.T) {
	t.Setenv("KEEL_GITHUB_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	_, err := provider.Resolve("github", "github.com/octocat/demo")
	require.ErrorContains(t, err, "KEEL_GITHUB_TOKEN")
}

func TestResolveOwnerOverride(t *testing.T) {
	t.Setenv("KEEL_GITHUB_TOKEN", "tok")
	t.Setenv("KEEL_GITHUB_OWNER", "override-org")
	// Owner override is accepted (no error); the provider builds.
	p, err := provider.Resolve("github", "github.com/octocat/demo")
	require.NoError(t, err)
	require.Equal(t, "github", p.Name())
}

func TestResolveGitLab(t *testing.T) {
	t.Setenv("KEEL_GITLAB_TOKEN", "tok")
	p, err := provider.Resolve("gitlab", "gitlab.com/mygroup/demo")
	require.NoError(t, err)
	require.Equal(t, "gitlab", p.Name())
}

func TestResolveBitbucket(t *testing.T) {
	t.Setenv("KEEL_BITBUCKET_TOKEN", "tok")
	p, err := provider.Resolve("bitbucket", "bitbucket.org/myws/demo")
	require.NoError(t, err)
	require.Equal(t, "bitbucket", p.Name())
}

func TestResolveSourceCraft(t *testing.T) {
	t.Setenv("KEEL_SOURCECRAFT_TOKEN", "tok")
	p, err := provider.Resolve("sourcecraft", "sourcecraft.dev/myorg/demo")
	require.NoError(t, err)
	require.Equal(t, "sourcecraft", p.Name())
}
