package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// codeStore é o par (conexão, dialeto) usado pelos helpers de code_*.
//
// O dialeto é detectado uma vez por conexão (DetectDialect memoiza) e as
// queries são reescritas na hora do placeholder: o SQLite aceita "?" e o
// lib/pq exige "$1" (ADR-061 — o code graph vive no engine configurado).
type codeStore struct {
	db      *sql.DB
	dialect Dialect
}

// newCodeStore empacota a conexão com o dialeto detectado.
func newCodeStore(ctx context.Context, database *sql.DB) (codeStore, error) {
	if database == nil {
		return codeStore{}, errors.New("code store: db == nil")
	}
	return codeStore{db: database, dialect: DetectDialect(ctx, database)}, nil
}

func (c codeStore) exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return c.db.ExecContext(ctx, c.dialect.Bind(query), args...)
}

func (c codeStore) queryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return c.db.QueryRowContext(ctx, c.dialect.Bind(query), args...)
}

func (c codeStore) query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return c.db.QueryContext(ctx, c.dialect.Bind(query), args...)
}

// EnsureCodeTables é uma migração idempotente que garante a existência das
// tabelas code_files / code_symbols / code_edges / code_edges_uncertain e seus índices.
//
// Já é executada em InitDB via Schema/FallbackSchema (CREATE TABLE IF NOT EXISTS).
// Esta função é mantida como ponto único para upgrades aditivos futuros
// (ex: adicionar colunas com ALTER TABLE em bases antigas), no mesmo padrão
// dos blocos `pragma_table_info` que aparecem em store.go (ADR-016, CA-16).
//
// A verificação de coluna existe nos dois engines: `pragma_table_info` no
// SQLite e `information_schema.columns` no PostgreSQL.
func EnsureCodeTables(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("EnsureCodeTables: db == nil")
	}
	store, err := newCodeStore(ctx, db)
	if err != nil {
		return err
	}

	// Bloco de migrações aditivas. Cada bloco é independente: se a coluna já
	// existir, a verificação devolverá count > 0 e pulamos o ALTER.
	additiveMigrations := []struct {
		table  string
		column string
		ddl    string
	}{
		{"code_files", "ast_hash", "ALTER TABLE code_files ADD COLUMN ast_hash TEXT"},
		{"code_edges", "confidence", "ALTER TABLE code_edges ADD COLUMN confidence REAL NOT NULL DEFAULT 1.0"},
	}

	for _, m := range additiveMigrations {
		cnt, err := store.columnExists(ctx, m.table, m.column)
		if err != nil {
			return fmt.Errorf("EnsureCodeTables: check %s.%s falhou: %w", m.table, m.column, err)
		}
		if cnt == 0 {
			if _, err := store.exec(ctx, m.ddl); err != nil {
				return fmt.Errorf("EnsureCodeTables: ALTER %s.%s falhou: %w", m.table, m.column, err)
			}
		}
	}
	return nil
}

// columnExists devolve quantas colunas com esse nome existem na tabela.
func (c codeStore) columnExists(ctx context.Context, table, column string) (int, error) {
	if c.dialect.IsPostgres() {
		var cnt int
		err := c.queryRow(ctx,
			`SELECT COUNT(*) FROM information_schema.columns WHERE table_name = ? AND column_name = ?`,
			table, column,
		).Scan(&cnt)
		return cnt, err
	}
	var cnt int
	err := c.queryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM pragma_table_info('%s') WHERE name = '%s'", table, column),
	).Scan(&cnt)
	return cnt, err
}

// UpsertCodeFile insere (ou atualiza) uma linha em code_files usando
// (path) como chave natural (UNIQUE). Devolve o id da linha.
func UpsertCodeFile(ctx context.Context, db *sql.DB, path, language, contentHash, astHash string, sizeBytes, indexedAt int64) (int64, error) {
	store, err := newCodeStore(ctx, db)
	if err != nil {
		return 0, err
	}
	res, err := store.exec(ctx, `
		INSERT INTO code_files (path, language, size_bytes, content_hash, ast_hash, indexed_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			language = excluded.language,
			size_bytes = excluded.size_bytes,
			content_hash = excluded.content_hash,
			ast_hash = excluded.ast_hash,
			indexed_at = excluded.indexed_at
	`, path, language, sizeBytes, contentHash, nullString(astHash), indexedAt)
	if err != nil {
		return 0, fmt.Errorf("UpsertCodeFile falhou: %w", err)
	}
	id, _ := res.LastInsertId()
	if id == 0 {
		// ON CONFLICT (UPDATE) em SQLite devolve LastInsertId = 0; resolve.
		// O lib/pq não implementa LastInsertId — o mesmo fallback vale lá.
		if err := store.queryRow(ctx, `SELECT id FROM code_files WHERE path = ?`, path).Scan(&id); err != nil {
			return 0, fmt.Errorf("UpsertCodeFile: locate id pós-conflict falhou: %w", err)
		}
	}
	return id, nil
}

