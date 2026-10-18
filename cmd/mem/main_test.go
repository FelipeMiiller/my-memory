package main

import (
	"os"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/config"
)

// envStorageVars são as variáveis que fazem o CLI rotear para um Postgres
// externo. resolveStorageAndRepo as lê do ambiente ANTES de qualquer config
// (main.go), então apontar o config global para um temp dir não basta: sem
// limpá-las, um teste que espera SQLite local acaba connectando no Postgres
// real do usuário.
var envStorageVars = []string{
	"MY_MEMORY_PG_URL",
	"MY_MEMORY_PG_PASSWORD",
	"POSTGRES_URL",
	"DATABASE_URL",
}

// TestMain isola o storage global de TODO o pacote de testes.
//
// Duas camadas, porque o vazamento vinha das duas:
//
//  1. Config global. Sem isolar, testes que chamam runInit() registravam
//     repositórios em ~/.memory/config.yaml — o config real do usuário. Cada
//     `go test` local adicionava entradas TestRunInit* e o arquivo chegou a
//     acumular ~160 registros (ISSUE-012).
//
//  2. Variáveis de ambiente de storage. MY_MEMORY_PG_URL tem precedência sobre
//     a config inteira, então um teste esperando SQLite local conectava no
//     Postgres do usuário e falhava com "Postgres detectado no config/env".
//
//  3. Descoberta de .env (ISSUE-016). A camada 2 sozinha não segurava: unclean,
//     todo subcomando que chama resolveConfig() faz FindAndLoadDotEnv("."),
//     que sobe até a raiz do repositório, acha o `.memory/.env` do projeto e
//     repõe MY_MEMORY_PG_URL/DATABASE_URL via os.Setenv. Como o processo de
//     teste é o pai dos `mem.exe` que a suíte executa, o vazamento alcançava
//     também os subprocessos. SkipDotEnvEnv desliga a descoberta na raiz.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "mem-cmd-test-global")
	if err != nil {
		// Sem temp dir não dá para isolar. Falhar alto é melhor do que
		// silenciosamente escrever no config do usuário.
		panic("nao foi possivel criar diretorio temporario para isolar o config global: " + err.Error())
	}

	if err := os.Setenv(config.GlobalConfigDirEnv, dir); err != nil {
		panic("nao foi possivel definir " + config.GlobalConfigDirEnv + ": " + err.Error())
	}

	// Guarda os valores originais e restaura ao final, para não afetar a
	// sessão do desenvolvedor que chamou `go test`.
	saved := map[string]string{}
	savedSkip, hadSkip := os.LookupEnv(config.SkipDotEnvEnv)
	if err := os.Setenv(config.SkipDotEnvEnv, "1"); err != nil {
		panic("nao foi possivel definir " + config.SkipDotEnvEnv + ": " + err.Error())
	}
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
	if hadSkip {
		_ = os.Setenv(config.SkipDotEnvEnv, savedSkip)
	} else {
		_ = os.Unsetenv(config.SkipDotEnvEnv)
	}
	os.RemoveAll(dir)
	os.Exit(code)
}
