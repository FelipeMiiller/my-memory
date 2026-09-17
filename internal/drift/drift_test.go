package drift

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/db"
)

func TestAnalyzeDrift_EmptyDiff(t *testing.T) {
	ctx := context.Background()
	mock := &MockGitRunner{
		Commits: []GitCommit{},
		Changes: []FileChange{},
	}

	report, err := AnalyzeDrift(ctx, mock, nil, "repo", "vault", "HEAD~1..HEAD", 0.3, true)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if report.TotalCommits != 0 || report.TotalChanged != 0 {
		t.Errorf("esperava zero commits e changes, obteve %+v", report)
	}
	if len(report.DriftedNotes) != 0 || len(report.UncoveredCode) != 0 {
		t.Errorf("esperava zero drifted notes e uncovered code")
	}
}

func TestAnalyzeDrift_DirectMatchAndUncovered(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_drift.db")

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("falha ao inicializar db: %v", err)
	}
	defer database.Close()

	// Insere documento de memória que cita 'internal/deeplink/deeplink.go' 2x (ADR-039: MinMatchOccurrences=2)
	nowMilli := time.Now().Add(-48 * time.Hour).UnixMilli()
	_, err = database.ExecContext(ctx, `
		INSERT INTO documents (id, path, title, updated_at)
		VALUES ('docs/adr/030-deeplink.md', 'docs/adr/030-deeplink.md', 'ADR-030 Deep Linking', ?)
	`, nowMilli)
	if err != nil {
		t.Fatalf("falha ao inserir documento: %v", err)
	}

	_, err = database.ExecContext(ctx, `
		INSERT INTO chunks (id, document_id, chunk_index, content)
		VALUES ('c1', 'docs/adr/030-deeplink.md', 0, 'O motor implementado em internal/deeplink/deeplink.go gerencia os esquemas de deep link. Mudanças em internal/deeplink/deeplink.go devem refletir aqui.')
	`)
	if err != nil {
		t.Fatalf("falha ao inserir chunk: %v", err)
	}

	// Mock do Git: 1 arquivo correspondente à nota e 1 arquivo de código novo órfão
	mock := &MockGitRunner{
		Commits: []GitCommit{
			{
				Hash:      "abc1234",
				ShortHash: "abc1234",
				Author:    "Dev",
				Date:      time.Now().Add(-2 * time.Hour),
				Message:   "feat: refactor deep link engine and add analytics",
			},
		},
		Changes: []FileChange{
			{
				Path:       "internal/deeplink/deeplink.go",
				Status:     "M",
				Additions:  120,
				Deletions:  15,
				IsCodeFile: true,
			},
			{
				Path:       "pkg/analytics/tracker.go",
				Status:     "A",
				Additions:  250,
				Deletions:  0,
				IsCodeFile: true,
			},
			{
				Path:       "README.md",
				Status:     "M",
				Additions:  5,
				Deletions:  2,
				IsCodeFile: false, // Documentação não deve ir para UncoveredCode
			},
		},
	}

	report, err := AnalyzeDrift(ctx, mock, database, tmpDir, "MyVault", "HEAD~1..HEAD", 0.1, true)
	if err != nil {
		t.Fatalf("erro ao analisar drift: %v", err)
	}

	if report.TotalCommits != 1 {
		t.Errorf("TotalCommits = %d; esperava 1", report.TotalCommits)
	}

	// 1. Verifica se a nota ADR-030 foi detectada com desvio
	if len(report.DriftedNotes) != 1 {
		t.Fatalf("esperava 1 nota com desvio, obteve %d", len(report.DriftedNotes))
	}
	drifted := report.DriftedNotes[0]
	if drifted.NoteID != "docs/adr/030-deeplink.md" {
		t.Errorf("NoteID = %q; esperava 'docs/adr/030-deeplink.md'", drifted.NoteID)
	}
	if drifted.DriftScore <= 0 {
		t.Errorf("DriftScore deveria ser positivo, obteve %.2f", drifted.DriftScore)
	}
	if len(drifted.AffectedFiles) != 1 || drifted.AffectedFiles[0] != "internal/deeplink/deeplink.go" {
		t.Errorf("AffectedFiles incorreto: %v", drifted.AffectedFiles)
	}
	if drifted.Links == nil || drifted.Links.Obsidian == "" {
		t.Errorf("Links deveriam estar populados")
	}

	// 2. Verifica se tracker.go foi detectado como código órfão
	if len(report.UncoveredCode) != 1 {
		t.Fatalf("esperava 1 código órfão, obteve %d", len(report.UncoveredCode))
	}
	uncovered := report.UncoveredCode[0]
	if uncovered.FilePath != "pkg/analytics/tracker.go" {
		t.Errorf("FilePath = %q; esperava 'pkg/analytics/tracker.go'", uncovered.FilePath)
	}
	if uncovered.Status != "A" {
		t.Errorf("Status = %q; esperava 'A'", uncovered.Status)
	}
}