// UpsertCodeSymbol insere (ou atualiza) uma linha em code_symbols.
// A unicidade é (file_id, kind, qualified_name, start_line).
func UpsertCodeSymbol(ctx context.Context, db *sql.DB, fileID int64, kind, name, qualifiedName, signature, docComment string, startLine, endLine, startCol, endCol int) (int64, error) {
	store, err := newCodeStore(ctx, db)
	if err != nil {
		return 0, err
	}
	res, err := store.exec(ctx, `
		INSERT INTO code_symbols (file_id, kind, name, qualified_name, signature, doc_comment, start_line, end_line, start_col, end_col)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(file_id, kind, qualified_name, start_line) DO UPDATE SET
			name = excluded.name,
			signature = excluded.signature,
			doc_comment = excluded.doc_comment,
			end_line = excluded.end_line,
			start_col = excluded.start_col,
			end_col = excluded.end_col
	`, fileID, kind, name, nullString(qualifiedName), nullString(signature), nullString(docComment), startLine, endLine, startCol, endCol)
	if err != nil {
		return 0, fmt.Errorf("UpsertCodeSymbol falhou: %w", err)
	}
	id, _ := res.LastInsertId()
	if id == 0 {
		// COALESCE: no SQLite e no PostgreSQL um NULL em qualified_name
		// nunca casa com o `IS` de um unique (NULLs são distintos), então a
		// busca pelo valor coalescido é a que acha a linha de fato.
		key := qualifiedName
		if err := store.queryRow(ctx,
			`SELECT id FROM code_symbols WHERE file_id = ? AND kind = ? AND COALESCE(qualified_name, '') = ? AND start_line = ? ORDER BY id ASC`,
			fileID, kind, key, startLine,
		).Scan(&id); err != nil {
			return 0, fmt.Errorf("UpsertCodeSymbol: locate id pós-conflict falhou: %w", err)
		}
	}
	return id, nil
}

// UpsertCodeEdge insere (ou atualiza) uma aresta em code_edges.
// Se confidence < 0.5 redireciona automaticamente para code_edges_uncertain.
func UpsertCodeEdge(ctx context.Context, db *sql.DB, srcSymbolID, dstSymbolID, fileID int64, kind string, startLine, endLine int, confidence float64) error {
	if confidence < 0.5 {
		return errors.New("UseInsertCodeEdgeUncertain para confidence < 0.5")
	}
	store, err := newCodeStore(ctx, db)
	if err != nil {
		return err
	}
	_, err = store.exec(ctx, `
		INSERT INTO code_edges (src_symbol_id, dst_symbol_id, kind, file_id, start_line, end_line, confidence)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(src_symbol_id, dst_symbol_id, kind) DO UPDATE SET
			file_id = excluded.file_id,
			start_line = excluded.start_line,
			end_line = excluded.end_line,
			confidence = excluded.confidence
	`, srcSymbolID, dstSymbolID, kind, fileID, startLine, endLine, confidence)
	return err
}

// InsertCodeEdgeUncertain insere uma aresta de baixa confiança em code_edges_uncertain.
func InsertCodeEdgeUncertain(ctx context.Context, db *sql.DB, srcSymbolID, fileID int64, dstCandidateName, kind, reason string, confidence float64, startLine int) error {
	store, err := newCodeStore(ctx, db)
	if err != nil {
		return err
	}
	_, err = store.exec(ctx, `
		INSERT INTO code_edges_uncertain (src_symbol_id, dst_candidate_name, kind, file_id, confidence, reason, start_line)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(src_symbol_id, dst_candidate_name, kind) DO UPDATE SET
			file_id = excluded.file_id,
			confidence = excluded.confidence,
			reason = excluded.reason,
			start_line = excluded.start_line
	`, srcSymbolID, dstCandidateName, kind, fileID, confidence, reason, startLine)
	return err
}

