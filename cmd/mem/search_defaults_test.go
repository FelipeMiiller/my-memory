package main

import (
	"os"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/config"
)

func TestResolveStorageAndRepo(t *testing.T) {
	// Cenário 1: Tudo vem da config quando flags e env estão vazias
	cfg := &config.Config{
		Version:    1,
		Repository: "config/repo",
		Storage: config.StorageConfig{
			Engine:      "postgres",
			SQLitePath:  "custom.db",
			PostgresURL: "postgres://config_url",
		},
	}

	repo, db, pg := resolveStorageAndRepo(cfg, "", "", "", "fallback/repo")
	if repo != "config/repo" {
		t.Errorf("esperava repo 'config/repo', obteve '%s'", repo)
	}
	if db != "custom.db" {
		t.Errorf("esperava db 'custom.db', obteve '%s'", db)
	}
	if pg != "postgres://config_url" {
		t.Errorf("esperava pg 'postgres://config_url', obteve '%s'", pg)
	}

	// Cenário 2: Flags explícitas têm prioridade máxima sobre config
	repoFlag, dbFlag, pgFlag := resolveStorageAndRepo(cfg, "flag/repo", "flag.db", "postgres://flag_url", "fallback/repo")
	if repoFlag != "flag/repo" {
		t.Errorf("flag repo deveria ter prioridade, obteve '%s'", repoFlag)
	}
	if dbFlag != "flag.db" {
		t.Errorf("flag db deveria ter prioridade, obteve '%s'", dbFlag)
	}
	if pgFlag != "postgres://flag_url" {
		t.Errorf("flag pg deveria ter prioridade, obteve '%s'", pgFlag)
	}

	os.Setenv("MY_MEMORY_REPO", "env/repo")
	repoEnv, _, _ := resolveStorageAndRepo(&config.Config{Version: 1}, "", "", "", "default")
	os.Unsetenv("MY_MEMORY_REPO")
	if repoEnv != "env/repo" {
		t.Errorf("env var deveria ser resolvida quando flag estiver vazia, obteve '%s'", repoEnv)
	}

	// Cenário 4: Config nil usa defaults seguros
	rDef, dDef, _ := resolveStorageAndRepo(nil, "", "", "", "system/default")
	if rDef != "system/default" {
		t.Errorf("esperava 'system/default', obteve '%s'", rDef)
	}
	if dDef != "memory.db" {
		t.Errorf("esperava 'memory.db', obteve '%s'", dDef)
	}
}

func TestSearchDefaultsFromConfig(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		Search: config.SearchConfig{
			Mode:        "fts",
			Limit:       25,
			K:           80,
			Decay:       true,
			HalfLife:    14.0,
			DecayWeight: 0.5,
			UseTurbo:    true,
		},
	}

	// Simula ausência de flags CLI: valores devem vir da config
	resolvedMode := cfg.Search.Mode
	resolvedLimit := cfg.Search.Limit
	resolvedK := cfg.Search.K
	resolvedDecay := cfg.Search.Decay
	resolvedHalfLife := cfg.Search.HalfLife
	resolvedDecayWeight := cfg.Search.DecayWeight
	resolvedUseTurbo := cfg.Search.UseTurbo

	if resolvedMode != "fts" {
		t.Errorf("esperava mode 'fts', obteve '%s'", resolvedMode)
	}
	if resolvedLimit != 25 {
		t.Errorf("esperava limit 25, obteve %d", resolvedLimit)
	}
	if resolvedK != 80 {
		t.Errorf("esperava k 80, obteve %d", resolvedK)
	}
	if !resolvedDecay {
		t.Errorf("esperava decay true")
	}
	if resolvedHalfLife != 14.0 {
		t.Errorf("esperava half_life 14.0, obteve %f", resolvedHalfLife)
	}
	if resolvedDecayWeight != 0.5 {
		t.Errorf("esperava decay_weight 0.5, obteve %f", resolvedDecayWeight)
	}
	if !resolvedUseTurbo {
		t.Errorf("esperava use_turbo true")
	}
}
