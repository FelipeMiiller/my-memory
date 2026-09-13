package db

import (
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/FelipeMiiller/my-memory/internal/store"
	"github.com/FelipeMiiller/my-memory/internal/turboquant"
)

// InitDB inicializa a conexão com o SQLite registrando a extensão sqlite-vec globalmente
func InitDB(dbPath string) (*sql.DB, error) {
	initSqliteVec()

	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir banco: %w", err)
	}

	if _, err := db.Exec(Schema); err != nil {
		return nil, fmt.Errorf("erro ao executar schema: %w", err)
	}

	// Migração retrocompatível: adiciona coluna content_hash se não existir
	var colCount int
	_ = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('documents') WHERE name = 'content_hash'").Scan(&colCount)
	if colCount == 0 {
		_, _ = db.Exec("ALTER TABLE documents ADD COLUMN content_hash TEXT")
	}

	// Migração retrocompatível: adiciona colunas epistemic_status e weight em graph_edges se não existirem
	var edgeColCount int
	_ = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('graph_edges') WHERE name = 'epistemic_status'").Scan(&edgeColCount)
	if edgeColCount == 0 {
		_, _ = db.Exec("ALTER TABLE graph_edges ADD COLUMN epistemic_status TEXT NOT NULL DEFAULT 'EXTRACTED'")
		_, _ = db.Exec("ALTER TABLE graph_edges ADD COLUMN weight REAL NOT NULL DEFAULT 1.0")
	}

	return db, nil
}

// InsertDocument salva documento com content_hash e seus nós no grafo
func InsertDocument(ctx context.Context, db *sql.DB, id, path, title string, updatedAt int64, contentHash string) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO documents (id, path, title, updated_at, content_hash)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			updated_at = excluded.updated_at,
			content_hash = excluded.content_hash
	`, id, path, title, updatedAt, contentHash)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		INSERT OR IGNORE INTO graph_nodes (id, type, name)
		VALUES (?, 'note', ?)
	`, id, title)
	return err
}

// GetDocumentHash retorna o hash SHA-256 de conteúdo armazenado de um documento (ou "" se não existir)
func GetDocumentHash(ctx context.Context, db *sql.DB, id string) (string, error) {
	var hash sql.NullString
	err := db.QueryRowContext(ctx, `
		SELECT content_hash FROM documents
		WHERE id = ?
	`, id).Scan(&hash)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return hash.String, nil
}

