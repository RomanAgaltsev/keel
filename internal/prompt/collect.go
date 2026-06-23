package prompt

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/RomanAgaltsev/keel/internal/answers"
	"github.com/RomanAgaltsev/keel/internal/manifest"
)

// supportedTypes are the question types keel knows how to prompt for and render.
var supportedTypes = map[string]bool{
	"": true, "string": true, "bool": true, "select": true, "multiselect": true, "int": true,
}

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
// A duplicate ID whose definition differs from the first (in any field —
// type, prompt, options, default, or required) is an error.
func MergeQuestions(core, mod []manifest.Question) ([]manifest.Question, error) {
	seen := map[string]manifest.Question{}
	out := make([]manifest.Question, 0, len(core)+len(mod))
	for _, q := range append(append([]manifest.Question{}, core...), mod...) {
		if !supportedTypes[q.Type] {
			return nil, fmt.Errorf("question %q: unsupported type %q", q.ID, q.Type)
		}
		if prev, ok := seen[q.ID]; ok {
			if !reflect.DeepEqual(prev, q) {
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
	if err := askMissing(qs, out, asker); err != nil {
		return nil, err
	}
	if err := applyDefaults(qs, out); err != nil {
		return nil, err
	}
	if err := coerceInts(qs, out); err != nil {
		return nil, err
	}
	if err := validate(out); err != nil {
		return nil, err
	}
	return out, nil
}

// askMissing fills any unanswered questions interactively (no-op when asker is nil).
func askMissing(qs []manifest.Question, out answers.Answers, asker Asker) error {
	if asker == nil {
		return nil
	}
	var missing []manifest.Question
	for _, q := range qs {
		if _, ok := out[q.ID]; !ok {
			missing = append(missing, q)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	if err := asker.Ask(missing, out); err != nil {
		return fmt.Errorf("interactive prompt: %w", err)
	}
	return nil
}

// applyDefaults fills still-unanswered questions from their default, and errors on
// a missing required answer.
func applyDefaults(qs []manifest.Question, out answers.Answers) error {
	for _, q := range qs {
		if _, ok := out[q.ID]; ok {
			continue
		}
		if q.Default != nil {
			out[q.ID] = q.Default
			continue
		}
		if q.Required {
			return fmt.Errorf("missing required answer %q", q.ID)
		}
	}
	return nil
}

// coerceInts normalizes int-typed answers to a Go int. The wizard collects them as
// strings, and an answers file may carry them as strings too; rendering and any
// numeric use expect a real number.
func coerceInts(qs []manifest.Question, a answers.Answers) error {
	for _, q := range qs {
		if q.Type != "int" {
			continue
		}
		v, ok := a[q.ID]
		if !ok {
			continue
		}
		switch n := v.(type) {
		case int:
			// already numeric
		case string:
			n = strings.TrimSpace(n)
			if n == "" {
				// An optional int left blank in the wizard arrives as "". Treat it as
				// unset rather than a parse error; applyDefaults/required already ran.
				delete(a, q.ID)
				continue
			}
			parsed, err := strconv.Atoi(n)
			if err != nil {
				return fmt.Errorf("answer %q must be an integer: %q", q.ID, n)
			}
			a[q.ID] = parsed
		default:
			// int64/float64 from a YAML decoder, etc. — leave as-is.
		}
	}
	return nil
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
