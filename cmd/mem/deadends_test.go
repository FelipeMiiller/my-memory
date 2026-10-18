package main

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/db"
)

// newDeadendFixture monta um grafo pequeno e legível:
//
//	cadeia1.md -> cadeia2.md     (cabeça: saída sem entrada = deadend)
//	meio.md -> fim.md            (outro deadend)
//	sozinho.md                   (sem aresta nenhuma = órfão, NÃO é deadend)
func newDeadendFixture(t *testing.T) string {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "vault_fixture.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	for _, path := range []string{"cadeia1.md", "cadeia2.md", "meio.md", "fim.md", "sozinho.md"} {
		if err := db.InsertDocumentWithMeta(ctx, database, path, path, path, 1700000000, "hash-"+path, "epoch", "", "resource", 1700000000, 10); err != nil {
			t.Fatalf("InsertDocumentWithMeta(%s) falhou: %v", path, err)
		}
	}
	for _, edge := range [][2]string{{"cadeia1.md", "cadeia2.md"}, {"meio.md", "fim.md"}} {
		if err := db.InsertEdgeWithProps(ctx, database, edge[0], edge[1], "links_to", "EXTRACTED", 1.0); err != nil {
			t.Fatalf("InsertEdgeWithProps(%s -> %s) falhou: %v", edge[0], edge[1], err)
		}
	}
	return dbPath
}

func runDeadendsFixture(t *testing.T, dbPath string) ([]string, error) {
	t.Helper()

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--storage", "sqlite"}
	err := runDeadendsCommand(context.Background(), "fixture-repo", args, &buf)

	var paths []string
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			paths = append(paths, trimmed)
		}
	}
	return paths, err
}

// TestRunDeadendsCommand_ListaFolhasDoGrafo cobre a cadeia de dois documentos
// da spec: só a cabeça é deadend, a cauda tem entrada e o nó do meio também.
func TestRunDeadendsCommand_ListaFolhasDoGrafo(t *testing.T) {
	paths, err := runDeadendsFixture(t, newDeadendFixture(t))
	if err != nil {
		t.Fatalf("runDeadendsCommand falhou: %v", err)
	}

	want := []string{"cadeia1.md", "meio.md"}
	if len(paths) != len(want) {
		t.Fatalf("esperava %v, obteve %v", want, paths)
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Errorf("posição %d: esperava %q, obteve %q", i, want[i], paths[i])
		}
	}
}

// TestRunDeadendsCommand_NotaIsoladaNaoEhDeadend é a armadilha da definição:
// "sem arestas de entrada" sozinho inclui toda nota solta do vault, e o
// resultado viria inflado. Deadend exige saída E ausência de entrada.
func TestRunDeadendsCommand_NotaIsoladaNaoEhDeadend(t *testing.T) {
	paths, err := runDeadendsFixture(t, newDeadendFixture(t))
	if err != nil {
		t.Fatalf("runDeadendsCommand falhou: %v", err)
	}

	for _, path := range paths {
		if path == "sozinho.md" {
			t.Errorf("nota isolada não é deadend, mas apareceu na lista: %v", paths)
		}
		if path == "cadeia2.md" || path == "fim.md" {
			t.Errorf("nota com aresta de entrada não é deadend: %v", paths)
		}
	}
}

func TestRunDeadendsCommand_GrafoVazioNaoErra(t *testing.T) {
	paths, err := runDeadendsFixture(t, newVaultFixture(t, "docs/alpha.md"))
	if err != nil {
		t.Fatalf("grafo sem arestas deveria sair zero, obteve erro: %v", err)
	}
	if len(paths) != 0 {
		t.Errorf("grafo sem arestas não tem deadend, obteve: %v", paths)
	}
}
