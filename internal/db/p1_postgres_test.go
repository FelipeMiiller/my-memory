package db

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/pgtest"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// Este arquivo fecha a lacuna P1 da ISSUE-031: o que ficou SEM PROVA no
// Postgres real.
//
// A ISSUE-031 §P1 lista quatro itens sem prova:
//
//	(a) o índice único COMPOSTO de `code_*`;
//	(b) o `ON CONFLICT (repository, path)` no Postgres;
//	(c) a conversão `?` -> `$1` em runtime real;
//	(d) a metade Postgres do critério de aceitação H4 ("code graph gravado no
//	    engine configurado, dialeto certo") — provada em cmd/mem.
//
// Cada teste aqui roda contra um BANCO DESCARTÁVEL criado pelo pgtest.New
// (`mymemory_pgtest_*`), nunca contra o `mymemory` do dono, e só roda com
// MY_MEMORY_PG_TEST=1 — a suíte padrão é hermética.
//
// SEGURANÇA: nada aqui escreve no Postgres do dono. O harness cria e derruba
// o próprio banco; o servidor do dono é usado só como ADMIN (CREATE/DROP
// DATABASE).

// openP1Postgres abre o banco descartável do harness e aplica o schema do
// engine — o mesmo schema que store.NewPostgresStore aplica em produção.
func openP1Postgres(t *testing.T, h *pgtest.Harness) *sql.DB {
	t.Helper()

	pgStore, err := store.NewPostgresStore(h.DSN)
	if err != nil {
		t.Fatalf("NewPostgresStore(%s): %v", h.DBName, err)
	}
	t.Cleanup(func() { _ = pgStore.Close() })
	return pgStore.DB()
}

// indexColumns devolve as colunas cobertas por um índice, na ordem do índice.
func indexColumns(t *testing.T, database *sql.DB, indexName, tableName string) []string {
	t.Helper()

	rows, err := database.Query(`
		SELECT a.attname
		FROM pg_index i
		JOIN pg_class idx ON idx.oid = i.indexrelid
		JOIN pg_class tbl ON tbl.oid = i.indrelid
		JOIN pg_namespace nsp ON nsp.oid = tbl.relnamespace
		CROSS JOIN LATERAL unnest(i.indkey) WITH ORDINALITY AS k(attnum, ord)
		JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = k.attnum
		WHERE idx.relname = $1 AND tbl.relname = $2 AND nsp.nspname = 'public'
		ORDER BY k.ord`, indexName, tableName)
	if err != nil {
		t.Fatalf("colunas do índice %s: %v", indexName, err)
	}
	defer rows.Close()

	var cols []string
	for rows.Next() {
		var col string
		if err := rows.Scan(&col); err != nil {
			t.Fatalf("SCAN colunas do índice %s: %v", indexName, err)
		}
		cols = append(cols, col)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows do índice %s: %v", indexName, err)
	}
	return cols
}

// uniqueIndexColumnSets devolve, para cada índice UNIQUE de code_files, as
// colunas que ele cobre.
func uniqueIndexColumnSets(t *testing.T, database *sql.DB) map[string][]string {
	t.Helper()

	rows, err := database.Query(`
		SELECT idx.relname
		FROM pg_index i
		JOIN pg_class idx ON idx.oid = i.indexrelid
		JOIN pg_class tbl ON tbl.oid = i.indrelid
		JOIN pg_namespace nsp ON nsp.oid = tbl.relnamespace
		WHERE tbl.relname = 'code_files' AND nsp.nspname = 'public' AND i.indisunique`)
	if err != nil {
		t.Fatalf("índices únicos de code_files: %v", err)
	}
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			t.Fatalf("SCAN índices únicos de code_files: %v", err)
		}
		names = append(names, name)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("rows índices únicos de code_files: %v", err)
	}

	out := map[string][]string{}
	for _, name := range names {
		out[name] = indexColumns(t, database, name, "code_files")
	}
	return out
}

