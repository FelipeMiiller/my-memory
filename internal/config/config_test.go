package config

import (
	"os"
	"path/filepath"
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
