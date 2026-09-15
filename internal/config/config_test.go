package config

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Version != 1 {
		t.Errorf("esperava version 1, obteve %d", cfg.Version)
	}
	if cfg.Storage.Engine != "sqlite" {
		t.Errorf("esperava storage engine 'sqlite', obteve '%s'", cfg.Storage.Engine)
	}
	if cfg.Storage.SQLitePath != "memory.db" {
		t.Errorf("esperava sqlite path 'memory.db', obteve '%s'", cfg.Storage.SQLitePath)
	}
	if cfg.Embedding.Model != "nomic-embed-text" {
		t.Errorf("esperava model 'nomic-embed-text', obteve '%s'", cfg.Embedding.Model)
	}
	if cfg.Search.Mode != "hybrid" {
		t.Errorf("esperava search mode 'hybrid', obteve '%s'", cfg.Search.Mode)
	}
	if cfg.Search.Limit != 5 {
		t.Errorf("esperava limit 5, obteve %d", cfg.Search.Limit)
	}
	if cfg.Search.HalfLife != 30.0 {
		t.Errorf("esperava half_life 30.0, obteve %f", cfg.Search.HalfLife)
	}
	if cfg.Watcher.DebounceMs != 500 {
		t.Errorf("esperava debounce_ms 500, obteve %d", cfg.Watcher.DebounceMs)
	}
	if cfg.Watcher.IntervalMs != 1000 {
		t.Errorf("esperava interval_ms 1000, obteve %d", cfg.Watcher.IntervalMs)
	}
	if len(cfg.Exclude) == 0 {
		t.Errorf("esperava excludes padrão, obteve lista vazia")
	}
}

func TestFindConfigFile(t *testing.T) {
	tmpDir := t.TempDir()

	memDir := filepath.Join(tmpDir, ".memory")
	if err := os.MkdirAll(memDir, 0755); err != nil {
		t.Fatalf("erro ao criar dir .memory: %v", err)
	}
	cfgPath := filepath.Join(memDir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("version: 1\nrepository: 'test/repo'\n"), 0644); err != nil {
		t.Fatalf("erro ao gravar config.yaml: %v", err)
	}

	nestedDir := filepath.Join(tmpDir, "subdir", "nested")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("erro ao criar dir aninhado: %v", err)
	}

	// 1. Busca a partir de nestedDir deve subir e encontrar .memory/config.yaml
	found, err := FindConfigFile(nestedDir)
	if err != nil {
		t.Fatalf("esperava encontrar config em %s, erro: %v", cfgPath, err)
	}
	if found != cfgPath {
		t.Errorf("esperava caminho %s, obteve %s", cfgPath, found)
	}

	// 2. Diretório sem config nem .git
	otherTmp := t.TempDir()
	_, err = FindConfigFile(otherTmp)
	if err == nil {
		t.Errorf("esperava erro os.ErrNotExist para diretório vazio sem config")
	}

	// 3. Diretório com .git interrompe busca
	gitRepoDir := filepath.Join(otherTmp, "repo")
	gitDir := filepath.Join(gitRepoDir, ".git")
	subSub := filepath.Join(gitRepoDir, "a", "b")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(subSub, 0755); err != nil {
		t.Fatal(err)
	}
	_, err = FindConfigFile(subSub)
	if err == nil {
		t.Errorf("esperava interrupção de busca pela raiz do git sem achar config")
	}
}

func TestLoadAndSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, ".memory", "config.yaml")

	cfg := DefaultConfig()
	cfg.Repository = "acme/docs"
	cfg.VaultName = "Acme Knowledge Vault"
	cfg.Search.Mode = "fts"
	cfg.Search.Limit = 10
	cfg.Search.Decay = true
	cfg.Search.HalfLife = 14.5
	cfg.Watcher.DebounceMs = 250
	cfg.Watcher.IntervalMs = 800

	if err := SaveConfig(cfgPath, &cfg); err != nil {
		t.Fatalf("erro ao salvar config: %v", err)
	}

	loaded, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("erro ao carregar config: %v", err)
	}

	if loaded.Repository != "acme/docs" {
		t.Errorf("esperava repo 'acme/docs', obteve '%s'", loaded.Repository)
	}
	if loaded.VaultName != "Acme Knowledge Vault" {
		t.Errorf("esperava vault_name 'Acme Knowledge Vault', obteve '%s'", loaded.VaultName)
	}
	if loaded.Search.Mode != "fts" {
		t.Errorf("esperava mode 'fts', obteve '%s'", loaded.Search.Mode)
	}
	if loaded.Search.Limit != 10 {
		t.Errorf("esperava limit 10, obteve %d", loaded.Search.Limit)
	}
	if !loaded.Search.Decay {
		t.Errorf("esperava decay true, obteve false")
	}
	if loaded.Search.HalfLife != 14.5 {
		t.Errorf("esperava half_life 14.5, obteve %f", loaded.Search.HalfLife)
	}
	if loaded.Watcher.DebounceMs != 250 {
		t.Errorf("esperava debounce_ms 250, obteve %d", loaded.Watcher.DebounceMs)
	}
	if loaded.Watcher.IntervalMs != 800 {
		t.Errorf("esperava interval_ms 800, obteve %d", loaded.Watcher.IntervalMs)
	}
	if loaded.Storage.Engine != "sqlite" {
		t.Errorf("esperava storage engine 'sqlite', obteve '%s'", loaded.Storage.Engine)
	}
	if loaded.Embedding.Dimension != 768 {
		t.Errorf("esperava dimension 768, obteve %d", loaded.Embedding.Dimension)
	}
}

func TestLoadConfigJSON(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, ".mem.json")

	content := `{
		"version": 1,
		"repository": "json/repo",
		"storage": {
			"engine": "postgres",
			"postgres_url": "postgres://localhost/test"
		}
	}`

	if err := os.WriteFile(jsonPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadConfig(jsonPath)
	if err != nil {
		t.Fatalf("erro ao ler config json: %v", err)
	}

	if loaded.Repository != "json/repo" {
		t.Errorf("esperava repo 'json/repo', obteve '%s'", loaded.Repository)
	}
	if loaded.Storage.Engine != "postgres" {
		t.Errorf("esperava engine 'postgres', obteve '%s'", loaded.Storage.Engine)
	}
	if loaded.Storage.PostgresURL != "postgres://localhost/test" {
		t.Errorf("esperava postgres_url, obteve '%s'", loaded.Storage.PostgresURL)
	}
	if loaded.Search.Limit != 5 {
		t.Errorf("esperava limit 5, obteve %d", loaded.Search.Limit)
	}
}

func TestEditorConfig_DefaultsAndCustom(t *testing.T) {
	// 1. Defaults
	def := DefaultConfig()
	if def.Editor.DefaultApp != "obsidian" {
		t.Errorf("esperava DefaultApp 'obsidian', obteve '%s'", def.Editor.DefaultApp)
	}
	if def.ResolveDefaultApp() != "obsidian" {
		t.Errorf("esperava ResolveDefaultApp 'obsidian', obteve '%s'", def.ResolveDefaultApp())
	}
	// Sem ObsidianVault e sem VaultName -> usa repoRoot
	if def.ResolveObsidianVault("/path/to/my-repo") != "my-repo" {
		t.Errorf("esperava 'my-repo', obteve '%s'", def.ResolveObsidianVault("/path/to/my-repo"))
	}

	// 2. Com VaultName
	def.VaultName = "Central Vault"
	if def.ResolveObsidianVault("/path/to/my-repo") != "Central Vault" {
		t.Errorf("esperava 'Central Vault', obteve '%s'", def.ResolveObsidianVault("/path/to/my-repo"))
	}

	// 3. Com ObsidianVault explícito (tem precedência máxima)
	def.Editor.ObsidianVault = "Custom Obsidian"
	if def.ResolveObsidianVault("/path/to/my-repo") != "Custom Obsidian" {
		t.Errorf("esperava 'Custom Obsidian', obteve '%s'", def.ResolveObsidianVault("/path/to/my-repo"))
	}

	// 4. Custom DefaultApp
	def.Editor.DefaultApp = "vscode"
	if def.ResolveDefaultApp() != "vscode" {
		t.Errorf("esperava 'vscode', obteve '%s'", def.ResolveDefaultApp())
	}
}

