//go:build cgo

package db

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/turboquant"
)

func TestSQLiteStore_Lifecycle(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_memory.db")
	database, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	docID := "test-doc"

	// 1. InsertDocument & GetDocumentHash
	err = InsertDocument(ctx, database, docID, "notes/doc.md", "Doc Test", time.Now().Unix(), "sha256hash")
	if err != nil {
		t.Fatalf("InsertDocument falhou: %v", err)
	}

	hash, err := GetDocumentHash(ctx, database, docID)
	if err != nil {
		t.Fatalf("GetDocumentHash falhou: %v", err)
	}
	if hash != "sha256hash" {
		t.Errorf("GetDocumentHash = %q; esperava 'sha256hash'", hash)
	}

	// 2. Chunks, FTS & TurboQuant
	dummyVec := make([]float32, 768)
	dummyVec[0] = 0.5
	err = InsertChunk(ctx, database, "test-doc#0", docID, "Texto de teste para FTS5 e sqlite-vec", 0, dummyVec)
	if err != nil {
		t.Fatalf("InsertChunk falhou: %v", err)
	}

	tq := turboquant.NewQuantizer(768)
	qVec, err := tq.Quantize(dummyVec)
	if err != nil {
		t.Fatalf("Quantize falhou: %v", err)
	}
	err = InsertTurboQuantChunk(ctx, database, "test-doc#0", qVec.Scale, qVec.Data)
	if err != nil {
		t.Fatalf("InsertTurboQuantChunk falhou: %v", err)
	}

	ftsRes, err := SearchFTS(ctx, database, "FTS5", 5)
	if err != nil {
		t.Fatalf("SearchFTS falhou: %v", err)
	}
	if len(ftsRes) == 0 {
		t.Errorf("SearchFTS não encontrou resultados")
	}

	knnRes, err := SearchKNN(ctx, database, dummyVec, 5)
	if err != nil {
		t.Fatalf("SearchKNN falhou: %v", err)
	}
	if len(knnRes) == 0 {
		t.Errorf("SearchKNN não encontrou resultados")
	}

	hybridRes, err := SearchHybridRRF(ctx, database, tq, "FTS5", dummyVec, 5, 60, false)
	if err != nil {
		t.Fatalf("SearchHybridRRF falhou: %v", err)
	}
	if len(hybridRes) == 0 {
		t.Errorf("SearchHybridRRF não encontrou resultados")
	}

	hybridTQRes, err := SearchHybridRRF(ctx, database, tq, "FTS5", dummyVec, 5, 60, true)
	if err != nil {
		t.Fatalf("SearchHybridRRF com TurboQuant falhou: %v", err)
	}
	if len(hybridTQRes) == 0 {
		t.Errorf("SearchHybridRRF com TurboQuant não encontrou resultados")
	}

	// 3. Arestas tipadas e God Nodes
	err = InsertEdgeWithProps(ctx, database, docID, "target-doc", "implements", "EXTRACTED", 1.0)
	if err != nil {
		t.Fatalf("InsertEdgeWithProps falhou: %v", err)
	}

	neighbors, err := GetNodeNeighbors(ctx, database, docID, 1)
	if err != nil {
		t.Fatalf("GetNodeNeighbors falhou: %v", err)
	}
	if len(neighbors) == 0 || neighbors[0] != "target-doc" {
		t.Errorf("GetNodeNeighbors = %v; esperava ['target-doc']", neighbors)
	}

	hubs, err := GetGodNodes(ctx, database, 5)
	if err != nil {
		t.Fatalf("GetGodNodes falhou: %v", err)
	}
	if len(hubs) == 0 {
		t.Errorf("GetGodNodes retornou lista vazia")
	}

	// 4. Conexões Inesperadas (Surprising Connections)
	// Insere doc-close com embedding similar a test-doc sem aresta
	closeDocID := "doc-close"
	_ = InsertDocument(ctx, database, closeDocID, "notes/close.md", "Close Note", time.Now().Unix(), "closehash")
	_ = InsertChunk(ctx, database, "doc-close#0", closeDocID, "Texto muito proximo para teste", 0, dummyVec)

	// test-doc e target-doc têm aresta. test-doc e doc-close NÃO têm aresta.
	surprising, err := FindSurprisingConnections(ctx, database, 10, 0.50)
	if err != nil {
		t.Fatalf("FindSurprisingConnections falhou: %v", err)
	}
	foundClose := false
	for _, sc := range surprising {
		if (sc.SourceID == docID && sc.TargetID == closeDocID) || (sc.SourceID == closeDocID && sc.TargetID == docID) {
			foundClose = true
		}
		if (sc.SourceID == docID && sc.TargetID == "target-doc") || (sc.SourceID == "target-doc" && sc.TargetID == docID) {
			t.Errorf("test-doc e target-doc têm aresta direta e NÃO devem ser conexão inesperada!")
		}
		if sc.SourceID == sc.TargetID {
			t.Errorf("Conexão inesperada reflexiva encontrada: %s == %s", sc.SourceID, sc.TargetID)
		}
	}
	if !foundClose {
		t.Errorf("Esperava encontrar conexão inesperada entre test-doc e doc-close")
	}

	// 5. DeleteDocumentData
	err = DeleteDocumentData(ctx, database, docID)
	if err != nil {
		t.Fatalf("DeleteDocumentData falhou: %v", err)
	}
}

func TestSQLite_FindSurprisingConnections_LexicalFallback(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_lexical.db")
	database, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	ctx := context.Background()

	// Insere documentos apenas com texto (sem vetores em chunks_vec ou chunks_turboquant)
	_ = InsertDocument(ctx, database, "doc-1", "doc1.md", "Microservices Architecture", time.Now().Unix(), "h1")
	_, _ = database.ExecContext(ctx, `INSERT INTO chunks (id, document_id, chunk_index, content) VALUES ('doc-1#0', 'doc-1', 0, 'arquitetura distribuída microsserviços resiliência')`)

	_ = InsertDocument(ctx, database, "doc-2", "doc2.md", "Resilient Architecture", time.Now().Unix(), "h2")
	_, _ = database.ExecContext(ctx, `INSERT INTO chunks (id, document_id, chunk_index, content) VALUES ('doc-2#0', 'doc-2', 0, 'arquitetura distribuída resiliência alta disponibilidade')`)

	_ = InsertDocument(ctx, database, "doc-3", "doc3.md", "Pasta Recipes", time.Now().Unix(), "h3")
	_, _ = database.ExecContext(ctx, `INSERT INTO chunks (id, document_id, chunk_index, content) VALUES ('doc-3#0', 'doc-3', 0, 'culinária italiana massas molho receitas')`)

	// Conexão inesperada léxica entre doc-1 e doc-2
	results, err := FindSurprisingConnections(ctx, database, 10, 0.40)
	if err != nil {
		t.Fatalf("FindSurprisingConnections falhou: %v", err)
	}

	found := false
	for _, sc := range results {
		if (sc.SourceID == "doc-1" && sc.TargetID == "doc-2") || (sc.SourceID == "doc-2" && sc.TargetID == "doc-1") {
			found = true
			if sc.Similarity < 0.40 {
				t.Errorf("Similaridade esperada >= 0.40, obteve %f", sc.Similarity)
			}
		}
		if sc.TargetID == "doc-3" || sc.SourceID == "doc-3" {
			t.Errorf("doc-3 é culinária e não deve ter sobreposição suficiente com doc-1 ou doc-2")
		}
	}

	if !found {
		t.Fatalf("Fallback léxico não detectou conexão surpreendente entre doc-1 e doc-2")
	}
}
