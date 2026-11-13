package store

import (
	"context"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/pgtest"
)

// TestPostgresDoctorNaoContaArestaDeTagVivaComoDeadLink prova a ISSUE-038 no
// engine de produção.
//
// No PostgreSQL o `graph_edges.target_id` **não** tem foreign key (só o
// `source_id` tem), então uma aresta de tag pode apontar para uma tag que não
// existe — e é exatamente esse caso que o doctor precisa continuar reportando.
func TestPostgresDoctorNaoContaArestaDeTagVivaComoDeadLink(t *testing.T) {
	// Banco DESCARTÁVEL próprio: o Postgres do dono nunca é alvo.
	h := pgtest.New(t)
	ctx := context.Background()

	s, err := NewPostgresStore(h.DSN)
	if err != nil {
		t.Fatalf("NewPostgresStore(%s): %v", h.DBName, err)
	}
	t.Cleanup(func() { s.Close() })
	database := s.DB()

	const repo = "repo-tag"
	if _, err := database.ExecContext(ctx, `
		INSERT INTO documents (id, repository, path, title, updated_at) VALUES ($1, $2, $3, $4, 0)
	`, "nota.md", repo, "nota.md", "Nota"); err != nil {
		t.Fatalf("insert do documento: %v", err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_nodes (id, repository, type, name) VALUES ($1, $2, $3, $4)
	`, "nota.md", repo, "note", "Nota"); err != nil {
		t.Fatalf("insert do nó da nota: %v", err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_nodes (id, repository, type, name) VALUES ($1, $2, $3, $4)
	`, "ddd", repo, "note", "ddd"); err != nil {
		t.Fatalf("insert do nó de tag: %v", err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO document_tags (repository, document_id, tag, tag_key, created_at) VALUES ($1, $2, $3, $4, 0)
	`, repo, "nota.md", "ddd", "ddd"); err != nil {
		t.Fatalf("insert da tag: %v", err)
	}
	// A aresta de tag VIVA cujo alvo é um NÓ (o caso real do cofre central:
	// 264 arestas de tag, e só 41 tags na tabela `document_tags`).
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_edges (source_id, target_id, repository, relation, epistemic_status) VALUES ($1, $2, $3, $4, $5)
	`, "nota.md", "ddd", repo, "tagged_as", "TAG"); err != nil {
		t.Fatalf("insert da aresta de tag viva: %v", err)
	}
	// A aresta de tag VIVA cujo alvo é uma TAG literal (só existe em
	// `document_tags`, sem nó no grafo).
	if _, err := database.ExecContext(ctx, `
		INSERT INTO document_tags (repository, document_id, tag, tag_key, created_at) VALUES ($1, $2, $3, $4, 0)
	`, repo, "nota.md", "tag-pura", "tag-pura"); err != nil {
		t.Fatalf("insert da tag pura: %v", err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_edges (source_id, target_id, repository, relation, epistemic_status) VALUES ($1, $2, $3, $4, $5)
	`, "nota.md", "tag-pura", repo, "tagged_as", "TAG"); err != nil {
		t.Fatalf("insert da aresta de tag pura: %v", err)
	}
	// A aresta de tag MORTA (sem tag e sem nó): tem de continuar sendo morta.
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_edges (source_id, target_id, repository, relation, epistemic_status) VALUES ($1, $2, $3, $4, $5)
	`, "nota.md", "tag-que-nao-existe", repo, "tagged_as", "TAG"); err != nil {
		t.Fatalf("insert da aresta de tag morta: %v", err)
	}

	report, err := s.DiagnoseHealth(ctx, repo)
	if err != nil {
		t.Fatalf("DiagnoseHealth falhou: %v", err)
	}
	if len(report.DeadLinks) != 1 {
		t.Fatalf("dead links = %d; want 1 (só a tag que não existe): %+v", len(report.DeadLinks), report.DeadLinks)
	}
	if report.DeadLinks[0].TargetID != "tag-que-nao-existe" {
		t.Errorf("target do dead link = %q; want %q", report.DeadLinks[0].TargetID, "tag-que-nao-existe")
	}
}

// TestPostgresFixHealthIssuesPreservaArestaDeTagViva prova o terceiro lado do
// contrato no engine de produção: a correção apaga a aresta morta e preserva a
// viva.
func TestPostgresFixHealthIssuesPreservaArestaDeTagViva(t *testing.T) {
	h := pgtest.New(t)
	ctx := context.Background()

	s, err := NewPostgresStore(h.DSN)
	if err != nil {
		t.Fatalf("NewPostgresStore(%s): %v", h.DBName, err)
	}
	t.Cleanup(func() { s.Close() })
	database := s.DB()

	const repo = "repo-tag"
	if _, err := database.ExecContext(ctx, `
		INSERT INTO documents (id, repository, path, title, updated_at) VALUES ($1, $2, $3, $4, 0)
	`, "nota.md", repo, "nota.md", "Nota"); err != nil {
		t.Fatalf("insert do documento: %v", err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_nodes (id, repository, type, name) VALUES ($1, $2, $3, $4)
	`, "nota.md", repo, "note", "Nota"); err != nil {
		t.Fatalf("insert do nó da nota: %v", err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO document_tags (repository, document_id, tag, tag_key, created_at) VALUES ($1, $2, $3, $4, 0)
	`, repo, "nota.md", "ddd", "ddd"); err != nil {
		t.Fatalf("insert da tag: %v", err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_edges (source_id, target_id, repository, relation, epistemic_status) VALUES ($1, $2, $3, $4, $5)
	`, "nota.md", "ddd", repo, "tagged_as", "TAG"); err != nil {
		t.Fatalf("insert da aresta de tag viva: %v", err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_edges (source_id, target_id, repository, relation, epistemic_status) VALUES ($1, $2, $3, $4, $5)
	`, "nota.md", "tag-que-nao-existe", repo, "tagged_as", "TAG"); err != nil {
		t.Fatalf("insert da aresta de tag morta: %v", err)
	}

	if _, err := s.FixHealthIssues(ctx, repo); err != nil {
		t.Fatalf("FixHealthIssues falhou: %v", err)
	}

	var vivas, mortas int
	if err := database.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM graph_edges WHERE repository = $1 AND target_id = 'ddd'
	`, repo).Scan(&vivas); err != nil {
		t.Fatalf("contando a aresta viva: %v", err)
	}
	if err := database.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM graph_edges WHERE repository = $1 AND target_id = 'tag-que-nao-existe'
	`, repo).Scan(&mortas); err != nil {
		t.Fatalf("contando a aresta morta: %v", err)
	}
	if vivas != 1 {
		t.Errorf("aresta de tag viva depois da correção = %d; want 1", vivas)
	}
	if mortas != 0 {
		t.Errorf("aresta de tag morta depois da correção = %d; want 0", mortas)
	}
}
