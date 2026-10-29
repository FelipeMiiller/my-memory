package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
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

// EpistemicStatus classifica a PROCEDÊNCIA de uma aresta do code graph.
//
// Os mesmos valores de graph_edges (ADR-011):
//
//	EpistemicExtracted — a aresta veio da ESTRUTURA do código: o destino foi
//	                    resolvido pelo qualified name dentro do pacote.
//	EpistemicInferred  — a aresta foi resolvida apenas pelo NOME (único no
//	                    repositório): é uma inferência, não uma extração.
const (
	EpistemicExtracted = "EXTRACTED"
	EpistemicInferred  = "INFERRED"
)

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
	//
	// O campo `backfill` roda UMA ÚNICA VEZ, logo após o ALTER — é o lugar
	// para corrigir o valor das linhas que já existiam (a provenance delas
	// não pode ser inventada pelo default da coluna nova).
	additiveMigrations := []struct {
		table    string
		column   string
		ddl      string
		backfill string
	}{
		{"code_files", "ast_hash", "ALTER TABLE code_files ADD COLUMN ast_hash TEXT", ""},
		{"code_edges", "confidence", "ALTER TABLE code_edges ADD COLUMN confidence REAL NOT NULL DEFAULT 1.0", ""},
		// CG-06: o code graph entra na federação de repo_id.
		{"code_files", "repository", "ALTER TABLE code_files ADD COLUMN repository TEXT NOT NULL DEFAULT '" + DefaultRepository + "'", ""},
		{"code_symbols", "repository", "ALTER TABLE code_symbols ADD COLUMN repository TEXT NOT NULL DEFAULT '" + DefaultRepository + "'", ""},
		{"code_edges", "repository", "ALTER TABLE code_edges ADD COLUMN repository TEXT NOT NULL DEFAULT '" + DefaultRepository + "'", ""},
		// A tabela de arestas INCERTAIN também é repo-scoped (CG-06): o DDL
		// pré-feature não tinha a coluna e o InsertCodeEdgeUncertain grava
		// nela — sem esta migração, o code stage quebra em banco existente
		// com "table code_edges_uncertain has no column named repository".
		{"code_edges_uncertain", "repository", "ALTER TABLE code_edges_uncertain ADD COLUMN repository TEXT NOT NULL DEFAULT '" + DefaultRepository + "'", ""},
		// CG-14: provenance por aresta. O default é EXTRACTED (aresta extraída
		// da estrutura do código). As linhas PRÉ-EXISTENTES de `references`
		// foram resolvidas por NOME (heurística legada), então são INFERRED —
		// o backfill roda uma única vez, junto com o ALTER.
		{
			table:    "code_edges",
			column:   "epistemic_status",
			ddl:      "ALTER TABLE code_edges ADD COLUMN epistemic_status TEXT NOT NULL DEFAULT '" + EpistemicExtracted + "'",
			backfill: "UPDATE code_edges SET epistemic_status = '" + EpistemicInferred + "' WHERE kind = 'references'",
		},
	}

	for _, m := range additiveMigrations {
		// A tabela pode não existir em bases muito antigas (o schema a cria
		// com a forma nova no próximo InitDB). Sem esta guarda o ALTER
		// falharia com "no such table".
		exists, err := store.tableExists(ctx, m.table)
		if err != nil {
			return fmt.Errorf("EnsureCodeTables: check table %s falhou: %w", m.table, err)
		}
		if !exists {
			continue
		}
		cnt, err := store.columnExists(ctx, m.table, m.column)
		if err != nil {
			return fmt.Errorf("EnsureCodeTables: check %s.%s falhou: %w", m.table, m.column, err)
		}
		if cnt == 0 {
			if _, err := store.exec(ctx, m.ddl); err != nil {
				return fmt.Errorf("EnsureCodeTables: ALTER %s.%s falhou: %w", m.table, m.column, err)
			}
			if m.backfill != "" {
				if _, err := store.exec(ctx, m.backfill); err != nil {
					return fmt.Errorf("EnsureCodeTables: backfill %s.%s falhou: %w", m.table, m.column, err)
				}
			}
		}
	}

	// O code graph é multi-tenant: a identidade do arquivo é (repository, path).
	// No PostgreSQL a chave única antiga (path) é substituída por um índice
	// único composto; no SQLite o UNIQUE(path) original permanece (o banco
	// local é de um vault só).
	if store.dialect.IsPostgres() {
		if _, err := store.exec(ctx, `ALTER TABLE code_files DROP CONSTRAINT IF EXISTS code_files_path_key`); err != nil {
			return fmt.Errorf("EnsureCodeTables: drop code_files_path_key falhou: %w", err)
		}
		if _, err := store.exec(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_code_files_repository_path ON code_files(repository, path)`); err != nil {
			return fmt.Errorf("EnsureCodeTables: idx_code_files_repository_path falhou: %w", err)
		}
	}
	return nil
}

// tableExists devolve true quando a tabela existe no engine conectado.
func (c codeStore) tableExists(ctx context.Context, table string) (bool, error) {
	var cnt int
	if c.dialect.IsPostgres() {
		err := c.queryRow(ctx,
			`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?`,
			table,
		).Scan(&cnt)
		return cnt > 0, err
	}
	err := c.queryRow(ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`,
		table,
	).Scan(&cnt)
	return cnt > 0, err
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
// (repository, path) como chave natural (UNIQUE). Devolve o id da linha.
//
// O repository gravado é o repo_id resolvido do vault, carregado pelo
// contexto (ADR-061 — o code graph entra na federação).
func UpsertCodeFile(ctx context.Context, db *sql.DB, path, language, contentHash, astHash string, sizeBytes, indexedAt int64) (int64, error) {
	store, err := newCodeStore(ctx, db)
	if err != nil {
		return 0, err
	}
	repository := RepositoryFromContext(ctx)
	// --ast-hash=false: o ast_hash não é armazenado (o cache incremental
	// perde a política "conteúdo mudou, AST não mudou").
	if !AstHashEnabled(ctx) {
		astHash = ""
	}
	// No SQLite o UNIQUE original é do path; no PostgreSQL a identidade é
	// (repository, path) — o conflict target precisa casar com o índice.
	conflict := "(path)"
	if store.dialect.IsPostgres() {
		conflict = "(repository, path)"
	}
	res, err := store.exec(ctx, fmt.Sprintf(`
		INSERT INTO code_files (repository, path, language, size_bytes, content_hash, ast_hash, indexed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT %s DO UPDATE SET
			repository = excluded.repository,
			language = excluded.language,
			size_bytes = excluded.size_bytes,
			content_hash = excluded.content_hash,
			ast_hash = excluded.ast_hash,
			indexed_at = excluded.indexed_at
	`, conflict), repository, path, language, sizeBytes, contentHash, nullString(astHash), indexedAt)
	if err != nil {
		return 0, fmt.Errorf("UpsertCodeFile falhou: %w", err)
	}
	id, _ := res.LastInsertId()
	if id == 0 {
		// ON CONFLICT (UPDATE) em SQLite devolve LastInsertId = 0; resolve.
		// O lib/pq não implementa LastInsertId — o mesmo fallback vale lá.
		if err := store.queryRow(ctx,
			`SELECT id FROM code_files WHERE repository = ? AND path = ?`, repository, path,
		).Scan(&id); err != nil {
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
		INSERT INTO code_symbols (repository, file_id, kind, name, qualified_name, signature, doc_comment, start_line, end_line, start_col, end_col)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(file_id, kind, qualified_name, start_line) DO UPDATE SET
			repository = excluded.repository,
			name = excluded.name,
			signature = excluded.signature,
			doc_comment = excluded.doc_comment,
			end_line = excluded.end_line,
			start_col = excluded.start_col,
			end_col = excluded.end_col
	`, RepositoryFromContext(ctx), fileID, kind, name, nullString(qualifiedName), nullString(signature), nullString(docComment), startLine, endLine, startCol, endCol)
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

// CodeSymbolRow é uma linha de code_symbols no upsert EM LOTE.
//
// O file_id fica de fora da linha: o lote é por arquivo (é o que faz do
// `INSERT` multi-linha valer a pena).
type CodeSymbolRow struct {
	Kind          string
	Name          string
	QualifiedName string
	Signature     string
	DocComment    string
	StartLine     int
	EndLine       int
	StartCol      int
	EndCol        int
}

// codeMaxBatchRows é o teto de LINHAS por statement do code graph.
//
// O limite que aperta é o de PARÂMETROS do engine: o SQLite aceita
// MAX_VARIABLE_NUMBER=32766 (medido no driver deste repo) e o Postgres aceita
// 65535. O insert de símbolo usa 11 colunas → 2000 linhas = 22 000
// parâmetros, folga larga nos dois. Acima do teto o lote é partido: o
// conteúdo gravado não muda.
const codeMaxBatchRows = 2000

// codeSymbolKey é a chave de identidade de um símbolo em code_symbols: a
// mesma do alvo de ON CONFLICT (file_id, kind, qualified_name, start_line),
// sem o file_id (que é o mesmo do arquivo inteiro).
//
// O qualified_name vazio é gravado como NULL; o COALESCE do SELECT de
// resolução devolve "" — as duas pontas falam a mesma língua.
func codeSymbolKey(kind, qualifiedName string, startLine int) string {
	return kind + "\x00" + qualifiedName + "\x00" + strconv.Itoa(startLine)
}

// dedupCodeSymbolRows remove do lote as chaves repetidas.
//
// O Postgres REJEITA um `ON CONFLICT DO UPDATE` que afete a mesma linha duas
// vezes ("ON CONFLICT DO UPDATE command cannot affect row a second time") —
// o SQLite aceita. Remover a duplicata deixa os dois engines iguais e
// reproduz o caminho por símbolo, onde o segundo upsert ATUALIZA a linha do
// primeiro (o payload do último vence, o id é o mesmo).
func dedupCodeSymbolRows(rows []CodeSymbolRow) []CodeSymbolRow {
	order := make([]string, 0, len(rows))
	byKey := make(map[string]CodeSymbolRow, len(rows))
	for _, row := range rows {
		key := codeSymbolKey(row.Kind, row.QualifiedName, row.StartLine)
		if _, seen := byKey[key]; !seen {
			order = append(order, key)
		}
		byKey[key] = row
	}
	out := make([]CodeSymbolRow, 0, len(order))
	for _, key := range order {
		out = append(out, byKey[key])
	}
	return out
}

// buildCodeSymbolInsert monta o INSERT multi-linha de code_symbols.
//
// O texto do SQL é o MESMO de `UpsertCodeSymbol` — as mesmas 11 colunas e o
// mesmo alvo de ON CONFLICT — muda só o número de tuplas no VALUES.
func buildCodeSymbolInsert(repository string, fileID int64, rows []CodeSymbolRow) (string, []any) {
	tuples := make([]string, 0, len(rows))
	args := make([]any, 0, len(rows)*11)
	for _, row := range rows {
		tuples = append(tuples, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		args = append(args,
			repository, fileID, row.Kind, row.Name,
			nullString(row.QualifiedName), nullString(row.Signature), nullString(row.DocComment),
			row.StartLine, row.EndLine, row.StartCol, row.EndCol,
		)
	}
	return `
		INSERT INTO code_symbols (repository, file_id, kind, name, qualified_name, signature, doc_comment, start_line, end_line, start_col, end_col)
		VALUES ` + strings.Join(tuples, ", ") + `
		ON CONFLICT(file_id, kind, qualified_name, start_line) DO UPDATE SET
			repository = excluded.repository,
			name = excluded.name,
			signature = excluded.signature,
			doc_comment = excluded.doc_comment,
			end_line = excluded.end_line,
			start_col = excluded.start_col,
			end_col = excluded.end_col
	`, args
}

// queryCodeSymbolIDs resolve o id de cada símbolo do arquivo.
//
// Um SELECT por arquivo: o lib/pq não implementa LastInsertId, então o id
// precisa ser lido. O casamento é feito em Go, pela chave
// (kind, qualified_name, start_line) — o MESMO do alvo de ON CONFLICT.
//
// Quando a chave tem mais de uma linha (o qualified_name vazio é NULL, e NULL
// não conflita no índice único), vence a linha de MAIOR id: é a que o lote
// acabou de gravar.
func (c codeStore) queryCodeSymbolIDs(ctx context.Context, fileID int64) (map[string]int64, error) {
	rows, err := c.query(ctx, `
		SELECT id, kind, COALESCE(qualified_name, ''), start_line
		  FROM code_symbols
		 WHERE file_id = ?
	`, fileID)
	if err != nil {
		return nil, fmt.Errorf("UpsertCodeSymbols: resolução de ids falhou: %w", err)
	}
	defer rows.Close()

	ids := make(map[string]int64)
	for rows.Next() {
		var id int64
		var kind, qualifiedName string
		var startLine int
		if err := rows.Scan(&id, &kind, &qualifiedName, &startLine); err != nil {
			return nil, fmt.Errorf("UpsertCodeSymbols: leitura de id falhou: %w", err)
		}
		key := codeSymbolKey(kind, qualifiedName, startLine)
		if current, ok := ids[key]; !ok || id > current {
			ids[key] = id
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("UpsertCodeSymbols: leitura de ids falhou: %w", err)
	}
	return ids, nil
}

// UpsertCodeSymbols grava TODAS as linhas de um arquivo em UM statement
// multi-linha e devolve os ids na MESMA ORDEM da entrada.
//
// É a forma em LOTE de `UpsertCodeSymbol`: as mesmas 11 colunas, o mesmo
// alvo de ON CONFLICT e a mesma idempotência — o que muda é o número de
// round-trips (1 statement + 1 SELECT de resolução por arquivo, em vez de 2
// por símbolo).
//
// Os ids devolvidos são os das linhas gravadas: é deles que as arestas do
// arquivo (src_symbol_id/dst_symbol_id) dependem.
func UpsertCodeSymbols(ctx context.Context, db *sql.DB, fileID int64, rows []CodeSymbolRow) ([]int64, error) {
	if db == nil {
		return nil, errors.New("UpsertCodeSymbols: db == nil")
	}
	if len(rows) == 0 {
		return nil, nil
	}
	store, err := newCodeStore(ctx, db)
	if err != nil {
		return nil, err
	}

	unique := dedupCodeSymbolRows(rows)
	repository := RepositoryFromContext(ctx)
	for start := 0; start < len(unique); start += codeMaxBatchRows {
		end := start + codeMaxBatchRows
		if end > len(unique) {
			end = len(unique)
		}
		query, args := buildCodeSymbolInsert(repository, fileID, unique[start:end])
		if _, err := store.exec(ctx, query, args...); err != nil {
			return nil, fmt.Errorf("UpsertCodeSymbols falhou: %w", err)
		}
	}

	ids, err := store.queryCodeSymbolIDs(ctx, fileID)
	if err != nil {
		return nil, err
	}

	out := make([]int64, 0, len(rows))
	for _, row := range rows {
		key := codeSymbolKey(row.Kind, row.QualifiedName, row.StartLine)
		id, ok := ids[key]
		if !ok {
			return nil, fmt.Errorf("UpsertCodeSymbols: id do símbolo %q não foi resolvido", key)
		}
		out = append(out, id)
	}
	return out, nil
}

// CodeEdgeRow é uma linha de code_edges no upsert EM LOTE.
type CodeEdgeRow struct {
	SrcSymbolID     int64
	DstSymbolID     int64
	FileID          int64
	Kind            string
	StartLine       int
	EndLine         int
	Confidence      float64
	EpistemicStatus string
}

// codeEdgeKey é a chave de identidade de uma aresta: a mesma do alvo de
// ON CONFLICT (src_symbol_id, dst_symbol_id, kind).
func codeEdgeKey(src, dst int64, kind string) string {
	return strconv.FormatInt(src, 10) + "\x00" + strconv.FormatInt(dst, 10) + "\x00" + kind
}

// dedupCodeEdgeRows funde as arestas repetidas do lote.
//
// O Postgres rejeita o conflito da mesma linha duas vezes no mesmo statement.
// A fusão reproduz o que o caminho por aresta faz: a POSIÇÃO que fica é a
// MENOR (o ON CONFLICT guarda a menor start_line/end_line) e o payload
// (file_id, confidence, epistemic_status) é o do ÚLTIMO.
func dedupCodeEdgeRows(rows []CodeEdgeRow) []CodeEdgeRow {
	order := make([]string, 0, len(rows))
	byKey := make(map[string]CodeEdgeRow, len(rows))
	for _, row := range rows {
		key := codeEdgeKey(row.SrcSymbolID, row.DstSymbolID, row.Kind)
		current, seen := byKey[key]
		if !seen {
			order = append(order, key)
			byKey[key] = row
			continue
		}
		if row.StartLine < current.StartLine {
			current.StartLine = row.StartLine
		}
		if row.EndLine < current.EndLine {
			current.EndLine = row.EndLine
		}
		// O payload (file_id, confidence, epistemic_status) é o do ÚLTIMO —
		// é o que o ON CONFLICT (DO UPDATE) grava a cada upsert.
		current.FileID = row.FileID
		current.Confidence = row.Confidence
		current.EpistemicStatus = row.EpistemicStatus
		byKey[key] = current
	}
	out := make([]CodeEdgeRow, 0, len(order))
	for _, key := range order {
		out = append(out, byKey[key])
	}
	return out
}

// buildCodeEdgeInsert monta o INSERT multi-linha de code_edges.
//
// O texto do SQL é o MESMO de `UpsertCodeEdgeWithStatus` — as mesmas 9
// colunas, o mesmo alvo de ON CONFLICT e o mesmo CASE WHEN que mantém a
// menor posição.
func buildCodeEdgeInsert(repository string, rows []CodeEdgeRow) (string, []any) {
	tuples := make([]string, 0, len(rows))
	args := make([]any, 0, len(rows)*9)
	for _, row := range rows {
		tuples = append(tuples, "(?, ?, ?, ?, ?, ?, ?, ?, ?)")
		args = append(args,
			repository, row.SrcSymbolID, row.DstSymbolID, row.Kind, row.FileID,
			row.StartLine, row.EndLine, row.Confidence, row.EpistemicStatus,
		)
	}
	return `
		INSERT INTO code_edges (repository, src_symbol_id, dst_symbol_id, kind, file_id, start_line, end_line, confidence, epistemic_status)
		VALUES ` + strings.Join(tuples, ", ") + `
		ON CONFLICT(src_symbol_id, dst_symbol_id, kind) DO UPDATE SET
			repository = excluded.repository,
			file_id = excluded.file_id,
			start_line = CASE WHEN excluded.start_line < code_edges.start_line THEN excluded.start_line ELSE code_edges.start_line END,
			end_line = CASE WHEN excluded.end_line < code_edges.end_line THEN excluded.end_line ELSE code_edges.end_line END,
			confidence = excluded.confidence,
			epistemic_status = excluded.epistemic_status
	`, args
}

// UpsertCodeEdges grava as arestas de um arquivo (ou do lote) em UM statement
// multi-linha.
//
// É a forma em LOTE de `UpsertCodeEdgeWithStatus`: as mesmas 9 colunas, o
// mesmo alvo de ON CONFLICT e a mesma idempotência — muda o número de
// round-trips (1 statement por arquivo, em vez de 1 por aresta).
//
// Aresta com confidence < 0.5 é recusada com o mesmo erro do caminho por
// aresta: quem grava essa aresta é `InsertCodeEdgeUncertain`.
func UpsertCodeEdges(ctx context.Context, db *sql.DB, rows []CodeEdgeRow) error {
	if db == nil {
		return errors.New("UpsertCodeEdges: db == nil")
	}
	if len(rows) == 0 {
		return nil
	}
	for _, row := range rows {
		if row.Confidence < 0.5 {
			return errors.New("UseInsertCodeEdgeUncertain para confidence < 0.5")
		}
	}
	store, err := newCodeStore(ctx, db)
	if err != nil {
		return err
	}

	unique := dedupCodeEdgeRows(rows)
	repository := RepositoryFromContext(ctx)
	for start := 0; start < len(unique); start += codeMaxBatchRows {
		end := start + codeMaxBatchRows
		if end > len(unique) {
			end = len(unique)
		}
		query, args := buildCodeEdgeInsert(repository, unique[start:end])
		if _, err := store.exec(ctx, query, args...); err != nil {
			return fmt.Errorf("UpsertCodeEdges falhou: %w", err)
		}
	}
	return nil
}

// UpsertCodeEdge insere (ou atualiza) uma aresta em code_edges com
// epistemic_status = EXTRACTED (a aresta veio da estrutura do código).
//
// Se confidence < 0.5 a inserção é recusada: use InsertCodeEdgeUncertain.
func UpsertCodeEdge(ctx context.Context, db *sql.DB, srcSymbolID, dstSymbolID, fileID int64, kind string, startLine, endLine int, confidence float64) error {
	return UpsertCodeEdgeWithStatus(ctx, db, srcSymbolID, dstSymbolID, fileID, kind, startLine, endLine, confidence, EpistemicExtracted)
}

// UpsertCodeEdgeWithStatus insere (ou atualiza) uma aresta em code_edges com o
// provenance declarado (EXTRACTED/INFERRED).
//
// SEMÂNTICA DA POSIÇÃO (Fase 2, decision 4): file_id/start_line/end_line são o
// PONTO DE CHAMADA (call-site) da relação — a linha/coluna da expressão que
// criou a aresta — e NÃO a posição da definição dos símbolos. Para as arestas
// `calls` isso muda o significado da coluna em relação à v1 (que guardava a
// linha de definição do símbolo origem).
//
// Quando a mesma aresta (src, dst, kind) é gravada de novo — o mesmo símbolo
// chama o mesmo destino em pontos diferentes — a linha mantida é a MENOR
// (o primeiro call-site), para que a posição seja estável entre indexações.
func UpsertCodeEdgeWithStatus(ctx context.Context, db *sql.DB, srcSymbolID, dstSymbolID, fileID int64, kind string, startLine, endLine int, confidence float64, epistemicStatus string) error {
	if confidence < 0.5 {
		return errors.New("UseInsertCodeEdgeUncertain para confidence < 0.5")
	}
	if strings.TrimSpace(epistemicStatus) == "" {
		epistemicStatus = EpistemicExtracted
	}
	store, err := newCodeStore(ctx, db)
	if err != nil {
		return err
	}
	_, err = store.exec(ctx, `
		INSERT INTO code_edges (repository, src_symbol_id, dst_symbol_id, kind, file_id, start_line, end_line, confidence, epistemic_status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(src_symbol_id, dst_symbol_id, kind) DO UPDATE SET
			repository = excluded.repository,
			file_id = excluded.file_id,
			start_line = CASE WHEN excluded.start_line < code_edges.start_line THEN excluded.start_line ELSE code_edges.start_line END,
			end_line = CASE WHEN excluded.end_line < code_edges.end_line THEN excluded.end_line ELSE code_edges.end_line END,
			confidence = excluded.confidence,
			epistemic_status = excluded.epistemic_status
	`, RepositoryFromContext(ctx), srcSymbolID, dstSymbolID, kind, fileID, startLine, endLine, confidence, epistemicStatus)
	return err
}

// InsertCodeEdgeUncertain insere uma aresta de baixa confiança em code_edges_uncertain.
func InsertCodeEdgeUncertain(ctx context.Context, db *sql.DB, srcSymbolID, fileID int64, dstCandidateName, kind, reason string, confidence float64, startLine int) error {
	query, args := buildCodeEdgeUncertainInsert(RepositoryFromContext(ctx), []CodeEdgeUncertainRow{{
		SrcSymbolID:      srcSymbolID,
		FileID:           fileID,
		DstCandidateName: dstCandidateName,
		Kind:             kind,
		Confidence:       confidence,
		Reason:           reason,
		StartLine:        startLine,
	}})
	store, err := newCodeStore(ctx, db)
	if err != nil {
		return err
	}
	_, err = store.exec(ctx, query, args...)
	return err
}

// CodeEdgeUncertainRow é uma linha de code_edges_uncertain no upsert EM LOTE.
type CodeEdgeUncertainRow struct {
	SrcSymbolID      int64
	FileID           int64
	DstCandidateName string
	Kind             string
	Confidence       float64
	Reason           string
	StartLine        int
}

// codeEdgeUncertainKey é a chave de identidade de uma aresta incerta: a
// mesma do alvo de ON CONFLICT (src_symbol_id, dst_candidate_name, kind).
func codeEdgeUncertainKey(srcSymbolID int64, dstCandidateName, kind string) string {
	return strconv.FormatInt(srcSymbolID, 10) + "\x00" + dstCandidateName + "\x00" + kind
}

// dedupCodeEdgeUncertainRows remove do lote as chaves repetidas.
//
// O Postgres REJEITA um `ON CONFLICT DO UPDATE` que afete a mesma linha duas
// vezes ("ON CONFLICT DO UPDATE command cannot affect row a second time") —
// o SQLite aceita. Remover a duplicata deixa os dois engines iguais e
// reproduz o caminho por aresta, onde o segundo upsert ATUALIZA a linha do
// primeiro (o payload do ÚLTIMO vence, porque o ON CONFLICT grava
// `excluded` em todas as colunas).
func dedupCodeEdgeUncertainRows(rows []CodeEdgeUncertainRow) []CodeEdgeUncertainRow {
	order := make([]string, 0, len(rows))
	byKey := make(map[string]CodeEdgeUncertainRow, len(rows))
	for _, row := range rows {
		key := codeEdgeUncertainKey(row.SrcSymbolID, row.DstCandidateName, row.Kind)
		if _, seen := byKey[key]; !seen {
			order = append(order, key)
		}
		byKey[key] = row
	}
	out := make([]CodeEdgeUncertainRow, 0, len(order))
	for _, key := range order {
		out = append(out, byKey[key])
	}
	return out
}

// buildCodeEdgeUncertainInsert monta o INSERT multi-linha de
// code_edges_uncertain.
//
// O texto do SQL é o MESMO de `InsertCodeEdgeUncertain` — as mesmas 8 colunas
// e o mesmo alvo de ON CONFLICT (a chave única
// UNIQUE(src_symbol_id, dst_candidate_name, kind) existe nos DOIS engines:
// `internal/db/schema.go` e `internal/store/postgres.go`).
func buildCodeEdgeUncertainInsert(repository string, rows []CodeEdgeUncertainRow) (string, []any) {
	tuples := make([]string, 0, len(rows))
	args := make([]any, 0, len(rows)*8)
	for _, row := range rows {
		tuples = append(tuples, "(?, ?, ?, ?, ?, ?, ?, ?)")
		args = append(args,
			repository, row.SrcSymbolID, row.DstCandidateName, row.Kind, row.FileID,
			row.Confidence, row.Reason, row.StartLine,
		)
	}
	return `
		INSERT INTO code_edges_uncertain (repository, src_symbol_id, dst_candidate_name, kind, file_id, confidence, reason, start_line)
		VALUES ` + strings.Join(tuples, ", ") + `
		ON CONFLICT(src_symbol_id, dst_candidate_name, kind) DO UPDATE SET
			repository = excluded.repository,
			file_id = excluded.file_id,
			confidence = excluded.confidence,
			reason = excluded.reason,
			start_line = excluded.start_line
	`, args
}

// InsertCodeEdgeUncertains grava as arestas incertas de um arquivo (ou do
// lote) em UM statement multi-linha.
//
// É a forma em LOTE de `InsertCodeEdgeUncertain`: as mesmas 8 colunas, o
// mesmo alvo de ON CONFLICT e a mesma idempotência — muda o número de
// round-trips (1 statement por arquivo, em vez de 1 por aresta).
//
// A medição que motivou o lote: no Postgres remoto (≈268 ms por round-trip)
// o bucket era gravado a ~1,6 aresta/s — ~110 min só para popular
// `code_edges_uncertain` do repo inteiro, contra ~5 min dos símbolos.
func InsertCodeEdgeUncertains(ctx context.Context, db *sql.DB, rows []CodeEdgeUncertainRow) error {
	if db == nil {
		return errors.New("InsertCodeEdgeUncertains: db == nil")
	}
	if len(rows) == 0 {
		return nil
	}
	store, err := newCodeStore(ctx, db)
	if err != nil {
		return err
	}

	unique := dedupCodeEdgeUncertainRows(rows)
	repository := RepositoryFromContext(ctx)
	for start := 0; start < len(unique); start += codeMaxBatchRows {
		end := start + codeMaxBatchRows
		if end > len(unique) {
			end = len(unique)
		}
		query, args := buildCodeEdgeUncertainInsert(repository, unique[start:end])
		if _, err := store.exec(ctx, query, args...); err != nil {
			return fmt.Errorf("InsertCodeEdgeUncertains falhou: %w", err)
		}
	}
	return nil
}

// GetCodeFileHash devolve (content_hash, ast_hash) do code_files.path.
// Retorna ("", "", false) quando o path não foi indexado no repositório
// carregado pelo contexto.
func GetCodeFileHash(ctx context.Context, db *sql.DB, path string) (contentHash, astHash string, ok bool, err error) {
	store, err := newCodeStore(ctx, db)
	if err != nil {
		return "", "", false, err
	}
	var ah sql.NullString
	err = store.queryRow(ctx,
		`SELECT content_hash, ast_hash FROM code_files WHERE repository = ? AND path = ?`,
		RepositoryFromContext(ctx), path,
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
	_, err = store.exec(ctx,
		`DELETE FROM code_files WHERE repository = ? AND path = ?`,
		RepositoryFromContext(ctx), path,
	)
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
