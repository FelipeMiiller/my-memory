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

	// Validação de exibição da data de atualização
	if !strings.Contains(html, "Atualizado em:") {
		t.Errorf("HTML deve conter 'Atualizado em:' no cabeçalho de estatísticas")
	}
	if !strings.Contains(html, "sb-updated-row") {
		t.Errorf("HTML deve conter 'sb-updated-row' no painel lateral")
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

// TestRenderHTML_UTF8Encoding valida que caracteres acentuados (Memória, Síntese, Usuário)
// sobrevivem intactos como bytes UTF-8 sem nenhum HTML-entity escaping indevido.
// Reproduz o bug A1 do ADR-036 (encoding quebrado em `mem graph`).
func TestRenderHTML_UTF8Encoding(t *testing.T) {
	gv := &GraphView{
		Title:      "Grafo de Memória — Síntese do Usuário",
		Repository: "central-memory",
		Nodes: []Node{
			{ID: "x.md", Title: "Configuração", Type: "guide"},
		},
	}

	htmlBytes, err := RenderHTML(gv)
	if err != nil {
		t.Fatalf("RenderHTML falhou: %v", err)
	}

	html := string(htmlBytes)

	// Bytes UTF-8 devem sobreviver intactos (NÃO convertidos em entidades HTML).
	if !strings.Contains(html, "Memória") {
		t.Errorf("HTML deve conter 'Memória' como bytes UTF-8 literais; encoding quebrado")
	}
	if !strings.Contains(html, "Síntese") {
		t.Errorf("HTML deve conter 'Síntese' como bytes UTF-8 literais; encoding quebrado")
	}
	if !strings.Contains(html, "Usuário") {
		t.Errorf("HTML deve conter 'Usuário' como bytes UTF-8 literais; encoding quebrado")
	}

	// Garantir que NÃO há entidades numéricas para caracteres que devem estar em UTF-8.
	// (Memória → &#243; seria HTML-entity escaping indevido)
	if strings.Contains(html, "&#243;") || strings.Contains(html, "&#237;") || strings.Contains(html, "&#225;") {
		t.Errorf("HTML contém entidades numéricas para caracteres acentuados; deveria ser UTF-8 direto")
	}

	// Verificar BOM ausente (BOM causa problemas com python -m http.server em algumas configs).
	if len(htmlBytes) >= 3 && htmlBytes[0] == 0xEF && htmlBytes[1] == 0xBB && htmlBytes[2] == 0xBF {
		t.Errorf("HTML não deve ter UTF-8 BOM; foi declarado charset UTF-8 no <head>")
	}

	// Verificar declarações de charset (dupla declaração para defesa contra proxies/legacy clients).
	if !strings.Contains(html, `<meta charset="UTF-8">`) {
		t.Errorf("HTML deve conter <meta charset=\"UTF-8\">")
	}
	if !strings.Contains(html, `charset=UTF-8`) {
		t.Errorf("HTML deve conter declaração adicional de charset (defesa contra proxies)")
	}
}
