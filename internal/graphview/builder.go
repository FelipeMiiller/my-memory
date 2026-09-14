package graphview

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/graph"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// RawDoc representa um documento bruto recuperado do storage
type RawDoc struct {
	ID        string
	Title     string
	UpdatedAt int64
}

// RawEdge representa uma aresta bruta recuperada do storage
type RawEdge struct {
	Source          string
	Target          string
	Relation        string
	EpistemicStatus string
	Weight          float64
}

// InferNoteType infere o tipo semântico da nota a partir de seu identificador/caminho
func InferNoteType(id string) string {
	lower := strings.ToLower(id)
	switch {
	case strings.Contains(lower, "decision") || strings.Contains(lower, "adr"):
		return "decision"
	case strings.Contains(lower, "concept"):
		return "concept"
	case strings.Contains(lower, "guide") || strings.Contains(lower, "howto") || strings.Contains(lower, "tutorial"):
		return "guide"
	case strings.Contains(lower, "synthes") || strings.Contains(lower, "compiled"):
		return "synthesis"
	case strings.Contains(lower, "reference") || strings.Contains(lower, "doc"):
		return "reference"
	default:
		return "other"
	}
}

// BuildGraphView processa listas brutas de documentos e arestas, aplicando filtros e métricas
func BuildGraphView(docs []RawDoc, edges []RawEdge, rootNode string, maxDepth int, repo string) *GraphView {
	docMap := make(map[string]RawDoc)
	for _, d := range docs {
		docMap[d.ID] = d
	}

	// Adiciona nós implícitos referenciados em arestas mas ausentes da tabela documents
	for _, e := range edges {
		if _, exists := docMap[e.Source]; !exists {
			docMap[e.Source] = RawDoc{ID: e.Source, Title: e.Source}
		}
		if _, exists := docMap[e.Target]; !exists {
			docMap[e.Target] = RawDoc{ID: e.Target, Title: e.Target}
		}
	}

	// Subgrafo focado caso rootNode seja especificado
	activeNodes := make(map[string]bool)
	if rootNode != "" {
		if maxDepth <= 0 {
			maxDepth = 2
		}
		activeNodes = extractSubgraphBFS(docMap, edges, rootNode, maxDepth)
	} else {
		for id := range docMap {
			activeNodes[id] = true
		}
	}

	// Filtrar arestas pertencentes aos nós ativos
	var activeEdges []RawEdge
	inDeg := make(map[string]int)
	outDeg := make(map[string]int)

	for _, e := range edges {
		if activeNodes[e.Source] && activeNodes[e.Target] {
			activeEdges = append(activeEdges, e)
			outDeg[e.Source]++
			inDeg[e.Target]++
		}
	}

	// Calcular PageRank ponderado sobre o grafo ativo
	nodeList := make([]string, 0, len(activeNodes))
	for id := range activeNodes {
		nodeList = append(nodeList, id)
	}

	weightedEdges := make([]graph.WeightedEdge, 0, len(activeEdges))
	for _, e := range activeEdges {
		weightedEdges = append(weightedEdges, graph.WeightedEdge{
			Source: e.Source,
			Target: e.Target,
			Weight: e.Weight,
			Type:   e.EpistemicStatus,
		})
	}

	prMap := graph.ComputePageRank(nodeList, weightedEdges, graph.DefaultPageRankOptions())
	maxPR := 0.0
	for _, score := range prMap {
		if score > maxPR {
			maxPR = score
		}
	}
	if maxPR <= 0 {
		maxPR = 1.0
	}

	// Detectar comunidades temáticas e clusters sobre o grafo ativo
	nodeTypes := make(map[string]string, len(activeNodes))
	for id := range activeNodes {
		nodeTypes[id] = InferNoteType(id)
	}

	commResult := graph.DetectCommunities(nodeList, weightedEdges, graph.CommunityOptions{
		MaxIterations:  graph.DefaultCommunityMaxIter,
		MinClusterSize: 1,
		NodeTypes:      nodeTypes,
	})

	nodeToCommunity := make(map[string]graph.Community, len(activeNodes))
	for _, c := range commResult.Communities {
		for _, m := range c.Members {
			nodeToCommunity[m] = c
		}
	}

	// Montar nós e calcular métricas
	nodes := make([]Node, 0, len(activeNodes))
	maxDegree := 0
	totalDegree := 0
	hubCount := 0

	for id := range activeNodes {
		d := docMap[id]
		title := d.Title
		if title == "" {
			title = id
		}
		in := inDeg[id]
		out := outDeg[id]
		deg := in + out
		if deg > maxDegree {
			maxDegree = deg
		}
		totalDegree += deg

		noteType := InferNoteType(id)
		color := GetColorForType(noteType)
		pr := prMap[id]

		// Raio visual entre 8px e 32px
		radius := 8.0 + (pr/maxPR)*20.0
		isRoot := (rootNode != "" && id == rootNode)
		if isRoot {
			radius += 6.0
			color = "#f59e0b" // Destaque em âmbar para raiz
		}

		isHub := deg >= 4 || (pr/maxPR) > 0.6
		if isHub {
			hubCount++
		}

		comm, hasComm := nodeToCommunity[id]
		commID := 0
		commLabel := ""
		commColor := "#64748b"
		if hasComm {
			commID = comm.ID
			commLabel = comm.Label
			commColor = GetColorForCommunity(comm.ID)
		}

		nodes = append(nodes, Node{
			ID:             id,
			Title:          title,
			Type:           noteType,
			PageRank:       pr,
			InDegree:       in,
			OutDegree:      out,
			Radius:         math.Round(radius*10) / 10,
			Color:          color,
			IsRoot:         isRoot,
			IsHub:          isHub,
			CommunityID:    commID,
			CommunityLabel: commLabel,
			CommunityColor: commColor,
		})
	}

	// Montar arestas finais
	edgesFinal := make([]Edge, 0, len(activeEdges))
	for _, e := range activeEdges {
		edgeColor := "#475569" // cinza escuro
		if e.EpistemicStatus == "INFERRED" {
			edgeColor = "#818cf8" // azul/índigo suave
		}
		edgesFinal = append(edgesFinal, Edge{
			Source:          e.Source,
			Target:          e.Target,
			Relation:        e.Relation,
			EpistemicStatus: e.EpistemicStatus,
			Weight:          e.Weight,
			Color:           edgeColor,
		})
	}

	avgDeg := 0.0
	density := 0.0
	n := len(nodes)
	if n > 0 {
		avgDeg = float64(totalDegree) / float64(n)
		if n > 1 {
			density = float64(len(edgesFinal)) / float64(n*(n-1))
		}
	}

	title := "Grafo de Memória do Repositório"
	if repo != "" {
		title = fmt.Sprintf("Grafo de Memória [%s]", repo)
	}
	if rootNode != "" {
		title = fmt.Sprintf("Subgrafo Centrado em '%s' (Profundidade: %d)", rootNode, maxDepth)
	}

	return &GraphView{
		Title:       title,
		Repository:  repo,
		RootNode:    rootNode,
		MaxDepth:    maxDepth,
		GeneratedAt: time.Now().UTC(),
		Stats: GraphStats{
			TotalNodes:     len(nodes),
			TotalEdges:     len(edgesFinal),
			MaxDegree:      maxDegree,
			AvgDegree:      math.Round(avgDeg*100) / 100,
			Density:        math.Round(density*1000) / 1000,
			HubCount:       hubCount,
			CommunityCount: len(commResult.Communities),
			Modularity:     math.Round(commResult.Modularity*1000) / 1000,
		},
		Nodes:       nodes,
		Edges:       edgesFinal,
		Communities: commResult.Communities,
	}
}

