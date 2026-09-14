package graphview

import (
	"time"

	"github.com/FelipeMiiller/my-memory/internal/graph"
)

// GraphView contém os dados estruturados do grafo prontos para renderização visual
type GraphView struct {
	Title       string            `json:"title"`
	Repository  string            `json:"repository,omitempty"`
	RootNode    string            `json:"root_node,omitempty"`
	MaxDepth    int               `json:"max_depth,omitempty"`
	GeneratedAt time.Time         `json:"generated_at"`
	Stats       GraphStats        `json:"stats"`
	Nodes       []Node            `json:"nodes"`
	Edges       []Edge            `json:"edges"`
	Communities []graph.Community `json:"communities,omitempty"`
}

// GraphStats sumariza as métricas topológicas do grafo
type GraphStats struct {
	TotalNodes     int     `json:"total_nodes"`
	TotalEdges     int     `json:"total_edges"`
	MaxDegree      int     `json:"max_degree"`
	AvgDegree      float64 `json:"avg_degree"`
	Density        float64 `json:"density"`
	HubCount       int     `json:"hub_count"`
	CommunityCount int     `json:"community_count,omitempty"`
	Modularity     float64 `json:"modularity,omitempty"`
}

// Node representa um nó na visualização do grafo
type Node struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Type           string   `json:"type"` // "concept", "decision", "guide", "reference", "other"
	Tags           []string `json:"tags,omitempty"`
	PageRank       float64  `json:"pagerank"`
	InDegree       int      `json:"in_degree"`
	OutDegree      int      `json:"out_degree"`
	Radius         float64  `json:"radius"`
	Color          string   `json:"color"`
	IsRoot         bool     `json:"is_root,omitempty"`
	IsHub          bool     `json:"is_hub,omitempty"`
	CommunityID    int      `json:"community_id,omitempty"`
	CommunityLabel string   `json:"community_label,omitempty"`
	CommunityColor string   `json:"community_color,omitempty"`
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

var communityPalette = []string{
	"#6366f1", // Indigo
	"#ec4899", // Pink
	"#14b8a6", // Teal
	"#f59e0b", // Amber
	"#8b5cf6", // Violet
	"#06b6d4", // Cyan
	"#10b981", // Emerald
	"#f43f5e", // Rose
	"#3b82f6", // Blue
	"#84cc16", // Lime
	"#d946ef", // Fuchsia
	"#eab308", // Yellow
}

// GetColorForCommunity retorna uma cor categórica determinística para o cluster
func GetColorForCommunity(communityID int) string {
	if communityID <= 0 {
		return "#64748b" // slate neutro
	}
	idx := (communityID - 1) % len(communityPalette)
	return communityPalette[idx]
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
