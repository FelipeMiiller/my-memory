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

	// 4. DeleteDocumentData
	err = DeleteDocumentData(ctx, database, docID)
	if err != nil {
		t.Fatalf("DeleteDocumentData falhou: %v", err)
	}
}
