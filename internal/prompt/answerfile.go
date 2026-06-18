package prompt

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/RomanAgaltsev/keel/internal/answers"
)

// LoadAnswersFile reads a YAML answers file into an Answers map.
func LoadAnswersFile(path string) (answers.Answers, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read answers file %q: %w", path, err)
	}
	var a answers.Answers
	if err := yaml.Unmarshal(b, &a); err != nil {
		return nil, fmt.Errorf("parse answers file %q: %w", path, err)
	}
	return a, nil
}
