package db

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// countingBridgeWriter conta os statements de ESCRITA da ponte.
//
// É a costura que prova que a ponte grava em LOTE: o *sql.DB concreto não dá
// hook, mas a ponte escreve através de uma interface de um método — o
// wrapper abaixo intercepta cada ExecContext sem mudar o banco.
type countingBridgeWriter struct {
	inner      *sql.DB
	statements []string
}

func (w *countingBridgeWriter) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	w.statements = append(w.statements, query)
	return w.inner.ExecContext(ctx, query, args...)
}

// bridgeLoteFixture semeia um documento que menciona N símbolos e devolve o
// banco pronto para a ponte.
func bridgeLoteFixture(t *testing.T, ctx context.Context, symbols int) *sql.DB {
	t.Helper()

	database, err := InitDB(filepath.Join(t.TempDir(), "bridge-lote.db"))
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}

	const docID = "doc-bridge-lote"
	if err := InsertDocumentWithMeta(ctx, database, docID, "docs/lote.md", "Lote",
		1700000000, "doc-hash-lote", "", "", "resource", 0, 0); err != nil {
		t.Fatalf("InsertDocumentWithMeta: %v", err)
	}

	var mentions strings.Builder
	for i := 0; i < symbols; i++ {
		name := fmt.Sprintf("Sym%d", i)
		qualified := fmt.Sprintf("pkg.%s", name)
		bridgeSeedSymbol(t, ctx, database, fmt.Sprintf("pkg/%s.go", name), name, qualified)
		mentions.WriteString(qualified)
		mentions.WriteString(" ")
	}
	if err := InsertChunk(ctx, database, "chunk-bridge-lote", docID, mentions.String(), 0, nil); err != nil {
		t.Fatalf("InsertChunk: %v", err)
	}
	return database
}