func TestGenerateRepoID(t *testing.T) {
	id1 := GenerateRepoID()
	id2 := GenerateRepoID()

	pattern := `^repo_[a-f0-9]{12}$`
	matched, err := regexp.MatchString(pattern, id1)
	if err != nil || !matched {
		t.Errorf("repo_id %s não corresponde ao padrão esperado %s", id1, pattern)
	}

	if id1 == id2 {
		t.Errorf("GenerateRepoID gerou IDs duplicados: %s vs %s", id1, id2)
	}
}

func TestEnsureRepoID(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.RepoID != "" {
		t.Errorf("esperava repo_id vazio por padrão, obteve %s", cfg.RepoID)
	}

	// 1. Gera quando vazio
	generated := EnsureRepoID(&cfg)
	if !generated {
		t.Errorf("esperava EnsureRepoID retornar true ao gerar novo ID")
	}
	if !strings.HasPrefix(cfg.RepoID, "repo_") || len(cfg.RepoID) != 17 {
		t.Errorf("repo_id gerado inválido: %s", cfg.RepoID)
	}

	// 2. Preserva quando já existente
	existingID := cfg.RepoID
	generatedAgain := EnsureRepoID(&cfg)
	if generatedAgain {
		t.Errorf("esperava EnsureRepoID retornar false quando ID já existe")
	}
	if cfg.RepoID != existingID {
		t.Errorf("esperava preservar repo_id %s, mas foi alterado para %s", existingID, cfg.RepoID)
	}
}

func TestResolveRepoDatabaseName(t *testing.T) {
	tests := []struct {
		repoID   string
		expected string
	}{
		{"repo_1234567890ab", "my_memory_repo_1234567890ab"},
		{"repo_central", "my_memory_central"},
		{"", "my_memory_central"},
	}

	for _, tc := range tests {
		got := ResolveRepoDatabaseName(tc.repoID)
		if got != tc.expected {
			t.Errorf("ResolveRepoDatabaseName(%q) = %q; esperava %q", tc.repoID, got, tc.expected)
		}
	}
}

func TestExpandPath(t *testing.T) {
	os.Setenv("TEST_CENTRAL_VAULT", "/data/vault")
	defer os.Unsetenv("TEST_CENTRAL_VAULT")

	expandedEnv := ExpandPath("${TEST_CENTRAL_VAULT}/standards")
	expectedEnv := filepath.Clean("/data/vault/standards")
	if filepath.Clean(expandedEnv) != expectedEnv {
		t.Errorf("esperava %s, obteve %s", expectedEnv, expandedEnv)
	}

	home, err := os.UserHomeDir()
	if err == nil {
		expandedHome := ExpandPath("~/my-vault")
		expectedHome := filepath.Join(home, "my-vault")
		if expandedHome != expectedHome {
			t.Errorf("esperava %s, obteve %s", expectedHome, expandedHome)
		}
	}
}

func TestLoadAndSaveGlobalConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(GlobalConfigDirEnv, tmpDir)

	// 1. Antes de salvar, LoadGlobalConfig deve retornar nil, nil
	initial, err := LoadGlobalConfig()
	if err != nil {
		t.Fatalf("erro inesperado ao carregar config inexistente: %v", err)
	}
	if initial != nil {
		t.Fatalf("esperava nil para config global não existente, obteve: %+v", initial)
	}

	// 2. Salva uma nova configuração global
	toSave := GlobalConfig{
		Version: 1,
		CentralVault: CentralVaultConfig{
			Path:     filepath.Join(tmpDir, "central-vault"),
			ReadOnly: false,
			Include:  []string{"standards/**", "architecture/**"},
		},
		Storage: StorageConfig{
			Engine:      "postgres",
			PostgresURL: "postgres://user:pass@localhost:5432/my_memory?sslmode=disable",
		},
		MCP: MCPConfig{
			Port: 8085,
		},
	}

	if err := SaveGlobalConfig(&toSave); err != nil {
		t.Fatalf("erro ao salvar config global: %v", err)
	}

	// 3. Lê e valida
	loaded, err := LoadGlobalConfig()
	if err != nil {
		t.Fatalf("erro ao carregar config global salva: %v", err)
	}
	if loaded == nil {
		t.Fatalf("esperava config carregada não nula")
	}

	if loaded.CentralVault.Path != filepath.Join(tmpDir, "central-vault") {
		t.Errorf("esperava central_vault.path %s, obteve %s", filepath.Join(tmpDir, "central-vault"), loaded.CentralVault.Path)
	}
	if loaded.Storage.Engine != "postgres" {
		t.Errorf("esperava storage.engine 'postgres', obteve %s", loaded.Storage.Engine)
	}
	if loaded.MCP.Port != 8085 {
		t.Errorf("esperava mcp.port 8085, obteve %d", loaded.MCP.Port)
	}
	if len(loaded.CentralVault.Include) != 2 {
		t.Errorf("esperava 2 includes no central_vault, obteve %d", len(loaded.CentralVault.Include))
	}
}

func TestLoadCascadingConfig(t *testing.T) {
	globalDir := t.TempDir()
	t.Setenv(GlobalConfigDirEnv, globalDir)

	// Salva config global com PostgreSQL e Central Vault
	globalCfg := GlobalConfig{
		Version: 1,
		CentralVault: CentralVaultConfig{
			Path:     filepath.Join(globalDir, "global-vault"),
			ReadOnly: true,
		},
		Storage: StorageConfig{
			Engine:      "postgres",
			PostgresURL: "postgres://global:secret@localhost:5432/my_memory?sslmode=disable",
		},
		MCP: MCPConfig{
			Port: 9090,
		},
	}
	if err := SaveGlobalConfig(&globalCfg); err != nil {
		t.Fatalf("erro ao salvar config global: %v", err)
	}

	// Caso 1: Repositório sem config local (.memory inexistente) -> herda global
	emptyRepoDir := t.TempDir()
	cascaded, _, err := LoadCascadingConfig(emptyRepoDir)
	if err != nil {
		t.Fatalf("erro ao carregar cascading config sem local: %v", err)
	}
	if cascaded.CentralVault.Path != filepath.Join(globalDir, "global-vault") {
		t.Errorf("esperava central vault herdado %s, obteve %s", filepath.Join(globalDir, "global-vault"), cascaded.CentralVault.Path)
	}
	if cascaded.Storage.Engine != "postgres" {
		t.Errorf("esperava storage engine postgres herdado, obteve %s", cascaded.Storage.Engine)
	}
	if cascaded.MCP.Port != 9090 {
		t.Errorf("esperava mcp port 9090 herdado, obteve %d", cascaded.MCP.Port)
	}

	// Caso 2: Repositório com config local definindo override de central_vault e storage sqlite explícito
	localRepoDir := t.TempDir()
	localMemDir := filepath.Join(localRepoDir, ".memory")
	if err := os.MkdirAll(localMemDir, 0755); err != nil {
		t.Fatal(err)
	}
	localYaml := `
version: 1
repo_id: "repo_custom123456"
repository: "custom/repo"
central_vault:
  path: "/custom/local/central"
  read_only: false
storage:
  engine: "sqlite"
  sqlite_path: "custom.db"
mcp:
  port: 7070
`
	if err := os.WriteFile(filepath.Join(localMemDir, "config.yaml"), []byte(localYaml), 0644); err != nil {
		t.Fatal(err)
	}

	cascadedLocal, path, err := LoadCascadingConfig(localRepoDir)
	if err != nil {
		t.Fatalf("erro ao carregar cascading config com local: %v", err)
	}
	if path == "" {
		t.Errorf("esperava path retornado")
	}
	if cascadedLocal.RepoID != "repo_custom123456" {
		t.Errorf("esperava repo_id 'repo_custom123456', obteve %s", cascadedLocal.RepoID)
	}
	if filepath.Clean(cascadedLocal.CentralVault.Path) != filepath.Clean("/custom/local/central") {
		t.Errorf("esperava central_vault sobrescrito '/custom/local/central', obteve %s", cascadedLocal.CentralVault.Path)
	}
	if cascadedLocal.Storage.Engine != "sqlite" || cascadedLocal.Storage.SQLitePath != "custom.db" {
		t.Errorf("esperava override de storage sqlite custom.db, obteve %+v", cascadedLocal.Storage)
	}
	if cascadedLocal.MCP.Port != 7070 {
		t.Errorf("esperava override de mcp.port 7070, obteve %d", cascadedLocal.MCP.Port)
	}
}

func TestRegisterRepositoryInGlobalConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(GlobalConfigDirEnv, tmpDir)

	// 1. Cadastra primeiro repositório
	repo1Path := filepath.Join(tmpDir, "repo1")
	entry1 := RepositoryCatalogEntry{
		ID:   "repo_111122223333",
		Path: repo1Path,
		Name: "acme/api-core",
	}
	if err := RegisterRepositoryInGlobalConfig(entry1); err != nil {
		t.Fatalf("erro ao registrar repo1: %v", err)
	}

	// 2. Cadastra segundo repositório
	repo2Path := filepath.Join(tmpDir, "repo2")
	entry2 := RepositoryCatalogEntry{
		ID:   "repo_444455556666",
		Path: repo2Path,
		Name: "acme/frontend",
	}
	if err := RegisterRepositoryInGlobalConfig(entry2); err != nil {
		t.Fatalf("erro ao registrar repo2: %v", err)
	}

	// 3. Lê config global e valida catálogo
	gcfg, err := LoadGlobalConfig()
	if err != nil {
		t.Fatalf("erro ao carregar config global: %v", err)
	}
	if len(gcfg.Repositories) != 2 {
		t.Fatalf("esperava 2 repositórios no catálogo, obteve %d", len(gcfg.Repositories))
	}

	// 4. Atualiza repo1 (idempotência com mesmo ID)
	entry1Updated := RepositoryCatalogEntry{
		ID:   "repo_111122223333",
		Path: filepath.Join(tmpDir, "repo1-moved"),
		Name: "acme/api-core-renamed",
	}
	if err := RegisterRepositoryInGlobalConfig(entry1Updated); err != nil {
		t.Fatalf("erro ao atualizar repo1: %v", err)
	}

	gcfgUpdated, err := LoadGlobalConfig()
	if err != nil {
		t.Fatalf("erro ao carregar config global atualizada: %v", err)
	}
	if len(gcfgUpdated.Repositories) != 2 {
		t.Fatalf("esperava manter 2 repositórios após atualização, obteve %d", len(gcfgUpdated.Repositories))
	}
	if gcfgUpdated.Repositories[0].Name != "acme/api-core-renamed" {
		t.Errorf("esperava nome atualizado 'acme/api-core-renamed', obteve %s", gcfgUpdated.Repositories[0].Name)
	}

	// 5. Validação de erro em campos vazios
	if err := RegisterRepositoryInGlobalConfig(RepositoryCatalogEntry{Path: "/foo"}); err == nil {
		t.Errorf("esperava erro ao registrar repo sem ID")
	}
	if err := RegisterRepositoryInGlobalConfig(RepositoryCatalogEntry{ID: "repo_123"}); err == nil {
		t.Errorf("esperava erro ao registrar repo sem Path")
	}
}
