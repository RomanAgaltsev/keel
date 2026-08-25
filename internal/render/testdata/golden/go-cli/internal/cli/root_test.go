package cli_test

import (
	"bytes"
	"testing"

	"github.com/RomanAgaltsev/demo/internal/cli"
)

func TestVersionCommandReportsBuildInfo(t *testing.T) {
	cmd := cli.NewRootCmd(cli.BuildInfo{Version: "1.2.3", Commit: "abc1234", Date: "2026-01-01"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("version command failed: %v", err)
	}
	want := "demo 1.2.3 (abc1234, 2026-01-01)\n"
	if got := out.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
