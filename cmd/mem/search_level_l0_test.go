package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/store"
	"github.com/FelipeMiiller/my-memory/internal/watcher"
)

// O fixture da renderização L0 da CLI.
//
// A nota é pequena de propósito: `parser.ChunkText` quebra por PALAVRAS
// (200 palavras, sobreposição 30), então ela cabe em um único chunk. Sem isso
// o marcador do corpo poderia cair em outro chunk que a busca não devolve — e
// o controle negativo (o corpo bruto ausente no L0) viraria vacuamente verde.
const (
	// l0FixtureSummary é o `summary` do frontmatter: o que o indexador grava
	// em `documents.abstract` e o que a renderização L0 tem que mostrar como
	// micro-abstract.
	l0FixtureSummary = "Resumo micro do fixture L0"

	// l0FixtureRawBody só existe no CORPO da nota. É o discriminador entre
	// "a saída L0 é compacta" e "a saída L0 imprimiu o chunk inteiro".
	l0FixtureRawBody = "MarcadorBrutoDoCorpo"

	// l0FixtureQuery é a palavra buscada: está no corpo e casa com o chunk.
	l0FixtureQuery = "searchable"
)

// TestSearch_L0 prova a RENDERIZAÇÃO do nível L0 da CLI no SQLite (AC3 da
// spec fix-progressive-context-test-coverage).
//
// POR QUE ESTE TESTE EXISTE: o gap PCL-01 (ISSUE-044) é que a saída L0 nunca
// foi asserida. O teste antigo (cmd/mem/search_defaults_test.go) verifica o
// DEFAULT da flag — e o default não muda quando a renderização morre. Por
// isso o mutante M6 (desligar o ramo L0 em cmd/mem/main.go) sobreviveu.
//
// O QUE ESTE TESTE PROVA:
//  1. no nível L0 a saída traz o badge de categoria em MAIÚSCULO ([MEMORY]);
//  2. no nível L0 a saída traz o micro-abstract (o `summary` do frontmatter);
//  3. no nível L0 a saída NÃO traz o corpo bruto do chunk;
//  4. no nível L1 (CONTROLE NEGATIVO) a saída traz o corpo bruto — é o que
//     prova que a asserção 3 não é vacuamente verde.
func TestSearch_L0(t *testing.T) {
	tmpDir := t.TempDir()

	notePath := filepath.Join(tmpDir, "nota-l0.md")
	note := "---\n" +
		"title: Nota do Fixture L0\n" +
		"category: memory\n" +
		"summary: " + l0FixtureSummary + "\n" +
		"---\n\n" +
		l0FixtureQuery + "\n\n" +
		l0FixtureRawBody + "\n"
	if err := os.WriteFile(notePath, []byte(note), 0644); err != nil {
		t.Fatalf("erro ao escrever a nota de fixture: %v", err)
	}

	database, err := db.InitDB(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		if strings.Contains(err.Error(), "CGO_ENABLED=0") || strings.Contains(err.Error(), "binary was compiled without cgo") {
			t.Skip("Pulando teste SQLite: ambiente sem CGO")
		}
		t.Fatalf("erro ao iniciar banco SQLite: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	if _, err := watcher.IndexSingleFileSQLite(ctx, database, nil, nil, notePath, false); err != nil {
		t.Fatalf("erro ao indexar a nota de fixture: %v", err)
	}

	// 1. Nível L0: badge + micro-abstract, sem o corpo bruto.
	var l0Code int
	l0Out := captureStdout(t, func() {
		l0Code = runSearchSQLite(ctx, database, nil, nil, l0FixtureQuery, "fts", false, 5, 60,
			store.DefaultDecayOptions(), store.SearchOptions{Level: "l0"})
	})
	if l0Code != 0 {
		t.Fatalf("runSearchSQLite no nível l0 devolveu %d; esperava 0\nsaída:\n%s", l0Code, l0Out)
	}
	if !strings.Contains(l0Out, "[MEMORY]") {
		t.Errorf("a saída L0 não traz o badge de categoria em maiúsculo [MEMORY]:\n%s", l0Out)
	}
	if !strings.Contains(l0Out, l0FixtureSummary) {
		t.Errorf("a saída L0 não traz o micro-abstract (summary do frontmatter):\n%s", l0Out)
	}
	if strings.Contains(l0Out, l0FixtureRawBody) {
		t.Errorf("a saída L0 traz o corpo bruto do chunk, que o nível l0 não renderiza:\n%s", l0Out)
	}

	// 2. Controle negativo: o MESMO fixture no nível l1 mostra o corpo bruto.
	//    Sem esta parte, a asserção 3 acima passaria mesmo com a renderização
	//    L0 morta.
	var l1Code int
	l1Out := captureStdout(t, func() {
		l1Code = runSearchSQLite(ctx, database, nil, nil, l0FixtureQuery, "fts", false, 5, 60,
			store.DefaultDecayOptions(), store.SearchOptions{Level: "l1"})
	})
	if l1Code != 0 {
		t.Fatalf("runSearchSQLite no nível l1 devolveu %d; esperava 0\nsaída:\n%s", l1Code, l1Out)
	}
	if !strings.Contains(l1Out, l0FixtureRawBody) {
		t.Errorf("controle negativo falhou: no nível l1 o corpo bruto do chunk NÃO aparece:\n%s", l1Out)
	}
}
