package graph

import (
	"fmt"
	"sort"
	"strings"
)

// PackTier representa o nível de detalhe atribuído a uma nota no pacote de contexto
type PackTier string

const (
	TierCore    PackTier = "core"    // Conteúdo integral L2
	TierFringe  PackTier = "fringe"  // Micro-abstract / sumário L0/L1
	TierOmitted PackTier = "omitted" // Omitido por restrição de orçamento de tokens
)

// PackOptions define as configurações de empacotamento do subgrafo
type PackOptions struct {
	MaxDepth               int                // Profundidade máxima da expansão BFS (padrão: 2)
	MaxTokens              int                // Orçamento máximo de tokens (padrão: 4000)
	Direction              string             // "both", "outbound", "inbound" (padrão: "both")
	IncludeFringeAbstracts bool               // Se verdadeiro, faz fallback para L0/L1 para nós periféricos (padrão: true)
	PageRanks              map[string]float64 // Scores de PageRank para ponderação prioritária
}

// DefaultPackOptions retorna as opções padrão de empacotamento
func DefaultPackOptions() PackOptions {
	return PackOptions{
		MaxDepth:               2,
		MaxTokens:              4000,
		Direction:              "both",
		IncludeFringeAbstracts: true,
	}
}

// PackGraphNode representa um documento ou nó disponível no grafo
type PackGraphNode struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Path      string `json:"path"`
	Content   string `json:"content"`
	Abstract  string `json:"abstract"`
	Category  string `json:"category"`
	UpdatedAt int64  `json:"updated_at"`
}

// PackedNode representa o estado de uma nota após o processo de alocação de orçamento
type PackedNode struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Path     string   `json:"path"`
	Depth    int      `json:"depth"`
	Tier     PackTier `json:"tier"`
	Tokens   int      `json:"tokens"`
	Content  string   `json:"content,omitempty"`
	Abstract string   `json:"abstract,omitempty"`
	Category string   `json:"category,omitempty"`
	PageRank float64  `json:"pagerank,omitempty"`
}

// PackResult consolida o pacote de subgrafo formatado e suas métricas
type PackResult struct {
	RootID       string         `json:"root_id"`
	RootTitle    string         `json:"root_title"`
	MaxTokens    int            `json:"max_tokens"`
	TotalTokens  int            `json:"total_tokens"`
	CoreCount    int            `json:"core_count"`
	FringeCount  int            `json:"fringe_count"`
	OmittedCount int            `json:"omitted_count"`
	Nodes        []PackedNode   `json:"nodes"`
	OmittedNodes []string       `json:"omitted_nodes,omitempty"`
	Edges        []WeightedEdge `json:"edges,omitempty"`
	MermaidGraph string         `json:"mermaid_graph,omitempty"`
	Markdown     string         `json:"markdown"`
}

// EstimateTokens calcula uma aproximação determinística de tokens (1 token ~ 4 caracteres)
func EstimateTokens(text string) int {
	clean := strings.TrimSpace(text)
	if clean == "" {
		return 0
	}
	runes := len([]rune(clean))
	tokens := (runes + 3) / 4
	if tokens < 1 {
		return 1
	}
	return tokens
}

