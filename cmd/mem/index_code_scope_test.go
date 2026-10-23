package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/config"
	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/embedder"
	"github.com/FelipeMiiller/my-memory/internal/turboquant"
)

// TestMemIndexNaoIndexaCodigoComoNota prova o CG-04: mesmo quando o include
// global de NOTAS casa com um arquivo de codigo (ex: **/*.go), o pipeline
// Markdown nao cria documento, chunk, embedding ou aresta de grafo para ele.
//
// O include global e escopo de notas; o codigo pertence ao code stage
// (ADR-061 §4). Sem esta separacao, um `include: **/*` transformaria cada
// arquivo de codigo do repositorio em nota — poluindo o grafo de notas, os
// embeddings e o drift.
func TestMemIndexNaoIndexaCodigoComoNota(t *testing.T) {
	vault := t.TempDir()
	dbFile := filepath.Join(t.TempDir(), "notes.db")

	if err := os.WriteFile(filepath.Join(vault, "nota.md"), []byte("# Nota\n\nconteudo da nota\n"), 0o644); err != nil {
		t.Fatalf("criando nota: %v", err)
	}
	if err := os.WriteFile(filepath.Join(vault, "main.go"), []byte("package x\n\nfunc F() {}\n"), 0o644); err != nil {
		t.Fatalf("criando arquivo de codigo: %v", err)
	}

	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()
	// O include global de notas casa com o .go — exatamente o caso perigoso.
	cfg.Include = []string{"**/*"}
	cfg.Exclude = nil

	tq := turboquant.NewQuantizer(embedder.EmbeddingDim)
	runIndexSQLite(context.Background(), database, nil, tq, &cfg, vault, true, false)

	titles, err := db.ListDocumentTitles(context.Background(), database, "")
	if err != nil {
		t.Fatalf("ListDocumentTitles falhou: %v", err)
	}

	temNota := false
	for _, title := range titles {
		if title == "nota" {
			temNota = true
		}
		if title == "main" {
			t.Errorf("arquivo de codigo virou documento no grafo de notas: %v", titles)
		}
	}
	if !temNota {
		t.Errorf("a nota Markdown deveria ter sido indexada: %v", titles)
	}

	// Nenhum chunk (e portanto nenhum embedding) pode existir para o .go.
	var chunks int
	if err := database.QueryRow("SELECT COUNT(*) FROM chunks WHERE document_id LIKE '%main.go'").Scan(&chunks); err != nil {
		t.Fatalf("contando chunks: %v", err)
	}
	if chunks != 0 {
		t.Errorf("o arquivo de codigo gerou %d chunks; esperado 0", chunks)
	}
}

// TestMemIndexExecutaCodeStage prova o CG-03: `mem index` roda o code stage no
// mesmo repositorio resolvido pelo pipeline Markdown e persiste code_files /
// code_symbols / code_edges junto dos documentos.
func TestMemIndexExecutaCodeStage(t *testing.T) {
	vault := t.TempDir()
	dbFile := filepath.Join(t.TempDir(), "vault.db")

	// Config do vault com escopo de codigo configurado (secao `code`).
	memDir := filepath.Join(vault, ".memory")
	if err := os.MkdirAll(memDir, 0o755); err != nil {
		t.Fatalf("criando .memory: %v", err)
	}
	configYAML := "version: 1\ncode:\n  include:\n    - '**/*.go'\n"
	if err := os.WriteFile(filepath.Join(memDir, "config.yaml"), []byte(configYAML), 0o644); err != nil {
		t.Fatalf("criando config: %v", err)
	}

	// Uma nota e um arquivo de codigo.
	if err := os.WriteFile(filepath.Join(vault, "nota.md"), []byte("# Nota\n\nconteudo\n"), 0o644); err != nil {
		t.Fatalf("criando nota: %v", err)
	}
	if err := os.WriteFile(filepath.Join(vault, "main.go"), []byte("package x\n\nfunc F() {}\n"), 0o644); err != nil {
		t.Fatalf("criando arquivo de codigo: %v", err)
	}

	if err := runIndexCLI(context.Background(), "test/repo", []string{
		"--dir", vault,
		"--db", dbFile,
		"--storage", "sqlite",
	}); err != nil {
		t.Fatalf("runIndexCLI falhou: %v", err)
	}

	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	// O code stage rodou: o arquivo de codigo virou code_file com simbolos.
	var files, symbols int
	if err := database.QueryRow("SELECT COUNT(*) FROM code_files").Scan(&files); err != nil {
		t.Fatalf("contando code_files: %v", err)
	}
	if files != 1 {
		t.Errorf("code_files = %d; esperado 1 (o .go do escopo de codigo)", files)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM code_symbols").Scan(&symbols); err != nil {
		t.Fatalf("contando code_symbols: %v", err)
	}
	if symbols == 0 {
		t.Error("code_symbols vazio: o code stage nao produziu simbolos")
	}

	// O grafo de notas continua sendo do pipeline Markdown: a nota e documento,
	// o .go nao.
	titles, err := db.ListDocumentTitles(context.Background(), database, "")
	if err != nil {
		t.Fatalf("ListDocumentTitles falhou: %v", err)
	}
	temNota := false
	for _, title := range titles {
		if title == "nota" {
			temNota = true
		}
		if title == "main" {
			t.Errorf("arquivo de codigo virou documento no grafo de notas: %v", titles)
		}
	}
	if !temNota {
		t.Errorf("a nota Markdown deveria ter sido indexada: %v", titles)
	}
}

// TestMemIndexSemEscopoDeCodigoNaoRodaCodeStage: sem escopo de codigo
// configurado, o `mem index` continua sendo so o pipeline Markdown — o
// comportamento anterior, sem code_files no banco.
func TestMemIndexSemEscopoDeCodigoNaoRodaCodeStage(t *testing.T) {
	vault := t.TempDir()
	dbFile := filepath.Join(t.TempDir(), "vault.db")

	memDir := filepath.Join(vault, ".memory")
	if err := os.MkdirAll(memDir, 0o755); err != nil {
		t.Fatalf("criando .memory: %v", err)
	}
	// Config SEM a secao `code`.
	if err := os.WriteFile(filepath.Join(memDir, "config.yaml"), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatalf("criando config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(vault, "main.go"), []byte("package x\n\nfunc F() {}\n"), 0o644); err != nil {
		t.Fatalf("criando arquivo de codigo: %v", err)
	}

	if err := runIndexCLI(context.Background(), "test/repo", []string{
		"--dir", vault,
		"--db", dbFile,
		"--storage", "sqlite",
	}); err != nil {
		t.Fatalf("runIndexCLI falhou: %v", err)
	}

	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	var files int
	if err := database.QueryRow("SELECT COUNT(*) FROM code_files").Scan(&files); err != nil {
		t.Fatalf("contando code_files: %v", err)
	}
	if files != 0 {
		t.Errorf("code_files = %d; esperado 0 (sem escopo de codigo o index e so Markdown)", files)
	}
}
