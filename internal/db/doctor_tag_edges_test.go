package db

import (
	"context"
	"path/filepath"
	"testing"
)

// TestDoctorNaoContaArestaDeTagVivaComoDeadLink prova a ISSUE-038: o alvo de
// uma aresta `tagged_as` é uma TAG — que no grafo é um nó (e vive também em
// `document_tags`, ADR-055) — e o doctor não pode chamá-la de link morto.
//
// No SQLite o `graph_edges.target_id` tem FK para `graph_nodes(id)`, então o
// alvo **sempre** existe como nó: é por isso que a checagem do doctor precisa
// reconhecer o nó, e não exigir que o alvo seja um documento.
//
// Antes da correção, as 80 arestas de tag do cofre central viravam "dead
// links" e o Health Score do vault caía de ~90 para 60.
func TestDoctorNaoContaArestaDeTagVivaComoDeadLink(t *testing.T) {
	database, err := InitDB(filepath.Join(t.TempDir(), "doctor_tag.db"))
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
	// A tag: o nó que o `graph_edges` referencia + a linha de `document_tags`.
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_nodes (id, type, name) VALUES ('ddd', 'tag', 'ddd')
	`); err != nil {
		t.Fatalf("insert do nó de tag: %v", err)
	}
	// A tag literal vive em `document_tags` (ADR-055) — é ela que prova que a
	// tag existe, mesmo quando não casa com nenhum documento.
	if _, err := database.ExecContext(ctx, `
		INSERT INTO document_tags (document_id, tag, tag_key, created_at) VALUES ('nota.md', 'ddd', 'ddd', 0)
	`); err != nil {
		t.Fatalf("insert da tag: %v", err)
	}
	// Aresta para a tag viva: o alvo é um NÓ do grafo (é assim que a ponte
	// indexa as tags do frontmatter, e é o caso real do cofre central: 264
	// arestas de tag, das quais só 41 tags estão na tabela `document_tags`).
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_edges (source_id, target_id, relation, epistemic_status) VALUES ('nota.md', 'ddd', 'tagged_as', 'TAG')
	`); err != nil {
		t.Fatalf("insert da aresta de tag viva: %v", err)
	}

	report, err := DiagnoseHealth(ctx, database)
	if err != nil {
		t.Fatalf("DiagnoseHealth falhou: %v", err)
	}
	if len(report.DeadLinks) != 0 {
		t.Errorf("dead links = %d; want 0 (a tag existe como nó e em document_tags): %+v", len(report.DeadLinks), report.DeadLinks)
	}
}

// TestCorrecaoDeDeadLinksPreservaArestaDeTagViva prova o terceiro lado do
// contrato: a rotina de correção **preserva** a aresta de tag viva. (A
// ISSUE-036 mostrou que essa rotina apagava o code graph inteiro.)
func TestCorrecaoDeDeadLinksPreservaArestaDeTagViva(t *testing.T) {
	database, err := InitDB(filepath.Join(t.TempDir(), "doctor_tag_fix.db"))
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
		INSERT INTO graph_nodes (id, type, name) VALUES ('ddd', 'tag', 'ddd')
	`); err != nil {
		t.Fatalf("insert do nó de tag: %v", err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO document_tags (document_id, tag, tag_key, created_at) VALUES ('nota.md', 'ddd', 'ddd', 0)
	`); err != nil {
		t.Fatalf("insert da tag: %v", err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_edges (source_id, target_id, relation, epistemic_status) VALUES ('nota.md', 'ddd', 'tagged_as', 'TAG')
	`); err != nil {
		t.Fatalf("insert da aresta de tag viva: %v", err)
	}

	if _, err := FixHealthIssues(ctx, database); err != nil {
		t.Fatalf("FixHealthIssues falhou: %v", err)
	}

	var vivas int
	if err := database.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM graph_edges WHERE target_id = 'ddd'
	`).Scan(&vivas); err != nil {
		t.Fatalf("contando a aresta viva: %v", err)
	}
	if vivas != 1 {
		t.Errorf("aresta de tag viva depois da correção = %d; want 1 (ela não é link morto)", vivas)
	}
}
