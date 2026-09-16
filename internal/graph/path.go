package graph

import (
	"container/heap"
	"fmt"
	"math"
	"strings"
)

// CostMode define a métrica de custo para o cálculo do menor caminho
type CostMode string

const (
	CostModeEpistemic CostMode = "epistemic"
	CostModeHops      CostMode = "hops"
)

// PathOptions define os parâmetros para busca de caminho entre nós
type PathOptions struct {
	MaxDepth int      `json:"max_depth"`
	Directed bool     `json:"directed"`
	CostMode CostMode `json:"cost_mode"`
}

// DefaultPathOptions retorna as opções recomendadas padrão
func DefaultPathOptions() PathOptions {
	return PathOptions{
		MaxDepth: 6,
		Directed: true,
		CostMode: CostModeEpistemic,
	}
}

// PathEdge representa um segmento de aresta percorrido no caminho
type PathEdge struct {
	From            string  `json:"from"`
	To              string  `json:"to"`
	Relation        string  `json:"relation"`
	EpistemicStatus string  `json:"epistemic_status"`
	Weight          float64 `json:"weight"`
	Cost            float64 `json:"cost"`
	Direction       string  `json:"direction"` // "forward" ou "reverse"
}

// PathResult representa a rota encontrada entre source e target
type PathResult struct {
	Found     bool       `json:"found"`
	Source    string     `json:"source"`
	Target    string     `json:"target"`
	Directed  bool       `json:"directed"`
	CostMode  CostMode   `json:"cost_mode"`
	Hops      int        `json:"hops"`
	TotalCost float64    `json:"total_cost"`
	Nodes     []string   `json:"nodes"`
	Edges     []PathEdge `json:"edges"`
	Summary   string     `json:"summary"`
}

// pqItem representa um estado na fila de prioridades do Dijkstra
type pqItem struct {
	node  string
	cost  float64
	depth int
	prev  *pqItem
	edge  *PathEdge
	index int
}

type priorityQueue []*pqItem

func (pq priorityQueue) Len() int           { return len(pq) }
func (pq priorityQueue) Less(i, j int) bool { return pq[i].cost < pq[j].cost }
func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}
func (pq *priorityQueue) Push(x any) {
	n := len(*pq)
	item := x.(*pqItem)
	item.index = n
	*pq = append(*pq, item)
}
func (pq *priorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*pq = old[0 : n-1]
	return item
}

// ResolveEdgeCost calcula o custo numérico de atravessar uma aresta
func ResolveEdgeCost(edge WeightedEdge, mode CostMode) (float64, float64, string) {
	status := strings.ToUpper(strings.TrimSpace(edge.EpistemicStatus))
	if status == "" {
		status = strings.ToUpper(strings.TrimSpace(edge.Type))
	}
	weight := ResolveEdgeWeight(status, edge.Weight)
	if weight <= 0 {
		weight = 1.0
	}

	if status != "EXTRACTED" && status != "INFERRED" && status != "TAG" {
		status = "EXTRACTED"
	}

	if mode == CostModeHops {
		return 1.0, weight, status
	}

	// Modo epistêmico: custo inversamente proporcional ao peso (certeza)
	cost := 1.0 / weight
	return cost, weight, status
}

type graphAdjEdge struct {
	target          string
	relation        string
	epistemicStatus string
	weight          float64
	cost            float64
	direction       string
}

