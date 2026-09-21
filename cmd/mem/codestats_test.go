package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/db"
)

func codeStatsTestFixture(t *testing.T) string {
	t.Helper()
	tmpVault := t.TempDir()
	dbDir := t.TempDir()
	dbFile := filepath.Join(dbDir, "codestats.db")

	// 2 Go files + 1 Python file (Python não vai ser indexado pelo mock → orphan).
	files := map[string]string{
		"pkg/foo.go": `package foo
func Hello() {}
func Bye() {}
`,
		"pkg/bar.go": `package bar
type Bar struct{}
func (b *Bar) Greet() string { return "" }
`,
		"scripts/empty.py": `# Python sem def/class — vira orphan
import os
x = 1
`,
	}
	for rel, content := range files {
		full := filepath.Join(tmpVault, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	if err := runCodeIndexCLI(context.Background(), "FelipeMiiller/my-memory", []string{
		"--db=" + dbFile,
		"--lang=go,python",
		"--include=**/*",
		"--exclude=**/.git/**",
		tmpVault,
	}); err != nil {
		t.Fatalf("index runCodeIndexCLI: %v", err)
	}

	return dbFile
}

func TestCodeStats_Counts(t *testing.T) {
	dbFile := codeStatsTestFixture(t)
	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer database.Close()

	stats, err := runCodeStatsQuery(context.Background(), database, 10)
	if err != nil {
		t.Fatalf("runCodeStatsQuery: %v", err)
	}
	if stats.FilesTotal < 1 {
		t.Errorf("FilesTotal = %d; want >=1", stats.FilesTotal)
	}
	if stats.SymbolsTotal < 1 {
		t.Errorf("SymbolsTotal = %d; want >=1", stats.SymbolsTotal)
	}
	if len(stats.Languages) == 0 {
		t.Error("Languages vazio")
	}
	if len(stats.SymbolKinds) == 0 {
		t.Error("SymbolKinds vazio")
	}
}

func TestCodeStats_LanguageDistribution(t *testing.T) {
	dbFile := codeStatsTestFixture(t)
	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer database.Close()

	stats, err := runCodeStatsQuery(context.Background(), database, 10)
	if err != nil {
		t.Fatalf("runCodeStatsQuery: %v", err)
	}
	found := false
	for _, l := range stats.Languages {
		if l.Language == "go" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("esperava 'go' em Languages; got %+v", stats.Languages)
	}
}

func TestCodeStats_OrphanFiles(t *testing.T) {
	dbFile := codeStatsTestFixture(t)
	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer database.Close()

	stats, err := runCodeStatsQuery(context.Background(), database, 10)
	if err != nil {
		t.Fatalf("runCodeStatsQuery: %v", err)
	}
	if len(stats.OrphanFilesTopN) == 0 {
		t.Error("esperava ≥1 orphan file (Python sem symbols)")
	}
}

func TestCodeStats_JSONMarshalStable(t *testing.T) {
	dbFile := codeStatsTestFixture(t)
	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer database.Close()

	stats, err := runCodeStatsQuery(context.Background(), database, 10)
	if err != nil {
		t.Fatalf("runCodeStatsQuery: %v", err)
	}
	b, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(b), `"files_total"`) {
		t.Errorf("JSON deve conter files_total; got %s", string(b))
	}
}

func TestCodeStats_CLI_EndToEnd(t *testing.T) {
	dbFile := codeStatsTestFixture(t)
	err := runCodeStatsCLI(context.Background(), "FelipeMiiller/my-memory", []string{
		"--db=" + dbFile,
		"--top=5",
	})
	if err != nil {
		t.Fatalf("runCodeStatsCLI falhou: %v", err)
	}
}

func TestCodeStats_CLI_JSON(t *testing.T) {
	dbFile := codeStatsTestFixture(t)
	err := runCodeStatsCLI(context.Background(), "FelipeMiiller/my-memory", []string{
		"--db=" + dbFile,
		"--json",
	})
	if err != nil {
		t.Fatalf("runCodeStatsCLI --json falhou: %v", err)
	}
}
