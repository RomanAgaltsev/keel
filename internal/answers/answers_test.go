package answers_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/internal/answers"
)

func TestAnswersBool(t *testing.T) {
	a := answers.Answers{"enable_codeql": true, "name": "x"}
	require.True(t, a.Bool("enable_codeql"))
	require.False(t, a.Bool("missing"))
	require.False(t, a.Bool("name")) // non-bool value is not true
}
