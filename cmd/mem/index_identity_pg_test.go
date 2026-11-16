package main

import (
	"context"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/config"
	"github.com/FelipeMiiller/my-memory/internal/pgtest"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// TestPostgresIndexIdentity prova o mesmo contrato de identidade do nó no
// indexador em lote, agora no engine de produção (ADR-066 — GNI-01 a GNI-04).
//
// O Postgres é o engine do cofre central: se a correção valesse só no SQLite,
// o defeito da ISSUE-047 continuaria vivo onde ele importa.
func TestPostgresIndexIdentity(t *testing.T) {
	h := pgtest.New(t)
	ctx := context.Background()

	pgStore, err := store.NewPostgresStore(h.DSN)
	if err != nil {
		t.Fatalf("NewPostgresStore(%s): %v", h.DBName, err)
	}
	defer pgStore.Close()

	vault := t.TempDir()
	docA := writeIdentityNote(t, vault, "nota-a.md", "Nota A", "Aponta para [[Nota B]] e para [[Alvo Inexistente]].")
	docB := writeIdentityNote(t, vault, "nota-b.md", "Nota B", "Nota de destino.")
	docC := writeTaggedNote(t, vault, "nota-c.md", "Nota C", "nota-b")

	const repo = "repo-identity"
	cfg := config.DefaultConfig()
	runIndexPostgres(ctx, pgStore, nil, &cfg, repo, vault, true, false)

	// 1. O destino resolvido é o `documents.id` do documento de destino.
	var target string
	err = pgStore.DB().QueryRowContext(ctx, `
		SELECT target_id FROM graph_edges
		WHERE repository = $1 AND source_id = $2 AND relation = 'links_to' AND target_id = $3
	`, repo, docA, docB).Scan(&target)
	if err != nil {
		t.Fatalf("aresta de links_to para o documento de destino não encontrada: %v", err)
	}
	if target != docB {
		t.Errorf("destino da aresta = %q; want %q (o documents.id do documento de destino)", target, docB)
	}

	// 2. Nenhuma aresta aponta para o texto do título.
	var porTitulo int
	if err := pgStore.DB().QueryRowContext(ctx, `
		SELECT COUNT(*) FROM graph_edges WHERE repository = $1 AND target_id = $2
	`, repo, "Nota B").Scan(&porTitulo); err != nil {
		t.Fatalf("contando as arestas por título: %v", err)
	}
	if porTitulo != 0 {
		t.Errorf("arestas com destino no título = %d; want 0 (a identidade canônica é o id do documento)", porTitulo)
	}

	// 3. O alvo que não resolve mantém o texto do wikilink.
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

	// 4. Toda origem gravada é o id de um documento indexado do MESMO repositório.
	var origensOrfas int
	if err := pgStore.DB().QueryRowContext(ctx, `
		SELECT COUNT(*) FROM graph_edges e
		WHERE e.repository = $1 AND e.source_id NOT IN (SELECT id FROM documents WHERE repository = $1)
	`, repo).Scan(&origensOrfas); err != nil {
		t.Fatalf("contando as origens órfãs: %v", err)
	}
	if origensOrfas != 0 {
		t.Errorf("arestas com origem fora do índice = %d; want 0", origensOrfas)
	}

	// 5. A aresta `tagged_as` que casa com um documento aponta para o id dele.
	var tagTarget string
	err = pgStore.DB().QueryRowContext(ctx, `
		SELECT target_id FROM graph_edges
		WHERE repository = $1 AND source_id = $2 AND relation = 'tagged_as'
	`, repo, docC).Scan(&tagTarget)
	if err != nil {
		t.Fatalf("aresta tagged_as de %q não encontrada: %v", docC, err)
	}
	if tagTarget != docB {
		t.Errorf("destino da aresta tagged_as = %q; want %q (o documents.id do documento casado)", tagTarget, docB)
	}
}
