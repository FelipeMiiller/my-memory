package store

import (
	"context"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/embedder"
	"github.com/FelipeMiiller/my-memory/internal/pgtest"
)

// TestPostgresStore_SearchPopulatesMTime prova, contra um PostgreSQL vivo, que
// o caminho de busca carrega o mtime do ARQUIVO e não o updated_at do índice.
//
// Sem isso, os três sites Postgres devolviam SearchResult.MTime = 0 e o
// rrf.go caía para updated_at — ou seja, o bug original do --decay continuava
// vivo no motor canônico (ADR-040) enquanto o SQLite já estava corrigido.
func TestPostgresStore_SearchPopulatesMTime(t *testing.T) {
	// Banco DESCARTÁVEL próprio: o Postgres do dono nunca é alvo (o TestMain
	// neutraliza MY_MEMORY_PG_URL e o harness cria o mymemory_pgtest_*).
	h := pgtest.New(t)

	ctx := context.Background()
	s, err := NewPostgresStore(h.DSN)
	if err != nil {
		t.Fatalf("conexão com o PostgreSQL descartável falhou: %v", err)
	}
	defer s.Close()

	repo := "test-mtime-probe"
	const docID = "mtime-doc-1"
	docPath := "notes/mtime-probe.md"

	// updated_at é gravado AGORA (é o instante da indexação). mtime é de 120
	// dias atrás (é quando o autor editou). Os dois divergem de propósito:
	// um documento cujo updated_at fosse lido daria multiplicador ~1.0.
	now := time.Now().Unix()
	fileEditedAt := now - int64(120*24*3600)

	err = s.InsertDocumentWithMeta(ctx, repo, docID, docPath, "MTime Probe",
		now, "hash-mtime-probe", "", "resumo", "resource", fileEditedAt, 1024)
	if err != nil {
		t.Fatalf("InsertDocumentWithMeta falhou: %v", err)
	}

	vec := make([]float32, embedder.EmbeddingDim)
	for i := range vec {
		vec[i] = 0.01
	}
	if err := s.InsertChunk(ctx, repo, docID+"#0", docID, "zettellmprobe termo unico deste documento", 0, vec); err != nil {
		t.Fatalf("InsertChunk falhou: %v", err)
	}

	res, err := s.SearchFTS(ctx, repo, "zettellmprobe", 5)
	if err != nil {
		t.Fatalf("SearchFTS falhou: %v", err)
	}
	if len(res) == 0 {
		t.Fatal("SearchFTS não devolveu o documento recién indexado")
	}

	got := res[0]
	if got.DocumentID != docID {
		t.Fatalf("resultado inesperado: esperava %q, veio %q", docID, got.DocumentID)
	}

	if got.MTime == 0 {
		t.Fatalf("MTime = 0: o caminho de busca Postgres NÃO está populando o mtime do arquivo. "+
			"O --decay vai cair para updated_at e repetir o bug original no motor canônico (referência %d).",
			got.UpdatedAt)
	}
	if got.MTime != fileEditedAt {
		t.Fatalf("MTime = %d, esperava o mtime do arquivo %d", got.MTime, fileEditedAt)
	}

	// A prova que importa: com mtime, o decaimento é o do documento antigo.
	// Com updated_at (o instante da indexação), seria ~1.0 e o --decay seria
	// um no-op silencioso.
	ref := now
	const halfLife = 30.0
	const weight = 0.3

	byMTime := CalculateTimeDecay(got.MTime, ref, halfLife, weight)
	byUpdatedAt := CalculateTimeDecay(got.UpdatedAt, ref, halfLife, weight)

	if byMTime >= byUpdatedAt {
		t.Fatalf("o decaimento por mtime (%.6f) deveria ser MENOR que o por updated_at (%.6f): "+
			"o documento tem 120 dias, então o decaimento tem de penalizá-lo",
			byMTime, byUpdatedAt)
	}
	if byUpdatedAt < 0.999 {
		t.Fatalf("precondição quebrada: updated_at é o instante da indexação, então o decaimento por ele "+
			"deveria ser ~1.0 (veio %.6f). Se não é, o fixture não prova a distinção", byUpdatedAt)
	}
	t.Logf("decaimento por mtime=%.6f (120 dias) vs por updated_at=%.6f (agora): o --decay distingue os dois",
		byMTime, byUpdatedAt)
}
