package db

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/deeplink"
	"github.com/FelipeMiiller/my-memory/internal/embedder"
	"github.com/FelipeMiiller/my-memory/internal/graph"
	"github.com/FelipeMiiller/my-memory/internal/store"
	"github.com/FelipeMiiller/my-memory/internal/turboquant"
	"github.com/FelipeMiiller/my-memory/internal/vaultfs"
)

type SearchResult struct {
	ChunkID    string              `json:"chunk_id"`
	DocumentID string              `json:"document_id"`
	Content    string              `json:"content"`
	Distance   float64             `json:"distance,omitempty"`
	Score      float64             `json:"score,omitempty"`
	Sources    []string            `json:"sources,omitempty"`
	Neighbors  []string            `json:"neighbors,omitempty"`  // Conexões descobertas no grafo
	UpdatedAt  int64               `json:"updated_at,omitempty"` // Instante da indexação
	MTime      int64               `json:"mtime,omitempty"`      // Instante da última edição do arquivo (ADR-057); 0 = desconhecido
	Abstract   string              `json:"abstract,omitempty"`
	Category   string              `json:"category,omitempty"`
	Links      *deeplink.DeepLinks `json:"links,omitempty"`
	// Identidade do símbolo de código quando o item veio da fonte "code"
	// do fan-in RRF (T14/CG-19). Vazio para notas.
	CodeSymbol   string `json:"code_symbol,omitempty"`
	CodeFile     string `json:"code_file,omitempty"`
	CodeLine     int    `json:"code_line,omitempty"`
	CodeEndLine  int    `json:"code_end_line,omitempty"`
	CodeKind     string `json:"code_kind,omitempty"`
	CodeLanguage string `json:"code_language,omitempty"`
}

// SearchKNN busca os pedaços mais próximos usando sqlite-vec nativo
func SearchKNN(ctx context.Context, db *sql.DB, queryVec []float32, limit int) ([]SearchResult, error) {
	return SearchKNNWithOptions(ctx, db, queryVec, limit, store.SearchOptions{Level: "l1"})
}

// SearchKNNWithOptions busca os pedaços mais próximos usando sqlite-vec nativo com opções de busca
func SearchKNNWithOptions(ctx context.Context, db *sql.DB, queryVec []float32, limit int, searchOpts store.SearchOptions) ([]SearchResult, error) {
	if !HasSqliteVec {
		return nil, fmt.Errorf("sqlite-vec não está disponível nesta plataforma (use TurboQuant)")
	}

	vecBlob, err := serializeFloat32(queryVec)
	if err != nil {
		return nil, fmt.Errorf("erro ao serializar vetor de busca: %w", err)
	}

	cat := strings.TrimSpace(searchOpts.Category)
	tag := strings.TrimSpace(searchOpts.Tag)
	// O padrão casa as descendentes da tag (`inbox` → `inbox/to-read`). O
	// escape é obrigatório: `_` e `%` são curingas do LIKE e a tag pode
	// conter `_` (o formato do Obsidian permite).
	tagPattern := ""
	if tag != "" {
		tagPattern = store.EscapeLikePattern(tag) + "/%"
	}
	query := `
	SELECT c.id, c.document_id, c.content, v.embedding, v.distance, COALESCE(d.updated_at, 0),
	       COALESCE(d.mtime, 0),
	       COALESCE(d.abstract, ''), COALESCE(d.category, 'resource')
	FROM chunks_vec v
	JOIN chunks c ON c.id = v.chunk_id
	LEFT JOIN documents d ON d.id = c.document_id
	WHERE v.embedding MATCH ? AND k = ?
	  AND (? = '' OR LOWER(d.category) = LOWER(?))
	  AND (? = '' OR EXISTS (
	      SELECT 1 FROM document_tags dt
	      WHERE dt.document_id = d.id
	        AND (dt.tag_key = ? OR dt.tag_key LIKE ? ESCAPE '\')
	  ))
	ORDER BY v.distance
	`

	rows, err := db.QueryContext(ctx, query, vecBlob, limit, cat, cat, tag, tag, tagPattern)
	if err != nil {
		return nil, fmt.Errorf("erro na busca vetorial: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var vetor []byte
		if err := rows.Scan(&r.ChunkID, &r.DocumentID, &r.Content, &vetor, &r.Distance, &r.UpdatedAt, &r.MTime, &r.Abstract, &r.Category); err != nil {
			return nil, err
		}

		// O chunk sem vetor real não sai da busca. O vetor zero (o defeito
		// da ISSUE-040) PREENCHE a coluna, então o KNN o devolve como
		// qualquer outro — e devolveria ruído com cara de resultado.
		//
		// O `embedding` lido aqui é o vetor GRAVADO: fora do MATCH, o
		// sqlite-vec devolve o blob armazenado, e o KNN é a única consulta
		// que carrega o vetor da consulta.
		if vetorCruzoEhZero(vetor) {
			continue
		}

		neighbors, err := GetNodeNeighbors(ctx, db, r.DocumentID, 1)
		if err == nil {
			r.Neighbors = neighbors
		}

		results = append(results, r)
	}

	return results, nil
}

// SearchTurboQuantWithOptions busca usando a projeção ortogonal e os vetores 4-bit comprimidos com opções de busca
func SearchTurboQuantWithOptions(ctx context.Context, db *sql.DB, q *turboquant.Quantizer, queryVec []float32, limit int, searchOpts store.SearchOptions) ([]SearchResult, error) {
	rotatedQuery := q.RotateQuery(queryVec)
	cat := strings.TrimSpace(searchOpts.Category)
	tag := strings.TrimSpace(searchOpts.Tag)
	// O padrão casa as descendentes da tag (`inbox` → `inbox/to-read`). O
	// escape é obrigatório: `_` e `%` são curingas do LIKE e a tag pode
	// conter `_` (o formato do Obsidian permite).
	tagPattern := ""
	if tag != "" {
		tagPattern = store.EscapeLikePattern(tag) + "/%"
	}

	rows, err := db.QueryContext(ctx, `
		SELECT tq.chunk_id, c.document_id, c.content, tq.scale, tq.data, COALESCE(d.updated_at, 0),
		       COALESCE(d.mtime, 0),
		       COALESCE(d.abstract, ''), COALESCE(d.category, 'resource')
		FROM chunks_turboquant tq
		JOIN chunks c ON c.id = tq.chunk_id
		LEFT JOIN documents d ON d.id = c.document_id
		WHERE (? = '' OR LOWER(d.category) = LOWER(?))
		  AND (? = '' OR EXISTS (
		      SELECT 1 FROM document_tags dt
		      WHERE dt.document_id = d.id
		        AND (dt.tag_key = ? OR dt.tag_key LIKE ? ESCAPE '\')
		  ))
	`, cat, cat, tag, tag, tagPattern)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar chunks turboquant: %w", err)
	}
	defer rows.Close()

	type scoredItem struct {
		chunkID    string
		documentID string
		content    string
		score      float32
		updatedAt  int64
		mtime      int64
		abstract   string
		category   string
	}

	var items []scoredItem
	for rows.Next() {
		var chunkID, docID, content string
		var scale float32
		var data []byte
		var updatedAt int64
		var mtime int64
		var abstract, category string

		if err := rows.Scan(&chunkID, &docID, &content, &scale, &data, &updatedAt, &mtime, &abstract, &category); err != nil {
			continue
		}

		// O chunk sem vetor real não sai da busca. No caminho comprimido, o
		// vetor zero tem assinatura exata (todo nibble vale 7 — todo byte
		// vale 0x77), então ele é reconhecível sem descompactar.
		//
		// E é por este caminho que o chunk voltava: no Windows
		// `db.HasSqliteVec` é false, então a busca vetorial lê
		// `chunks_turboquant` — não `chunks_vec`.
		if turboQuantEhZero(data, embedder.EmbeddingDim) {
			continue
		}

		cv := &turboquant.CompressedVector{
			Dim:   len(queryVec),
			Scale: scale,
			Data:  data,
		}

		dot := q.DotProduct(rotatedQuery, cv)
		items = append(items, scoredItem{
			chunkID:    chunkID,
			documentID: docID,
			content:    content,
			score:      dot,
			updatedAt:  updatedAt,
			mtime:      mtime,
			abstract:   abstract,
			category:   category,
		})
	}

	// Ordena por maior similaridade (produto escalar)
	sort.Slice(items, func(i, j int) bool {
		return items[i].score > items[j].score
	})

	if len(items) > limit {
		items = items[:limit]
	}

	var results []SearchResult
	for _, it := range items {
		// Distância invertida para visualização consistente (1 - cos_sim normalizado)
		dist := float64(1.0 - it.score)
		neighbors, _ := GetNodeNeighbors(ctx, db, it.documentID, 1)

		results = append(results, SearchResult{
			ChunkID:    it.chunkID,
			DocumentID: it.documentID,
			Content:    it.content,
			Distance:   dist,
			Neighbors:  neighbors,
			UpdatedAt:  it.updatedAt,
			MTime:      it.mtime,
			Abstract:   it.abstract,
			Category:   it.category,
		})
	}

	return results, nil
}

