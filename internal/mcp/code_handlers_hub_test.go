package mcp

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/db"
)

// chainGraphFixture monta um subgrafo PEQUENO (3 nós) em cadeia:
// alpha → beta → gamma. O nó do meio (beta) é o de MAIOR grau e também o
// vizinho de hop 1 — é o caso que o corte por hub eliminava (defeito D4).
func chainGraphFixture(t *testing.T) *sql.DB {
	t.Helper()

	database, err := db.InitDB(filepath.Join(t.TempDir(), "chain.db"))
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	ctx := context.Background()
	fileID, err := db.UpsertCodeFile(ctx, database, "pkg/chain.go", "go", "hash", "ast", 64, 1700000000)
	if err != nil {
		t.Fatalf("UpsertCodeFile: %v", err)
	}
	upsert := func(name, qn string) int64 {
		t.Helper()
		id, err := db.UpsertCodeSymbol(ctx, database, fileID, "function", name, qn,
			"func "+name+"()", "", 1, 3, 0, 0)
		if err != nil {
			t.Fatalf("UpsertCodeSymbol(%s): %v", qn, err)
		}
		return id
	}

	alphaID := upsert("Alpha", "alpha.Alpha")
	betaID := upsert("Beta", "beta.Beta")
	gammaID := upsert("Gamma", "gamma.Gamma")

	if err := db.UpsertCodeEdge(ctx, database, alphaID, betaID, fileID, "calls", 10, 10, 1.0); err != nil {
		t.Fatalf("UpsertCodeEdge(alpha->beta): %v", err)
	}
	if err := db.UpsertCodeEdge(ctx, database, betaID, gammaID, fileID, "calls", 20, 20, 1.0); err != nil {
		t.Fatalf("UpsertCodeEdge(beta->gamma): %v", err)
	}
	return database
}

// TestHubThresholdNaoPodaSubgrafoPequeno prova o defeito D4: em um subgrafo
// pequeno (3 nós), o corte por hub NÃO pode rodar.
//
// Antes: o p99 sobre 3 nós caía no penúltimo grau e removia beta.Beta (o
// `calls` de hop 1, o vizinho mais próximo da semente), mantendo
// gamma.Gamma (hop 2) — a relevância ficava invertida.
func TestHubThresholdNaoPodaSubgrafoPequeno(t *testing.T) {
	database := chainGraphFixture(t)
	ctx := context.Background()

	t.Setenv(HubThresholdPercentileEnv, "99")
	resp, err := SQLCodeNeighborsTraverse(ctx, database, "alpha.Alpha", 2, "out", "", 0)
	if err != nil {
		t.Fatalf("SQLCodeNeighborsTraverse: %v", err)
	}

	if resp.NodesPruned != 0 {
		t.Errorf("nodes_pruned = %d; want 0 (subgrafo pequeno não pode ser podado)", resp.NodesPruned)
	}
	if resp.HubThreshold != 0 {
		t.Errorf("hub_threshold = %d; want 0 (nenhum corte foi aplicado)", resp.HubThreshold)
	}

	symbols := neighborSymbols(resp.Neighbors)
	if !symbols["beta.Beta"] {
		t.Errorf("o vizinho de hop 1 (beta.Beta) não pode ser removido: %+v", resp.Neighbors)
	}
	if !symbols["gamma.Gamma"] {
		t.Errorf("o vizinho de hop 2 (gamma.Gamma) sumiu: %+v", resp.Neighbors)
	}
	if len(resp.Neighbors) != 2 {
		t.Errorf("vizinhos = %d; want 2 (o subgrafo inteiro)", len(resp.Neighbors))
	}

	// A ordem por hop é preservada: o vizinho mais próximo vem primeiro.
	if resp.Neighbors[0].Symbol != "beta.Beta" {
		t.Errorf("o primeiro vizinho deveria ser beta.Beta (hop 1), obteve %q", resp.Neighbors[0].Symbol)
	}
	if resp.Neighbors[0].Hop != 1 || resp.Neighbors[1].Hop != 2 {
		t.Errorf("a ordem por hop não foi preservada: %+v", resp.Neighbors)
	}
}

// TestHubThresholdPodaOMaiorGrauEmSubgrafoGrande prova o outro lado do
// defeito D4: acima do mínimo de nós (MinHubPruneNodes), o corte volta a
// valer E o nó removido é o de MAIOR grau.
func TestHubThresholdPodaOMaiorGrauEmSubgrafoGrande(t *testing.T) {
	database, leaves := hubGraphFixture(t)
	defer database.Close()
	ctx := context.Background()

	t.Setenv(HubThresholdPercentileEnv, "99")
	resp, err := SQLCodeNeighborsTraverse(ctx, database, "seed.Seed", 2, "out", "", 0)
	if err != nil {
		t.Fatalf("SQLCodeNeighborsTraverse: %v", err)
	}

	if resp.NodesPruned == 0 {
		t.Fatalf("nenhum nó foi removido pelo hub threshold: %+v", resp)
	}

	// O nó de MAIOR grau é o hub (grau = 1 + leaves); ele tem que ter sido
	// o removido (e não o penúltimo por erro de índice).
	symbols := neighborSymbols(resp.Neighbors)
	if symbols["hub.Hub"] {
		t.Errorf("o nó de maior grau (hub.Hub) devia ter sido podado: %+v", resp.Neighbors)
	}
	if len(resp.Neighbors) != leaves {
		t.Errorf("vizinhos = %d; want %d (as folhas sobrevivem)", len(resp.Neighbors), leaves)
	}

	// A ordem por hop é preservada depois da poda.
	for i := 1; i < len(resp.Neighbors); i++ {
		if resp.Neighbors[i].Hop < resp.Neighbors[i-1].Hop {
			t.Fatalf("a ordem não é por hop: %d depois de %d",
				resp.Neighbors[i].Hop, resp.Neighbors[i-1].Hop)
		}
	}
}

// TestApplyHubThresholdMinimoDeNos fixa o número mínimo de nós documentado.
func TestApplyHubThresholdMinimoDeNos(t *testing.T) {
	if MinHubPruneNodes != 50 {
		t.Fatalf("MinHubPruneNodes = %d; want 50 (o número é documentado no código e na doc da tool)", MinHubPruneNodes)
	}
}
