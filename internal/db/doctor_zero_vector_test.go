package db

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/embedder"
	"github.com/FelipeMiiller/my-memory/internal/turboquant"
)

// TestDoctorZeroVectorSQLite prova a AC2 no SQLite: o doctor reporta como
// dessincronizado o chunk que não tem vetor real — tanto o AUSENTE (sem
// linha em `chunks_vec` nem em `chunks_turboquant`) quanto o ZERO (as duas
// linhas existem, mas com o vetor zero).
//
// O SQLite tem DOIS caminhos de vetor, e a busca vetorial lê um ou outro
// conforme a plataforma: o cru (`chunks_vec`) quando há sqlite-vec e o
// comprimido (`chunks_turboquant`) no Windows, onde `HasSqliteVec` é false.
// Por isso o doctor olha os dois.
func TestDoctorZeroVectorSQLite(t *testing.T) {
	database, err := InitDB(filepath.Join(t.TempDir(), "doctor_zero_vector.db"))
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	tq := turboquant.NewQuantizer(embedder.EmbeddingDim)

	docID := "doc-1"
	if err := InsertDocumentWithMeta(ctx, database, docID, "notas/doc.md", "Doc", 1700000000, "hash-1", "epoch", "", "resource", 1700000000, 128); err != nil {
		t.Fatalf("InsertDocumentWithMeta falhou: %v", err)
	}

	// 1. Chunk SEM vetor: nenhuma linha em `chunks_vec` nem em
	// `chunks_turboquant` — o que o indexador grava sem provider.
	if err := InsertChunk(ctx, database, docID+"#0", docID, "conteudo sem vetor", 0, nil); err != nil {
		t.Fatalf("InsertChunk sem vetor falhou: %v", err)
	}

	// 2. Chunk com VETOR ZERO: as duas linhas existem, com o vetor zero.
	zero := make([]float32, embedder.EmbeddingDim)
	if err := InsertChunk(ctx, database, docID+"#1", docID, "conteudo com vetor zero", 1, zero); err != nil {
		t.Fatalf("InsertChunk com vetor zero falhou: %v", err)
	}
	cvZero, err := tq.Quantize(zero)
	if err != nil {
		t.Fatalf("Quantize do vetor zero falhou: %v", err)
	}
	if err := InsertTurboQuantChunk(ctx, database, docID+"#1", cvZero.Scale, cvZero.Data); err != nil {
		t.Fatalf("InsertTurboQuantChunk do vetor zero falhou: %v", err)
	}

	// 3. Chunk com vetor REAL: não é dessincronizado.
	real := make([]float32, embedder.EmbeddingDim)
	real[0] = 1.0
	real[1] = 0.5
	if err := InsertChunk(ctx, database, docID+"#2", docID, "conteudo com vetor real", 2, real); err != nil {
		t.Fatalf("InsertChunk com vetor real falhou: %v", err)
	}
	cvReal, err := tq.Quantize(real)
	if err != nil {
		t.Fatalf("Quantize do vetor real falhou: %v", err)
	}
	if err := InsertTurboQuantChunk(ctx, database, docID+"#2", cvReal.Scale, cvReal.Data); err != nil {
		t.Fatalf("InsertTurboQuantChunk do vetor real falhou: %v", err)
	}

	// 4. Chunk com vetor cru e SEM o comprimido: a busca no caminho
	// comprimido (o do Windows) não o encontraria.
	if err := InsertChunk(ctx, database, docID+"#3", docID, "conteudo so com vetor cru", 3, real); err != nil {
		t.Fatalf("InsertChunk com vetor cru falhou: %v", err)
	}

	// 5. Chunk com vetor cru ZERO e sem o comprimido: isola o caminho cru.
	if err := InsertChunk(ctx, database, docID+"#4", docID, "conteudo com vetor cru zero", 4, zero); err != nil {
		t.Fatalf("InsertChunk com vetor cru zero falhou: %v", err)
	}

	// 6. Chunk com vetor COMPRIMIDO zero e sem o cru: isola o caminho
	// comprimido — que é o que a busca vetorial lê no Windows.
	if err := InsertChunk(ctx, database, docID+"#5", docID, "conteudo so com vetor comprimido zero", 5, nil); err != nil {
		t.Fatalf("InsertChunk sem vetor falhou: %v", err)
	}
	if err := InsertTurboQuantChunk(ctx, database, docID+"#5", cvZero.Scale, cvZero.Data); err != nil {
		t.Fatalf("InsertTurboQuantChunk do vetor comprimido zero falhou: %v", err)
	}

	report, err := DiagnoseHealth(ctx, database, t.TempDir())
	if err != nil {
		t.Fatalf("DiagnoseHealth falhou: %v", err)
	}

	porChunk := map[string]string{}
	for _, dc := range report.DesyncedChunks {
		porChunk[dc.ChunkID] = dc.Issue
	}

	if len(report.DesyncedChunks) != 5 {
		t.Errorf("o doctor reportou %d chunks dessincronizados, esperado 5 (o ausente, o de vetor zero, o sem comprimido, o cru zero e o comprimido zero): %v",
			len(report.DesyncedChunks), report.DesyncedChunks)
	}

	if issue, ok := porChunk[docID+"#0"]; !ok {
		t.Error("o chunk sem vetor nenhum não foi reportado como dessincronizado")
	} else if issue != "Missing vector embedding" {
		t.Errorf("o chunk sem vetor foi reportado como %q, esperado %q", issue, "Missing vector embedding")
	}

	if issue, ok := porChunk[docID+"#1"]; !ok {
		t.Error("o chunk com vetor ZERO não foi reportado como dessincronizado: as linhas existem, então 'existe' não denuncia nada")
	} else if issue != "Zero vector embedding" {
		t.Errorf("o chunk com vetor zero foi reportado como %q, esperado %q", issue, "Zero vector embedding")
	}

	if _, ok := porChunk[docID+"#2"]; ok {
		t.Error("o chunk com vetor real foi reportado como dessincronizado: ele tem similaridade de verdade")
	}

	if issue, ok := porChunk[docID+"#3"]; !ok {
		t.Error("o chunk com vetor cru e sem o comprimido não foi reportado como dessincronizado")
	} else if issue != "Missing TurboQuant 4-bit compression" {
		t.Errorf("o chunk sem o vetor comprimido foi reportado como %q, esperado %q", issue, "Missing TurboQuant 4-bit compression")
	}

	// O caminho cru e o comprimido são detectados de forma independente:
	// cada um destes chunks tem vetor zero em UM dos caminhos só.
	if issue, ok := porChunk[docID+"#4"]; !ok {
		t.Error("o chunk com vetor cru ZERO não foi reportado como dessincronizado")
	} else if issue != "Zero vector embedding" {
		t.Errorf("o chunk com vetor cru zero foi reportado como %q, esperado %q", issue, "Zero vector embedding")
	}

	if issue, ok := porChunk[docID+"#5"]; !ok {
		t.Error("o chunk com vetor COMPRIMIDO zero não foi reportado como dessincronizado")
	} else if issue != "Zero vector embedding" {
		t.Errorf("o chunk com vetor comprimido zero foi reportado como %q, esperado %q", issue, "Zero vector embedding")
	}
}
