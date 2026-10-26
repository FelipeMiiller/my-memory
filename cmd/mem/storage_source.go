package main

import (
	"log"
	"os"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/config"
)

// Fontes que podem vencer na resolução do storage.
//
// O rótulo é o que o usuário lê no log: ele precisa dizer QUAL fonte
// ganhou. Nenhum rótulo carrega URL nem credencial — só o nome da
// origem.
const (
	storageSourceFlagPostgres = "flag --postgres"
	storageSourceEnvMyMemory  = "MY_MEMORY_PG_URL"
	storageSourceEnvPostgres  = "POSTGRES_URL"
	storageSourceEnvDatabase  = "DATABASE_URL"
	storageSourceConfigGlobal = "config global"
	storageSourceConfigLocal  = "config local"
)

// resolvePostgresURL resolve a URL do Postgres e devolve a FONTE que venceu.
//
// Precedência (ADR-040, "Detection Logic" — a mesma hierarquia da ADR-016):
// flag CLI > variável de ambiente > config (global/local).
//
// A precedência em si NÃO mudou: o que mudou foi a documentação (que
// dizia "global > local > env", invertendo a ordem real) e a visibilidade.
// Sem a fonte, um config de teste era ignorado em silêncio e o binário
// conectava no Postgres real — foi o incidente do verificador (D8).
func resolvePostgresURL(cfg *config.Config, pgURLFlag string) (url string, source string) {
	if pgURLFlag != "" {
		return pgURLFlag, storageSourceFlagPostgres
	}
	if envPG := os.Getenv("MY_MEMORY_PG_URL"); envPG != "" {
		return envPG, storageSourceEnvMyMemory
	}
	if envPG := os.Getenv("POSTGRES_URL"); envPG != "" {
		return envPG, storageSourceEnvPostgres
	}
	if envPG := os.Getenv("DATABASE_URL"); envPG != "" {
		return envPG, storageSourceEnvDatabase
	}
	if cfg != nil && cfg.Storage.Engine == "postgres" && cfg.Storage.PostgresURL != "" {
		return cfg.Storage.PostgresURL, postgresConfigSource(cfg)
	}
	return "", ""
}

// postgresConfigSource diz se a URL do config veio do global ou do local.
//
// O config em cascata sobrescreve o global pelo local (ADR-040), então a
// origem se descobre comparando a URL efetiva com a do config global.
func postgresConfigSource(cfg *config.Config) string {
	if cfg == nil {
		return storageSourceConfigGlobal
	}
	gcfg, err := config.LoadGlobalConfig()
	if err != nil || gcfg == nil {
		return storageSourceConfigLocal
	}
	if strings.TrimSpace(gcfg.Storage.PostgresURL) == strings.TrimSpace(cfg.Storage.PostgresURL) {
		return storageSourceConfigGlobal
	}
	return storageSourceConfigLocal
}

// logStorageSource anuncia a fonte que venceu na resolução do storage.
//
// O `log` do Go escreve no stderr: o stdout do relatório JSON da CLI tem
// que continuar puro (o mesmo cuidado do boot log do code stage).
//
// A mensagem carrega só o engine e a origem — nunca a URL.
func logStorageSource(engine, source string) {
	if source == "" {
		return
	}
	log.Printf("storage: %s (fonte: %s)", engine, source)
}
