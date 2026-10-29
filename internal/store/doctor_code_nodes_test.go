package store

import (
	"context"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/pgtest"
)

// TestPostgresDoctorNaoContaNoDeCodigoComoDeadLink prova o ISSUE-036 no
// PostgreSQL: o alvo de uma aresta `references_code` (ADR-061) é um NÓ de
// código — ele existe no grafo, então não é link morto.
//
// Antes da correção, as 38 304 arestas da ponte do vault real contavam como
// dead link (Health Score 75 → 60) e a rotina de correção as apagava.
func TestPostgresDoctorNaoContaNoDeCodigoComoDeadLink(t *testing.T) {
	// Banco DESCARTÁVEL próprio: o Postgres do dono nunca é alvo.
	h := pgtest.New(t)
	ctx := context.Background()
	s, err := NewPostgresStore(h.DSN)
	if err != nil {
		t.Fatalf("NewPostgresStore(%s): %v", h.DBName, err)
	}
	defer s.Close()
	database := s.DB()

	const repo = "repo-doctor"
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
	nodeID := "code:" + repo + ":pkg.Alpha"
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_nodes (id, repository, type, name) VALUES ($1, $2, $3, $4)
	`, nodeID, repo, codeSymbolNodeType, "Alpha"); err != nil {
		t.Fatalf("insert do nó de código: %v", err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_edges (source_id, target_id, repository, relation, epistemic_status)
		VALUES ($1, $2, $3, $4, $5)
	`, "nota.md", nodeID, repo, "references_code", "EXTRACTED"); err != nil {
		t.Fatalf("insert da aresta references_code: %v", err)
	}

	report, err := s.DiagnoseHealth(ctx, repo)
	if err != nil {
		t.Fatalf("DiagnoseHealth falhou: %v", err)
	}
	if len(report.DeadLinks) != 0 {
		t.Errorf("dead links = %d; want 0 (o nó de código existe no grafo): %+v", len(report.DeadLinks), report.DeadLinks)
	}
}

// TestPostgresDoctorContinuaContandoAlvoInexistenteComoDeadLink é o
// contrapeso: no PostgreSQL o `target_id` NÃO tem foreign key, então um alvo
// que não existe é possível — e tem que continuar contando como link morto.
func TestPostgresDoctorContinuaContandoAlvoInexistenteComoDeadLink(t *testing.T) {
	h := pgtest.New(t)
	ctx := context.Background()
	s, err := NewPostgresStore(h.DSN)
	if err != nil {
		t.Fatalf("NewPostgresStore(%s): %v", h.DBName, err)
	}
	defer s.Close()
	database := s.DB()

	const repo = "repo-doctor"
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
		INSERT INTO graph_edges (source_id, target_id, repository, relation, epistemic_status)
		VALUES ($1, $2, $3, $4, $5)
	`, "nota.md", "alvo-que-nao-existe", repo, "links_to", "EXTRACTED"); err != nil {
		t.Fatalf("insert da aresta órfã: %v", err)
	}

	report, err := s.DiagnoseHealth(ctx, repo)
	if err != nil {
		t.Fatalf("DiagnoseHealth falhou: %v", err)
	}
	if len(report.DeadLinks) != 1 {
		t.Fatalf("dead links = %d; want 1 (o alvo não existe): %+v", len(report.DeadLinks), report.DeadLinks)
	}
	if report.DeadLinks[0].TargetID != "alvo-que-nao-existe" {
		t.Errorf("target do dead link = %q; want %q", report.DeadLinks[0].TargetID, "alvo-que-nao-existe")
	}
}

// TestPostgresFixHealthIssuesNaoApagaCodeGraph prova o outro lado do
// ISSUE-036: a correção não pode apagar a ponte notas↔código.
func TestPostgresFixHealthIssuesNaoApagaCodeGraph(t *testing.T) {
	h := pgtest.New(t)
	ctx := context.Background()
	s, err := NewPostgresStore(h.DSN)
	if err != nil {
		t.Fatalf("NewPostgresStore(%s): %v", h.DBName, err)
	}
	defer s.Close()
	database := s.DB()

	const repo = "repo-doctor"
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
	nodeID := "code:" + repo + ":pkg.Alpha"
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_nodes (id, repository, type, name) VALUES ($1, $2, $3, $4)
	`, nodeID, repo, codeSymbolNodeType, "Alpha"); err != nil {
		t.Fatalf("insert do nó de código: %v", err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_edges (source_id, target_id, repository, relation, epistemic_status)
		VALUES ($1, $2, $3, $4, $5)
	`, "nota.md", nodeID, repo, "references_code", "EXTRACTED"); err != nil {
		t.Fatalf("insert da aresta references_code: %v", err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_edges (source_id, target_id, repository, relation, epistemic_status)
		VALUES ($1, $2, $3, $4, $5)
	`, "nota.md", "alvo-que-nao-existe", repo, "links_to", "EXTRACTED"); err != nil {
		t.Fatalf("insert da aresta órfã: %v", err)
	}

	if _, err := s.FixHealthIssues(ctx, repo); err != nil {
		t.Fatalf("FixHealthIssues falhou: %v", err)
	}

	var ponte, orfas int
	if err := database.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM graph_edges WHERE relation = 'references_code' AND target_id = $1
	`, nodeID).Scan(&ponte); err != nil {
		t.Fatalf("contando a ponte: %v", err)
	}
	if err := database.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM graph_edges WHERE target_id = 'alvo-que-nao-existe'
	`).Scan(&orfas); err != nil {
		t.Fatalf("contando a aresta órfã: %v", err)
	}
	if ponte != 1 {
		t.Errorf("arestas references_code depois da correção = %d; want 1 (a ponte não pode ser apagada)", ponte)
	}
	if orfas != 0 {
		t.Errorf("arestas órfãs depois da correção = %d; want 0 (a órfã tem que ser apagada)", orfas)
	}
}
