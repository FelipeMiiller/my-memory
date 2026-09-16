package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/deeplink"
	"github.com/FelipeMiiller/my-memory/internal/drift"
)

func TestToolMemoryGetDrift_Registration(t *testing.T) {
	s := NewServer("test", "1.0", nil, nil, nil)
	tools := s.GetTools()

	found := false
	for _, tool := range tools {
		if tool.Name == "memory_get_drift" {
			found = true
			if tool.InputSchema == nil {
				t.Errorf("schema não deveria ser nil")
			}
			break
		}
	}
	if !found {
		t.Errorf("ferramenta memory_get_drift não foi encontrada no catálogo")
	}
}

func TestNewMemoryGetDriftHandler_NilAnalyzer(t *testing.T) {
	ctx := context.Background()
	handler := NewMemoryGetDriftHandler(nil)

	res, err := handler(ctx, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	callRes, ok := res.(CallToolResult)
	if !ok || len(callRes.Content) == 0 {
		t.Fatalf("resultado inválido")
	}
	if !strings.Contains(callRes.Content[0].Text, "Nenhum motor de drift configurado") {
		t.Errorf("mensagem inesperada: %s", callRes.Content[0].Text)
	}
}

func TestNewMemoryGetDriftHandler_WithReport(t *testing.T) {
	ctx := context.Background()

	mockAnalyzer := func(ctx context.Context, rangeStr string, threshold float64, includeUncovered bool, repoSlug string) (*drift.DriftReport, error) {
		links := deeplink.GenerateLinks("/repo", "TestVault", "docs/adr/001.md", 0)
		return &drift.DriftReport{
			RangeStr:      rangeStr,
			TotalCommits:  3,
			TotalChanged:  2,
			OverallScore:  72.5,
			CriticalCount: 1,
			AnalyzedAt:    time.Now(),
			DriftedNotes: []drift.NoteDrift{
				{
					NoteID:        "docs/adr/001.md",
					Title:         "ADR-001 SQLite",
					Path:          "docs/adr/001.md",
					DriftScore:    72.5,
					Severity:      drift.SeverityCritical,
					CommitsBehind: 3,
					LinesChanged:  120,
					AffectedFiles: []string{"internal/db/sqlite.go"},
					Reason:        "3 commits alteraram 1 arquivo associado",
					Links:         &links,
				},
			},
			UncoveredCode: []drift.UncoveredCode{
				{
					FilePath:        "pkg/orphan.go",
					Status:          "A",
					Additions:       80,
					Deletions:       0,
					SuggestedAction: "Criar nota para novo módulo",
				},
			},
		}, nil
	}

	handler := NewMemoryGetDriftHandler(mockAnalyzer)

	args := json.RawMessage(`{"since": "HEAD~3..HEAD", "threshold": 0.5}`)
	res, err := handler(ctx, args)
	if err != nil {
		t.Fatalf("erro no handler: %v", err)
	}

	callRes, ok := res.(CallToolResult)
	if !ok || len(callRes.Content) == 0 {
		t.Fatalf("resultado inválido")
	}

	text := callRes.Content[0].Text
	if !strings.Contains(text, "Diagnóstico de Desvio Código-Memória") {
		t.Errorf("esperava título no markdown")
	}
	if !strings.Contains(text, "[CRITICAL]") {
		t.Errorf("esperava badge CRITICAL no markdown")
	}
	if !strings.Contains(text, "ADR-001 SQLite") {
		t.Errorf("esperava título da nota")
	}
	if !strings.Contains(text, "pkg/orphan.go") {
		t.Errorf("esperava código órfão listado")
	}
	if !strings.Contains(text, "obsidian://") {
		t.Errorf("esperava link Obsidian")
	}
}
