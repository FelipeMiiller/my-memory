package store

import (
	"context"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/embedder"
	"github.com/FelipeMiiller/my-memory/internal/pgtest"
)

// TestDoctorZeroVectorPostgres prova a AC2 no Postgres: o doctor reporta
// como dessincronizado o chunk que não tem vetor real — tanto o AUSENTE
// (embedding NULL) quanto o ZERO (a coluna preenchida com o vetor zero).
//
// O vetor zero é o defeito da ISSUE-040: ele "existe" na coluna, então o
// filtro antigo (`embedding IS NULL`) não o via, e o vault sem um único
// vetor real passava como saudável.
//
// O banco é descartável (pgtest) — o Postgres do dono nunca é alvo.
func TestDoctorZeroVectorPostgres(t *testing.T) {
	h := pgtest.New(t)

	ctx := context.Background()
	s, err := NewPostgresStore(h.DSN)
	if err != nil {
		t.Fatalf("conectar no PostgreSQL descartável (%s): %v", h.DBName, err)
	}
	defer s.Close()

	repo := "repo-doctor-zero-vector"
	docID := "doc-1"
	if err := s.InsertDocument(ctx, repo, docID, "notas/doc.md", "Doc", 1700000000, "hash-1"); err != nil {
		t.Fatalf("InsertDocument falhou: %v", err)
	}

	// 1. Chunk SEM vetor: o embedding é NULL.
	if err := s.InsertChunk(ctx, repo, "doc-1#0", docID, "conteudo sem vetor", 0, nil); err != nil {
		t.Fatalf("InsertChunk sem vetor falhou: %v", err)
	}

	// 2. Chunk com VETOR ZERO: a coluna é preenchida, como o indexador antigo
	// fazia quando não havia provider.
	if err := s.InsertChunk(ctx, repo, "doc-1#1", docID, "conteudo com vetor zero", 1, make([]float32, embedder.EmbeddingDim)); err != nil {
		t.Fatalf("InsertChunk com vetor zero falhou: %v", err)
	}

	// 3. Chunk com vetor REAL: não é dessincronizado.
	real := make([]float32, embedder.EmbeddingDim)
	real[0] = 1.0
	if err := s.InsertChunk(ctx, repo, "doc-1#2", docID, "conteudo com vetor real", 2, real); err != nil {
		t.Fatalf("InsertChunk com vetor real falhou: %v", err)
	}

	report, err := s.DiagnoseHealth(ctx, repo, t.TempDir())
	if err != nil {
		t.Fatalf("DiagnoseHealth falhou: %v", err)
	}

	porChunk := map[string]string{}
	for _, dc := range report.DesyncedChunks {
		porChunk[dc.ChunkID] = dc.Issue
	}

	if len(report.DesyncedChunks) != 2 {
		t.Errorf("o doctor reportou %d chunks dessincronizados, esperado 2 (o ausente e o de vetor zero): %v",
			len(report.DesyncedChunks), report.DesyncedChunks)
	}

	if issue, ok := porChunk["doc-1#0"]; !ok {
		t.Error("o chunk sem vetor (embedding NULL) não foi reportado como dessincronizado")
	} else if issue != "Missing vector embedding" {
		t.Errorf("o chunk sem vetor foi reportado como %q, esperado %q", issue, "Missing vector embedding")
	}

	if issue, ok := porChunk["doc-1#1"]; !ok {
		t.Error("o chunk com vetor ZERO não foi reportado como dessincronizado: o filtro antigo (embedding IS NULL) não o enxerga")
	} else if issue != "Zero vector embedding" {
		t.Errorf("o chunk com vetor zero foi reportado como %q, esperado %q", issue, "Zero vector embedding")
	}

	if _, ok := porChunk["doc-1#2"]; ok {
		t.Error("o chunk com vetor real foi reportado como dessincronizado: ele tem similaridade de verdade")
	}

	// AC3: o Health Score penaliza o chunk sem vetor real em 2 pontos (ADR-013).
	// A comparação isola a penalidade: o mesmo relatório sem os chunks
	// dessincronizados vale 4 pontos a mais (2 chunks x 2 pontos).
	semDesinc := *report
	semDesinc.DesyncedChunks = nil
	esperado := CalculateHealthScore(semDesinc) - 4
	if report.HealthScore != esperado {
		t.Errorf("o Health Score é %d, esperado %d (2 pontos por chunk sem vetor real, ADR-013)", report.HealthScore, esperado)
	}
}
