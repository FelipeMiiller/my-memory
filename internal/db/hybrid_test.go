package db

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/store"
)

func TestConversion_StoreToDb_UpdatedAt(t *testing.T) {
	dbRes := []SearchResult{
		{
			ChunkID:    "c1",
			DocumentID: "d1",
			Content:    "content 1",
			Distance:   0.25,
			Score:      0.88,
			Sources:    []string{"fts:1"},
			Neighbors:  []string{"d2"},
			UpdatedAt:  1700000000,
		},
	}
	storeRes := toStoreResults(dbRes)
	if len(storeRes) != 1 {
		t.Fatalf("esperava 1 item em toStoreResults, obteve %d", len(storeRes))
	}
	if storeRes[0].UpdatedAt != 1700000000 {
		t.Errorf("esperava UpdatedAt 1700000000 em store.SearchResult, obteve %d", storeRes[0].UpdatedAt)
	}

	convertedBack := fromStoreResults(storeRes)
	if len(convertedBack) != 1 {
		t.Fatalf("esperava 1 item em fromStoreResults, obteve %d", len(convertedBack))
	}
	if convertedBack[0].UpdatedAt != 1700000000 {
		t.Errorf("esperava UpdatedAt 1700000000 em db.SearchResult, obteve %d", convertedBack[0].UpdatedAt)
	}
}

func TestSearchHybridRRFWithDecay_SQLite(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test_decay.db")

	database, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("falha ao inicializar SQLite: %v", err)
	}
	defer database.Close()

	now := time.Now().Unix()
	daySeconds := int64(86400)

	oldTime := now - (90 * daySeconds) // 90 dias atrás
	newTime := now - (1 * daySeconds)  // ontem

	// Inserir dois documentos
	if err := InsertDocument(ctx, database, "doc_old", "notes/old.md", "Old Architecture", oldTime, "hash_old"); err != nil {
		t.Fatalf("falha ao inserir doc_old: %v", err)
	}
	if err := InsertDocument(ctx, database, "doc_new", "notes/new.md", "New Architecture", newTime, "hash_new"); err != nil {
		t.Fatalf("falha ao inserir doc_new: %v", err)
	}

	// Inserir chunks
	_, err = database.ExecContext(ctx, `
		INSERT INTO chunks (id, document_id, chunk_index, content)
		VALUES 
			('c_old', 'doc_old', 0, 'go concurrency patterns go concurrency patterns go concurrency patterns memory management'),
			('c_new', 'doc_new', 0, 'go concurrency patterns other text here')
	`)
	if err != nil {
		t.Fatalf("falha ao inserir chunks: %v", err)
	}

	// Inserir na tabela chunks_fts
	_, err = database.ExecContext(ctx, `
		INSERT INTO chunks_fts (chunk_id, content)
		VALUES 
			('c_old', 'go concurrency patterns go concurrency patterns go concurrency patterns memory management'),
			('c_new', 'go concurrency patterns other text here')
	`)
	if err != nil {
		t.Fatalf("falha ao indexar chunks no FTS: %v", err)
	}

	// 1. Testa SearchFTS isolado e verifica se UpdatedAt é populado corretamente
	ftsResults, err := SearchFTS(ctx, database, "go concurrency", 10)
	if err != nil {
		t.Fatalf("SearchFTS falhou: %v", err)
	}
	if len(ftsResults) != 2 {
		t.Fatalf("esperava 2 resultados FTS, obteve %d", len(ftsResults))
	}

	foundOld := false
	foundNew := false
	for _, r := range ftsResults {
		if r.ChunkID == "c_old" {
			foundOld = true
			if r.UpdatedAt != oldTime {
				t.Errorf("esperava UpdatedAt de c_old %d, obteve %d", oldTime, r.UpdatedAt)
			}
		}
		if r.ChunkID == "c_new" {
			foundNew = true
			if r.UpdatedAt != newTime {
				t.Errorf("esperava UpdatedAt de c_new %d, obteve %d", newTime, r.UpdatedAt)
			}
		}
	}
	if !foundOld || !foundNew {
		t.Errorf("esperava encontrar c_old e c_new no FTS, encontrou old=%v, new=%v", foundOld, foundNew)
	}

	// 2. Busca Híbrida sem decaimento: c_old deve vir em primeiro (mais menções de palavras-chave no FTS)
	resultsNoDecay, err := SearchHybridRRF(ctx, database, nil, "go concurrency", nil, 10, 60, false)
	if err != nil {
		t.Fatalf("SearchHybridRRF falhou: %v", err)
	}
	if len(resultsNoDecay) != 2 {
		t.Fatalf("esperava 2 resultados na busca híbrida sem decay, obteve %d", len(resultsNoDecay))
	}
	if resultsNoDecay[0].ChunkID != "c_old" {
		t.Errorf("sem decay, esperava c_old em 1º devido ao BM25 maior, obteve %s", resultsNoDecay[0].ChunkID)
	}

	// 3. Busca Híbrida com decaimento temporal acentuado (halfLife: 15 dias, peso: 0.7)
	decayOpts := store.DecayOptions{
		Enabled:  true,
		HalfLife: 15.0,
		Weight:   0.7,
		RefTime:  now,
	}
	resultsWithDecay, err := SearchHybridRRFWithDecay(ctx, database, nil, "go concurrency", nil, 10, 60, false, decayOpts)
	if err != nil {
		t.Fatalf("SearchHybridRRFWithDecay falhou: %v", err)
	}
	if len(resultsWithDecay) != 2 {
		t.Fatalf("esperava 2 resultados na busca híbrida com decay, obteve %d", len(resultsWithDecay))
	}

	// c_new deve ultrapassar c_old pela pontuação temporal
	if resultsWithDecay[0].ChunkID != "c_new" {
		t.Errorf("com decay, esperava c_new em 1º lugar por recência, obteve %s (score new=%f, old=%f)",
			resultsWithDecay[0].ChunkID, resultsWithDecay[0].Score, resultsWithDecay[1].Score)
	}

	// Verifica se UpdatedAt está preservado no SearchResult final
	if resultsWithDecay[0].UpdatedAt != newTime {
		t.Errorf("esperava UpdatedAt preservado %d, obteve %d", newTime, resultsWithDecay[0].UpdatedAt)
	}
}
