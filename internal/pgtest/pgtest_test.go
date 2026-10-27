package pgtest

import (
	"os"
	"testing"
)

// TestNeutralizeRemoveEDepoisRestauraAsVariaveisDeStorage prova o contrato
// central da trava: enquanto a suíte roda, as variáveis que fazem o store
// rotearem para o Postgres do dono não estão no ambiente — e depois da suíte
// voltam ao valor original.
func TestNeutralizeRemoveEDepoisRestauraAsVariaveisDeStorage(t *testing.T) {
	const marker = "valor-de-teste"
	for _, k := range StorageEnvVars {
		t.Setenv(k, marker)
	}

	restore := Neutralize()

	for _, k := range StorageEnvVars {
		if v, ok := os.LookupEnv(k); ok {
			t.Errorf("%s continua no ambiente (%q) depois de Neutralize", k, v)
		}
	}

	restore()

	for _, k := range StorageEnvVars {
		if v := os.Getenv(k); v != marker {
			t.Errorf("restore não devolveu %s: got %q, want %q", k, v, marker)
		}
	}
}

// TestAdminURLSoExisteComOptInECaptura prova que a integração só fica
// habilitada com MY_MEMORY_PG_TEST=1 — a suíte padrão é hermética.
func TestAdminURLSoExisteComOptInECaptura(t *testing.T) {
	const url = "postgres://user:pass@host:5432/db?sslmode=disable"

	t.Setenv("MY_MEMORY_PG_URL", url)
	t.Setenv(OptInEnvVar, "")
	Capture()
	if got := AdminURL(); got != "" {
		t.Errorf("AdminURL() = %q; want %q (sem %s=1 a integração fica desligada)", got, "", OptInEnvVar)
	}

	t.Setenv(OptInEnvVar, "1")
	if got := AdminURL(); got != url {
		t.Errorf("AdminURL() = %q; want %q", got, url)
	}
}
