package graph

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

const (
	// DefaultCommunityMaxIter é o número padrão de iterações do LPA
	DefaultCommunityMaxIter = 50
	// DefaultMinClusterSize é o tamanho mínimo padrão para incluir clusters no resultado
	DefaultMinClusterSize = 1
	// FloatTolerance para comparações de pesos no desempate
	FloatTolerance = 1e-9
)

// Community representa um cluster ou comunidade detectada no grafo
type Community struct {
	ID           int      `json:"id"`
	Label        string   `json:"label"`
	LeadNode     string   `json:"lead_node"`
	Members      []string `json:"members"`
	Size         int      `json:"size"`
	DominantType string   `json:"dominant_type,omitempty"`
}

// CommunityOptions define opções para a detecção de comunidades
type CommunityOptions struct {
	MaxIterations  int               `json:"max_iterations"`
	MinClusterSize int               `json:"min_cluster_size"`
	NodeTypes      map[string]string `json:"node_types,omitempty"`
}

// CommunityResult reúne os clusters identificados e métricas estruturais do grafo
type CommunityResult struct {
	Communities []Community `json:"communities"`
	Modularity  float64     `json:"modularity"`
	TotalNodes  int         `json:"total_nodes"`
	TotalEdges  int         `json:"total_edges"`
}

// DefaultCommunityOptions retorna as opções padrão de detecção de comunidades
func DefaultCommunityOptions() CommunityOptions {
	return CommunityOptions{
		MaxIterations:  DefaultCommunityMaxIter,
		MinClusterSize: DefaultMinClusterSize,
	}
}

