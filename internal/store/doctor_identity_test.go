package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/pgtest"
)

// TestPostgresDoctorLegacyNodes prova o contrato de higiene do grafo para os
// nós legados de identidade no engine de produção (ADR-066 — GNI-13, GNI-14 e
// GNI-16):
//
//  1. o relatório expõe a contagem de nós que não são `documents.id` nem
//     `code_symbol`;
//  2. o `--fix` remove exatamente esses nós SEM GRAU, escopados por
//     repositório, e nenhum outro;
//  3. um nó de código (`code_symbol`) nunca é removido;
//  4. uma aresta cujo destino é um documento indexado não é dead link.
func TestPostgresDoctorLegacyNodes(t *testing.T) {
	h := pgtest.New(t)
	ctx := context.Background()

	s, err := NewPostgresStore(h.DSN)
	if err != nil {
		t.Fatalf("NewPostgresStore(%s): %v", h.DBName, err)
	}
	defer s.Close()
	database := s.DB()

	const repo = "repo-doctor-legacy"
	const outroRepo = "repo-doctor-legacy-2"
	now := time.Now().Unix()

	// Documentos indexados (o nó deles tem id = documents.id).
	if err := s.InsertDocumentWithMeta(ctx, repo, "nota.md", "nota.md", "Minha Nota", now, "h1", "", "resumo", "resource", 0, 0); err != nil {
		t.Fatalf("inserindo o documento: %v", err)
	}
	if err := s.InsertDocumentWithMeta(ctx, repo, "outra.md", "outra.md", "Outra Nota", now, "h2", "", "resumo", "resource", 0, 0); err != nil {
		t.Fatalf("inserindo o segundo documento: %v", err)
	}

	// Nó legado de OUTRO repositório: a correção é escopada e não toca nele.
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_nodes (id, repository, type, name) VALUES ($1, $2, 'note', $3)
	`, "Minha Nota", outroRepo, "Minha Nota"); err != nil {
		t.Fatalf("inserindo o nó legado de outro repositório: %v", err)
	}

	// Nó de estilo título SEM GRAU: o resto do fenda — o `--fix` remove.
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_nodes (id, repository, type, name) VALUES ($1, $2, 'note', $3)
	`, "Minha Nota", repo, "Minha Nota"); err != nil {
		t.Fatalf("inserindo o nó legado sem grau: %v", err)
	}

	// Nó de estilo título COM GRAU: é a representação viva de um link que
	// existe em disco mas está fora do índice — o `--fix` preserva.
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_nodes (id, repository, type, name) VALUES ($1, $2, 'note', $3)
	`, "Alvo Vivo", repo, "Alvo Vivo"); err != nil {
		t.Fatalf("inserindo o nó legado com grau: %v", err)
	}
	if err := s.InsertEdgeWithProps(ctx, repo, "nota.md", "Alvo Vivo", "links_to", "EXTRACTED", 1.0); err != nil {
		t.Fatalf("inserindo a aresta viva: %v", err)
	}

	// Nó de código: nunca é removido (ADR-061).
	if _, err := database.ExecContext(ctx, `
		INSERT INTO graph_nodes (id, repository, type, name) VALUES ($1, $2, 'code_symbol', $3)
	`, "code:repo:pkg.Alpha", repo, "Alpha"); err != nil {
		t.Fatalf("inserindo o nó de código: %v", err)
	}

	// Aresta entre dois documentos indexados: as duas pontas em id de
	// documento — o doctor NÃO pode acusar isso como link rompido.
	if err := s.InsertEdgeWithProps(ctx, repo, "nota.md", "outra.md", "links_to", "EXTRACTED", 1.0); err != nil {
		t.Fatalf("inserindo a aresta entre documentos: %v", err)
	}

	// O alvo "Alvo Vivo" existe em disco (fora do índice por decisão).
	vaultRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(vaultRoot, "Alvo Vivo.md"), []byte("# Alvo Vivo\n"), 0644); err != nil {
		t.Fatalf("escrevendo o alvo em disco: %v", err)
	}

	// 1. O relatório expõe a contagem de nós legados.
	report, err := s.DiagnoseHealth(ctx, repo, vaultRoot)
	if err != nil {
		t.Fatalf("DiagnoseHealth falhou: %v", err)
	}
	if report.LegacyNodeCount != 2 {
		t.Errorf("nós legados = %d; want 2 (o de estilo título sem grau e o com grau)", report.LegacyNodeCount)
	}
	if report.OrphanLegacyNodeCount != 1 {
		t.Errorf("nós legados sem grau = %d; want 1", report.OrphanLegacyNodeCount)
	}

	// 4. A aresta entre documentos indexados não é dead link.
	for _, dl := range report.DeadLinks {
		if dl.TargetID == "outra.md" {
			t.Errorf("a aresta para o documento indexado %q foi accusationada como dead link: %+v", dl.TargetID, dl)
		}
	}

	// 2 e 3. O `--fix` remove só o nó legado sem grau.
	if _, err := s.FixHealthIssues(ctx, repo, vaultRoot); err != nil {
		t.Fatalf("FixHealthIssues falhou: %v", err)
	}

	var semGrau int
	if err := database.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM graph_nodes WHERE repository = $1 AND id = $2
	`, repo, "Minha Nota").Scan(&semGrau); err != nil {
		t.Fatalf("contando o nó legado sem grau: %v", err)
	}
	if semGrau != 0 {
		t.Errorf("nó legado sem grau = %d; want 0 (o --fix remove)", semGrau)
	}

	var restantes int
	if err := database.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM graph_nodes
		WHERE repository = $1 AND id IN ('Alvo Vivo', 'code:repo:pkg.Alpha', 'nota.md')
	`, repo).Scan(&restantes); err != nil {
		t.Fatalf("contando os nós restantes: %v", err)
	}
	if restantes != 3 {
		t.Errorf("nós que sobraram = %d; want 3 (o nó de código, o nó com grau e o nó do documento)", restantes)
	}

	// O nó legado de OUTRO repositório não foi tocado.
	var outroRepoInt int
	if err := database.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM graph_nodes WHERE repository = $1 AND id = $2
	`, outroRepo, "Minha Nota").Scan(&outroRepoInt); err != nil {
		t.Fatalf("contando o nó legado de outro repositório: %v", err)
	}
	if outroRepoInt != 1 {
		t.Errorf("nó legado de outro repositório = %d; want 1 (a correção é escopada)", outroRepoInt)
	}
}
