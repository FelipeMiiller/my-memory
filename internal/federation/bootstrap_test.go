package federation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/config"
)

func TestBootstrapCentralVault_VirginDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	vaultPath := filepath.Join(tmpDir, "central-vault")

	// 1. Antes do bootstrap, não está inicializado
	if IsCentralVaultInitialized(vaultPath) {
		t.Fatalf("esperava vault não inicializado antes do bootstrap")
	}

	// 2. Executa bootstrap
	if err := BootstrapCentralVault(vaultPath); err != nil {
		t.Fatalf("erro ao executar BootstrapCentralVault: %v", err)
	}

	// 3. Verifica se agora está inicializado
	if !IsCentralVaultInitialized(vaultPath) {
		t.Fatalf("esperava IsCentralVaultInitialized retornar true após bootstrap")
	}

	// 4. Verifica se as 11 pastas canônicas e seus README.md foram criados
	for _, folder := range StandardFolders {
		dir := filepath.Join(vaultPath, folder)
		stat, err := os.Stat(dir)
		if err != nil || !stat.IsDir() {
			t.Errorf("pasta canônica '%s' não foi criada", folder)
		}

		readme := filepath.Join(dir, "README.md")
		readmeStat, err := os.Stat(readme)
		if err != nil || readmeStat.IsDir() {
			t.Errorf("README da pasta '%s' não foi criado", folder)
		}
	}

	// 5. Verifica os 4 templates canônicos em templates/
	expectedTemplates := []string{"adr.md", "rfc.md", "runbook.md", "spec.md"}
	for _, tmpl := range expectedTemplates {
		tmplPath := filepath.Join(vaultPath, "templates", tmpl)
		stat, err := os.Stat(tmplPath)
		if err != nil || stat.IsDir() {
			t.Errorf("template '%s' não foi criado em templates/", tmpl)
		}
	}

	// 6. Verifica o Mapa de Conteúdo raiz (README.md)
	rootMOC := filepath.Join(vaultPath, "README.md")
	mocData, err := os.ReadFile(rootMOC)
	if err != nil {
		t.Fatalf("erro ao ler README.md raiz do cofre: %v", err)
	}
	mocContent := string(mocData)
	if !strings.Contains(mocContent, "Central Knowledge Vault") {
		t.Errorf("README raiz não contém título esperado")
	}
	for _, folder := range StandardFolders {
		expectedWikilink := folder + "/README"
		if !strings.Contains(mocContent, expectedWikilink) {
			t.Errorf("README raiz não contém wikilink para %s", expectedWikilink)
		}
	}

	// 7. Verifica a subpasta isolada de dados .memory/
	memCfgPath := filepath.Join(vaultPath, ".memory", "config.yaml")
	cfg, err := config.LoadConfig(memCfgPath)
	if err != nil {
		t.Fatalf("erro ao carregar .memory/config.yaml do cofre central: %v", err)
	}
	if cfg.RepoID != config.CentralRepoID {
		t.Errorf("esperava repo_id '%s', obteve '%s'", config.CentralRepoID, cfg.RepoID)
	}
	if cfg.Storage.SQLitePath != "storage/memory.db" {
		t.Errorf("esperava sqlite_path 'storage/memory.db', obteve '%s'", cfg.Storage.SQLitePath)
	}

	// Verifica se a pasta storage/ foi criada
	storageDir := filepath.Join(vaultPath, ".memory", "storage")
	storageStat, err := os.Stat(storageDir)
	if err != nil || !storageStat.IsDir() {
		t.Errorf("pasta .memory/storage não foi criada")
	}

	// Verifica .memory/.gitignore
	gitIgnorePath := filepath.Join(vaultPath, ".memory", ".gitignore")
	if _, err := os.Stat(gitIgnorePath); err != nil {
		t.Errorf(".memory/.gitignore não foi criado")
	}
}

func TestBootstrapCentralVault_Idempotency(t *testing.T) {
	tmpDir := t.TempDir()
	vaultPath := filepath.Join(tmpDir, "central-vault")

	// Primeiro bootstrap
	if err := BootstrapCentralVault(vaultPath); err != nil {
		t.Fatalf("falha no primeiro bootstrap: %v", err)
	}

	// Cria uma nota customizada para verificar se ela é preservada
	customNotePath := filepath.Join(vaultPath, "standards", "custom-standard.md")
	if err := os.WriteFile(customNotePath, []byte("# Custom Standard\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Segundo bootstrap (deve ser no-op)
	if err := BootstrapCentralVault(vaultPath); err != nil {
		t.Fatalf("falha no segundo bootstrap: %v", err)
	}

	// Verifica que a nota customizada ainda existe
	if _, err := os.Stat(customNotePath); err != nil {
		t.Errorf("nota customizada foi perdida no segundo bootstrap")
	}
}

func TestBootstrapCentralVault_EmptyPath(t *testing.T) {
	err := BootstrapCentralVault("")
	if err == nil {
		t.Errorf("esperava erro para caminho vazio")
	}
}