// PackContext extrai o subgrafo ao redor de rootID e aloca o orçamento de tokens
func PackContext(nodes []PackGraphNode, edges []WeightedEdge, rootID string, opts PackOptions) (*PackResult, error) {
	if opts.MaxDepth <= 0 {
		opts.MaxDepth = 2
	}
	if opts.MaxTokens <= 0 {
		opts.MaxTokens = 4000
	}
	if opts.Direction == "" {
		opts.Direction = "both"
	}

	// 1. Mapeamento de nós por ID
	nodeMap := make(map[string]PackGraphNode, len(nodes))
	for _, n := range nodes {
		nodeMap[n.ID] = n
	}

	rootNode, found := nodeMap[rootID]
	if !found {
		return nil, fmt.Errorf("nó raiz '%s' não encontrado no grafo", rootID)
	}

	// 2. Construir listas de adjacência
	outAdj := make(map[string][]WeightedEdge)
	inAdj := make(map[string][]WeightedEdge)
	for _, e := range edges {
		outAdj[e.Source] = append(outAdj[e.Source], e)
		inAdj[e.Target] = append(inAdj[e.Target], e)
	}

	// 3. Travessia BFS com profundidade
	type queueItem struct {
		id    string
		depth int
	}

	depths := make(map[string]int)
	visited := make(map[string]bool)
	queue := []queueItem{{id: rootID, depth: 0}}
	visited[rootID] = true
	depths[rootID] = 0

	subgraphEdges := make([]WeightedEdge, 0)
	edgeSeen := make(map[string]bool)

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if curr.depth >= opts.MaxDepth {
			continue
		}

		var neighbors []WeightedEdge
		if opts.Direction == "outbound" || opts.Direction == "both" {
			neighbors = append(neighbors, outAdj[curr.id]...)
		}
		if opts.Direction == "inbound" || opts.Direction == "both" {
			neighbors = append(neighbors, inAdj[curr.id]...)
		}

		for _, e := range neighbors {
			var nextID string
			if e.Source == curr.id {
				nextID = e.Target
			} else {
				nextID = e.Source
			}

			// Gravar aresta do subgrafo se ambos os nós forem relevantes
			edgeKey := fmt.Sprintf("%s->%s:%s", e.Source, e.Target, e.Type)
			if !edgeSeen[edgeKey] {
				edgeSeen[edgeKey] = true
				subgraphEdges = append(subgraphEdges, e)
			}

			if !visited[nextID] {
				visited[nextID] = true
				depths[nextID] = curr.depth + 1
				queue = append(queue, queueItem{id: nextID, depth: curr.depth + 1})
			}
		}
	}

	// 4. Coletar candidatos e ordenar por prioridade:
	//    - Nível 0 (raiz) sempre primeiro
	//    - Profundidade menor primeiro
	//    - Maior PageRank primeiro
	//    - Ordem alfabética de título/ID como desempate determinístico
	type candidateNode struct {
		node     PackGraphNode
		depth    int
		pageRank float64
	}

	candidates := make([]candidateNode, 0, len(visited))
	for id, d := range depths {
		n, ok := nodeMap[id]
		if !ok {
			n = PackGraphNode{
				ID:    id,
				Title: id,
			}
		}
		pr := 0.0
		if opts.PageRanks != nil {
			pr = opts.PageRanks[id]
		}
		candidates = append(candidates, candidateNode{
			node:     n,
			depth:    d,
			pageRank: pr,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].depth != candidates[j].depth {
			return candidates[i].depth < candidates[j].depth
		}
		if candidates[i].pageRank != candidates[j].pageRank {
			return candidates[i].pageRank > candidates[j].pageRank
		}
		return candidates[i].node.ID < candidates[j].node.ID
	})

	// 5. Alocação gananciosa com orçamento de tokens
	// Reserva de overhead inicial (~80 tokens para headers estruturais)
	headerOverhead := 80
	availableBudget := opts.MaxTokens - headerOverhead
	if availableBudget < 100 {
		availableBudget = opts.MaxTokens
	}

	packedNodes := make([]PackedNode, 0, len(candidates))
	omittedList := make([]string, 0)
	totalTokensUsed := headerOverhead
	coreCount := 0
	fringeCount := 0
	omittedCount := 0

	for _, cand := range candidates {
		c := cand.node
		contentTokens := EstimateTokens(c.Content)
		abstractTokens := EstimateTokens(c.Abstract)
		if abstractTokens == 0 && c.Content != "" {
			// Fallback de abstract se vazio: primeiros 200 caracteres
			if len(c.Content) > 200 {
				abstractTokens = EstimateTokens(c.Content[:200] + "...")
			} else {
				abstractTokens = contentTokens
			}
		}

		// Nó raiz tem tratamento preferencial
		if cand.depth == 0 {
			if totalTokensUsed+contentTokens <= opts.MaxTokens {
				packedNodes = append(packedNodes, PackedNode{
					ID:       c.ID,
					Title:    c.Title,
					Path:     c.Path,
					Depth:    0,
					Tier:     TierCore,
					Tokens:   contentTokens,
					Content:  c.Content,
					Abstract: c.Abstract,
					Category: c.Category,
					PageRank: cand.pageRank,
				})
				totalTokensUsed += contentTokens
				coreCount++
				continue
			} else if opts.IncludeFringeAbstracts && totalTokensUsed+abstractTokens <= opts.MaxTokens {
				packedNodes = append(packedNodes, PackedNode{
					ID:       c.ID,
					Title:    c.Title,
					Path:     c.Path,
					Depth:    0,
					Tier:     TierFringe,
					Tokens:   abstractTokens,
					Abstract: c.Abstract,
					Category: c.Category,
					PageRank: cand.pageRank,
				})
				totalTokensUsed += abstractTokens
				fringeCount++
				continue
			}
		}

		// Nós vizinhos:
		// Tenta TierCore
		if contentTokens > 0 && totalTokensUsed+contentTokens+20 <= opts.MaxTokens {
			packedNodes = append(packedNodes, PackedNode{
				ID:       c.ID,
				Title:    c.Title,
				Path:     c.Path,
				Depth:    cand.depth,
				Tier:     TierCore,
				Tokens:   contentTokens,
				Content:  c.Content,
				Abstract: c.Abstract,
				Category: c.Category,
				PageRank: cand.pageRank,
			})
			totalTokensUsed += contentTokens + 20
			coreCount++
		} else if opts.IncludeFringeAbstracts && abstractTokens > 0 && totalTokensUsed+abstractTokens+15 <= opts.MaxTokens {
			// Tenta TierFringe
			abstractContent := c.Abstract
			if abstractContent == "" && c.Content != "" {
				if len(c.Content) > 200 {
					abstractContent = c.Content[:200] + "..."
				} else {
					abstractContent = c.Content
				}
			}
			packedNodes = append(packedNodes, PackedNode{
				ID:       c.ID,
				Title:    c.Title,
				Path:     c.Path,
				Depth:    cand.depth,
				Tier:     TierFringe,
				Tokens:   abstractTokens,
				Abstract: abstractContent,
				Category: c.Category,
				PageRank: cand.pageRank,
			})
			totalTokensUsed += abstractTokens + 15
			fringeCount++
		} else {
			// Não coube: TierOmitted
			packedNodes = append(packedNodes, PackedNode{
				ID:       c.ID,
				Title:    c.Title,
				Path:     c.Path,
				Depth:    cand.depth,
				Tier:     TierOmitted,
				Tokens:   0,
				Category: c.Category,
				PageRank: cand.pageRank,
			})
			omittedList = append(omittedList, c.ID)
			omittedCount++
		}
	}

	// 6. Gerar Diagrama Mermaid
	mermaidDiagram := buildMermaidSubgraph(rootID, packedNodes, subgraphEdges)

	// 7. Renderizar Markdown Consolidado
	markdownBundle := renderMarkdownBundle(rootNode, opts, packedNodes, omittedList, totalTokensUsed, mermaidDiagram)

	rootTitle := rootNode.Title
	if rootTitle == "" {
		rootTitle = rootNode.ID
	}

	return &PackResult{
		RootID:       rootID,
		RootTitle:    rootTitle,
		MaxTokens:    opts.MaxTokens,
		TotalTokens:  totalTokensUsed,
		CoreCount:    coreCount,
		FringeCount:  fringeCount,
		OmittedCount: omittedCount,
		Nodes:        packedNodes,
		OmittedNodes: omittedList,
		Edges:        subgraphEdges,
		MermaidGraph: mermaidDiagram,
		Markdown:     markdownBundle,
	}, nil
}

