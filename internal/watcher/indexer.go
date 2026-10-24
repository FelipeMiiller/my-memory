package watcher

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/embedder"
	"github.com/FelipeMiiller/my-memory/internal/parser"
	"github.com/FelipeMiiller/my-memory/internal/store"
	"github.com/FelipeMiiller/my-memory/internal/turboquant"
)

// IndexResult reporta o resultado da indexação ou purga cirúrgica de uma nota
type IndexResult struct {
	Action      string        // "indexed", "cached", "purged", "skipped"
	DocID       string        // Identificador do documento (caminho relativo ou absoluto)
	Title       string        // Título extraído do arquivo
	EdgesCount  int           // Quantidade de arestas de grafo extraídas
	ChunksCount int           // Quantidade de chunks gerados
	Duration    time.Duration // Tempo de execução da operação
}

// IndexSingleFileSQLite reindexa cirurgicamente uma única nota no SQLite
func IndexSingleFileSQLite(
	ctx context.Context,
	database *sql.DB,
	emb embedder.Embedder,
	tq *turboquant.Quantizer,
	filePath string,
	force bool,
) (*IndexResult, error) {
	start := time.Now()

	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler arquivo para indexação cirúrgica: %w", err)
	}

	docID := filePath
	title := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	// ISSUE-010 (2026-09-18): usar frontmatter.title quando existir (alinhado com
	// preCollectDocTitles e ListDocumentTitles). Sem isso, fuzzy substring match
	// grava edges tagged_as com target longo (frontmatter.title) enquanto o DB
	// armazena basename em documents.title — gerando dead links sistemicos.
	if len(contentBytes) > 0 {
		if fmProbe, _ := parser.ExtractFrontmatter(string(contentBytes)); fmProbe != nil && fmProbe.Title != "" {
			title = fmProbe.Title
		}
	}
	currentHash := store.CalculateContentHash(contentBytes)

	// Verificação de cache incremental via SHA-256
	if !force {
		storedHash, err := db.GetDocumentHash(ctx, database, docID)
		if err == nil && storedHash != "" && storedHash == currentHash {
			return &IndexResult{
				Action:   "cached",
				DocID:    docID,
				Title:    title,
				Duration: time.Since(start),
			}, nil
		}
	}

	content := string(contentBytes)
	fm, body := parser.ExtractFrontmatter(content)
	category := "resource"
	if fm != nil && fm.Category != "" {
		category = fm.Category
	}
	if category != "resource" && category != "memory" && category != "skill" {
		category = "resource"
	}
	var abstract string
	if fm != nil && fm.Summary != "" {
		abstract = fm.Summary
	} else if fm != nil && fm.Abstract != "" {
		abstract = fm.Abstract
	} else {
		abstract = parser.ExtractMicroAbstract(body, 160)
	}

	// 1. Limpa dados anteriores do documento para reindexação limpa
	_ = db.DeleteDocumentData(ctx, database, docID)

	// Metadados reais do arquivo (ADR-057): o mtime do disco é o instante em que
	// o AUTOR editou a nota, ao contrário de updated_at, que é o instante em que
	// nós indexamos. Sem esta distinção o decaimento temporal mede a indexação.
	mtime, size := store.FileStatValues(filePath)

	// 2. Salva documento com content_hash, embed_epoch, abstract e category
	if err := db.InsertDocumentWithMeta(ctx, database, docID, filePath, title, time.Now().Unix(), currentHash, embedder.EpochFor(emb), abstract, category, mtime, size); err != nil {
		return nil, fmt.Errorf("erro ao inserir documento SQLite: %w", err)
	}

	// 3. Extrai e salva conexões do grafo
	connections := parser.ExtractConnections(content)

	// 3.1 Tags: a lista completa (órfãs incluídas) é gravada antes do fuzzy —
	// a aresta `tagged_as` continua sendo só a das tags resolvidas (ADR-041).
	titleToID, _ := db.ListDocumentIDTitles(ctx, database)
	_ = db.InsertDocumentTags(ctx, database, docID, buildTagBindings(connections.Tags, titleToID))

	// 3.2 Fuzzy resolve de tags: edges `tagged_as` apontando para tags como
	// `sqlite` são reescritas para o doc correspondente (ex: ADR-001) quando
	// possível, evitando dead links e aumentando a densidade do grafo.
	if titles, err := db.ListDocumentTitles(ctx, database, ""); err == nil {
		parser.ResolveTagConnections(&connections, titles)
	}

	for _, edge := range connections.Edges {
		_ = db.InsertEdgeWithProps(ctx, database, docID, edge.Target, edge.Relation, edge.EpistemicStatus, edge.Weight)
	}

	// 4. Divide em chunks e indexa no FTS e vetorial (com fallback para FTS se offline)
	chunks := parser.ChunkText(content, 200, 30)
	for i, c := range chunks {
		chunkID := fmt.Sprintf("%s#%d", docID, i)
		var vec []float32
		if emb != nil {
			v, err := emb.GenerateEmbedding(c)
			if err == nil {
				vec = v
			}
		}
		if vec == nil {
			// embedder.EmbeddingDim, não literal: o fallback sem provider é o caminho
			// mais exercitado em máquina sem Ollama/ONNX, e um literal divergente
			// grava vetor de dimensão errada no schema.
			vec = make([]float32, embedder.EmbeddingDim)
		}

		_ = db.InsertChunk(ctx, database, chunkID, docID, c, i, vec)

		if tq != nil {
			cv, err := tq.Quantize(vec)
			if err == nil {
				_ = db.InsertTurboQuantChunk(ctx, database, chunkID, cv.Scale, cv.Data)
			}
		}
	}

	// 5. Ponte notas↔código (CG-17/CG-18): o documento novo é casado contra
	// os code_symbols já indexados e ganha a aresta tipada
	// `references_code` no grafo único. Idempotente — uma falha aqui não
	// derruba a indexação da nota.
	_, _ = db.BuildCodeBridgeForDocument(ctx, database, docID)

	return &IndexResult{
		Action:      "indexed",
		DocID:       docID,
		Title:       title,
		EdgesCount:  len(connections.Edges),
		ChunksCount: len(chunks),
		Duration:    time.Since(start),
	}, nil
}

