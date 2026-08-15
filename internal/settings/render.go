package settings

import (
	"fmt"
	"io"
)

// Render writes a human-readable summary of the run. It is the single place the
// report is formatted, so `keel new` and `keel settings` print it identically.
func (r Report) Render(w io.Writer) {
	switch {
	case r.InSync() && r.Applied:
		fmt.Fprintln(w, "settings: in sync")
	case r.InSync():
		fmt.Fprintln(w, "settings: in sync (checked, nothing to change)")
	case r.Applied:
		fmt.Fprintf(w, "settings: applied %s\n", plural(len(r.Changes), "setting"))
	default:
		fmt.Fprintf(w, "settings: drift: %s differ\n", plural(len(r.Changes), "setting"))
	}
	for _, c := range r.Changes {
		fmt.Fprintf(w, "    %s: %s -> %s\n", c.Key, c.From, c.To)
	}
	for _, f := range r.Failed {
		fmt.Fprintf(w, "  ! %s: %s\n", f.Group, f.Reason)
	}
	for _, u := range r.Unsupported {
		fmt.Fprintf(w, "  - %s: not supported on %s (%s)\n", u.Key, u.Provider, u.Reason)
	}
}

// plural renders "1 setting" / "3 settings".
func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