// buildMermaidSubgraph cria o diagrama Mermaid representando os nós do pacote
func buildMermaidSubgraph(rootID string, nodes []PackedNode, edges []WeightedEdge) string {
	var sb strings.Builder
	sb.WriteString("```mermaid\ngraph TD\n")

	nodeIDs := make(map[string]bool)
	idToLabel := make(map[string]string)
	for idx, n := range nodes {
		if n.Tier == TierOmitted {
			continue
		}
		safeID := fmt.Sprintf("N%d", idx)
		label := n.Title
		if label == "" {
			label = n.ID
		}
		label = strings.ReplaceAll(label, "\"", "'")
		if len(label) > 30 {
			label = label[:27] + "..."
		}
		idToLabel[n.ID] = safeID
		nodeIDs[n.ID] = true

		if n.ID == rootID {
			sb.WriteString(fmt.Sprintf("    %s[\"⭐ %s (Raiz)\"]:::root\n", safeID, label))
		} else if n.Tier == TierCore {
			sb.WriteString(fmt.Sprintf("    %s[\"📄 %s\"]:::core\n", safeID, label))
		} else {
			sb.WriteString(fmt.Sprintf("    %s[\"📑 %s (L0/L1)\"]:::fringe\n", safeID, label))
		}
	}

	for _, e := range edges {
		srcID, ok1 := idToLabel[e.Source]
		tgtID, ok2 := idToLabel[e.Target]
		if ok1 && ok2 {
			rel := e.Type
			if rel == "" {
				rel = "links_to"
			}
			sb.WriteString(fmt.Sprintf("    %s -->|%s| %s\n", srcID, rel, tgtID))
		}
	}

	sb.WriteString("    classDef root fill:#4a154b,stroke:#e01e5a,stroke-width:2px,color:#fff;\n")
	sb.WriteString("    classDef core fill:#1e3d59,stroke:#17b978,stroke-width:1.5px,color:#fff;\n")
	sb.WriteString("    classDef fringe fill:#2b2e4a,stroke:#ff8e71,stroke-width:1px,color:#fff,stroke-dasharray: 4 2;\n")
	sb.WriteString("```")
	return sb.String()
}

