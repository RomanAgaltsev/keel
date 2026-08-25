package render_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlanReportsUserOwnedFiles(t *testing.T) {
	owned := planForRecipe(t, "go-service").UserOwned()
	require.True(t, owned["go.mod"], "go.mod becomes the user's once they run go get")
	require.False(t, owned["Taskfile.yml"], "an ordinary file is not user-owned")
}

func TestUserOwnedMapIsACopy(t *testing.T) {
	// Mirrors Plan.Owner(): a caller mutating the returned map must not corrupt
	// the plan.
	p := planForRecipe(t, "go-service")
	p.UserOwned()["go.mod"] = false
	require.True(t, p.UserOwned()["go.mod"])
}
