package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/config"
	"github.com/FelipeMiiller/my-memory/internal/federation"
)

func TestRunSetupCommand(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(config.GlobalConfigDirEnv, tmpDir)

	centralDir := filepath.Join(tmpDir, "my-central-vault")
	var stdout bytes.Buffer
	stdin := strings.NewReader("")

	args := []string{
		"--central", centralDir,
		"--engine", "postgres",
		"--postgres-url", "postgres://user:pass@localhost:5432/my_memory?sslmode=disable",
		"--mcp-port", "8099",
		"--yes",
	}

	if err := runSetupCommand(args, stdin, &stdout); err != nil {
		t.Fatalf("erro ao executar runSetupCommand: %v", err)
	}

	// 1. Valida config global salva
	gcfg, err := config.LoadGlobalConfig()
	if err != nil {
		t.Fatalf("erro ao carregar global config: %v", err)
	}
	if gcfg.CentralVault.Path != centralDir {
		t.Errorf("esperava central_vault %s, obteve %s", centralDir, gcfg.CentralVault.Path)
	}
	if gcfg.Storage.Engine != "postgres" {
		t.Errorf("esperava engine postgres, obteve %s", gcfg.Storage.Engine)
	}
	if gcfg.MCP.Port != 8099 {
		t.Errorf("esperava mcp port 8099, obteve %d", gcfg.MCP.Port)
	}

	// 2. Valida auto-bootstrap acionado pelo setup
	if !federation.IsCentralVaultInitialized(centralDir) {
		t.Errorf("esperava cofre central auto-inicializado após mem setup")
	}
}

func TestRunCentralCommand_StatusAndBootstrap(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(config.GlobalConfigDirEnv, tmpDir)

	centralDir := filepath.Join(tmpDir, "cloud-vault")

	// 1. Bootstrap explícito
	var bootOut bytes.Buffer
	if err := runCentralCommand([]string{"bootstrap", centralDir}, &bootOut); err != nil {
		t.Fatalf("erro no mem central bootstrap: %v", err)
	}
	if !federation.IsCentralVaultInitialized(centralDir) {
		t.Errorf("esperava cofre central inicializado")
	}

	// Salva na global para que 'mem central status' o localize
	_ = config.SaveGlobalConfig(&config.GlobalConfig{
		Version: 1,
		CentralVault: config.CentralVaultConfig{
			Path: centralDir,
		},
	})

	// 2. Status
	var statusOut bytes.Buffer
	if err := runCentralCommand([]string{"status"}, &statusOut); err != nil {
		t.Fatalf("erro no mem central status: %v", err)
	}
	output := statusOut.String()
	if !strings.Contains(output, "Inicializado e Conectado") {
		t.Errorf("status inesperado: %s", output)
	}
}

func TestRunReposCommand(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(config.GlobalConfigDirEnv, tmpDir)

	// Registra repositório
	repoDir := filepath.Join(tmpDir, "local-repo")
	_ = os.MkdirAll(filepath.Join(repoDir, ".memory"), 0755)

	_ = config.RegisterRepositoryInGlobalConfig(config.RepositoryCatalogEntry{
		ID:   "repo_abc123def456",
		Path: repoDir,
		Name: "test/local-repo",
	})

	var out bytes.Buffer
	if err := runReposCommand(nil, &out); err != nil {
		t.Fatalf("erro ao executar mem repos: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "repo_abc123def456") || !strings.Contains(output, "test/local-repo") {
		t.Errorf("saída de mem repos não contém repositório esperado: %s", output)
	}
}