// renderMarkdownBundle monta o documento final pronto para injeção em prompts
func renderMarkdownBundle(root PackGraphNode, opts PackOptions, nodes []PackedNode, omitted []string, tokens int, mermaid string) string {
	var sb strings.Builder

	title := root.Title
	if title == "" {
		title = root.ID
	}

	sb.WriteString(fmt.Sprintf("# 📦 Pacote de Contexto: %s\n\n", title))
	sb.WriteString(fmt.Sprintf("> **Nó Raiz:** `%s` | **Profundidade:** %d saltos | **Orçamento:** ~%d / %d tokens | **Nós Coletados:** %d\n\n",
		root.ID, opts.MaxDepth, tokens, opts.MaxTokens, len(nodes)-len(omitted)))

	// Seção 1: Topologia
	if mermaid != "" {
		sb.WriteString("## 🗺 Topologia do Subgrafo\n\n")
		sb.WriteString(mermaid)
		sb.WriteString("\n\n---\n\n")
	}

	// Seção 2: Documentos Centrais (TierCore)
	coreFound := false
	for _, n := range nodes {
		if n.Tier == TierCore {
			if !coreFound {
				sb.WriteString("## 📄 Documentos Centrais (Texto Integral)\n\n")
				coreFound = true
			}
			nodeTitle := n.Title
			if nodeTitle == "" {
				nodeTitle = n.ID
			}
			sb.WriteString(fmt.Sprintf("### %s\n", nodeTitle))
			sb.WriteString(fmt.Sprintf("- **ID:** `%s`\n", n.ID))
			if n.Path != "" {
				sb.WriteString(fmt.Sprintf("- **Arquivo:** `%s`\n", n.Path))
			}
			if n.Category != "" {
				sb.WriteString(fmt.Sprintf("- **Categoria:** `%s`\n", n.Category))
			}
			sb.WriteString(fmt.Sprintf("- **Distância:** %d salto(s) | **Tokens Est.:** ~%d\n\n", n.Depth, n.Tokens))
			sb.WriteString(n.Content)
			sb.WriteString("\n\n---\n\n")
		}
	}

	// Seção 3: Resumos Periféricos (TierFringe)
	fringeFound := false
	for _, n := range nodes {
		if n.Tier == TierFringe {
			if !fringeFound {
				sb.WriteString("## 📑 Resumos Periféricos (L0/L1 Micro-Abstracts)\n\n")
				sb.WriteString("| Distância | Título / ID | Categoria | Tokens | Resumo |\n")
				sb.WriteString("| :---: | :--- | :---: | :---: | :--- |\n")
				fringeFound = true
			}
			nodeTitle := n.Title
			if nodeTitle == "" {
				nodeTitle = n.ID
			}
			cleanAbstract := strings.ReplaceAll(n.Abstract, "\n", " ")
			cleanAbstract = strings.ReplaceAll(cleanAbstract, "|", "\\|")
			sb.WriteString(fmt.Sprintf("| %d | **%s** (`%s`) | `%s` | ~%d | %s |\n",
				n.Depth, nodeTitle, n.ID, n.Category, n.Tokens, cleanAbstract))
		}
	}
	if fringeFound {
		sb.WriteString("\n---\n\n")
	}

	// Seção 4: Omitidos
	if len(omitted) > 0 {
		sb.WriteString(fmt.Sprintf("## ⚠️ Nós Omitidos por Orçamento de Tokens (%d)\n\n", len(omitted)))
		sb.WriteString("Os seguintes nós conectados foram identificados no subgrafo, mas omitidos para não exceder o limite de tokens:\n\n")
		for _, o := range omitted {
			sb.WriteString(fmt.Sprintf("- `%s`\n", o))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
