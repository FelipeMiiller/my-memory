package supervisor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeProfileFile is a small wrapper around os.WriteFile that
// joins dir + name and creates intermediate directories. Used by
// inheritance tests that need multiple profiles side-by-side.
func writeProfileFile(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// TestProfile_Inheritance_MergesWorkers covers spec P3 AC 2 +
// tasks.md T5 done-when #1. Voice profile extends default;
// effective workers = parent (embedder, events) + child-only
// additions (tts) + overrides for any matching name.
func TestProfile_Inheritance_MergesWorkers(t *testing.T) {
	profilesDir := filepath.Join(t.TempDir(), "profiles")
	writeProfileFile(t, profilesDir, "default.yaml", `schema_version: 1
workers:
  - name: embedder
    command: .memory/workers/embedder
    required: true
  - name: events
    command: .memory/workers/events
`)
	writeProfileFile(t, profilesDir, "voice.yaml", `schema_version: 1
extends: default
workers:
  - name: tts
    command: .memory/workers/tts
    required: false
  - name: embedder
    command: .memory/workers/embedder-voice
    required: false
`)

	child, err := LoadProfile(filepath.Join(profilesDir, "voice.yaml"))
	if err != nil {
		t.Fatalf("LoadProfile child: %v", err)
	}
	resolved, err := ResolveProfile(child, profilesDir)
	if err != nil {
		t.Fatalf("ResolveProfile: %v", err)
	}

	wantOrder := []string{"embedder", "events", "tts"}
	if len(resolved.Workers) != len(wantOrder) {
		t.Fatalf("workers count = %d, want %d (%v)",
			len(resolved.Workers), len(wantOrder), wantOrder)
	}
	for i, want := range wantOrder {
		if resolved.Workers[i].Name != want {
			t.Fatalf("workers[%d].Name = %q, want %q",
				i, resolved.Workers[i].Name, want)
		}
	}

	// Child override must have replaced the parent's command +
	// flipped required to false. The embedder at index 0 is the
	// parent's entry but with the child's values applied.
	emb := resolved.Workers[0]
	if emb.Command != ".memory/workers/embedder-voice" {
		t.Errorf("embedder.Command = %q, want override", emb.Command)
	}
	if emb.Required {
		t.Error("embedder.Required = true, want false (child override)")
	}

	// tts at index 2 is child-only.
	if resolved.Workers[2].Required {
		t.Error("tts.Required = true, want false (child-only default)")
	}

	// Extends must be cleared so downstream code never sees the
	// inheritance pointer post-merge.
	if resolved.Extends != "" {
		t.Errorf("resolved.Extends = %q, want empty", resolved.Extends)
	}
}

// TestProfile_Inheritance_ConfigMergedPerWorker verifies the
// Config map merge rule (parent keys + child keys, child wins on
// collision). Different worker entries must NOT cross-contaminate.
func TestProfile_Inheritance_ConfigMergedPerWorker(t *testing.T) {
	profilesDir := filepath.Join(t.TempDir(), "profiles")
	writeProfileFile(t, profilesDir, "base.yaml", `schema_version: 1
workers:
  - name: embedder
    command: .memory/workers/embedder
    config:
      model: minilm
      batch_size: 32
  - name: tts
    command: .memory/workers/tts
    config:
      voice: en-US-A
`)
	writeProfileFile(t, profilesDir, "overlay.yaml", `schema_version: 1
extends: base
workers:
  - name: embedder
    command: .memory/workers/embedder
    config:
      model: e5-large
      cache_dir: /var/cache/e5
`)

	child, err := LoadProfile(filepath.Join(profilesDir, "overlay.yaml"))
	if err != nil {
		t.Fatalf("LoadProfile child: %v", err)
	}
	resolved, err := ResolveProfile(child, profilesDir)
	if err != nil {
		t.Fatalf("ResolveProfile: %v", err)
	}

	// embedder: model overridden to e5-large, cache_dir added.
	// batch_size is preserved from the parent because mergeWorker
	// merges Config maps (child wins on conflict, parent keys
	// without a child override are kept). This is the documented
	// behavior of mergeWorker and intentional — config keys
	// inherit individually even though scalar fields don't.
	emb := resolved.Workers[0]
	if emb.Config["model"] != "e5-large" {
		t.Errorf("embedder.Config[model] = %v, want e5-large", emb.Config["model"])
	}
	if emb.Config["cache_dir"] != "/var/cache/e5" {
		t.Errorf("embedder.Config[cache_dir] = %v, want /var/cache/e5",
			emb.Config["cache_dir"])
	}
	if emb.Config["batch_size"] != 32 {
		t.Errorf("embedder.Config[batch_size] = %v, want 32 (inherited from parent)", emb.Config["batch_size"])
	}

	// tts (parent-only): config preserved as-is.
	tts := resolved.Workers[1]
	if tts.Config["voice"] != "en-US-A" {
		t.Errorf("tts.Config[voice] = %v, want en-US-A", tts.Config["voice"])
	}
}

// TestProfile_Inheritance_RecursiveLoad covers grandparent→parent→child.
func TestProfile_Inheritance_RecursiveLoad(t *testing.T) {
	profilesDir := filepath.Join(t.TempDir(), "profiles")
	writeProfileFile(t, profilesDir, "base.yaml", `schema_version: 1
workers:
  - name: logger
    command: .memory/workers/logger
`)
	writeProfileFile(t, profilesDir, "mid.yaml", `schema_version: 1
extends: base
workers:
  - name: events
    command: .memory/workers/events
`)
	writeProfileFile(t, profilesDir, "leaf.yaml", `schema_version: 1
extends: mid
workers:
  - name: tts
    command: .memory/workers/tts
`)

	leaf, err := LoadProfile(filepath.Join(profilesDir, "leaf.yaml"))
	if err != nil {
		t.Fatalf("LoadProfile leaf: %v", err)
	}
	resolved, err := ResolveProfile(leaf, profilesDir)
	if err != nil {
		t.Fatalf("ResolveProfile: %v", err)
	}

	wantOrder := []string{"logger", "events", "tts"}
	if len(resolved.Workers) != len(wantOrder) {
		t.Fatalf("workers count = %d, want %d", len(resolved.Workers), len(wantOrder))
	}
	for i, want := range wantOrder {
		if resolved.Workers[i].Name != want {
			t.Errorf("workers[%d].Name = %q, want %q",
				i, resolved.Workers[i].Name, want)
		}
	}
}

// TestProfile_Inheritance_MissingParentReturnsError covers the
// fail-loud requirement: a child with extends:missing must error
// instead of silently dropping workers.
func TestProfile_Inheritance_MissingParentReturnsError(t *testing.T) {
	profilesDir := filepath.Join(t.TempDir(), "profiles")
	writeProfileFile(t, profilesDir, "orphan.yaml", `schema_version: 1
extends: nonexistent
workers:
  - name: tts
    command: .memory/workers/tts
`)

	child, err := LoadProfile(filepath.Join(profilesDir, "orphan.yaml"))
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	_, err = ResolveProfile(child, profilesDir)
	if err == nil {
		t.Fatal("expected error on missing parent, got nil")
	}
	if !errors.Is(err, ErrInvalidProfile) {
		t.Fatalf("expected ErrInvalidProfile, got %v", err)
	}
}

// TestProfile_Inheritance_DuplicateInChildStillRejected ensures
// the child's own validation runs even when extends is set. A
// child declaring the same worker name twice must fail loud at
// LoadProfile (before merge logic kicks in).
func TestProfile_Inheritance_DuplicateInChildStillRejected(t *testing.T) {
	profilesDir := filepath.Join(t.TempDir(), "profiles")
	writeProfileFile(t, profilesDir, "default.yaml", `schema_version: 1
workers:
  - name: logger
    command: .memory/workers/logger
`)
	writeProfileFile(t, profilesDir, "dup.yaml", `schema_version: 1
extends: default
workers:
  - name: dup
    command: .memory/workers/dup-1
  - name: dup
    command: .memory/workers/dup-2
`)

	// LoadProfile's Validate already rejects duplicate names —
	// the error surfaces before ResolveProfile ever runs.
	_, err := LoadProfile(filepath.Join(profilesDir, "dup.yaml"))
	if err == nil {
		t.Fatal("expected duplicate-name error on LoadProfile, got nil")
	}
	if !errors.Is(err, ErrInvalidProfile) {
		t.Fatalf("expected ErrInvalidProfile, got %v", err)
	}
}

// TestProfile_RequiredWorkerMissing_FailsClosed — spec-mandated
// (tasks.md T5 done-when #5). A required worker whose binary
// doesn't exist must make StartAll return ErrRequiredFailed so
// the entrypoint can map it to exit 2.
func TestProfile_RequiredWorkerMissing_FailsClosed(t *testing.T) {
	db := newTestDB(t)
	mgr := NewManager(t.TempDir(), "default", db)

	missing := filepath.Join(t.TempDir(), "does-not-exist.exe")
	err := mgr.StartAll(context.Background(), []WorkerSpec{
		{Name: "ghost", Command: missing, Required: true},
	})
	if err == nil {
		t.Fatal("expected error on missing required worker, got nil")
	}
	if !errors.Is(err, ErrRequiredFailed) {
		t.Fatalf("expected ErrRequiredFailed, got %v", err)
	}

	// supervisor.required_failed event must have been emitted.
	if !pollUntil(t, 2*_timeSecond, func() bool {
		return countEventTypeByType(t, db, EventSupervisorRequired) >= 1
	}) {
		t.Fatalf("supervisor.required_failed never emitted")
	}
}

// TestProfile_OptionalWorkerMissing_ContinuesWithEvent —
// spec-mandated. An optional worker whose binary doesn't exist
// must NOT abort StartAll; only worker.optional_failed is emitted
// and the loop moves on.
func TestProfile_OptionalWorkerMissing_ContinuesWithEvent(t *testing.T) {
	db := newTestDB(t)
	mgr := NewManager(t.TempDir(), "default", db)

	missing := filepath.Join(t.TempDir(), "does-not-exist.exe")
	err := mgr.StartAll(context.Background(), []WorkerSpec{
		{Name: "extra", Command: missing, Required: false},
		{Name: "second", Command: fakeWorkerPath},
	})
	if err != nil {
		t.Fatalf("StartAll should not error on optional failure, got %v", err)
	}

	if !pollUntil(t, 2*_timeSecond, func() bool {
		return countEventTypeByType(t, db, EventWorkerOptionalFailed) >= 1
	}) {
		t.Fatalf("worker.optional_failed never emitted")
	}
	// Cleanup the real worker so the test doesn't leak a process.
	defer mgr.Stop(context.Background(), "second", 1*_timeSecond)
}

// TestStartAll_RequiredFailureStopsLoop ensures the loop bails as
// soon as a required worker fails — subsequent specs in the same
// slice must NOT be started.
func TestStartAll_RequiredFailureStopsLoop(t *testing.T) {
	db := newTestDB(t)
	mgr := NewManager(t.TempDir(), "default", db)

	missing := filepath.Join(t.TempDir(), "does-not-exist.exe")
	err := mgr.StartAll(context.Background(), []WorkerSpec{
		{Name: "first", Command: fakeWorkerPath},
		{Name: "ghost", Command: missing, Required: true},
		{Name: "third", Command: fakeWorkerPath},
	})
	if !errors.Is(err, ErrRequiredFailed) {
		t.Fatalf("expected ErrRequiredFailed, got %v", err)
	}

	// Wait briefly and check: "first" was started but "third" was NOT.
	// ActiveWorkers is the cheap read; "third" never appears.
	if !pollUntil(t, 2*_timeSecond, func() bool {
		active := mgr.ActiveWorkers()
		for _, id := range active {
			if id == "third" {
				return false
			}
		}
		return true
	}) {
		t.Fatal("third worker started despite required failure earlier in the loop")
	}

	// Cleanup the first worker.
	defer mgr.Stop(context.Background(), "first", 1*_timeSecond)
}

// time aliases for readability — keeps tests on single-line time
// literals without importing time in each test.
const _timeSecond = time.Second