// GetNodeNeighbors realiza travessia de grafo em SQL usando Recursive CTE
func GetNodeNeighbors(ctx context.Context, db *sql.DB, nodeID string, maxDepth int) ([]string, error) {
	cteQuery := `
	WITH RECURSIVE traversal AS (
		SELECT target_id, 1 AS depth
		FROM graph_edges
		WHERE source_id = ?
		
		UNION
		
		SELECT e.target_id, t.depth + 1
		FROM graph_edges e
		JOIN traversal t ON e.source_id = t.target_id
		WHERE t.depth < ?
	)
	SELECT DISTINCT target_id FROM traversal;
	`

	rows, err := db.QueryContext(ctx, cteQuery, nodeID, maxDepth)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var neighbors []string
	for rows.Next() {
		var target string
		if err := rows.Scan(&target); err == nil {
			neighbors = append(neighbors, target)
		}
	}
	return neighbors, nil
}

// chunksSemVetorReal lista os chunks que não têm vetor real — nem ausente,
// nem vetor zero.
//
// O SQLite guarda o vetor em DOIS caminhos, e a busca vetorial lê um ou
// outro conforme a plataforma: o cru (`chunks_vec`) quando há sqlite-vec e
// o comprimido (`chunks_turboquant`) no Windows, onde `HasSqliteVec` é
// false. O doctor olha os dois, porque "não tem vetor" precisa ter o mesmo
// significado nos dois caminhos.
//
// Três estados dessincronizados:
//   - sem vetor nenhum: não tem linha em `chunks_vec` nem em
//     `chunks_turboquant` (falta provider);
//   - vetor zero: a linha existe, então "existe" não denuncia nada — é
//     preciso comparar com o vetor zero do schema;
//   - vetor cru sem o comprimido: a busca no caminho comprimido (o do
//     Windows) não encontraria o chunk.
//
// A comparação fica em Go porque o `chunks_vec` muda de forma conforme a
// plataforma: tabela virtual `vec0` na Unix com cgo, tabela BLOB no resto.
func chunksSemVetorReal(ctx context.Context, db *sql.DB) ([]store.DesyncedChunk, error) {
	type chunkRef struct {
		id  string
		doc string
	}

	var chunks []chunkRef
	rows, err := db.QueryContext(ctx, `SELECT id, document_id FROM chunks ORDER BY id`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var c chunkRef
		if err := rows.Scan(&c.id, &c.doc); err == nil {
			chunks = append(chunks, c)
		}
	}
	rows.Close()

	// O vetor cru, por chunk.
	vetoresCruzos := map[string][]byte{}
	if rawRows, err := db.QueryContext(ctx, `SELECT chunk_id, embedding FROM chunks_vec`); err == nil {
		for rawRows.Next() {
			var chunkID string
			var blob []byte
			if err := rawRows.Scan(&chunkID, &blob); err == nil {
				vetoresCruzos[chunkID] = blob
			}
		}
		rawRows.Close()
	}

	// O vetor comprimido, por chunk.
	comprimidos := map[string][]byte{}
	if tqRows, err := db.QueryContext(ctx, `SELECT chunk_id, data FROM chunks_turboquant`); err == nil {
		for tqRows.Next() {
			var chunkID string
			var data []byte
			if err := tqRows.Scan(&chunkID, &data); err == nil {
				comprimidos[chunkID] = data
			}
		}
		tqRows.Close()
	}

	var desync []store.DesyncedChunk
	for _, c := range chunks {
		raw, temRaw := vetoresCruzos[c.id]
		comprimido, temComprimido := comprimidos[c.id]

		issue := ""
		switch {
		case temRaw && vetorCruzoEhZero(raw):
			issue = "Zero vector embedding"
		case temComprimido && turboQuantEhZero(comprimido, embedder.EmbeddingDim):
			issue = "Zero vector embedding"
		case !temRaw && !temComprimido:
			issue = "Missing vector embedding"
		case temRaw && !temComprimido:
			issue = "Missing TurboQuant 4-bit compression"
		}

		if issue != "" {
			desync = append(desync, store.DesyncedChunk{ChunkID: c.id, DocumentID: c.doc, Issue: issue})
		}
	}
	return desync, nil
}

