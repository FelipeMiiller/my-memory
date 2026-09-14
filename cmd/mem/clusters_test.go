package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/graph"
	"github.com/FelipeMiiller/my-memory/internal/graphview"
)

func TestClustersCLI_EmptyAndDegraded(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer

	// Executa com um arquivo de banco inexistente; deve degradar graciosamente sem pânico
	err := runClustersCommand(ctx, "test-repo", []string{"--db", "non_existent_db_12345.db", "--min-size", "2"}, &buf)
	if err != nil {
		t.Fatalf("esperava sucesso na degradação graciosa, obteve erro: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Nenhum cluster com tamanho >= 2 encontrado") {
		t.Errorf("saída inesperada para banco vazio/inexistente:\n%s", out)
	}
}

func TestClustersCLI_JSONOutput(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer

	err := runClustersCommand(ctx, "test-repo", []string{"--db", "non_existent.db", "--json", "--min-size", "1"}, &buf)
	if err != nil {
		t.Fatalf("erro ao executar comando com --json: %v", err)
	}

	var resp ClustersResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("falha ao desserializar JSON de saída: %v\nJSON bruto: %s", err, buf.String())
	}

	if resp.MinSize != 1 {
		t.Errorf("esperava min_size 1, obteve: %d", resp.MinSize)
	}
	if resp.Communities == nil {
		t.Error("campo communities não deve ser nil")
	}
}

func TestClustersCLI_FormatTableAndJSON(t *testing.T) {
	mockCommunities := []graph.Community{
		{
			ID:           1,
			Label:        "concept/auth",
			LeadNode:     "concept/auth",
			Members:      []string{"concept/auth", "decision/jwt", "guide/auth"},
			Size:         3,
			DominantType: "concept",
		},
		{
			ID:           2,
			Label:        "concept/db",
			LeadNode:     "concept/db",
			Members:      []string{"concept/db", "decision/postgres"},
			Size:         2,
			DominantType: "decision",
		},
	}

	stats := graphview.GraphStats{
		TotalNodes:     5,
		TotalEdges:     6,
		CommunityCount: 2,
		Modularity:     0.485,
	}

	// 1. Testa formatClustersTable
	var tableBuf bytes.Buffer
	if err := formatClustersTable(mockCommunities, stats, 2, &tableBuf); err != nil {
		t.Fatalf("erro ao formatar tabela: %v", err)
	}
	tableStr := tableBuf.String()
	if !strings.Contains(tableStr, "Newman-Girvan Q: 0.485") {
		t.Errorf("esperava modularidade no cabeçalho:\n%s", tableStr)
	}
	if !strings.Contains(tableStr, "concept/auth") || !strings.Contains(tableStr, "concept/db") {
		t.Errorf("esperava nós líderes na tabela:\n%s", tableStr)
	}
	if !strings.Contains(tableStr, "decision/jwt") {
		t.Errorf("esperava membros na tabela:\n%s", tableStr)
	}

	// 2. Testa formatClustersJSON
	var jsonBuf bytes.Buffer
	if err := formatClustersJSON(mockCommunities, stats, 2, &jsonBuf); err != nil {
		t.Fatalf("erro ao formatar JSON: %v", err)
	}
	var resp ClustersResponse
	if err := json.Unmarshal(jsonBuf.Bytes(), &resp); err != nil {
		t.Fatalf("falha ao parsear JSON: %v", err)
	}
	if resp.Modularity != 0.485 || resp.TotalNodes != 5 || len(resp.Communities) != 2 {
		t.Errorf("resposta JSON diverge dos dados: %+v", resp)
	}
	if resp.Communities[0].LeadNode != "concept/auth" {
		t.Errorf("comunidade 1 líder incorreto: %s", resp.Communities[0].LeadNode)
	}
}

