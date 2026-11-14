package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/embedder"
	"github.com/FelipeMiiller/my-memory/internal/store"
	"github.com/FelipeMiiller/my-memory/internal/turboquant"
)

// TestKNNZeroVectorSQLite prova a AC4 no SQLite: a busca vetorial não
// devolve chunk sem vetor real — nem o vetor zero, nem o chunk sem linha.
//
// O vetor zero (o defeito da ISSUE-040) PREENCHE a coluna, então ele sai do
// KNN como qualquer outro — e a busca devolvia ruído com cara de resultado.
//
// O SQLite tem dois caminhos de vetor, e a busca lê um ou outro conforme a
// plataforma: o cru (`chunks_vec`, via sqlite-vec) e o comprimido
// (`chunks_turboquant`, o do Windows, onde `HasSqliteVec` é false). Os dois
// são exercitados aqui — o cru só quando a plataforma tem sqlite-vec.
func TestKNNZeroVectorSQLite(t *testing.T) {
	database, err := InitDB(filepath.Join(t.TempDir(), "knn_zero_vector.db"))
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

	// A consulta: um vetor real, parecido com o do chunk "real".
	queryVec := make([]float32, embedder.EmbeddingDim)
	queryVec[0] = 1.0
	queryVec[1] = 1.0

	zero := make([]float32, embedder.EmbeddingDim)
	real := make([]float32, embedder.EmbeddingDim)
	real[0] = 1.0
	real[1] = 1.0

	// chunk 0: vetor ZERO (o defeito) — com as duas linhas.
	insereChunkComVetor(t, database, docID+"#0", docID, "conteudo da arquitetura", 0, zero, tq)
	// chunk 1: vetor REAL.
	insereChunkComVetor(t, database, docID+"#1", docID, "conteudo da escalabilidade", 1, real, tq)
	// chunk 2: sem vetor nenhum (nem linha crua, nem comprimida).
	if err := InsertChunk(ctx, database, docID+"#2", docID, "conteudo da observabilidade", 2, nil); err != nil {
		t.Fatalf("InsertChunk sem vetor falhou: %v", err)
	}

	opts := store.SearchOptions{Level: "l1"}

	// 1. O caminho COMPRIMIDO (o do Windows): o chunk com vetor zero não sai.
	comprimidos, err := SearchTurboQuantWithOptions(ctx, database, tq, queryVec, 10, opts)
	if err != nil {
		t.Fatalf("SearchTurboQuantWithOptions falhou: %v", err)
	}
	ids := idsDosChunks(comprimidos)
	if _, ok := ids[docID+"#0"]; ok {
		t.Error("a busca vetorial devolveu o chunk com vetor ZERO (caminho comprimido): ele não tem similaridade de verdade")
	}
	if _, ok := ids[docID+"#1"]; !ok {
		t.Error("a busca vetorial NÃO devolveu o chunk com vetor real (caminho comprimido)")
	}
	if _, ok := ids[docID+"#2"]; ok {
		t.Error("a busca vetorial devolveu o chunk sem vetor nenhum (caminho comprimido)")
	}

	// 2. O caminho CRU (sqlite-vec): só existe onde a extensão está
	// disponível — na CI (ubuntu, -tags sqlite_fts5).
	if HasSqliteVec {
		cru, err := SearchKNNWithOptions(ctx, database, queryVec, 10, opts)
		if err != nil {
			t.Fatalf("SearchKNNWithOptions falhou: %v", err)
		}
		idsCru := idsDosChunks(cru)
		if _, ok := idsCru[docID+"#0"]; ok {
			t.Error("a busca vetorial devolveu o chunk com vetor ZERO (caminho cru): ele não tem similaridade de verdade")
		}
		if _, ok := idsCru[docID+"#1"]; !ok {
			t.Error("a busca vetorial NÃO devolveu o chunk com vetor real (caminho cru)")
		}
	}

	// 3. A busca com TODOS os chunks sem vetor devolve lista vazia — e não
	// erro. É o caso do vault indexado sem provider: ele tem chunk, mas
	// nenhum deles tem vetor.
	vazio, err := InitDB(filepath.Join(t.TempDir(), "knn_todo_zero.db"))
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer vazio.Close()
	if err := InsertDocumentWithMeta(ctx, vazio, docID, "notas/doc.md", "Doc", 1700000000, "hash-1", "epoch", "", "resource", 1700000000, 128); err != nil {
		t.Fatalf("InsertDocumentWithMeta falhou: %v", err)
	}
	insereChunkComVetor(t, vazio, docID+"#0", docID, "conteudo so com vetor zero", 0, zero, tq)

	somenteZeros, err := SearchTurboQuantWithOptions(ctx, vazio, tq, queryVec, 10, opts)
	if err != nil {
		t.Fatalf("a busca com todos os chunks sem vetor deu erro em vez de lista vazia: %v", err)
	}
	if len(somenteZeros) != 0 {
		t.Errorf("a busca com todos os chunks sem vetor devolveu %d resultado(s), esperado lista vazia", len(somenteZeros))
	}

	if HasSqliteVec {
		somenteZerosCru, err := SearchKNNWithOptions(ctx, vazio, queryVec, 10, opts)
		if err != nil {
			t.Fatalf("a busca (caminho cru) com todos os chunks sem vetor deu erro em vez de lista vazia: %v", err)
		}
		if len(somenteZerosCru) != 0 {
			t.Errorf("a busca (caminho cru) com todos os chunks sem vetor devolveu %d resultado(s), esperado lista vazia", len(somenteZerosCru))
		}
	}
}