// DetectCommunities executa o Weighted Label Propagation Algorithm (LPA) determinístico
// sobre os nós e arestas fornecidos, calculando a Modularidade Newman-Girvan Q.
func DetectCommunities(nodes []string, edges []WeightedEdge, opts CommunityOptions) CommunityResult {
	if opts.MaxIterations <= 0 {
		opts.MaxIterations = DefaultCommunityMaxIter
	}
	if opts.MinClusterSize <= 0 {
		opts.MinClusterSize = DefaultMinClusterSize
	}

	// 1. Coletar todos os nós únicos e ordená-los lexicograficamente para determinismo
	nodeSet := make(map[string]bool)
	for _, n := range nodes {
		trimmed := strings.TrimSpace(n)
		if trimmed != "" {
			nodeSet[trimmed] = true
		}
	}
	for _, e := range edges {
		src := strings.TrimSpace(e.Source)
		tgt := strings.TrimSpace(e.Target)
		if src != "" {
			nodeSet[src] = true
		}
		if tgt != "" {
			nodeSet[tgt] = true
		}
	}

	totalNodes := len(nodeSet)
	if totalNodes == 0 {
		return CommunityResult{
			Communities: []Community{},
			Modularity:  0.0,
			TotalNodes:  0,
			TotalEdges:  0,
		}
	}

	sortedNodes := make([]string, 0, totalNodes)
	for n := range nodeSet {
		sortedNodes = append(sortedNodes, n)
	}
	sort.Strings(sortedNodes)

	// Caso trivial: 1 nó
	if totalNodes == 1 {
		dominantType := ""
		if opts.NodeTypes != nil {
			dominantType = opts.NodeTypes[sortedNodes[0]]
		}
		return CommunityResult{
			Communities: []Community{
				{
					ID:           1,
					Label:        sortedNodes[0],
					LeadNode:     sortedNodes[0],
					Members:      []string{sortedNodes[0]},
					Size:         1,
					DominantType: dominantType,
				},
			},
			Modularity: 0.0,
			TotalNodes: 1,
			TotalEdges: len(edges),
		}
	}

	// 2. Construir lista de adjacência simétrica ponderada (grafo não-direcionado para clustering)
	adj := make(map[string]map[string]float64, totalNodes)
	nodeDegree := make(map[string]float64, totalNodes)
	var twoM float64 // soma de todos os graus (2 * soma dos pesos das arestas)

	for _, e := range edges {
		src := strings.TrimSpace(e.Source)
		tgt := strings.TrimSpace(e.Target)
		if !nodeSet[src] || !nodeSet[tgt] || src == tgt {
			continue
		}
		w := ResolveEdgeWeight(e.Type, e.Weight)
		if w <= 0 {
			w = 1.0
		}

		if adj[src] == nil {
			adj[src] = make(map[string]float64)
		}
		if adj[tgt] == nil {
			adj[tgt] = make(map[string]float64)
		}

		adj[src][tgt] += w
		adj[tgt][src] += w
		nodeDegree[src] += w
		nodeDegree[tgt] += w
		twoM += 2.0 * w
	}

	// 3. Inicializar rótulos: cada nó inicia com seu próprio ID como rótulo
	labels := make(map[string]string, totalNodes)
	for _, n := range sortedNodes {
		labels[n] = n
	}

	// 4. Propagação iterativa assíncrona com desempate determinístico
	for iter := 0; iter < opts.MaxIterations; iter++ {
		changed := false

		for _, u := range sortedNodes {
			neighbors := adj[u]
			if len(neighbors) == 0 {
				continue
			}

			// Acumular pesos por rótulo vizinho
			labelWeights := make(map[string]float64)
			for v, w := range neighbors {
				vLabel := labels[v]
				labelWeights[vLabel] += w
			}

			// Encontrar rótulo(s) com peso máximo
			var maxWeight float64 = -1.0
			for _, w := range labelWeights {
				if w > maxWeight {
					maxWeight = w
				}
			}

			// Coletar candidatos com peso máximo (dentro da tolerância float)
			var candidates []string
			for lbl, w := range labelWeights {
				if math.Abs(w-maxWeight) <= FloatTolerance || w > maxWeight {
					candidates = append(candidates, lbl)
				}
			}

			// Desempate determinístico: ordenação lexicográfica
			sort.Strings(candidates)
			bestLabel := candidates[0]

			if labels[u] != bestLabel {
				labels[u] = bestLabel
				changed = true
			}
		}

		if !changed {
			break
		}
	}

	// 5. Agrupar nós por rótulo final
	communityGroups := make(map[string][]string)
	for _, n := range sortedNodes {
		lbl := labels[n]
		communityGroups[lbl] = append(communityGroups[lbl], n)
	}

	// 6. Calcular PageRank para identificação do LeadNode (hub do cluster)
	prScores := ComputePageRank(sortedNodes, edges, DefaultPageRankOptions())

	// 7. Construir structs de comunidades
	var allCommunities []Community
	for _, members := range communityGroups {
		sort.Strings(members)

		// Determinar nó líder (maior PageRank, desempate lexicográfico)
		leadNode := members[0]
		maxPR := prScores[leadNode]
		for _, m := range members[1:] {
			pr := prScores[m]
			if pr > maxPR+FloatTolerance || (math.Abs(pr-maxPR) <= FloatTolerance && m < leadNode) {
				maxPR = pr
				leadNode = m
			}
		}

		// Determinar tipo dominante se fornecido
		dominantType := ""
		if opts.NodeTypes != nil && len(opts.NodeTypes) > 0 {
			typeCounts := make(map[string]int)
			for _, m := range members {
				t := opts.NodeTypes[m]
				if t != "" {
					typeCounts[t]++
				}
			}
			maxCount := 0
			for t, cnt := range typeCounts {
				if cnt > maxCount || (cnt == maxCount && (dominantType == "" || t < dominantType)) {
					maxCount = cnt
					dominantType = t
				}
			}
		}

		allCommunities = append(allCommunities, Community{
			Label:        leadNode,
			LeadNode:     leadNode,
			Members:      members,
			Size:         len(members),
			DominantType: dominantType,
		})
	}

	// 8. Ordenar comunidades por tamanho decrescente, e por LeadNode ascendente
	sort.Slice(allCommunities, func(i, j int) bool {
		if allCommunities[i].Size != allCommunities[j].Size {
			return allCommunities[i].Size > allCommunities[j].Size
		}
		return allCommunities[i].LeadNode < allCommunities[j].LeadNode
	})

	// Atribuir IDs sequenciais 1, 2, 3...
	for i := range allCommunities {
		allCommunities[i].ID = i + 1
		if allCommunities[i].Label == "" {
			allCommunities[i].Label = fmt.Sprintf("Cluster %d", allCommunities[i].ID)
		}
	}

	// 9. Calcular Modularidade Newman-Girvan Q sobre todas as partições do grafo
	// Q = sum_c [ (Sigma_in(c) / 2m) - (Sigma_tot(c) / 2m)^2 ]
	var modularity float64
	if twoM > 0 {
		nodeToCommunity := make(map[string]int, totalNodes)
		for _, c := range allCommunities {
			for _, m := range c.Members {
				nodeToCommunity[m] = c.ID
			}
		}

		// Para cada comunidade, calcular Sigma_in e Sigma_tot
		sigmaIn := make(map[int]float64)
		sigmaTot := make(map[int]float64)

		for u, neighbors := range adj {
			commU := nodeToCommunity[u]
			for v, w := range neighbors {
				sigmaTot[commU] += w
				if commU == nodeToCommunity[v] {
					sigmaIn[commU] += w
				}
			}
		}

		for _, c := range allCommunities {
			sin := sigmaIn[c.ID]
			stot := sigmaTot[c.ID]
			modularity += (sin / twoM) - math.Pow(stot/twoM, 2.0)
		}
	}

	// 10. Filtrar por MinClusterSize se solicitado
	filteredCommunities := make([]Community, 0, len(allCommunities))
	for _, c := range allCommunities {
		if c.Size >= opts.MinClusterSize {
			filteredCommunities = append(filteredCommunities, c)
		}
	}

	return CommunityResult{
		Communities: filteredCommunities,
		Modularity:  modularity,
		TotalNodes:  totalNodes,
		TotalEdges:  len(edges),
	}
}
