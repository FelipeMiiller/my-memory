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

func TestWatchWithVaultConfigScoping(t *testing.T) {
	vaultDir := t.TempDir()
	memDir := filepath.Join(vaultDir, ".memory")
	if err := os.MkdirAll(memDir, 0755); err != nil {
		t.Fatalf("erro ao criar pasta .memory: %v", err)
	}

	cfgContent := `version: 1
repository: "test/scoping-vault"
vault_name: "Scoped Vault"
storage:
  engine: "sqlite"
  sqlite_path: "scoped.db"
watcher:
  debounce_ms: 150
  interval_ms: 60
`
	cfgFile := filepath.Join(memDir, "config.yaml")
	if err := os.WriteFile(cfgFile, []byte(cfgContent), 0644); err != nil {
		t.Fatalf("erro ao criar config.yaml: %v", err)
	}

	// 1. Auto-scoping a partir de um subdiretório
	subDir := filepath.Join(vaultDir, "nested", "folder")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("erro ao criar subdir: %v", err)
	}

	foundCfgPath, err := config.FindConfigFile(subDir)
	if err != nil {
		t.Fatalf("falha no auto-scoping de config: %v", err)
	}
	if foundCfgPath != cfgFile {
		t.Fatalf("esperava %s, obteve %s", cfgFile, foundCfgPath)
	}

	loadedCfg, err := config.LoadConfig(foundCfgPath)
	if err != nil {
		t.Fatalf("falha ao carregar config encontrada: %v", err)
	}

	if loadedCfg.Watcher.DebounceMs != 150 {
		t.Errorf("esperava debounce 150, obteve %d", loadedCfg.Watcher.DebounceMs)
	}
	if loadedCfg.Watcher.IntervalMs != 60 {
		t.Errorf("esperava interval 60, obteve %d", loadedCfg.Watcher.IntervalMs)
	}

	// 2. Criação de watcher com a configuração carregada e verificação de exclusão padrão
	debounce := time.Duration(loadedCfg.Watcher.DebounceMs) * time.Millisecond
	interval := time.Duration(loadedCfg.Watcher.IntervalMs) * time.Millisecond

	w := watcher.NewWatcher(vaultDir, loadedCfg, interval, debounce)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := w.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// Arquivo dentro de pasta excluída (.obsidian) não deve gerar evento
	ignoredDir := filepath.Join(vaultDir, ".obsidian")
	_ = os.MkdirAll(ignoredDir, 0755)
	_ = os.WriteFile(filepath.Join(ignoredDir, "app.json"), []byte("{}"), 0644)

	select {
	case ev := <-events:
		t.Fatalf("não deveria emitir evento para arquivo ignorado: %s", ev.Path)
	case <-time.After(200 * time.Millisecond):
		// Sucesso: arquivo ignorado
	}

	// Arquivo markdown válido deve emitir evento
	validFile := filepath.Join(vaultDir, "Note.md")
	_ = os.WriteFile(validFile, []byte("# Valid note"), 0644)

	select {
	case ev := <-events:
		if ev.RelPath != "Note.md" {
			t.Errorf("esperava evento para Note.md, obteve %s", ev.RelPath)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout aguardando evento para Note.md")
	}

	w.Stop()
}