// PurgeSingleFileSQLite remove completamente uma nota, seus chunks e arestas do SQLite
func PurgeSingleFileSQLite(ctx context.Context, database *sql.DB, filePath string) (*IndexResult, error) {
	start := time.Now()
	docID := filePath
	title := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))

	// Tags: remoção explícita. O cascade do SQLite cobre a purge, mas a chamada
	// explícita é a garantia comum aos dois engines.
	_ = db.DeleteDocumentTags(ctx, database, docID)

	if err := db.DeleteDocumentComplete(ctx, database, docID); err != nil {
		return nil, fmt.Errorf("erro ao purgar documento SQLite: %w", err)
	}

	return &IndexResult{
		Action:   "purged",
		DocID:    docID,
		Title:    title,
		Duration: time.Since(start),
	}, nil
}

// IndexSingleFilePostgres reindexa cirurgicamente uma única nota no PostgreSQL com pgvector
func IndexSingleFilePostgres(
	ctx context.Context,
	s *store.PostgresStore,
	emb embedder.Embedder,
	targetRepo string,
	filePath string,
	force bool,
) (*IndexResult, error) {
	start := time.Now()

	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler arquivo para indexação cirúrgica: %w", err)
	}

	docID := filePath
	title := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	// ISSUE-010 (2026-09-18): usar frontmatter.title quando existir (alinhado com
	// preCollectDocTitles e ListDocumentTitles). Sem isso, fuzzy substring match
	// grava edges tagged_as com target longo (frontmatter.title) enquanto o DB
	// armazena basename em documents.title — gerando dead links sistemicos.
	if len(contentBytes) > 0 {
		if fmProbe, _ := parser.ExtractFrontmatter(string(contentBytes)); fmProbe != nil && fmProbe.Title != "" {
			title = fmProbe.Title
		}
	}
	currentHash := store.CalculateContentHash(contentBytes)

	if !force {
		storedHash, err := s.GetDocumentHash(ctx, targetRepo, docID)
		if err == nil && storedHash != "" && storedHash == currentHash {
			return &IndexResult{
				Action:   "cached",
				DocID:    docID,
				Title:    title,
				Duration: time.Since(start),
			}, nil
		}
	}

	content := string(contentBytes)
	fm, body := parser.ExtractFrontmatter(content)
	category := "resource"
	if fm != nil && fm.Category != "" {
		category = fm.Category
	}
	if category != "resource" && category != "memory" && category != "skill" {
		category = "resource"
	}
	var abstract string
	if fm != nil && fm.Summary != "" {
		abstract = fm.Summary
	} else if fm != nil && fm.Abstract != "" {
		abstract = fm.Abstract
	} else {
		abstract = parser.ExtractMicroAbstract(body, 160)
	}

	_ = s.DeleteDocumentData(ctx, targetRepo, docID)

	mtime, size := store.FileStatValues(filePath)

	if err := s.InsertDocumentWithMeta(ctx, targetRepo, docID, filePath, title, time.Now().Unix(), currentHash, embedder.EpochFor(emb), abstract, category, mtime, size); err != nil {
		return nil, fmt.Errorf("erro ao inserir documento Postgres: %w", err)
	}
	connections := parser.ExtractConnections(content)

	// Tags: a lista completa (órfãs incluídas) é gravada antes do fuzzy — a
	// aresta `tagged_as` continua sendo só a das tags resolvidas (ADR-041).
	titleToID, _ := s.ListDocumentIDTitles(ctx, targetRepo)
	_ = s.InsertDocumentTags(ctx, targetRepo, docID, buildTagBindings(connections.Tags, titleToID))

	// Fuzzy resolve de tags: ver IndexSingleFileSQLite para detalhes.
	if titles, err := s.ListDocumentTitles(ctx, targetRepo); err == nil {
		parser.ResolveTagConnections(&connections, titles)
	}

	for _, edge := range connections.Edges {
		_ = s.InsertEdgeWithProps(ctx, targetRepo, docID, edge.Target, edge.Relation, edge.EpistemicStatus, edge.Weight)
	}

	chunks := parser.ChunkText(content, 200, 30)
	for i, c := range chunks {
		chunkID := fmt.Sprintf("%s#%d", docID, i)
		var vec []float32
		if emb != nil {
			v, err := emb.GenerateEmbedding(c)
			if err == nil {
				vec = v
			}
		}
		if vec == nil {
			// embedder.EmbeddingDim, não literal: o fallback sem provider é o caminho
			// mais exercitado em máquina sem Ollama/ONNX, e um literal divergente
			// grava vetor de dimensão errada no schema.
			vec = make([]float32, embedder.EmbeddingDim)
		}

		_ = s.InsertChunk(ctx, targetRepo, chunkID, docID, c, i, vec)
	}

	// Ponte notas↔código (CG-17/CG-18) — o mesmo contrato do caminho
	// SQLite. O contexto carrega o repository do vault porque a ponte, no
	// PostgreSQL, é escopada por repositório.
	_, _ = db.BuildCodeBridgeForDocument(db.WithRepository(ctx, targetRepo), s.DB(), docID)

	return &IndexResult{
		Action:      "indexed",
		DocID:       docID,
		Title:       title,
		EdgesCount:  len(connections.Edges),
		ChunksCount: len(chunks),
		Duration:    time.Since(start),
	}, nil
}

