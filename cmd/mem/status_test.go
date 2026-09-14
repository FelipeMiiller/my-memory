package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/staleness"
)

func TestRunStatusCommand_InSync(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "status_test.db")

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}

	docPath := filepath.Join(tempDir, "sync.md")
	now := time.Now().Truncate(time.Second)
	if err := os.WriteFile(docPath, []byte("# Sincronizado"), 0644); err != nil {
		t.Fatalf("WriteFile falhou: %v", err)
	}
	_ = os.Chtimes(docPath, now, now)

	ctx := context.Background()
	if err := db.InsertDocument(ctx, database, docPath, "sync.md", "Sync", now.Unix(), "hash1"); err != nil {
		t.Fatalf("InsertDocument falhou: %v", err)
	}
	database.Close()

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--dir", tempDir}
	if err := runStatusCommand(ctx, "test-repo", args, &buf); err != nil {
		t.Fatalf("runStatusCommand falhou: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "✅ Atualizado (In Sync)") {
		t.Errorf("Esperava mensagem de atualizado, obteve:\n%s", out)
	}
	if !strings.Contains(out, "Arquivos no Disco:  1") || !strings.Contains(out, "Arquivos Indexados: 1") {
		t.Errorf("Contagens incorretas na saída:\n%s", out)
	}
}

func TestRunStatusCommand_Stale(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "status_test.db")

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}

	docPath := filepath.Join(tempDir, "old.md")
	pastTime := time.Now().Add(-20 * time.Second).Truncate(time.Second)
	if err := os.WriteFile(docPath, []byte("# Velho"), 0644); err != nil {
		t.Fatalf("WriteFile falhou: %v", err)
	}
	_ = os.Chtimes(docPath, pastTime, pastTime)

	ctx := context.Background()
	if err := db.InsertDocument(ctx, database, docPath, "old.md", "Old", pastTime.Unix(), "hashold"); err != nil {
		t.Fatalf("InsertDocument falhou: %v", err)
	}
	database.Close()

	// Modifica o arquivo no disco
	newTime := pastTime.Add(10 * time.Second)
	_ = os.Chtimes(docPath, newTime, newTime)

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--dir", tempDir}
	if err := runStatusCommand(ctx, "test-repo", args, &buf); err != nil {
		t.Fatalf("runStatusCommand falhou: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "⚠️  Desatualizado (Stale Data)") {
		t.Errorf("Esperava mensagem de desatualizado, obteve:\n%s", out)
	}
	if !strings.Contains(out, "mem index") {
		t.Errorf("Esperava recomendação de mem index, obteve:\n%s", out)
	}
}

func TestRunStatusCommand_JSON(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "status_test.db")

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	database.Close()

	// Cria arquivo sem indexar
	_ = os.WriteFile(filepath.Join(tempDir, "unindexed.md"), []byte("# Novo"), 0644)

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--dir", tempDir, "--json"}
	ctx := context.Background()
	if err := runStatusCommand(ctx, "test-repo", args, &buf); err != nil {
		t.Fatalf("runStatusCommand --json falhou: %v", err)
	}

	var rep staleness.StalenessReport
	if err := json.Unmarshal(buf.Bytes(), &rep); err != nil {
		t.Fatalf("Falha ao desserializar JSON do status: %v, saída: %s", err, buf.String())
	}

	if !rep.IsStale {
		t.Errorf("Esperava IsStale=true no JSON")
	}
	if rep.StaleFilesCount != 1 {
		t.Errorf("StaleFilesCount esperado 1, obteve %d", rep.StaleFilesCount)
	}
	if rep.DiskCount != 1 || rep.IndexedCount != 0 {
		t.Errorf("Contagens incorretas no JSON: disk=%d, indexed=%d", rep.DiskCount, rep.IndexedCount)
	}
}
