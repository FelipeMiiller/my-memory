package mcp

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/db"
)

// seedCodeGraph popula dois repositórios no mesmo banco e devolve os ids dos
// símbolos usados pelos testes de filtro.
func seedCodeGraph(t *testing.T) (database *sql.DB, alphaID, betaID, fileBID int64) {
	t.Helper()

	database, err := db.InitDB(filepath.Join(t.TempDir(), "code_filter.db"))
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if err := db.EnsureCodeTables(context.Background(), database); err != nil {
		t.Fatalf("EnsureCodeTables falhou: %v", err)
	}

	ctxA := db.WithRepository(context.Background(), "repo-a")
	ctxB := db.WithRepository(context.Background(), "repo-b")

	fileA, err := db.UpsertCodeFile(ctxA, database, "a/alpha.go", "go", "h-a", "", 10, 1)
	if err != nil {
		t.Fatalf("UpsertCodeFile(repo-a) falhou: %v", err)
	}
	alphaID, err = db.UpsertCodeSymbol(ctxA, database, fileA, "function", "alpha", "repoa.alpha", "", "", 1, 2, 0, 0)
	if err != nil {
		t.Fatalf("UpsertCodeSymbol(repo-a) falhou: %v", err)
	}

	fileB, err := db.UpsertCodeFile(ctxB, database, "b/beta.go", "go", "h-b", "", 10, 1)
	if err != nil {
		t.Fatalf("UpsertCodeFile(repo-b) falhou: %v", err)
	}
	betaID, err = db.UpsertCodeSymbol(ctxB, database, fileB, "function", "beta", "repob.beta", "", "", 1, 2, 0, 0)
	if err != nil {
		t.Fatalf("UpsertCodeSymbol(repo-b) falhou: %v", err)
	}

	return database, alphaID, betaID, fileB
}

// TestFiltroPorRepositoryNaBusca cobre CG-07: a busca de código com filtro
// de repositório não devolve símbolos de outro repositório.
func TestFiltroPorRepositoryNaBusca(t *testing.T) {
	database, _, _, _ := seedCodeGraph(t)
	ctx := context.Background()

	hits, err := SQLCodeSearchScoped(ctx, database, "alpha", "", "", 10, false, "repo-a")
	if err != nil {
		t.Fatalf("SQLCodeSearchScoped falhou: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("hits(repo-a) = %d; want 1 (%+v)", len(hits), hits)
	}
	if hits[0].QualifiedName != "repoa.alpha" {
		t.Errorf("qualified_name = %q; want repoa.alpha", hits[0].QualifiedName)
	}

	// O filtro de repo-b devolve o outro símbolo — e nada mais.
	hitsB, err := SQLCodeSearchScoped(ctx, database, "alpha", "", "", 10, false, "repo-b")
	if err != nil {
		t.Fatalf("SQLCodeSearchScoped(repo-b) falhou: %v", err)
	}
	if len(hitsB) != 0 {
		t.Fatalf("hits(repo-b) = %d; want 0 (%+v)", len(hitsB), hitsB)
	}

	// Sem filtro, a busca legacy continua devolvendo os dois.
	hitsAll, err := SQLCodeSearch(ctx, database, "alpha", "", "", 10, false)
	if err != nil {
		t.Fatalf("SQLCodeSearch falhou: %v", err)
	}
	if len(hitsAll) != 1 {
		t.Fatalf("hits(sem filtro) = %d; want 1", len(hitsAll))
	}
}

// TestFiltroPorRepositoryNaTravessia cobre CG-07 na travessia: as arestas de
// outro repositório não aparecem quando o filtro está ativo.
func TestFiltroPorRepositoryNaTravessia(t *testing.T) {
	database, alphaID, betaID, fileBID := seedCodeGraph(t)
	ctx := context.Background()

	// Aresta gravada no contexto de repo-b: liga o símbolo de repo-a ao de repo-b.
	ctxB := db.WithRepository(context.Background(), "repo-b")
	if err := db.UpsertCodeEdge(ctxB, database, alphaID, betaID, fileBID, "references", 1, 2, 1.0); err != nil {
		t.Fatalf("UpsertCodeEdge falhou: %v", err)
	}

	// Com filtro repo-a, a aresta de repo-b fica de fora.
	scoped, _, err := SQLCodeNeighborsScoped(ctx, database, "repoa.alpha", 1, "out", "repo-a")
	if err != nil {
		t.Fatalf("SQLCodeNeighborsScoped falhou: %v", err)
	}
	if len(scoped) != 0 {
		t.Fatalf("neighbors(repo-a) = %d; want 0 (%+v)", len(scoped), scoped)
	}

	// Sem filtro, a aresta aparece — o filtro é que a exclui.
	unscoped, _, err := SQLCodeNeighbors(ctx, database, "repoa.alpha", 1, "out")
	if err != nil {
		t.Fatalf("SQLCodeNeighbors falhou: %v", err)
	}
	if len(unscoped) != 1 {
		t.Fatalf("neighbors(sem filtro) = %d; want 1 (%+v)", len(unscoped), unscoped)
	}

	// A raiz também respeita o filtro: repo-b não acha o símbolo de repo-a.
	if _, _, err := SQLCodeNeighborsScoped(ctx, database, "repoa.alpha", 1, "out", "repo-b"); err == nil {
		t.Error("raiz de repo-a não deveria ser resolvida no filtro de repo-b")
	}
}
