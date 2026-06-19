package prompt

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/RomanAgaltsev/keel/internal/answers"
	"github.com/RomanAgaltsev/keel/internal/manifest"
)

// Wizard is an Asker backed by a charmbracelet/huh form.
type Wizard struct{}

// Ask prompts for each question and stores the result in into.
func (Wizard) Ask(qs []manifest.Question, into answers.Answers) error {
	fields := make([]huh.Field, 0, len(qs))
	// Backing storage the huh fields write through.
	strVals := map[string]*string{}
	boolVals := map[string]*bool{}

	for _, q := range qs {
		fields = append(fields, field(q, strVals, boolVals))
	}

	if err := huh.NewForm(huh.NewGroup(fields...)).Run(); err != nil {
		return fmt.Errorf("run wizard: %w", err)
	}
	for id, p := range strVals {
		into[id] = *p
	}
	for id, p := range boolVals {
		into[id] = *p
	}
	return nil
}

// field builds the huh field for q, recording its backing pointer in strVals or
// boolVals so Ask can copy the answered value back out after the form runs.
func field(q manifest.Question, strVals map[string]*string, boolVals map[string]*bool) huh.Field {
	switch q.Type {
	case "bool":
		b := new(bool)
		if d, ok := q.Default.(bool); ok {
			*b = d
		}
		boolVals[q.ID] = b
		return huh.NewConfirm().Title(q.Prompt).Value(b)
	case "select":
		s := new(string)
		if d, ok := q.Default.(string); ok {
			*s = d
		}
		strVals[q.ID] = s
		opts := make([]huh.Option[string], len(q.Options))
		for i, o := range q.Options {
			opts[i] = huh.NewOption(o, o)
		}
		return huh.NewSelect[string]().Title(q.Prompt).Options(opts...).Value(s)
	default: // string and others fall back to text input
		s := new(string)
		if d, ok := q.Default.(string); ok {
			*s = d
		}
		strVals[q.ID] = s
		in := huh.NewInput().Title(q.Prompt).Value(s)
		if v := validatorFor(q); v != nil {
			in = in.Validate(v)
		}
		return in
	}
}

// validatorFor returns the inline validator for a known question id (so invalid
// input is corrected in place instead of aborting the whole form afterwards), a
// required-non-empty check for required fields, or nil for free-form optional input.
func validatorFor(q manifest.Question) func(string) error {
	switch q.ID {
	case "repo_name":
		return answers.ValidateRepoName
	case "module_path":
		return answers.ValidateModulePath
	case "author_email":
		return answers.ValidateEmail
	}
	if q.Required {
		return func(v string) error {
			if strings.TrimSpace(v) == "" {
				return fmt.Errorf("%s is required", q.ID)
			}
			return nil
		}
	}
	return nil
}
