package store

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/pgtest"
)

// newPostgresDoctorFixture monta um banco descartavel com uma nota e as
// arestas informadas.
func newPostgresDoctorFixture(t *testing.T, repo string, edges [][3]string) (*PostgresStore, *sql.DB) {
	t.Helper()

	h := pgtest.New(t)
	s, err := NewPostgresStore(h.DSN)
	if err != nil {
		t.Fatalf("NewPostgresStore(%s): %v", h.DBName, err)
	}
	t.Cleanup(func() { s.Close() })
	database := s.DB()

	ctx := context.Background()
	if _, err := database.ExecContext(ctx, `
		INSERT INTO documents (id, repository, path, title, updated_at) VALUES ($1, $2, $3, $4, 0)
	`, "nota.md", repo, "nota.md", "Nota"); err != nil {
		t.Fatalf("insert do documento: %v", err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_nodes (id, repository, type, name) VALUES ($1, $2, $3, $4)
	`, "nota.md", repo, "note", "Nota"); err != nil {
		t.Fatalf("insert do no da nota: %v", err)
	}

	for _, edge := range edges {
		if _, err := database.ExecContext(ctx, `
			INSERT INTO graph_edges (source_id, target_id, repository, relation, epistemic_status) VALUES ($1, $2, $3, $4, 'EXTRACTED')
		`, edge[0], edge[1], repo, edge[2]); err != nil {
			t.Fatalf("insert da aresta %v: %v", edge, err)
		}
	}

	return s, database
}

// newPostgresVaultFixture monta um vault de disco com os arquivos informados.
func newPostgresVaultFixture(t *testing.T, files ...string) string {
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

// TestPostgresDoctorUnindexedLinkComAlvoEmDiscoVaiParaCategoriaNova prova a
// AC1 no engine de producao: o link cujo alvo existe em disco (mas fora do
// escopo indexado) nao e link morto e nao derruba o Health Score.
func TestPostgresDoctorUnindexedLinkComAlvoEmDiscoVaiParaCategoriaNova(t *testing.T) {
	const repo = "repo-unindexed"
	s, _ := newPostgresDoctorFixture(t, repo, [][3]string{{"nota.md", "material", "links_to"}})
	vault := newPostgresVaultFixture(t, "catalog-skills/skills/x/material.md")

	report, err := s.DiagnoseHealth(context.Background(), repo, vault)
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

// TestPostgresDoctorUnindexedLinkSemAlvoEmDiscoContinuaMorto prova a AC2 no
// engine de producao.
func TestPostgresDoctorUnindexedLinkSemAlvoEmDiscoContinuaMorto(t *testing.T) {
	const repo = "repo-unindexed"
	s, _ := newPostgresDoctorFixture(t, repo, [][3]string{{"nota.md", "conceito", "links_to"}})
	vault := newPostgresVaultFixture(t, "catalog-skills/skills/x/material.md")

	report, err := s.DiagnoseHealth(context.Background(), repo, vault)
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

// TestPostgresDoctorUnindexedCorrecaoApagaSoORompido prova a AC6 no engine
// de producao: a correcao apaga exatamente a aresta listada como rompida e
// preserva a aresta cujo alvo existe em disco.
func TestPostgresDoctorUnindexedCorrecaoApagaSoORompido(t *testing.T) {
	const repo = "repo-unindexed"
	s, database := newPostgresDoctorFixture(t, repo, [][3]string{
		{"nota.md", "material", "links_to"},
		{"nota.md", "conceito", "links_to"},
	})
	vault := newPostgresVaultFixture(t, "catalog-skills/skills/x/material.md")
	ctx := context.Background()

	if _, err := s.FixHealthIssues(ctx, repo, vault); err != nil {
		t.Fatalf("FixHealthIssues falhou: %v", err)
	}

	var vivas, mortas int
	if err := database.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM graph_edges WHERE repository = $1 AND target_id = 'material'
	`, repo).Scan(&vivas); err != nil {
		t.Fatalf("contando a aresta viva: %v", err)
	}
	if err := database.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM graph_edges WHERE repository = $1 AND target_id = 'conceito'
	`, repo).Scan(&mortas); err != nil {
		t.Fatalf("contando a aresta morta: %v", err)
	}

	if vivas != 1 {
		t.Errorf("aresta com alvo em disco depois da correcao = %d; want 1 (ela esta viva no vault)", vivas)
	}
	if mortas != 0 {
		t.Errorf("aresta rompida depois da correcao = %d; want 0 (ela foi listada como morta)", mortas)
	}
}

// TestPostgresDoctorUnindexedCorrecaoPreservaNaoVerificavel prova a AC7 no
// engine de producao: sem raiz acessivel, a correcao preserva a aresta.
func TestPostgresDoctorUnindexedCorrecaoPreservaNaoVerificavel(t *testing.T) {
	const repo = "repo-unindexed"
	s, database := newPostgresDoctorFixture(t, repo, [][3]string{{"nota.md", "conceito", "links_to"}})
	ctx := context.Background()

	if _, err := s.FixHealthIssues(ctx, repo, ""); err != nil {
		t.Fatalf("FixHealthIssues falhou: %v", err)
	}

	var restantes int
	if err := database.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM graph_edges WHERE repository = $1 AND target_id = 'conceito'
	`, repo).Scan(&restantes); err != nil {
		t.Fatalf("contando as arestas: %v", err)
	}
	if restantes != 1 {
		t.Errorf("arestas depois da correcao = %d; want 1 (sem prova, a aresta e preservada)", restantes)
	}
}