// DiagnoseHealth audita a integridade topológica do grafo e tabelas relacionais do SQLite
// querier e o que o doctor precisa para listar candidatos: tanto o banco
// quanto a transacao da correcao implementam.
type querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// deadLinkCandidates lista as arestas que o SQL suspeita de link morto: o
// alvo nao bate com documento indexado, titulo, tag conhecida nem simbolo de
// codigo.
//
// A suspeita nao e veredito. O candidato so e link rompido depois de a
// checagem em disco dizer que o alvo nao existe; se existir, o link esta fora
// do indice por decisao (ISSUE-039).
func deadLinkCandidates(ctx context.Context, q querier) ([]store.LinkCandidate, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT e.source_id, e.target_id, e.relation
		FROM graph_edges e
		WHERE e.target_id NOT LIKE '#%'
		  AND e.target_id NOT IN (SELECT id FROM documents)
		  AND e.target_id NOT IN (SELECT title FROM documents)
		  AND e.target_id NOT IN (
		      SELECT id FROM graph_nodes WHERE type = 'code_symbol'
		  )
		  AND e.target_id NOT IN (
		      SELECT tag FROM document_tags
		      UNION
		      SELECT tag_key FROM document_tags
		  )
		ORDER BY e.source_id, e.target_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []store.LinkCandidate
	for rows.Next() {
		var c store.LinkCandidate
		if err := rows.Scan(&c.SourceID, &c.TargetID, &c.Relation); err != nil {
			return nil, err
		}
		candidates = append(candidates, c)
	}
	return candidates, rows.Err()
}

// legacyNodeCounts conta os nós do grafo que não são a identidade canônica
// de um documento (`documents.id`) nem um nó de código (`code_symbol`).
//
// ADR-066: são os nós do OUTRO lado da fenda — os que o indexador antigo
// gravava com o id de título. O primeiro número conta todos; o segundo
// conta só os que estão SEM GRAU, que é exatamente o que a correção remove.
//
// O nó de um alvo não resolvido (o texto do wikilink) tem grau: ele é a
// representação viva de um link rompido, e o `mem doctor` precisa dele
// visível (ADR-063).
func legacyNodeCounts(ctx context.Context, q querier) (total int, orphans int, err error) {
	err = q.QueryRowContext(ctx, `
		SELECT COUNT(*),
		       SUM(CASE WHEN NOT EXISTS (
		           SELECT 1 FROM graph_edges e
		           WHERE e.source_id = n.id OR e.target_id = n.id
		       ) THEN 1 ELSE 0 END)
		FROM graph_nodes n
		WHERE n.type <> 'code_symbol'
		  AND n.id NOT IN (SELECT id FROM documents)
	`).Scan(&total, &orphans)
	if err != nil {
		return 0, 0, err
	}
	return total, orphans, nil
}

