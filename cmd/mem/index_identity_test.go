package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/config"
	"github.com/FelipeMiiller/my-memory/internal/db"
)

// writeIdentityNote grava uma nota de teste com `title:` no frontmatter e
// devolve o caminho absoluto do arquivo — que é o `documents.id` do documento.
func writeIdentityNote(t *testing.T, vault, name, title, body string) string {
	t.Helper()
	path := filepath.Join(vault, name)
	content := "---\ntitle: " + title + "\n---\n\n" + body + "\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("escrevendo %s: %v", path, err)
	}
	return path
}

// writeTaggedNote grava uma nota de teste com `tags:` no frontmatter e
// devolve o caminho absoluto do arquivo.
func writeTaggedNote(t *testing.T, vault, name, title, tag string) string {
	t.Helper()
	path := filepath.Join(vault, name)
	content := "---\ntitle: " + title + "\ntags: [" + tag + "]\n---\n\nNota com tag.\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("escrevendo %s: %v", path, err)
	}
	return path
}

// TestIndexIdentity prova o contrato de identidade do nó no indexador em lote
// (ADR-066 — GNI-01 a GNI-04):
//
//  1. a aresta de wikilink tem as DUAS pontas no `documents.id`;
//  2. a origem gravada é o `documents.id` do documento indexado;
//  3. o alvo que não resolve para documento indexado mantém o texto do
//     wikilink — é assim que o `mem doctor` enxerga o link rompido.
func TestIndexIdentity(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	docA := writeIdentityNote(t, vault, "nota-a.md", "Nota A", "Aponta para [[Nota B]] e para [[Alvo Inexistente]].")
	docB := writeIdentityNote(t, vault, "nota-b.md", "Nota B", "Nota de destino.")
	docC := writeTaggedNote(t, vault, "nota-c.md", "Nota C", "nota-b")

	database, err := db.InitDB(filepath.Join(t.TempDir(), "identity.db"))
	if err != nil {
		if strings.Contains(err.Error(), "no such module: fts5") {
			t.Skip("Pulando teste: ambiente sem FTS5")
		}
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()
	runIndexSQLite(ctx, database, nil, nil, &cfg, vault, true, false)

	// 1. O destino resolvido é o `documents.id` do documento de destino.
	var target string
	err = database.QueryRowContext(ctx,
		`SELECT target_id FROM graph_edges WHERE source_id = ? AND relation = 'links_to' AND target_id = ?`,
		docA, docB).Scan(&target)
	if err != nil {
		t.Fatalf("aresta de links_to para o documento de destino não encontrada: %v", err)
	}
	if target != docB {
		t.Errorf("destino da aresta = %q; want %q (o documents.id do documento de destino)", target, docB)
	}

	// 2. Nenhuma aresta aponta para o texto do título: a identidade antiga
	// (id de título) não pode sobreviver à correção.
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

	// 4. Toda origem gravada é o id de um documento indexado.
	var origensOrfas int
	if err := database.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM graph_edges e
		WHERE e.source_id NOT IN (SELECT id FROM documents)
	`).Scan(&origensOrfas); err != nil {
		t.Fatalf("contando as origens órfãs: %v", err)
	}
	if origensOrfas != 0 {
		t.Errorf("arestas com origem fora do índice = %d; want 0", origensOrfas)
	}

	// 5. A aresta `tagged_as` que casa com um documento aponta para o id dele.
	var tagTarget string
	err = database.QueryRowContext(ctx,
		`SELECT target_id FROM graph_edges WHERE source_id = ? AND relation = 'tagged_as'`,
		docC).Scan(&tagTarget)
	if err != nil {
		t.Fatalf("aresta tagged_as de %q não encontrada: %v", docC, err)
	}
	if tagTarget != docB {
		t.Errorf("destino da aresta tagged_as = %q; want %q (o documents.id do documento casado)", tagTarget, docB)
	}
}
