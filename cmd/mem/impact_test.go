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

func TestImpactCLI_MissingTargetNode(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer

	err := runImpactCommand(ctx, "test-repo", []string{}, &buf)
	if err == nil {
		t.Fatal("esperava erro ao omitir nó alvo, obteve nil")
	}
	if !strings.Contains(err.Error(), "identificador do nó alvo é obrigatório") {
		t.Errorf("mensagem de erro inesperada: %v", err)
	}
}

func TestImpactCLI_NonExistentNode(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	dbFile := filepath.Join(t.TempDir(), "impact_test.db")

	database, err := db.InitDB(dbFile)
	if err != nil {
		if strings.Contains(err.Error(), "no such module: fts5") {
			t.Skip("ambiente sem FTS5")
		}
		t.Fatalf("falha ao inicializar SQLite: %v", err)
	}
	database.Close()

	err = runImpactCommand(ctx, "test-repo", []string{"--db", dbFile, "non_existent_node"}, &buf)
	if err == nil {
		t.Fatal("esperava erro para nó inexistente, obteve nil")
	}
}

func TestImpactCLI_TextOutputAndJSON(t *testing.T) {
	ctx := context.Background()
	dbFile := filepath.Join(t.TempDir(), "impact_cli.db")

	database, err := db.InitDB(dbFile)
	if err != nil {
		if strings.Contains(err.Error(), "no such module: fts5") {
			t.Skip("ambiente sem FTS5")
		}
		t.Fatalf("falha ao inicializar SQLite: %v", err)
	}

	now := time.Now().Unix()
	_ = db.InsertDocument(ctx, database, "adr-001", "docs/adr/001.md", "ADR 001", now, "h1")
	_ = db.InsertDocument(ctx, database, "adr-002", "docs/adr/002.md", "ADR 002", now, "h2")
	_ = db.InsertEdgeWithProps(ctx, database, "adr-002", "adr-001", "implements", "EXTRACTED", 1.0)
	database.Close()

	// 1. Teste de Saída Textual
	var textBuf bytes.Buffer
	err = runImpactCommand(ctx, "test-repo", []string{"--db", dbFile, "--depth", "2", "adr-001"}, &textBuf)
	if err != nil {
		t.Fatalf("falha ao executar mem impact texto: %v", err)
	}

	textOut := textBuf.String()
	if !strings.Contains(textOut, "Análise de Impacto (Blast Radius)") {
		t.Errorf("título não encontrado na saída textual:\n%s", textOut)
	}
	if !strings.Contains(textOut, "adr-002") {
		t.Errorf("nó dependente adr-002 não encontrado na saída:\n%s", textOut)
	}
	if !strings.Contains(textOut, "[CRÍTICO]") && !strings.Contains(textOut, "CRITICAL") {
		t.Errorf("badge de severidade não encontrado:\n%s", textOut)
	}

	// 2. Teste de Saída JSON
	var jsonBuf bytes.Buffer
	err = runImpactCommand(ctx, "test-repo", []string{"--db", dbFile, "--json", "adr-001"}, &jsonBuf)
	if err != nil {
		t.Fatalf("falha ao executar mem impact --json: %v", err)
	}

	var res graph.ImpactResult
	if err := json.Unmarshal(jsonBuf.Bytes(), &res); err != nil {
		t.Fatalf("falha ao desserializar JSON de mem impact: %v\nJSON: %s", err, jsonBuf.String())
	}

	if res.TargetNode != "adr-001" {
		t.Errorf("esperava TargetNode 'adr-001', obteve '%s'", res.TargetNode)
	}
	if res.TotalImpacted != 1 {
		t.Errorf("esperava 1 nó impactado, obteve %d", res.TotalImpacted)
	}
	if res.DirectDependents != 1 {
		t.Errorf("esperava 1 direto, obteve %d", res.DirectDependents)
	}
	if res.Nodes[0].ID != "adr-002" {
		t.Errorf("esperava nó dependente 'adr-002', obteve '%s'", res.Nodes[0].ID)
	}
}

func TestRearrangeImpactArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "flags after target node",
			input:    []string{"my-target-node", "--depth", "3", "--db", "test.db", "--json"},
			expected: []string{"--depth", "3", "--db", "test.db", "--json", "my-target-node"},
		},
		{
			name:     "flags before target node",
			input:    []string{"--db", "test.db", "--depth", "1", "my-target-node"},
			expected: []string{"--db", "test.db", "--depth", "1", "my-target-node"},
		},
		{
			name:     "flags with equals",
			input:    []string{"my-target-node", "--db=test.db", "--depth=2"},
			expected: []string{"--db=test.db", "--depth=2", "my-target-node"},
		},
		{
			name:     "mixed order with single dash",
			input:    []string{"my-target-node", "-json", "-depth", "4"},
			expected: []string{"-json", "-depth", "4", "my-target-node"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rearrangeImpactArgs(tt.input)
			if len(got) != len(tt.expected) {
				t.Fatalf("tamanho diferente: obteve %d, esperava %d\nGot: %v\nExpected: %v", len(got), len(tt.expected), got, tt.expected)
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("índice %d: obteve '%s', esperava '%s'", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestImpactCLI_FlagAfterTargetNode(t *testing.T) {
	ctx := context.Background()
	dbFile := filepath.Join(t.TempDir(), "impact_flags.db")

	database, err := db.InitDB(dbFile)
	if err != nil {
		if strings.Contains(err.Error(), "no such module: fts5") {
			t.Skip("ambiente sem FTS5")
		}
		t.Fatalf("falha ao inicializar SQLite: %v", err)
	}

	now := time.Now().Unix()
	_ = db.InsertDocument(ctx, database, "core-node", "docs/core.md", "Core Note", now, "h1")
	_ = db.InsertDocument(ctx, database, "dep-node", "docs/dep.md", "Dependent Note", now, "h2")
	_ = db.InsertEdgeWithProps(ctx, database, "dep-node", "core-node", "depends_on", "EXTRACTED", 1.0)
	database.Close()

	// Testar chamada onde a flag vem DEPOIS do identificador do nó alvo
	var textBuf bytes.Buffer
	err = runImpactCommand(ctx, "test-repo", []string{"core-node", "--db", dbFile, "--depth", "2"}, &textBuf)
	if err != nil {
		t.Fatalf("falha ao executar mem impact com flags após o nó: %v", err)
	}

	out := textBuf.String()
	if !strings.Contains(out, "Análise de Impacto (Blast Radius)") {
		t.Errorf("título não encontrado na saída:\n%s", out)
	}
	if !strings.Contains(out, "dep-node") {
		t.Errorf("nó dependente 'dep-node' não encontrado na saída:\n%s", out)
	}
}
