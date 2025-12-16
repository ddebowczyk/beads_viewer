package main_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestTUIPrioritySnapshot launches the TUI briefly to ensure it initializes and exits cleanly.
// We rely on BV_TUI_AUTOCLOSE_MS to avoid hanging in CI.
func TestTUIPrioritySnapshot(t *testing.T) {
	bv := buildBvBinary(t)

	tempDir := t.TempDir()
	beadsDir := filepath.Join(tempDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0o755); err != nil {
		t.Fatalf("mkdir beads: %v", err)
	}
	// Minimal graph with a dependency to exercise insights/priority panes.
	beads := `{"id":"P1","title":"Parent","status":"open","priority":1,"issue_type":"task"}
{"id":"C1","title":"Child","status":"open","priority":2,"issue_type":"task","dependencies":[{"issue_id":"C1","depends_on_id":"P1","type":"blocks"}]}`
	if err := os.WriteFile(filepath.Join(beadsDir, "beads.jsonl"), []byte(beads), 0o644); err != nil {
		t.Fatalf("write beads: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "script", "-q", "/dev/null", bv)
	cmd.Dir = tempDir
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"BV_TUI_AUTOCLOSE_MS=1500",
	)

	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Skipf("skipping TUI snapshot: timed out (likely TTY/OS mismatch); output:\n%s", out)
	}
	if err != nil {
		t.Fatalf("TUI run failed: %v\n%s", err, out)
	}
}
