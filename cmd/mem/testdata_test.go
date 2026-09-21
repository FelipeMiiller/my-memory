package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// compileFakeMymemoryd builds the testdata/fakemymemoryd helper
// into a temp dir and returns its absolute path. Each test that
// needs the helper calls this once via the shared setup fixture
// so the binary is built at most once per package test run.
func compileFakeMymemoryd(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	exe := "fakemymemoryd"
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	out := filepath.Join(tmpDir, exe)

	srcDir, err := filepath.Abs("testdata/fakemymemoryd")
	if err != nil {
		t.Fatalf("abs src: %v", err)
	}
	cmd := exec.Command("go", "build", "-o", out, srcDir)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("build fakemymemoryd: %v", err)
	}
	return out
}

// writeMemoryDir creates a minimal <dir>/.memory/profiles/default.yaml
// fixture so mymemoryd's profile gate (T1) passes when mem up
// spawns the fake supervisor.
func writeMemoryDir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "profiles"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	profile := `schema_version: 1
workers:
  - name: dummy
    command: .memory/workers/dummy
`
	if err := os.WriteFile(
		filepath.Join(dir, "profiles", "default.yaml"),
		[]byte(profile), 0o644,
	); err != nil {
		t.Fatalf("write profile: %v", err)
	}
}