func TestClustersCLI_WithSQLiteData(t *testing.T) {
	ctx := context.Background()
	tmpDir, err := os.MkdirTemp("", "mem_clusters_test_*")
	if err != nil {
		t.Fatalf("falha ao criar temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbFile := filepath.Join(tmpDir, "test.db")
	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Skipf("SQLite não disponível no ambiente de teste: %v", err)
	}

	// Criar tabelas se não existirem
	setupTables(t, database)

	// Inserir documentos (Cluster 1: auth, Cluster 2: storage, Nó isolado)
	insertDoc(t, database, "concept/auth", "Authentication Concept")
	insertDoc(t, database, "decision/jwt", "JWT Decision")
	insertDoc(t, database, "guide/auth", "Auth Integration Guide")

	insertDoc(t, database, "concept/db", "Database Concept")
	insertDoc(t, database, "decision/postgres", "PostgreSQL Decision")

	insertDoc(t, database, "other/orphan", "Orphan Note")

	// Inserir arestas
	insertEdge(t, database, "concept/auth", "decision/jwt", "EXTRACTED", 1.0)
	insertEdge(t, database, "decision/jwt", "guide/auth", "EXTRACTED", 1.0)
	insertEdge(t, database, "guide/auth", "concept/auth", "EXTRACTED", 1.0)

	insertEdge(t, database, "concept/db", "decision/postgres", "EXTRACTED", 1.0)

	database.Close()

	// 1. Teste de saída tabular padrão com min-size 2
	var tableBuf bytes.Buffer
	err = runClustersCommand(ctx, "", []string{"--db", dbFile, "--min-size", "2"}, &tableBuf)
	if err != nil {
		t.Fatalf("erro ao rodar mem clusters com sqlite: %v", err)
	}

	tableOut := tableBuf.String()
	if !strings.Contains(tableOut, "Clusters e Comunidades no Grafo") {
		t.Errorf("cabeçalho ausente na saída:\n%s", tableOut)
	}
	if !strings.Contains(tableOut, "LÍDER") || !strings.Contains(tableOut, "TAMANHO") {
		t.Errorf("colunas tabulares ausentes na saída:\n%s", tableOut)
	}
	// O nó isolado (tamanho 1) deve ter sido filtrado pelo min-size 2
	if strings.Contains(tableOut, "other/orphan") {
		t.Errorf("nó isolado other/orphan não deveria aparecer com --min-size 2:\n%s", tableOut)
	}

	// 2. Teste de saída JSON com min-size 1
	var jsonBuf bytes.Buffer
	err = runClustersCommand(ctx, "", []string{"--db", dbFile, "--json", "--min-size", "1"}, &jsonBuf)
	if err != nil {
		t.Fatalf("erro ao rodar mem clusters --json: %v", err)
	}

	var jsonResp ClustersResponse
	if err := json.Unmarshal(jsonBuf.Bytes(), &jsonResp); err != nil {
		t.Fatalf("falha ao ler JSON gerado: %v", err)
	}

	if jsonResp.TotalNodes != 6 {
		t.Errorf("esperava 6 nós totais, obteve %d", jsonResp.TotalNodes)
	}
	if len(jsonResp.Communities) < 2 {
		t.Errorf("esperava ao menos 2 comunidades detectadas, obteve: %d", len(jsonResp.Communities))
	}
	if jsonResp.Modularity <= 0.0 {
		t.Errorf("modularidade esperada > 0.0 para clusters bem divididos, obteve: %f", jsonResp.Modularity)
	}
}

func setupTables(t *testing.T, dbConn *sql.DB) {
	schema := `
	CREATE TABLE IF NOT EXISTS documents (
		id TEXT PRIMARY KEY,
		title TEXT,
		path TEXT,
		source_type TEXT,
		hash TEXT,
		content TEXT,
		updated_at INTEGER,
		repository TEXT DEFAULT ''
	);
	CREATE TABLE IF NOT EXISTS graph_edges (
		source_id TEXT,
		target_id TEXT,
		relation TEXT DEFAULT 'links_to',
		epistemic_status TEXT DEFAULT 'EXTRACTED',
		weight REAL DEFAULT 1.0,
		repository TEXT DEFAULT '',
		PRIMARY KEY (source_id, target_id, relation)
	);
	`
	if _, err := dbConn.Exec(schema); err != nil {
		t.Fatalf("falha ao criar schema para teste: %v", err)
	}
}

func insertDoc(t *testing.T, dbConn *sql.DB, id, title string) {
	_, err := dbConn.Exec("INSERT OR REPLACE INTO documents (id, title, updated_at) VALUES (?, ?, ?)", id, title, 1000)
	if err != nil {
		t.Fatalf("falha ao inserir documento %s: %v", id, err)
	}
}

func insertEdge(t *testing.T, dbConn *sql.DB, src, tgt, status string, weight float64) {
	_, err := dbConn.Exec("INSERT OR REPLACE INTO graph_edges (source_id, target_id, epistemic_status, weight) VALUES (?, ?, ?, ?)",
		src, tgt, status, weight)
	if err != nil {
		t.Fatalf("falha ao inserir aresta %s -> %s: %v", src, tgt, err)
	}
}
