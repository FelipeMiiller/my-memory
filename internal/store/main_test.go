package store

import (
	"os"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/pgtest"
)

// TestMain isola o storage global do pacote de testes do internal/store.
//
// SEM este guard, os testes de integração PostgreSQL deste pacote escreviam
// no banco REAL do dono: eles liam MY_MEMORY_PG_URL do ambiente e conectavam
// direto nele, criando documentos, chunks, tags e arestas de teste no banco
// de produção. Foi assim que o banco do dono acumulou 80 repositórios e 362
// documentos de lixo (medido na ISSUE-031).
//
// O padrão é o de cmd/mem/main_test.go e internal/config/main_test.go:
//
//  1. capturar a URL real ANTES de neutralizar — é o que permite ao teste de
//     integração opt-in chegar ao servidor para criar o banco descartável;
//  2. neutralizar as variáveis de storage;
//  3. restaurá-las no fim.
//
// A URL capturada só é usada por teste que pede o Postgres explicitamente
// (MY_MEMORY_PG_TEST=1) e SEMPRE contra um banco descartável criado pelo
// pgtest.New. O banco `mymemory` do dono nunca é alvo.
func TestMain(m *testing.M) {
	pgtest.Capture()
	restore := pgtest.Neutralize()

	code := m.Run()

	restore()
	os.Exit(code)
}
