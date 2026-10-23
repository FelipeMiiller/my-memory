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
