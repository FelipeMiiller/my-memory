package db_test

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/pgtest"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// A integração PostgreSQL do caminho de ESCRITA em LOTE (a ponte e o code
// graph).
//
// POR QUE ELA EXISTE: o defeito medido pelo dono (≈268 ms por round-trip no
// Neon) só aparece no engine remoto — no SQLite local cada statement é
// instantâneo. O `INSERT` multi-linha com `ON CONFLICT` precisa ser provado
// no SERVIDOR (o `ON CONFLICT DO UPDATE` do Postgres rejeita o conflito da
// mesma linha duas vezes no mesmo statement — o que obriga a remover a
// duplicata do lote) e o NÚMERO de statements precisa ser contado no SQL que
// chega ao servidor (o `pgtest.Recorder` grava exatamente esse SQL).
//
// SEGURANÇA: o banco é DESCARTÁVEL (`pgtest.New` cria e derruba
// `mymemory_pgtest_*`). O Postgres do dono nunca é alvo.
//
// COMO RODAR:
//
//	MY_MEMORY_PG_URL=<url> MY_MEMORY_PG_TEST=1 go test -count=1 ./internal/db/...
//
// Sem MY_MEMORY_PG_TEST=1 o teste é pulado (a suíte padrão é hermética).

// batchPostgresFixture aplica o schema no banco descartável e devolve uma
// conexão INSTRUMENTADA (o recorder conta o SQL que chega ao servidor).
func batchPostgresFixture(t *testing.T) (*sql.DB, *pgtest.Recorder, *store.PostgresStore) {
	t.Helper()

	h := pgtest.New(t)

	pgStore, err := store.NewPostgresStore(h.DSN)
	if err != nil {
		t.Fatalf("NewPostgresStore(%s): %v", h.DBName, err)
	}
	t.Cleanup(func() { pgStore.Close() })

	rec := pgtest.NewRecorder()
	recDB, err := rec.OpenDB(h.DSN)
	if err != nil {
		t.Fatalf("conexão instrumentada: %v", err)
	}
	t.Cleanup(func() { recDB.Close() })

	return recDB, rec, pgStore
}

// batchCountInserts conta quantos INSERTs o recorder viu para uma tabela.
//
// O " (" no fim do nome da tabela é o que separa `code_edges` de
// `code_edges_uncertain` (o nome da primeira é prefixo do da segunda).
func batchCountInserts(rec *pgtest.Recorder, table string) int {
	count := 0
	for _, q := range rec.Queries() {
		if strings.Contains(q, "INSERT INTO "+table+" (") || strings.Contains(q, "INSERT OR IGNORE INTO "+table+" (") {
			count++
		}
	}
	return count
}

