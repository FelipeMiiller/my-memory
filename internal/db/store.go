package db

import (
	"context"
	"database/sql"
	"fmt"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	_ "github.com/mattn/go-sqlite3"
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

// InsertEdge cria uma conexão no grafo
func InsertEdge(ctx context.Context, db *sql.DB, sourceID, targetID, relation string) error {
	_, err := db.ExecContext(ctx, `
		INSERT OR IGNORE INTO graph_edges (source_id, target_id, relation)
		VALUES (?, ?, ?)
	`, sourceID, targetID, relation)
	return err
}