func DiagnoseHealth(ctx context.Context, db *sql.DB, vaultRoot string) (*store.DoctorReport, error) {
	report := &store.DoctorReport{
		DeadLinks:         []store.DeadLink{},
		OutOfIndexLinks:   []store.OutOfIndexLink{},
		UnverifiableLinks: []store.DeadLink{},
		OrphanNotes:       []store.OrphanNote{},
		SelfLoops:         []store.SelfLoop{},
		DesyncedChunks:    []store.DesyncedChunk{},
		OrthographyIssues: []store.PortugueseOrthographyIssue{},
	}

	// 1. Contagens gerais
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM documents`).Scan(&report.TotalDocuments)
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM chunks`).Scan(&report.TotalChunks)
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM graph_edges`).Scan(&report.TotalEdges)
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM graph_nodes`).Scan(&report.TotalNodes)
	// ADR-057: documentos indexados antes de mtime existir ficam com 0. Reportar
	// a contagem torna a lacuna visível em vez de silenciosa.
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM documents WHERE COALESCE(mtime, 0) = 0`).Scan(&report.MissingMTimeCount)
	// ADR-066: nós que não são `documents.id` nem `code_symbol` — o outro
	// lado da fenda da identidade do nó.
	if total, orphans, err := legacyNodeCounts(ctx, db); err == nil {
		report.LegacyNodeCount = total
		report.OrphanLegacyNodeCount = orphans
	}

	// 2. Links: o SQL levanta a suspeita, o disco decide.
	//
	// O alvo que resolve para um arquivo do vault está fora do índice por
	// decisão — não é link rompido. O alvo que não resolve é link rompido.
	// Sem raiz legível, todos ficam não verificáveis e o score é provisório.
	if candidates, err := deadLinkCandidates(ctx, db); err == nil {
		report.ApplyLinkClassification(store.ClassifyLinks(candidates, store.NewVaultResolver(vaultRoot)))
	}

	// 3. Orphan Notes (documentos sem conexões de entrada ou saída).
	// ISSUE-006 (2026-09-18): docs by-design (ADRs, specs, skills, configs, .github)
	// são excluídos da contagem de órfãos — eles existem para leitura isolada, não
	// como hubs do grafo. Sem isso, o Health Score fica saturado em ~74/100 só
	// porque a maioria dos docs do repo é by-design.
	orphanRows, err := db.QueryContext(ctx, `
		SELECT id, title
		FROM documents
		WHERE id NOT IN (SELECT source_id FROM graph_edges)
		  AND id NOT IN (SELECT target_id FROM graph_edges)
		  AND title NOT IN (SELECT target_id FROM graph_edges)
		  AND (path NOT LIKE '%docs\adr\%' AND path NOT LIKE '%docs/adr/%')
		  AND (path NOT LIKE '%.specs\%'   AND path NOT LIKE '%.specs/%')
		  AND (path NOT LIKE '%.agents\skills\%' AND path NOT LIKE '%.agents/skills/%')
		  AND (path NOT LIKE '%.memory\%'  AND path NOT LIKE '%.memory/%')
		  AND (path NOT LIKE '%.github\%'  AND path NOT LIKE '%.github/%')
		ORDER BY title
	`)
	if err == nil {
		defer orphanRows.Close()
		for orphanRows.Next() {
			var on store.OrphanNote
			if err := orphanRows.Scan(&on.ID, &on.Title); err == nil {
				report.OrphanNotes = append(report.OrphanNotes, on)
			}
		}
	}

	// 4. Self-Loops (arestas onde source_id = target_id)
	loopRows, err := db.QueryContext(ctx, `
		SELECT source_id, relation
		FROM graph_edges
		WHERE source_id = target_id
		ORDER BY source_id
	`)
	if err == nil {
		defer loopRows.Close()
		for loopRows.Next() {
			var sl store.SelfLoop
			if err := loopRows.Scan(&sl.NodeID, &sl.Relation); err == nil {
				report.SelfLoops = append(report.SelfLoops, sl)
			}
		}
	}

	// 5. Chunks dessincronizados (sem vetor REAL: ausente OU vetor zero)
	if desync, err := chunksSemVetorReal(ctx, db); err == nil {
		report.DesyncedChunks = append(report.DesyncedChunks, desync...)
	}

	// 6. Cálculo de Health Score (0 a 100). ISSUE-019: a guarda que pulava o
	// cálculo inteiro quando o vault não tinha documentos deixava o score no
	// valor inicial (100) — nota máxima para um vault que não existe. A
	// matemática é compartilhada com o PostgreSQL (store.CalculateHealthScore)
	// e o score só existe quando há denominador.
	report.HealthScore = store.CalculateHealthScore(*report)

	// 7. Aviso de ortografia (ISSUE-017). Fica DEPOIS do Health Score de propósito:
	// "sao" e "são" tokenizam diferente e a busca falha, mas isso não é corrupção
	// de dado — não pode virar penalidade sem decisão de produto.
	acc := &store.OrthographyAccumulator{}
	titleRows, err := db.QueryContext(ctx, `
		SELECT id, title FROM documents WHERE title IS NOT NULL
	`)
	if err == nil {
		for titleRows.Next() {
			var id, title string
			if err := titleRows.Scan(&id, &title); err == nil {
				acc.Add(id, title)
			}
		}
		titleRows.Close()
	}
	abstractRows, err := db.QueryContext(ctx, `
		SELECT id, abstract FROM documents WHERE abstract IS NOT NULL
	`)
	if err == nil {
		for abstractRows.Next() {
			var id, abstract string
			if err := abstractRows.Scan(&id, &abstract); err == nil {
				acc.Add(id, abstract)
			}
		}
		abstractRows.Close()
	}
	chunkRows, err := db.QueryContext(ctx, `
		SELECT document_id, content FROM chunks
	`)
	if err == nil {
		for chunkRows.Next() {
			var documentID, content string
			if err := chunkRows.Scan(&documentID, &content); err == nil {
				acc.Add(documentID, content)
			}
		}
		chunkRows.Close()
	}
	report.OrthographyDocuments, report.OrthographyIssues = acc.Report(store.OrthographyIssueLimit)

	return report, nil
}

// FixHealthIssues repara anomalias conhecidas do grafo (remove self-loops e links mortos)
func FixHealthIssues(ctx context.Context, db *sql.DB, vaultRoot string) (int, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// 1. Remove self-loops
	resLoops, err := tx.ExecContext(ctx, `DELETE FROM graph_edges WHERE source_id = target_id`)
	if err != nil {
		return 0, err
	}
	loopsFixed, _ := resLoops.RowsAffected()

	// 2. Remove apenas os links comprovadamente rompidos.
	//
	// O que o SQL sozinho suspeitava era apagado inteiro, inclusive o link
	// que aponta para material fora do indice por decisao - um link vivo no
	// Obsidian. Agora a correcao apaga exatamente a lista que o relatorio
	// lista como rompida, e preserva o que nao pode ser provado quebrado.
	candidates, err := deadLinkCandidates(ctx, tx)
	if err != nil {
		return 0, err
	}
	classification := store.ClassifyLinks(candidates, store.NewVaultResolver(vaultRoot))
	var deadFixed int64
	for _, dl := range classification.Broken {
		res, err := tx.ExecContext(ctx,
			`DELETE FROM graph_edges WHERE source_id = ? AND target_id = ? AND relation = ?`,
			dl.SourceID, dl.TargetID, dl.Relation)
		if err != nil {
			return 0, err
		}
		n, _ := res.RowsAffected()
		deadFixed += n
	}

	// 3. Remove os nós legados de identidade (ADR-066).
	//
	// São os nós que o indexador antigo gravava com o id de TÍTULO — o
	// outro lado da fenda. A correção remove os que estão SEM GRAU: eles
	// não são mais o alvo de nenhuma aresta (as arestas passam a apontar
	// para o `documents.id` do documento).
	//
	// O nó com GRAU fica: ele é a representação viva de um link rompido —
	// apagar ele apagaria a aresta junto (FK ON DELETE CASCADE) e o
	// `mem doctor` perderia a detecção de link morto (ADR-063).
	res, err := tx.ExecContext(ctx, `
		DELETE FROM graph_nodes
		WHERE type <> 'code_symbol'
		  AND id NOT IN (SELECT id FROM documents)
		  AND NOT EXISTS (
		      SELECT 1 FROM graph_edges e
		      WHERE e.source_id = graph_nodes.id OR e.target_id = graph_nodes.id
		  )
	`)
	if err != nil {
		return 0, err
	}
	legacyFixed, _ := res.RowsAffected()

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return int(loopsFixed + deadFixed + legacyFixed), nil
}

type nodeCandidate struct {
	id       string
	relPath  string
	title    string
	isDoc    bool
	segments int
}

func countPathSegments(p string) int {
	clean := strings.Trim(strings.ReplaceAll(p, "\\", "/"), "/")
	if clean == "" {
		return 0
	}
	return strings.Count(clean, "/") + 1
}

func isBetterNodeCandidate(a, b nodeCandidate) bool {
	// 1. Documento do mesmo repositório vence (ADR-066 / GPR-03)
	if a.isDoc != b.isDoc {
		return a.isDoc
	}
	// 2. Menos segmentos de caminho vence (GPR-04)
	if a.segments != b.segments {
		return a.segments < b.segments
	}
	// 3. Menor ID lexicográfico desempata (GPR-04)
	return a.id < b.id
}

func selectCanonicalCandidate(cands []nodeCandidate) nodeCandidate {
	best := cands[0]
	for _, c := range cands[1:] {
		if isBetterNodeCandidate(c, best) {
			best = c
		}
	}
	return best
}

func toGraphCandidates(cands []nodeCandidate) []graph.Candidate {
	res := make([]graph.Candidate, len(cands))
	for i, c := range cands {
		res[i] = graph.Candidate{
			ID:      c.id,
			RelPath: c.relPath,
			Title:   c.title,
		}
	}
	return res
}

// ResolveNodeCanonicalID resolve um identificador informal, slug, título ou caminho relativo para o ID canônico no grafo.
// Retorna também o relatório de ambiguidade quando houver mais de um candidato correspondente (GPR-02).
//
// ADR-066: o id canônico do nó é o `documents.id`. O path, o `title:` e o
// `alias:` do MESMO documento são aliases do mesmo nó — e a consulta pelo
// título devolve o `documents.id`, nunca o id de um nó de estilo título.
//
// A raiz do vault — necessária para o `alias:`, que só existe no disco — vem
// da precedência padrão do `internal/vaultfs`: diretório de trabalho atual.
func ResolveNodeCanonicalID(ctx context.Context, db *sql.DB, query string) (string, *graph.AmbiguityReport, error) {
	return ResolveNodeCanonicalIDWithRoot(ctx, db, query, vaultfs.ResolveRoot("", "").Path)
}

// ResolveNodeCanonicalIDWithRoot resolve o nó canônico com a raiz do vault
// explícita, coletando candidatos e determinando desempate determinístico (GPR-01 a GPR-07).
func ResolveNodeCanonicalIDWithRoot(ctx context.Context, db *sql.DB, query string, vaultRoot string) (string, *graph.AmbiguityReport, error) {
	clean := strings.TrimSpace(query)
	clean = strings.TrimPrefix(clean, "[[")
	clean = strings.TrimSuffix(clean, "]]")
	clean = strings.TrimSpace(clean)
	if clean == "" {
		return "", nil, fmt.Errorf("identificador de busca não pode ser vazio")
	}

	// 0a. `alias:` nomeia um alias de frontmatter: ele vive no disco, não no
	// índice, então a resolução é contra o vault.
	if alias, isAlias := strings.CutPrefix(clean, "alias:"); isAlias {
		alias = strings.TrimSpace(alias)
		if canonicalID, ok := resolveAliasToDocumentID(ctx, db, alias, vaultRoot); ok {
			return canonicalID, nil, nil
		}
		return "", nil, fmt.Errorf("nó '%s' não encontrado no grafo", query)
	}

	// 0b. `title:` nomeia explicitamente o rótulo do documento (ADR-066).
	explicitTitle := false
	if rawTitle, isTitle := strings.CutPrefix(clean, "title:"); isTitle {
		explicitTitle = true
		clean = strings.TrimSpace(rawTitle)
		if clean == "" {
			return "", nil, fmt.Errorf("identificador de busca não pode ser vazio")
		}
	}

	if explicitTitle {
		titleRows, err := db.QueryContext(ctx, `
			SELECT id, path, COALESCE(title, '') FROM documents
			WHERE LOWER(title) = LOWER(?)
			ORDER BY path ASC, id ASC
		`, clean)
		if err == nil {
			defer titleRows.Close()
			var cands []nodeCandidate
			for titleRows.Next() {
				var id, path, title string
				if err := titleRows.Scan(&id, &path, &title); err == nil {
					cands = append(cands, nodeCandidate{
						id:       id,
						relPath:  path,
						title:    title,
						isDoc:    true,
						segments: countPathSegments(path),
					})
				}
			}
			if len(cands) > 0 {
				best := selectCanonicalCandidate(cands)
				var amb *graph.AmbiguityReport
				if len(cands) > 1 {
					amb = graph.NewAmbiguityReport(query, toGraphCandidates(cands))
				}
				return best.id, amb, nil
			}
		}
		return "", nil, fmt.Errorf("nó '%s' não encontrado no grafo", query)
	}

	cleanSlash := strings.ReplaceAll(clean, "\\", "/")
	cleanBack := strings.ReplaceAll(clean, "/", "\\")

	// 1. Checagem exata em documents por id (GPR-05)
	var docID string
	err := db.QueryRowContext(ctx, `SELECT id FROM documents WHERE id = ?`, clean).Scan(&docID)
	if err == nil && docID != "" {
		return docID, nil, nil
	}

	// 2. Checagem por caminho relativo em documents (GPR-05, GPR-06)
	// Se tiver barras (/ ou \), é um caminho relativo explícito (ex: docs/guide ou docs/guide.md).
	// Se não tiver barras, é um caminho se corresponder exatamente a path ou se terminar com .md (ex: README.md).
	hasSlash := strings.ContainsAny(clean, "/\\")
	hasMDExt := strings.EqualFold(filepath.Ext(clean), ".md")

	if hasSlash || hasMDExt {
		p1 := clean
		p2 := cleanSlash
		p3 := cleanBack
		p4 := strings.TrimSuffix(cleanSlash, ".md") + ".md"
		p5 := strings.TrimSuffix(cleanBack, ".md") + ".md"
		p6 := strings.TrimSuffix(cleanSlash, ".md")
		p7 := strings.TrimSuffix(cleanBack, ".md")

		pathRows, err := db.QueryContext(ctx, `
			SELECT DISTINCT id, path, COALESCE(title, '') FROM documents
			WHERE path IN (?, ?, ?, ?, ?, ?, ?)
			ORDER BY path ASC, id ASC
		`, p1, p2, p3, p4, p5, p6, p7)
		if err == nil {
			defer pathRows.Close()
			var cands []nodeCandidate
			for pathRows.Next() {
				var id, path, title string
				if err := pathRows.Scan(&id, &path, &title); err == nil {
					cands = append(cands, nodeCandidate{
						id:       id,
						relPath:  path,
						title:    title,
						isDoc:    true,
						segments: countPathSegments(path),
					})
				}
			}
			if len(cands) > 0 {
				best := selectCanonicalCandidate(cands)
				var amb *graph.AmbiguityReport
				if len(cands) > 1 {
					amb = graph.NewAmbiguityReport(query, toGraphCandidates(cands))
				}
				return best.id, amb, nil
			}
		}
	}

	// 3. Checagem em graph_nodes por name ou id (GPR-03, GPR-04, GPR-05)
	gnRows, err := db.QueryContext(ctx, `
		SELECT DISTINCT gn.id, COALESCE(d.path, gn.id), COALESCE(d.title, gn.name), CASE WHEN d.id IS NOT NULL THEN 1 ELSE 0 END AS is_doc
		FROM graph_nodes gn
		LEFT JOIN documents d ON d.id = gn.id
		WHERE gn.name = ? OR gn.name = ? OR gn.name = ? OR gn.id = ? OR gn.id = ? OR gn.id = ?
		ORDER BY gn.id ASC
	`, clean, cleanSlash, cleanBack, clean, cleanSlash, cleanBack)
	if err == nil {
		defer gnRows.Close()
		var cands []nodeCandidate
		for gnRows.Next() {
			var id, path, title string
			var isDocInt int
			if err := gnRows.Scan(&id, &path, &title, &isDocInt); err == nil {
				cands = append(cands, nodeCandidate{
					id:       id,
					relPath:  path,
					title:    title,
					isDoc:    isDocInt == 1,
					segments: countPathSegments(path),
				})
			}
		}
		if len(cands) > 0 {
			best := selectCanonicalCandidate(cands)
			var amb *graph.AmbiguityReport
			if len(cands) > 1 {
				amb = graph.NewAmbiguityReport(query, toGraphCandidates(cands))
			}
			return best.id, amb, nil
		}
	}

	// 4. Checagem exata por título em documents (GPR-05)
	titleRows, err := db.QueryContext(ctx, `
		SELECT id, path, COALESCE(title, '') FROM documents
		WHERE LOWER(title) = LOWER(?)
		ORDER BY path ASC, id ASC
	`, clean)
	if err == nil {
		defer titleRows.Close()
		var cands []nodeCandidate
		for titleRows.Next() {
			var id, path, title string
			if err := titleRows.Scan(&id, &path, &title); err == nil {
				cands = append(cands, nodeCandidate{
					id:       id,
					relPath:  path,
					title:    title,
					isDoc:    true,
					segments: countPathSegments(path),
				})
			}
		}
		if len(cands) > 0 {
			best := selectCanonicalCandidate(cands)
			var amb *graph.AmbiguityReport
			if len(cands) > 1 {
				amb = graph.NewAmbiguityReport(query, toGraphCandidates(cands))
			}
			return best.id, amb, nil
		}
	}

	// 5. Checagem difusa (LIKE) em id, path ou title (GPR-05)
	likeSlash := "%" + cleanSlash + "%"
	likeBack := "%" + cleanBack + "%"
	likeClean := "%" + clean + "%"
	fuzzyRows, err := db.QueryContext(ctx, `
		SELECT id, path, COALESCE(title, '') FROM documents
		WHERE id LIKE ? OR path LIKE ? OR path LIKE ? OR title LIKE ?
		ORDER BY
			CASE
				WHEN id LIKE ? THEN 1
				WHEN path LIKE ? OR path LIKE ? THEN 2
				ELSE 3
			END,
			id ASC
	`, likeSlash, likeSlash, likeBack, likeClean, clean+"%", cleanSlash+"%", cleanBack+"%")
	if err == nil {
		defer fuzzyRows.Close()
		var cands []nodeCandidate
		for fuzzyRows.Next() {
			var id, path, title string
			if err := fuzzyRows.Scan(&id, &path, &title); err == nil {
				cands = append(cands, nodeCandidate{
					id:       id,
					relPath:  path,
					title:    title,
					isDoc:    true,
					segments: countPathSegments(path),
				})
			}
		}
		if len(cands) > 0 {
			best := selectCanonicalCandidate(cands)
			var amb *graph.AmbiguityReport
			if len(cands) > 1 {
				amb = graph.NewAmbiguityReport(query, toGraphCandidates(cands))
			}
			return best.id, amb, nil
		}
	}

	// 6. Se existe nas arestas do grafo como source ou target direto
	edgeRows, err := db.QueryContext(ctx, `
		SELECT target_id FROM graph_edges WHERE target_id = ? OR target_id = ? OR target_id = ?
		UNION
		SELECT source_id FROM graph_edges WHERE source_id = ? OR source_id = ? OR source_id = ?
		ORDER BY 1 ASC
	`, clean, cleanSlash, cleanBack, clean, cleanSlash, cleanBack)
	if err == nil {
		defer edgeRows.Close()
		var edgeIDs []string
		for edgeRows.Next() {
			var edgeID string
			if err := edgeRows.Scan(&edgeID); err == nil {
				edgeIDs = append(edgeIDs, edgeID)
			}
		}
		if len(edgeIDs) > 0 {
			return edgeIDs[0], nil, nil
		}
	}

	return "", nil, fmt.Errorf("nó '%s' não encontrado no grafo", query)
}

// resolveAliasToDocumentID resolve um alias de frontmatter contra o disco do
// vault e devolve o `documents.id` do documento encontrado.
//
// O alias não é persistido no índice (a ADR-056 não foi implementada), então
// a resolução precisa do disco — e só acontece quando a raiz do vault é
// conhecida.
func resolveAliasToDocumentID(ctx context.Context, db *sql.DB, alias, vaultRoot string) (string, bool) {
	if strings.TrimSpace(alias) == "" || strings.TrimSpace(vaultRoot) == "" {
		return "", false
	}
	resolver, err := vaultfs.NewResolver(vaultRoot)
	if err != nil {
		return "", false
	}
	resolution := resolver.Resolve(alias)
	if !resolution.Found {
		return "", false
	}
	// O caminho do resolver é relativo à raiz; o id do documento é o path.
	abs := filepath.Join(vaultRoot, filepath.FromSlash(resolution.Path))
	slash := filepath.ToSlash(abs)
	back := strings.ReplaceAll(slash, "/", "\\")
	var canonicalID string
	err = db.QueryRowContext(ctx, `
		SELECT id FROM documents
		WHERE id = ? OR path = ? OR path = ? OR path = ?
		LIMIT 1
	`, abs, abs, slash, back).Scan(&canonicalID)
	if err != nil || canonicalID == "" {
		return "", false
	}
	return canonicalID, true
}

// CalculateImpactForTarget resolve o nó alvo e calcula a análise de impacto (blast radius) completa no SQLite
func CalculateImpactForTarget(ctx context.Context, db *sql.DB, targetQuery string, maxDepth int) (*graph.ImpactResult, error) {
	canonicalID, _, err := ResolveNodeCanonicalID(ctx, db, targetQuery)
	if err != nil {
		return nil, err
	}

	// 1. Carregar todos os nós e seus tipos
	nodeTypes := make(map[string]string)
	nodesSet := make(map[string]bool)

	docRows, err := db.QueryContext(ctx, "SELECT id FROM documents")
	if err == nil {
		defer docRows.Close()
		for docRows.Next() {
			var id string
			if err := docRows.Scan(&id); err == nil {
				nodesSet[id] = true
				nodeTypes[id] = "note"
			}
		}
	}

	nodeRows, err := db.QueryContext(ctx, "SELECT id, type FROM graph_nodes")
	if err == nil {
		defer nodeRows.Close()
		for nodeRows.Next() {
			var id, nType string
			if err := nodeRows.Scan(&id, &nType); err == nil {
				nodesSet[id] = true
				nodeTypes[id] = nType
			}
		}
	}

	// 2. Carregar arestas
	edgeRows, err := db.QueryContext(ctx, "SELECT source_id, target_id, COALESCE(relation, 'links_to'), COALESCE(weight, 1.0) FROM graph_edges")
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar arestas para análise de impacto: %w", err)
	}
	defer edgeRows.Close()

	var edges []graph.WeightedEdge
	for edgeRows.Next() {
		var src, tgt, rel string
		var weight float64
		if err := edgeRows.Scan(&src, &tgt, &rel, &weight); err != nil {
			return nil, err
		}
		nodesSet[src] = true
		nodesSet[tgt] = true
		edges = append(edges, graph.WeightedEdge{
			Source: src,
			Target: tgt,
			Type:   rel,
			Weight: weight,
		})
	}

	allNodes := make([]string, 0, len(nodesSet))
	for n := range nodesSet {
		allNodes = append(allNodes, n)
	}

	// 3. Calcular PageRanks e Comunidades para enriquecer a análise
	prScores := graph.ComputePageRank(allNodes, edges, graph.DefaultPageRankOptions())

	commRes := graph.DetectCommunities(allNodes, edges, graph.DefaultCommunityOptions())
	commMap := make(map[string]int)
	for _, c := range commRes.Communities {
		for _, m := range c.Members {
			commMap[m] = c.ID
		}
	}

	opts := graph.ImpactOptions{
		MaxDepth:    maxDepth,
		NodeTypes:   nodeTypes,
		Communities: commMap,
		PageRanks:   prScores,
	}

	return graph.CalculateImpact(allNodes, edges, canonicalID, opts)
}

// InspectNode constrói a visualização cirúrgica em 3 colunas (Triptych) de um nó no banco SQLite
func InspectNode(ctx context.Context, db *sql.DB, targetQuery string, maxContentLen int) (*graph.TriptychView, error) {
	canonicalID, _, err := ResolveNodeCanonicalID(ctx, db, targetQuery)
	if err != nil {
		return nil, err
	}

	// 1. Carregar metadados do documento alvo
	var targetTitle, targetPath string
	var targetUpdatedAt int64
	_ = db.QueryRowContext(ctx, "SELECT COALESCE(title, ''), COALESCE(path, ''), COALESCE(updated_at, 0) FROM documents WHERE id = ?", canonicalID).Scan(&targetTitle, &targetPath, &targetUpdatedAt)

	var targetType string
	_ = db.QueryRowContext(ctx, "SELECT COALESCE(type, 'note') FROM graph_nodes WHERE id = ?", canonicalID).Scan(&targetType)
	if targetType == "" {
		targetType = "note"
	}
	if targetTitle == "" {
		targetTitle = canonicalID
	}

	// 2. Carregar preview do conteúdo do nó alvo
	var contentBuilder strings.Builder
	chunkRows, err := db.QueryContext(ctx, "SELECT content FROM chunks WHERE document_id = ? ORDER BY chunk_index ASC", canonicalID)
	if err == nil {
		defer chunkRows.Close()
		for chunkRows.Next() {
			var chunkContent string
			if err := chunkRows.Scan(&chunkContent); err == nil {
				if contentBuilder.Len() > 0 {
					contentBuilder.WriteString("\n\n")
				}
				contentBuilder.WriteString(chunkContent)
				if maxContentLen > 0 && contentBuilder.Len() > maxContentLen*3 {
					break
				}
			}
		}
	}

	// 3. Tags associadas ao nó
	var tags []string
	tagRows, err := db.QueryContext(ctx, "SELECT target_id FROM graph_edges WHERE source_id = ? AND (relation = 'tagged_as' OR target_id LIKE '#%')", canonicalID)
	if err == nil {
		defer tagRows.Close()
		for tagRows.Next() {
			var t string
			if err := tagRows.Scan(&t); err == nil {
				tags = append(tags, t)
			}
		}
	}

	// 4. Carregar todos os nós, tipos, títulos e status de existência
	nodeTypes := make(map[string]string)
	titles := make(map[string]string)
	existingNodes := make(map[string]bool)
	nodesSet := make(map[string]bool)

	docRows, err := db.QueryContext(ctx, "SELECT id, COALESCE(title, id) FROM documents")
	if err == nil {
		defer docRows.Close()
		for docRows.Next() {
			var id, t string
			if err := docRows.Scan(&id, &t); err == nil {
				nodesSet[id] = true
				existingNodes[id] = true
				if t != "" {
					existingNodes[t] = true
				}
				titles[id] = t
				nodeTypes[id] = "note"
			}
		}
	}

	gnRows, err := db.QueryContext(ctx, "SELECT id, COALESCE(name, id), COALESCE(type, 'other') FROM graph_nodes")
	if err == nil {
		defer gnRows.Close()
		for gnRows.Next() {
			var id, n, t string
			if err := gnRows.Scan(&id, &n, &t); err == nil {
				nodesSet[id] = true
				if titles[id] == "" || titles[id] == id {
					titles[id] = n
				}
				if nodeTypes[id] == "" {
					nodeTypes[id] = t
				}
				if strings.HasPrefix(id, "#") || (t != "note" && t != "other") {
					existingNodes[id] = true
				}
			}
		}
	}

	// 5. Carregar todas as arestas
	edgeRows, err := db.QueryContext(ctx, "SELECT source_id, target_id, COALESCE(relation, 'links_to'), COALESCE(weight, 1.0) FROM graph_edges")
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar arestas para inspeção: %w", err)
	}
	defer edgeRows.Close()

	var edges []graph.WeightedEdge
	for edgeRows.Next() {
		var src, tgt, rel string
		var weight float64
		if err := edgeRows.Scan(&src, &tgt, &rel, &weight); err != nil {
			return nil, err
		}
		nodesSet[src] = true
		nodesSet[tgt] = true
		edges = append(edges, graph.WeightedEdge{
			Source: src,
			Target: tgt,
			Type:   rel,
			Weight: weight,
		})
	}

	allNodes := make([]string, 0, len(nodesSet))
	for n := range nodesSet {
		allNodes = append(allNodes, n)
	}

	// 6. Calcular PageRank e Comunidades
	prScores := graph.ComputePageRank(allNodes, edges, graph.DefaultPageRankOptions())
	commRes := graph.DetectCommunities(allNodes, edges, graph.DefaultCommunityOptions())
	commMap := make(map[string]int)
	commLabels := make(map[int]string)
	for _, c := range commRes.Communities {
		commLabels[c.ID] = c.DominantType
		for _, m := range c.Members {
			commMap[m] = c.ID
		}
	}

	// 7. Calcular Impacto / Risco para o alvo
	impactRes, _ := graph.CalculateImpact(allNodes, edges, canonicalID, graph.ImpactOptions{
		MaxDepth:    2,
		NodeTypes:   nodeTypes,
		Communities: commMap,
		PageRanks:   prScores,
	})

	var updatedTime time.Time
	if targetUpdatedAt > 0 {
		updatedTime = time.Unix(targetUpdatedAt, 0)
	}

	targetSummary := graph.NodeSummary{
		ID:             canonicalID,
		Title:          targetTitle,
		Path:           targetPath,
		Type:           targetType,
		Tags:           tags,
		PageRank:       prScores[canonicalID],
		CommunityID:    commMap[canonicalID],
		CommunityLabel: commLabels[commMap[canonicalID]],
		ContentPreview: contentBuilder.String(),
		UpdatedAt:      updatedTime,
	}

	opts := graph.InspectorOptions{
		MaxContentLength: maxContentLen,
		PageRanks:        prScores,
		Communities:      commMap,
		CommunityLabels:  commLabels,
		NodeTypes:        nodeTypes,
		Titles:           titles,
		ExistingNodes:    existingNodes,
		RiskResult:       impactRes,
	}

	return graph.BuildTriptychView(targetSummary, edges, opts)
}

// FindPath resolve os nós de origem e destino e calcula o menor caminho entre eles no SQLite
func FindPath(ctx context.Context, db *sql.DB, sourceQuery, targetQuery string, opts graph.PathOptions) (*graph.PathResult, error) {
	canonicalSource, ambSource, err := ResolveNodeCanonicalID(ctx, db, sourceQuery)
	if err != nil {
		return nil, fmt.Errorf("falha ao resolver nó de origem: %w", err)
	}

	canonicalTarget, ambTarget, err := ResolveNodeCanonicalID(ctx, db, targetQuery)
	if err != nil {
		return nil, fmt.Errorf("falha ao resolver nó de destino: %w", err)
	}

	// 1. Carregar todos os nós
	nodesSet := make(map[string]bool)
	docRows, err := db.QueryContext(ctx, "SELECT id FROM documents")
	if err == nil {
		defer docRows.Close()
		for docRows.Next() {
			var id string
			if err := docRows.Scan(&id); err == nil {
				nodesSet[id] = true
			}
		}
	}

	nodeRows, err := db.QueryContext(ctx, "SELECT id FROM graph_nodes")
	if err == nil {
		defer nodeRows.Close()
		for nodeRows.Next() {
			var id string
			if err := nodeRows.Scan(&id); err == nil {
				nodesSet[id] = true
			}
		}
	}

	// 2. Carregar arestas
	edgeRows, err := db.QueryContext(ctx, "SELECT source_id, target_id, COALESCE(relation, 'links_to'), COALESCE(weight, 1.0), COALESCE(epistemic_status, 'EXTRACTED') FROM graph_edges")
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar arestas para busca de caminho: %w", err)
	}
	defer edgeRows.Close()

	var edges []graph.WeightedEdge
	for edgeRows.Next() {
		var src, tgt, rel, epStatus string
		var weight float64
		if err := edgeRows.Scan(&src, &tgt, &rel, &weight, &epStatus); err != nil {
			return nil, err
		}
		nodesSet[src] = true
		nodesSet[tgt] = true
		edges = append(edges, graph.WeightedEdge{
			Source:          src,
			Target:          tgt,
			Type:            rel,
			Weight:          weight,
			EpistemicStatus: epStatus,
		})
	}

	allNodes := make([]string, 0, len(nodesSet))
	for n := range nodesSet {
		allNodes = append(allNodes, n)
	}

	pathRes, err := graph.FindPath(allNodes, edges, canonicalSource, canonicalTarget, opts)
	if err != nil {
		return nil, err
	}
	if pathRes != nil {
		if ambSource != nil && ambTarget != nil {
			allCands := append(ambSource.Candidates, ambTarget.Candidates...)
			pathRes.Ambiguity = graph.NewAmbiguityReport(ambSource.Query+", "+ambTarget.Query, allCands)
		} else if ambSource != nil {
			pathRes.Ambiguity = ambSource
		} else if ambTarget != nil {
			pathRes.Ambiguity = ambTarget
		}
	}
	return pathRes, nil
}

// PackContext resolve o nó raiz e constrói o pacote consolidado de subgrafo no SQLite
func PackContext(ctx context.Context, db *sql.DB, rootQuery string, opts graph.PackOptions) (*graph.PackResult, error) {
	canonicalRoot, _, err := ResolveNodeCanonicalID(ctx, db, rootQuery)
	if err != nil {
		return nil, fmt.Errorf("falha ao resolver nó raiz: %w", err)
	}

	// 1. Carregar arestas
	edgeRows, err := db.QueryContext(ctx, "SELECT source_id, target_id, COALESCE(relation, 'links_to'), COALESCE(weight, 1.0), COALESCE(epistemic_status, 'EXTRACTED') FROM graph_edges")
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar arestas para empacotamento: %w", err)
	}
	defer edgeRows.Close()

	nodesSet := make(map[string]bool)
	var edges []graph.WeightedEdge
	for edgeRows.Next() {
		var src, tgt, rel, epStatus string
		var weight float64
		if err := edgeRows.Scan(&src, &tgt, &rel, &weight, &epStatus); err != nil {
			return nil, err
		}
		nodesSet[src] = true
		nodesSet[tgt] = true
		edges = append(edges, graph.WeightedEdge{
			Source:          src,
			Target:          tgt,
			Type:            rel,
			Weight:          weight,
			EpistemicStatus: epStatus,
		})
	}

	// 2. Carregar metadados dos documentos
	docMap := make(map[string]graph.PackGraphNode)
	docRows, err := db.QueryContext(ctx, "SELECT id, path, COALESCE(title, id), updated_at, COALESCE(abstract, ''), COALESCE(category, 'resource') FROM documents")
	if err == nil {
		defer docRows.Close()
		for docRows.Next() {
			var id, path, title, abstract, category string
			var updatedAt int64
			if err := docRows.Scan(&id, &path, &title, &updatedAt, &abstract, &category); err == nil {
				nodesSet[id] = true
				docMap[id] = graph.PackGraphNode{
					ID:        id,
					Title:     title,
					Path:      path,
					Abstract:  abstract,
					Category:  category,
					UpdatedAt: updatedAt,
				}
			}
		}
	}

	// 3. Nós adicionais em graph_nodes
	gnRows, err := db.QueryContext(ctx, "SELECT id, COALESCE(name, id) FROM graph_nodes")
	if err == nil {
		defer gnRows.Close()
		for gnRows.Next() {
			var id, name string
			if err := gnRows.Scan(&id, &name); err == nil {
				nodesSet[id] = true
				if _, exists := docMap[id]; !exists {
					docMap[id] = graph.PackGraphNode{
						ID:    id,
						Title: name,
					}
				}
			}
		}
	}

	// 4. Carregar chunks e agregar conteúdo por documento
	chunkRows, err := db.QueryContext(ctx, "SELECT document_id, content FROM chunks ORDER BY document_id, chunk_index ASC")
	if err == nil {
		defer chunkRows.Close()
		contentBuilders := make(map[string]*strings.Builder)
		for chunkRows.Next() {
			var docID, chunkContent string
			if err := chunkRows.Scan(&docID, &chunkContent); err == nil {
				b, ok := contentBuilders[docID]
				if !ok {
					b = &strings.Builder{}
					contentBuilders[docID] = b
				} else {
					b.WriteString("\n\n")
				}
				b.WriteString(chunkContent)
			}
		}
		for docID, b := range contentBuilders {
			if node, ok := docMap[docID]; ok {
				node.Content = b.String()
				docMap[docID] = node
			}
		}
	}

	// 5. Calcular PageRank para priorização
	allNodes := make([]string, 0, len(nodesSet))
	for n := range nodesSet {
		allNodes = append(allNodes, n)
	}
	prScores := graph.ComputePageRank(allNodes, edges, graph.DefaultPageRankOptions())
	opts.PageRanks = prScores

	// 6. Preparar lista de PackGraphNodes
	packNodes := make([]graph.PackGraphNode, 0, len(docMap))
	for _, node := range docMap {
		packNodes = append(packNodes, node)
	}

	return graph.PackContext(packNodes, edges, canonicalRoot, opts)
}

// ListVaultDeadends devolve as notas que apontam para alguma coisa e não são
// apontadas por ninguém — as folhas do grafo.
//
// A definição importa: uma nota sem nenhuma aresta (nem de entrada nem de
// saída) NÃO é deadend, é órfã, e `mem doctor` já reporta isso. Deadend é só
// a ponta de saída, o que a spec e o Obsidian chamam de dead end. Por isso os
// dois NOT EXISTS são assimétricos — exigir o segundo transformaria toda nota
// isolada em deadend e inflaria o relatório.
func ListVaultDeadends(ctx context.Context, database *sql.DB) ([]store.VaultDocument, error) {
	rows, err := database.QueryContext(ctx, `
		SELECT d.id, d.path, COALESCE(d.title, ''), COALESCE(d.mtime, 0), COALESCE(d.size, 0)
		FROM documents d
		WHERE EXISTS (SELECT 1 FROM graph_edges e WHERE e.source_id = d.id)
		  AND NOT EXISTS (SELECT 1 FROM graph_edges e WHERE e.target_id = d.id)
		ORDER BY d.path ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar deadends: %w", err)
	}
	defer rows.Close()

	docs := make([]store.VaultDocument, 0, 8)
	for rows.Next() {
		var doc store.VaultDocument
		if err := rows.Scan(&doc.ID, &doc.Path, &doc.Title, &doc.MTime, &doc.Size); err != nil {
			return nil, fmt.Errorf("erro ao escanear deadend: %w", err)
		}
		docs = append(docs, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar deadends: %w", err)
	}
	return docs, nil
}
