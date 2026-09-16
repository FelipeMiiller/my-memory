package graph

import (
	"strings"
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	if got := EstimateTokens(""); got != 0 {
		t.Errorf("EstimateTokens('') = %d, esperado 0", got)
	}
	if got := EstimateTokens("   "); got != 0 {
		t.Errorf("EstimateTokens('   ') = %d, esperado 0", got)
	}
	if got := EstimateTokens("ABCD"); got != 1 {
		t.Errorf("EstimateTokens('ABCD') = %d, esperado 1", got)
	}
	if got := EstimateTokens("12345678"); got != 2 {
		t.Errorf("EstimateTokens('12345678') = %d, esperado 2", got)
	}
	longText := strings.Repeat("palavra ", 100) // 800 caracteres
	if got := EstimateTokens(longText); got < 190 || got > 210 {
		t.Errorf("EstimateTokens(800 chars) = %d, esperado ~200", got)
	}
}

func TestPackContext_Basic(t *testing.T) {
	nodes := []PackGraphNode{
		{ID: "A", Title: "Nota A", Content: "Conteúdo completo da nota A para testes de empacotamento."},
		{ID: "B", Title: "Nota B", Content: "Conteúdo completo da nota B com referências ao módulo."},
		{ID: "C", Title: "Nota C", Content: "Conteúdo da nota C, nó periférico."},
	}
	edges := []WeightedEdge{
		{Source: "A", Target: "B", Type: "links_to", Weight: 1.0},
		{Source: "B", Target: "C", Type: "depends_on", Weight: 0.8},
	}

	opts := PackOptions{
		MaxDepth:               2,
		MaxTokens:              1000,
		Direction:              "both",
		IncludeFringeAbstracts: true,
	}

	res, err := PackContext(nodes, edges, "A", opts)
	if err != nil {
		t.Fatalf("PackContext falhou: %v", err)
	}

	if res.RootID != "A" {
		t.Errorf("RootID = %s, esperado 'A'", res.RootID)
	}
	if res.CoreCount < 2 {
		t.Errorf("CoreCount = %d, esperado pelo menos 2 nós em TierCore", res.CoreCount)
	}
	if res.OmittedCount != 0 {
		t.Errorf("OmittedCount = %d, esperado 0", res.OmittedCount)
	}
	if !strings.Contains(res.Markdown, "Pacote de Contexto: Nota A") {
		t.Errorf("Markdown não contém título esperado")
	}
	if !strings.Contains(res.MermaidGraph, "graph TD") {
		t.Errorf("MermaidGraph inválido")
	}
}

func TestPackContext_BudgetConstraintAndFallback(t *testing.T) {
	nodes := []PackGraphNode{
		{
			ID:       "Root",
			Title:    "Raiz",
			Content:  "Conteúdo base curto da raiz.",
			Abstract: "Resumo da raiz.",
		},
		{
			ID:       "HeavyChild",
			Title:    "Filho Pesado",
			Content:  strings.Repeat("Texto longo com detalhes excessivos que devem estourar o orçamento. ", 30), // ~2100 chars -> ~525 tokens
			Abstract: "Resumo curto de 1 linha do filho pesado.",
		},
	}
	edges := []WeightedEdge{
		{Source: "Root", Target: "HeavyChild", Type: "links_to", Weight: 1.0},
	}

	// Orçamento apertado de 200 tokens
	opts := PackOptions{
		MaxDepth:               1,
		MaxTokens:              200,
		Direction:              "outbound",
		IncludeFringeAbstracts: true,
	}

	res, err := PackContext(nodes, edges, "Root", opts)
	if err != nil {
		t.Fatalf("PackContext falhou: %v", err)
	}

	// HeavyChild deve sofrer fallback para TierFringe
	var heavyNode *PackedNode
	for i := range res.Nodes {
		if res.Nodes[i].ID == "HeavyChild" {
			heavyNode = &res.Nodes[i]
			break
		}
	}

	if heavyNode == nil {
		t.Fatalf("HeavyChild não encontrado nos nós retornados")
	}
	if heavyNode.Tier != TierFringe {
		t.Errorf("HeavyChild tier = %s, esperado %s (TierFringe)", heavyNode.Tier, TierFringe)
	}
	if res.TotalTokens > opts.MaxTokens {
		t.Errorf("TotalTokens (%d) excedeu MaxTokens (%d)", res.TotalTokens, opts.MaxTokens)
	}
}

func TestPackContext_OmittedNodes(t *testing.T) {
	nodes := []PackGraphNode{
		{ID: "Root", Title: "Raiz", Content: "Texto raiz."},
		{ID: "Child1", Title: "C1", Content: strings.Repeat("Texto longo C1. ", 50), Abstract: strings.Repeat("Abstract longo C1. ", 30)},
	}
	edges := []WeightedEdge{
		{Source: "Root", Target: "Child1", Type: "links_to", Weight: 1.0},
	}

	// Orçamento extremamente pequeno (100 tokens, que praticamente só cabe o overhead e a raiz)
	opts := PackOptions{
		MaxDepth:               1,
		MaxTokens:              100,
		Direction:              "outbound",
		IncludeFringeAbstracts: true,
	}

	res, err := PackContext(nodes, edges, "Root", opts)
	if err != nil {
		t.Fatalf("PackContext falhou: %v", err)
	}

	if res.OmittedCount != 1 {
		t.Errorf("OmittedCount = %d, esperado 1", res.OmittedCount)
	}
	if len(res.OmittedNodes) != 1 || res.OmittedNodes[0] != "Child1" {
		t.Errorf("OmittedNodes = %v, esperado ['Child1']", res.OmittedNodes)
	}
	if !strings.Contains(res.Markdown, "Nós Omitidos por Orçamento") {
		t.Errorf("Markdown não documentou seção de omitidos")
	}
}

func TestPackContext_DirectionAndCycles(t *testing.T) {
	nodes := []PackGraphNode{
		{ID: "1", Title: "Nó 1", Content: "Conteúdo 1"},
		{ID: "2", Title: "Nó 2", Content: "Conteúdo 2"},
		{ID: "3", Title: "Nó 3", Content: "Conteúdo 3"},
	}
	// Ciclo: 1 -> 2 -> 3 -> 1
	edges := []WeightedEdge{
		{Source: "1", Target: "2", Type: "links_to", Weight: 1.0},
		{Source: "2", Target: "3", Type: "links_to", Weight: 1.0},
		{Source: "3", Target: "1", Type: "links_to", Weight: 1.0},
	}

	opts := PackOptions{
		MaxDepth:  3,
		MaxTokens: 2000,
		Direction: "outbound",
	}

	res, err := PackContext(nodes, edges, "1", opts)
	if err != nil {
		t.Fatalf("PackContext em grafo cíclico falhou: %v", err)
	}

	if len(res.Nodes) != 3 {
		t.Errorf("Esperado 3 nós únicos no resultado, obtido %d", len(res.Nodes))
	}
}

func TestPackContext_MissingRoot(t *testing.T) {
	nodes := []PackGraphNode{
		{ID: "A", Title: "Nota A"},
	}
	edges := []WeightedEdge{}

	_, err := PackContext(nodes, edges, "Inexistente", DefaultPackOptions())
	if err == nil {
		t.Fatalf("Esperado erro ao empacotar nó raiz inexistente, mas retornou nil")
	}
	if !strings.Contains(err.Error(), "não encontrado") {
		t.Errorf("Mensagem de erro inesperada: %v", err)
	}
}
