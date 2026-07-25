package e2e_test

import (
	"strings"
	"testing"
)

func TestVersionCommandPrintsBuildVersionWithoutStartingAgent(t *testing.T) {
	repo := tempRepo(t)
	runner := &fakeRunner{}
	dependencies := deps(t, repo, runner)
	dependencies.Version = "1.2.3"

	code, stdout, stderr := execute(t, []string{"--version"}, dependencies)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, "1.2.3") {
		t.Fatalf("stdout = %q, want build version", stdout)
	}
	if len(runner.seen) != 0 {
		t.Fatalf("runner calls = %d, want 0", len(runner.seen))
	}
}