// FindPath calcula o caminho mínimo ponderado entre sourceID e targetID no grafo
func FindPath(nodes []string, edges []WeightedEdge, sourceID, targetID string, opts PathOptions) (*PathResult, error) {
	cleanSource := strings.TrimSpace(sourceID)
	cleanTarget := strings.TrimSpace(targetID)

	if cleanSource == "" {
		return nil, fmt.Errorf("identificador do nó de origem não pode ser vazio")
	}
	if cleanTarget == "" {
		return nil, fmt.Errorf("identificador do nó de destino não pode ser vazio")
	}

	maxDepth := opts.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 6
	}
	costMode := opts.CostMode
	if costMode != CostModeHops {
		costMode = CostModeEpistemic
	}

	// 1. Validação de existência de nós
	allNodesMap := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		allNodesMap[strings.TrimSpace(n)] = true
	}
	for _, e := range edges {
		allNodesMap[strings.TrimSpace(e.Source)] = true
		allNodesMap[strings.TrimSpace(e.Target)] = true
	}

	if !allNodesMap[cleanSource] {
		return nil, fmt.Errorf("nó de origem '%s' não encontrado no grafo", cleanSource)
	}
	if !allNodesMap[cleanTarget] {
		return nil, fmt.Errorf("nó de destino '%s' não encontrado no grafo", cleanTarget)
	}

	// Caso trivial: origem idêntica ao destino
	if cleanSource == cleanTarget {
		return &PathResult{
			Found:     true,
			Source:    cleanSource,
			Target:    cleanTarget,
			Directed:  opts.Directed,
			CostMode:  costMode,
			Hops:      0,
			TotalCost: 0.0,
			Nodes:     []string{cleanSource},
			Edges:     []PathEdge{},
			Summary:   fmt.Sprintf("Nó de origem e destino são idênticos ('%s')", cleanSource),
		}, nil
	}

	// 2. Montar lista de adjacências
	adj := make(map[string][]graphAdjEdge)
	for _, e := range edges {
		s := strings.TrimSpace(e.Source)
		t := strings.TrimSpace(e.Target)
		if s == "" || t == "" {
			continue
		}

		cost, weight, epStatus := ResolveEdgeCost(e, costMode)
		rel := strings.TrimSpace(e.Type)
		if rel == "" || strings.EqualFold(rel, "EXTRACTED") || strings.EqualFold(rel, "INFERRED") || strings.EqualFold(rel, "TAG") {
			rel = "links_to"
		}

		// Aresta forward
		adj[s] = append(adj[s], graphAdjEdge{
			target:          t,
			relation:        rel,
			epistemicStatus: epStatus,
			weight:          weight,
			cost:            cost,
			direction:       "forward",
		})

		// Aresta reverse (caso não-direcionado)
		if !opts.Directed {
			adj[t] = append(adj[t], graphAdjEdge{
				target:          s,
				relation:        rel,
				epistemicStatus: epStatus,
				weight:          weight,
				cost:            cost,
				direction:       "reverse",
			})
		}
	}

	// 3. Execução de Dijkstra com podas por profundidade
	// bestCostSeen mapeia node -> menor custo já expandido para uma dada profundidade <= d
	type depthCost struct {
		depth int
		cost  float64
	}
	visited := make(map[string][]depthCost)

	isDominated := func(node string, depth int, cost float64) bool {
		entries := visited[node]
		for _, entry := range entries {
			if entry.depth <= depth && entry.cost <= cost {
				return true
			}
		}
		return false
	}

	pq := make(priorityQueue, 0)
	heap.Init(&pq)

	heap.Push(&pq, &pqItem{
		node:  cleanSource,
		cost:  0.0,
		depth: 0,
		prev:  nil,
		edge:  nil,
	})

	var targetItem *pqItem

	for pq.Len() > 0 {
		curr := heap.Pop(&pq).(*pqItem)

		// Se atingimos o destino, como Dijkstra garante ordem crescente de custo, achamos o menor caminho
		if curr.node == cleanTarget {
			targetItem = curr
			break
		}

		if curr.depth >= maxDepth {
			continue
		}

		if isDominated(curr.node, curr.depth, curr.cost) {
			continue
		}
		visited[curr.node] = append(visited[curr.node], depthCost{depth: curr.depth, cost: curr.cost})

		for _, e := range adj[curr.node] {
			nextDepth := curr.depth + 1
			if nextDepth > maxDepth {
				continue
			}

			nextCost := curr.cost + e.cost
			if isDominated(e.target, nextDepth, nextCost) {
				continue
			}

			// Evita ciclos no caminho atual
			cycleDetected := false
			for p := curr; p != nil; p = p.prev {
				if p.node == e.target {
					cycleDetected = true
					break
				}
			}
			if cycleDetected {
				continue
			}

			pEdge := &PathEdge{
				From:            curr.node,
				To:              e.target,
				Relation:        e.relation,
				EpistemicStatus: e.epistemicStatus,
				Weight:          e.weight,
				Cost:            math.Round(e.cost*1000) / 1000,
				Direction:       e.direction,
			}

			heap.Push(&pq, &pqItem{
				node:  e.target,
				cost:  nextCost,
				depth: nextDepth,
				prev:  curr,
				edge:  pEdge,
			})
		}
	}

	// 4. Se não encontramos o caminho
	if targetItem == nil {
		return &PathResult{
			Found:     false,
			Source:    cleanSource,
			Target:    cleanTarget,
			Directed:  opts.Directed,
			CostMode:  costMode,
			Hops:      0,
			TotalCost: 0.0,
			Nodes:     []string{},
			Edges:     []PathEdge{},
			Summary:   fmt.Sprintf("Nenhum caminho encontrado entre '%s' e '%s' dentro de %d saltos", cleanSource, cleanTarget, maxDepth),
		}, nil
	}

	// 5. Reconstruir a rota
	var reversedEdges []PathEdge
	var reversedNodes []string

	for it := targetItem; it != nil; it = it.prev {
		reversedNodes = append(reversedNodes, it.node)
		if it.edge != nil {
			reversedEdges = append(reversedEdges, *it.edge)
		}
	}

	// Inverter para ordem natural (Source -> Target)
	pathNodes := make([]string, len(reversedNodes))
	for i := 0; i < len(reversedNodes); i++ {
		pathNodes[i] = reversedNodes[len(reversedNodes)-1-i]
	}

	pathEdges := make([]PathEdge, len(reversedEdges))
	for i := 0; i < len(reversedEdges); i++ {
		pathEdges[i] = reversedEdges[len(reversedEdges)-1-i]
	}

	totalCost := math.Round(targetItem.cost*1000) / 1000

	summary := fmt.Sprintf("Caminho encontrado com %d salto(s) e custo total %.3f (modo: %s)",
		targetItem.depth, totalCost, costMode)

	return &PathResult{
		Found:     true,
		Source:    cleanSource,
		Target:    cleanTarget,
		Directed:  opts.Directed,
		CostMode:  costMode,
		Hops:      targetItem.depth,
		TotalCost: totalCost,
		Nodes:     pathNodes,
		Edges:     pathEdges,
		Summary:   summary,
	}, nil
}
