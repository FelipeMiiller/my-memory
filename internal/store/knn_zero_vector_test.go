package store

import (
	"context"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/embedder"
	"github.com/FelipeMiiller/my-memory/internal/pgtest"
)

// TestKNNZeroVectorPostgres prova a AC4 no Postgres: a busca vetorial não
// devolve chunk sem vetor real — nem o vetor zero, nem o chunk sem
// embedding.
//
// O vetor zero (o defeito da ISSUE-040) PREENCHE a coluna, então ele sai do
// KNN como qualquer outro — e a distância de cosseno de um vetor zero é
// indefinida: o hit sai sem ordem confiável e com aparência de acerto.
//
// O banco é descartável (pgtest) — o Postgres do dono nunca é alvo.
func TestKNNZeroVectorPostgres(t *testing.T) {
	h := pgtest.New(t)

	ctx := context.Background()
	s, err := NewPostgresStore(h.DSN)
	if err != nil {
		t.Fatalf("conectar no PostgreSQL descartável (%s): %v", h.DBName, err)
	}
	defer s.Close()

	repo := "repo-knn-zero-vector"
	docID := "doc-1"
	if err := s.InsertDocument(ctx, repo, docID, "notas/doc.md", "Doc", 1700000000, "hash-1"); err != nil {
		t.Fatalf("InsertDocument falhou: %v", err)
	}

	// chunk 0: vetor ZERO (o defeito).
	if err := s.InsertChunk(ctx, repo, "doc-1#0", docID, "conteudo com vetor zero", 0, make([]float32, embedder.EmbeddingDim)); err != nil {
		t.Fatalf("InsertChunk com vetor zero falhou: %v", err)
	}

	// chunk 1: vetor REAL.
	real := make([]float32, embedder.EmbeddingDim)
	real[0] = 1.0
	real[1] = 1.0
	if err := s.InsertChunk(ctx, repo, "doc-1#1", docID, "conteudo com vetor real", 1, real); err != nil {
		t.Fatalf("InsertChunk com vetor real falhou: %v", err)
	}

	// chunk 2: sem embedding nenhum.
	if err := s.InsertChunk(ctx, repo, "doc-1#2", docID, "conteudo sem vetor", 2, nil); err != nil {
		t.Fatalf("InsertChunk sem vetor falhou: %v", err)
	}

	queryVec := make([]float32, embedder.EmbeddingDim)
	queryVec[0] = 1.0
	queryVec[1] = 1.0

	resultados, err := s.SearchKNN(ctx, repo, queryVec, 10)
	if err != nil {
		t.Fatalf("SearchKNN falhou: %v", err)
	}

	ids := map[string]bool{}
	for _, r := range resultados {
		ids[r.ChunkID] = true
	}

	if _, ok := ids["doc-1#0"]; ok {
		t.Error("a busca vetorial devolveu o chunk com vetor ZERO: ele não tem similaridade de verdade")
	}
	if _, ok := ids["doc-1#2"]; ok {
		t.Error("a busca vetorial devolveu o chunk sem embedding (NULL)")
	}
	if _, ok := ids["doc-1#1"]; !ok {
		t.Error("a busca vetorial NÃO devolveu o chunk com vetor real")
	}
}

// TestKNNTodosSemVetorPostgres prova que a busca com todos os chunks sem
// vetor devolve lista vazia — e não erro.
//
// É o caso do vault indexado sem provider: ele tem chunk, mas nenhum deles
// tem vetor.
func TestKNNTodosSemVetorPostgres(t *testing.T) {
	h := pgtest.New(t)

	ctx := context.Background()
	s, err := NewPostgresStore(h.DSN)
	if err != nil {
		t.Fatalf("conectar no PostgreSQL descartável (%s): %v", h.DBName, err)
	}
	defer s.Close()

	repo := "repo-knn-todo-zero"
	docID := "doc-1"
	if err := s.InsertDocument(ctx, repo, docID, "notas/doc.md", "Doc", 1700000000, "hash-1"); err != nil {
		t.Fatalf("InsertDocument falhou: %v", err)
	}
	if err := s.InsertChunk(ctx, repo, "doc-1#0", docID, "conteudo so com vetor zero", 0, make([]float32, embedder.EmbeddingDim)); err != nil {
		t.Fatalf("InsertChunk com vetor zero falhou: %v", err)
	}

	queryVec := make([]float32, embedder.EmbeddingDim)
	queryVec[0] = 1.0

	resultados, err := s.SearchKNN(ctx, repo, queryVec, 10)
	if err != nil {
		t.Fatalf("a busca com todos os chunks sem vetor deu erro em vez de lista vazia: %v", err)
	}
	if len(resultados) != 0 {
		t.Errorf("a busca com todos os chunks sem vetor devolveu %d resultado(s), esperado lista vazia", len(resultados))
	}
}