// TestLotePostgresPonteGravaDoisStatementsPorDocumento prova (BR-01, PRV-02)
// que a ponte grava NO SERVIDOR com 2 statements por documento, mesmo com
// vários símbolos casados — e que o grafo final é o mesmo.
func TestLotePostgresPonteGravaDoisStatementsPorDocumento(t *testing.T) {
	database, rec, pgStore := batchPostgresFixture(t)
	ctx := db.WithRepository(context.Background(), "lote-repo")

	// A migração aditiva (epistemic_status/confidence/repository) é o mesmo
	// passo que o code pipeline roda antes de indexar.
	if err := db.EnsureCodeTables(ctx, database); err != nil {
		t.Fatalf("EnsureCodeTables: %v", err)
	}

	const docID = "doc-lote-pg"
	if err := pgStore.InsertDocumentWithMeta(ctx, "lote-repo", docID, "docs/lote.md", "lote-repo",
		1700000000, "hash-doc", "", "", "resource", 0, 0); err != nil {
		t.Fatalf("InsertDocumentWithMeta: %v", err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO chunks (id, repository, document_id, chunk_index, content)
		VALUES ($1, $2, $3, 0, $4)`,
		"chunk-lote-pg", "lote-repo", docID, "a nota menciona pkg.Alpha e pkg.Beta e pkg.Gamma"); err != nil {
		t.Fatalf("INSERT chunks: %v", err)
	}

	fileID, err := db.UpsertCodeFile(ctx, database, "pkg/a.go", "go", "hash-a", "ast-a", 128, 1700000000)
	if err != nil {
		t.Fatalf("UpsertCodeFile: %v", err)
	}
	for _, qn := range []string{"pkg.Alpha", "pkg.Beta", "pkg.Gamma"} {
		if _, err := db.UpsertCodeSymbol(ctx, database, fileID, "function", qn[strings.LastIndex(qn, ".")+1:],
			qn, "func "+qn+"()", "", 1, 3, 0, 0); err != nil {
			t.Fatalf("UpsertCodeSymbol(%s): %v", qn, err)
		}
	}

	// A partir daqui só conta o que a PONTE faz.
	rec.Reset()

	result, err := db.BuildCodeBridge(ctx, database)
	if err != nil {
		t.Fatalf("BuildCodeBridge: %v", err)
	}
	if result.Matched != 3 {
		t.Errorf("matched = %d; want 3", result.Matched)
	}

	if got := batchCountInserts(rec, "graph_nodes"); got != 1 {
		t.Errorf("INSERTs em graph_nodes = %d; want 1 (um statement multi-linha por documento)\n%v",
			got, rec.Queries())
	}
	if got := batchCountInserts(rec, "graph_edges"); got != 1 {
		t.Errorf("INSERTs em graph_edges = %d; want 1 (um statement multi-linha por documento)\n%v",
			got, rec.Queries())
	}

	// O grafo final tem os 3 nós e as 3 arestas.
	var nodes, edges int
	if err := database.QueryRow(`SELECT COUNT(*) FROM graph_nodes WHERE repository = $1 AND type = $2`,
		"lote-repo", db.CodeSymbolNodeType).Scan(&nodes); err != nil {
		t.Fatalf("COUNT graph_nodes: %v", err)
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM graph_edges WHERE repository = $1 AND relation = $2`,
		"lote-repo", db.BridgeRelation).Scan(&edges); err != nil {
		t.Fatalf("COUNT graph_edges: %v", err)
	}
	if nodes != 3 || edges != 3 {
		t.Errorf("grafo = %d nós / %d arestas; want 3 / 3", nodes, edges)
	}

	// Idempotência: rodar de novo não duplica e continua com 2 statements.
	rec.Reset()
	if _, err := db.BuildCodeBridge(ctx, database); err != nil {
		t.Fatalf("BuildCodeBridge (2a vez): %v", err)
	}
	if got := batchCountInserts(rec, "graph_nodes"); got != 1 {
		t.Errorf("INSERTs em graph_nodes (2a vez) = %d; want 1", got)
	}
	if got := batchCountInserts(rec, "graph_edges"); got != 1 {
		t.Errorf("INSERTs em graph_edges (2a vez) = %d; want 1", got)
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM graph_edges WHERE repository = $1 AND relation = $2`,
		"lote-repo", db.BridgeRelation).Scan(&edges); err != nil {
		t.Fatalf("COUNT graph_edges: %v", err)
	}
	if edges != 3 {
		t.Errorf("arestas depois de 2 passes = %d; want 3", edges)
	}
}

// TestLotePostgresSimbolosGravaUmStatementPorArquivo prova (SYM-01, SYM-03)
// que o `INSERT` multi-linha em code_symbols é aceito pelo servidor, resolve
// os mesmos ids e custa 1 statement de escrita por arquivo.
func TestLotePostgresSimbolosGravaUmStatementPorArquivo(t *testing.T) {
	database, rec, _ := batchPostgresFixture(t)
	ctx := db.WithRepository(context.Background(), "lote-repo")
	if err := db.EnsureCodeTables(ctx, database); err != nil {
		t.Fatalf("EnsureCodeTables: %v", err)
	}

	fileID, err := db.UpsertCodeFile(ctx, database, "pkg/a.go", "go", "hash-a", "ast-a", 128, 1700000000)
	if err != nil {
		t.Fatalf("UpsertCodeFile: %v", err)
	}

	rows := make([]db.CodeSymbolRow, 0, 5)
	for i := 0; i < 5; i++ {
		rows = append(rows, db.CodeSymbolRow{
			Kind:          "function",
			Name:          fmt.Sprintf("Sym%d", i),
			QualifiedName: fmt.Sprintf("pkg.Sym%d", i),
			Signature:     "func()",
			StartLine:     i*10 + 1,
			EndLine:       i*10 + 3,
			StartCol:      1,
			EndCol:        2,
		})
	}

	rec.Reset()
	ids, err := db.UpsertCodeSymbols(ctx, database, fileID, rows)
	if err != nil {
		t.Fatalf("UpsertCodeSymbols: %v", err)
	}
	if len(ids) != len(rows) {
		t.Fatalf("ids = %d; want %d", len(ids), len(rows))
	}
	if got := batchCountInserts(rec, "code_symbols"); got != 1 {
		t.Errorf("INSERTs em code_symbols = %d; want 1 (um statement por arquivo)\n%v", got, rec.Queries())
	}

	// Cada id devolvido é o id do símbolo correspondente.
	for i, id := range ids {
		var name string
		if err := database.QueryRow(`SELECT name FROM code_symbols WHERE id = $1`, id).Scan(&name); err != nil {
			t.Fatalf("símbolo %d (id %d) ausente: %v", i, id, err)
		}
		if name != rows[i].Name {
			t.Errorf("id[%d] = %d aponta para %q; want %q", i, id, name, rows[i].Name)
		}
	}

	// Idempotência: rodar de novo não duplica nem muda os ids.
	ids2, err := db.UpsertCodeSymbols(ctx, database, fileID, rows)
	if err != nil {
		t.Fatalf("UpsertCodeSymbols (2a vez): %v", err)
	}
	for i := range ids {
		if ids[i] != ids2[i] {
			t.Errorf("id[%d] mudou entre as rodadas: %d → %d", i, ids[i], ids2[i])
		}
	}
	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM code_symbols WHERE file_id = $1`, fileID).Scan(&count); err != nil {
		t.Fatalf("COUNT code_symbols: %v", err)
	}
	if count != len(rows) {
		t.Errorf("code_symbols = %d depois de 2 rodadas; want %d", count, len(rows))
	}
}

// TestCustoRealUpsertCodeFilePostgres mede o custo REAL do `UpsertCodeFile`
// no Postgres (a pergunta do dono: "1 statement por arquivo já é 1
// round-trip — se for isso, não mexa").
//
// A resposta: são **2 round-trips por arquivo** no Postgres — o INSERT e um
// SELECT extra para resolver o id, porque o `lib/pq` não implementa
// `LastInsertId`. No SQLite são 1 (o `LastInsertId` do modernc devolve o id
// da linha gravada).
//
// O teste NÃO fixa esse número (uma otimização futura não pode quebrar o
// gate): ele falha só se o custo CRESCER — e registra o valor medido.
func TestCustoRealUpsertCodeFilePostgres(t *testing.T) {
	database, rec, _ := batchPostgresFixture(t)
	ctx := db.WithRepository(context.Background(), "lote-repo")
	if err := db.EnsureCodeTables(ctx, database); err != nil {
		t.Fatalf("EnsureCodeTables: %v", err)
	}

	rec.Reset()
	if _, err := db.UpsertCodeFile(ctx, database, "pkg/a.go", "go", "hash", "ast", 100, 1700000000); err != nil {
		t.Fatalf("UpsertCodeFile: %v", err)
	}

	queries := rec.Queries()
	t.Logf("UpsertCodeFile no Postgres: %d round-trip(s) por arquivo", len(queries))
	for _, q := range queries {
		t.Logf("  %s", strings.TrimSpace(strings.SplitN(q, "\n", 2)[0]))
	}
	if len(queries) > 2 {
		t.Errorf("UpsertCodeFile custou %d round-trips; want <= 2 (1 INSERT + 1 SELECT de id)", len(queries))
	}
}

// TestLotePostgresIncertasGravaUmStatementPorArquivo prova (UNC-01, UNC-03)
// que o `INSERT` multi-linha em `code_edges_uncertain` é aceito pelo
// servidor, custa 1 statement por arquivo, funde a aresta repetida no lote
// e é idempotente.
//
// O bucket era o último gargalo medido no Neon: ~1,6 aresta/s (1 statement
// por aresta) contra ~110 min para popular o repo inteiro.
func TestLotePostgresIncertasGravaUmStatementPorArquivo(t *testing.T) {
	database, rec, _ := batchPostgresFixture(t)
	ctx := db.WithRepository(context.Background(), "lote-repo")
	if err := db.EnsureCodeTables(ctx, database); err != nil {
		t.Fatalf("EnsureCodeTables: %v", err)
	}

	fileID, err := db.UpsertCodeFile(ctx, database, "pkg/a.go", "go", "hash-a", "ast-a", 128, 1700000000)
	if err != nil {
		t.Fatalf("UpsertCodeFile: %v", err)
	}
	symbolID, err := db.UpsertCodeSymbol(ctx, database, fileID, "function", "Caller", "pkg.Caller",
		"func Caller()", "", 1, 3, 0, 0)
	if err != nil {
		t.Fatalf("UpsertCodeSymbol: %v", err)
	}

	rows := []db.CodeEdgeUncertainRow{
		{SrcSymbolID: symbolID, FileID: fileID, DstCandidateName: "pkg.Foo", Kind: "calls",
			Confidence: 0.3, Reason: "destino não encontrado", StartLine: 10},
		{SrcSymbolID: symbolID, FileID: fileID, DstCandidateName: "pkg.Bar", Kind: "calls",
			Confidence: 0.5, Reason: "ambíguo: 2 símbolos", StartLine: 20},
		// Duplicata da primeira (mesma chave): o payload do ÚLTIMO vence —
		// é o que o ON CONFLICT DO UPDATE grava.
		{SrcSymbolID: symbolID, FileID: fileID, DstCandidateName: "pkg.Foo", Kind: "calls",
			Confidence: 0.8, Reason: "atualizado", StartLine: 99},
	}

	rec.Reset()
	if err := db.InsertCodeEdgeUncertains(ctx, database, rows); err != nil {
		t.Fatalf("InsertCodeEdgeUncertains: %v", err)
	}
	if got := batchCountInserts(rec, "code_edges_uncertain"); got != 1 {
		t.Errorf("INSERTs em code_edges_uncertain = %d; want 1 (um statement por arquivo)\n%v",
			got, rec.Queries())
	}
	if got := batchCountInserts(rec, "code_edges"); got != 0 {
		t.Errorf("INSERTs em code_edges = %d; want 0 (este teste é só do bucket incerto)", got)
	}

	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM code_edges_uncertain`).Scan(&count); err != nil {
		t.Fatalf("COUNT code_edges_uncertain: %v", err)
	}
	if count != 2 {
		t.Errorf("code_edges_uncertain = %d; want 2 (a duplicata do lote vira uma linha)", count)
	}

	// A linha duplicada ficou com o payload do ÚLTIMO.
	var confidence float64
	var reason string
	var startLine int
	if err := database.QueryRow(`
		SELECT confidence, reason, start_line
		  FROM code_edges_uncertain
		 WHERE src_symbol_id = $1 AND dst_candidate_name = $2 AND kind = $3`,
		symbolID, "pkg.Foo", "calls").Scan(&confidence, &reason, &startLine); err != nil {
		t.Fatalf("SELECT code_edges_uncertain: %v", err)
	}
	if confidence != 0.8 || reason != "atualizado" || startLine != 99 {
		t.Errorf("linha duplicada = %.1f/%q/%d; want 0.8/atualizado/99 (o payload do último vence)",
			confidence, reason, startLine)
	}

	// Idempotência: rodar de novo não duplica nem altera o conteúdo.
	if err := db.InsertCodeEdgeUncertains(ctx, database, rows); err != nil {
		t.Fatalf("InsertCodeEdgeUncertains (2a vez): %v", err)
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM code_edges_uncertain`).Scan(&count); err != nil {
		t.Fatalf("COUNT code_edges_uncertain: %v", err)
	}
	if count != 2 {
		t.Errorf("code_edges_uncertain = %d depois de 2 rodadas; want 2", count)
	}
}

