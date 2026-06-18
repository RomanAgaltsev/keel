package prompt

import (
	"fmt"

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
		switch q.Type {
		case "bool":
			b := new(bool)
			if d, ok := q.Default.(bool); ok {
				*b = d
			}
			boolVals[q.ID] = b
			fields = append(fields, huh.NewConfirm().Title(q.Prompt).Value(b))
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
			fields = append(fields, huh.NewSelect[string]().Title(q.Prompt).Options(opts...).Value(s))
		default: // string and others fall back to text input
			s := new(string)
			if d, ok := q.Default.(string); ok {
				*s = d
			}
			strVals[q.ID] = s
			fields = append(fields, huh.NewInput().Title(q.Prompt).Value(s))
		}
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
