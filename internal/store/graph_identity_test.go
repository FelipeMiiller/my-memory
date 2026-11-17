package store

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

// TestPostgresResolveCanonical prova o contrato do resolvedor canônico no
// PostgreSQL (ADR-066 — GNI-07 a GNI-09), o mesmo do SQLite:
//
//  1. o path, o `title:` e o `alias:` do MESMO documento devolvem o mesmo
//     `documents.id`, com o filtro de repositório respeitado;
//  2. uma consulta pelo título devolve o `documents.id` — nunca o id de um
//     nó de estilo título;
//  3. sem raiz do vault, um alias devolve o erro de nó não encontrado.
func TestPostgresResolveCanonical(t *testing.T) {
	h := pgtest.New(t)
	ctx := context.Background()

	s, err := NewPostgresStore(h.DSN)
	if err != nil {
		t.Fatalf("NewPostgresStore(%s): %v", h.DBName, err)
	}
	defer s.Close()
	database := s.DB()

	const repo = "repo-resolve-canonical"
	const outroRepo = "repo-resolve-canonical-2"
	now := time.Now().Unix()

	// Vault de teste com uma nota que declara `title:` e `alias:`.
	vault := t.TempDir()
	notaPath := filepath.Join(vault, "nota.md")
	content := "---\ntitle: Minha Nota\naliases: [Meu Alias]\n---\n\nCorpo da nota.\n"
	if err := os.WriteFile(notaPath, []byte(content), 0644); err != nil {
		t.Fatalf("escrevendo a nota: %v", err)
	}

	if err := s.InsertDocumentWithMeta(ctx, repo, notaPath, notaPath, "Minha Nota", now, "h1", "", "resumo", "resource", 0, 0); err != nil {
		t.Fatalf("inserindo o documento: %v", err)
	}

	// Documento homônimo em OUTRO repositório: a resolução tem que respeitar
	// o repositório auditado.
	outroPath := filepath.Join(vault, "outro.md")
	if err := s.InsertDocumentWithMeta(ctx, outroRepo, outroPath, outroPath, "Minha Nota", now, "h2", "", "resumo", "resource", 0, 0); err != nil {
		t.Fatalf("inserindo o documento homônimo: %v", err)
	}

	// Nó de estilo título (o outro lado da fenda): existe no grafo, mas
	// NUNCA pode sequestrar a identidade de um documento.
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_nodes (id, repository, type, name) VALUES ($1, $2, 'note', $3)
	`, "Minha Nota", repo, "Minha Nota"); err != nil {
		t.Fatalf("inserindo o nó de estilo título: %v", err)
	}

	// 1. Path, `title:` e `alias:` do mesmo documento → o mesmo id.
	byPath, _, err := s.ResolveNodeCanonicalIDWithRoot(ctx, repo, notaPath, vault)
	if err != nil {
		t.Fatalf("resolvendo pelo path: %v", err)
	}
	byTitle, _, err := s.ResolveNodeCanonicalIDWithRoot(ctx, repo, "title:Minha Nota", vault)
	if err != nil {
		t.Fatalf("resolvendo pelo title:: %v", err)
	}
	byAlias, _, err := s.ResolveNodeCanonicalIDWithRoot(ctx, repo, "alias:Meu Alias", vault)
	if err != nil {
		t.Fatalf("resolvendo pelo alias:: %v", err)
	}
	if byPath != notaPath {
		t.Errorf("id pelo path = %q; want %q", byPath, notaPath)
	}
	if byTitle != byPath {
		t.Errorf("id pelo title: = %q; want %q (o mesmo nó)", byTitle, byPath)
	}
	if byAlias != byPath {
		t.Errorf("id pelo alias: = %q; want %q (o mesmo nó)", byAlias, byPath)
	}

	// 2. A consulta pelo título devolve o `documents.id` do repositório
	// auditado — nunca o id do nó de estilo título nem o do outro repo.
	byPlainTitle, _, err := s.ResolveNodeCanonicalIDWithRoot(ctx, repo, "Minha Nota", vault)
	if err != nil {
		t.Fatalf("resolvendo pelo título: %v", err)
	}
	if byPlainTitle != notaPath {
		t.Errorf("id pelo título = %q; want %q (o documents.id do repositório auditado)", byPlainTitle, notaPath)
	}

	// 3. Sem raiz do vault, o alias devolve o erro de nó não encontrado.
	_, _, err = s.ResolveNodeCanonicalIDWithRoot(ctx, repo, "alias:Meu Alias", "")
	if err == nil {
		t.Fatal("sem raiz do vault, o alias devolve erro; obteve nil")
	}
	if !strings.Contains(err.Error(), "não encontrado no grafo") {
		t.Errorf("erro do alias sem raiz = %q; want %q", err.Error(), "nó '...' não encontrado no grafo")
	}
}
