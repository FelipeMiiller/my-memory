package watcher

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/config"
)

func TestDebouncerConsolidation(t *testing.T) {
	var count int32
	var lastEvent FileEvent

	debouncer := NewDebouncer(100*time.Millisecond, func(ev FileEvent) {
		atomic.AddInt32(&count, 1)
		lastEvent = ev
	})
	defer debouncer.Stop()

	ev := FileEvent{Type: EventModify, Path: "test.md"}

	// Dispara 5 eventos rápidos em menos de 50ms
	for i := 0; i < 5; i++ {
		debouncer.Add(ev)
		time.Sleep(10 * time.Millisecond)
	}

	// Aguarda expirar a janela de 100ms
	time.Sleep(150 * time.Millisecond)

	finalCount := atomic.LoadInt32(&count)
	if finalCount != 1 {
		t.Errorf("esperava 1 evento consolidado pelo debouncer, obteve %d", finalCount)
	}
	if lastEvent.Path != "test.md" {
		t.Errorf("esperava path 'test.md', obteve '%s'", lastEvent.Path)
	}
}

func TestWatcherLifecycleAndEvents(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.DefaultConfig()

	// Arquivo inicial para baseline (não deve disparar evento na inicialização)
	initFile := filepath.Join(tmpDir, "initial.md")
	if err := os.WriteFile(initFile, []byte("# Initial Note\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Cria watcher com polling rápido (50ms) e debounce (50ms) para testes ágeis
	w := NewWatcher(tmpDir, &cfg, 50*time.Millisecond, 50*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := w.Start(ctx)

	// Aguarda estabilização da baseline
	time.Sleep(100 * time.Millisecond)

	// 1. Criar novo arquivo Markdown: deve gerar EventCreate
	newFile := filepath.Join(tmpDir, "notes", "created.md")
	if err := os.MkdirAll(filepath.Dir(newFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newFile, []byte("# Created Note\n"), 0644); err != nil {
		t.Fatal(err)
	}

	select {
	case ev := <-events:
		if ev.Type != EventCreate {
			t.Errorf("esperava EventCreate, obteve %s", ev.Type)
		}
		if ev.RelPath != filepath.Join("notes", "created.md") && filepath.ToSlash(ev.RelPath) != "notes/created.md" {
			t.Errorf("esperava relPath notes/created.md, obteve '%s'", ev.RelPath)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("tempo limite esgotado aguardando EventCreate")
	}

	// 2. Modificar o arquivo: deve gerar EventModify
	time.Sleep(100 * time.Millisecond) // Garante timestamp distinto
	if err := os.WriteFile(newFile, []byte("# Created Note - Modificada\nConteudo adicional"), 0644); err != nil {
		t.Fatal(err)
	}

	select {
	case ev := <-events:
		if ev.Type != EventModify {
			t.Errorf("esperava EventModify, obteve %s", ev.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("tempo limite esgotado aguardando EventModify")
	}

	// 3. Deletar o arquivo: deve gerar EventDelete
	if err := os.Remove(newFile); err != nil {
		t.Fatal(err)
	}

	select {
	case ev := <-events:
		if ev.Type != EventDelete {
			t.Errorf("esperava EventDelete, obteve %s", ev.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("tempo limite esgotado aguardando EventDelete")
	}

	// 4. Criar arquivo em pasta ignorada (.git/config): NÃO deve gerar evento
	ignoredGit := filepath.Join(tmpDir, ".git", "config.md")
	_ = os.MkdirAll(filepath.Dir(ignoredGit), 0755)
	_ = os.WriteFile(ignoredGit, []byte("ignored"), 0644)

	// Arquivo de extensão não incluída (.png)
	ignoredExt := filepath.Join(tmpDir, "image.png")
	_ = os.WriteFile(ignoredExt, []byte("fake png"), 0644)

	select {
	case ev := <-events:
		t.Fatalf("recebeu evento inesperado para arquivo ignorado: %v", ev)
	case <-time.After(300 * time.Millisecond):
		// Sucesso: nenhum evento gerado para arquivos ignorados
	}

	// 5. Parar graciosamente
	w.Stop()
}
