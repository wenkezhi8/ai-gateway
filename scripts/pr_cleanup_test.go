package scripts_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func projectRootForPRCleanup(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
}

func TestPRCleanupScript_DryRunPrintsExpectedCommands(t *testing.T) {
	root := projectRootForPRCleanup(t)
	scriptPath := filepath.Join(root, "scripts", "pr-cleanup.sh")

	if _, err := os.Stat(scriptPath); err != nil {
		t.Fatalf("expected %s to exist: %v", scriptPath, err)
	}

	output, err := runScript(
		t,
		scriptPath,
		append(os.Environ(), "PATH="+os.Getenv("PATH")),
		"--worktree", "settings-trace-logo-remediation",
		"--branch", "feature/settings-trace-logo-remediation",
		"--dry-run",
	)
	if err != nil {
		t.Fatalf("expected dry-run to exit successfully, got err=%v output=%s", err, output)
	}

	expected := []string{
		"git worktree remove",
		"git checkout main",
		"git branch -D feature/settings-trace-logo-remediation",
		"git push origin --delete feature/settings-trace-logo-remediation",
	}
	for _, marker := range expected {
		if !strings.Contains(output, marker) {
			t.Fatalf("expected output to contain %q, output=%s", marker, output)
		}
	}
}
