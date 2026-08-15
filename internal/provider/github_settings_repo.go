package provider

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/RomanAgaltsev/keel/v2/internal/settings"
)

// diffBool stages field into body and records a Change when want is declared and
// differs from got. Undeclared (nil) means "leave alone" and is the whole reason
// the schema uses pointers.
func diffBool(out *[]settings.Change, body map[string]any, key, field string, got bool, want *bool) {
	if want == nil || *want == got {
		return
	}
	body[field] = *want
	*out = append(*out, settings.Change{Key: key, From: strconv.FormatBool(got), To: strconv.FormatBool(*want)})
}

// ghRepoGroup reconciles the general repository options, all of which the single
// PATCH /repos/{owner}/{repo} call sets at once.
type ghRepoGroup struct {
	gh   *GitHub
	url  string
	body map[string]any // staged by Plan, sent by Apply
}

func (r *ghRepoGroup) Name() string { return "repository" }

// ghRepoState is the subset of the repo object this group reconciles.
type ghRepoState struct {
	AllowSquashMerge    bool `json:"allow_squash_merge"`
	AllowMergeCommit    bool `json:"allow_merge_commit"`
	AllowRebaseMerge    bool `json:"allow_rebase_merge"`
	AllowAutoMerge      bool `json:"allow_auto_merge"`
	DeleteBranchOnMerge bool `json:"delete_branch_on_merge"`
	HasWiki             bool `json:"has_wiki"`
	HasProjects         bool `json:"has_projects"`
}

func (r *ghRepoGroup) Plan(ctx context.Context, d settings.Desired) ([]settings.Change, error) {
	if d.Repository == nil {
		return nil, nil
	}
	var cur ghRepoState
	if err := r.gh.getJSON(ctx, r.url, &cur); err != nil {
		return nil, err
	}
	body := map[string]any{}
	var out []settings.Change
	w := d.Repository
	diffBool(&out, body, "repository.allow_squash_merge", "allow_squash_merge", cur.AllowSquashMerge, w.AllowSquashMerge)
	diffBool(&out, body, "repository.allow_merge_commit", "allow_merge_commit", cur.AllowMergeCommit, w.AllowMergeCommit)
	diffBool(&out, body, "repository.allow_rebase_merge", "allow_rebase_merge", cur.AllowRebaseMerge, w.AllowRebaseMerge)
	diffBool(&out, body, "repository.allow_auto_merge", "allow_auto_merge", cur.AllowAutoMerge, w.AllowAutoMerge)
	diffBool(&out, body, "repository.delete_branch_on_merge", "delete_branch_on_merge", cur.DeleteBranchOnMerge, w.DeleteBranchOnMerge)
	diffBool(&out, body, "repository.has_wiki", "has_wiki", cur.HasWiki, w.HasWiki)
	diffBool(&out, body, "repository.has_projects", "has_projects", cur.HasProjects, w.HasProjects)
	r.body = body
	return out, nil
}

func (r *ghRepoGroup) Apply(ctx context.Context) error {
	if len(r.body) == 0 {
		return nil
	}
	return r.gh.sendJSON(ctx, http.MethodPatch, r.url, r.body)
}

// ghTopicsGroup reconciles repository topics. PUT replaces the whole list, so
// declaring topics hands keel ownership of all of them.
type ghTopicsGroup struct {
	gh   *GitHub
	url  string
	want []string
}

func (t *ghTopicsGroup) Name() string { return "topics" }

func (t *ghTopicsGroup) Plan(ctx context.Context, d settings.Desired) ([]settings.Change, error) {
	if d.Repository == nil || d.Repository.Topics == nil {
		return nil, nil
	}
	var cur struct {
		Names []string `json:"names"`
	}
	if err := t.gh.getJSON(ctx, t.url, &cur); err != nil {
		return nil, err
	}
	want := *d.Repository.Topics
	if equalStrings(cur.Names, want) {
		return nil, nil
	}
	t.want = want
	return []settings.Change{{
		Key:  "repository.topics",
		From: fmt.Sprint(cur.Names),
		To:   fmt.Sprint(want),
	}}, nil
}

func (t *ghTopicsGroup) Apply(ctx context.Context) error {
	if t.want == nil {
		return nil
	}
	return t.gh.sendJSON(ctx, http.MethodPut, t.url, map[string]any{"names": t.want})
}

// equalStrings compares two lists order-sensitively. Topic order is meaningful
// to the API's round-trip, so a reorder is a real change.
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
