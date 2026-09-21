// fakemymemoryd is a test helper binary used by the CLI subcommand
// tests (cmd/mem/up_test.go et al). It mimics a long-running
// mymemoryd: writes a supervisor.lock, prints the readiness line
// on stdout, then blocks until SIGTERM (graceful) or SIGKILL.
//
// Args (env vars):
//
//	FAKE_READY_DELAY_MS  — sleep before writing lock + printing
//	                       readiness line (default 0).
//	FAKE_FAIL_IMMEDIATELY — when "1", exit 1 BEFORE writing the
//	                        lock or printing ready. Used by tests
//	                        that exercise the "exited before
//	                        ready" path in runUpCommand.
//	FAKE_FAIL_AFTER_READY — when "1", exit 1 AFTER printing the
//	                        readiness line (the lock is still
//	                        written).
//
// Output is line-buffered via fmt.Println so the test's stdout
// scanner sees the readiness line promptly.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

func main() {
	if os.Getenv("FAKE_FAIL_IMMEDIATELY") == "1" {
		os.Exit(1)
	}

	readyDelay := 0
	if raw := os.Getenv("FAKE_READY_DELAY_MS"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			readyDelay = v
		}
	}
	if readyDelay > 0 {
		time.Sleep(time.Duration(readyDelay) * time.Millisecond)
	}

	// Write a minimal supervisor.lock so `mem up`'s idempotent
	// guard (T8 done-when #8) can detect the running instance on
	// the second invocation. Real mymemoryd writes the full
	// {format, pid, started_at, profile} payload; tests don't care
	// about format/profile, just pid + recent timestamp.
	memDir := os.Getenv("FAKE_MEMORY_DIR")
	if memDir == "" {
		memDir = ".memory"
	}
	if err := os.MkdirAll(memDir, 0o755); err == nil {
		lock := map[string]any{
			"format":     1,
			"pid":        os.Getpid(),
			"started_at": time.Now().UTC(),
			"profile":    "default",
		}
		if data, err := json.Marshal(lock); err == nil {
			_ = os.WriteFile(filepath.Join(memDir, "supervisor.lock"), data, 0o644)
		}
	}

	fmt.Println("mymemoryd: ready (pid=test profile=default lock=.memory/supervisor.lock)")
	if os.Getenv("FAKE_FAIL_AFTER_READY") == "1" {
		os.Exit(1)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh
}
