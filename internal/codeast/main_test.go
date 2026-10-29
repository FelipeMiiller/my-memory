package codeast

import (
	"os"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/pgtest"
)

// TestMain isola o storage global do pacote de testes do internal/codeast.
//
// É o mesmo guard de `internal/db/main_test.go` (e de `cmd/mem/main_test.go`):
//
//  1. capturar a URL real ANTES de neutralizar — é o que permite ao teste de
//     integração opt-in (`internal/codeast/batch_postgres_test.go`) chegar ao
//     servidor para criar o banco descartável;
//  2. neutralizar as variáveis de storage;
//  3. restaurá-las no fim.
//
// SEM este guard, um teste de internal/codeast que reachasse o Postgres
// escreveria no banco real do dono: MY_MEMORY_PG_URL tem precedência sobre
// qualquer config, então apontar o config global para um temp dir não basta —
// a variável de ambiente vence (a INCIDENTE registrada na ISSUE-031).
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
