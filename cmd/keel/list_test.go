package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListCommand(t *testing.T) {
	cmd := newListCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	require.NoError(t, cmd.Execute())

	s := out.String()
	require.Contains(t, s, "go-service")  // recipe
	require.Contains(t, s, "base-layout") // module
	require.Contains(t, s, "go-mod")
}
