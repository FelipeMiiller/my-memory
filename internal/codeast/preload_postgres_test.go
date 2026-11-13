package codeast

import (
	"context"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/pgtest"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// codeastCountSelects conta quantos SELECTs sobre `code_symbols` o recorder viu.
func codeastCountSelects(rec *pgtest.Recorder) int {
	count := 0
	for _, q := range rec.Queries() {
		if strings.Contains(q, "FROM code_symbols") {
			count++
		}
	}
	return count
}

// TestPreloadPostgresReduzAsQueriesDaResolucao prova, no SERVIDOR, que a
// resolução das arestas deixa de custar UMA query por destino.
//
// No Neon (~268 ms/round-trip) a primeira população do code graph pagava ~4 500
// queries só na resolução (~20 min). Com o preload são ~10.
//
// SEGURANÇA: banco DESCARTÁVEL (`pgtest.New` cria e derruba
// `mymemory_pgtest_*`). O Postgres do dono nunca é alvo.
//
// COMO RODAR:
//
//	MY_MEMORY_PG_URL=<url> MY_MEMORY_PG_TEST=1 go test -count=1 ./internal/codeast/...
func TestPreloadPostgresReduzAsQueriesDaResolucao(t *testing.T) {
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

	ctx := db.WithRepository(context.Background(), "preload-repo")
	if err := db.EnsureCodeTables(ctx, recDB); err != nil {
		t.Fatalf("EnsureCodeTables: %v", err)
	}

	// Um arquivo com 10 símbolos de destino, cada um com qualified name único.
	fileID, err := db.UpsertCodeFile(ctx, recDB, "pkg/alvo.go", "go", "hash", "ast", 10, 1)
	if err != nil {
		t.Fatalf("UpsertCodeFile: %v", err)
	}
	const destinos = 10
	pending := make([]pendingEdge, 0, destinos)
	for i := 0; i < destinos; i++ {
		nome := "Alvo" + string(rune('A'+i))
		if _, err := db.UpsertCodeSymbol(ctx, recDB, fileID, "function", nome, "pkg."+nome, "func()", "", 1, 2, 0, 0); err != nil {
			t.Fatalf("UpsertCodeSymbol(%s): %v", nome, err)
		}
		pending = append(pending, pendingEdge{
			kind: EdgeCalls, srcSymbolID: 1, dstName: nome, dstQualifiedName: "pkg." + nome,
			fileID: fileID, line: 10 + i,
		})
	}

	fp := NewFileProcessor(recDB, NewCodeFileCache(recDB), nil)

	// (1) O caminho por destino: uma query por destino.
	rec.Reset()
	resolverSemPreload := newEdgeResolver(ctx, recDB)
	for _, pe := range pending {
		if _, err := fp.resolveEdgeRow(ctx, pe, resolverSemPreload); err != nil {
			t.Fatalf("resolveEdgeRow (sem preload): %v", err)
		}
	}
	queriesSemPreload := codeastCountSelects(rec)

	// (2) Com o preload: um punhado de queries, e nenhuma na resolução.
	rec.Reset()
	resolverComPreload := newEdgeResolver(ctx, recDB)
	if err := resolverComPreload.preload(pending); err != nil {
		t.Fatalf("preload: %v", err)
	}
	queriesDoPreload := codeastCountSelects(rec)
	for _, pe := range pending {
		if _, err := fp.resolveEdgeRow(ctx, pe, resolverComPreload); err != nil {
			t.Fatalf("resolveEdgeRow (com preload): %v", err)
		}
	}
	queriesDuranteAResolucao := codeastCountSelects(rec) - queriesDoPreload

	if queriesSemPreload < destinos {
		t.Errorf("queries no caminho por destino = %d; want >= %d (um por destino)\n%v",
			queriesSemPreload, destinos, rec.Queries())
	}
	if queriesDoPreload > 2 {
		t.Errorf("queries do preload = %d; want <= 2 (um punhado)\n%v", queriesDoPreload, rec.Queries())
	}
	if queriesDuranteAResolucao != 0 {
		t.Errorf("queries DURANTE a resolução com o preload = %d; want 0 (o cache já está cheio)\n%v",
			queriesDuranteAResolucao, rec.Queries())
	}
}
