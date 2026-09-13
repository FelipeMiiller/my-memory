package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunGraphCLI_Export(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mem_graph_cli_test_*")
	if err != nil {
		t.Fatalf("falha ao criar pasta temporária: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	outPath := filepath.Join(tmpDir, "test_global.html")
	ctx := context.Background()

	// Testa 'mem graph export --out <arquivo> --open=false'
	args := []string{"export", "--out", outPath, "--open=false", "--db", filepath.Join(tmpDir, "dummy.db")}
	err = runGraphCLI(ctx, "test-repo", args)
	if err != nil {
		t.Fatalf("runGraphCLI falhou: %v", err)
	}

	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("arquivo exportado não encontrado: %v", err)
	}

	html := string(content)
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Errorf("saída exportada não é um HTML válido")
	}
	if !strings.Contains(html, "Grafo de Memória") {
		t.Errorf("saída exportada não contém o título esperado")
	}
}

func TestRunGraphCLI_RootSubgraph(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mem_graph_root_test_*")
	if err != nil {
		t.Fatalf("falha ao criar pasta temporária: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	outPath := filepath.Join(tmpDir, "test_subgraph.html")
	ctx := context.Background()

	args := []string{"export", "--root", "concepts/auth.md", "--depth", "1", "--out", outPath, "--open=false", "--db", filepath.Join(tmpDir, "dummy.db")}
	err = runGraphCLI(ctx, "test-repo", args)
	if err != nil {
		t.Fatalf("runGraphCLI com subgrafo falhou: %v", err)
	}

	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("arquivo de subgrafo não encontrado: %v", err)
	}

	if !strings.Contains(string(content), "concepts/auth.md") {
		t.Errorf("HTML não contém a nota raiz concepts/auth.md")
	}
}
