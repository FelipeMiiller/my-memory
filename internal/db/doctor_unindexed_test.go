package db

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// newDoctorFixture monta um banco com uma nota, o no da nota, o no de cada
// alvo e as arestas informadas.
//
// O `graph_edges.target_id` tem FK para `graph_nodes(id)` no SQLite, então o
// alvo de toda aresta precisa existir como no — e a checagem do doctor é que
// decide se ele e link morto.
func newDoctorFixture(t *testing.T, edges [][3]string) *sql.DB {
	t.Helper()

	database, err := InitDB(filepath.Join(t.TempDir(), "doctor_unindexed.db"))
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	ctx := context.Background()
	if _, err := database.ExecContext(ctx, `
		INSERT INTO documents (id, path, title, updated_at) VALUES ('nota.md', 'nota.md', 'Nota', 0)
	`); err != nil {
		t.Fatalf("insert do documento: %v", err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_nodes (id, type, name) VALUES ('nota.md', 'note', 'Nota')
	`); err != nil {
		t.Fatalf("insert do no da nota: %v", err)
	}

	for _, edge := range edges {
		if _, err := database.ExecContext(ctx, `
			INSERT OR IGNORE INTO graph_nodes (id, type, name) VALUES (?, 'note', ?)
		`, edge[1], edge[1]); err != nil {
			t.Fatalf("insert do no do alvo %q: %v", edge[1], err)
		}
		if _, err := database.ExecContext(ctx, `
			INSERT INTO graph_edges (source_id, target_id, relation, epistemic_status) VALUES (?, ?, ?, 'EXTRACTED')
		`, edge[0], edge[1], edge[2]); err != nil {
			t.Fatalf("insert da aresta %v: %v", edge, err)
		}
	}

	return database
}

// newVaultFixture monta um vault de disco com os arquivos informados.
func newVaultFixture(t *testing.T, files ...string) string {
	t.Helper()

	root := t.TempDir()
	for _, name := range files {
		full := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("criando diretorio do vault de teste: %v", err)
		}
		if err := os.WriteFile(full, []byte("material bruto\n"), 0644); err != nil {
			t.Fatalf("escrevendo %q no vault de teste: %v", full, err)
		}
	}
	return root
}

// TestDoctorUnindexedLinkComAlvoEmDiscoVaiParaCategoriaNova prova a AC1: o
// link cujo alvo existe em disco (mas fora do escopo indexado) NAO e link
// morto e nao derruba o Health Score.
func TestDoctorUnindexedLinkComAlvoEmDiscoVaiParaCategoriaNova(t *testing.T) {
	database := newDoctorFixture(t, [][3]string{{"nota.md", "material", "links_to"}})
	vault := newVaultFixture(t, "catalog-skills/skills/x/material.md")

	report, err := DiagnoseHealth(context.Background(), database, vault)
	if err != nil {
		t.Fatalf("DiagnoseHealth falhou: %v", err)
	}

	if len(report.DeadLinks) != 0 {
		t.Fatalf("dead links = %d; want 0 (o alvo existe em disco): %+v", len(report.DeadLinks), report.DeadLinks)
	}
	if len(report.OutOfIndexLinks) != 1 {
		t.Fatalf("links fora do indice = %d; want 1: %+v", len(report.OutOfIndexLinks), report.OutOfIndexLinks)
	}
	if report.OutOfIndexLinks[0].TargetID != "material" {
		t.Errorf("alvo esperado %q, obtido %q", "material", report.OutOfIndexLinks[0].TargetID)
	}
	if report.OutOfIndexLinks[0].SourceID != "nota.md" {
		t.Errorf("origem esperada %q, obtida %q", "nota.md", report.OutOfIndexLinks[0].SourceID)
	}
	if report.OutOfIndexLinks[0].Relation != "links_to" {
		t.Errorf("relacao esperada %q, obtida %q", "links_to", report.OutOfIndexLinks[0].Relation)
	}
	if report.OutOfIndexLinks[0].ResolvedPath != "catalog-skills/skills/x/material.md" {
		t.Errorf("caminho resolvido esperado %q, obtido %q", "catalog-skills/skills/x/material.md", report.OutOfIndexLinks[0].ResolvedPath)
	}
	if report.HealthScore != 100 {
		t.Errorf("health score = %d; want 100 (a categoria nova nao penaliza): %+v", report.HealthScore, report)
	}
}

// TestDoctorUnindexedLinkSemAlvoEmDiscoContinuaMorto prova a AC2: o link
// cujo alvo nao existe em disco e nem como documento indexado continua sendo
// link morto.
func TestDoctorUnindexedLinkSemAlvoEmDiscoContinuaMorto(t *testing.T) {
	database := newDoctorFixture(t, [][3]string{{"nota.md", "conceito", "links_to"}})
	vault := newVaultFixture(t, "catalog-skills/skills/x/material.md")

	report, err := DiagnoseHealth(context.Background(), database, vault)
	if err != nil {
		t.Fatalf("DiagnoseHealth falhou: %v", err)
	}

	if len(report.DeadLinks) != 1 {
		t.Fatalf("dead links = %d; want 1 (o alvo nao existe em disco): %+v", len(report.DeadLinks), report.DeadLinks)
	}
	if report.DeadLinks[0].TargetID != "conceito" {
		t.Errorf("alvo esperado %q, obtido %q", "conceito", report.DeadLinks[0].TargetID)
	}
	if len(report.OutOfIndexLinks) != 0 {
		t.Fatalf("links fora do indice = %d; want 0: %+v", len(report.OutOfIndexLinks), report.OutOfIndexLinks)
	}
}

// TestDoctorUnindexedCorrecaoApagaSoORompido prova a AC6: a correcao apaga
// exatamente a aresta que o relatorio lista como rompida e preserva a aresta
// cujo alvo existe em disco.
//
// Este e o teste que falha no codigo anterior: a correcao apagava pelo
// criterio do SQL e levava junto a aresta viva — o defeito que a ISSUE-038
// ja tinha ensinado e que aqui se repete por outra causa.
func TestDoctorUnindexedCorrecaoApagaSoORompido(t *testing.T) {
	database := newDoctorFixture(t, [][3]string{
		{"nota.md", "material", "links_to"},
		{"nota.md", "conceito", "links_to"},
	})
	vault := newVaultFixture(t, "catalog-skills/skills/x/material.md")
	ctx := context.Background()

	if _, err := FixHealthIssues(ctx, database, vault); err != nil {
		t.Fatalf("FixHealthIssues falhou: %v", err)
	}

	var vivas, mortas int
	if err := database.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM graph_edges WHERE target_id = 'material'
	`).Scan(&vivas); err != nil {
		t.Fatalf("contando a aresta viva: %v", err)
	}
	if err := database.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM graph_edges WHERE target_id = 'conceito'
	`).Scan(&mortas); err != nil {
		t.Fatalf("contando a aresta morta: %v", err)
	}

	if vivas != 1 {
		t.Errorf("aresta com alvo em disco depois da correcao = %d; want 1 (ela esta viva no vault)", vivas)
	}
	if mortas != 0 {
		t.Errorf("aresta rompida depois da correcao = %d; want 0 (ela foi listada como morta)", mortas)
	}
}

// TestDoctorUnindexedCorrecaoPreservaNaoVerificavel prova a AC7: sem raiz
// acessivel, o doctor nao tem prova de que o link esta quebrado, entao a
// correcao preserva a aresta.
func TestDoctorUnindexedCorrecaoPreservaNaoVerificavel(t *testing.T) {
	database := newDoctorFixture(t, [][3]string{{"nota.md", "conceito", "links_to"}})
	ctx := context.Background()

	// Raiz desconhecida: o doctor nao tem como verificar o alvo.
	if _, err := FixHealthIssues(ctx, database, ""); err != nil {
		t.Fatalf("FixHealthIssues falhou: %v", err)
	}

	var restantes int
	if err := database.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM graph_edges WHERE target_id = 'conceito'
	`).Scan(&restantes); err != nil {
		t.Fatalf("contando as arestas: %v", err)
	}
	if restantes != 1 {
		t.Errorf("arestas depois da correcao = %d; want 1 (sem prova, a aresta e preservada)", restantes)
	}
}

// TestDoctorUnindexedSemRaizFicaProvisorio prova a AC5: sem raiz legivel, os
// links ficam nao verificaveis, mantem a penalidade e marcam o score como
// provisorio.
func TestDoctorUnindexedSemRaizFicaProvisorio(t *testing.T) {
	database := newDoctorFixture(t, [][3]string{{"nota.md", "conceito", "links_to"}})

	report, err := DiagnoseHealth(context.Background(), database, "")
	if err != nil {
		t.Fatalf("DiagnoseHealth falhou: %v", err)
	}

	if len(report.UnverifiableLinks) != 1 {
		t.Fatalf("links nao verificaveis = %d; want 1: %+v", len(report.UnverifiableLinks), report.UnverifiableLinks)
	}
	if len(report.DeadLinks) != 0 {
		t.Fatalf("dead links = %d; want 0 (sem prova, nenhum link e declarado rompido): %+v", len(report.DeadLinks), report.DeadLinks)
	}
	if !report.HealthScoreProvisional {
		t.Error("o score deveria estar marcado como provisorio: ha link nao verificavel")
	}
	if report.HealthScore != 95 {
		t.Errorf("health score = %d; want 95 (a suspeita mantem a penalidade de 5 pontos)", report.HealthScore)
	}
}