// TestPonteLoteGravaDoisStatementsPorDocumento prova o requisito BR-01: a
// ponte emite 2 statements de escrita por documento (1 nós + 1 arestas),
// e não 2 por símbolo casado.
func TestPonteLoteGravaDoisStatementsPorDocumento(t *testing.T) {
	ctx := context.Background()
	const symbols = 8

	database := bridgeLoteFixture(t, ctx, symbols)
	defer database.Close()

	writer := &countingBridgeWriter{inner: database}
	result, err := buildCodeBridgeWithWriter(ctx, database, "", writer)
	if err != nil {
		t.Fatalf("buildCodeBridgeWithWriter: %v", err)
	}

	if len(writer.statements) != 2 {
		t.Fatalf("statements de escrita = %d; want 2 (1 graph_nodes + 1 graph_edges)\n%v",
			len(writer.statements), writer.statements)
	}
	if !strings.Contains(writer.statements[0], "graph_nodes") {
		t.Errorf("o 1º statement não é o INSERT de graph_nodes: %q", writer.statements[0])
	}
	if !strings.Contains(writer.statements[1], "graph_edges") {
		t.Errorf("o 2º statement não é o INSERT de graph_edges: %q", writer.statements[1])
	}

	// As contagens do resultado são as mesmas do caminho por símbolo.
	if result.Matched != symbols || result.Nodes != symbols || result.Edges != symbols {
		t.Errorf("resultado = %+v; want nodes=%d edges=%d matched=%d",
			result, symbols, symbols, symbols)
	}

	// O grafo final tem N nós de código e N arestas references_code.
	var nodes, edges int
	if err := database.QueryRow(`SELECT COUNT(*) FROM graph_nodes WHERE type = ?`, CodeSymbolNodeType).Scan(&nodes); err != nil {
		t.Fatalf("COUNT graph_nodes: %v", err)
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM graph_edges WHERE relation = ?`, BridgeRelation).Scan(&edges); err != nil {
		t.Fatalf("COUNT graph_edges: %v", err)
	}
	if nodes != symbols || edges != symbols {
		t.Errorf("grafo = %d nós / %d arestas; want %d / %d", nodes, edges, symbols, symbols)
	}
}

// TestPonteLoteNumeroDeStatementsConstante prova que o número de statements
// é CONSTANTE (2) e não O(N): o mesmo documento casa com 1 símbolo ou com 25
// e o custo de escrita é o mesmo.
func TestPonteLoteNumeroDeStatementsConstante(t *testing.T) {
	ctx := context.Background()

	for _, symbols := range []int{1, 5, 25} {
		symbols := symbols
		t.Run(fmt.Sprintf("simbolos=%d", symbols), func(t *testing.T) {
			database := bridgeLoteFixture(t, ctx, symbols)
			defer database.Close()

			writer := &countingBridgeWriter{inner: database}
			if _, err := buildCodeBridgeWithWriter(ctx, database, "", writer); err != nil {
				t.Fatalf("buildCodeBridgeWithWriter: %v", err)
			}
			if len(writer.statements) != 2 {
				t.Fatalf("statements = %d com %d símbolos; want 2 (constante, não O(N))",
					len(writer.statements), symbols)
			}
		})
	}
}

// TestPonteLoteEquivaleAoCaminhoPorSimbolo prova que o LOTE grava o mesmo
// grafo que o caminho por símbolo (upsertBridgeNode + upsertBridgeEdge).
func TestPonteLoteEquivaleAoCaminhoPorSimbolo(t *testing.T) {
	ctx := context.Background()
	const symbols = 6

	loteDB := bridgeLoteFixture(t, ctx, symbols)
	defer loteDB.Close()
	porSimboloDB := bridgeLoteFixture(t, ctx, symbols)
	defer porSimboloDB.Close()

	// 1. O caminho em lote roda no primeiro banco.
	writer := &countingBridgeWriter{inner: loteDB}
	if _, err := buildCodeBridgeWithWriter(ctx, loteDB, "", writer); err != nil {
		t.Fatalf("buildCodeBridgeWithWriter: %v", err)
	}

	// 2. O caminho por símbolo roda no segundo, com os MESMOS pares
	//    (documento → nó de código) e o MESMO status epistêmico.
	rows, err := loteDB.Query(`
		SELECT e.source_id, e.target_id, e.epistemic_status, n.name
		  FROM graph_edges e
		  JOIN graph_nodes n ON n.id = e.target_id
		 WHERE e.relation = ?
		 ORDER BY e.target_id ASC`, BridgeRelation)
	if err != nil {
		t.Fatalf("SELECT graph_edges: %v", err)
	}
	defer rows.Close()
	type pair struct {
		source string
		target string
		status string
		name   string
	}
	var pairs []pair
	for rows.Next() {
		var p pair
		if err := rows.Scan(&p.source, &p.target, &p.status, &p.name); err != nil {
			t.Fatalf("scan: %v", err)
		}
		pairs = append(pairs, p)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	if len(pairs) != symbols {
		t.Fatalf("pares casados = %d; want %d", len(pairs), symbols)
	}

	dialect := DetectDialect(ctx, porSimboloDB)
	for _, p := range pairs {
		if err := upsertBridgeNode(ctx, porSimboloDB, dialect, p.target, p.name); err != nil {
			t.Fatalf("upsertBridgeNode(%s): %v", p.target, err)
		}
		if err := upsertBridgeEdge(ctx, porSimboloDB, dialect, p.source, p.target, p.status); err != nil {
			t.Fatalf("upsertBridgeEdge(%s): %v", p.target, err)
		}
	}

	// 3. Os dois grafos são idênticos.
	assertBridgeStateEqual(t, loteDB, porSimboloDB)
}

// TestPonteLoteIdempotente prova que rodar a ponte duas vezes não duplica
// nada e continua custando 2 statements por documento.
func TestPonteLoteIdempotente(t *testing.T) {
	ctx := context.Background()
	const symbols = 4

	database := bridgeLoteFixture(t, ctx, symbols)
	defer database.Close()

	for round := 1; round <= 2; round++ {
		writer := &countingBridgeWriter{inner: database}
		if _, err := buildCodeBridgeWithWriter(ctx, database, "", writer); err != nil {
			t.Fatalf("buildCodeBridgeWithWriter (rodada %d): %v", round, err)
		}
		if len(writer.statements) != 2 {
			t.Fatalf("statements (rodada %d) = %d; want 2", round, len(writer.statements))
		}
	}

	var nodes, edges int
	if err := database.QueryRow(`SELECT COUNT(*) FROM graph_nodes WHERE type = ?`, CodeSymbolNodeType).Scan(&nodes); err != nil {
		t.Fatalf("COUNT graph_nodes: %v", err)
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM graph_edges WHERE relation = ?`, BridgeRelation).Scan(&edges); err != nil {
		t.Fatalf("COUNT graph_edges: %v", err)
	}
	if nodes != symbols || edges != symbols {
		t.Errorf("após 2 passes: %d nós / %d arestas; want %d / %d", nodes, edges, symbols, symbols)
	}
}

