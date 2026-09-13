package db

import (
	"context"
	"database/sql"
	"fmt"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	_ "github.com/mattn/go-sqlite3"

	"github.com/FelipeMiiller/my-memory/internal/store"
)

// InitDB inicializa a conexão com o SQLite registrando a extensão sqlite-vec globalmente
func InitDB(dbPath string) (*sql.DB, error) {
	sqlite_vec.Auto()

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
	vecBlob, err := sqlite_vec.SerializeFloat32(vec)
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
