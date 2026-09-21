package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/supervisor"
)

// MymemorydReadyTimeout is how long `mem up` waits for the
// subprocess to print the readiness line on stdout before
// declaring the supervisor healthy. Mirrors the handshake TTL
// discussed in the mymemoryd-supervisor spec (P1 AC 4 —
// fail-closed within 5s).
const MymemorydReadyTimeout = 5 * time.Second

// runUpCommand is the entry point for `mem up [--profile <name>]`
// (tasks.md T8 done-when #1). Behavior:
//   - Reads the lock file at <memoryDir>/supervisor.lock.
//   - If a fresh lock with a live PID is present, prints
//     "Already running on PID <N>" and exits 0 (idempotent,
//     done-when #8).
//   - Otherwise spawns the `mymemoryd` binary and waits for it to
//     emit the readiness line on stdout. Returns 0 on success,
//     non-zero on failure to start or handshake timeout.
//
// mymemorydPath may be empty — `mem up` then resolves the binary
// via MYMEMORYD_BIN env var, then sibling-of-mem, then PATH. Tests
// inject a fake binary via the explicit argument.
func runUpCommand(ctx context.Context, args []string, memoryDir, mymemorydPath string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("up", flag.ContinueOnError)
	fs.SetOutput(stderr)
	profile := fs.String("profile", "default", "profile name under <memory-dir>/profiles/")
	memoryDirFlag := fs.String("memory-dir", memoryDir, "vault root (where supervisor.lock lives)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *memoryDirFlag == "" {
		*memoryDirFlag = ".memory"
	}

	// Step 1: idempotent guard — if a fresh lock is present and the
	// PID is alive, exit 0 with a friendly message instead of
	// spawning a second supervisor (done-when #8).
	existing, err := supervisor.ReadLock(*memoryDirFlag)
	if err == nil {
		if time.Since(existing.StartedAt) < supervisor.StaleTimeout &&
			supervisor.IsAlivePID(existing.PID) {
			fmt.Fprintf(stdout, "Already running on PID %d\n", existing.PID)
			return 0
		}
		// Stale or dead PID: fall through and spawn a fresh
		// supervisor. The spawned mymemoryd will overwrite the
		// lock on its own startup.
	} else if !errors.Is(err, os.ErrNotExist) &&
		!errors.Is(err, supervisor.ErrLockCorrupt) {
		fmt.Fprintf(stderr, "mem up: read lock: %v\n", err)
		return 1
	}

	// Step 2: locate the mymemoryd binary.
	if mymemorydPath == "" {
		var lookupErr error
		mymemorydPath, lookupErr = resolveMymemorydPath()
		if lookupErr != nil {
			fmt.Fprintf(stderr, "mem up: locate mymemoryd: %v\n", lookupErr)
			return 1
		}
	}

	cmd := exec.CommandContext(ctx, mymemorydPath,
		"--profile", *profile,
		"--memory-dir", *memoryDirFlag,
	)

	// Capture stdout through a pipe so we can both tee to the
	// caller's writer and detect the readiness line. Without the
	// pipe we can't observe the supervisor's startup signal without
	// blocking on cmd.Wait().
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		fmt.Fprintf(stderr, "mem up: pipe: %v\n", err)
		return 1
	}
	cmd.Stdout = stdoutW
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		stdoutR.Close()
		stdoutW.Close()
		fmt.Fprintf(stderr, "mem up: spawn mymemoryd: %v\n", err)
		return 1
	}
	stdoutW.Close() // parent doesn't write; close so the reader sees EOF on exit

	ready := make(chan error, 1)
	processExited := make(chan struct{})
	go func() {
		// Tee stdout to caller's writer while watching for the
		// readiness line. mymemoryd emits "mymemoryd: ready
		// (pid=...)" right after lock acquisition.
		scanner := bufio.NewScanner(stdoutR)
		scanner.Buffer(make([]byte, 0, 1<<16), 1<<16)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Fprintln(stdout, line)
			if strings.HasPrefix(line, "mymemoryd: ready") {
				ready <- nil
				return
			}
		}
		// Scanner stopped (EOF or error). If we never saw "ready"
		// the supervisor exited before becoming healthy.
		close(processExited)
	}()

	go func() {
		waitErr := cmd.Wait()
		// If we never sent a ready signal, surface the wait error.
		select {
		case ready <- waitErr:
		default:
		}
	}()

	select {
	case err := <-ready:
		if err != nil {
			fmt.Fprintf(stderr, "mem up: mymemoryd exited before READY: %v\n", err)
			return 1
		}
		return 0
	case <-processExited:
		<-ready // drain so the Wait goroutine exits cleanly
		return 1
	case <-time.After(MymemorydReadyTimeout):
		// Supervisor didn't print ready in time — fail-closed.
		_ = cmd.Process.Kill()
		<-ready
		fmt.Fprintf(stderr, "mem up: mymemoryd did not become ready within %s\n", MymemorydReadyTimeout)
		return 1
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		<-ready
		return 1
	}
}

// resolveMymemorydPath finds the mymemoryd binary using the
// following precedence:
//  1. Explicit argument (callers like tests inject here).
//  2. $MYMEMORYD_BIN env var.
//  3. Same directory as the running `mem` binary.
//  4. PATH lookup.
func resolveMymemorydPath() (string, error) {
	if env := os.Getenv("MYMEMORYD_BIN"); env != "" {
		if _, err := os.Stat(env); err == nil {
			return env, nil
		} else {
			return "", fmt.Errorf("MYMEMORYD_BIN=%s: %w", env, err)
		}
	}
	exe, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "mymemoryd"+executableSuffix())
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	// PATH fallback.
	if path, err := exec.LookPath("mymemoryd"); err == nil {
		return path, nil
	}
	return "", errors.New("mymemoryd binary not found (set MYMEMORYD_BIN or add to PATH)")
}

// executableSuffix returns ".exe" on Windows, "" elsewhere.
func executableSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}
