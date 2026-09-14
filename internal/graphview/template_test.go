package graphview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRenderHTML(t *testing.T) {
	gv := &GraphView{
		Title:       "Grafo de Teste Unitário",
		Repository:  "org/repo",
		GeneratedAt: time.Now(),
		Stats: GraphStats{
			TotalNodes: 2,
			TotalEdges: 1,
			Density:    0.5,
		},
		Nodes: []Node{
			{ID: "node1.md", Title: "Nota 1", Type: "concept", Color: "#3b82f6", Radius: 12.0},
			{ID: "node2.md", Title: "Nota 2", Type: "decision", Color: "#f43f5e", Radius: 15.0},
		},
		Edges: []Edge{
			{Source: "node1.md", Target: "node2.md", Relation: "depends_on", EpistemicStatus: "EXTRACTED", Color: "#475569"},
		},
	}

	htmlBytes, err := RenderHTML(gv)
	if err != nil {
		t.Fatalf("RenderHTML falhou: %v", err)
	}

	html := string(htmlBytes)

	// Validações estruturais do HTML
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Errorf("HTML deve conter <!DOCTYPE html>")
	}
	if !strings.Contains(html, "Grafo de Teste Unitário") {
		t.Errorf("HTML deve conter título do grafo")
	}
	if !strings.Contains(html, "org/repo") {
		t.Errorf("HTML deve conter identificador do repositório")
	}
	if !strings.Contains(html, "node1.md") || !strings.Contains(html, "node2.md") {
		t.Errorf("HTML deve conter os nós serializados no JSON")
	}
	if !strings.Contains(html, "tickSimulation") {
		t.Errorf("HTML deve conter o motor de simulação de física de forças")
	}

	// Validação Zero-CDN: Não deve conter links para unpkg, cdnjs ou cdn.jsdelivr
	if strings.Contains(html, "unpkg.com") || strings.Contains(html, "cdnjs.cloudflare.com") || strings.Contains(html, "cdn.jsdelivr.net") {
		t.Errorf("Violação do requisito Zero-CDN: encontrado link externo para CDN")
	}
}

func TestExportHTML(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "graphview_test_*")
	if err != nil {
		t.Fatalf("falha ao criar pasta temporária: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gv := &GraphView{
		Title: "Teste Export",
		Nodes: []Node{
			{ID: "test.md", Title: "Test"},
		},
	}

	outPath := filepath.Join(tmpDir, "view.html")
	if err := ExportHTML(gv, outPath); err != nil {
		t.Fatalf("ExportHTML falhou: %v", err)
	}

	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("arquivo exportado não encontrado: %v", err)
	}
	if info.Size() < 500 {
		t.Errorf("tamanho do arquivo exportado muito pequeno (%d bytes)", info.Size())
	}

	// Teste de falha na gravação do arquivo (diretório inexistente)
	invalidPath := filepath.Join(tmpDir, "missing_subdir", "deep", "view.html")
	if err := ExportHTML(gv, invalidPath); err == nil {
		t.Error("esperava erro ao exportar para caminho inexistente sem diretório pai")
	}
}

func TestRenderHTML_EmptyGraph(t *testing.T) {
	gv := &GraphView{
		Title: "Grafo Vazio",
	}

	htmlBytes, err := RenderHTML(gv)
	if err != nil {
		t.Fatalf("RenderHTML não deveria falhar com grafo vazio: %v", err)
	}

	if !strings.Contains(string(htmlBytes), "Grafo Vazio") {
		t.Errorf("deve conter título Grafo Vazio")
	}
}
