package settings

import (
	"context"
	"fmt"
)

// Change is one setting that differs from the desired state. It is descriptive
// only: it exists to be printed and diffed, never executed. The Group that
// produced it holds the request body it will send.
type Change struct {
	Key      string // dotted path, e.g. "repository.has_wiki"
	From, To string // rendered current and desired values
}

// Failure is one group that could not be planned or applied, with a reason
// already classified into something a human can act on.
type Failure struct {
	Group  string
	Reason string
}

// Unsupported is a declared key the provider cannot express. It is reported
// rather than ignored: a key that silently does nothing is how a settings tool
// loses trust.
type Unsupported struct {
	Key      string
	Provider string
	Reason   string
}

// Report is the outcome of one reconcile run.
type Report struct {
	Changes     []Change // drift found; applied when Applied is true
	Applied     bool     // false under --check
	Failed      []Failure
	Unsupported []Unsupported
}

// InSync reports whether the provider already matches every declared key.
// Unsupported keys are not drift — they are not expressible either way.
func (r Report) InSync() bool { return len(r.Changes) == 0 }

// Group reconciles one API surface. Plan reads current state and returns the
// changes needed, staging whatever request body it computed on the receiver;
// Apply sends it. Splitting the two is what lets --check show a real diff
// without a second code path.
//
// A Group is stateful across the pair: Apply applies what the most recent Plan
// computed, and Reconcile is the only caller, so the order is guaranteed.
type Group interface {
	Name() string
	Plan(ctx context.Context, d Desired) ([]Change, error)
	Apply(ctx context.Context, changes []Change) error
}

// Unsupporter is implemented by Groups that can report declared keys their
// provider cannot express. Groups that support everything they are given need
// not implement it.
type Unsupporter interface {
	Unsupported(d Desired) []Unsupported
}

// Reconcile plans every group and, when apply is true, applies the groups that
// have changes. It never aborts: a group that fails to plan or apply is recorded
// in Report.Failed and the remaining groups still run. There is no whole-run
// error, by design — the caller in scaffold must not be able to fail a scaffold
// over a settings problem.
func Reconcile(ctx context.Context, groups []Group, d Desired, apply bool) Report {
	rep := Report{Applied: apply}
	for _, g := range groups {
		if u, ok := g.(Unsupporter); ok {
			rep.Unsupported = append(rep.Unsupported, u.Unsupported(d)...)
		}
		changes, err := g.Plan(ctx, d)
		if err != nil {
			rep.Failed = append(rep.Failed, Failure{Group: g.Name(), Reason: err.Error()})
			continue
		}
		if len(changes) == 0 {
			continue
		}
		rep.Changes = append(rep.Changes, changes...)
		if !apply {
			continue
		}
		if err := g.Apply(ctx, changes); err != nil {
			rep.Failed = append(rep.Failed, Failure{Group: g.Name(), Reason: err.Error()})
		}
	}
	return rep
}

// Errorf is a helper for drivers building a Failure-worthy error with a group
// prefix, mirroring the error style in internal/provider.
func Errorf(group, format string, args ...any) error {
	return fmt.Errorf("%s: %s", group, fmt.Sprintf(format, args...))
}
