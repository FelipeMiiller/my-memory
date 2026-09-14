package store

import (
	"context"
	"math"
	"strings"
	"unicode"

	"github.com/FelipeMiiller/my-memory/internal/graph"
)

// Document representa a entidade de um documento armazenado
type Document struct {
	ID          string `json:"id"`
	Repository  string `json:"repository,omitempty"`
	Path        string `json:"path"`
	Title       string `json:"title"`
	UpdatedAt   int64  `json:"updated_at"`
	ContentHash string `json:"content_hash,omitempty"`
	Abstract    string `json:"abstract,omitempty"`
	Category    string `json:"category,omitempty"`
}

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
	UpdatedAt  int64    `json:"updated_at,omitempty"`
	Abstract   string   `json:"abstract,omitempty"`
	Category   string   `json:"category,omitempty"`
}

// GodNode representa um nó com alta centralidade estrutural (in-degree + out-degree) no grafo
type GodNode struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	InDegree    int    `json:"in_degree"`
	OutDegree   int    `json:"out_degree"`
	TotalDegree int    `json:"total_degree"`
}

// PageRankNode representa um nó com autoridade calculada via algoritmo de PageRank ponderado
type PageRankNode struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Score     float64 `json:"score"`
	Rank      int     `json:"rank"`
	InDegree  int     `json:"in_degree"`
	OutDegree int     `json:"out_degree"`
}

// SurprisingConnection representa uma conexão semântica/conceitual latente sem link direto no grafo
type SurprisingConnection struct {
	SourceID   string  `json:"source_id"`
	SourceName string  `json:"source_name"`
	TargetID   string  `json:"target_id"`
	TargetName string  `json:"target_name"`
	Similarity float64 `json:"similarity"`
	Reason     string  `json:"reason"`
}

// DeadLink representa um link no grafo para uma nota ou alvo inexistente
type DeadLink struct {
	SourceID string `json:"source_id"`
	TargetID string `json:"target_id"`
	Relation string `json:"relation"`
}

// OrphanNote representa uma nota isolada com zero conexões de entrada e saída
type OrphanNote struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// SelfLoop representa uma aresta circular onde a nota aponta para si mesma
type SelfLoop struct {
	NodeID   string `json:"node_id"`
	Relation string `json:"relation"`
}

// DesyncedChunk representa um chunk que existe no banco relacional mas carece de vetor ou TurboQuant
type DesyncedChunk struct {
	ChunkID    string `json:"chunk_id"`
	DocumentID string `json:"document_id"`
	Issue      string `json:"issue"`
}

// DoctorReport agrega as métricas estruturais e anomalias de saúde do grafo
type DoctorReport struct {
	TotalDocuments int             `json:"total_documents"`
	TotalChunks    int             `json:"total_chunks"`
	TotalEdges     int             `json:"total_edges"`
	TotalNodes     int             `json:"total_nodes"`
	HealthScore    int             `json:"health_score"` // 0 a 100
	DeadLinks      []DeadLink      `json:"dead_links"`
	OrphanNotes    []OrphanNote    `json:"orphan_notes"`
	SelfLoops      []SelfLoop      `json:"self_loops"`
	DesyncedChunks []DesyncedChunk `json:"desynced_chunks"`
}

