package watcher

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/pgtest"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// capturePgAdminURL guarda a URL do Postgres antes que qualquer neutralização
// do ambiente a apague — o mesmo init() que cmd/mem usa.
//
// POR QUE UM init(): o init() de um pacote roda na inicialização do binário,
// ou seja, ANTES do TestMain. A captura tem que acontecer nessa ordem.
func init() {
	pgtest.Capture()
}

// writeWatcherNote grava uma nota de teste com `title:` no frontmatter.
func writeWatcherNote(t *testing.T, path, title, body string) {
	t.Helper()
	content := "---\ntitle: " + title + "\n---\n\n" + body + "\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("escrevendo %s: %v", path, err)
	}
}

// TestIndexIdentity prova o contrato de identidade do nó na indexação
// cirúrgica (ADR-066 — GNI-01 a GNI-04):
//
//  1. a aresta gravada tem as DUAS pontas no `documents.id`;
//  2. o alvo que não resolve para documento indexado mantém o texto do
//     wikilink — é assim que o `mem doctor` enxerga o link rompido.
//
// O watcher não pode reintroduzir a fenda: o primeiro arquivo que ele toca
// reescreve as arestas dele.
func TestIndexIdentity(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()

	database, err := db.InitDB(filepath.Join(vault, "watcher.db"))
	if err != nil {
		if strings.Contains(err.Error(), "no such module: fts5") {
			t.Skip("Pulando teste: ambiente sem FTS5")
		}
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	// A nota de destino entra primeiro no índice; a que a referencia entra
	// depois — no watcher, o alvo de um arquivo já está indexado.
	docB := filepath.Join(vault, "nota-b.md")
	writeWatcherNote(t, docB, "Nota B", "Nota de destino.")

	docA := filepath.Join(vault, "nota-a.md")
	writeWatcherNote(t, docA, "Nota A", "Aponta para [[Nota B]] e para [[Alvo Inexistente]].")

	if _, err := IndexSingleFileSQLite(ctx, database, nil, nil, docB, false); err != nil {
		t.Fatalf("indexando a nota de destino: %v", err)
	}
	res, err := IndexSingleFileSQLite(ctx, database, nil, nil, docA, false)
	if err != nil {
		t.Fatalf("indexando a nota que referencia: %v", err)
	}
	if res.EdgesCount != 2 {
		t.Errorf("arestas extraídas = %d; want 2", res.EdgesCount)
	}

	// 1. A aresta resolvida tem as duas pontas no id do documento.
	var target string
	err = database.QueryRowContext(ctx,
		`SELECT target_id FROM graph_edges WHERE source_id = ? AND relation = 'links_to' AND target_id = ?`,
		docA, docB).Scan(&target)
	if err != nil {
		t.Fatalf("aresta para o documento de destino não encontrada: %v", err)
	}
	if target != docB {
		t.Errorf("destino da aresta = %q; want %q (o documents.id do documento de destino)", target, docB)
	}

	// 2. Nenhuma aresta aponta para o texto do título.
	var porTitulo int
	if err := database.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM graph_edges WHERE target_id = ?`, "Nota B").Scan(&porTitulo); err != nil {
		t.Fatalf("contando as arestas por título: %v", err)
	}
	if porTitulo != 0 {
		t.Errorf("arestas com destino no título = %d; want 0 (a identidade canônica é o id do documento)", porTitulo)
	}

	// 3. O alvo que não resolve mantém o texto do wikilink.
	var dangling string
	err = database.QueryRowContext(ctx,
		`SELECT target_id FROM graph_edges WHERE source_id = ? AND target_id = ?`,
		docA, "Alvo Inexistente").Scan(&dangling)
	if err != nil {
		t.Fatalf("aresta com alvo não resolvido não encontrada: %v", err)
	}
	if dangling != "Alvo Inexistente" {
		t.Errorf("destino do link rompido = %q; want %q (o texto do wikilink é preservado)", dangling, "Alvo Inexistente")
	}
}

// TestIndexIdentityPostgres prova o mesmo contrato de identidade do nó na
// indexação cirúrgica do engine de produção: o watcher do Postgres grava o
// MESMO destino que o do SQLite (ADR-066, GNI-04).
func TestIndexIdentityPostgres(t *testing.T) {
	h := pgtest.New(t)
	ctx := context.Background()

	pgStore, err := store.NewPostgresStore(h.DSN)
	if err != nil {
		t.Fatalf("NewPostgresStore(%s): %v", h.DBName, err)
	}
	defer pgStore.Close()

	vault := t.TempDir()
	docB := filepath.Join(vault, "nota-b.md")
	writeWatcherNote(t, docB, "Nota B", "Nota de destino.")

	docA := filepath.Join(vault, "nota-a.md")
	writeWatcherNote(t, docA, "Nota A", "Aponta para [[Nota B]] e para [[Alvo Inexistente]].")

	const repo = "repo-watcher-identity"
	if _, err := IndexSingleFilePostgres(ctx, pgStore, nil, repo, docB, false); err != nil {
		t.Fatalf("indexando a nota de destino: %v", err)
	}
	if _, err := IndexSingleFilePostgres(ctx, pgStore, nil, repo, docA, false); err != nil {
		t.Fatalf("indexando a nota que referencia: %v", err)
	}

	var target string
	err = pgStore.DB().QueryRowContext(ctx, `
		SELECT target_id FROM graph_edges
		WHERE repository = $1 AND source_id = $2 AND relation = 'links_to' AND target_id = $3
	`, repo, docA, docB).Scan(&target)
	if err != nil {
		t.Fatalf("aresta para o documento de destino não encontrada: %v", err)
	}
	if target != docB {
		t.Errorf("destino da aresta = %q; want %q (o documents.id do documento de destino)", target, docB)
	}

	var dangling string
	err = pgStore.DB().QueryRowContext(ctx, `
		SELECT target_id FROM graph_edges
		WHERE repository = $1 AND source_id = $2 AND target_id = $3
	`, repo, docA, "Alvo Inexistente").Scan(&dangling)
	if err != nil {
		t.Fatalf("aresta com alvo não resolvido não encontrada: %v", err)
	}
	if dangling != "Alvo Inexistente" {
		t.Errorf("destino do link rompido = %q; want %q (o texto do wikilink é preservado)", dangling, "Alvo Inexistente")
	}
}