// TestPonteLoteConstrutoresDeStatement prova a forma do SQL gerado: N
// tuplas no VALUES, o número de parâmetros certo e o Bind numerando $1..$N
// sem tocar o `?` que exista dentro de literal de string.
func TestPonteLoteConstrutoresDeStatement(t *testing.T) {
	rows := []bridgeNodeRow{
		{ID: "code:repo:a.A", Name: "A"},
		{ID: "code:repo:b.B", Name: "B"},
		{ID: "code:repo:c.C", Name: ""}, // sem nome → o id vira o nome
	}

	t.Run("nos sqlite", func(t *testing.T) {
		query, args := buildNodeInsert(DialectSQLite, "repo", rows)
		if got := strings.Count(query, "(?"); got != len(rows) {
			t.Errorf("tuplas = %d; want %d", got, len(rows))
		}
		if got := strings.Count(query, "?"); got != len(rows)*2 {
			t.Errorf("placeholders = %d; want %d", got, len(rows)*2)
		}
		if len(args) != len(rows)*2 {
			t.Errorf("args = %d; want %d", len(args), len(rows)*2)
		}
		if args[0] != "code:repo:a.A" || args[1] != "A" {
			t.Errorf("args[0:2] = %v; want [code:repo:a.A A]", args[0:2])
		}
		// O 3º símbolo não tem nome: o id é o nome.
		if args[4] != "code:repo:c.C" {
			t.Errorf("args[4] = %v; want code:repo:c.C", args[4])
		}
		if !strings.Contains(query, "INSERT OR IGNORE INTO graph_nodes") {
			t.Errorf("query sem INSERT OR IGNORE: %q", query)
		}
	})

	t.Run("nos postgres", func(t *testing.T) {
		query, args := buildNodeInsert(DialectPostgres, "repo", rows)
		if got := strings.Count(query, "(?"); got != len(rows) {
			t.Errorf("tuplas = %d; want %d", got, len(rows))
		}
		if got := strings.Count(query, "?"); got != len(rows)*3 {
			t.Errorf("placeholders = %d; want %d", got, len(rows)*3)
		}
		if len(args) != len(rows)*3 {
			t.Errorf("args = %d; want %d", len(args), len(rows)*3)
		}
		if args[0] != "code:repo:a.A" || args[1] != "repo" || args[2] != "A" {
			t.Errorf("args[0:3] = %v; want [code:repo:a.A repo A]", args[0:3])
		}
		bound := DialectPostgres.Bind(query)
		if strings.Contains(bound, "?") {
			t.Errorf("a query convertida ainda tem `?`: %q", bound)
		}
		if !strings.Contains(bound, "$1") || !strings.Contains(bound, "$9") {
			t.Errorf("a query convertida não numera $1..$9: %q", bound)
		}
		if !strings.Contains(bound, "'"+CodeSymbolNodeType+"'") {
			t.Errorf("o literal do tipo foi alterado pelo Bind: %q", bound)
		}
		if !strings.Contains(bound, "ON CONFLICT (repository, id) DO NOTHING") {
			t.Errorf("query sem o ON CONFLICT do Postgres: %q", bound)
		}
	})

	t.Run("arestas sqlite", func(t *testing.T) {
		edgeRows := []bridgeEdgeRow{
			{TargetID: "code:repo:a.A", Status: EpistemicExtracted},
			{TargetID: "code:repo:b.B", Status: EpistemicInferred},
		}
		query, args := buildEdgeInsert(DialectSQLite, "repo", "doc-1", edgeRows)
		if got := strings.Count(query, "(?"); got != len(edgeRows) {
			t.Errorf("tuplas = %d; want %d", got, len(edgeRows))
		}
		if got := strings.Count(query, "?"); got != len(edgeRows)*3 {
			t.Errorf("placeholders = %d; want %d", got, len(edgeRows)*3)
		}
		if len(args) != len(edgeRows)*3 {
			t.Errorf("args = %d; want %d", len(args), len(edgeRows)*3)
		}
		if args[0] != "doc-1" || args[1] != "code:repo:a.A" || args[2] != EpistemicExtracted {
			t.Errorf("args[0:3] = %v; want [doc-1 code:repo:a.A %s]", args[0:3], EpistemicExtracted)
		}
		if !strings.Contains(query, "ON CONFLICT(source_id, target_id, relation)") {
			t.Errorf("query sem o ON CONFLICT do SQLite: %q", query)
		}
	})

	t.Run("arestas postgres", func(t *testing.T) {
		edgeRows := []bridgeEdgeRow{
			{TargetID: "code:repo:a.A", Status: EpistemicExtracted},
			{TargetID: "code:repo:b.B", Status: ""}, // status vazio → INFERRED
		}
		query, args := buildEdgeInsert(DialectPostgres, "repo", "doc-1", edgeRows)
		if got := strings.Count(query, "?"); got != len(edgeRows)*4 {
			t.Errorf("placeholders = %d; want %d", got, len(edgeRows)*4)
		}
		if len(args) != len(edgeRows)*4 {
			t.Errorf("args = %d; want %d", len(args), len(edgeRows)*4)
		}
		if args[2] != "repo" {
			t.Errorf("args[2] = %v; want repo", args[2])
		}
		// A 2ª linha começa em args[4]: (doc, alvo, repo, status).
		if args[7] != EpistemicInferred {
			t.Errorf("args[7] = %v; want %s (status vazio → INFERRED)", args[7], EpistemicInferred)
		}
		bound := DialectPostgres.Bind(query)
		if strings.Contains(bound, "?") {
			t.Errorf("a query convertida ainda tem `?`: %q", bound)
		}
		if !strings.Contains(bound, "ON CONFLICT (repository, source_id, target_id, relation)") {
			t.Errorf("query sem o ON CONFLICT do Postgres: %q", bound)
		}
		if !strings.Contains(bound, "'"+BridgeRelation+"'") {
			t.Errorf("o literal da relação foi alterado pelo Bind: %q", bound)
		}
	})
}

