package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/config"
)

type mockMCPLauncher struct {
	lastCmd  string
	lastArgs []string
	calls    int
}

func (m *mockMCPLauncher) Launch(command string, args ...string) error {
	m.calls++
	m.lastCmd = command
	m.lastArgs = args
	return nil
}

func TestNewMemoryOpenNodeHandler_MissingNode(t *testing.T) {
	ctx := context.Background()
	handler := NewMemoryOpenNodeHandler(nil, "", "", nil)

	_, err := handler(ctx, []byte(`{}`))
	if err == nil {
		t.Errorf("esperava erro de validação para parâmetro node_id ausente")
	}
}

func TestNewMemoryOpenNodeHandler_LinksOnly(t *testing.T) {
	ctx := context.Background()
	mock := &mockMCPLauncher{}

	resolveFn := func(ctx context.Context, repo, nodeID string) (string, error) {
		if nodeID == "minha-nota" {
			return "docs/minha-nota.md", nil
		}
		return nodeID, nil
	}

	handler := NewMemoryOpenNodeHandler(resolveFn, "/repo/vault", "MyVault", mock)

	args := map[string]any{
		"node_id": "minha-nota",
		"app":     "vscode",
		"line":    35,
		"action":  "links_only",
	}
	argsJSON, _ := json.Marshal(args)

	res, err := handler(ctx, argsJSON)
	if err != nil {
		t.Fatalf("erro inesperado no handler: %v", err)
	}

	callRes, ok := res.(CallToolResult)
	if !ok || len(callRes.Content) == 0 {
		t.Fatalf("esperava CallToolResult com conteúdo válido, obteve: %+v", res)
	}

	text := callRes.Content[0].Text
	if !strings.Contains(text, "docs/minha-nota.md") {
		t.Errorf("esperava caminho resolvido docs/minha-nota.md, obteve:\n%s", text)
	}
	if !strings.Contains(text, "vscode://") {
		t.Errorf("esperava URI do vscode no resultado, obteve:\n%s", text)
	}
	if !strings.Contains(text, ":35") {
		t.Errorf("esperava número de linha :35 no resultado, obteve:\n%s", text)
	}
	if !strings.Contains(text, "obsidian://open?") {
		t.Errorf("esperava URI do obsidian no resultado, obteve:\n%s", text)
	}
	if mock.calls != 0 {
		t.Errorf("launcher não deve ser chamado em action=links_only, obteve %d chamadas", mock.calls)
	}
}

func TestNewMemoryOpenNodeHandler_OpenAction(t *testing.T) {
	ctx := context.Background()
	mock := &mockMCPLauncher{}

	handler := NewMemoryOpenNodeHandler(nil, "/repo/vault", "MyVault", mock)

	args := map[string]any{
		"node_id": "docs/guia.md",
		"app":     "obsidian",
		"action":  "open",
	}
	argsJSON, _ := json.Marshal(args)

	res, err := handler(ctx, argsJSON)
	if err != nil {
		t.Fatalf("erro inesperado ao abrir: %v", err)
	}

	callRes := res.(CallToolResult)
	text := callRes.Content[0].Text
	if !strings.Contains(text, "com sucesso") {
		t.Errorf("esperava confirmação de sucesso, obteve:\n%s", text)
	}
	if mock.calls != 1 {
		t.Errorf("esperava 1 chamada no launcher para action=open, obteve %d", mock.calls)
	}
}

func TestServer_OpenToolRegistered(t *testing.T) {
	s := NewServer("test-srv", "1.0.0", strings.NewReader(""), os.Stdout, os.Stderr)

	tools := s.GetTools()
	found := false
	for _, tool := range tools {
		if tool.Name == ToolMemoryOpenNode.Name {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ferramenta %s não encontrada registrada no servidor MCP", ToolMemoryOpenNode.Name)
	}
}

func TestNewMemoryOpenNodeHandler_FederatedNode(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	centralDir := filepath.Join(tmpDir, "central_vault")
	if err := os.MkdirAll(filepath.Join(centralDir, "standards"), 0755); err != nil {
		t.Fatalf("erro criando diretório: %v", err)
	}
	notePath := filepath.Join(centralDir, "standards", "oauth2.md")
	if err := os.WriteFile(notePath, []byte("# OAuth2 Standards"), 0644); err != nil {
		t.Fatalf("erro criando nota: %v", err)
	}

	globalDir := filepath.Join(tmpDir, "global")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatalf("erro criando diretório global: %v", err)
	}
	t.Setenv(config.GlobalConfigDirEnv, globalDir)

	globalYAML := fmt.Sprintf(`
central_vault:
  path: "%s"
  vault_name: "CentralVault"
`, filepath.ToSlash(centralDir))
	if err := os.WriteFile(filepath.Join(globalDir, "config.yaml"), []byte(globalYAML), 0644); err != nil {
		t.Fatalf("erro gravando config global: %v", err)
	}

	mock := &mockMCPLauncher{}
	handler := NewMemoryOpenNodeHandler(nil, "", "", mock)

	args := map[string]any{
		"node_id": "memory://central/standards/oauth2",
		"app":     "obsidian",
		"action":  "open",
	}
	argsJSON, _ := json.Marshal(args)

	res, err := handler(ctx, argsJSON)
	if err != nil {
		t.Fatalf("erro inesperado ao abrir nó federado: %v", err)
	}

	callRes := res.(CallToolResult)
	text := callRes.Content[0].Text
	if !strings.Contains(text, "Federado (cross-vault)") {
		t.Errorf("esperava indicador de tipo federado, obteve:\n%s", text)
	}
	if !strings.Contains(text, "Cofre Alvo**: `CentralVault`") {
		t.Errorf("esperava Cofre Alvo CentralVault, obteve:\n%s", text)
	}
	if !strings.Contains(text, "obsidian://open?vault=CentralVault") {
		t.Errorf("esperava URI do Obsidian para CentralVault, obteve:\n%s", text)
	}
	if mock.calls != 1 {
		t.Errorf("esperava 1 chamada no launcher, obteve %d", mock.calls)
	}
}

