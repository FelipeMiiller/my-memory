// Command mymemoryd is the long-lived supervisor that owns the
// worker pool. See ADR-042 and the mymemoryd-supervisor feature spec.
//
// Usage:
//
//	mymemoryd [--profile <name>] [--memory-dir <dir>]
//
// On startup it acquires .memory/supervisor.lock (refuses with exit 1
// if a fresh lock with a live PID already exists; overwrites stale
// locks). On SIGINT/SIGTERM it releases the lock and exits 0. Missing
// or unreadable profile YAML exits 2 (ADR-050 LLM10 fail-closed).
//
// T1 ships the entrypoint + lock + signal handling. Worker lifecycle
// (T2), profile parsing (T4), IPC (T6), and CLI subcommands (T8) are
// scheduled in subsequent tasks.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/FelipeMiiller/my-memory/internal/supervisor"
)

// Exit codes per ADR-050 LLM10 fail-closed semantics + POSIX convention.
const (
	exitOK             = 0
	exitGenericError   = 1
	exitInvalidProfile = 2
)

// DefaultMemoryDir is the relative path under cwd where the
// supervisor expects to find profiles/ and writes supervisor.lock.
const DefaultMemoryDir = ".memory"

func main() {
	code := run(os.Args[1:], os.Stdout, os.Stderr)
	os.Exit(code)
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("mymemoryd", flag.ContinueOnError)
	fs.SetOutput(stderr)
	profile := fs.String("profile", "default", "profile name under <memory-dir>/profiles/")
	memoryDir := fs.String("memory-dir", DefaultMemoryDir, "vault root (where supervisor.lock lives)")
	if err := fs.Parse(args); err != nil {
		return exitGenericError
	}

	// Profile validity gate: the YAML file must exist. YAML parsing is
	// T4's job — for now we just refuse when the file is missing so the
	// entrypoint honors ADR-050 fail-closed (exit 2 on bad input).
	profilePath := filepath.Join(*memoryDir, "profiles", *profile+".yaml")
	if _, err := os.Stat(profilePath); err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(stderr, "mymemoryd: profile %q not found at %s\n", *profile, profilePath)
			return exitInvalidProfile
		}
		fmt.Fprintf(stderr, "mymemoryd: stat profile: %v\n", err)
		return exitGenericError
	}

	lock, err := supervisor.AcquireLock(*memoryDir, *profile)
	if err != nil {
		if errors.Is(err, supervisor.ErrAlreadyRunning) {
			fmt.Fprintf(stderr, "mymemoryd: %v\n", err)
			return exitGenericError
		}
		if errors.Is(err, supervisor.ErrInvalidProfile) {
			fmt.Fprintf(stderr, "mymemoryd: %v\n", err)
			return exitInvalidProfile
		}
		fmt.Fprintf(stderr, "mymemoryd: acquire lock: %v\n", err)
		return exitGenericError
	}

	// Release the lock on every exit path — including panic-induced
	// crashes, since the next start must not see a stale lock.
	defer func() {
		if rerr := supervisor.ReleaseLock(*memoryDir); rerr != nil {
			fmt.Fprintf(stderr, "mymemoryd: release lock: %v\n", rerr)
		}
	}()

	mgr := supervisor.NewManager(*memoryDir, *profile, nil)

	// Wire signals through context so manager.Run() can block on it
	// without owning signal.Notify itself.
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	fmt.Fprintf(stdout, "mymemoryd: ready (pid=%d profile=%s lock=%s)\n",
		lock.PID, lock.Profile, filepath.Join(*memoryDir, supervisor.LockFileName))

	if err := mgr.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintf(stderr, "mymemoryd: run: %v\n", err)
		return exitGenericError
	}
	fmt.Fprintln(stdout, "mymemoryd: shutting down")
	return exitOK
}
