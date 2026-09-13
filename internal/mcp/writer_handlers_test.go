package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMCPWriterTools_List(t *testing.T) {
	srv := NewServer("test-server", "1.0.0", nil, nil, nil)
	tools := srv.GetTools()

	expectedTools := []string{
		"memory_search",
		"memory_get_neighbors",
		"memory_export_canvas",
		"memory_get_hubs",
		"memory_get_insights",
		"memory_doctor",
		"memory_write_note",
		"memory_append_section",
		"memory_compile_note",
	}

	toolMap := make(map[string]bool)
	for _, tool := range tools {
		toolMap[tool.Name] = true
	}

	for _, expected := range expectedTools {
		if !toolMap[expected] {
			t.Errorf("ferramenta esperada '%s' não encontrada na listagem de tools do MCP", expected)
		}
	}
}

func TestMCPWriterTools_Execute(t *testing.T) {
	ctx := context.Background()
	tempVault := t.TempDir()

	inBuf := &bytes.Buffer{}
	outBuf := &bytes.Buffer{}
	srv := NewServer("test-server", "1.0.0", inBuf, outBuf, nil)

	// Configura o motor com defaultVaultRoot apontando para tempVault
	srv.SetCompilerEngine(nil, func(ctx context.Context, params SearchParams) ([]SearchResult, error) {
		return []SearchResult{
			{
				DocumentID: "concepts/security.md",
				Content:    "Detalhamento de criptografia e hashing seguro.",
				Score:      0.92,
			},
		}, nil
	}, tempVault)

	// 1. Teste memory_write_note
	writeArgs, _ := json.Marshal(map[string]any{
		"path":      "concepts/api-design.md",
		"title":     "Design de APIs",
		"content":   "Diretrizes para endpoints REST e RPC com [[concepts/security.md]].",
		"tags":      []string{"api", "rest"},
		"note_type": "concept",
		"relations": []map[string]string{
			{"target": "SecurityGuideline", "relation": "depends_on"},
		},
	})

	res, err := srv.handleToolsCall(ctx, mustRawMessage(map[string]any{
		"name":      "memory_write_note",
		"arguments": json.RawMessage(writeArgs),
	}))
	if err != nil {
		t.Fatalf("erro ao executar memory_write_note: %v", err)
	}

	callRes, ok := res.(CallToolResult)
	if !ok {
		t.Fatalf("tipo de resultado inesperado: %T", res)
	}
	if callRes.IsError {
		t.Fatalf("chamada retornou erro: %s", callRes.Content[0].Text)
	}
	if !strings.Contains(callRes.Content[0].Text, "Design de APIs") {
		t.Errorf("resposta deve conter o título: %s", callRes.Content[0].Text)
	}

	// Verifica se arquivo foi criado no disco
	createdFile := filepath.Join(tempVault, "concepts", "api-design.md")
	data, err := os.ReadFile(createdFile)
	if err != nil {
		t.Fatalf("arquivo não encontrado no disco: %v", err)
	}
	if !strings.Contains(string(data), "[[rel:depends_on:SecurityGuideline]]") {
		t.Errorf("arquivo deve conter relação tipada: %s", string(data))
	}

	// 2. Teste memory_append_section
	appendArgs, _ := json.Marshal(map[string]any{
		"path":    "concepts/api-design.md",
		"heading": "## Versionamento",
		"content": "Utilizar versionamento por URL (/v1/...) ou cabeçalhos.",
	})

	resAppend, err := srv.handleToolsCall(ctx, mustRawMessage(map[string]any{
		"name":      "memory_append_section",
		"arguments": json.RawMessage(appendArgs),
	}))
	if err != nil {
		t.Fatalf("erro ao executar memory_append_section: %v", err)
	}
	callResAppend := resAppend.(CallToolResult)
	if callResAppend.IsError {
		t.Fatalf("append retornou erro: %s", callResAppend.Content[0].Text)
	}

	dataAfterAppend, _ := os.ReadFile(createdFile)
	if !strings.Contains(string(dataAfterAppend), "## Versionamento") {
		t.Errorf("arquivo não contém nova seção: %s", string(dataAfterAppend))
	}

	// 3. Teste memory_compile_note (Compile-not-Retrieve)
	compileArgs, _ := json.Marshal(map[string]any{
		"topic":       "segurança e criptografia",
		"target_path": "syntheses/security-summary.md",
		"title":       "Síntese de Segurança",
		"tags":        []string{"crypto"},
	})

	resCompile, err := srv.handleToolsCall(ctx, mustRawMessage(map[string]any{
		"name":      "memory_compile_note",
		"arguments": json.RawMessage(compileArgs),
	}))
	if err != nil {
		t.Fatalf("erro ao executar memory_compile_note: %v", err)
	}
	callResCompile := resCompile.(CallToolResult)
	if callResCompile.IsError {
		t.Fatalf("compile retornou erro: %s", callResCompile.Content[0].Text)
	}

	compFile := filepath.Join(tempVault, "syntheses", "security-summary.md")
	dataComp, err := os.ReadFile(compFile)
	if err != nil {
		t.Fatalf("arquivo compilado não encontrado: %v", err)
	}
	if !strings.Contains(string(dataComp), "[[rel:derived_from:concepts/security.md]]") {
		t.Errorf("arquivo compilado deve conter backlinks para fontes: %s", string(dataComp))
	}

	// 4. Teste de path traversal
	evilArgs, _ := json.Marshal(map[string]any{
		"path":    "../../evil.md",
		"content": "Tentativa maliciosa",
	})
	resEvil, _ := srv.handleToolsCall(ctx, mustRawMessage(map[string]any{
		"name":      "memory_write_note",
		"arguments": json.RawMessage(evilArgs),
	}))
	evilResult := resEvil.(CallToolResult)
	if !evilResult.IsError {
		t.Fatalf("deveria retornar erro para path traversal")
	}
}

func mustRawMessage(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
