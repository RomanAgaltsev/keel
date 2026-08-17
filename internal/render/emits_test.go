package render_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/RomanAgaltsev/keel/v2"
	"github.com/RomanAgaltsev/keel/v2/internal/answers"
	"github.com/RomanAgaltsev/keel/v2/internal/manifest"
	"github.com/RomanAgaltsev/keel/v2/internal/module"
	"github.com/RomanAgaltsev/keel/v2/internal/render"
	"github.com/RomanAgaltsev/keel/v2/internal/settings"
)

// The guards in this file exist because the test they replace could not work.
// TestRepoSettingsGoChecksMatchEmittedJobNames compared the rendered settings
// file against a hardcoded list transcribed by eye -- both sides carried the
// same assumption, so it could not detect a settings/workflow divergence, and
// it hardcoded "test", the one context that could never report. A test that
// restates the code's belief instead of deriving it independently is the same
// claim written twice.
//
// These derive instead: they parse the workflows a module actually renders and
// compare the settings file against that.

// everyAnswer is every question any shipped module asks, all switched on, so a
// module can be rendered on its own and every conditional file appears.
func everyAnswer() answers.Answers {
	return answers.Answers{
		"repo_name": "demo", "description": "a demo", "module_path": "github.com/acme/demo",
		"author_name": "Roman Agaltsev", "author_email": "a@b.c", "license": "MIT", "year": 2026,
		"provider": "github", "visibility": "public", "code_owner": "acme",
		"dep_bot": "dependabot", "enable_codecov": true,
		"enable_codeql": true, "enable_govulncheck": true,
		"enable_cargo_audit": true, "enable_cargo_deny": true,
	}
}

// workflow is the slice of GitHub Actions syntax these guards care about.
type workflow struct {
	Jobs map[string]struct {
		Strategy struct {
			Matrix map[string]any `yaml:"matrix"`
		} `yaml:"strategy"`
		Steps []struct {
			Uses string `yaml:"uses"`
		} `yaml:"steps"`
	} `yaml:"jobs"`
}

// moduleWorkflows renders one module alone and parses every workflow it emits.
func moduleWorkflows(t *testing.T, name string) map[string]workflow {
	t.Helper()
	plan, err := render.BuildRecipe(module.NewFSLoader(keel.BuiltinFS), []string{name}, everyAnswer())
	require.NoError(t, err, "module %s must render on its own", name)

	out := map[string]workflow{}
	for path, body := range plan.Files {
		if !strings.HasPrefix(path, ".github/workflows/") {
			continue
		}
		var wf workflow
		require.NoError(t, yaml.Unmarshal([]byte(body), &wf), "parsing %s from %s", path, name)
		out[path] = wf
	}
	return out
}

// bareAction strips the version from a `uses:` reference.
//
// The two forms differ: a normal action is "owner/repo@v3", a Docker action is
// "docker://image:tag". Splitting only on "@" leaves the Docker tag attached,
// which is what the first run of this guard caught.
func bareAction(uses string) string {
	if rest, ok := strings.CutPrefix(uses, "docker://"); ok {
		image, _, _ := strings.Cut(rest, ":")
		return "docker://" + image
	}
	return strings.SplitN(uses, "@", 2)[0]
}

// thirdParty reports whether a `uses:` reference needs an allow-list entry.
// GitHub owns the actions and github organisations, and both are covered by
// github_owned_allowed, so neither needs naming.
func thirdParty(uses string) bool {
	switch {
	case uses == "":
		return false
	case strings.HasPrefix(uses, "actions/"), strings.HasPrefix(uses, "github/"):
		return false
	default:
		return true
	}
}

// GUARD 1: every third-party action a module's workflows use is declared.
//
// An undeclared action is not in the rendered allow-list, so its workflow gets
// startup_failure at 0s, reports no check at all, and cannot be re-run.
func TestEveryModuleDeclaresTheActionsItUses(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	names, err := l.ModuleNames()
	require.NoError(t, err)
	require.NotEmpty(t, names)

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			m, err := l.Load(name)
			require.NoError(t, err)
			require.NoError(t, m.Emits.Validate())

			for path, wf := range moduleWorkflows(t, name) {
				for job, spec := range wf.Jobs {
					for _, step := range spec.Steps {
						if !thirdParty(step.Uses) {
							continue
						}
						bare := bareAction(step.Uses)
						require.Contains(t, m.Emits.Actions, bare,
							"%s uses %s in %s (job %s) but does not declare it in emits.actions",
							name, step.Uses, path, job)
					}
				}
			}
		})
	}
}

// GUARD 2: every check a module declares is produced by a job that reports that
// exact context.
//
// A matrixed job reports one context per cell, so declaring its name would
// require a context nothing ever reports -- which is #53, and blocks every pull
// request while all runs show green.
func TestEveryDeclaredCheckHasAnUnmatrixedProducer(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	names, err := l.ModuleNames()
	require.NoError(t, err)

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			m, err := l.Load(name)
			require.NoError(t, err)
			if len(m.Emits.Checks) == 0 {
				return
			}

			unmatrixed := map[string]bool{}
			for _, wf := range moduleWorkflows(t, name) {
				for job, spec := range wf.Jobs {
					if len(spec.Strategy.Matrix) == 0 {
						unmatrixed[job] = true
					}
				}
			}

			for _, check := range m.Emits.Checks {
				require.True(t, unmatrixed[check],
					"%s declares check %q, but no unmatrixed job of that name exists; "+
						"a matrixed job reports `%s (os)` and never a bare %q, so the required check would never report",
					name, check, check, check)
			}
		})
	}
}

// GUARDS 3, 4 and 5: across every recipe keel ships, no required check lacks a
// producer, no used action is unpermitted, and no declared need is ungranted.
func TestRecipeSettingsSatisfyTheEmittedContract(t *testing.T) {
	for _, name := range []string{"go-service", "go-library", "rust-service"} {
		t.Run(name, func(t *testing.T) {
			raw, plan := renderRecipeSettings(t, name)

			var d settings.Desired
			require.NoError(t, yaml.Unmarshal([]byte(raw), &d))
			require.NoError(t, d.Validate())
			require.Len(t, d.Rulesets, 1)

			checks := plan.Answers["emitted_checks"].([]string)
			actions := plan.Answers["emitted_actions"].([]string)
			needs := plan.Answers["emitted_needs"].(map[string]bool)

			// GUARD 3: subset, not equality -- some checks are answer-gated, so
			// the ruleset may legitimately require fewer than the union.
			for _, req := range d.Rulesets[0].RequiredStatusChecks {
				require.Contains(t, checks, req,
					"ruleset requires %q but no module emits it -- every PR would sit BLOCKED with all runs green", req)
			}

			// GUARD 4: every action a workflow uses must be permitted.
			for _, act := range actions {
				want := act + "@*"
				if strings.HasPrefix(act, "docker://") {
					want = act
				}
				require.Contains(t, d.Actions.AllowedPatterns, want,
					"%s is used by a workflow but not permitted -- the run would startup_failure at 0s and report nothing", act)
			}

			// GUARD 5: every declared need must be granted.
			for need := range needs {
				switch need {
				case manifest.NeedCanApprovePullRequestReviews:
					require.NotNil(t, d.Actions.CanApprovePullRequestReviews)
					require.True(t, *d.Actions.CanApprovePullRequestReviews,
						"a module needs %s but the settings disable it", need)
				default:
					// Adding a Needs value without teaching this guard to check
					// it would leave the new capability unverified, which is how
					// this whole class of defect got in.
					t.Fatalf("guard 5 does not know how to check need %q; add a case", need)
				}
			}
		})
	}
}