// GetCodeFileHash devolve (content_hash, ast_hash) do code_files.path.
// Retorna ("", "", false) quando o path não foi indexado.
func GetCodeFileHash(ctx context.Context, db *sql.DB, path string) (contentHash, astHash string, ok bool, err error) {
	store, err := newCodeStore(ctx, db)
	if err != nil {
		return "", "", false, err
	}
	var ah sql.NullString
	err = store.queryRow(ctx,
		`SELECT content_hash, ast_hash FROM code_files WHERE path = ?`, path,
	).Scan(&contentHash, &ah)
	if err == sql.ErrNoRows {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, err
	}
	if ah.Valid {
		astHash = ah.String
	}
	return contentHash, astHash, true, nil
}

// DeleteCodeFile remove um code_files (e cascateia symbols/edges) por path.
func DeleteCodeFile(ctx context.Context, db *sql.DB, path string) error {
	store, err := newCodeStore(ctx, db)
	if err != nil {
		return err
	}
	_, err = store.exec(ctx, `DELETE FROM code_files WHERE path = ?`, path)
	return err
}

// CountCodeFiles devolve o total de code_files indexados.
func CountCodeFiles(ctx context.Context, db *sql.DB) (int64, error) {
	store, err := newCodeStore(ctx, db)
	if err != nil {
		return 0, err
	}
	var n int64
	err = store.queryRow(ctx, `SELECT COUNT(*) FROM code_files`).Scan(&n)
	return n, err
}

// CountCodeSymbols devolve o total de code_symbols indexados.
func CountCodeSymbols(ctx context.Context, db *sql.DB) (int64, error) {
	store, err := newCodeStore(ctx, db)
	if err != nil {
		return 0, err
	}
	var n int64
	err = store.queryRow(ctx, `SELECT COUNT(*) FROM code_symbols`).Scan(&n)
	return n, err
}

// CountCodeEdges devolve o total de code_edges indexados (somente arestas
// de alta confiança; code_edges_uncertain é contado separadamente).
func CountCodeEdges(ctx context.Context, db *sql.DB) (int64, error) {
	store, err := newCodeStore(ctx, db)
	if err != nil {
		return 0, err
	}
	var n int64
	err = store.queryRow(ctx, `SELECT COUNT(*) FROM code_edges`).Scan(&n)
	return n, err
}

// OrphanFile representa um code_file sem symbols extraídos (T6/CA-09).
// Exportado para permitir serialização JSON no CLI.
type OrphanFile struct {
	Path string `json:"path"`
}

// LanguageDistribution devolve um mapa `lang → count` ordenado por count desc.
// Útil para mem code-stats (T6).
type LanguageCount struct {
	Language string `json:"language"`
	Count    int64  `json:"count"`
}

func ListLanguageDistribution(ctx context.Context, db *sql.DB, limit int) ([]LanguageCount, error) {
	store, err := newCodeStore(ctx, db)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 100
	}
	rows, err := store.query(ctx, `
		SELECT language, COUNT(*) AS c
		FROM code_files
		GROUP BY language
		ORDER BY c DESC, language ASC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LanguageCount
	for rows.Next() {
		var lc LanguageCount
		if err := rows.Scan(&lc.Language, &lc.Count); err != nil {
			return nil, err
		}
		out = append(out, lc)
	}
	return out, rows.Err()
}

// SymbolKindCount representa a contagem de símbolos por kind para code-stats (T6).
type SymbolKindCount struct {
	Kind  string `json:"kind"`
	Count int64  `json:"count"`
}

func ListSymbolKindDistribution(ctx context.Context, db *sql.DB) ([]SymbolKindCount, error) {
	store, err := newCodeStore(ctx, db)
	if err != nil {
		return nil, err
	}
	rows, err := store.query(ctx, `
		SELECT kind, COUNT(*) AS c
		FROM code_symbols
		GROUP BY kind
		ORDER BY c DESC, kind ASC
		`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SymbolKindCount
	for rows.Next() {
		var sc SymbolKindCount
		if err := rows.Scan(&sc.Kind, &sc.Count); err != nil {
			return nil, err
		}
		out = append(out, sc)
	}
	return out, rows.Err()
}

// ListOrphanFiles devolve code_files.path que não têm symbol algum (orphan — T6).
func ListOrphanFiles(ctx context.Context, db *sql.DB, limit int) ([]string, error) {
	store, err := newCodeStore(ctx, db)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	}
	rows, err := store.query(ctx, `
		SELECT f.path
		FROM code_files f
		LEFT JOIN code_symbols s ON s.file_id = f.id
		WHERE s.id IS NULL
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// nullString devolve sql.NullString para valores opcionais (zero = NULL).
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
