package db

import (
	"context"
	"path/filepath"
	"testing"
)

func TestDoctorNaoContaNoDeCodigoComoDeadLink(t *testing.T) {
	database, err := InitDB(filepath.Join(t.TempDir(), "doctor_code.db"))
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()
	ctx := context.Background()

	if _, err := database.ExecContext(ctx, `
		INSERT INTO documents (id, path, title, updated_at) VALUES ('nota.md', 'nota.md', 'Nota', 0)
	`); err != nil {
		t.Fatalf("insert do documento: %v", err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_nodes (id, type, name) VALUES ('nota.md', 'note', 'Nota')
	`); err != nil {
		t.Fatalf("insert do nó da nota: %v", err)
	}

	nodeID := CodeSymbolNodeID("repo-x", "pkg.Alpha")
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_nodes (id, type, name) VALUES (?, ?, ?)
	`, nodeID, CodeSymbolNodeType, "Alpha"); err != nil {
		t.Fatalf("insert do nó de código: %v", err)
	}

	// A aresta da ponte: documento → nó de código.
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_edges (source_id, target_id, relation, epistemic_status) VALUES (?, ?, ?, ?)
	`, "nota.md", nodeID, BridgeRelation, "EXTRACTED"); err != nil {
		t.Fatalf("insert da aresta references_code: %v", err)
	}

	report, err := DiagnoseHealth(ctx, database)
	if err != nil {
		t.Fatalf("DiagnoseHealth falhou: %v", err)
	}
	if len(report.DeadLinks) != 0 {
		t.Errorf("dead links = %d; want 0 (o nó de código existe no grafo): %+v", len(report.DeadLinks), report.DeadLinks)
	}
}

// TestDoctorContinuaContandoAlvoQueNaoENotaComoDeadLink é o contrapeso: a
// correção não pode transformar "alvo que não é documento nem nó de código" em
// "vivo".
func TestDoctorContinuaContandoAlvoQueNaoENotaComoDeadLink(t *testing.T) {
	database, err := InitDB(filepath.Join(t.TempDir(), "doctor_dead.db"))
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()
	ctx := context.Background()

	if _, err := database.ExecContext(ctx, `
		INSERT INTO documents (id, path, title, updated_at) VALUES ('nota.md', 'nota.md', 'Nota', 0)
	`); err != nil {
		t.Fatalf("insert do documento: %v", err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_nodes (id, type, name) VALUES ('nota.md', 'note', 'Nota')
	`); err != nil {
		t.Fatalf("insert do nó da nota: %v", err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_nodes (id, type, name) VALUES ('conceito.md', 'concept', 'Conceito')
	`); err != nil {
		t.Fatalf("insert do nó de conceito: %v", err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_edges (source_id, target_id, relation, epistemic_status) VALUES (?, ?, ?, ?)
	`, "nota.md", "conceito.md", "links_to", "EXTRACTED"); err != nil {
		t.Fatalf("insert da aresta para conceito: %v", err)
	}

	report, err := DiagnoseHealth(ctx, database)
	if err != nil {
		t.Fatalf("DiagnoseHealth falhou: %v", err)
	}
	if len(report.DeadLinks) != 1 {
		t.Fatalf("dead links = %d; want 1 (o alvo não é documento nem nó de código): %+v", len(report.DeadLinks), report.DeadLinks)
	}
	if report.DeadLinks[0].TargetID != "conceito.md" {
		t.Errorf("target do dead link = %q; want %q", report.DeadLinks[0].TargetID, "conceito.md")
	}
}

// TestCorrecaoDeDeadLinksNaoApagaCodeGraph prova o outro lado do ISSUE-036: a
// rotina de correção não pode apagar a ponte notas↔código.
func TestCorrecaoDeDeadLinksNaoApagaCodeGraph(t *testing.T) {
	database, err := InitDB(filepath.Join(t.TempDir(), "doctor_fix.db"))
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()
	ctx := context.Background()

	if _, err := database.ExecContext(ctx, `
		INSERT INTO documents (id, path, title, updated_at) VALUES ('nota.md', 'nota.md', 'Nota', 0)
	`); err != nil {
		t.Fatalf("insert do documento: %v", err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_nodes (id, type, name) VALUES ('nota.md', 'note', 'Nota')
	`); err != nil {
		t.Fatalf("insert do nó da nota: %v", err)
	}
	nodeID := CodeSymbolNodeID("repo-x", "pkg.Alpha")
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_nodes (id, type, name) VALUES (?, ?, ?)
	`, nodeID, CodeSymbolNodeType, "Alpha"); err != nil {
		t.Fatalf("insert do nó de código: %v", err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_nodes (id, type, name) VALUES ('conceito.md', 'concept', 'Conceito')
	`); err != nil {
		t.Fatalf("insert do nó de conceito: %v", err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_edges (source_id, target_id, relation, epistemic_status) VALUES (?, ?, ?, ?)
	`, "nota.md", nodeID, BridgeRelation, "EXTRACTED"); err != nil {
		t.Fatalf("insert da aresta references_code: %v", err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_edges (source_id, target_id, relation, epistemic_status) VALUES (?, ?, ?, ?)
	`, "nota.md", "conceito.md", "links_to", "EXTRACTED"); err != nil {
		t.Fatalf("insert da aresta para conceito: %v", err)
	}

	if _, err := FixHealthIssues(ctx, database); err != nil {
		t.Fatalf("FixHealthIssues falhou: %v", err)
	}

	var ponte, orfas int
	if err := database.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM graph_edges WHERE relation = ? AND target_id = ?
	`, BridgeRelation, nodeID).Scan(&ponte); err != nil {
		t.Fatalf("contando a ponte: %v", err)
	}
	if err := database.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM graph_edges WHERE target_id = 'conceito.md'
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
