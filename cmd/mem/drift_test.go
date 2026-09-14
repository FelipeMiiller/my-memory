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
	"github.com/FelipeMiiller/my-memory/internal/drift"
)

func TestRunDriftCommand_JSONAndTerminal(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_drift_cmd.db")

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("falha ao criar db: %v", err)
	}

	_, _ = database.ExecContext(ctx, `
		INSERT INTO documents (id, path, title, updated_at)
		VALUES ('docs/adr/001.md', 'docs/adr/001.md', 'ADR SQLite', 1000)
	`)
	_, _ = database.ExecContext(ctx, `
		INSERT INTO chunks (id, document_id, chunk_index, content)
		VALUES ('c1', 'docs/adr/001.md', 0, 'Implementado em internal/db/sqlite.go')
	`)
	database.Close()

	mock := &drift.MockGitRunner{
		Commits: []drift.GitCommit{
			{
				Hash:      "1111",
				ShortHash: "1111",
				Author:    "Tester",
				Date:      time.Now(),
				Message:   "feat: rewrite internal/db/sqlite.go",
			},
		},
		Changes: []drift.FileChange{
			{
				Path:       "internal/db/sqlite.go",
				Status:     "M",
				Additions:  100,
				Deletions:  20,
				IsCodeFile: true,
			},
			{
				Path:       "pkg/orphan/orphan.go",
				Status:     "A",
				Additions:  50,
				Deletions:  0,
				IsCodeFile: true,
			},
		},
	}

	// 1. Teste de saída JSON
	var jsonBuf bytes.Buffer
	err = runDriftCommand(ctx, "repo", []string{"--db", dbPath, "--json"}, &jsonBuf, mock)
	if err != nil {
		t.Fatalf("erro ao executar mem drift --json: %v", err)
	}

	var rep drift.DriftReport
	if err := json.Unmarshal(jsonBuf.Bytes(), &rep); err != nil {
		t.Fatalf("falha ao decodificar JSON: %v", err)
	}

	if rep.TotalCommits != 1 {
		t.Errorf("TotalCommits = %d; esperava 1", rep.TotalCommits)
	}
	if len(rep.DriftedNotes) != 1 {
		t.Errorf("esperava 1 drifted note, obteve %d", len(rep.DriftedNotes))
	}
	if len(rep.UncoveredCode) != 1 {
		t.Errorf("esperava 1 uncovered code, obteve %d", len(rep.UncoveredCode))
	}

	// 2. Teste de saída de terminal
	var termBuf bytes.Buffer
	err = runDriftCommand(ctx, "repo", []string{"--db", dbPath}, &termBuf, mock)
	if err != nil {
		t.Fatalf("erro ao executar mem drift terminal: %v", err)
	}

	termStr := termBuf.String()
	if !strings.Contains(termStr, "Análise de Desvio Código-Memória") {
		t.Errorf("esperava cabeçalho de drift na saída, obteve:\n%s", termStr)
	}
	if !strings.Contains(termStr, "ADR SQLite") {
		t.Errorf("esperava título da nota na saída, obteve:\n%s", termStr)
	}
	if !strings.Contains(termStr, "pkg/orphan/orphan.go") {
		t.Errorf("esperava código órfão listado na saída, obteve:\n%s", termStr)
	}
}

func TestRunDriftCommand_StrictFailure(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_drift_strict.db")

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("falha ao criar db: %v", err)
	}

	_, _ = database.ExecContext(ctx, `
		INSERT INTO documents (id, path, title, updated_at)
		VALUES ('docs/adr/001.md', 'docs/adr/001.md', 'ADR SQLite', 1000)
	`)
	_, _ = database.ExecContext(ctx, `
		INSERT INTO chunks (id, document_id, chunk_index, content)
		VALUES ('c1', 'docs/adr/001.md', 0, 'Reflete internal/core/engine.go')
	`)
	database.Close()

	// Simula grande número de alterações para gerar status crítico
	mock := &drift.MockGitRunner{
		Commits: []drift.GitCommit{
			{Hash: "1", ShortHash: "1", Author: "A", Date: time.Now(), Message: "c1"},
			{Hash: "2", ShortHash: "2", Author: "A", Date: time.Now(), Message: "c2"},
			{Hash: "3", ShortHash: "3", Author: "A", Date: time.Now(), Message: "c3"},
			{Hash: "4", ShortHash: "4", Author: "A", Date: time.Now(), Message: "c4"},
		},
		Changes: []drift.FileChange{
			{Path: "internal/core/engine.go", Status: "M", Additions: 1500, Deletions: 400, IsCodeFile: true},
		},
	}

	var buf bytes.Buffer
	err = runDriftCommand(ctx, "repo", []string{"--db", dbPath, "--strict"}, &buf, mock)
	if err == nil {
		t.Errorf("esperava erro em modo --strict devido a desvio crítico")
	}
	if !strings.Contains(err.Error(), "--strict") {
		t.Errorf("mensagem de erro inesperada: %v", err)
	}
}
