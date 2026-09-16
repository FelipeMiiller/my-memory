package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/graph"
)

func TestRearrangePackArgs(t *testing.T) {
	args := []string{"auth.md", "--depth", "3", "--max-tokens", "2500", "--json"}
	rearranged := rearrangePackArgs(args)

	expectedPrefix := []string{"--depth", "3", "--max-tokens", "2500", "--json"}
	for i, exp := range expectedPrefix {
		if rearranged[i] != exp {
			t.Errorf("rearranged[%d] = %s, esperado %s", i, rearranged[i], exp)
		}
	}
	if rearranged[len(rearranged)-1] != "auth.md" {
		t.Errorf("último argumento = %s, esperado 'auth.md'", rearranged[len(rearranged)-1])
	}
}

func TestRunPackCommand_MissingRoot(t *testing.T) {
	var buf bytes.Buffer
	err := runPackCommand(context.Background(), "repo", []string{}, &buf)
	if err == nil {
		t.Fatalf("esperava erro por falta de nó raiz, retornou nil")
	}
	if !strings.Contains(err.Error(), "nó raiz obrigatório") {
		t.Errorf("erro inesperado: %v", err)
	}
}

func TestRunPackCommand_SuccessAndJSON(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_cli_pack.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		if strings.Contains(err.Error(), "no such module: fts5") {
			t.Skip("Pulando teste: ambiente sem FTS5")
		}
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	now := time.Now().Unix()

	_ = db.InsertDocumentWithMeta(ctx, database, "doc-root", "docs/root.md", "Raiz Arquitetura", now, "h1", "Resumo Raiz", "resource")
	_ = db.InsertDocumentWithMeta(ctx, database, "doc-sub", "docs/sub.md", "Subsistema", now, "h2", "Resumo Sub", "resource")
	_ = db.InsertChunk(ctx, database, "c1", "doc-root", "Conteúdo estrutural da raiz.", 0, nil)
	_ = db.InsertChunk(ctx, database, "c2", "doc-sub", "Conteúdo do subsistema conectado.", 0, nil)
	_ = db.InsertEdgeWithProps(ctx, database, "doc-root", "doc-sub", "depends_on", "EXTRACTED", 1.0)

	// 1. Teste de execução normal (Markdown)
	var buf bytes.Buffer
	args := []string{"docs/root.md", "--db", dbPath, "--depth", "1", "--max-tokens", "2000"}
	if err := runPackCommand(ctx, "repo", args, &buf); err != nil {
		t.Fatalf("runPackCommand falhou: %v", err)
	}

	outStr := buf.String()
	if !strings.Contains(outStr, "Pacote de Contexto: Raiz Arquitetura") {
		t.Errorf("Markdown não contém título esperado: %s", outStr)
	}
	if !strings.Contains(outStr, "graph TD") {
		t.Errorf("Markdown não contém topologia Mermaid")
	}

	// 2. Teste de saída JSON
	var jsonBuf bytes.Buffer
	jsonArgs := []string{"docs/root.md", "--db", dbPath, "--json"}
	if err := runPackCommand(ctx, "repo", jsonArgs, &jsonBuf); err != nil {
		t.Fatalf("runPackCommand --json falhou: %v", err)
	}

	var res graph.PackResult
	if err := json.Unmarshal(jsonBuf.Bytes(), &res); err != nil {
		t.Fatalf("falha ao deserializar JSON gerado: %v", err)
	}
	if res.RootID != "doc-root" {
		t.Errorf("JSON RootID = %s, esperado 'doc-root'", res.RootID)
	}
	if res.CoreCount < 2 {
		t.Errorf("JSON CoreCount = %d, esperado >= 2", res.CoreCount)
	}

	// 3. Teste de gravação em arquivo --out
	outFile := filepath.Join(t.TempDir(), "subgraph_bundle.md")
	var outBuf bytes.Buffer
	fileArgs := []string{"docs/root.md", "--db", dbPath, "--out", outFile}
	if err := runPackCommand(ctx, "repo", fileArgs, &outBuf); err != nil {
		t.Fatalf("runPackCommand --out falhou: %v", err)
	}

	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("falha ao ler arquivo gerado: %v", err)
	}
	if !strings.Contains(string(content), "Pacote de Contexto: Raiz Arquitetura") {
		t.Errorf("conteúdo do arquivo salvo inesperado: %s", string(content))
	}
	if !strings.Contains(outBuf.String(), "Arquivo Salvo:") {
		t.Errorf("resumo do terminal não indicou arquivo salvo")
	}
}
