package codeast

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/pgtest"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// A integração PostgreSQL do LOTE do code stage.
//
// POR QUE ELA EXISTE: o dono mediu no Neon (≈268 ms por round-trip) que o
// bucket `code_edges_uncertain` era o último gargalo: 1 statement por aresta
// incerta → ~1,6 aresta/s → ≈110 min para popular o repo inteiro. O lote
// tem de ser provado no SERVIDOR (o `ON CONFLICT DO UPDATE` do Postgres
// rejeita o conflito da mesma linha duas vezes no mesmo statement) e o NÚMERO
// de statements por arquivo tem de ser contado no SQL que chega ao servidor
// (o `pgtest.Recorder` grava exatamente esse SQL).
//
// SEGURANÇA: o banco é DESCARTÁVEL (`pgtest.New` cria e derruba
// `mymemory_pgtest_*`). O Postgres do dono nunca é alvo.
//
// COMO RODAR:
//
//	MY_MEMORY_PG_URL=<url> MY_MEMORY_PG_TEST=1 go test -count=1 ./internal/codeast/...
//
// Sem MY_MEMORY_PG_TEST=1 o teste é pulado (a suíte padrão é hermética).

// codeastCountInserts conta quantos INSERTs o recorder viu para uma tabela.
//
// O " (" no fim do nome da tabela é o que separa `code_edges` de
// `code_edges_uncertain` (o nome da primeira é prefixo do da segunda).
func codeastCountInserts(rec *pgtest.Recorder, table string) int {
	count := 0
	for _, q := range rec.Queries() {
		if strings.Contains(q, "INSERT INTO "+table+" (") {
			count++
		}
	}
	return count
}