// TestLotePostgresArestasGravaUmStatementPorArquivo prova (EDG-01, EDG-02)
// que o `INSERT` multi-linha em code_edges é aceito pelo servidor, funde a
// aresta repetida no lote e é idempotente.
func TestLotePostgresArestasGravaUmStatementPorArquivo(t *testing.T) {
	database, rec, _ := batchPostgresFixture(t)
	ctx := db.WithRepository(context.Background(), "lote-repo")
	if err := db.EnsureCodeTables(ctx, database); err != nil {
		t.Fatalf("EnsureCodeTables: %v", err)
	}

	fileID, err := db.UpsertCodeFile(ctx, database, "pkg/a.go", "go", "hash-a", "ast-a", 128, 1700000000)
	if err != nil {
		t.Fatalf("UpsertCodeFile: %v", err)
	}
	ids := make([]int64, 0, 2)
	for i := 0; i < 2; i++ {
		id, err := db.UpsertCodeSymbol(ctx, database, fileID, "function", fmt.Sprintf("Sym%d", i),
			fmt.Sprintf("pkg.Sym%d", i), "func()", "", i*10+1, i*10+3, 0, 0)
		if err != nil {
			t.Fatalf("UpsertCodeSymbol: %v", err)
		}
		ids = append(ids, id)
	}

	edges := []db.CodeEdgeRow{
		{SrcSymbolID: ids[0], DstSymbolID: ids[1], FileID: fileID, Kind: "calls",
			StartLine: 30, EndLine: 30, Confidence: 1.0, EpistemicStatus: db.EpistemicExtracted},
		// Duplicata com posição MENOR: a menor vence (como no ON CONFLICT).
		{SrcSymbolID: ids[0], DstSymbolID: ids[1], FileID: fileID, Kind: "calls",
			StartLine: 10, EndLine: 12, Confidence: 0.9, EpistemicStatus: db.EpistemicInferred},
		// Uma segunda aresta DISTINTA: o lote tem 2 chaves → 1 statement.
		{SrcSymbolID: ids[1], DstSymbolID: ids[0], FileID: fileID, Kind: "references",
			StartLine: 40, EndLine: 40, Confidence: 0.8, EpistemicStatus: db.EpistemicInferred},
	}

	rec.Reset()
	if err := db.UpsertCodeEdges(ctx, database, edges); err != nil {
		t.Fatalf("UpsertCodeEdges: %v", err)
	}
	if got := batchCountInserts(rec, "code_edges"); got != 1 {
		t.Errorf("INSERTs em code_edges = %d; want 1 (um statement por arquivo)\n%v", got, rec.Queries())
	}

	var count int
	var startLine, endLine int
	var confidence float64
	var status string
	if err := database.QueryRow(`
		SELECT COUNT(*), MIN(start_line), MIN(end_line), MIN(confidence), MIN(epistemic_status)
		  FROM code_edges WHERE src_symbol_id = $1 AND dst_symbol_id = $2 AND kind = $3`,
		ids[0], ids[1], "calls").Scan(&count, &startLine, &endLine, &confidence, &status); err != nil {
		t.Fatalf("SELECT code_edges: %v", err)
	}
	if count != 1 {
		t.Errorf("code_edges = %d; want 1 (a duplicata do lote vira uma linha)", count)
	}
	if startLine != 10 || endLine != 12 {
		t.Errorf("posição = %d-%d; want 10-12 (a menor posição vence)", startLine, endLine)
	}

	// Idempotência.
	if err := db.UpsertCodeEdges(ctx, database, edges); err != nil {
		t.Fatalf("UpsertCodeEdges (2a vez): %v", err)
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM code_edges`).Scan(&count); err != nil {
		t.Fatalf("COUNT code_edges: %v", err)
	}
	if count != 2 {
		t.Errorf("code_edges = %d depois de 2 rodadas; want 2", count)
	}
}
