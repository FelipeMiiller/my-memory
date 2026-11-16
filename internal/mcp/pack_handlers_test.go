package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/graph"
)

func TestMemoryPackContextTool_Registered(t *testing.T) {
	srv := NewServer("test", "1.0", &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{})
	tools := srv.GetTools()

	found := false
	for _, tool := range tools {
		if tool.Name == ToolMemoryPackContext.Name {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("Ferramenta '%s' não foi registrada automaticamente no servidor", ToolMemoryPackContext.Name)
	}
}

func TestMemoryPackContextTool_MissingRoot(t *testing.T) {
	handler := NewMemoryPackContextHandler(func(ctx context.Context, repo, rootQuery string, opts graph.PackOptions) (*graph.PackResult, error) {
		return &graph.PackResult{}, nil
	})

	_, err := handler(context.Background(), json.RawMessage(`{"root_node": ""}`))
	if err == nil {
		t.Fatalf("Esperava erro ao omitir 'root_node', retornou nil")
	}
	mcpErr, ok := err.(*Error)
	if !ok || mcpErr.Code != CodeInvalidParams {
		t.Errorf("Código de erro inesperado: %v", err)
	}
}

func TestMemoryPackContextTool_Success(t *testing.T) {
	mockFn := func(ctx context.Context, repo, rootQuery string, opts graph.PackOptions) (*graph.PackResult, error) {
		if rootQuery != "docs/root.md" {
			return nil, fmt.Errorf("nó inesperado: %s", rootQuery)
		}
		if opts.MaxDepth != 3 {
			t.Errorf("MaxDepth = %d, esperado 3", opts.MaxDepth)
		}
		if opts.MaxTokens != 2500 {
			t.Errorf("MaxTokens = %d, esperado 2500", opts.MaxTokens)
		}
		return &graph.PackResult{
			RootID:       "docs/root.md",
			RootTitle:    "Raiz Teste",
			TotalTokens:  1200,
			MaxTokens:    opts.MaxTokens,
			CoreCount:    2,
			FringeCount:  1,
			OmittedCount: 0,
			Markdown:     "# 📦 Pacote de Contexto: Raiz Teste\n\nConteúdo consolidado.",
		}, nil
	}

	srv := NewServer("test", "1.0", &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{})
	srv.SetPackHandler(mockFn)

	callArgs, _ := json.Marshal(map[string]any{
		"name": "memory_pack_context",
		"arguments": map[string]any{
			"root_node":  "docs/root.md",
			"max_depth":  3,
			"max_tokens": 2500,
			"direction":  "both",
		},
	})

	res, err := srv.handleToolsCall(context.Background(), callArgs)
	if err != nil {
		t.Fatalf("handleToolsCall falhou: %v", err)
	}

	callRes, ok := res.(CallToolResult)
	if !ok || len(callRes.Content) == 0 {
		t.Fatalf("Resultado inválido retornado: %+v", res)
	}

	txt := callRes.Content[0].Text
	if !strings.Contains(txt, "Pacote de Contexto: Raiz Teste") {
		t.Errorf("Texto Markdown gerado inesperado: %s", txt)
	}
}

// TestMemoryPackContextTool_DefaultsVemDaFonteUnica prova FPB-14: sem os parâmetros
// opcionais, as opções passadas ao motor são iguais a graph.DefaultPackOptions()
// campo a campo (MaxDepth 2, MaxTokens 4000, Direction "both",
// IncludeFringeAbstracts true).
func TestMemoryPackContextTool_DefaultsVemDaFonteUnica(t *testing.T) {
	var captured graph.PackOptions
	handler := NewMemoryPackContextHandler(func(ctx context.Context, repo, rootQuery string, opts graph.PackOptions) (*graph.PackResult, error) {
		captured = opts
		return &graph.PackResult{Markdown: "# 📦 Pacote de Contexto: Raiz Teste"}, nil
	})

	if _, err := handler(context.Background(), json.RawMessage(`{"root_node": "docs/root.md"}`)); err != nil {
		t.Fatalf("handler falhou: %v", err)
	}

	defaults := graph.DefaultPackOptions()
	if captured.MaxDepth != defaults.MaxDepth {
		t.Errorf("MaxDepth = %d, esperado o default compartilhado %d", captured.MaxDepth, defaults.MaxDepth)
	}
	if captured.MaxTokens != defaults.MaxTokens {
		t.Errorf("MaxTokens = %d, esperado o default compartilhado %d", captured.MaxTokens, defaults.MaxTokens)
	}
	if captured.Direction != defaults.Direction {
		t.Errorf("Direction = %q, esperado o default compartilhado %q", captured.Direction, defaults.Direction)
	}
	if captured.IncludeFringeAbstracts != defaults.IncludeFringeAbstracts {
		t.Errorf("IncludeFringeAbstracts = %v, esperado o default compartilhado %v", captured.IncludeFringeAbstracts, defaults.IncludeFringeAbstracts)
	}
}

func TestMemoryPackContextTool_PackError(t *testing.T) {
	mockFn := func(ctx context.Context, repo, rootQuery string, opts graph.PackOptions) (*graph.PackResult, error) {
		return nil, fmt.Errorf("falha interna simulada")
	}

	srv := NewServer("test", "1.0", &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{})
	srv.SetPackHandler(mockFn)

	callArgs, _ := json.Marshal(map[string]any{
		"name": "memory_pack_context",
		"arguments": map[string]any{
			"root_node": "erro",
		},
	})

	_, err := srv.handleToolsCall(context.Background(), callArgs)
	if err == nil {
		t.Fatalf("esperava erro de execução, retornou nil")
	}
	if !strings.Contains(err.Error(), "falha interna simulada") {
		t.Errorf("mensagem de erro inesperada: %v", err)
	}
}
