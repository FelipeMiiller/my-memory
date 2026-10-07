package main

import (
	"os"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/config"
)

// TestMain isola o config global de TODO o pacote de testes.
//
// Sem isso, testes que chamam runInit() acabavam registrando repositórios em
// ~/.memory/config.yaml — o config real do usuário. Cada `go test` local
// adicionava entradas `TestRunInit*` e o arquivo chegou a acumular ~160
// registros (ver ISSUE-012).
//
// Isolamos o diretório global aqui uma única vez, em vez de lembrar de
// t.Setenv em cada teste. Testes que precisam do seu próprio diretório podem
// sobrescrever com t.Setenv(config.GlobalConfigDirEnv, t.TempDir()).
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "mem-cmd-test-global")
	if err != nil {
		// Sem temp dir não dá para isolar. Falhar alto é melhor do que
		// silenciosamente escrever no config do usuário.
		panic("nao foi possivel criar diretorio temporario para isolar o config global: " + err.Error())
	}
	defer os.RemoveAll(dir)

	if err := os.Setenv(config.GlobalConfigDirEnv, dir); err != nil {
		panic("nao foi possivel definir " + config.GlobalConfigDirEnv + ": " + err.Error())
	}

	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
