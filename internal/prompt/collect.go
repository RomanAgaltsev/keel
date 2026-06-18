package prompt

import (
	"fmt"

	"github.com/RomanAgaltsev/keel/internal/answers"
	"github.com/RomanAgaltsev/keel/internal/manifest"
)

// Asker supplies answers for questions still unanswered after the preset.
type Asker interface {
	Ask(qs []manifest.Question, into answers.Answers) error
}

// AskerFunc adapts a function to the Asker interface.
type AskerFunc func(qs []manifest.Question, into answers.Answers) error

// AskerFunc implements Asker.
func (f AskerFunc) Ask(qs []manifest.Question, into answers.Answers) error {
	return f(qs, into)
}

// MergeQuestions concatenates core + module questions, deduping by ID.
// A duplicate ID whoose definition differs from the first is an error.
func MergeQuestions(core, mod []manifest.Question) ([]manifest.Question, error) {
	seen := map[string]manifest.Question{}
	out := make([]manifest.Question, 0, len(core)+len(mod))
	for _, q := range append(append([]manifest.Question{}, core...), mod...) {
		if prev, ok := seen[q.ID]; ok {
			if prev.Type != q.Type || prev.Prompt != q.Prompt {
				return nil, fmt.Errorf("question %q defined twice with conflicting definitions", q.ID)
			}
			continue
		}
		seen[q.ID] = q
		out = append(out, q)
	}
	return out, nil
}

// Collect resolves a final answer set: preset values win. An Asker (if non-nil)
// fills the rest interactively. Otherwise defaults apply and a missing required
// answer is an error. Known IDs are validated by rule.
func Collect(qs []manifest.Question, preset answers.Answers, asker Asker) (answers.Answers, error) {
	out := answers.Answers{}
	for k, v := range preset {
		out[k] = v
	}

	if asker != nil {
		var missing []manifest.Question
		for _, q := range qs {
			if _, ok := out[q.ID]; !ok {
				missing = append(missing, q)
			}
		}
		if len(missing) > 0 {
			if err := asker.Ask(missing, out); err != nil {
				return nil, fmt.Errorf("interactive prompt: %w", err)
			}
		}
	}

	for _, q := range qs {
		if _, ok := out[q.ID]; ok {
			continue
		}
		if q.Default != nil {
			out[q.ID] = q.Default
			continue
		}
		if q.Required {
			return nil, fmt.Errorf("missing required answer %q", q.ID)
		}
	}

	if err := validate(out); err != nil {
		return nil, err
	}
	return out, nil
}

// validate applies the known per-ID rules to the resolved answers.
func validate(a answers.Answers) error {
	if v, ok := a["repo_name"].(string); ok {
		if err := answers.ValidateRepoName(v); err != nil {
			return err
		}
	}
	if v, ok := a["module_path"].(string); ok {
		if err := answers.ValidateModulePath(v); err != nil {
			return err
		}
	}
	if v, ok := a["author_email"].(string); ok {
		if err := answers.ValidateEmail(v); err != nil {
			return err
		}
	}
	return nil
}