// extractSubgraphBFS extrai nós alcançáveis em ambas as direções até maxDepth
func extractSubgraphBFS(docMap map[string]RawDoc, edges []RawEdge, root string, maxDepth int) map[string]bool {
	adj := make(map[string][]string)
	for _, e := range edges {
		adj[e.Source] = append(adj[e.Source], e.Target)
		adj[e.Target] = append(adj[e.Target], e.Source) // bidirecional para isolar a vizinhança
	}

	visited := make(map[string]bool)
	visited[root] = true

	queue := []string{root}
	depth := 0

	for len(queue) > 0 && depth < maxDepth {
		levelSize := len(queue)
		for i := 0; i < levelSize; i++ {
			curr := queue[0]
			queue = queue[1:]

			for _, neighbor := range adj[curr] {
				if !visited[neighbor] {
					visited[neighbor] = true
					queue = append(queue, neighbor)
				}
			}
		}
		depth++
	}

	return visited
}

// BuildFromSQLite extrai o grafo e metadados diretamente do SQLite
func BuildFromSQLite(ctx context.Context, database *sql.DB, rootNode string, maxDepth int, repo string) (*GraphView, error) {
	docRows, err := database.QueryContext(ctx, "SELECT id, title, COALESCE(updated_at, 0) FROM documents")
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar documents no sqlite: %w", err)
	}
	defer docRows.Close()

	var docs []RawDoc
	for docRows.Next() {
		var d RawDoc
		if err := docRows.Scan(&d.ID, &d.Title, &d.UpdatedAt); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}

	edgeRows, err := database.QueryContext(ctx, "SELECT source_id, target_id, COALESCE(relation, 'links_to'), COALESCE(epistemic_status, 'EXTRACTED'), COALESCE(weight, 1.0) FROM graph_edges")
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar graph_edges no sqlite: %w", err)
	}
	defer edgeRows.Close()

	var edges []RawEdge
	for edgeRows.Next() {
		var e RawEdge
		if err := edgeRows.Scan(&e.Source, &e.Target, &e.Relation, &e.EpistemicStatus, &e.Weight); err != nil {
			return nil, err
		}
		edges = append(edges, e)
	}

	return BuildGraphView(docs, edges, rootNode, maxDepth, repo), nil
}

