package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/supervisor"
)

// StopGrace is the SIGTERM → SIGKILL escalation timeout (tasks.md
// T8 done-when #2). After this duration the supervisor and its
// workers get a SIGKILL if they haven't exited.
const StopGrace = 10 * time.Second

// runDownCommand is the entry point for `mem down` (tasks.md T8
// done-when #2). Behavior:
//   - If no lock or stale lock → "Not running", exit 0.
//   - Send SIGTERM to the PID; wait up to StopGrace.
//   - If still alive after StopGrace → send SIGKILL.
//   - Exit 0 once the supervisor process is gone (success) or 1
//     if the PID couldn't be signalled.
func runDownCommand(ctx context.Context, args []string, memoryDir string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("down", flag.ContinueOnError)
	fs.SetOutput(stderr)
	memoryDirFlag := fs.String("memory-dir", memoryDir, "vault root (where supervisor.lock lives)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *memoryDirFlag == "" {
		*memoryDirFlag = ".memory"
	}

	lock, err := supervisor.ReadLock(*memoryDirFlag)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) ||
			errors.Is(err, supervisor.ErrLockCorrupt) {
			fmt.Fprintln(stdout, "Not running")
			return 0
		}
		fmt.Fprintf(stderr, "mem down: read lock: %v\n", err)
		return 1
	}
	if !supervisor.IsAlivePID(lock.PID) {
		// PID no longer alive — clean up the stale lock so the
		// next `mem up` doesn't trip the "already running" guard.
		_ = supervisor.ReleaseLock(*memoryDirFlag)
		fmt.Fprintln(stdout, "Not running")
		return 0
	}

	proc, err := os.FindProcess(lock.PID)
	if err != nil {
		fmt.Fprintf(stderr, "mem down: lookup PID %d: %v\n", lock.PID, err)
		return 1
	}

	// SIGTERM first (polite shutdown). On Windows signal is a
	// no-op so the grace path will fall through to SIGKILL.
	if err := proc.Signal(syscall.SIGTERM); err != nil && !isProcessGone(err) {
		fmt.Fprintf(stderr, "mem down: SIGTERM: %v\n", err)
		return 1
	}

	timer := time.NewTimer(StopGrace)
	defer timer.Stop()
	select {
	case <-timer.C:
		// Grace expired — escalate.
	case <-ctx.Done():
		_ = proc.Signal(syscall.SIGKILL)
		return 1
	}
	if supervisor.IsAlivePID(lock.PID) {
		if err := proc.Signal(syscall.SIGKILL); err != nil && !isProcessGone(err) {
			fmt.Fprintf(stderr, "mem down: SIGKILL: %v\n", err)
			return 1
		}
	}

	// Best-effort wait for the supervisor to release the lock.
	deadline := time.Now().Add(StopGrace)
	for time.Now().Before(deadline) {
		if !supervisor.IsAlivePID(lock.PID) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	fmt.Fprintln(stdout, "Stopped")
	return 0
}

// isProcessGone mirrors the supervisor helper so we don't import
// the lifecycle subpackage just for the sentinel match.
func isProcessGone(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return msg == "os: process already finished" ||
		strings.Contains(msg, "no such process") ||
		strings.Contains(msg, "invalid parameter") ||
		strings.Contains(msg, "not supported by windows")
}
