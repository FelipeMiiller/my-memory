package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
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

// writeVaultConfig cria um vault em dir com o repo_id declarado e devolve o
// caminho do arquivo de configuração gravado.
func writeVaultConfig(t *testing.T, dir, repoID string) string {
	t.Helper()
	memDir := filepath.Join(dir, ".memory")
	if err := os.MkdirAll(memDir, 0755); err != nil {
		t.Fatalf("erro ao criar diretório .memory: %v", err)
	}
	cfgPath := filepath.Join(memDir, "config.yaml")
	content := fmt.Sprintf("version: 1\nrepo_id: %q\nrepository: \"test/repo\"\n", repoID)
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("erro ao escrever config do vault: %v", err)
	}
	return cfgPath
}

// listFiles devolve os nomes dos arquivos de um diretório, ordenados.
func listFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("erro ao listar %s: %v", dir, err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names
}

// TestRunReposCommand_DivergenciaRepoID: um id do catálogo que não corresponde
// ao repo_id do vault é erro explícito — nunca uma linha listada como OK
// (ISSUE-028).
func TestRunReposCommand_DivergenciaRepoID(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(config.GlobalConfigDirEnv, tmpDir)

	repoDir := filepath.Join(tmpDir, "local-repo")
	writeVaultConfig(t, repoDir, "repo_disk_actual")

	// O catálogo global registra um id diferente do repo_id do disco.
	_ = config.RegisterRepositoryInGlobalConfig(config.RepositoryCatalogEntry{
		ID:   "repo_stale_000000",
		Path: repoDir,
		Name: "test/local-repo",
	})

	var out bytes.Buffer
	err := runReposCommand(nil, &out)
	if err == nil {
		t.Fatal("esperava erro na divergência de repo_id, mas o comando saiu sem erro")
	}

	msg := err.Error()
	for _, esperado := range []string{
		"repo_stale_000000", // id do catálogo
		"repo_disk_actual",  // repo_id do disco
		repoDir,             // path registrado
		"repositories[].id", // o que fazer
	} {
		if !strings.Contains(msg, esperado) {
			t.Errorf("a mensagem de erro não contém %q: %s", esperado, msg)
		}
	}

	output := out.String()
	if !strings.Contains(output, "❌") {
		t.Errorf("a entrada divergente deve ser marcada com ❌: %s", output)
	}
	if strings.Contains(output, "✅") {
		t.Errorf("a entrada divergente não pode ser listada como OK: %s", output)
	}
	if !strings.Contains(output, "repo_disk_actual") {
		t.Errorf("a saída deve mostrar o repo_id do disco: %s", output)
	}
}