func TestAnalyzeDrift_ThresholdFiltering(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_drift_thresh.db")

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("falha ao inicializar db: %v", err)
	}
	defer database.Close()

	_, _ = database.ExecContext(ctx, `
		INSERT INTO documents (id, path, title, updated_at)
		VALUES ('docs/adr/001.md', 'docs/adr/001.md', 'ADR 1', 1000)
	`)
	_, _ = database.ExecContext(ctx, `
		INSERT INTO chunks (id, document_id, chunk_index, content)
		VALUES ('c1', 'docs/adr/001.md', 0, 'utiliza pkg/util.go e toca em pkg/util.go brevemente')
	`)

	mock := &MockGitRunner{
		Commits: []GitCommit{
			{Hash: "h1", ShortHash: "h1", Author: "Dev", Date: time.Now(), Message: "minor tweak"},
		},
		Changes: []FileChange{
			{Path: "pkg/util.go", Status: "M", Additions: 1, Deletions: 0, IsCodeFile: true},
		},
	}

	// Threshold de 99% (0.99) deve filtrar desvios leves
	report, err := AnalyzeDrift(ctx, mock, database, tmpDir, "Vault", "HEAD~1..HEAD", 0.99, false)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if len(report.DriftedNotes) != 0 {
		t.Errorf("esperava zero notas devido ao threshold alto, obteve %d", len(report.DriftedNotes))
	}
}

// TestAnalyzeDrift_FiltersSingleMention valida ADR-039: nota que menciona
// o path alterado APENAS UMA VEZ deve ser filtrada (MinMatchOccurrences=2).
// Caso comum: validation.md e tasks.md que LISTAM todos os arquivos do projeto
// sem correlação semântica real.
func TestAnalyzeDrift_FiltersSingleMention(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_drift_single.db")

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("falha ao inicializar db: %v", err)
	}
	defer database.Close()

	// Nota com apenas 1 menção ao path - deve ser filtrada
	_, _ = database.ExecContext(ctx, `
		INSERT INTO documents (id, path, title, updated_at)
		VALUES ('docs/lista-generica.md', 'docs/lista-generica.md', 'Lista Genérica', 1000)
	`)
	_, _ = database.ExecContext(ctx, `
		INSERT INTO chunks (id, document_id, chunk_index, content)
		VALUES ('c1', 'docs/lista-generica.md', 0, 'Componentes do projeto: internal/foo/foo.go, internal/bar/bar.go.')
	`)

	// Nota com 2 menções - deve passar
	_, _ = database.ExecContext(ctx, `
		INSERT INTO documents (id, path, title, updated_at)
		VALUES ('docs/especifico.md', 'docs/especifico.md', 'Específico', 1000)
	`)
	_, _ = database.ExecContext(ctx, `
		INSERT INTO chunks (id, document_id, chunk_index, content)
		VALUES ('c2', 'docs/especifico.md', 0, 'O arquivo internal/foo/foo.go é o coração do módulo. Mudanças em internal/foo/foo.go afetam X.')
	`)

	mock := &MockGitRunner{
		Commits: []GitCommit{
			{Hash: "h1", ShortHash: "h1", Author: "Dev", Date: time.Now(), Message: "refactor foo"},
		},
		Changes: []FileChange{
			{Path: "internal/foo/foo.go", Status: "M", Additions: 50, Deletions: 10, IsCodeFile: true},
		},
	}

	report, err := AnalyzeDrift(ctx, mock, database, tmpDir, "Vault", "HEAD~1..HEAD", 0.1, false)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if len(report.DriftedNotes) != 1 {
		t.Fatalf("esperava 1 nota com desvio (específico), obteve %d", len(report.DriftedNotes))
	}
	if report.DriftedNotes[0].NoteID != "docs/especifico.md" {
		t.Errorf("NoteID = %q; esperava 'docs/especifico.md' (lista-generica deve ter sido filtrada)", report.DriftedNotes[0].NoteID)
	}
}

