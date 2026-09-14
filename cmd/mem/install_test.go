package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallCLI_DryRun(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	tempDir := t.TempDir()

	err := runInstallCommand(ctx, "test-repo", []string{"--dry-run", "--workspace", "--dir", tempDir}, &buf)
	if err != nil {
		t.Fatalf("falha ao executar mem install --dry-run: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "[DRY-RUN]") {
		t.Errorf("esperava indicador [DRY-RUN] na saída:\n%s", out)
	}
	if !strings.Contains(out, "Cursor IDE (Workspace)") {
		t.Errorf("esperava menção ao Cursor Workspace na saída:\n%s", out)
	}

	// Garantir que nenhum arquivo foi criado no disco
	cursorFile := filepath.Join(tempDir, ".cursor", "mcp.json")
	if _, err := os.Stat(cursorFile); !os.IsNotExist(err) {
		t.Errorf("arquivo não deveria ter sido criado durante dry-run: %s", cursorFile)
	}
}

func TestInstallCLI_WorkspaceExecution(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	tempDir := t.TempDir()

	err := runInstallCommand(ctx, "test-repo", []string{
		"--workspace",
		"--target", "cursor",
		"--dir", tempDir,
		"--command", "C:/custom/mem.exe",
		"--db", "custom.db",
	}, &buf)
	if err != nil {
		t.Fatalf("falha ao executar mem install no workspace: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "[INSTALADO]") {
		t.Errorf("esperava [INSTALADO] na saída:\n%s", out)
	}

	cursorFile := filepath.Join(tempDir, ".cursor", "mcp.json")
	data, err := os.ReadFile(cursorFile)
	if err != nil {
		t.Fatalf("falha ao ler arquivo gerado: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON inválido gerado: %v", err)
	}

	servers, ok := parsed["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatalf("mcpServers ausente no JSON")
	}

	memSrv, ok := servers["my-memory"].(map[string]interface{})
	if !ok {
		t.Fatalf("my-memory ausente no mcpServers")
	}

	if memSrv["command"] != "C:/custom/mem.exe" {
		t.Errorf("command incorreto: %v", memSrv["command"])
	}
}

func TestRearrangeInstallArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "mixed flags with target and dry-run",
			input:    []string{"--dry-run", "--target", "cursor", "--workspace"},
			expected: []string{"--dry-run", "--target", "cursor", "--workspace"},
		},
		{
			name:     "flags with equals",
			input:    []string{"--target=vscode", "--dir=/tmp", "--force"},
			expected: []string{"--target=vscode", "--dir=/tmp", "--force"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rearrangeInstallArgs(tt.input)
			if len(got) != len(tt.expected) {
				t.Fatalf("tamanho diferente: obteve %d, esperava %d", len(got), len(tt.expected))
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("índice %d: obteve '%s', esperava '%s'", i, got[i], tt.expected[i])
				}
			}
		})
	}
}
