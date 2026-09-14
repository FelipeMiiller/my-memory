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

func TestInspectCLI_MissingTargetNode(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer

	err := runInspectCommand(ctx, "test-repo", []string{}, &buf)
	if err == nil {
		t.Fatal("esperava erro ao omitir nó alvo, obteve nil")
	}
	if !strings.Contains(err.Error(), "identificador do nó alvo é obrigatório") {
		t.Errorf("mensagem de erro inesperada: %v", err)
	}
}

func TestInspectCLI_NonExistentNode(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	dbFile := filepath.Join(t.TempDir(), "inspect_test.db")

	database, err := db.InitDB(dbFile)
	if err != nil {
		if strings.Contains(err.Error(), "no such module: fts5") {
			t.Skip("ambiente sem FTS5")
		}
		t.Fatalf("falha ao inicializar SQLite: %v", err)
	}
	database.Close()

	err = runInspectCommand(ctx, "test-repo", []string{"--db", dbFile, "non_existent_node"}, &buf)
	if err == nil {
		t.Fatal("esperava erro para nó inexistente, obteve nil")
	}
}

func TestInspectCLI_TextAndJSONOutput(t *testing.T) {
	ctx := context.Background()
	dbFile := filepath.Join(t.TempDir(), "inspect_cli.db")

	database, err := db.InitDB(dbFile)
	if err != nil {
		if strings.Contains(err.Error(), "no such module: fts5") {
			t.Skip("ambiente sem FTS5")
		}
		t.Fatalf("falha ao inicializar SQLite: %v", err)
	}

	now := time.Now().Unix()
	_ = db.InsertDocument(ctx, database, "adr-024", "docs/adr/024.md", "Visualização 3 Colunas", now, "h24")
	_ = db.InsertChunk(ctx, database, "chk-1", "adr-024", "Conteúdo arquitetural da visualização cirúrgica em 3 colunas.", 0, nil)

	_ = db.InsertDocument(ctx, database, "caller-cli", "cmd/mem/cli.md", "Caller CLI", now, "h-cli")
	_ = db.InsertDocument(ctx, database, "callee-mcp", "internal/mcp/mcp.md", "Callee MCP", now, "h-mcp")

	_ = db.InsertEdgeWithProps(ctx, database, "caller-cli", "adr-024", "implements", "EXTRACTED", 1.0)
	_ = db.InsertEdgeWithProps(ctx, database, "adr-024", "callee-mcp", "depends_on", "EXTRACTED", 1.0)
	database.Close()

	// 1. Teste de Saída Textual
	var textBuf bytes.Buffer
	err = runInspectCommand(ctx, "test-repo", []string{"--db", dbFile, "adr-024"}, &textBuf)
	if err != nil {
		t.Fatalf("falha ao executar mem inspect texto: %v", err)
	}

	textOut := textBuf.String()
	if !strings.Contains(textOut, "Visualização Cirúrgica de Nó (Triptych Inspector)") {
		t.Errorf("cabeçalho não encontrado na saída textual:\n%s", textOut)
	}
	if !strings.Contains(textOut, "Visualização 3 Colunas") {
		t.Errorf("título central não encontrado na saída:\n%s", textOut)
	}
	if !strings.Contains(textOut, "caller-cli") {
		t.Errorf("inbound link caller-cli não encontrado na saída:\n%s", textOut)
	}
	if !strings.Contains(textOut, "callee-mcp") {
		t.Errorf("outbound link callee-mcp não encontrado na saída:\n%s", textOut)
	}

	// 2. Teste de Saída JSON
	var jsonBuf bytes.Buffer
	err = runInspectCommand(ctx, "test-repo", []string{"--db", dbFile, "--json", "adr-024"}, &jsonBuf)
	if err != nil {
		t.Fatalf("falha ao executar mem inspect --json: %v", err)
	}

	var view graph.TriptychView
	if err := json.Unmarshal(jsonBuf.Bytes(), &view); err != nil {
		t.Fatalf("falha ao desserializar JSON de mem inspect: %v\nJSON: %s", err, jsonBuf.String())
	}

	if view.Target.ID != "adr-024" {
		t.Errorf("esperava Target.ID 'adr-024', obteve '%s'", view.Target.ID)
	}
	if view.TotalInbound != 1 {
		t.Errorf("esperava TotalInbound 1, obteve %d", view.TotalInbound)
	}
	if view.TotalOutbound != 1 {
		t.Errorf("esperava TotalOutbound 1, obteve %d", view.TotalOutbound)
	}
}

func TestInspectCLI_FlagRearranging(t *testing.T) {
	ctx := context.Background()
	dbFile := filepath.Join(t.TempDir(), "inspect_rearrange.db")

	database, err := db.InitDB(dbFile)
	if err != nil {
		if strings.Contains(err.Error(), "no such module: fts5") {
			t.Skip("ambiente sem FTS5")
		}
		t.Fatalf("falha ao inicializar SQLite: %v", err)
	}

	now := time.Now().Unix()
	_ = db.InsertDocument(ctx, database, "target-rearrange", "docs/rearrange.md", "Rearrange Target", now, "hr")
	database.Close()

	// Testa passagem de flag --json DEPOIS do identificador do nó
	var buf bytes.Buffer
	err = runInspectCommand(ctx, "test-repo", []string{"--db", dbFile, "target-rearrange", "--json"}, &buf)
	if err != nil {
		t.Fatalf("falha ao executar inspect com flag posicional posterior: %v", err)
	}

	var res graph.TriptychView
	if err := json.Unmarshal(buf.Bytes(), &res); err != nil {
		t.Fatalf("esperado JSON válido mesmo com --json após o nome do nó, obteve: %s", buf.String())
	}
	if res.Target.ID != "target-rearrange" {
		t.Errorf("esperava ID 'target-rearrange', obteve '%s'", res.Target.ID)
	}
}

func TestInspectCLI_FullFlag(t *testing.T) {
	ctx := context.Background()
	dbFile := filepath.Join(t.TempDir(), "inspect_full.db")

	database, err := db.InitDB(dbFile)
	if err != nil {
		if strings.Contains(err.Error(), "no such module: fts5") {
			t.Skip("ambiente sem FTS5")
		}
		t.Fatalf("falha ao inicializar SQLite: %v", err)
	}

	longContent := strings.Repeat("Texto longo para teste de conteúdo completo sem truncamento. ", 30)
	now := time.Now().Unix()
	_ = db.InsertDocument(ctx, database, "long-doc", "docs/long.md", "Long Doc", now, "hl")
	_ = db.InsertChunk(ctx, database, "chk-long", "long-doc", longContent, 0, nil)
	database.Close()

	var buf bytes.Buffer
	err = runInspectCommand(ctx, "test-repo", []string{"--db", dbFile, "long-doc", "--full", "--json"}, &buf)
	if err != nil {
		t.Fatalf("falha ao executar inspect com --full: %v", err)
	}

	var res graph.TriptychView
	if err := json.Unmarshal(buf.Bytes(), &res); err != nil {
		t.Fatalf("falha ao parsear json: %v", err)
	}

	if strings.Contains(res.Target.ContentPreview, "[truncado]") {
		t.Errorf("com flag --full, o conteúdo não deveria ser truncado")
	}
}

