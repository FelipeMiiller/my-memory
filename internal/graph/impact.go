package graph

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// ImpactSeverity define o nível de criticidade do impacto de uma alteração
type ImpactSeverity string

const (
	SeverityCritical ImpactSeverity = "CRITICAL"
	SeverityHigh     ImpactSeverity = "HIGH"
	SeverityMedium   ImpactSeverity = "MEDIUM"
	SeverityLow      ImpactSeverity = "LOW"
)

// ImpactedNode representa um nó/documento afetado no fechamento de dependências reversas
type ImpactedNode struct {
	ID           string         `json:"id"`
	Depth        int            `json:"depth"`
	Relation     string         `json:"relation"`
	ViaNode      string         `json:"via_node"`
	Severity     ImpactSeverity `json:"severity"`
	PageRank     float64        `json:"pagerank,omitempty"`
	NodeType     string         `json:"node_type,omitempty"`
	CommunityID  int            `json:"community_id,omitempty"`
	CommunityTag string         `json:"community_tag,omitempty"`
}

// ImpactOptions define os parâmetros para cálculo do raio de destruição
type ImpactOptions struct {
	MaxDepth    int                `json:"max_depth"`
	NodeTypes   map[string]string  `json:"node_types,omitempty"`
	Communities map[string]int     `json:"communities,omitempty"`
	PageRanks   map[string]float64 `json:"page_ranks,omitempty"`
}

// DefaultImpactOptions retorna as opções padrão de análise de impacto
func DefaultImpactOptions() ImpactOptions {
	return ImpactOptions{
		MaxDepth: 2,
	}
}

// ImpactResult reúne as métricas e a lista de nós dependentes afetados
type ImpactResult struct {
	TargetNode         string                 `json:"target_node"`
	TotalImpacted      int                    `json:"total_impacted"`
	MaxDepthReached    int                    `json:"max_depth_reached"`
	RiskScore          float64                `json:"risk_score"`
	RiskLevel          string                 `json:"risk_level"`
	DirectDependents   int                    `json:"direct_dependents"`
	IndirectDependents int                    `json:"indirect_dependents"`
	SeverityCounts     map[ImpactSeverity]int `json:"severity_counts"`
	AffectedClusters   []int                  `json:"affected_clusters"`
	Nodes              []ImpactedNode         `json:"nodes"`
}

// DetermineSeverity classifica a severidade semântica da relação com base no tipo e profundidade
func DetermineSeverity(relation string, depth int) ImpactSeverity {
	rel := strings.ToLower(strings.TrimSpace(relation))

	// Relações com forte acoplamento e contratos de implementação
	isBreakingRel := strings.Contains(rel, "implement") ||
		strings.Contains(rel, "depend") ||
		strings.Contains(rel, "contradict") ||
		strings.Contains(rel, "block") ||
		strings.Contains(rel, "supersede")

	if isBreakingRel {
		if depth <= 1 {
			return SeverityCritical
		}
		return SeverityHigh
	}

	// Links associativos e referências conceituais
	if strings.Contains(rel, "link") || strings.Contains(rel, "refer") || strings.Contains(rel, "relate") {
		if depth <= 1 {
			return SeverityMedium
		}
		return SeverityLow
	}

	// Relações fracas, tags e categorizações
	return SeverityLow
}

// SeverityWeight retorna o peso multiplicador numérico associado à severidade
func SeverityWeight(s ImpactSeverity) float64 {
	switch s {
	case SeverityCritical:
		return 1.0
	case SeverityHigh:
		return 0.7
	case SeverityMedium:
		return 0.4
	case SeverityLow:
		return 0.1
	default:
		return 0.2
	}
}

