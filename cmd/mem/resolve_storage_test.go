package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/config"
)

// TestResolveStorageAndRepo_DBPathFallback cobre ISSUE-004:
// quando `mem index` (ou outro subcomando) é executado sem `--db` e sem
// `storage.sqlite_path` no config, o fallback deve respeitar o local canônico
// do vault (`.memory/memory.db` quando `.memory/` é diretório).
//
// Bug histórico: fallback caía em CWD `memory.db`, deixando o vault em
// `.memory/memory.db` órfão.
func TestResolveStorageAndRepo_DBPathFallback(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) (workDir string, cfg *config.Config)
		dbPathArg   string
		wantDBPath  string
		description string
	}{
		{
			name: "fallback para .memory/memory.db quando .memory/ existe e DB ausente",
			setup: func(t *testing.T) (string, *config.Config) {
				dir := t.TempDir()
				mustMkdir(t, filepath.Join(dir, ".memory"))
				oldWd, _ := os.Getwd()
				mustChdir(t, dir)
				t.Cleanup(func() { _ = os.Chdir(oldWd) })
				return dir, &config.Config{}
			},
			dbPathArg:   "",
			wantDBPath:  filepath.Join(".memory", "memory.db"),
			description: "Regressão ISSUE-004: sem --db e sem config, .memory/ deve ganhar o DB canônico (não CWD memory.db)",
		},
		{
			name: "fallback para CWD memory.db quando .memory/ não existe",
			setup: func(t *testing.T) (string, *config.Config) {
				dir := t.TempDir()
				oldWd, _ := os.Getwd()
				mustChdir(t, dir)
				t.Cleanup(func() { _ = os.Chdir(oldWd) })
				return dir, &config.Config{}
			},
			dbPathArg:   "",
			wantDBPath:  "memory.db",
			description: "Sem .memory/, mantém fallback em CWD (vault não-inicializado)",
		},
		{
			name: "config storage.sqlite_path tem precedência sobre heurística .memory/",
			setup: func(t *testing.T) (string, *config.Config) {
				dir := t.TempDir()
				mustMkdir(t, filepath.Join(dir, ".memory"))
				oldWd, _ := os.Getwd()
				mustChdir(t, dir)
				t.Cleanup(func() { _ = os.Chdir(oldWd) })
				return dir, &config.Config{
					Storage: config.StorageConfig{SQLitePath: "/custom/path/forced.db"},
				}
			},
			dbPathArg:   "",
			wantDBPath:  "/custom/path/forced.db",
			description: "config explícito ganha do fallback automático",
		},
		{
			name: "--db flag explícito tem precedência sobre config e heurística",
			setup: func(t *testing.T) (string, *config.Config) {
				dir := t.TempDir()
				mustMkdir(t, filepath.Join(dir, ".memory"))
				oldWd, _ := os.Getwd()
				mustChdir(t, dir)
				t.Cleanup(func() { _ = os.Chdir(oldWd) })
				return dir, &config.Config{
					Storage: config.StorageConfig{SQLitePath: "/from/config.db"},
				}
			},
			dbPathArg:   "/explicit/flag.db",
			wantDBPath:  "/explicit/flag.db",
			description: "Flag --db sempre vence (override consciente do usuário)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, cfg := tc.setup(t)
			_, db, _ := resolveStorageAndRepo(cfg, "", tc.dbPathArg, "", "default-repo")

			// Resolve para path absoluto para comparar contra filepath.Join relativo
			absWant, err := filepath.Abs(tc.wantDBPath)
			if err != nil {
				t.Fatalf("falha ao resolver path absoluto: %v", err)
			}
			absGot, err := filepath.Abs(db)
			if err != nil {
				t.Fatalf("falha ao resolver path retornado: %v", err)
			}

			if absGot != absWant {
				t.Errorf("%s\n  got  = %s\n  want = %s", tc.description, db, tc.wantDBPath)
			}
		})
	}
}

func mustMkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("falha ao criar %s: %v", dir, err)
	}
}

func mustChdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("falha ao chdir para %s: %v", dir, err)
	}
}
