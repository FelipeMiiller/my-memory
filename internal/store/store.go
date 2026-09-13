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
	Score      float64  `json:"score,omitempty"`   // Pontuação acumulada de RRF
	Sources    []string `json:"sources,omitempty"` // Origens e posições (ex: ["fts:1", "vector:3"])
	Neighbors  []string `json:"neighbors,omitempty"`
}

// GodNode representa um nó com alta centralidade estrutural (in-degree + out-degree) no grafo
type GodNode struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	InDegree    int    `json:"in_degree"`
	OutDegree   int    `json:"out_degree"`
	TotalDegree int    `json:"total_degree"`
}

// Store define o contrato agnóstico de armazenamento para SQLite e PostgreSQL
type Store interface {
	// InsertDocument insere ou atualiza um documento atrelado ao repositório
	InsertDocument(ctx context.Context, repo, id, path, title string, updatedAt int64, contentHash string) error

	// GetDocumentHash retorna o hash SHA-256 armazenado de um documento (ou "" se não existir)
	GetDocumentHash(ctx context.Context, repo, id string) (string, error)

	// DeleteDocumentData remove chunks e arestas originadas do documento para reindexação limpa
	DeleteDocumentData(ctx context.Context, repo, id string) error

	// InsertChunk insere um pedaço de texto e seu vetor de embedding
	InsertChunk(ctx context.Context, repo, chunkID, docID, content string, index int, vec []float32) error

	// InsertEdge adiciona uma aresta padrão no grafo relacional
	InsertEdge(ctx context.Context, repo, sourceID, targetID, relation string) error

	// InsertEdgeWithProps adiciona uma aresta tipada com status epistêmico e peso no grafo
	InsertEdgeWithProps(ctx context.Context, repo, sourceID, targetID, relation, epistemicStatus string, weight float64) error

	// GetGodNodes retorna os nós centrais com maior centralidade de conexões
	GetGodNodes(ctx context.Context, repo string, limit int) ([]GodNode, error)

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
