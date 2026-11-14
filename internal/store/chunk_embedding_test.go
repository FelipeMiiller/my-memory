package store

import (
	"context"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/embedder"
	"github.com/FelipeMiiller/my-memory/internal/pgtest"
)

// TestChunkEmbeddingAusenteGravadoComoNULL prova a AC1 no Postgres: um chunk
// indexado sem provider é gravado com `embedding` NULL — e não com um vetor
// zero.
//
// O vetor zero é o defeito: ele "existe" para quem só olha a coluna, e o
// doctor procura `embedding IS NULL` para dizer que o chunk está
// dessincronizado. Gravar zero é esconder a ausência.
//
// O banco é descartável (pgtest) — o Postgres do dono nunca é alvo.
func TestChunkEmbeddingAusenteGravadoComoNULL(t *testing.T) {
	h := pgtest.New(t)

	ctx := context.Background()
	s, err := NewPostgresStore(h.DSN)
	if err != nil {
		t.Fatalf("conectar no PostgreSQL descartável (%s): %v", h.DBName, err)
	}
	defer s.Close()

	repo := "repo-chunk-embedding"
	docID := "doc-1"
	if err := s.InsertDocument(ctx, repo, docID, "notas/doc.md", "Doc", 1700000000, "hash-1"); err != nil {
		t.Fatalf("InsertDocument falhou: %v", err)
	}

	// 1. Sem vetor: o embedding é NULL.
	if err := s.InsertChunk(ctx, repo, "doc-1#0", docID, "conteudo sem vetor", 0, nil); err != nil {
		t.Fatalf("InsertChunk sem vetor falhou: %v", err)
	}

	var nulo bool
	err = s.db.QueryRowContext(ctx,
		`SELECT embedding IS NULL FROM chunks WHERE repository = $1 AND id = $2`,
		repo, "doc-1#0").Scan(&nulo)
	if err != nil {
		t.Fatalf("consultando o embedding do chunk sem vetor: %v", err)
	}
	if !nulo {
		t.Error("o chunk sem vetor real deveria estar com embedding NULL, não com um vetor (zero ou de qualquer outra forma)")
	}

	// 2. Com vetor real: gravado como antes.
	vec := make([]float32, embedder.EmbeddingDim)
	vec[0] = 1.0
	if err := s.InsertChunk(ctx, repo, "doc-1#1", docID, "conteudo com vetor", 1, vec); err != nil {
		t.Fatalf("InsertChunk com vetor falhou: %v", err)
	}

	var gravado string
	err = s.db.QueryRowContext(ctx,
		`SELECT embedding::text FROM chunks WHERE repository = $1 AND id = $2`,
		repo, "doc-1#1").Scan(&gravado)
	if err != nil {
		t.Fatalf("consultando o embedding do chunk com vetor: %v", err)
	}
	if esperado := FormatVector(vec); gravado != esperado {
		t.Errorf("o vetor real foi gravado como %q, esperado %q", gravado, esperado)
	}
}
