package store

import (
	"context"
)

// SearchResult representa um trecho relevante retornado na busca semântica
type SearchResult struct {
	ChunkID    string   `json:"chunk_id"`
	DocumentID string   `json:"document_id"`
	Repository string   `json:"repository"`
	Content    string   `json:"content"`
	Distance   float64  `json:"distance"`
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

	// GetNodeNeighbors executa busca recursiva de nós vizinhos conectados via CTE
	GetNodeNeighbors(ctx context.Context, repo string, nodeID string, maxDepth int) ([]string, error)

	// Close encerra a conexão com o banco de dados
	Close() error
}