// CalculateImpact executa a travessia reversa e computa o raio de destruição de um nó alvo
func CalculateImpact(nodes []string, edges []WeightedEdge, targetID string, opts ImpactOptions) (*ImpactResult, error) {
	cleanTarget := strings.TrimSpace(targetID)
	if cleanTarget == "" {
		return nil, fmt.Errorf("identificador do nó alvo não pode ser vazio")
	}

	maxDepth := opts.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 2
	}

	// 1. Validação de existência do nó no grafo
	allNodesMap := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		allNodesMap[n] = true
	}
	for _, e := range edges {
		allNodesMap[e.Source] = true
		allNodesMap[e.Target] = true
	}

	if !allNodesMap[cleanTarget] {
		return nil, fmt.Errorf("nó alvo '%s' não encontrado no grafo", cleanTarget)
	}

	// 2. Monta mapa de arestas de entrada (inbound edges: quem aponta para X)
	inboundMap := make(map[string][]WeightedEdge)
	for _, e := range edges {
		inboundMap[e.Target] = append(inboundMap[e.Target], e)
	}

	// 3. Fila BFS para travessia reversa
	type queueItem struct {
		nodeID   string
		depth    int
		viaNode  string
		relation string
	}

	visited := make(map[string]int) // nodeID -> menor profundidade visitada
	visited[cleanTarget] = 0

	queue := []queueItem{{
		nodeID:  cleanTarget,
		depth:   0,
		viaNode: "",
	}}

	var impacted []ImpactedNode
	severityCounts := map[ImpactSeverity]int{
		SeverityCritical: 0,
		SeverityHigh:     0,
		SeverityMedium:   0,
		SeverityLow:      0,
	}
	clusterSet := make(map[int]bool)
	maxDepthReached := 0

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if curr.depth >= maxDepth {
			continue
		}

		inEdges := inboundMap[curr.nodeID]
		for _, e := range inEdges {
			source := e.Source
			if source == cleanTarget {
				continue // evita loops para o próprio alvo
			}

			nextDepth := curr.depth + 1
			if prevDepth, seen := visited[source]; seen && prevDepth <= nextDepth {
				continue
			}
			visited[source] = nextDepth

			if nextDepth > maxDepthReached {
				maxDepthReached = nextDepth
			}

			rel := e.Type
			if strings.TrimSpace(rel) == "" {
				rel = "links_to"
			}
			sev := DetermineSeverity(rel, nextDepth)
			severityCounts[sev]++

			pr := 0.0
			if opts.PageRanks != nil {
				pr = opts.PageRanks[source]
			}

			nodeType := "other"
			if opts.NodeTypes != nil && opts.NodeTypes[source] != "" {
				nodeType = opts.NodeTypes[source]
			}

			commID := 0
			if opts.Communities != nil {
				commID = opts.Communities[source]
				if commID > 0 {
					clusterSet[commID] = true
				}
			}

			impacted = append(impacted, ImpactedNode{
				ID:          source,
				Depth:       nextDepth,
				Relation:    rel,
				ViaNode:     curr.nodeID,
				Severity:    sev,
				PageRank:    pr,
				NodeType:    nodeType,
				CommunityID: commID,
			})

			queue = append(queue, queueItem{
				nodeID:   source,
				depth:    nextDepth,
				viaNode:  curr.nodeID,
				relation: rel,
			})
		}
	}

	// 4. Ordenação determinística: Profundidade -> Severidade -> ID
	severityRank := func(s ImpactSeverity) int {
		switch s {
		case SeverityCritical:
			return 0
		case SeverityHigh:
			return 1
		case SeverityMedium:
			return 2
		case SeverityLow:
			return 3
		default:
			return 4
		}
	}

	sort.Slice(impacted, func(i, j int) bool {
		if impacted[i].Depth != impacted[j].Depth {
			return impacted[i].Depth < impacted[j].Depth
		}
		rI := severityRank(impacted[i].Severity)
		rJ := severityRank(impacted[j].Severity)
		if rI != rJ {
			return rI < rJ
		}
		return impacted[i].ID < impacted[j].ID
	})

	// 5. Cálculo do RiskScore e contagens
	directCount := 0
	indirectCount := 0
	rawRisk := 0.0

	for _, item := range impacted {
		if item.Depth == 1 {
			directCount++
		} else {
			indirectCount++
		}

		sWeight := SeverityWeight(item.Severity)
		depthDecay := 1.0 / float64(item.Depth)
		prBonus := 1.0 + (item.PageRank * 10.0)

		rawRisk += sWeight * depthDecay * prBonus
	}

	// Normalização sigmoidal suave de 0.0 a 100.0: S = 100 * (1 - 1 / (1 + rawRisk / 5.0))
	riskScore := 0.0
	if rawRisk > 0 {
		riskScore = 100.0 * (1.0 - (1.0 / (1.0 + (rawRisk / 5.0))))
	}
	riskScore = math.Round(riskScore*10) / 10.0 // 1 casa decimal

	riskLevel := "BAIXO"
	switch {
	case riskScore >= 75.0:
		riskLevel = "CRÍTICO"
	case riskScore >= 50.0:
		riskLevel = "ALTO"
	case riskScore >= 25.0:
		riskLevel = "MODERADO"
	default:
		riskLevel = "BAIXO"
	}

	affectedClusters := make([]int, 0, len(clusterSet))
	for c := range clusterSet {
		affectedClusters = append(affectedClusters, c)
	}
	sort.Ints(affectedClusters)

	return &ImpactResult{
		TargetNode:         cleanTarget,
		TotalImpacted:      len(impacted),
		MaxDepthReached:    maxDepthReached,
		RiskScore:          riskScore,
		RiskLevel:          riskLevel,
		DirectDependents:   directCount,
		IndirectDependents: indirectCount,
		SeverityCounts:     severityCounts,
		AffectedClusters:   affectedClusters,
		Nodes:              impacted,
	}, nil
}
