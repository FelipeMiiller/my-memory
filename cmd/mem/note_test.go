package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunNoteCLI_CreateAndAppend(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// 1. mem note create
	createArgs := []string{
		"create",
		"--vault", tmpDir,
		"--title", "Arquitetura Limpa",
		"--tags", "design,go",
		"--aliases", "CleanArch",
		"--type", "concept",
		"--content", "Princípios de desacoplamento e isolamento de domínio.",
		"--db", dbPath,
		"concepts/clean-arch.md",
	}

	if err := runNoteCLI(ctx, nil, nil, "test/repo", createArgs); err != nil {
		t.Fatalf("erro em runNoteCLI create: %v", err)
	}

	createdPath := filepath.Join(tmpDir, "concepts", "clean-arch.md")
	data, err := os.ReadFile(createdPath)
	if err != nil {
		t.Fatalf("arquivo de nota não foi criado no disco: %v", err)
	}

	s := string(data)
	if !strings.Contains(s, "title: \"Arquitetura Limpa\"") {
		t.Errorf("nota deve conter título no frontmatter: %s", s)
	}
	if !strings.Contains(s, "design") || !strings.Contains(s, "go") {
		t.Errorf("nota deve conter tags: %s", s)
	}

	// 2. mem note append
	appendArgs := []string{
		"append",
		"--vault", tmpDir,
		"--heading", "## Casos de Uso",
		"--content", "Camada de orquestração intermediária.",
		"--db", dbPath,
		"concepts/clean-arch.md",
	}

	if err := runNoteCLI(ctx, nil, nil, "test/repo", appendArgs); err != nil {
		t.Fatalf("erro em runNoteCLI append: %v", err)
	}

	dataAfterAppend, err := os.ReadFile(createdPath)
	if err != nil {
		t.Fatalf("falha ao ler após append: %v", err)
	}
	sAfter := string(dataAfterAppend)
	if !strings.Contains(sAfter, "## Casos de Uso") || !strings.Contains(sAfter, "Camada de orquestração intermediária.") {
		t.Errorf("arquivo deve conter seção anexada: %s", sAfter)
	}
}

func TestRunCompileCLI(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	compileArgs := []string{
		"--vault", tmpDir,
		"--topic", "microsserviços e mensageria",
		"--out", "syntheses/microservices.md",
		"--title", "Síntese de Microsserviços",
		"--tags", "distributed,kafka",
		"--mode", "fts",
		"--db", dbPath,
	}

	if err := runCompileCLI(ctx, nil, nil, "test/repo", compileArgs); err != nil {
		t.Fatalf("erro em runCompileCLI: %v", err)
	}

	compPath := filepath.Join(tmpDir, "syntheses", "microservices.md")
	data, err := os.ReadFile(compPath)
	if err != nil {
		t.Fatalf("arquivo compilado não encontrado no disco: %v", err)
	}

	s := string(data)
	if !strings.Contains(s, "title: \"Síntese de Microsserviços\"") {
		t.Errorf("arquivo deve conter título no frontmatter: %s", s)
	}
	if !strings.Contains(s, "type: summary") {
		t.Errorf("tipo da nota compilada deve ser summary: %s", s)
	}
	if !strings.Contains(s, "Compilação de Conhecimento (Compile-not-Retrieve)") {
		t.Errorf("documento compilado deve conter banner de síntese: %s", s)
	}
}
