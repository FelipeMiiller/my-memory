package main

import (
	"context"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/pgtest"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// O fixture da renderização L0 da CLI no Postgres.
//
// O corpo bruto é carregado no `chunks.content` — é ele que o nível L1
// imprime e que o nível L0 tem que omitir.
const (
	// pgL0Summary é o micro-abstract gravado em `documents.abstract`.
	pgL0Summary = "Resumo micro do fixture L0 no Postgres"

	// pgL0RawBody só existe no corpo bruto do chunk.
	pgL0RawBody = "MarcadorBrutoDoCorpoPostgres"

	// pgL0Query é a palavra buscada: casa com o chunk no full-text search.
	pgL0Query = "searchable"
)

// init captura a URL do Postgres antes que o TestMain de cmd/mem a neutralize.
//
// O init() de um pacote roda ANTES do TestMain, então é aqui que a URL do
// ambiente ainda existe. Sem esta captura, pgtest.AdminURL() devolveria "" e
// todo teste de integração pularia — inclusive o de renderização L0, que é
// exatamente onde o mutante M6 foi injetado.
//
// A captura é idempotente: o mesmo init() existe em
// cmd/mem/codeindex_postgres_test.go (mesmo pacote), e as duas chamadas só
// releem a mesma variável de ambiente.
//
// COMO RODAR:
//
//	MY_MEMORY_PG_URL=<url> MY_MEMORY_PG_TEST=1 go test -count=1 ./cmd/mem/...
func init() {
	pgtest.Capture()
}

// TestSearch_L0Postgres prova a RENDERIZAÇÃO do nível L0 da CLI no Postgres
// (AC4 da spec fix-progressive-context-test-coverage).
//
// POR QUE ESTE TESTE EXISTE: o gap PCL-01 (ISSUE-044) é que a saída L0 nunca
// foi asserida. O mutante M6 foi injetado EXATAMENTE no ramo Postgres da
// renderização (cmd/mem/main.go, ramo L0 de runSearchPostgres) — e um teste
// só no SQLite (cmd/mem/search_level_l0_test.go) NÃO mataria M6, porque os
// dois ramos são cópias independentes da mesma renderização.
//
// O QUE ESTE TESTE PROVA:
//  1. no nível L0 a saída traz o badge de categoria em MAIÚSCULO ([MEMORY]);
//  2. no nível L0 a saída traz o micro-abstract (o `abstract` do documento);
//  3. no nível L0 a saída NÃO traz o corpo bruto do chunk;
//  4. no nível L1 (CONTROLE NEGATIVO) a saída traz o corpo bruto.
//
// O banco é o DESCARTÁVEL do internal/pgtest — nunca o Postgres configurado
// do dono.
func TestSearch_L0Postgres(t *testing.T) {
	h := pgtest.New(t)

	ctx := context.Background()

	s, err := store.NewPostgresStore(h.DSN)
	if err != nil {
		t.Fatalf("erro ao conectar no Postgres descartável: %v", err)
	}
	defer s.Close()

	const repo = "repo-l0-postgres"
	const docID = "notes/nota-l0.md"

	if err := s.InsertDocumentWithMeta(ctx, repo, docID, "notes/nota-l0.md", "Nota L0",
		0, "hash-l0-postgres", "", pgL0Summary, "memory", 0, 0); err != nil {
		t.Fatalf("erro ao inserir documento no Postgres: %v", err)
	}

	chunkContent := pgL0Query + "\n\n" + pgL0RawBody + "\n"
	if err := s.InsertChunk(ctx, repo, docID+"#0", docID, chunkContent, 0, nil); err != nil {
		t.Fatalf("erro ao inserir chunk no Postgres: %v", err)
	}

	// 1. Nível L0: badge + micro-abstract, sem o corpo bruto.
	var l0Code int
	l0Out := captureStdout(t, func() {
		l0Code = runSearchPostgres(ctx, s, nil, pgL0Query, repo, "fts", 5, 60,
			store.DefaultDecayOptions(), store.SearchOptions{Level: "l0"})
	})
	if l0Code != 0 {
		t.Fatalf("runSearchPostgres no nível l0 devolveu %d; esperava 0\nsaída:\n%s", l0Code, l0Out)
	}
	if !strings.Contains(l0Out, "[MEMORY]") {
		t.Errorf("a saída L0 no Postgres não traz o badge de categoria em maiúsculo [MEMORY]:\n%s", l0Out)
	}
	if !strings.Contains(l0Out, pgL0Summary) {
		t.Errorf("a saída L0 no Postgres não traz o micro-abstract (abstract do documento):\n%s", l0Out)
	}
	if strings.Contains(l0Out, pgL0RawBody) {
		t.Errorf("a saída L0 no Postgres traz o corpo bruto do chunk, que o nível l0 não renderiza:\n%s", l0Out)
	}

	// 2. Controle negativo: o MESMO fixture no nível l1 mostra o corpo bruto.
	//    Sem esta parte, a asserção 3 acima passaria mesmo com a renderização
	//    L0 do Postgres morta.
	var l1Code int
	l1Out := captureStdout(t, func() {
		l1Code = runSearchPostgres(ctx, s, nil, pgL0Query, repo, "fts", 5, 60,
			store.DefaultDecayOptions(), store.SearchOptions{Level: "l1"})
	})
	if l1Code != 0 {
		t.Fatalf("runSearchPostgres no nível l1 devolveu %d; esperava 0\nsaída:\n%s", l1Code, l1Out)
	}
	if !strings.Contains(l1Out, pgL0RawBody) {
		t.Errorf("controle negativo falhou: no nível l1 o corpo bruto do chunk NÃO aparece:\n%s", l1Out)
	}
}
