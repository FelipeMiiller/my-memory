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

func TestRunGraphCLI_DefaultRootFilename(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err == nil {
		_ = os.Chdir(tmpDir)
		defer os.Chdir(origDir)
	}

	ctx := context.Background()
	args := []string{"export", "--root", "guides/deploy-prod.md", "--open=false", "--db", "dummy.db"}
	err = runGraphCLI(ctx, "test-repo", args)
	if err != nil {
		t.Fatalf("runGraphCLI falhou: %v", err)
	}

	expectedFile := filepath.Join(tmpDir, "graph_guides_deploy-prod.html")
	if _, err := os.Stat(expectedFile); err != nil {
		t.Fatalf("arquivo padrão %s não foi criado", expectedFile)
	}
}

func TestRunGraphCLI_ViewSubcommand(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "view_test.html")
	ctx := context.Background()

	args := []string{"view", "--out", outPath, "--open=false", "--db", filepath.Join(tmpDir, "dummy.db")}
	err := runGraphCLI(ctx, "test-repo", args)
	if err != nil {
		t.Fatalf("runGraphCLI com subcomando view falhou: %v", err)
	}

	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("arquivo exportado via view não encontrado: %v", err)
	}
}

func TestRunGraphCLI_ExportError(t *testing.T) {
	ctx := context.Background()
	invalidOut := filepath.Join(os.TempDir(), "non_existent_folder_abc123", "sub", "graph.html")
	args := []string{"export", "--out", invalidOut, "--open=false", "--db", filepath.Join(t.TempDir(), "dummy.db")}
	err := runGraphCLI(ctx, "test-repo", args)
	if err == nil {
		t.Fatal("esperava erro ao exportar para diretório inexistente")
	}
}

func TestRunGraphCLI_PostgresConnectionError(t *testing.T) {
	ctx := context.Background()
	args := []string{"export", "--postgres", "postgres://invalid:pass@127.0.0.1:54329/nonexistent?sslmode=disable", "--open=false"}
	err := runGraphCLI(ctx, "test-repo", args)
	if err == nil {
		t.Fatal("esperava erro de conexão com postgres inválido, obteve nil")
	}
}
