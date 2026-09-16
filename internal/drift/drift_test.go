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

	// Insere documento de memória que cita 'internal/deeplink/deeplink.go'
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
		VALUES ('c1', 'docs/adr/030-deeplink.md', 0, 'O motor implementado em internal/deeplink/deeplink.go gerencia os esquemas de deep link.')
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
		VALUES ('c1', 'docs/adr/001.md', 0, 'utiliza pkg/util.go')
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