// PurgeSingleFilePostgres remove chunks e arestas de uma nota no PostgreSQL
func PurgeSingleFilePostgres(
	ctx context.Context,
	s *store.PostgresStore,
	targetRepo string,
	filePath string,
) (*IndexResult, error) {
	start := time.Now()
	docID := filePath
	title := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))

	// Tags: o Postgres não tem cascade em `documents`, então a remoção das tags do
	// documento purgado é explícita.
	_ = s.DeleteDocumentTags(ctx, targetRepo, docID)

	_ = s.DeleteDocumentData(ctx, targetRepo, docID)

	return &IndexResult{
		Action:   "purged",
		DocID:    docID,
		Title:    title,
		Duration: time.Since(start),
	}, nil
}

// buildTagBindings monta os vínculos documento→tag a partir das tags cruas do
// documento e do mapa título → id dos documentos indexados.
//
// A lista completa é gravada, tags órfãs incluídas: no Obsidian a tag existe
// mesmo sem casa no grafo. A aresta `tagged_as` continua sendo só a das tags
// resolvidas (ADR-041) — o que muda aqui é a tabela `document_tags` (ADR-055).
//
// `resolved_doc_id` é o id do documento que a tag NOMEIA (não do documento que
// carrega a tag). Tag órfã fica com o campo vazio.
func buildTagBindings(tags []string, titleToID map[string]string) []store.TagBinding {
	if len(tags) == 0 {
		return nil
	}

	resolved := parser.ResolveTagTargets(tags, tagTargetTitles(titleToID))

	bindings := make([]store.TagBinding, 0, len(tags))
	for _, raw := range tags {
		key, ok := parser.NormalizeTag(raw)
		if !ok {
			// Tag inválida segundo a norma do Obsidian não tem forma canônica,
			// logo não tem identidade para ser gravada.
			continue
		}
		binding := store.NormalizeTagBinding(store.TagBinding{Tag: raw, Key: key})
		if title, found := resolved[key]; found && title != "" {
			binding.ResolvedDocID = titleToID[title]
		}
		bindings = append(bindings, binding)
	}
	return bindings
}

// tagTargetTitles devolve os títulos do mapa título → id em ordem alfabética:
// a lista alimenta o fuzzy resolve, que precisa ser determinístico.
func tagTargetTitles(titleToID map[string]string) []string {
	titles := make([]string, 0, len(titleToID))
	for title := range titleToID {
		titles = append(titles, title)
	}
	sort.Strings(titles)
	return titles
}