// assertBridgeStateEqual compara o grafo dos dois bancos (nós e arestas).
func assertBridgeStateEqual(t *testing.T, a, b *sql.DB) {
	t.Helper()

	nodesA := bridgeState(t, a, `SELECT id, type, name FROM graph_nodes ORDER BY id ASC`)
	nodesB := bridgeState(t, b, `SELECT id, type, name FROM graph_nodes ORDER BY id ASC`)
	if len(nodesA) != len(nodesB) {
		t.Fatalf("nós: %d vs %d", len(nodesA), len(nodesB))
	}
	for i := range nodesA {
		if nodesA[i] != nodesB[i] {
			t.Errorf("nó %d: %q vs %q", i, nodesA[i], nodesB[i])
		}
	}

	edgesA := bridgeState(t, a, `SELECT source_id, target_id, relation, epistemic_status, weight FROM graph_edges ORDER BY source_id, target_id, relation`)
	edgesB := bridgeState(t, b, `SELECT source_id, target_id, relation, epistemic_status, weight FROM graph_edges ORDER BY source_id, target_id, relation`)
	if len(edgesA) != len(edgesB) {
		t.Fatalf("arestas: %d vs %d", len(edgesA), len(edgesB))
	}
	for i := range edgesA {
		if edgesA[i] != edgesB[i] {
			t.Errorf("aresta %d: %q vs %q", i, edgesA[i], edgesB[i])
		}
	}
}

// bridgeState roda a query e devolve as linhas como strings.
func bridgeState(t *testing.T, database *sql.DB, query string) []string {
	t.Helper()

	rows, err := database.Query(query)
	if err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		t.Fatalf("columns: %v", err)
	}
	var out []string
	for rows.Next() {
		values := make([]any, len(cols))
		holders := make([]any, len(cols))
		for i := range values {
			holders[i] = &values[i]
		}
		if err := rows.Scan(holders...); err != nil {
			t.Fatalf("scan: %v", err)
		}
		parts := make([]string, len(cols))
		for i, v := range values {
			parts[i] = fmt.Sprintf("%v", v)
		}
		out = append(out, strings.Join(parts, "|"))
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	return out
}
