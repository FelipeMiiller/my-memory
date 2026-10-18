package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/config"
)

// TestDotEnvDoesNotReinjectStorageEnvVar é a regressão da ISSUE-016.
//
// O TestMain limpa MY_MEMORY_PG_URL/POSTGRES_URL/DATABASE_URL para isolar o
// storage do pacote de testes. Só que resolveConfig() chama
// config.FindAndLoadDotEnv("."), que sobe do diretório do teste até a raiz do
// repositório, encontra o `.memory/.env` do projeto e chama os.Setenv de volta
// (internal/config/env.go, LoadDotEnvFile). O isolamento do TestMain era
// desfeito no meio da execução, e todo subcomando que chama resolveConfig()
// voltava a rotear para o Postgres real do usuário.
func TestDotEnvDoesNotReinjectStorageEnvVar(t *testing.T) {
	// Reproduz o estado logo após o TestMain: variáveis presentes no ambiente
	// do desenvolvedor, mas limpas para o processo de teste.
	for _, k := range envStorageVars {
		t.Setenv(k, "")
	}

	foundPath, err := config.FindAndLoadDotEnv(".")
	t.Logf("FindAndLoadDotEnv(\".\") = %q (err=%v)", foundPath, err)

	for _, k := range envStorageVars {
		if v := os.Getenv(k); v != "" {
			t.Errorf("vazamento de ambiente: %s foi repopulada por %s (%d bytes). "+
				"resolveStorageAndRepo vai ler essa URL e rotear para o Postgres real do usuário.",
				k, foundPath, len(v))
		}
	}
}

// TestResolveConfigAndStorage_NeverReachUserGlobalState fecha oISSUE-016 pelo
// lado do efeito: mesmo que alguma variável escape, o storage resolvido por um
// comando do CLI não pode apontar para dentro do ~/.memory real do usuário, nem
// para um Postgres herdado do ambiente.
func TestResolveConfigAndStorage_NeverReachUserGlobalState(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Skipf("UserHomeDir indisponível: %v", err)
	}
	homeAbs, err := filepath.Abs(home)
	if err != nil {
		t.Fatalf("filepath.Abs(%s): %v", home, err)
	}

	// Usa exatamente a mesma cadeia do CLI: resolveConfig() -> resolveStorageAndRepo().
	cfg := resolveConfig()
	_, db, pg := resolveStorageAndRepo(cfg, "", "", "", "isolation-probe")

	if pg != "" {
		t.Errorf("vazamento: resolveStorageAndRepo retornou Postgres herdado do ambiente/config "+
			"(%d bytes). O comando leria o banco real do usuário em vez do banco do teste.", len(pg))
	}

	if db != "" && !filepath.IsAbs(db) {
		abs, absErr := filepath.Abs(db)
		if absErr != nil {
			t.Fatalf("filepath.Abs(%s): %v", db, absErr)
		}
		db = abs
	}
	if db != "" && strings.HasPrefix(db, homeAbs+string(os.PathSeparator)) {
		t.Errorf("vazamento: resolveStorageAndRepo resolveu o SQLite para dentro do home do usuário: %s", db)
	}
}
