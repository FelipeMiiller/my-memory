package main

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// chdirToVault move o processo para o vault temporário e devolve o caminho
// original. O detector de staleness usa o diretório de trabalho atual como
// raiz do vault (mesmo default do `mem status`), então o teste precisa estar
// dentro do vault para exercitar o caminho real.
func chdirToVault(t *testing.T, dir string) {
	t.Helper()

	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd falhou: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir falhou: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
}

// newSearchStalenessFixture monta um vault temporário com uma nota indexada e
// (quando stale) editada depois da indexação, e deixa o processo dentro dele.
// O banco é temporário: nenhum vault real e nenhum `.memory/memory.db` é
// tocado.
func newSearchStalenessFixture(t *testing.T, stale bool) *sql.DB {
	t.Helper()

	vaultDir := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "search_banner.db")

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	docPath := filepath.Join(vaultDir, "nota.md")
	if err := os.WriteFile(docPath, []byte("# Nota\n\narquitetura do sistema"), 0644); err != nil {
		t.Fatalf("WriteFile falhou: %v", err)
	}
	editedAt := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
	_ = os.Chtimes(docPath, editedAt, editedAt)

	ctx := context.Background()
	if err := db.InsertDocument(ctx, database, "nota", "nota.md", "Nota", editedAt.Unix(), "hash1"); err != nil {
		t.Fatalf("InsertDocument falhou: %v", err)
	}
	if err := db.InsertChunk(ctx, database, "nota#0", "nota", "arquitetura do sistema", 0, nil); err != nil {
		t.Fatalf("InsertChunk falhou: %v", err)
	}

	if stale {
		// Edição posterior à indexação: o mtime do disco passa o persistido.
		newEdit := time.Now().Truncate(time.Second)
		_ = os.Chtimes(docPath, newEdit, newEdit)
	}

	chdirToVault(t, vaultDir)
	return database
}

// runSearchStalenessFixture roda a busca e devolve o stdout capturado.
func runSearchStalenessFixture(t *testing.T, database *sql.DB) string {
	t.Helper()

	return captureStdout(t, func() {
		runSearchSQLite(context.Background(), database, nil, nil, "arquitetura", "fts", false, 10, 60,
			store.DefaultDecayOptions(), store.SearchOptions{Level: "l1"})
	})
}

// TestSearchStalenessBannerVemAntesDoResultado: num vault desatualizado o
// `mem search` avisa ANTES da linha de cabeçalho e do primeiro resultado.
func TestSearchStalenessBannerVemAntesDoResultado(t *testing.T) {
	database := newSearchStalenessFixture(t, true)

	out := runSearchStalenessFixture(t, database)

	bannerIdx := strings.Index(out, "O índice local está desatualizado")
	if bannerIdx < 0 {
		t.Fatalf("esperava o aviso de desatualização na saída do search, obteve:\n%s", out)
	}
	headerIdx := strings.Index(out, "🔎 Buscando")
	if headerIdx < 0 {
		t.Fatalf("esperava o cabeçalho da busca na saída, obteve:\n%s", out)
	}
	if bannerIdx > headerIdx {
		t.Errorf("o aviso tem de sair antes do cabeçalho da busca, obteve:\n%s", out)
	}
	resultIdx := strings.Index(out, "--- [1]")
	if resultIdx < 0 {
		t.Fatalf("esperava um resultado na saída do search, obteve:\n%s", out)
	}
	if bannerIdx > resultIdx {
		t.Errorf("o aviso tem de sair antes do primeiro resultado, obteve:\n%s", out)
	}
}

// TestSearchStalenessBannerAusenteEmSync: com o índice em sincronia o
// `mem search` não imprime aviso nenhum.
func TestSearchStalenessBannerAusenteEmSync(t *testing.T) {
	database := newSearchStalenessFixture(t, false)

	out := runSearchStalenessFixture(t, database)

	if strings.Contains(out, "O índice local está desatualizado") {
		t.Errorf("não deveria haver aviso de desatualização com o índice em sincronia, obteve:\n%s", out)
	}
	if !strings.Contains(out, "🔎 Buscando") {
		t.Errorf("a busca deveria continuar normalmente, obteve:\n%s", out)
	}
}