// Store define o contrato agnóstico de armazenamento para SQLite e PostgreSQL
type Store interface {
	// InsertDocument insere ou atualiza um documento atrelado ao repositório
	InsertDocument(ctx context.Context, repo, id, path, title string, updatedAt int64, contentHash string) error

	// InsertDocumentWithMeta insere ou atualiza um documento com metadados de abstract (L0) e categoria
	InsertDocumentWithMeta(ctx context.Context, repo, id, path, title string, updatedAt int64, contentHash, abstract, category string) error

	// GetDocumentHash retorna o hash SHA-256 armazenado de um documento (ou "" se não existir)
	GetDocumentHash(ctx context.Context, repo, id string) (string, error)

	// DeleteDocumentData remove chunks e arestas originadas do documento para reindexação limpa
	DeleteDocumentData(ctx context.Context, repo, id string) error

	// PruneDeletedDocuments remove documentos, chunks e arestas de arquivos ausentes do disco
	PruneDeletedDocuments(ctx context.Context, repo, rootDir string, activeDocIDs []string) ([]string, error)

	// InsertChunk insere um pedaço de texto e seu vetor de embedding
	InsertChunk(ctx context.Context, repo, chunkID, docID, content string, index int, vec []float32) error

	// InsertEdge adiciona uma aresta padrão no grafo relacional
	InsertEdge(ctx context.Context, repo, sourceID, targetID, relation string) error

	// InsertEdgeWithProps adiciona uma aresta tipada com status epistêmico e peso no grafo
	InsertEdgeWithProps(ctx context.Context, repo, sourceID, targetID, relation, epistemicStatus string, weight float64) error

	// GetGodNodes retorna os nós centrais com maior centralidade de conexões
	GetGodNodes(ctx context.Context, repo string, limit int) ([]GodNode, error)

	// ComputePageRank calcula os nós mais autoritativos via algoritmo de PageRank ponderado
	ComputePageRank(ctx context.Context, repo string, damping float64, maxIter int) ([]PageRankNode, error)

	// FindSurprisingConnections descobre conexões latentes entre documentos conceitualmente similares sem arestas no grafo
	FindSurprisingConnections(ctx context.Context, repo string, limit int, minSimilarity float64) ([]SurprisingConnection, error)

	// SearchKNN busca os K pedaços mais próximos vetorialmente (se repo != "", filtra por repositório)
	SearchKNN(ctx context.Context, repo string, queryVec []float32, limit int) ([]SearchResult, error)

	// SearchFTS busca trechos via texto completo (FTS5 no SQLite / tsvector no Postgres)
	SearchFTS(ctx context.Context, repo string, query string, limit int) ([]SearchResult, error)

	// SearchHybridRRF executa busca híbrida fundindo FTS, vetores e grafo via RRF
	SearchHybridRRF(ctx context.Context, repo string, query string, queryVec []float32, limit int, k int) ([]SearchResult, error)

	// SearchHybridRRFWithDecay executa busca híbrida com RRF ponderado por decaimento temporal
	SearchHybridRRFWithDecay(ctx context.Context, repo string, query string, queryVec []float32, limit int, k int, opts DecayOptions) ([]SearchResult, error)

	// GetNodeNeighbors executa busca recursiva de nós vizinhos conectados via CTE
	GetNodeNeighbors(ctx context.Context, repo string, nodeID string, maxDepth int) ([]string, error)

	// DiagnoseHealth audita a integridade do grafo gerando um relatório de saúde
	DiagnoseHealth(ctx context.Context, repo string) (*DoctorReport, error)

	// CalculateImpact calcula o fechamento de dependências reversas e score de risco (blast radius)
	CalculateImpact(ctx context.Context, repo string, targetQuery string, maxDepth int) (*graph.ImpactResult, error)

	// InspectNode constrói a visualização cirúrgica em 3 colunas (Triptych) de um nó
	InspectNode(ctx context.Context, repo string, targetQuery string, maxContentLen int) (*graph.TriptychView, error)

	// FixHealthIssues repara problemas comuns como self-loops e links mortos
	FixHealthIssues(ctx context.Context, repo string) (int, error)

	// Close encerra a conexão com o banco de dados
	Close() error
}

// CosineSimilarity calcula a similaridade de cosseno entre dois vetores float32
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		va := float64(a[i])
		vb := float64(b[i])
		dot += va * vb
		normA += va * va
		normB += vb * vb
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// CalculateJaccardSimilarity calcula a similaridade léxica de Jaccard baseada no conjunto de palavras
func CalculateJaccardSimilarity(textA, textB string) float64 {
	wordsA := TokenizeWords(textA)
	wordsB := TokenizeWords(textB)
	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0
	}
	union := make(map[string]bool)
	for w := range wordsA {
		union[w] = true
	}
	for w := range wordsB {
		union[w] = true
	}
	if len(union) == 0 {
		return 0
	}
	intersection := 0
	for w := range wordsA {
		if wordsB[w] {
			intersection++
		}
	}
	return float64(intersection) / float64(len(union))
}

// TokenizeWords extrai palavras minúsculas únicas com 3 ou mais caracteres alfanuméricos
func TokenizeWords(s string) map[string]bool {
	words := make(map[string]bool)
	var current strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(r)
		} else if current.Len() > 0 {
			w := current.String()
			if len(w) >= 3 {
				words[w] = true
			}
			current.Reset()
		}
	}
	if current.Len() >= 3 {
		words[current.String()] = true
	}
	return words
}
