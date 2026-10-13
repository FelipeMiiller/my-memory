package config

import (
	"os"
	"testing"
)

// envStorageVars são lidas por resolveStorageAndRepo antes de qualquer config.
var envStorageVars = []string{
	"MY_MEMORY_PG_URL",
	"MY_MEMORY_PG_PASSWORD",
	"POSTGRES_URL",
	"DATABASE_URL",
}

// TestMain isola o config global de todo o pacote de testes.
//
// Sem isso, LoadCascadingConfig lia o ~/.memory/config.yaml REAL do
// desenvolvedor. Isso ficava mascarado enquanto o sqlite_path global era
// relativo ("memory.db", que por acaso era o default esperado pelo teste) e
// passou a falhar de verdade quando o path virou absoluto — um teste que espera
// o default do CWD recebia C:\Users\<user>\.memory\memory.db.
//
// Mesma classe de bug do ISSUE-012, agora no sentido inverso: o ambiente real
// do usuário vazando para dentro dos testes.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "mem-config-test-global")
	if err != nil {
		panic("nao foi possivel criar diretorio temporario para isolar o config global: " + err.Error())
	}

	if err := os.Setenv(GlobalConfigDirEnv, dir); err != nil {
		panic("nao foi possivel definir " + GlobalConfigDirEnv + ": " + err.Error())
	}

	saved := map[string]string{}
	for _, k := range envStorageVars {
		if v, ok := os.LookupEnv(k); ok {
			saved[k] = v
		}
		if err := os.Unsetenv(k); err != nil {
			panic("nao foi possivel limpar " + k + ": " + err.Error())
		}
	}

	code := m.Run()

	for k, v := range saved {
		_ = os.Setenv(k, v)
	}
	os.RemoveAll(dir)
	os.Exit(code)
}