// TestP1Postgres_IndiceUnicoCompostoDeCodeFiles prova o item (a) da ISSUE-031
// §P1: no Postgres, `code_files` tem um índice ÚNICO COMPOSTO (repository, path)
// — e não um UNIQUE(path) isolado.
//
// POR QUE IMPORTA: a identidade do arquivo no code graph é (repository, path)
// (ADR-061, CG-06). Um UNIQUE(path) isolado colidiria entre repositórios
// diferentes que têm o mesmo path — e o ON CONFLICT do upsert aponta para o
// índice, então o índice errado quebra o upsert (e não só a leitura).
func TestP1Postgres_IndiceUnicoCompostoDeCodeFiles(t *testing.T) {
	h := pgtest.New(t)
	database := openP1Postgres(t, h)
	ctx := context.Background()

	// 1. O índice existe, é UNIQUE, e está válido e pronto (indisvalid/indisready
	//    são o que o planner exige para usar o índice num ON CONFLICT).
	var unique, valid, ready bool
	var indexDef string
	err := database.QueryRowContext(ctx, `
		SELECT i.indisunique, i.indisvalid, i.indisready, pg_get_indexdef(i.indexrelid)
		FROM pg_index i
		JOIN pg_class idx ON idx.oid = i.indexrelid
		JOIN pg_class tbl ON tbl.oid = i.indrelid
		JOIN pg_namespace nsp ON nsp.oid = tbl.relnamespace
		WHERE idx.relname = 'idx_code_files_repository_path'
		  AND tbl.relname = 'code_files'
		  AND nsp.nspname = 'public'`,
	).Scan(&unique, &valid, &ready, &indexDef)
	if err != nil {
		t.Fatalf("catálogo de idx_code_files_repository_path: %v", err)
	}

	if !unique {
		t.Error("idx_code_files_repository_path NÃO é UNIQUE: o ON CONFLICT (repository, path) não teria índice para casar")
	}
	if !valid {
		t.Error("idx_code_files_repository_path não é indisvalid: o planner não o usa")
	}
	if !ready {
		t.Error("idx_code_files_repository_path não é indisready: o planner não o usa")
	}
	if !strings.Contains(indexDef, "(repository, path)") {
		t.Errorf("definição do índice = %q; want as colunas (repository, path)", indexDef)
	}

	// 2. As colunas são EXATAMENTE (repository, path), nessa ordem.
	cols := indexColumns(t, database, "idx_code_files_repository_path", "code_files")
	want := []string{"repository", "path"}
	if len(cols) != len(want) {
		t.Fatalf("colunas de idx_code_files_repository_path = %v; want %v", cols, want)
	}
	for i := range want {
		if cols[i] != want[i] {
			t.Errorf("coluna %d de idx_code_files_repository_path = %q; want %q (colunas: %v)", i, cols[i], want[i], cols)
		}
	}

	// 3. NENHUM índice único de code_files cobre `path` isolado. É essa
	//    ausência que torna `(repository, path)` OBRIGATÓRIO como conflict
	//    target — e é o que o teste (b) confirma com o erro do servidor.
	for name, cols := range uniqueIndexColumnSets(t, database) {
		if len(cols) == 1 && cols[0] == "path" {
			t.Errorf("o índice único %s cobre só `path`: a identidade do arquivo no code graph "+
				"é (repository, path) e um UNIQUE(path) colidiria entre repositórios", name)
		}
	}

	// 4. O índice EXISTE de fato: sem ele o comportamento seria o mesmo.
	var count int
	if err := database.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM code_files WHERE repository = $1 AND path = $2`,
		"acme", "pkg/alpha.go").Scan(&count); err != nil {
		t.Fatalf("COUNT code_files: %v", err)
	}
	if count != 0 {
		t.Fatalf("o banco descartável já tinha linhas em code_files (%d); o schema não foi aplicado do zero", count)
	}
}

// TestP1Postgres_OnConflictRepositoryPathAtualizaEIsolaRepositorio prova o
// item (b) da ISSUE-031 §P1: no Postgres, `UpsertCodeFile` entra no ramo
// DO UPDATE do ON CONFLICT (repository, path) — e o MESMO path em OUTRO
// repository é uma linha DISTINTA.
//
// POR QUE IMPORTA: o SQLite usa UNIQUE(path) e o Postgres usa
// UNIQUE(repository, path) (ADR-061, CG-06). Se o conflict target do Postgres
// fosse `(path)`, o servidor rejeitaria a query inteira ("no unique or
// exclusion constraint matching the ON CONFLICT specification") e o code
// pipeline deixaria de gravar em Postgres — o defeito silencioso que a
// ISSUE-031ifdefecia.
func TestP1Postgres_OnConflictRepositoryPathAtualizaEIsolaRepositorio(t *testing.T) {
	h := pgtest.New(t)

	// A conexão instrumentada: o SQL que chega ao servidor é gravado, para
	// provar QUAL conflict target foi enviado.
	rec := pgtest.NewRecorder()
	recDB, err := rec.OpenDB(h.DSN)
	if err != nil {
		t.Fatalf("abertura da conexão instrumentada: %v", err)
	}
	defer recDB.Close()

	database := openP1Postgres(t, h)
	ctx := context.Background()

	const path = "pkg/alpha.go"
	acmeCtx := WithRepository(ctx, "acme")

	// 1. Primeiro upsert: insere.
	before := rec.Len()
	id1, err := UpsertCodeFile(acmeCtx, recDB, path, "go", "hash-1", "ast-1", 111, 1700000000)
	if err != nil {
		t.Fatalf("1o UpsertCodeFile: %v", err)
	}
	if id1 == 0 {
		t.Fatal("1o UpsertCodeFile devolveu id 0")
	}

	// 2. Segundo upsert do MESMO (repository, path) com outro conteúdo: tem que
	//    ATUALIZAR a linha existente, não inserir outra nem errar.
	id2, err := UpsertCodeFile(acmeCtx, recDB, path, "rust", "hash-2", "ast-2", 222, 1700000001)
	if err != nil {
		t.Fatalf("2o UpsertCodeFile: %v", err)
	}
	if id1 != id2 {
		t.Fatalf("o 2o upsert criou uma linha nova (id %d != %d): o ON CONFLICT NÃO entrou no DO UPDATE", id2, id1)
	}

	var language, contentHash string
	var sizeBytes int64
	err = database.QueryRowContext(ctx,
		`SELECT language, content_hash, size_bytes FROM code_files WHERE repository = $1 AND path = $2`,
		"acme", path).Scan(&language, &contentHash, &sizeBytes)
	if err != nil {
		t.Fatalf("leitura da linha de code_files: %v", err)
	}
	if language != "rust" {
		t.Errorf("language = %q; want %q (a linha não foi atualizada)", language, "rust")
	}
	if contentHash != "hash-2" {
		t.Errorf("content_hash = %q; want %q (a linha não foi atualizada)", contentHash, "hash-2")
	}
	if sizeBytes != 222 {
		t.Errorf("size_bytes = %d; want 222 (a linha não foi atualizada)", sizeBytes)
	}

	// 3. O MESMO path em OUTRO repository é uma linha DISTINTA — é todo o
	//    sentido do índice composto.
	globexCtx := WithRepository(ctx, "globex")
	id3, err := UpsertCodeFile(globexCtx, recDB, path, "go", "hash-3", "ast-3", 333, 1700000002)
	if err != nil {
		t.Fatalf("UpsertCodeFile no repository globex: %v", err)
	}
	if id3 == id1 {
		t.Errorf("o mesmo path em outro repository colidiu com a linha de acme (id %d): o índice não é composto", id3)
	}

	var total int
	if err := database.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM code_files WHERE path = $1`, path).Scan(&total); err != nil {
		t.Fatalf("COUNT code_files por path: %v", err)
	}
	if total != 2 {
		t.Errorf("linhas com path %q = %d; want 2 (uma por repository)", path, total)
	}

	// 4. O conflict target que CHEGOU AO SERVIDOR é (repository, path).
	queries := rec.Queries()[before:]
	sawComposite := false
	for _, q := range queries {
		if strings.Contains(q, "ON CONFLICT (repository, path) DO UPDATE") {
			sawComposite = true
		}
		if strings.Contains(q, "ON CONFLICT (path)") {
			t.Errorf("o conflict target que chegou ao servidor foi (path): %s", q)
		}
	}
	if !sawComposite {
		t.Fatalf("nenhuma query com 'ON CONFLICT (repository, path) DO UPDATE' chegou ao servidor; queries gravadas: %v", queries)
	}

	// 5. E o servidor REALMENTE rejeita (path) — porque não existe índice único
	//    em path isolado. É a prova de que o item (a) e o item (b) são o mesmo
	//    fato visto de dois lados.
	_, err = database.ExecContext(ctx, `
		INSERT INTO code_files (repository, path, language, size_bytes, content_hash, indexed_at)
		VALUES ('acme', 'pkg/beta.go', 'go', 1, 'h', 1)
		ON CONFLICT (path) DO UPDATE SET language = excluded.language`)
	if err == nil {
		t.Fatal("o servidor ACEITOU ON CONFLICT (path): existe um índice único em path isolado, " +
			"e a premissa do conflict target composto não vale")
	}
	if !strings.Contains(err.Error(), "no unique or exclusion constraint matching the ON CONFLICT specification") {
		t.Errorf("erro inesperado do servidor para ON CONFLICT (path): %v", err)
	}
}

