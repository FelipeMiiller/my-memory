package store

import (
	"context"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/pgtest"
)

// TestPostgresNodeIdentity prova o contrato do nó canônico no PostgreSQL
// (ADR-066 — GNI-05 e GNI-06), o mesmo do SQLite:
//
//  1. o nó de um documento indexado tem `id` = `documents.id` e `name` =
//     `documents.title` — o id é a chave, o título é o rótulo;
//  2. um documento NÃO ocupa dois nós (não existe nó com o id = título);
//  3. o nó de destino não resolvido (o texto do wikilink) continua sendo
//     criado, com o texto como nome — o doctor continua enxergando o link;
//  4. o nó de código (ADR-061) é preservado.
func TestPostgresNodeIdentity(t *testing.T) {
	h := pgtest.New(t)
	ctx := context.Background()

	s, err := NewPostgresStore(h.DSN)
	if err != nil {
		t.Fatalf("NewPostgresStore(%s): %v", h.DBName, err)
	}
	defer s.Close()
	database := s.DB()

	const repo = "repo-node-identity"
	now := time.Now().Unix()

	if err := s.InsertDocumentWithMeta(ctx, repo, "nota.md", "nota.md", "Minha Nota", now, "h1", "", "resumo", "resource", 0, 0); err != nil {
		t.Fatalf("inserindo o documento: %v", err)
	}

	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_nodes (id, repository, type, name) VALUES ($1, $2, 'code_symbol', $3)
	`, "code:repo:pkg.Alpha", repo, "Alpha"); err != nil {
		t.Fatalf("inserindo o nó de código: %v", err)
	}

	if err := s.InsertEdgeWithProps(ctx, repo, "nota.md", "Alvo Inexistente", "links_to", "EXTRACTED", 1.0); err != nil {
		t.Fatalf("inserindo a aresta: %v", err)
	}

	// 1. O nó do documento tem id = documents.id e name = documents.title.
	var nodeType, name string
	if err := database.QueryRowContext(ctx, `
		SELECT type, name FROM graph_nodes WHERE repository = $1 AND id = $2
	`, repo, "nota.md").Scan(&nodeType, &name); err != nil {
		t.Fatalf("nó do documento não encontrado: %v", err)
	}
	if name != "Minha Nota" {
		t.Errorf("name do nó = %q; want %q (o título é o rótulo do nó)", name, "Minha Nota")
	}
	if nodeType != "note" {
		t.Errorf("type do nó = %q; want %q", nodeType, "note")
	}

	// 2. O documento não ocupa dois nós.
	var porTitulo int
	if err := database.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM graph_nodes WHERE repository = $1 AND id = $2
	`, repo, "Minha Nota").Scan(&porTitulo); err != nil {
		t.Fatalf("contando os nós por título: %v", err)
	}
	if porTitulo != 0 {
		t.Errorf("nós com id = título do documento = %d; want 0 (uma identidade por documento)", porTitulo)
	}

	// 3. O nó do alvo não resolvido continua existindo, com o texto como nome.
	var danglingName string
	if err := database.QueryRowContext(ctx, `
		SELECT name FROM graph_nodes WHERE repository = $1 AND id = $2
	`, repo, "Alvo Inexistente").Scan(&danglingName); err != nil {
		t.Fatalf("nó do alvo não resolvido não encontrado: %v", err)
	}
	if danglingName != "Alvo Inexistente" {
		t.Errorf("name do nó não resolvido = %q; want %q", danglingName, "Alvo Inexistente")
	}

	// 4. O nó de código é preservado.
	var codeName string
	if err := database.QueryRowContext(ctx, `
		SELECT name FROM graph_nodes WHERE repository = $1 AND id = $2
	`, repo, "code:repo:pkg.Alpha").Scan(&codeName); err != nil {
		t.Fatalf("nó de código não encontrado: %v", err)
	}
	if codeName != "Alpha" {
		t.Errorf("name do nó de código = %q; want %q (o nó de código é preservado)", codeName, "Alpha")
	}

	// 5. O rótulo acompanha o título: reindexar com outro título renomeia o nó.
	if err := s.InsertDocumentWithMeta(ctx, repo, "nota.md", "nota.md", "Titulo Novo", now, "h2", "", "resumo", "resource", 0, 0); err != nil {
		t.Fatalf("reindexando o documento: %v", err)
	}
	if err := database.QueryRowContext(ctx, `
		SELECT name FROM graph_nodes WHERE repository = $1 AND id = $2
	`, repo, "nota.md").Scan(&name); err != nil {
		t.Fatalf("nó do documento não encontrado: %v", err)
	}
	if name != "Titulo Novo" {
		t.Errorf("name do nó após a reindexação = %q; want %q", name, "Titulo Novo")
	}
}