// DeleteDocumentData remove chunks (relacional, FTS, vec, turboquant) e arestas originadas do documento
func DeleteDocumentData(ctx context.Context, db *sql.DB, id string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Chunks FTS5, vec0 e turboquant via subquery dos chunks do documento
	_, _ = tx.ExecContext(ctx, `
		DELETE FROM chunks_fts WHERE chunk_id IN (SELECT id FROM chunks WHERE document_id = ?)
	`, id)

	_, _ = tx.ExecContext(ctx, `
		DELETE FROM chunks_vec WHERE chunk_id IN (SELECT id FROM chunks WHERE document_id = ?)
	`, id)

	_, _ = tx.ExecContext(ctx, `
		DELETE FROM chunks_turboquant WHERE chunk_id IN (SELECT id FROM chunks WHERE document_id = ?)
	`, id)

	// 2. Chunks relacionais
	_, err = tx.ExecContext(ctx, `
		DELETE FROM chunks WHERE document_id = ?
	`, id)
	if err != nil {
		return err
	}

	// 3. Arestas do grafo onde este documento é origem (source_id)
	_, err = tx.ExecContext(ctx, `
		DELETE FROM graph_edges WHERE source_id = ?
	`, id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// DeleteDocumentComplete remove completamente o documento, seus chunks (FTS5, vec0, turboquant)
// e arestas associadas (origem e destino) do banco SQLite
func DeleteDocumentComplete(ctx context.Context, db *sql.DB, id string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Chunks FTS5, vec0 e turboquant via subquery dos chunks do documento
	_, _ = tx.ExecContext(ctx, `
		DELETE FROM chunks_fts WHERE chunk_id IN (SELECT id FROM chunks WHERE document_id = ?)
	`, id)

	_, _ = tx.ExecContext(ctx, `
		DELETE FROM chunks_vec WHERE chunk_id IN (SELECT id FROM chunks WHERE document_id = ?)
	`, id)

	_, _ = tx.ExecContext(ctx, `
		DELETE FROM chunks_turboquant WHERE chunk_id IN (SELECT id FROM chunks WHERE document_id = ?)
	`, id)

	// 2. Chunks relacionais
	_, err = tx.ExecContext(ctx, `DELETE FROM chunks WHERE document_id = ?`, id)
	if err != nil {
		return err
	}

	// 3. Arestas do grafo onde este documento é origem ou destino
	_, err = tx.ExecContext(ctx, `DELETE FROM graph_edges WHERE source_id = ? OR target_id = ?`, id, id)
	if err != nil {
		return err
	}

	// 4. Remove o registro do documento
	_, err = tx.ExecContext(ctx, `DELETE FROM documents WHERE id = ?`, id)
	if err != nil {
		return err
	}

	// 5. Remove nó do grafo se existir
	_, _ = tx.ExecContext(ctx, `DELETE FROM graph_nodes WHERE id = ?`, id)

	return tx.Commit()
}

// PruneDeletedDocuments identifica e remove documentos sob rootDir que não constam em activeDocIDs
func PruneDeletedDocuments(ctx context.Context, db *sql.DB, rootDir string, activeDocIDs []string) ([]string, error) {
	activeMap := make(map[string]bool, len(activeDocIDs))
	for _, id := range activeDocIDs {
		activeMap[id] = true
		activeMap[filepath.Clean(id)] = true
		if abs, err := filepath.Abs(id); err == nil {
			activeMap[abs] = true
			activeMap[filepath.Clean(abs)] = true
		}
	}

	rows, err := db.QueryContext(ctx, `SELECT id, path FROM documents`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var toPrune []string
	cleanRoot := ""
	if rootDir != "" {
		if abs, err := filepath.Abs(rootDir); err == nil {
			cleanRoot = strings.ToLower(filepath.Clean(abs))
		} else {
			cleanRoot = strings.ToLower(filepath.Clean(rootDir))
		}
	}

	for rows.Next() {
		var id, path string
		if err := rows.Scan(&id, &path); err != nil {
			continue
		}

		cleanPath := path
		if abs, err := filepath.Abs(path); err == nil {
			cleanPath = abs
		}
		cleanPathLower := strings.ToLower(filepath.Clean(cleanPath))

		if cleanRoot != "" {
			if !strings.HasPrefix(cleanPathLower, cleanRoot) {
				continue
			}
		}

		cleanID := id
		if abs, err := filepath.Abs(id); err == nil {
			cleanID = abs
		}

		if !activeMap[id] && !activeMap[path] && !activeMap[cleanPath] && !activeMap[cleanID] {
			toPrune = append(toPrune, id)
		}
	}

	var pruned []string
	for _, id := range toPrune {
		if err := DeleteDocumentComplete(ctx, db, id); err == nil {
			pruned = append(pruned, id)
		}
	}

	return pruned, nil
}

// InsertChunk insere um pedaço de texto no relacional, FTS5 e no sqlite-vec
func InsertChunk(ctx context.Context, db *sql.DB, chunkID, docID, content string, index int, vec []float32) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Tabela relacional
	_, err = tx.ExecContext(ctx, `
		INSERT INTO chunks (id, document_id, chunk_index, content)
		VALUES (?, ?, ?, ?)
	`, chunkID, docID, index, content)
	if err != nil {
		return err
	}

	// 2. FTS5 (busca textual léxica)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO chunks_fts (chunk_id, content)
		VALUES (?, ?)
	`, chunkID, content)
	if err != nil {
		return err
	}

	// 3. sqlite-vec (busca vetorial padrão float32)
	vecBlob, err := serializeFloat32(vec)
	if err != nil {
		return fmt.Errorf("erro serializando vetor: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO chunks_vec (chunk_id, embedding)
		VALUES (?, ?)
	`, chunkID, vecBlob)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// InsertTurboQuantChunk armazena o vetor quantizado em 4-bits com fator de escala
func InsertTurboQuantChunk(ctx context.Context, db *sql.DB, chunkID string, scale float32, data []byte) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO chunks_turboquant (chunk_id, scale, data)
		VALUES (?, ?, ?)
		ON CONFLICT(chunk_id) DO UPDATE SET
			scale = excluded.scale,
			data = excluded.data
	`, chunkID, scale, data)
	return err
}

// InsertEdgeWithProps cria uma conexão tipada com status epistêmico e peso no grafo
func InsertEdgeWithProps(ctx context.Context, db *sql.DB, sourceID, targetID, relation, epistemicStatus string, weight float64) error {
	if epistemicStatus == "" {
		epistemicStatus = "EXTRACTED"
	}
	if weight <= 0 {
		weight = 1.0
	}

	// Garante que nós de origem e destino existam para satisfazer integridade referencial
	_, _ = db.ExecContext(ctx, `INSERT OR IGNORE INTO graph_nodes (id, type, name) VALUES (?, 'note', ?)`, sourceID, sourceID)
	_, _ = db.ExecContext(ctx, `INSERT OR IGNORE INTO graph_nodes (id, type, name) VALUES (?, 'note', ?)`, targetID, targetID)

	_, err := db.ExecContext(ctx, `
		INSERT INTO graph_edges (source_id, target_id, relation, epistemic_status, weight)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(source_id, target_id, relation) DO UPDATE SET
			epistemic_status = excluded.epistemic_status,
			weight = excluded.weight
	`, sourceID, targetID, relation, epistemicStatus, weight)
	return err
}

// InsertEdge cria uma conexão padrão no grafo
func InsertEdge(ctx context.Context, db *sql.DB, sourceID, targetID, relation string) error {
	return InsertEdgeWithProps(ctx, db, sourceID, targetID, relation, "EXTRACTED", 1.0)
}

// GetGodNodes calcula e retorna os nós com maior centralidade de grau no grafo SQLite
func GetGodNodes(ctx context.Context, db *sql.DB, limit int) ([]store.GodNode, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `
		WITH degrees AS (
			SELECT source_id AS node_id, 0 AS in_cnt, 1 AS out_cnt FROM graph_edges
			UNION ALL
			SELECT target_id AS node_id, 1 AS in_cnt, 0 AS out_cnt FROM graph_edges
		)
		SELECT d.node_id, COALESCE(n.name, d.node_id) AS name,
		       SUM(d.in_cnt) AS in_degree,
		       SUM(d.out_cnt) AS out_degree,
		       COUNT(*) AS total_degree
		FROM degrees d
		LEFT JOIN graph_nodes n ON n.id = d.node_id
		GROUP BY d.node_id
		ORDER BY total_degree DESC, in_degree DESC
		LIMIT ?;
	`

	rows, err := db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar god nodes sqlite: %w", err)
	}
	defer rows.Close()

	var hubs []store.GodNode
	for rows.Next() {
		var h store.GodNode
		if err := rows.Scan(&h.ID, &h.Name, &h.InDegree, &h.OutDegree, &h.TotalDegree); err != nil {
			return nil, err
		}
		hubs = append(hubs, h)
	}
	return hubs, nil
}

// FindSurprisingConnections descobre conexões latentes entre notas com alta similaridade sem arestas no grafo SQLite
func FindSurprisingConnections(ctx context.Context, db *sql.DB, limit int, minSimilarity float64) ([]store.SurprisingConnection, error) {
	if limit <= 0 {
		limit = 10
	}
	if minSimilarity <= 0 {
		minSimilarity = 0.70
	}

	// 1. Carrega todas as arestas existentes para anti-join rápido em memória
	edgeRows, err := db.QueryContext(ctx, `SELECT source_id, target_id FROM graph_edges`)
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar arestas sqlite: %w", err)
	}
	defer edgeRows.Close()

	linked := make(map[string]bool)
	for edgeRows.Next() {
		var src, tgt string
		if err := edgeRows.Scan(&src, &tgt); err == nil {
			linked[src+"->"+tgt] = true
			linked[tgt+"->"+src] = true
		}
	}

	// 2. Carrega todos os documentos
	docRows, err := db.QueryContext(ctx, `SELECT id, COALESCE(title, id) FROM documents ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar documentos sqlite: %w", err)
	}
	defer docRows.Close()

	type docInfo struct {
		id    string
		title string
	}
	var docs []docInfo
	for docRows.Next() {
		var di docInfo
		if err := docRows.Scan(&di.id, &di.title); err == nil {
			docs = append(docs, di)
		}
	}

	if len(docs) < 2 {
		return []store.SurprisingConnection{}, nil
	}

	// 3. Tenta carregar vetores float32 de chunks_vec
	docVectors := make(map[string][][]float32)
	vecRows, err := db.QueryContext(ctx, `
		SELECT c.document_id, v.embedding
		FROM chunks_vec v
		JOIN chunks c ON c.id = v.chunk_id
	`)
	if err == nil {
		defer vecRows.Close()
		for vecRows.Next() {
			var docID string
			var blob []byte
			if err := vecRows.Scan(&docID, &blob); err == nil && len(blob) > 0 {
				vec := deserializeFloat32(blob)
				if len(vec) > 0 {
					docVectors[docID] = append(docVectors[docID], vec)
				}
			}
		}
	}

	// Se chunks_vec estiver vazio, tenta carregar de chunks_turboquant descompactando
	if len(docVectors) == 0 {
		tqRows, err := db.QueryContext(ctx, `
			SELECT c.document_id, tq.scale, tq.data
			FROM chunks_turboquant tq
			JOIN chunks c ON c.id = tq.chunk_id
		`)
		if err == nil {
			defer tqRows.Close()
			tq := turboquant.NewQuantizer(768)
			for tqRows.Next() {
				var docID string
				var scale float32
				var data []byte
				if err := tqRows.Scan(&docID, &scale, &data); err == nil && len(data) > 0 {
					cv := &turboquant.CompressedVector{
						Dim:   768,
						Scale: scale,
						Data:  data,
					}
					vec := tq.Dequantize(cv)
					if len(vec) > 0 {
						docVectors[docID] = append(docVectors[docID], vec)
					}
				}
			}
		}
	}

	var results []store.SurprisingConnection

	if len(docVectors) > 0 {
		for i := 0; i < len(docs); i++ {
			for j := i + 1; j < len(docs); j++ {
				d1 := docs[i]
				d2 := docs[j]

				if linked[d1.id+"->"+d2.id] {
					continue
				}

				vecs1 := docVectors[d1.id]
				vecs2 := docVectors[d2.id]
				if len(vecs1) == 0 || len(vecs2) == 0 {
					continue
				}

				var maxSim float64
				for _, v1 := range vecs1 {
					for _, v2 := range vecs2 {
						sim := store.CosineSimilarity(v1, v2)
						if sim > maxSim {
							maxSim = sim
						}
					}
				}

				if maxSim >= minSimilarity {
					results = append(results, store.SurprisingConnection{
						SourceID:   d1.id,
						SourceName: d1.title,
						TargetID:   d2.id,
						TargetName: d2.title,
						Similarity: maxSim,
						Reason:     fmt.Sprintf("Alta proximidade semântica (%.0f%%) sem conexão direta no grafo", maxSim*100),
					})
				}
			}
		}
	} else {
		// Fallback léxico: Jaccard sobre o conteúdo dos chunks
		contentRows, err := db.QueryContext(ctx, `
			SELECT document_id, content FROM chunks ORDER BY document_id, chunk_index
		`)
		if err == nil {
			defer contentRows.Close()
			docContent := make(map[string]string)
			for contentRows.Next() {
				var docID, content string
				if err := contentRows.Scan(&docID, &content); err == nil {
					if existing, ok := docContent[docID]; ok {
						docContent[docID] = existing + " " + content
					} else {
						docContent[docID] = content
					}
				}
			}

			for i := 0; i < len(docs); i++ {
				for j := i + 1; j < len(docs); j++ {
					d1 := docs[i]
					d2 := docs[j]

					if linked[d1.id+"->"+d2.id] {
						continue
					}

					c1 := docContent[d1.id]
					c2 := docContent[d2.id]
					if c1 == "" || c2 == "" {
						continue
					}

					sim := store.CalculateJaccardSimilarity(c1, c2)
					if sim >= minSimilarity {
						results = append(results, store.SurprisingConnection{
							SourceID:   d1.id,
							SourceName: d1.title,
							TargetID:   d2.id,
							TargetName: d2.title,
							Similarity: sim,
							Reason:     fmt.Sprintf("Alta sobreposição léxica (%.0f%%) sem conexão direta no grafo", sim*100),
						})
					}
				}
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	if len(results) > limit {
		results = results[:limit]
	}
	if results == nil {
		results = []store.SurprisingConnection{}
	}
	return results, nil
}

func deserializeFloat32(b []byte) []float32 {
	n := len(b) / 4
	if n == 0 {
		return nil
	}
	res := make([]float32, n)
	for i := 0; i < n; i++ {
		bits := binary.LittleEndian.Uint32(b[i*4 : (i+1)*4])
		res[i] = math.Float32frombits(bits)
	}
	return res
}