// TestPostgresDoctorUnindexedSemRaizFicaProvisorio prova a AC5 no engine de
// producao: sem raiz legivel, os links ficam nao verificaveis e o score fica
// provisorio.
func TestPostgresDoctorUnindexedSemRaizFicaProvisorio(t *testing.T) {
	const repo = "repo-unindexed"
	s, _ := newPostgresDoctorFixture(t, repo, [][3]string{{"nota.md", "conceito", "links_to"}})

	report, err := s.DiagnoseHealth(context.Background(), repo, "")
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

// TestPostgresDoctorUnindexedFiltraPorRepository prova que o filtro e por
// repository, como o resto do doctor: a aresta de outro repositorio nao
// entra na lista.
func TestPostgresDoctorUnindexedFiltraPorRepository(t *testing.T) {
	h := pgtest.New(t)
	s, err := NewPostgresStore(h.DSN)
	if err != nil {
		t.Fatalf("NewPostgresStore(%s): %v", h.DBName, err)
	}
	t.Cleanup(func() { s.Close() })
	database := s.DB()
	ctx := context.Background()

	for _, repo := range []string{"repo-auditado", "repo-outro"} {
		if _, err := database.ExecContext(ctx, `
			INSERT INTO documents (id, repository, path, title, updated_at) VALUES ($1, $2, $3, $4, 0)
		`, "nota.md", repo, "nota.md", "Nota"); err != nil {
			t.Fatalf("insert do documento: %v", err)
		}
		if _, err := database.ExecContext(ctx, `
			INSERT INTO graph_nodes (id, repository, type, name) VALUES ($1, $2, $3, $4)
		`, "nota.md", repo, "note", "Nota"); err != nil {
			t.Fatalf("insert do no da nota: %v", err)
		}
		if _, err := database.ExecContext(ctx, `
			INSERT INTO graph_edges (source_id, target_id, repository, relation, epistemic_status) VALUES ($1, $2, $3, $4, 'EXTRACTED')
		`, "nota.md", "conceito", repo, "links_to"); err != nil {
			t.Fatalf("insert da aresta: %v", err)
		}
	}

	vault := newPostgresVaultFixture(t)

	report, err := s.DiagnoseHealth(ctx, "repo-auditado", vault)
	if err != nil {
		t.Fatalf("DiagnoseHealth falhou: %v", err)
	}
	if len(report.DeadLinks) != 1 {
		t.Fatalf("dead links = %d; want 1 (so a aresta do repo auditado): %+v", len(report.DeadLinks), report.DeadLinks)
	}
}
