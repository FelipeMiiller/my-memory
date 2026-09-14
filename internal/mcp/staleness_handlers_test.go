package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/config"
	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/staleness"
)

func TestMCP_StalenessBannerInjection(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "mcp_staleness.db")

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	docPath := filepath.Join(tempDir, "doc1.md")
	pastTime := time.Now().Add(-30 * time.Second).Truncate(time.Second)
	if err := os.WriteFile(docPath, []byte("# Nota 1\nConteudo de teste"), 0644); err != nil {
		t.Fatalf("WriteFile falhou: %v", err)
	}
	_ = os.Chtimes(docPath, pastTime, pastTime)

	// Registra no SQLite
	if err := db.InsertDocument(ctx, database, docPath, "doc1.md", "Nota 1", pastTime.Unix(), "hashold"); err != nil {
		t.Fatalf("InsertDocument falhou: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Include = []string{"**/*.md"}
	cfg.Exclude = []string{".git/**"}

	detector := staleness.NewDetector(tempDir, &cfg, database, nil, "")
	detector.SetTTL(10 * time.Millisecond)

	srv := NewServer("test-mcp", "1.0.0", nil, nil, nil)
	srv.SetStalenessDetector(detector)

	// Mock para search handler
	srv.SetAdvancedSearchHandler(func(ctx context.Context, params SearchParams) ([]SearchResult, error) {
		return []SearchResult{
			{
				ChunkID:    "chunk-1",
				DocumentID: docPath,
				Content:    "Conteudo encontrado",
				Score:      0.95,
			},
		}, nil
	})

	// 1. Em estado sincronizado (pastTime == updatedAt)
	callArgs, _ := json.Marshal(map[string]any{
		"name": ToolMemorySearch.Name,
		"arguments": map[string]any{
			"query": "Conteudo",
		},
	})
	resSync, err := srv.handleToolsCall(ctx, callArgs)
	if err != nil {
		t.Fatalf("handleToolsCall falhou: %v", err)
	}
	toolResSync, ok := resSync.(CallToolResult)
	if !ok || len(toolResSync.Content) == 0 {
		t.Fatalf("Resposta inválida: %+v", resSync)
	}
	if strings.Contains(toolResSync.Content[0].Text, "AVISO: Memória Desatualizada") {
		t.Errorf("Não deveria conter banner de desatualização quando em sincronia!")
	}

	// 2. Modifica arquivo no disco para torná-lo desatualizado
	newTime := pastTime.Add(10 * time.Second)
	_ = os.Chtimes(docPath, newTime, newTime)
	detector.ResetCache()

	resStale, err := srv.handleToolsCall(ctx, callArgs)
	if err != nil {
		t.Fatalf("handleToolsCall falhou: %v", err)
	}
	toolResStale, ok := resStale.(CallToolResult)
	if !ok || len(toolResStale.Content) == 0 {
		t.Fatalf("Resposta inválida: %+v", resStale)
	}
	if !strings.Contains(toolResStale.Content[0].Text, "AVISO: Memória Desatualizada") {
		t.Errorf("Deveria conter banner de desatualização após modificação do arquivo! Texto:\n%s", toolResStale.Content[0].Text)
	}
	if !strings.Contains(toolResStale.Content[0].Text, "mem index") {
		t.Errorf("Banner deveria recomendar 'mem index'")
	}
}

func TestMCP_StalenessCacheResetOnWrite(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "write_reset.db")

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	cfg := config.DefaultConfig()
	detector := staleness.NewDetector(tempDir, &cfg, database, nil, "")
	detector.SetTTL(1 * time.Hour) // TTL longo

	// Executa checagem inicial para popular o cache com 0 arquivos
	rep, err := detector.CheckStaleness(ctx)
	if err != nil {
		t.Fatalf("CheckStaleness falhou: %v", err)
	}
	if rep.DiskCount != 0 {
		t.Errorf("DiskCount inicial esperado 0, obteve %d", rep.DiskCount)
	}

	srv := NewServer("test-mcp", "1.0.0", nil, nil, nil)
	srv.SetStalenessDetector(detector)

	// Cria nota via memory_write_note
	writeArgs, _ := json.Marshal(map[string]any{
		"name": ToolMemoryWriteNote.Name,
		"arguments": map[string]any{
			"vault_root": tempDir,
			"path":       "new_note.md",
			"title":      "Nova Nota",
			"content":    "Conteúdo da nota atômica.",
		},
	})
	_, err = srv.handleToolsCall(ctx, writeArgs)
	if err != nil {
		t.Fatalf("handleToolsCall memory_write_note falhou: %v", err)
	}

	// Por causa do resetCache no handleToolsCall, a próxima checagem deve reavaliar o disco imediatamente!
	repAfter, err := detector.CheckStaleness(ctx)
	if err != nil {
		t.Fatalf("CheckStaleness pós-escrita falhou: %v", err)
	}
	if repAfter.DiskCount != 1 {
		t.Errorf("Cache deveria ter sido resetado na escrita! DiskCount esperado 1, obteve %d", repAfter.DiskCount)
	}
}