// TestLotePostgresFlushEdgesGravaUmStatementPorArquivo prova (EDG-04,
// UNC-02) que o `FlushEdges` grava UM statement por arquivo em CADA bucket:
// as arestas resolvidas (`code_edges`) e as incertas (`code_edges_uncertain`).
//
// São 2 arquivos, cada um com 1 aresta resolvida e 1 incerta → 2 + 2
// statements. No caminho antigo (1 statement por aresta) seriam 4 + 4.
func TestLotePostgresFlushEdgesGravaUmStatementPorArquivo(t *testing.T) {
	h := pgtest.New(t)

	pgStore, err := store.NewPostgresStore(h.DSN)
	if err != nil {
		t.Fatalf("NewPostgresStore(%s): %v", h.DBName, err)
	}
	// A conexão do store aplica o schema no banco descartável.
	t.Cleanup(func() { pgStore.Close() })

	rec := pgtest.NewRecorder()
	recDB, err := rec.OpenDB(h.DSN)
	if err != nil {
		t.Fatalf("conexão instrumentada: %v", err)
	}
	t.Cleanup(func() { recDB.Close() })

	ctx := db.WithRepository(context.Background(), "lote-repo")
	if err := db.EnsureCodeTables(ctx, recDB); err != nil {
		t.Fatalf("EnsureCodeTables: %v", err)
	}

	// 2 arquivos, cada um com um símbolo ORIGEM.
	var srcIDs, fileIDs []int64
	for i := 0; i < 2; i++ {
		fileID, err := db.UpsertCodeFile(ctx, recDB, fmt.Sprintf("pkg/file%d.go", i), "go",
			fmt.Sprintf("hash-%d", i), fmt.Sprintf("ast-%d", i), 100, 1700000000)
		if err != nil {
			t.Fatalf("UpsertCodeFile: %v", err)
		}
		srcID, err := db.UpsertCodeSymbol(ctx, recDB, fileID, "function",
			fmt.Sprintf("Caller%d", i), fmt.Sprintf("pkg.Caller%d", i), "func()", "", 1, 3, 0, 0)
		if err != nil {
			t.Fatalf("UpsertCodeSymbol: %v", err)
		}
		srcIDs = append(srcIDs, srcID)
		fileIDs = append(fileIDs, fileID)
	}
	// O destino das arestas resolvidas (nível 1: qualified name único).
	if _, err := db.UpsertCodeSymbol(ctx, recDB, fileIDs[0], "function", "Target", "pkg.Target",
		"func()", "", 1, 3, 0, 0); err != nil {
		t.Fatalf("UpsertCodeSymbol(destino): %v", err)
	}

	// A partir daqui só conta o que o FlushEdges faz.
	rec.Reset()

	fp := NewFileProcessor(recDB, NewCodeFileCache(recDB), nil)
	fp.pending = []pendingEdge{
		// arquivo 0: uma resolvida (nível 1) e DUAS incertas (nível 3).
		{kind: EdgeCalls, srcSymbolID: srcIDs[0], dstQualifiedName: "pkg.Target", dstName: "Target",
			fileID: fileIDs[0], line: 10},
		{kind: EdgeCalls, srcSymbolID: srcIDs[0], dstQualifiedName: "pkg.Missing", dstName: "Missing",
			fileID: fileIDs[0], line: 20},
		{kind: EdgeCalls, srcSymbolID: srcIDs[0], dstQualifiedName: "pkg.Absent", dstName: "Absent",
			fileID: fileIDs[0], line: 25},
		// arquivo 1: uma resolvida e DUAS incertas.
		{kind: EdgeCalls, srcSymbolID: srcIDs[1], dstQualifiedName: "pkg.Target", dstName: "Target",
			fileID: fileIDs[1], line: 30},
		{kind: EdgeCalls, srcSymbolID: srcIDs[1], dstQualifiedName: "pkg.Missing", dstName: "Missing",
			fileID: fileIDs[1], line: 40},
		{kind: EdgeCalls, srcSymbolID: srcIDs[1], dstQualifiedName: "pkg.Absent", dstName: "Absent",
			fileID: fileIDs[1], line: 45},
	}

	if err := fp.FlushEdges(ctx); err != nil {
		t.Fatalf("FlushEdges: %v", err)
	}

	if got := codeastCountInserts(rec, "code_edges"); got != 2 {
		t.Errorf("INSERTs em code_edges = %d; want 2 (um statement por arquivo)\n%v", got, rec.Queries())
	}
	if got := codeastCountInserts(rec, "code_edges_uncertain"); got != 2 {
		t.Errorf("INSERTs em code_edges_uncertain = %d; want 2 (um statement por arquivo)\n%v",
			got, rec.Queries())
	}

	// O estado final: 2 arestas resolvidas e 4 incertas.
	var edges, uncertain int
	if err := recDB.QueryRow(`SELECT COUNT(*) FROM code_edges`).Scan(&edges); err != nil {
		t.Fatalf("COUNT code_edges: %v", err)
	}
	if err := recDB.QueryRow(`SELECT COUNT(*) FROM code_edges_uncertain`).Scan(&uncertain); err != nil {
		t.Fatalf("COUNT code_edges_uncertain: %v", err)
	}
	if edges != 2 || uncertain != 4 {
		t.Errorf("grafo = %d arestas / %d incertas; want 2 / 4", edges, uncertain)
	}

	// Idempotência: rodar de novo não duplica e continua custando o mesmo.
	rec.Reset()
	fp.pending = []pendingEdge{
		{kind: EdgeCalls, srcSymbolID: srcIDs[0], dstQualifiedName: "pkg.Target", dstName: "Target",
			fileID: fileIDs[0], line: 10},
		{kind: EdgeCalls, srcSymbolID: srcIDs[0], dstQualifiedName: "pkg.Missing", dstName: "Missing",
			fileID: fileIDs[0], line: 20},
		{kind: EdgeCalls, srcSymbolID: srcIDs[0], dstQualifiedName: "pkg.Absent", dstName: "Absent",
			fileID: fileIDs[0], line: 25},
		{kind: EdgeCalls, srcSymbolID: srcIDs[1], dstQualifiedName: "pkg.Target", dstName: "Target",
			fileID: fileIDs[1], line: 30},
		{kind: EdgeCalls, srcSymbolID: srcIDs[1], dstQualifiedName: "pkg.Missing", dstName: "Missing",
			fileID: fileIDs[1], line: 40},
		{kind: EdgeCalls, srcSymbolID: srcIDs[1], dstQualifiedName: "pkg.Absent", dstName: "Absent",
			fileID: fileIDs[1], line: 45},
	}
	if err := fp.FlushEdges(ctx); err != nil {
		t.Fatalf("FlushEdges (2a vez): %v", err)
	}
	if got := codeastCountInserts(rec, "code_edges"); got != 2 {
		t.Errorf("INSERTs em code_edges (2a vez) = %d; want 2", got)
	}
	if got := codeastCountInserts(rec, "code_edges_uncertain"); got != 2 {
		t.Errorf("INSERTs em code_edges_uncertain (2a vez) = %d; want 2", got)
	}
	if err := recDB.QueryRow(`SELECT COUNT(*) FROM code_edges`).Scan(&edges); err != nil {
		t.Fatalf("COUNT code_edges: %v", err)
	}
	if err := recDB.QueryRow(`SELECT COUNT(*) FROM code_edges_uncertain`).Scan(&uncertain); err != nil {
		t.Fatalf("COUNT code_edges_uncertain: %v", err)
	}
	if edges != 2 || uncertain != 4 {
		t.Errorf("após 2 passes: %d arestas / %d incertas; want 2 / 4", edges, uncertain)
	}
}
