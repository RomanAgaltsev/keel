package provider_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/internal/provider"
)

func TestFakeRecordsCreate(t *testing.T) {
	f := &provider.Fake{Repo: provider.RemoteRepo{CloneURL: "https://x/y.git"}}

	exists, _, err := f.RepoExists(context.Background(), provider.RepoSpec{Name: "y"})
	require.NoError(t, err)
	require.False(t, exists)

	repo, err := f.CreateRepo(context.Background(), provider.RepoSpec{Name: "y"})
	require.NoError(t, err)
	require.Equal(t, "https://x/y.git", repo.CloneURL)
	require.True(t, f.Created)
}
