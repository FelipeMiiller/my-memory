package store

import (
	"context"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/embedder"
	"github.com/FelipeMiiller/my-memory/internal/pgtest"
)

// TestPostgresStore_HybridDeclaraFonteVector prova o AC4 no motor que o
// cofre central usa: quando a busca vetorial participa da fusão RRF, o
// resultado declara a fonte `vector`.
//
// O caminho SQLite tem o mesmo contrato, asseverado em `cmd/mem`. Aqui o
// banco é descartável (pgtest) — o Postgres do dono nunca é alvo.
func TestPostgresStore_HybridDeclaraFonteVector(t *testing.T) {
	h := pgtest.New(t)

	ctx := context.Background()
	s, err := NewPostgresStore(h.DSN)
	if err != nil {
		t.Fatalf("conectar no PostgreSQL descartável (%s): %v", h.DBName, err)
	}
	defer s.Close()

	repo := "repo-fonte-vector"
	docID := "doc-arquitetura"

	if err := s.InsertDocument(ctx, repo, docID, "notas/arquitetura.md", "arquitetura do sistema", 1700000000, "hash-arquitetura"); err != nil {
		t.Fatalf("InsertDocument falhou: %v", err)
	}

	// Vetor normalizado com norma 1: é o que a distância de cosseno do
	// pgvector espera.
	vec := make([]float32, embedder.EmbeddingDim)
	vec[0] = 1.0
	if err := s.InsertChunk(ctx, repo, docID+"#0", docID, "arquitetura do sistema e seus componentes", 0, vec); err != nil {
		t.Fatalf("InsertChunk falhou: %v", err)
	}

	res, err := s.SearchHybridWithOptions(ctx, repo, "arquitetura", vec, 10, 60, DefaultDecayOptions(), SearchOptions{Level: "l1"})
	if err != nil {
		t.Fatalf("SearchHybridWithOptions falhou: %v", err)
	}
	if len(res) == 0 {
		t.Fatal("a busca híbrida não retornou resultados; o teste não exercitou a fonte vetorial")
	}

	var declarou bool
	for _, r := range res {
		for _, fonte := range r.Sources {
			if strings.HasPrefix(fonte, "vector:") {
				declarou = true
			}
		}
	}
	if !declarou {
		t.Errorf("nenhum resultado declarou a fonte `vector`; sources = %+v", res[0].Sources)
	}
}
