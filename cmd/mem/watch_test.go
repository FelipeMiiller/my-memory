package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/config"
	"github.com/FelipeMiiller/my-memory/internal/watcher"
)

func TestWatchLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.DefaultConfig()

	// Arquivo inicial
	initialFile := filepath.Join(tmpDir, "hello.md")
	_ = os.WriteFile(initialFile, []byte("# Hello World"), 0644)

	w := watcher.NewWatcher(tmpDir, &cfg, 50*time.Millisecond, 50*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	events := w.Start(ctx)

	// Garante que o watcher está ativo
	time.Sleep(100 * time.Millisecond)

	// Adiciona novo arquivo
	newFile := filepath.Join(tmpDir, "world.md")
	_ = os.WriteFile(newFile, []byte("# Another Note"), 0644)

	select {
	case ev := <-events:
		if ev.Type != watcher.EventCreate {
			t.Errorf("esperava EventCreate, obteve %s", ev.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout aguardando evento do watcher")
	}

	// Cancela e para
	cancel()
	w.Stop()
}
