package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/config"
)

func TestRunInit(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Inicializar config pela primeira vez
	err := runInit(tmpDir, "test-owner/test-vault", "custom.db", false)
	if err != nil {
		t.Fatalf("erro ao executar runInit: %v", err)
	}

	cfgPath := filepath.Join(tmpDir, ".memory", "config.yaml")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Fatalf("arquivo de configuração não foi gerado em %s", cfgPath)
	}

	// 2. Carregar com o motor de configuração para validar integridade
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("erro ao carregar configuração gerada por runInit: %v", err)
	}

	if cfg.Repository != "test-owner/test-vault" {
		t.Errorf("esperava repository 'test-owner/test-vault', obteve '%s'", cfg.Repository)
	}
	if cfg.Storage.SQLitePath != "custom.db" {
		t.Errorf("esperava sqlite_path 'custom.db', obteve '%s'", cfg.Storage.SQLitePath)
	}
	if cfg.Search.Mode != "hybrid" {
		t.Errorf("esperava mode 'hybrid', obteve '%s'", cfg.Search.Mode)
	}
	if !cfg.ShouldIndex("notes/daily.md") {
		t.Errorf("deveria indexar notes/daily.md com a config gerada")
	}
	if cfg.ShouldIndex(".git/config") {
		t.Errorf("não deveria indexar .git/config")
	}

	// 3. Tentar executar novamente sem force não deve sobrescrever
	err = runInit(tmpDir, "outra-coisa", "outro.db", false)
	if err != nil {
		t.Fatalf("runInit sem force não deveria retornar erro: %v", err)
	}
	cfgRecheck, _ := config.LoadConfig(cfgPath)
	if cfgRecheck.Repository != "test-owner/test-vault" {
		t.Errorf("configuração não deveria ter sido sobrescrita sem --force")
	}

	// 4. Executar com force=true deve sobrescrever
	err = runInit(tmpDir, "novo-repo", "novo.db", true)
	if err != nil {
		t.Fatalf("runInit com force deveria funcionar: %v", err)
	}
	cfgOverwritten, _ := config.LoadConfig(cfgPath)
	if cfgOverwritten.Repository != "novo-repo" {
		t.Errorf("esperava novo repository 'novo-repo', obteve '%s'", cfgOverwritten.Repository)
	}
	if cfgOverwritten.Storage.SQLitePath != "novo.db" {
		t.Errorf("esperava novo sqlite_path 'novo.db', obteve '%s'", cfgOverwritten.Storage.SQLitePath)
	}
}
