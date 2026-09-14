package main

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/graph"
)

func TestPathCLI_MissingArgs(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer

	err := runPathCommand(ctx, "test-repo", []string{}, &buf)
	if err == nil {
		t.Fatal("esperava erro ao omitir argumentos, obteve nil")
	}
	if !strings.Contains(err.Error(), "origem e destino são obrigatórios") {
		t.Errorf("mensagem de erro inesperada: %v", err)
	}

	err1 := runPathCommand(ctx, "test-repo", []string{"only_source"}, &buf)
	if err1 == nil {
		t.Fatal("esperava erro com apenas 1 argumento, obteve nil")
	}
}

func TestPathCLI_NonExistentNode(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	dbFile := filepath.Join(t.TempDir(), "path_test.db")

	database, err := db.InitDB(dbFile)
	if err != nil {
		if strings.Contains(err.Error(), "no such module: fts5") {
			t.Skip("ambiente sem FTS5")
		}
		t.Fatalf("falha ao inicializar SQLite: %v", err)
	}
	database.Close()

	err = runPathCommand(ctx, "test-repo", []string{"--db", dbFile, "nodeA", "nodeB"}, &buf)
	if err == nil {
		t.Fatal("esperava erro para nó inexistente, obteve nil")
	}
}

func TestPathCLI_TextOutputAndJSON(t *testing.T) {
	ctx := context.Background()
	dbFile := filepath.Join(t.TempDir(), "path_cli.db")

	database, err := db.InitDB(dbFile)
	if err != nil {
		if strings.Contains(err.Error(), "no such module: fts5") {
			t.Skip("ambiente sem FTS5")
		}
		t.Fatalf("falha ao inicializar SQLite: %v", err)
	}

	now := time.Now().Unix()
	// Estrutura: auth -> session -> redis
	_ = db.InsertDocument(ctx, database, "auth", "services/auth.md", "Auth Service", now, "h1")
	_ = db.InsertDocument(ctx, database, "session", "services/session.md", "Session Manager", now, "h2")
	_ = db.InsertDocument(ctx, database, "redis", "infra/redis.md", "Redis Cache", now, "h3")

	_ = db.InsertEdgeWithProps(ctx, database, "auth", "session", "implements", "EXTRACTED", 1.0)
	_ = db.InsertEdgeWithProps(ctx, database, "session", "redis", "depends_on", "EXTRACTED", 1.0)
	database.Close()

	// 1. Teste de saída textual amigável
	var textBuf bytes.Buffer
	err = runPathCommand(ctx, "test-repo", []string{"--db", dbFile, "auth", "redis"}, &textBuf)
	if err != nil {
		t.Fatalf("runPathCommand texto falhou: %v", err)
	}

	outStr := textBuf.String()
	if !strings.Contains(outStr, "=== Descoberta de Rotas & Menor Caminho ===") {
		t.Errorf("cabeçalho não encontrado na saída:\n%s", outStr)
	}
	if !strings.Contains(outStr, "Saltos:          2") {
		t.Errorf("contagem de 2 saltos esperada na saída:\n%s", outStr)
	}
	if !strings.Contains(outStr, "Rota Conectada:") {
		t.Errorf("bloco visual da rota esperado:\n%s", outStr)
	}
	if !strings.Contains(outStr, "implements") || !strings.Contains(outStr, "depends_on") {
		t.Errorf("relações das arestas esperadas:\n%s", outStr)
	}

	// 2. Teste de saída estruturada em JSON
	var jsonBuf bytes.Buffer
	err = runPathCommand(ctx, "test-repo", []string{"--db", dbFile, "--json", "auth", "redis"}, &jsonBuf)
	if err != nil {
		t.Fatalf("runPathCommand json falhou: %v", err)
	}

	var res graph.PathResult
	if err := json.Unmarshal(jsonBuf.Bytes(), &res); err != nil {
		t.Fatalf("falha ao decodificar JSON: %v. Raw:\n%s", err, jsonBuf.String())
	}

	if !res.Found {
		t.Errorf("res.Found esperado true")
	}
	if res.Hops != 2 {
		t.Errorf("res.Hops esperado 2, obteve %d", res.Hops)
	}
	if len(res.Nodes) != 3 || res.Nodes[0] != "auth" || res.Nodes[2] != "redis" {
		t.Errorf("res.Nodes incorreto: %v", res.Nodes)
	}
	if len(res.Edges) != 2 {
		t.Errorf("res.Edges esperado 2, obteve %d", len(res.Edges))
	}
}

func TestPathCLI_UndirectedAndHops(t *testing.T) {
	ctx := context.Background()
	dbFile := filepath.Join(t.TempDir(), "path_undir.db")

	database, err := db.InitDB(dbFile)
	if err != nil {
		if strings.Contains(err.Error(), "no such module: fts5") {
			t.Skip("ambiente sem FTS5")
		}
		t.Fatalf("falha ao inicializar SQLite: %v", err)
	}

	now := time.Now().Unix()
	// A -> B <- C
	_ = db.InsertDocument(ctx, database, "node-a", "docs/a.md", "A", now, "h1")
	_ = db.InsertDocument(ctx, database, "node-b", "docs/b.md", "B", now, "h2")
	_ = db.InsertDocument(ctx, database, "node-c", "docs/c.md", "C", now, "h3")

	_ = db.InsertEdgeWithProps(ctx, database, "node-a", "node-b", "links_to", "EXTRACTED", 1.0)
	_ = db.InsertEdgeWithProps(ctx, database, "node-c", "node-b", "links_to", "EXTRACTED", 1.0)
	database.Close()

	// 1. Direcionado: de A para C não deve encontrar caminho
	var dirBuf bytes.Buffer
	err = runPathCommand(ctx, "test-repo", []string{"--db", dbFile, "--json", "node-a", "node-c"}, &dirBuf)
	if err != nil {
		t.Fatalf("falha: %v", err)
	}
	var dirRes graph.PathResult
	_ = json.Unmarshal(dirBuf.Bytes(), &dirRes)
	if dirRes.Found {
		t.Errorf("esperava found=false no modo direcionado")
	}

	// 2. Não-direcionado (--undirected): deve encontrar caminho A -> B -> C
	var undirBuf bytes.Buffer
	err = runPathCommand(ctx, "test-repo", []string{"node-a", "node-c", "--db", dbFile, "--undirected", "--mode", "hops", "--json"}, &undirBuf)
	if err != nil {
		t.Fatalf("falha no modo undirected: %v", err)
	}
	var undirRes graph.PathResult
	if err := json.Unmarshal(undirBuf.Bytes(), &undirRes); err != nil {
		t.Fatalf("falha unmarshal: %v", err)
	}
	if !undirRes.Found {
		t.Fatalf("esperava found=true no modo undirected")
	}
	if undirRes.Hops != 2 {
		t.Errorf("esperava 2 saltos, obteve %d", undirRes.Hops)
	}
	if undirRes.CostMode != graph.CostModeHops {
		t.Errorf("esperava CostMode hops, obteve %s", undirRes.CostMode)
	}
}
