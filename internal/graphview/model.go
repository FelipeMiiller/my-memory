package graphview

import "time"

// GraphView contém os dados estruturados do grafo prontos para renderização visual
type GraphView struct {
	Title       string     `json:"title"`
	Repository  string     `json:"repository,omitempty"`
	RootNode    string     `json:"root_node,omitempty"`
	MaxDepth    int        `json:"max_depth,omitempty"`
	GeneratedAt time.Time  `json:"generated_at"`
	Stats       GraphStats `json:"stats"`
	Nodes       []Node     `json:"nodes"`
	Edges       []Edge     `json:"edges"`
}

// GraphStats sumariza as métricas topológicas do grafo
type GraphStats struct {
	TotalNodes int     `json:"total_nodes"`
	TotalEdges int     `json:"total_edges"`
	MaxDegree  int     `json:"max_degree"`
	AvgDegree  float64 `json:"avg_degree"`
	Density    float64 `json:"density"`
	HubCount   int     `json:"hub_count"`
}

// Node representa um nó na visualização do grafo
type Node struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Type      string   `json:"type"` // "concept", "decision", "guide", "reference", "other"
	Tags      []string `json:"tags,omitempty"`
	PageRank  float64  `json:"pagerank"`
	InDegree  int      `json:"in_degree"`
	OutDegree int      `json:"out_degree"`
	Radius    float64  `json:"radius"`
	Color     string   `json:"color"`
	IsRoot    bool     `json:"is_root,omitempty"`
	IsHub     bool     `json:"is_hub,omitempty"`
}

// Edge representa uma aresta direcionada entre dois nós
type Edge struct {
	Source          string  `json:"source"`
	Target          string  `json:"target"`
	Relation        string  `json:"relation"`
	EpistemicStatus string  `json:"epistemic_status"` // "EXTRACTED", "INFERRED"
	Weight          float64 `json:"weight"`
	Color           string  `json:"color,omitempty"`
}

// GetColorForType retorna uma cor hexadecimal harmoniosa baseada no tipo da nota
func GetColorForType(noteType string) string {
	switch noteType {
	case "decision", "adr":
		return "#f43f5e" // Rose vibrante
	case "concept":
		return "#3b82f6" // Azul vibrante
	case "guide", "howto", "tutorial":
		return "#10b981" // Esmeralda
	case "reference", "doc", "docs":
		return "#a855f7" // Púrpura
	case "synthesis", "compiled":
		return "#ec4899" // Rosa fúcsia
	case "hub", "god_node":
		return "#eab308" // Dourado
	default:
		return "#64748b" // Slate neutro
	}
}
