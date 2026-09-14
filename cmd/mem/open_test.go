package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/db"
)

type mockOpenLauncher struct {
	lastCmd  string
	lastArgs []string
	calls    int
}

func (m *mockOpenLauncher) Launch(command string, args ...string) error {
	m.calls++
	m.lastCmd = command
	m.lastArgs = args
	return nil
}

func TestRearrangeOpenArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "flags after target node",
			input:    []string{"my-note.md", "--app", "vscode", "--line", "25"},
			expected: []string{"--app", "vscode", "--line", "25", "my-note.md"},
		},
		{
			name:     "flags before target node",
			input:    []string{"--app", "obsidian", "--dry-run", "my-note.md"},
			expected: []string{"--app", "obsidian", "--dry-run", "my-note.md"},
		},
		{
			name:     "mixed with equals syntax",
			input:    []string{"doc.md", "--app=vscode", "--line=10", "--json"},
			expected: []string{"--app=vscode", "--line=10", "--json", "doc.md"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := rearrangeOpenArgs(tc.input)
			if len(result) != len(tc.expected) {
				t.Fatalf("tamanho diferente: obteve %v, esperava %v", result, tc.expected)
			}
			for i := range result {
				if result[i] != tc.expected[i] {
					t.Errorf("pos %d: obteve %s, esperava %s", i, result[i], tc.expected[i])
				}
			}
		})
	}
}

func TestRunOpenCommand_MissingTarget(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	mock := &mockOpenLauncher{}

	err := runOpenCommand(ctx, "repo", []string{}, &buf, mock)
	if err == nil {
		t.Errorf("esperava erro ao omitir nó alvo")
	}
}

func TestRunOpenCommand_DryRunAndJSON(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// Cria arquivo físico de teste
	docPath := filepath.Join(tmpDir, "concept.md")
	if err := os.WriteFile(docPath, []byte("# Concept"), 0644); err != nil {
		t.Fatalf("erro criando arquivo: %v", err)
	}

	mock := &mockOpenLauncher{}

	// 1. Teste Dry-Run
	var dryBuf bytes.Buffer
	err := runOpenCommand(ctx, "repo", []string{docPath, "--dry-run", "--app", "vscode", "--line", "42"}, &dryBuf, mock)
	if err != nil {
		t.Fatalf("erro no dry-run: %v", err)
	}
	outStr := dryBuf.String()
	if !strings.Contains(outStr, "[dry-run]") {
		t.Errorf("esperava saída contendo [dry-run], obteve:\n%s", outStr)
	}
	if !strings.Contains(outStr, "vscode://") {
		t.Errorf("esperava URI do vscode no dry-run, obteve:\n%s", outStr)
	}
	if !strings.Contains(outStr, ":42") {
		t.Errorf("esperava linha :42 no dry-run, obteve:\n%s", outStr)
	}
	if mock.calls != 0 {
		t.Errorf("launcher não deve ser chamado em modo dry-run, obteve %d chamadas", mock.calls)
	}

	// 2. Teste JSON Output
	var jsonBuf bytes.Buffer
	err = runOpenCommand(ctx, "repo", []string{docPath, "--json", "--app", "obsidian"}, &jsonBuf, mock)
	if err != nil {
		t.Fatalf("erro no json: %v", err)
	}

	var jsonRes OpenOutput
	if err := json.Unmarshal(jsonBuf.Bytes(), &jsonRes); err != nil {
		t.Fatalf("saída JSON inválida: %v", err)
	}
	if jsonRes.Node != docPath {
		t.Errorf("esperava nó %s, obteve %s", docPath, jsonRes.Node)
	}
	if !strings.HasPrefix(jsonRes.TargetURI, "obsidian://") {
		t.Errorf("esperava TargetURI do Obsidian, obteve %s", jsonRes.TargetURI)
	}
	if jsonRes.Links.Obsidian == "" || jsonRes.Links.VSCode == "" || jsonRes.Links.File == "" {
		t.Errorf("esperava todos os links populados, obteve %+v", jsonRes.Links)
	}
}

func TestRunOpenCommand_WithDatabaseResolution(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_open.db")

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("erro ao criar banco SQLite de teste: %v", err)
	}

	// Insere documento no banco
	_, err = database.ExecContext(ctx, `
		INSERT INTO documents (id, path, title, updated_at)
		VALUES ('docs/adr/001-test.md', 'docs/adr/001-test.md', 'ADR Teste', 1000)
	`)
	if err != nil {
		database.Close()
		t.Fatalf("erro inserindo documento: %v", err)
	}
	database.Close()

	mock := &mockOpenLauncher{}
	var buf bytes.Buffer

	// Executa abertura buscando por título ou slug cadastrado no banco
	err = runOpenCommand(ctx, "test-repo", []string{"ADR Teste", "--db", dbPath, "--app", "vscode"}, &buf, mock)
	if err != nil {
		t.Fatalf("erro ao abrir nó resolvido por título: %v", err)
	}

	if mock.calls != 1 {
		t.Fatalf("esperava 1 chamada no launcher, obteve %d", mock.calls)
	}

	outStr := buf.String()
	if !strings.Contains(outStr, "🚀 Abrindo no vscode") {
		t.Errorf("esperava mensagem de abertura no terminal, obteve:\n%s", outStr)
	}
	if !strings.Contains(outStr, "docs/adr/001-test.md") {
		t.Errorf("esperava caminho canônico resolvido, obteve:\n%s", outStr)
	}
}

func TestRunOpenCommand_NonExistentNode(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "empty.db")
	database, _ := db.InitDB(dbPath)
	database.Close()

	mock := &mockOpenLauncher{}
	var buf bytes.Buffer

	err := runOpenCommand(ctx, "repo", []string{"nao_existe_xyz_12345", "--db", dbPath}, &buf, mock)
	if err == nil {
		t.Errorf("esperava erro para nó inexistente no grafo e disco")
	}
}