// TestHybridZeroVectorSoAlcancadoPelaFonteLexica prova a AC5: o chunk sem
// vetor real continua alcançável pelo cofre — mas APENAS pela fonte léxica.
//
// Ele não pode sumir da busca (o conteúdo dele é real), e não pode aparecer
// como se tivesse vindo do vetor (a fonte vetorial é a que mente: devolve
// chunk sem similaridade como se fosse relevante).
func TestHybridZeroVectorSoAlcancadoPelaFonteLexica(t *testing.T) {
	database, err := InitDB(filepath.Join(t.TempDir(), "hybrid_zero_vector.db"))
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

	zero := make([]float32, embedder.EmbeddingDim)
	insereChunkComVetor(t, database, docID+"#0", docID, "arquitetura do cofre central", 0, zero, tq)

	queryVec := make([]float32, embedder.EmbeddingDim)
	queryVec[0] = 1.0

	// A consulta lexical casa com o chunk; a consulta vetorial também
	// poderia, se o vetor zero fosse aceito.
	resultados, err := SearchHybridRRF(ctx, database, tq, "arquitetura", queryVec, 10, 60, false)
	if err != nil {
		t.Fatalf("SearchHybridRRF falhou: %v", err)
	}

	var achado *SearchResult
	for i := range resultados {
		if resultados[i].ChunkID == docID+"#0" {
			achado = &resultados[i]
			break
		}
	}
	if achado == nil {
		t.Fatal("o chunk sem vetor real sumiu da busca híbrida: ele é alcançável pelo conteúdo, e a fonte léxica o traz")
	}

	for _, fonte := range achado.Sources {
		if strings.HasPrefix(fonte, "vector") {
			t.Errorf("o chunk sem vetor real apareceu na fonte %q da busca híbrida: ele não tem similaridade de verdade", fonte)
		}
	}
}

// insereChunkComVetor grava um chunk com o vetor cru e o comprimido — o
// que o indexador faz quando há provider.
func insereChunkComVetor(t *testing.T, database *sql.DB, chunkID, docID, content string, index int, vetor []float32, tq *turboquant.Quantizer) {
	t.Helper()

	if err := InsertChunk(context.Background(), database, chunkID, docID, content, index, vetor); err != nil {
		t.Fatalf("InsertChunk(%s) falhou: %v", chunkID, err)
	}
	cv, err := tq.Quantize(vetor)
	if err != nil {
		t.Fatalf("Quantize(%s) falhou: %v", chunkID, err)
	}
	if err := InsertTurboQuantChunk(context.Background(), database, chunkID, cv.Scale, cv.Data); err != nil {
		t.Fatalf("InsertTurboQuantChunk(%s) falhou: %v", chunkID, err)
	}
}

// idsDosChunks devolve o conjunto de ids presentes no resultado.
func idsDosChunks(resultados []SearchResult) map[string]bool {
	ids := map[string]bool{}
	for _, r := range resultados {
		ids[r.ChunkID] = true
	}
	return ids
}
