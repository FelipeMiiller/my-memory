package graph

import (
	"fmt"
	"reflect"
	"sort"
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

	// Orçamento apertado de 300 tokens: a raiz e o cabeçalho renderizado já
	// consomem ~160 tokens, então o filho pesado não cabe em texto integral.
	opts := PackOptions{
		MaxDepth:               1,
		MaxTokens:              300,
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

	// Orçamento apertado de 250 tokens: a raiz consome ~160 tokens e o filho
	// não cabe nem como resumo — ele é omitido.
	opts := PackOptions{
		MaxDepth:               1,
		MaxTokens:              250,
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

func TestPackContext_BundleRespeitaOrcamentoReal(t *testing.T) {
	nodes := []PackGraphNode{
		{ID: "root", Title: "Raiz", Content: strings.Repeat("Conteúdo da raiz do subgrafo empacotado. ", 8)},
	}
	edges := make([]WeightedEdge, 0, 12)
	for i := 0; i < 12; i++ {
		id := fmt.Sprintf("vizinho-%02d", i)
		nodes = append(nodes, PackGraphNode{
			ID:       id,
			Title:    fmt.Sprintf("Vizinho %02d", i),
			Content:  strings.Repeat(fmt.Sprintf("Conteúdo do vizinho %02d com detalhes que ocupam espaço. ", i), 12),
			Abstract: fmt.Sprintf("Resumo do vizinho %02d", i),
		})
		edges = append(edges, WeightedEdge{Source: "root", Target: id, Type: "links_to", Weight: 1.0})
	}

	opts := PackOptions{
		MaxDepth:               1,
		MaxTokens:              1000,
		Direction:              "outbound",
		IncludeFringeAbstracts: true,
	}

	res, err := PackContext(nodes, edges, "root", opts)
	if err != nil {
		t.Fatalf("PackContext falhou: %v", err)
	}

	real := EstimateTokens(res.Markdown)
	if real > opts.MaxTokens {
		t.Errorf("o bundle emitido mede %d tokens, acima do orçamento de %d", real, opts.MaxTokens)
	}
	if res.TotalTokens != real {
		t.Errorf("TotalTokens = %d, esperado exatamente o tamanho real do bundle (%d)", res.TotalTokens, real)
	}
	if res.CoreCount+res.FringeCount == 0 {
		t.Errorf("o bundle não coletou nenhum nó: a medição do orçamento não pode ser vazia")
	}
}

func TestPackContext_OrcamentoInsuficiente(t *testing.T) {
	nodes := []PackGraphNode{
		{ID: "root", Title: "Raiz", Content: "Conteúdo da raiz."},
	}
	edges := []WeightedEdge{}

	opts := PackOptions{
		MaxDepth:               1,
		MaxTokens:              50,
		Direction:              "outbound",
		IncludeFringeAbstracts: true,
	}

	res, err := PackContext(nodes, edges, "root", opts)
	if err == nil {
		t.Fatalf("esperava erro de orçamento insuficiente, mas PackContext emitiu bundle com %d tokens", res.TotalTokens)
	}
	if res != nil {
		t.Errorf("nenhum bundle pode ser emitido com orçamento insuficiente, obtido %d tokens", res.TotalTokens)
	}
	if !strings.Contains(err.Error(), "orçamento insuficiente") {
		t.Errorf("mensagem de erro inesperada: %v", err)
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

// TestPackContext_ConjuntoExatoPorDirecao prova FPB-05, FPB-06 e FPB-07: o filtro
// de direção de R1 produz conjuntos distintos, afirmados sobre o conjunto exato de
// nós coletados — nunca sobre a contagem. O vizinho de entrada é alcançável só
// por aresta de entrada e o de saída só por aresta de saída, sem ciclo.
func TestPackContext_ConjuntoExatoPorDirecao(t *testing.T) {
	nodes := []PackGraphNode{
		{ID: "root", Title: "Raiz", Content: "Conteúdo da raiz do subgrafo empacotado."},
		{ID: "vizinho-saida", Title: "Vizinho de Saída", Content: "Conteúdo do vizinho alcançado só por aresta de saída."},
		{ID: "vizinho-entrada", Title: "Vizinho de Entrada", Content: "Conteúdo do vizinho alcançado só por aresta de entrada."},
	}
	edges := []WeightedEdge{
		{Source: "root", Target: "vizinho-saida", Type: "links_to", Weight: 1.0},
		{Source: "vizinho-entrada", Target: "root", Type: "links_to", Weight: 1.0},
	}

	cases := []struct {
		direction string
		expected  []string
	}{
		{direction: "both", expected: []string{"root", "vizinho-entrada", "vizinho-saida"}},
		{direction: "outbound", expected: []string{"root", "vizinho-saida"}},
		{direction: "inbound", expected: []string{"root", "vizinho-entrada"}},
	}

	for _, tc := range cases {
		t.Run(tc.direction, func(t *testing.T) {
			opts := PackOptions{
				MaxDepth:               2,
				MaxTokens:              2000,
				Direction:              tc.direction,
				IncludeFringeAbstracts: true,
			}

			res, err := PackContext(nodes, edges, "root", opts)
			if err != nil {
				t.Fatalf("PackContext falhou: %v", err)
			}

			got := make([]string, 0, len(res.Nodes))
			for _, n := range res.Nodes {
				got = append(got, n.ID)
			}
			sort.Strings(got)

			expected := append([]string(nil), tc.expected...)
			sort.Strings(expected)

			if !reflect.DeepEqual(got, expected) {
				t.Errorf("direction=%s coletou o conjunto %v, esperado exatamente %v", tc.direction, got, expected)
			}
		})
	}
}

// TestPackContext_OrdemDePrioridadePorPageRank prova FPB-08 e FPB-09: a ordem de
// priorização do motor é profundidade ascendente, depois PageRank descendente, depois
// ID ascendente (FPB-08). Dois nós de mesma profundidade com PageRanks invertidos
// fixam a posição esperada: o de maior PageRank vem primeiro (FPB-09).
func TestPackContext_OrdemDePrioridadePorPageRank(t *testing.T) {
	nodes := []PackGraphNode{
		{ID: "root", Title: "Raiz", Content: "Conteúdo da raiz."},
		{ID: "n1", Title: "Nó 1", Content: "Conteúdo do nó 1."},
		{ID: "n2", Title: "Nó 2", Content: "Conteúdo do nó 2."},
		{ID: "n3", Title: "Nó 3", Content: "Conteúdo do nó 3."},
		{ID: "n4", Title: "Nó 4", Content: "Conteúdo do nó 4."},
	}
	edges := []WeightedEdge{
		{Source: "root", Target: "n1", Type: "links_to", Weight: 1.0},
		{Source: "root", Target: "n2", Type: "links_to", Weight: 1.0},
		{Source: "root", Target: "n3", Type: "links_to", Weight: 1.0},
		{Source: "root", Target: "n4", Type: "links_to", Weight: 1.0},
	}

	opts := PackOptions{
		MaxDepth:               1,
		MaxTokens:              2000,
		Direction:              "outbound",
		IncludeFringeAbstracts: true,
		PageRanks: map[string]float64{
			"n1": 0.10,
			"n2": 0.90,
			"n3": 0.50,
			"n4": 0.50,
		},
	}

	res, err := PackContext(nodes, edges, "root", opts)
	if err != nil {
		t.Fatalf("PackContext falhou: %v", err)
	}

	got := make([]string, 0, len(res.Nodes))
	position := make(map[string]int, len(res.Nodes))
	for i, n := range res.Nodes {
		got = append(got, n.ID)
		position[n.ID] = i
	}

	// FPB-08: profundidade ↑, PageRank ↓, ID ↑ — a raiz (profundidade 0) primeiro,
	// depois n2 (0.90), n3 e n4 (0.50, desempate pelo ID) e por fim n1 (0.10).
	expected := []string{"root", "n2", "n3", "n4", "n1"}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("ordem dos nós = %v, esperado %v (profundidade ↑, PageRank ↓, ID ↑)", got, expected)
	}

	// FPB-09: o nó de maior PageRank ocupa a posição anterior ao de menor PageRank.
	if position["n2"] > position["n1"] {
		t.Errorf("nó de maior PageRank (n2, 0.90) ficou na posição %d, depois do de menor PageRank (n1, 0.10) na posição %d",
			position["n2"], position["n1"])
	}
}

// TestPackContext_TierDaRaizECabecalhosDoBundle prova FPB-10, FPB-11 e FPB-12: a
// raiz é classificada como TierCore quando o seu conteúdo cabe no orçamento (o tier
// da raiz, não a contagem agregada) e o bundle traz os cabeçalhos R3.3 e R3.4
// sempre que há nó central e nó periférico.
func TestPackContext_TierDaRaizECabecalhosDoBundle(t *testing.T) {
	nodes := []PackGraphNode{
		{ID: "root", Title: "Raiz", Content: "Conteúdo integral da raiz do pacote de contexto."},
		{ID: "periferico", Title: "Periférico", Abstract: "Resumo curto do vizinho periférico."},
	}
	edges := []WeightedEdge{
		{Source: "root", Target: "periferico", Type: "links_to", Weight: 1.0},
	}

	opts := PackOptions{
		MaxDepth:               1,
		MaxTokens:              2000,
		Direction:              "outbound",
		IncludeFringeAbstracts: true,
	}

	res, err := PackContext(nodes, edges, "root", opts)
	if err != nil {
		t.Fatalf("PackContext falhou: %v", err)
	}

	var root *PackedNode
	for i := range res.Nodes {
		if res.Nodes[i].ID == "root" {
			root = &res.Nodes[i]
			break
		}
	}
	if root == nil {
		t.Fatalf("a raiz não está entre os nós coletados: %+v", res.Nodes)
	}
	if root.Tier != TierCore {
		t.Errorf("tier da raiz = %s, esperado %s (TierCore)", root.Tier, TierCore)
	}

	if !strings.Contains(res.Markdown, "## 📄 Documentos Centrais") {
		t.Errorf("o bundle não traz o cabeçalho de documentos centrais:\n%s", res.Markdown)
	}
	// R3.4: a especificação de origem nomeia a seção como
	// `## 📑 Resumos Periféricos (L0/L1)`; o renderizador emite o mesmo cabeçalho
	// com o sufixo "Micro-Abstracts" — a asserção exige o cabeçalho da seção.
	if !strings.Contains(res.Markdown, "## 📑 Resumos Periféricos (L0/L1 Micro-Abstracts)") {
		t.Errorf("o bundle não traz o cabeçalho de resumos periféricos:\n%s", res.Markdown)
	}
}

// TestPackContext_RaizPreferencialNaoEhOmitida prova o tratamento preferencial da
// raiz (FPB-10): a raiz é o objeto pedido pelo chamador — quando ela não cabe nem
// como texto integral nem como resumo, o empacotador devolve o erro de orçamento
// insuficiente (identificando o mínimo exigido) e não emite um bundle que trocou a
// raiz por um nó omitido.
func TestPackContext_RaizPreferencialNaoEhOmitida(t *testing.T) {
	nodes := []PackGraphNode{
		{
			ID:       "root",
			Title:    "Raiz",
			Content:  strings.Repeat("Conteúdo integral da raiz que não cabe no orçamento. ", 90),
			Abstract: strings.Repeat("Resumo da raiz que também não cabe no orçamento. ", 90),
		},
		{ID: "vizinho", Title: "Vizinho", Content: "Conteúdo curto do vizinho."},
	}
	edges := []WeightedEdge{
		{Source: "root", Target: "vizinho", Type: "links_to", Weight: 1.0},
	}

	opts := PackOptions{
		MaxDepth:               1,
		MaxTokens:              400,
		Direction:              "outbound",
		IncludeFringeAbstracts: true,
	}

	res, err := PackContext(nodes, edges, "root", opts)
	if err == nil {
		t.Fatalf("esperava erro de orçamento insuficiente para a raiz, mas PackContext emitiu bundle com %d tokens", res.TotalTokens)
	}
	if res != nil {
		t.Errorf("nenhum bundle pode ser emitido quando a raiz não cabe: obtido %d tokens", res.TotalTokens)
	}
	if !strings.Contains(err.Error(), "orçamento insuficiente") {
		t.Errorf("mensagem de erro inesperada: %v", err)
	}
	if !strings.Contains(err.Error(), "root") {
		t.Errorf("o erro não identifica a raiz cujo mínimo foi excedido: %v", err)
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
