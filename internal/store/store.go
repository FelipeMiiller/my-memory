package store

import (
	"context"
)

// SearchResult representa um trecho relevante retornado na busca
type SearchResult struct {
	ChunkID    string   `json:"chunk_id"`
	DocumentID string   `json:"document_id"`
	Repository string   `json:"repository,omitempty"`
	Content    string   `json:"content"`
	Distance   float64  `json:"distance,omitempty"`
	Score      float64  `json:"score,omitempty"`      // Pontuação acumulada de RRF
	Sources    []string `json:"sources,omitempty"`    // Origens e posições (ex: ["fts:1", "vector:3"])
	Neighbors  []string `json:"neighbors,omitempty"`
}

// Store define o contrato agnóstico de armazenamento para SQLite e PostgreSQL
type Store interface {
	// InsertDocument insere ou atualiza um documento atrelado ao repositório
	InsertDocument(ctx context.Context, repo, id, path, title string, updatedAt int64) error

	// InsertChunk insere um pedaço de texto e seu vetor de embedding
	InsertChunk(ctx context.Context, repo, chunkID, docID, content string, index int, vec []float32) error

	// InsertEdge adiciona uma aresta no grafo relacional
	InsertEdge(ctx context.Context, repo, sourceID, targetID, relation string) error

	// SearchKNN busca os K pedaços mais próximos vetorialmente (se repo != "", filtra por repositório)
	SearchKNN(ctx context.Context, repo string, queryVec []float32, limit int) ([]SearchResult, error)

	// SearchFTS busca trechos via texto completo (FTS5 no SQLite / tsvector no Postgres)
	SearchFTS(ctx context.Context, repo string, query string, limit int) ([]SearchResult, error)

	// SearchHybridRRF executa busca híbrida fundindo FTS, vetores e grafo via RRF
	SearchHybridRRF(ctx context.Context, repo string, query string, queryVec []float32, limit int, k int) ([]SearchResult, error)

	// GetNodeNeighbors executa busca recursiva de nós vizinhos conectados via CTE
	GetNodeNeighbors(ctx context.Context, repo string, nodeID string, maxDepth int) ([]string, error)

	// Close encerra a conexão com o banco de dados
	Close() error
}
