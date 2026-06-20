package prompt

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/RomanAgaltsev/keel/internal/answers"
	"github.com/RomanAgaltsev/keel/internal/manifest"
)

// Wizard is an Asker backed by a charmbracelet/huh form.
type Wizard struct{}

// vals holds the pointers the huh fields write through, so Ask can copy each
// answered value back into the Answers map after the form runs.
type vals struct {
	str   map[string]*string
	bool_ map[string]*bool
	multi map[string]*[]string
}

// Ask prompts for each question and stores the result in into.
func (Wizard) Ask(qs []manifest.Question, into answers.Answers) error {
	v := vals{
		str:   map[string]*string{},
		bool_: map[string]*bool{},
		multi: map[string]*[]string{},
	}
	fields := make([]huh.Field, 0, len(qs))
	for _, q := range qs {
		fields = append(fields, field(q, v))
	}

	if err := huh.NewForm(huh.NewGroup(fields...)).Run(); err != nil {
		return fmt.Errorf("run wizard: %w", err)
	}
	for id, p := range v.str {
		into[id] = *p
	}
	for id, p := range v.bool_ {
		into[id] = *p
	}
	for id, p := range v.multi {
		into[id] = *p
	}
	return nil
}

// field builds the huh field for q, recording its backing pointer in v so Ask can
// copy the answered value back out after the form runs. int values are collected as
// numeric-validated strings and coerced to a real int later, in Collect.
func field(q manifest.Question, v vals) huh.Field {
	switch q.Type {
	case "bool":
		b := new(bool)
		if d, ok := q.Default.(bool); ok {
			*b = d
		}
		v.bool_[q.ID] = b
		return huh.NewConfirm().Title(q.Prompt).Value(b)
	case "select":
		s := new(string)
		if d, ok := q.Default.(string); ok {
			*s = d
		}
		v.str[q.ID] = s
		return huh.NewSelect[string]().Title(q.Prompt).Options(stringOptions(q.Options)...).Value(s)
	case "multiselect":
		sel := new([]string)
		if d, ok := toStringSlice(q.Default); ok {
			*sel = d
		}
		v.multi[q.ID] = sel
		return huh.NewMultiSelect[string]().Title(q.Prompt).Options(stringOptions(q.Options)...).Value(sel)
	default: // string and int both render as a text input (int gets a numeric validator)
		s := new(string)
		if q.Default != nil {
			*s = fmt.Sprint(q.Default)
		}
		v.str[q.ID] = s
		in := huh.NewInput().Title(q.Prompt).Value(s)
		if val := validatorFor(q); val != nil {
			in = in.Validate(val)
		}
		return in
	}
}

// stringOptions builds huh options whose key and value are the option string.
func stringOptions(opts []string) []huh.Option[string] {
	out := make([]huh.Option[string], len(opts))
	for i, o := range opts {
		out[i] = huh.NewOption(o, o)
	}
	return out
}

// toStringSlice coerces a question default into a []string (for multiselect),
// accepting either a []string or a YAML-decoded []any of strings.
func toStringSlice(d any) ([]string, bool) {
	switch s := d.(type) {
	case []string:
		return s, true
	case []any:
		out := make([]string, 0, len(s))
		for _, e := range s {
			str, ok := e.(string)
			if !ok {
				return nil, false
			}
			out = append(out, str)
		}
		return out, true
	default:
		return nil, false
	}
}

// validatorFor returns the inline validator for a known question id (so invalid
// input is corrected in place instead of aborting the whole form afterwards), a
// numeric check for int questions, a required-non-empty check for required fields,
// or nil for free-form optional input.
func validatorFor(q manifest.Question) func(string) error {
	switch q.ID {
	case "repo_name":
		return answers.ValidateRepoName
	case "module_path":
		return answers.ValidateModulePath
	case "author_email":
		return answers.ValidateEmail
	}
	if q.Type == "int" {
		return func(val string) error {
			val = strings.TrimSpace(val)
			if val == "" {
				if q.Required {
					return fmt.Errorf("%s is required", q.ID)
				}
				return nil
			}
			if _, err := strconv.Atoi(val); err != nil {
				return fmt.Errorf("%s must be an integer", q.ID)
			}
			return nil
		}
	}
	if q.Required {
		return func(val string) error {
			if strings.TrimSpace(val) == "" {
				return fmt.Errorf("%s is required", q.ID)
			}
			return nil
		}
	}
	return nil
}
