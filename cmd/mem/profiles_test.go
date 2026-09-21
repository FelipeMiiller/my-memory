package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCliProfiles_ListAndUse is spec-mandated (tasks.md T8 done-when
// #11). Writes two profiles, runs `mem profiles list` and asserts
// the table; then runs `mem profiles use voice` and asserts that
// .memory/config.yaml gets `active_profile: voice`.
func TestCliProfiles_ListAndUse(t *testing.T) {
	memDir := t.TempDir()

	// Write 2 profiles so `list` has something to enumerate.
	if err := os.MkdirAll(filepath.Join(memDir, "profiles"), 0o755); err != nil {
		t.Fatalf("mkdir profiles: %v", err)
	}
	writeProfileFile(t, memDir, "default.yaml", `schema_version: 1
workers:
  - name: embedder
    command: .memory/workers/embedder
  - name: events
    command: .memory/workers/events
`)
	writeProfileFile(t, memDir, "voice.yaml", `schema_version: 1
extends: default
workers:
  - name: tts
    command: .memory/workers/tts
`)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// list
	var listOut, listErr bytes.Buffer
	if code := runProfilesCommand(ctx, []string{"list", "--memory-dir", memDir}, memDir, &listOut, &listErr); code != 0 {
		t.Fatalf("profiles list: code=%d stderr=%s", code, listErr.String())
	}
	out := listOut.String()
	if !strings.Contains(out, "default") {
		t.Errorf("expected 'default' in list output: %s", out)
	}
	if !strings.Contains(out, "voice") {
		t.Errorf("expected 'voice' in list output: %s", out)
	}
	// voice extends default → effective worker count = 3
	// (embedder + events from parent + tts appended).
	if !strings.Contains(out, "voice") || !strings.Contains(out, "3") {
		t.Errorf("expected voice worker count '3' in list output: %s", out)
	}

	// use voice
	var useOut, useErr bytes.Buffer
	if code := runProfilesCommand(ctx,
		[]string{"use", "voice", "--memory-dir", memDir},
		memDir, &useOut, &useErr); code != 0 {
		t.Fatalf("profiles use: code=%d stderr=%s", code, useErr.String())
	}
	if !strings.Contains(useOut.String(), `voice`) {
		t.Errorf("expected 'voice' in use output: %s", useOut.String())
	}

	// Config file should now have active_profile: voice.
	cfgPath := filepath.Join(memDir, "config.yaml")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(data), "active_profile: voice") {
		t.Errorf("expected 'active_profile: voice' in config, got: %s", data)
	}
}

// TestCliProfiles_ListEmpty confirms the no-profiles branch
// prints a friendly message and exits 0.
func TestCliProfiles_ListEmpty(t *testing.T) {
	memDir := t.TempDir()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var out, errOut bytes.Buffer
	code := runProfilesCommand(ctx, []string{"list", "--memory-dir", memDir}, memDir, &out, &errOut)
	if code != 0 {
		t.Fatalf("expected 0, got %d (stderr=%s)", code, errOut.String())
	}
	if !strings.Contains(out.String(), "No profiles found") {
		t.Errorf("expected 'No profiles found', got: %s", out.String())
	}
}

// TestCliProfiles_UseMissingProfile asserts the validation gate
// (done-when #7): pointing `use` at a non-existent profile must
// fail without touching config.yaml.
func TestCliProfiles_UseMissingProfile(t *testing.T) {
	memDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(memDir, "profiles"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var out, errOut bytes.Buffer
	code := runProfilesCommand(ctx,
		[]string{"use", "ghost", "--memory-dir", memDir},
		memDir, &out, &errOut)
	if code == 0 {
		t.Fatalf("expected non-zero exit on missing profile, got 0")
	}
	cfgPath := filepath.Join(memDir, "config.yaml")
	if _, err := os.Stat(cfgPath); err == nil {
		t.Fatalf("config.yaml should not have been created on failure")
	}
}

// TestCliProfiles_UsePreservesOtherKeys confirms the line-merge
// in writeActiveProfile: existing config keys (repo, db) must
// survive the active_profile update.
func TestCliProfiles_UsePreservesOtherKeys(t *testing.T) {
	memDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(memDir, "profiles"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeProfileFile(t, memDir, "default.yaml", `schema_version: 1
workers:
  - name: logger
    command: .memory/workers/logger
`)
	cfgPath := filepath.Join(memDir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("repo: my-vault\ndb: memory.db\n"), 0o644); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if code := runProfilesCommand(ctx,
		[]string{"use", "default", "--memory-dir", memDir},
		memDir, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("profiles use failed")
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, "repo: my-vault") {
		t.Errorf("repo: my-vault should be preserved, got: %s", body)
	}
	if !strings.Contains(body, "db: memory.db") {
		t.Errorf("db: memory.db should be preserved, got: %s", body)
	}
	if !strings.Contains(body, "active_profile: default") {
		t.Errorf("active_profile: default should be added, got: %s", body)
	}
}

// writeProfileFile is a helper that joins memDir/<name> and
// writes the YAML body. Used by the profile-list fixture so the
// supervisor.LoadProfile parser exercises real YAML.
func writeProfileFile(t *testing.T, memDir, name, body string) {
	t.Helper()
	path := filepath.Join(memDir, "profiles", name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
