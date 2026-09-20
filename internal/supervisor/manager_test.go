package supervisor

import (
	"context"
	"testing"
	"time"
)

func TestManager_RunBlocksUntilContextCancelled(t *testing.T) {
	mgr := NewManager(t.TempDir(), "default", nil)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- mgr.Run(ctx) }()

	// Give Run a moment to start; verify it has not returned.
	select {
	case err := <-done:
		t.Fatalf("Run returned before cancellation: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	cancel()

	select {
	case err := <-done:
		if err != context.Canceled {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Run did not return within 1s after cancel")
	}
}

func TestManager_DoneClosesAfterRunReturns(t *testing.T) {
	mgr := NewManager(t.TempDir(), "default", nil)

	ctx, cancel := context.WithCancel(context.Background())
	runDone := make(chan struct{})
	go func() {
		_ = mgr.Run(ctx)
		close(runDone)
	}()

	select {
	case <-mgr.Done():
		t.Fatal("Done closed before Run returned")
	default:
	}

	cancel()
	<-runDone

	select {
	case <-mgr.Done():
	case <-time.After(1 * time.Second):
		t.Fatal("Done did not close after Run returned")
	}
}

func TestManager_ActiveWorkersEmptyOnT1(t *testing.T) {
	mgr := NewManager(t.TempDir(), "default", nil)
	if got := mgr.ActiveWorkers(); len(got) != 0 {
		t.Fatalf("T1 ActiveWorkers should be empty, got %v", got)
	}
}

func TestManager_ProfileRoundTrip(t *testing.T) {
	mgr := NewManager(t.TempDir(), "voice", nil)
	if got := mgr.Profile(); got != "voice" {
		t.Fatalf("Profile() = %q, want %q", got, "voice")
	}
}

func TestManager_StringIncludesProfileAndDir(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir, "default", nil)
	s := mgr.String()
	if s == "" {
		t.Fatal("String() returned empty")
	}
	// Spot check: both fields should appear in the debug string.
	if !contains(s, "default") {
		t.Fatalf("expected default in String, got %q", s)
	}
	if !contains(s, dir) {
		t.Fatalf("expected memoryDir in String, got %q", s)
	}
}

func TestManager_RegisterSpecAndLookup(t *testing.T) {
	mgr := NewManager(t.TempDir(), "default", nil)
	mgr.RegisterSpec(WorkerSpec{Name: "embedder", Command: ".memory/workers/embedder", Args: []string{"--port", "49156"}})

	spec, ok := mgr.Spec("embedder")
	if !ok {
		t.Fatal("expected embedder to be registered")
	}
	if spec.Command != ".memory/workers/embedder" {
		t.Fatalf("expected Command preserved, got %q", spec.Command)
	}
	if len(spec.Args) != 2 || spec.Args[0] != "--port" || spec.Args[1] != "49156" {
		t.Fatalf("Args not preserved: %v", spec.Args)
	}

	if _, ok := mgr.Spec("missing"); ok {
		t.Fatal("expected missing worker to be absent")
	}
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