// TestAnalyzeDrift_ScoreClamping valida ADR-039: cap dos fatores para evitar
// saturação. Cenário extremo: PR alto + muitos commits + muitas linhas.
func TestAnalyzeDrift_ScoreClamping(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_drift_clamp.db")

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("falha ao inicializar db: %v", err)
	}
	defer database.Close()

	// Nota com PageRank alto (simulado via inserção repetida? PageRank vem do grafo, então usamos o default 0.015)
	// Workaround: a fórmula usa nd.PageRank que vem do docMap. Aqui testamos só os caps relativos.
	_, _ = database.ExecContext(ctx, `
		INSERT INTO documents (id, path, title, updated_at)
		VALUES ('docs/high-pr.md', 'docs/high-pr.md', 'High PR', 1000)
	`)
	_, _ = database.ExecContext(ctx, `
		INSERT INTO chunks (id, document_id, chunk_index, content)
		VALUES ('c1', 'docs/high-pr.md', 0, 'módulo internal/big/big.go é crítico. mudanças em internal/big/big.go impactam muito')
	`)

	mock := &MockGitRunner{
		Commits: []GitCommit{
			{Hash: "h1", ShortHash: "h1", Author: "Dev", Date: time.Now(), Message: "1"},
			{Hash: "h2", ShortHash: "h2", Author: "Dev", Date: time.Now(), Message: "2"},
			{Hash: "h3", ShortHash: "h3", Author: "Dev", Date: time.Now(), Message: "3"},
			{Hash: "h4", ShortHash: "h4", Author: "Dev", Date: time.Now(), Message: "4"},
			{Hash: "h5", ShortHash: "h5", Author: "Dev", Date: time.Now(), Message: "5"},
		},
		Changes: []FileChange{
			{Path: "internal/big/big.go", Status: "M", Additions: 1000, Deletions: 500, IsCodeFile: true},
		},
	}

	report, err := AnalyzeDrift(ctx, mock, database, tmpDir, "Vault", "HEAD~5..HEAD", 0.0, false)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if len(report.DriftedNotes) != 1 {
		t.Fatalf("esperava 1 nota com desvio, obteve %d", len(report.DriftedNotes))
	}
	drifted := report.DriftedNotes[0]
	// PageRank default = 0.015, então prFactor = 0.015 * 30 = 0.45 (não satura)
	// commitFactor = min(40, 60) = 40
	// linesFactor = min(25, ln(1501)*5) ≈ min(25, 36) = 25
	// score ≈ 40 + 0.45 + 25 = 65.45 → HIGH (entre 55 e 75)
	if drifted.DriftScore >= 100.0 {
		t.Errorf("score saturado em 100 (%.2f) — fórmula ainda não calibrada", drifted.DriftScore)
	}
	if drifted.DriftScore >= 75.0 {
		t.Errorf("score muito alto para PageRank default (%.2f) — deveria ser HIGH, não CRITICAL", drifted.DriftScore)
	}
}