// TestP1Postgres_PlaceholderOrdinalEmRuntimeReal prova o item (c) da
// ISSUE-031 §P1: a conversão `?` -> `$1` acontece em runtime real, contra o
// Postgres de verdade — e um `?` dentro de literal de string ou de
// identificador NÃO é placeholder.
//
// POR QUE IMPORTA: o SQLite aceita `?` e o lib/pq exige `$N`. A conversão
// (internal/db/dialect.go, bindOrdinal) é feita no processo, e nada no SQL
// enviado ao servidor mostra que ela rodou. Um erro ali (contabilizar o `?` de
// um literal) desloca os ordinais: o `$1` viraria `$2` e os valores iriam
// para as colunas erradas — sem erro visível.
func TestP1Postgres_PlaceholderOrdinalEmRuntimeReal(t *testing.T) {
	h := pgtest.New(t)

	rec := pgtest.NewRecorder()
	recDB, err := rec.OpenDB(h.DSN)
	if err != nil {
		t.Fatalf("abertura da conexão instrumentada: %v", err)
	}
	defer recDB.Close()

	database := openP1Postgres(t, h)
	ctx := context.Background()

	// codeStore é o caminho de produção da escrita: codeStore.exec/queryRow →
	// dialect.Bind → bindOrdinal → db.ExecContext/QueryContext.
	cs, err := newCodeStore(ctx, recDB)
	if err != nil {
		t.Fatalf("newCodeStore: %v", err)
	}
	if !cs.dialect.IsPostgres() {
		t.Fatalf("dialeto detectado = %q; want %q", cs.dialect, DialectPostgres)
	}

	// 1. Query com `?` de verdade, um `?` dentro de literal de string e um `?`
	//    dentro de identificador double-quoted.
	const query = `SELECT ? AS "col?x", 'a?b' AS "lit", ? AS plain`

	before := rec.Len()
	var colx, lit, plain string
	if err := cs.queryRow(ctx, query, "primeiro", "segundo").Scan(&colx, &lit, &plain); err != nil {
		t.Fatalf("query com placeholders: %v", err)
	}

	// 1.1. Os VALORES chegaram na coluna certa (nada deslocado).
	if colx != "primeiro" {
		t.Errorf("primeira coluna = %q; want %q", colx, "primeiro")
	}
	if lit != "a?b" {
		t.Errorf("coluna do literal = %q; want %q (o servidor recebeu o literal intacto)", lit, "a?b")
	}
	if plain != "segundo" {
		t.Errorf("terceira coluna = %q; want %q", plain, "segundo")
	}

	// 1.2. O SQL que CHEGOU AO SERVIDOR tem os ordinais certos, e o `?` dos
	//      literais/identificadores ficou intacto. A comparação é exata: um
	//      `?` contado como placeholder mudaria `$2` para `$3` e a linha
	//      inteira aqui mudaria.
	queries := rec.Queries()[before:]
	if len(queries) == 0 {
		t.Fatal("nenhuma query foi gravada pelo recorder")
	}
	got := queries[len(queries)-1]
	want := `SELECT $1 AS "col?x", 'a?b' AS "lit", $2 AS plain`
	if got != want {
		t.Errorf("SQL que chegou ao servidor:\n  got  %s\n  want %s", got, want)
	}

	// 2. O caminho de ESCRITA também converte: o upsert de production usa `?`
	//    nos sete campos e o servidor recebe `$1`..`$7`.
	before = rec.Len()
	const weirdPath = "pkg/we?ird.go"
	const weirdHash = "hash?com?marcas"
	id, err := UpsertCodeFile(WithRepository(ctx, "acme"), recDB, weirdPath, "go", weirdHash, "", 42, 1700000000)
	if err != nil {
		t.Fatalf("UpsertCodeFile com `?` no path: %v", err)
	}

	var upsertSQL string
	for _, q := range rec.Queries()[before:] {
		if strings.Contains(q, "INSERT INTO code_files") {
			upsertSQL = q
		}
	}
	if upsertSQL == "" {
		t.Fatal("o INSERT INTO code_files não foi gravado pelo recorder")
	}
	if strings.Contains(upsertSQL, "?") {
		t.Errorf("o INSERT que chegou ao servidor ainda tem `?` de placeholder: %s", upsertSQL)
	}
	if !strings.Contains(upsertSQL, "$7") {
		t.Errorf("o INSERT que chegou ao servidor não tem o 7o ordinal ($7): %s", upsertSQL)
	}

	// 2.1. E os valores aterram na coluna certa: um binding deslocado
	//      trocaria path por content_hash (e o hash pelo path).
	var gotPath, gotHash, gotLanguage string
	var gotSize int64
	err = database.QueryRowContext(ctx,
		`SELECT path, content_hash, language, size_bytes FROM code_files WHERE id = $1`, id).
		Scan(&gotPath, &gotHash, &gotLanguage, &gotSize)
	if err != nil {
		t.Fatalf("leitura da linha gravada: %v", err)
	}
	if gotPath != weirdPath {
		t.Errorf("path gravado = %q; want %q", gotPath, weirdPath)
	}
	if gotHash != weirdHash {
		t.Errorf("content_hash gravado = %q; want %q", gotHash, weirdHash)
	}
	if gotLanguage != "go" {
		t.Errorf("language gravada = %q; want %q", gotLanguage, "go")
	}
	if gotSize != 42 {
		t.Errorf("size_bytes gravado = %d; want 42", gotSize)
	}
}
