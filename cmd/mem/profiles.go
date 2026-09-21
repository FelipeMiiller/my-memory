package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/supervisor"
)

// runProfilesCommand dispatches `mem profiles list` and
// `mem profiles use <name>` (tasks.md T8 done-when #6/#7).
//
// Usage:
//
//	mem profiles list [--memory-dir <dir>]
//	mem profiles use <name> [--memory-dir <dir>]
//
// `list` reads every *.yaml under <memory-dir>/profiles/ and
// prints the profile name plus the effective worker count (after
// inheritance resolution, when a parent is reachable).
//
// `use` writes the chosen profile name to .memory/config.yaml
// under the `active_profile` key, so subsequent `mem up` calls
// pick it up without an explicit --profile flag.
func runProfilesCommand(ctx context.Context, args []string, memoryDir string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "mem profiles: missing subcommand (list|use)")
		return 2
	}
	switch args[0] {
	case "list":
		return runProfilesList(ctx, args[1:], memoryDir, stdout, stderr)
	case "use":
		return runProfilesUse(ctx, args[1:], memoryDir, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "mem profiles: unknown subcommand %q\n", args[0])
		return 2
	}
}

func runProfilesList(ctx context.Context, args []string, memoryDir string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("profiles-list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	memoryDirFlag := fs.String("memory-dir", memoryDir, "vault root (where profiles/ lives)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *memoryDirFlag == "" {
		*memoryDirFlag = ".memory"
	}

	profilesDir := filepath.Join(*memoryDirFlag, "profiles")
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintln(stdout, "No profiles found")
			return 0
		}
		fmt.Fprintf(stderr, "mem profiles list: read %s: %v\n", profilesDir, err)
		return 1
	}

	type profileRow struct {
		name        string
		workerCount int
		err         string
	}
	var rows []profileRow
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".yaml")
		path := filepath.Join(profilesDir, e.Name())
		p, err := supervisor.LoadProfile(path)
		if err != nil {
			rows = append(rows, profileRow{name: name, err: err.Error()})
			continue
		}
		resolved, err := supervisor.ResolveProfile(p, profilesDir)
		if err != nil {
			rows = append(rows, profileRow{name: name, err: err.Error()})
			continue
		}
		rows = append(rows, profileRow{name: name, workerCount: len(resolved.Workers)})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].name < rows[j].name })

	fmt.Fprintf(stdout, "%-32s %s\n", "PROFILE", "WORKERS")
	for _, r := range rows {
		if r.err != "" {
			fmt.Fprintf(stdout, "%-32s ERROR: %s\n", r.name, r.err)
			continue
		}
		fmt.Fprintf(stdout, "%-32s %d\n", r.name, r.workerCount)
	}
	return 0
}

func runProfilesUse(ctx context.Context, args []string, memoryDir string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("profiles-use", flag.ContinueOnError)
	fs.SetOutput(stderr)
	memoryDirFlag := fs.String("memory-dir", memoryDir, "vault root")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *memoryDirFlag == "" {
		*memoryDirFlag = ".memory"
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(stderr, "mem profiles use: missing <name>")
		return 2
	}
	name := fs.Arg(0)

	// Validate that the profile exists before writing — surface
	// the error to the operator instead of corrupting config.
	profilePath := filepath.Join(*memoryDirFlag, "profiles", name+".yaml")
	if _, err := os.Stat(profilePath); err != nil {
		fmt.Fprintf(stderr, "mem profiles use: profile %q not found at %s\n", name, profilePath)
		return 1
	}

	configPath := filepath.Join(*memoryDirFlag, "config.yaml")
	if err := writeActiveProfile(configPath, name); err != nil {
		fmt.Fprintf(stderr, "mem profiles use: write %s: %v\n", configPath, err)
		return 1
	}
	fmt.Fprintf(stdout, "Active profile set to %q\n", name)
	return 0
}

// writeActiveProfile persists name under the active_profile key
// in the YAML config at path. The file may already exist; we
// preserve other keys and only replace the active_profile line.
// For new files we emit a minimal config with just the active
// profile set.
func writeActiveProfile(path, name string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	// Read existing content if any so we preserve unrelated keys.
	body := []byte("active_profile: " + name + "\n")
	if existing, err := os.ReadFile(path); err == nil {
		body = mergeActiveProfile(existing, name)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// mergeActiveProfile replaces any existing `active_profile:` line in
// body with the new value, keeping all other lines intact. Naive
// line scan — sufficient for the simple YAML keys we touch
// today (active_profile, repo, db) but not a full YAML parser.
func mergeActiveProfile(body []byte, name string) []byte {
	lines := strings.Split(string(body), "\n")
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "active_profile:") {
			lines[i] = "active_profile: " + name
			found = true
			break
		}
	}
	if !found {
		// Append — preserve any trailing newline so the file stays
		// well-formed.
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, "")
		}
		lines = append(lines, "active_profile: "+name)
	}
	return []byte(strings.Join(lines, "\n"))
}
