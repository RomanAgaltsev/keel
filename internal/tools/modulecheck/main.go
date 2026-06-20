// Command modulecheck fails id any module changed in a diff (vs a base ref)
// without a version bump in its module.yaml. Not part of the keel binary.
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/RomanAgaltsev/keel/internal/modver"
)

func main() {
	base := "origin/main"
	if len(os.Args) > 1 {
		base = os.Args[1]
	}
	if err := run(base); err != nil {
		fmt.Fprintln(os.Stderr, "modulecheck:", err)
		os.Exit(1)
	}
}

func run(base string) error {
	changed, err := changedFiles(base)
	if err != nil {
		return err
	}
	touched := modver.ModulesTouched(changed)
	if len(touched) == 0 {
		return nil
	}
	prevVers, headVers := map[string]string{}, map[string]string{}
	for _, m := range touched {
		nv, err := versionAt("", m)
		if err != nil {
			return fmt.Errorf("read head version of %q: %w", m, err)
		}
		headVers[m] = nv
		if ov, err := versionAt(base, m); err == nil {
			prevVers[m] = ov // absent at base => new module, left out of prevVers
		}
	}
	if off := modver.Offenders(touched, prevVers, headVers); len(off) > 0 {
		return fmt.Errorf("modules changed without a version bump in module.yaml: %s\n"+
			"bump them (semver: patch=tool/SHA/typo, minor=new file/question, major=removed/renamed/retyped) "+
			"or run `task modules:bump`", strings.Join(off, ", "))
	}
	return nil
}

func changedFiles(base string) ([]string, error) {
	out, err := exec.CommandContext(context.Background(), "git", "diff", "--name-only", base+"...HEAD").Output() //nolint:gosec // dev tool; base is a local git ref
	if err != nil {
		return nil, fmt.Errorf("git diff against %q: %w", base, err)
	}
	var files []string
	for _, l := range strings.Split(string(out), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			files = append(files, l)
		}
	}
	return files, nil
}

// versionAt reads modules/<m>/module.yaml's version at git ref, or the working
// tree when ref == "".
func versionAt(ref, m string) (string, error) {
	path := "modules/" + m + "/module.yaml"
	var (
		b   []byte
		err error
	)
	if ref == "" {
		b, err = os.ReadFile(path)
	} else {
		b, err = exec.CommandContext(context.Background(), "git", "show", ref+":"+path).Output() //nolint:gosec // dev tool; ref and path are local
	}
	if err != nil {
		return "", err
	}
	var doc struct {
		Version string `yaml:"version"`
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return "", err
	}
	return doc.Version, nil
}
