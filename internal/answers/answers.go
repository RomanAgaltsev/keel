// Package answers holds the answer values fed to template rendering.
package answers

// Answers maps question IDs (and core fields) to their answered values.
type Answers map[string]any

// Bool reports whether key holds a boolean true.
func (a Answers) Bool(key string) bool {
	v, ok := a[key].(bool)
	return ok && v
}

// String returns the string value at key, or "" if it is absent or not a string.
func (a Answers) String(key string) string {
	s, _ := a[key].(string)
	return s
}
