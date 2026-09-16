package federation

import "github.com/FelipeMiiller/my-memory/internal/deeplink"

// SearchParams agrupa os parâmetros da busca para flexibilidade de múltiplos modos
type SearchParams struct {
	Repo        string
	Query       string
	Mode        string // "hybrid" (default), "vector", "fts"
	Limit       int
	K           int     // constante RRF, default 60
	Decay       bool    // ativa decaimento temporal exponencial
	HalfLife    float64 // meia-vida em dias (padrão: 30.0)
	DecayWeight float64 // peso do decaimento temporal w in [0.0, 1.0] (padrão: 0.3)
	DetailLevel string  // "l0", "l1", "l2" (default: "l1")
	Category    string  // "resource", "memory", "skill" ou ""
}

// SearchResult representa um item de resultado de busca no vault
type SearchResult struct {
	ChunkID        string              `json:"chunk_id,omitempty"`
	DocumentID     string              `json:"document_id,omitempty"`
	Repository     string              `json:"repository,omitempty"`
	Content        string              `json:"content,omitempty"`
	Distance       float64             `json:"distance,omitempty"`
	Score          float64             `json:"score,omitempty"`
	Sources        []string            `json:"sources,omitempty"`
	Neighbors      []string            `json:"neighbors,omitempty"`
	UpdatedAt      int64               `json:"updated_at,omitempty"`
	Abstract       string              `json:"abstract,omitempty"`
	Category       string              `json:"category,omitempty"`
	Links          *deeplink.DeepLinks `json:"links,omitempty"`
	IsCentralVault bool                `json:"is_central_vault,omitempty"`
	ID             string              `json:"id,omitempty"`
	Path           string              `json:"path,omitempty"`
	Title          string              `json:"title,omitempty"`
	Excerpt        string              `json:"excerpt,omitempty"`
	FTSScore       float64             `json:"fts_score,omitempty"`
	VectorScore    float64             `json:"vector_score,omitempty"`
	Summary        string              `json:"summary,omitempty"`
	DecayedScore   float64             `json:"decayed_score,omitempty"`
	DecayFactor    float64             `json:"decay_factor,omitempty"`
	DaysOld        float64             `json:"days_old,omitempty"`
}