// TestRunReposCommand_RepoIDAlinhado: com o id do catálogo igual ao repo_id do
// vault, o comando não acusa nada e a saída segue o formato de sempre.
func TestRunReposCommand_RepoIDAlinhado(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(config.GlobalConfigDirEnv, tmpDir)

	repoDir := filepath.Join(tmpDir, "local-repo")
	writeVaultConfig(t, repoDir, "repo_aligned_111")

	_ = config.RegisterRepositoryInGlobalConfig(config.RepositoryCatalogEntry{
		ID:   "repo_aligned_111",
		Path: repoDir,
		Name: "test/local-repo",
	})

	var out bytes.Buffer
	if err := runReposCommand(nil, &out); err != nil {
		t.Fatalf("não esperava erro com o repo_id alinhado: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "✅") {
		t.Errorf("a entrada alinhada deve ser marcada com ✅: %s", output)
	}
	if strings.Contains(output, "❌") {
		t.Errorf("a entrada alinhada não pode ser marcada como divergente: %s", output)
	}
	if !strings.Contains(output, "REPO ID") || !strings.Contains(output, "CAMINHO NO DISCO") {
		t.Errorf("a tabela perdeu o formato de sempre: %s", output)
	}
}

// TestRunReposCommand_SemVaultInicializado: entrada cujo path não tem config
// de vault não é divergência — é aviso, listada sem erro.
func TestRunReposCommand_SemVaultInicializado(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(config.GlobalConfigDirEnv, tmpDir)

	t.Run("path sem diretório .memory", func(t *testing.T) {
		repoDir := filepath.Join(tmpDir, "sem-vault")
		if err := os.MkdirAll(repoDir, 0755); err != nil {
			t.Fatalf("erro ao criar diretório: %v", err)
		}
		_ = config.RegisterRepositoryInGlobalConfig(config.RepositoryCatalogEntry{
			ID:   "repo_sem_vault_0",
			Path: repoDir,
			Name: "test/sem-vault",
		})

		var out bytes.Buffer
		if err := runReposCommand(nil, &out); err != nil {
			t.Fatalf("vault não inicializado não pode ser erro: %v", err)
		}
		if !strings.Contains(out.String(), "⚠️") {
			t.Errorf("esperava o aviso de vault não inicializado: %s", out.String())
		}
	})

	t.Run("diretório .memory sem config.yaml", func(t *testing.T) {
		repoDir := filepath.Join(tmpDir, "vault-sem-config")
		if err := os.MkdirAll(filepath.Join(repoDir, ".memory"), 0755); err != nil {
			t.Fatalf("erro ao criar diretório: %v", err)
		}
		_ = config.RegisterRepositoryInGlobalConfig(config.RepositoryCatalogEntry{
			ID:   "repo_vault_sem_cfg",
			Path: repoDir,
			Name: "test/vault-sem-config",
		})

		var out bytes.Buffer
		if err := runReposCommand(nil, &out); err != nil {
			t.Fatalf("vault sem config não pode ser erro: %v", err)
		}
		if !strings.Contains(out.String(), "⚠️") {
			t.Errorf("esperava o aviso de vault não inicializado: %s", out.String())
		}
		if strings.Contains(out.String(), "❌") {
			t.Errorf("vault sem config não é divergência: %s", out.String())
		}
	})
}

// TestRunReposCommand_VaultSemRepoID: um vault que existe mas não declara
// repo_id não tem contra o que comparar — é listado sem erro.
func TestRunReposCommand_VaultSemRepoID(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(config.GlobalConfigDirEnv, tmpDir)

	repoDir := filepath.Join(tmpDir, "vault-sem-id")
	memDir := filepath.Join(repoDir, ".memory")
	if err := os.MkdirAll(memDir, 0755); err != nil {
		t.Fatalf("erro ao criar diretório: %v", err)
	}
	if err := os.WriteFile(filepath.Join(memDir, "config.yaml"), []byte("version: 1\nrepository: \"test/sem-id\"\n"), 0644); err != nil {
		t.Fatalf("erro ao escrever config: %v", err)
	}

	_ = config.RegisterRepositoryInGlobalConfig(config.RepositoryCatalogEntry{
		ID:   "repo_sem_id_00000",
		Path: repoDir,
		Name: "test/sem-id",
	})

	var out bytes.Buffer
	if err := runReposCommand(nil, &out); err != nil {
		t.Fatalf("vault sem repo_id não pode ser erro: %v", err)
	}
	output := out.String()
	if strings.Contains(output, "❌") || strings.Contains(output, "ivergência") {
		t.Errorf("vault sem repo_id não é divergência: %s", output)
	}
}

// TestRunReposCommand_EhReadOnly: a auditoria não escreve no catálogo global
// nem no vault do path registrado (ISSUE-028).
func TestRunReposCommand_EhReadOnly(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(config.GlobalConfigDirEnv, tmpDir)

	repoDir := filepath.Join(tmpDir, "local-repo")
	vaultCfgPath := writeVaultConfig(t, repoDir, "repo_readonly_1")

	_ = config.RegisterRepositoryInGlobalConfig(config.RepositoryCatalogEntry{
		ID:   "repo_readonly_1",
		Path: repoDir,
		Name: "test/local-repo",
	})

	globalPath, err := config.GlobalConfigPath()
	if err != nil {
		t.Fatalf("erro ao obter caminho do config global: %v", err)
	}
	globalBefore, err := os.ReadFile(globalPath)
	if err != nil {
		t.Fatalf("erro ao ler config global: %v", err)
	}
	vaultBefore, err := os.ReadFile(vaultCfgPath)
	if err != nil {
		t.Fatalf("erro ao ler config do vault: %v", err)
	}
	filesBefore := listFiles(t, filepath.Join(repoDir, ".memory"))

	var out bytes.Buffer
	if err := runReposCommand(nil, &out); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	globalAfter, err := os.ReadFile(globalPath)
	if err != nil {
		t.Fatalf("erro ao reler config global: %v", err)
	}
	vaultAfter, err := os.ReadFile(vaultCfgPath)
	if err != nil {
		t.Fatalf("erro ao reler config do vault: %v", err)
	}
	filesAfter := listFiles(t, filepath.Join(repoDir, ".memory"))

	if !bytes.Equal(globalBefore, globalAfter) {
		t.Errorf("a auditoria alterou o catálogo global:\n%s", string(globalAfter))
	}
	if !bytes.Equal(vaultBefore, vaultAfter) {
		t.Errorf("a auditoria alterou o config do vault:\n%s", string(vaultAfter))
	}
	if !reflect.DeepEqual(filesBefore, filesAfter) {
		t.Errorf("a auditoria criou ou removeu arquivos no vault: antes=%v depois=%v", filesBefore, filesAfter)
	}
}