// BuildFromPostgres extrai o grafo e metadados a partir do PostgreSQL com pgvector
func BuildFromPostgres(ctx context.Context, pgStore *store.PostgresStore, repository, rootNode string, maxDepth int) (*GraphView, error) {
	// Acessa o banco subjacente do store via GetDB se exposto ou queries
	dbConn := pgStore.DB()
	docRows, err := dbConn.QueryContext(ctx, "SELECT id, title, COALESCE(updated_at, 0) FROM documents WHERE ($1 = '' OR repository = $1)", repository)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar documents no postgres: %w", err)
	}
	defer docRows.Close()

	var docs []RawDoc
	for docRows.Next() {
		var d RawDoc
		if err := docRows.Scan(&d.ID, &d.Title, &d.UpdatedAt); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}

	edgeRows, err := dbConn.QueryContext(ctx, "SELECT source_id, target_id, COALESCE(relation, 'links_to'), COALESCE(epistemic_status, 'EXTRACTED'), COALESCE(weight, 1.0) FROM graph_edges WHERE ($1 = '' OR repository = $1)", repository)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar graph_edges no postgres: %w", err)
	}
	defer edgeRows.Close()

	var edges []RawEdge
	for edgeRows.Next() {
		var e RawEdge
		if err := edgeRows.Scan(&e.Source, &e.Target, &e.Relation, &e.EpistemicStatus, &e.Weight); err != nil {
			return nil, err
		}
		edges = append(edges, e)
	}

	return BuildGraphView(docs, edges, rootNode, maxDepth, repository), nil
}

// FindCommunityLeader identifica o nó com maior centralidade/PageRank dentro da comunidade
func FindCommunityLeader(members []string, prScores map[string]float64) string {
	if len(members) == 0 {
		return ""
	}
	lead := members[0]
	maxPR := prScores[lead]
	for _, m := range members[1:] {
		pr := prScores[m]
		if pr > maxPR || (pr == maxPR && m < lead) {
			maxPR = pr
			lead = m
		}
	}
	return lead
}

// FindDominantType identifica o tipo de nota mais frequente dentro do grupo de membros
func FindDominantType(members []string, nodeTypes map[string]string) string {
	if len(members) == 0 || len(nodeTypes) == 0 {
		return ""
	}
	counts := make(map[string]int)
	for _, m := range members {
		t := nodeTypes[m]
		if t != "" {
			counts[t]++
		}
	}
	maxCount := 0
	dominant := ""
	for t, count := range counts {
		if count > maxCount || (count == maxCount && (dominant == "" || t < dominant)) {
			maxCount = count
			dominant = t
		}
	}
	return dominant
}
