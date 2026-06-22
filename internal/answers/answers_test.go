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

func TestAnswersString(t *testing.T) {
	a := answers.Answers{"name": "demo", "enable_codeql": true}
	require.Equal(t, "demo", a.String("name"))
	require.Equal(t, "", a.String("missing"))       // absent key
	require.Equal(t, "", a.String("enable_codeql")) // non-string value
}
