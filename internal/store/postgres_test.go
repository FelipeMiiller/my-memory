package store

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestFormatVector(t *testing.T) {
	vec := []float32{0.1, -0.5, 1.25}
	got := FormatVector(vec)
	expected := "[0.1,-0.5,1.25]"
	if got != expected {
		t.Errorf("FormatVector(%v) = %s, esperava %s", vec, got, expected)
	}
}

func TestFormatVector_Empty(t *testing.T) {
	vec := []float32{}
	got := FormatVector(vec)
	expected := "[]"
	if got != expected {
		t.Errorf("FormatVector(%v) = %s, esperava %s", vec, got, expected)
	}
}

func TestPostgresStore_InterfaceCompliance(t *testing.T) {
	var _ Store = (*PostgresStore)(nil)
}

func TestPostgresStore_SearchFTS_EmptyQuery(t *testing.T) {
	s := &PostgresStore{}
	res, err := s.SearchFTS(nil, "repo", "   ", 10)
	if err != nil {
		t.Fatalf("Esperava erro nulo para query vazia, obteve: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("Esperava resultado vazio para query vazia, obteve %d itens", len(res))
	}
}

func TestPostgresStore_SearchHybridRRFWithDecay_EmptyInputs(t *testing.T) {
	s := &PostgresStore{}
	res, err := s.SearchHybridRRFWithDecay(nil, "repo", "   ", nil, 10, 60, DefaultDecayOptions())
	if err != nil {
		t.Fatalf("Esperava erro nulo para inputs vazios, obteve: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("Esperava resultado vazio para inputs vazios, obteve %d itens", len(res))
	}
}

func TestPostgresSchema_ContentHash(t *testing.T) {
	if !strings.Contains(PostgresSchema, "content_hash TEXT") {
		t.Fatalf("PostgresSchema deve conter coluna content_hash TEXT")
	}
	if !strings.Contains(PostgresSchema, "ALTER TABLE documents ADD COLUMN IF NOT EXISTS content_hash TEXT;") {
		t.Fatalf("PostgresSchema deve conter migração retrocompatível para content_hash")
	}
}

func TestPostgresSchema_EpistemicEdges(t *testing.T) {
	if !strings.Contains(PostgresSchema, "epistemic_status TEXT NOT NULL DEFAULT 'EXTRACTED'") {
		t.Fatalf("PostgresSchema deve conter coluna epistemic_status")
	}
	if !strings.Contains(PostgresSchema, "weight REAL NOT NULL DEFAULT 1.0") {
		t.Fatalf("PostgresSchema deve conter coluna weight")
	}
	if !strings.Contains(PostgresSchema, "ALTER TABLE graph_edges ADD COLUMN IF NOT EXISTS epistemic_status TEXT NOT NULL DEFAULT 'EXTRACTED';") {
		t.Fatalf("PostgresSchema deve conter migração para epistemic_status")
	}
	if !strings.Contains(PostgresSchema, "ALTER TABLE graph_edges ADD COLUMN IF NOT EXISTS weight REAL NOT NULL DEFAULT 1.0;") {
		t.Fatalf("PostgresSchema deve conter migração para weight")
	}
}

func TestGodNode_Struct(t *testing.T) {
	node := GodNode{
		ID:          "Arquitetura",
		Name:        "Arquitetura",
		InDegree:    5,
		OutDegree:   3,
		TotalDegree: 8,
	}
	if node.TotalDegree != node.InDegree+node.OutDegree {
		t.Fatalf("TotalDegree inconsistente: %d != %d", node.TotalDegree, node.InDegree+node.OutDegree)
	}
}

func TestPostgresStore_Integration(t *testing.T) {
	pgURL := os.Getenv("MY_MEMORY_PG_URL")
	if pgURL == "" {
		t.Skip("Pulando teste de integração PostgreSQL: MY_MEMORY_PG_URL não configurada")
	}

	ctx := context.Background()
	s, err := NewPostgresStore(pgURL)
	if err != nil {
		t.Fatalf("Falha ao conectar no PostgreSQL (%s): %v", pgURL, err)
	}
	defer s.Close()

	repo := "test-repo"
	docID := "doc-1"

	// 1. InsertDocument e GetDocumentHash
	err = s.InsertDocument(ctx, repo, docID, "notes/test.md", "Test Note", time.Now().Unix(), "hash123")
	if err != nil {
		t.Fatalf("InsertDocument falhou: %v", err)
	}

	hash, err := s.GetDocumentHash(ctx, repo, docID)
	if err != nil {
		t.Fatalf("GetDocumentHash falhou: %v", err)
	}
	if hash != "hash123" {
		t.Errorf("GetDocumentHash = %q; esperava 'hash123'", hash)
	}

	// 2. Chunks e busca vetorial + FTS
	dummyVec := make([]float32, 768)
	dummyVec[0] = 1.0
	err = s.InsertChunk(ctx, repo, "doc-1#0", docID, "Conteúdo de teste sobre inteligência artificial e grafos", 0, dummyVec)
	if err != nil {
		t.Fatalf("InsertChunk falhou: %v", err)
	}

	ftsRes, err := s.SearchFTS(ctx, repo, "inteligência", 5)
	if err != nil {
		t.Fatalf("SearchFTS falhou: %v", err)
	}
	if len(ftsRes) == 0 {
		t.Errorf("SearchFTS não retornou resultados para 'inteligência'")
	}

	knnRes, err := s.SearchKNN(ctx, repo, dummyVec, 5)
	if err != nil {
		t.Fatalf("SearchKNN falhou: %v", err)
	}
	if len(knnRes) == 0 {
		t.Errorf("SearchKNN não retornou resultados")
	}

	// 3. Arestas tipadas e GodNodes
	err = s.InsertEdgeWithProps(ctx, repo, docID, "doc-2", "depends_on", "EXTRACTED", 1.0)
	if err != nil {
		t.Fatalf("InsertEdgeWithProps falhou: %v", err)
	}

	neighbors, err := s.GetNodeNeighbors(ctx, repo, docID, 1)
	if err != nil {
		t.Fatalf("GetNodeNeighbors falhou: %v", err)
	}
	if len(neighbors) == 0 || neighbors[0] != "doc-2" {
		t.Errorf("GetNodeNeighbors = %v; esperava ['doc-2']", neighbors)
	}

	godNodes, err := s.GetGodNodes(ctx, repo, 5)
	if err != nil {
		t.Fatalf("GetGodNodes falhou: %v", err)
	}
	if len(godNodes) == 0 {
		t.Errorf("GetGodNodes retornou lista vazia")
	}

	// 3.1. PageRank no PostgreSQL
	prNodes, err := s.ComputePageRank(ctx, repo, 0.85, 20)
	if err != nil {
		t.Fatalf("ComputePageRank falhou: %v", err)
	}
	if len(prNodes) == 0 {
		t.Errorf("ComputePageRank retornou lista vazia")
	}
	if prNodes[0].Rank != 1 || prNodes[0].Score <= 0 {
		t.Errorf("Primeiro nó do PageRank PostgreSQL com rank ou score inválido: %+v", prNodes[0])
	}

	// 4. Conexões Inesperadas (Surprising Connections)
	// Insere doc-close com embedding similar a doc-1 mas sem aresta no grafo
	closeDocID := "doc-close"
	_ = s.InsertDocument(ctx, repo, closeDocID, "notes/close.md", "Close Note", time.Now().Unix(), "hashclose")
	_ = s.InsertChunk(ctx, repo, "doc-close#0", closeDocID, "Conteudo quase identico para teste", 0, dummyVec)

	// doc-1 e doc-2 tem aresta. doc-1 e doc-close NÃO tem aresta.
	surprising, err := s.FindSurprisingConnections(ctx, repo, 10, 0.50)
	if err != nil {
		t.Fatalf("FindSurprisingConnections falhou: %v", err)
	}
	foundClose := false
	for _, sc := range surprising {
		if (sc.SourceID == docID && sc.TargetID == closeDocID) || (sc.SourceID == closeDocID && sc.TargetID == docID) {
			foundClose = true
		}
		if (sc.SourceID == docID && sc.TargetID == "doc-2") || (sc.SourceID == "doc-2" && sc.TargetID == docID) {
			t.Errorf("doc-1 e doc-2 tem aresta direta e NÃO deveriam ser retornados como conexão inesperada!")
		}
		if sc.SourceID == sc.TargetID {
			t.Errorf("Conexão inesperada reflexiva encontrada: %s == %s", sc.SourceID, sc.TargetID)
		}
	}
	if !foundClose {
		t.Errorf("Esperava encontrar conexao inesperada entre doc-1 e doc-close")
	}

	// 4.1. CalculateImpact e InspectNode no PostgreSQL
	impact, err := s.CalculateImpact(ctx, repo, docID, 2)
	if err != nil {
		t.Fatalf("CalculateImpact falhou no PostgreSQL: %v", err)
	}
	if impact.TargetNode != docID {
		t.Errorf("CalculateImpact retornou target incorreto: %s", impact.TargetNode)
	}

	view, err := s.InspectNode(ctx, repo, docID, 200)
	if err != nil {
		t.Fatalf("InspectNode falhou no PostgreSQL: %v", err)
	}
	if view.Target.ID != docID {
		t.Errorf("InspectNode retornou target incorreto: %s", view.Target.ID)
	}

	// 5. DeleteDocumentData
	err = s.DeleteDocumentData(ctx, repo, docID)
	if err != nil {
		t.Fatalf("DeleteDocumentData falhou: %v", err)
	}
	_ = s.DeleteDocumentData(ctx, repo, closeDocID)
}
