package main

import (
	"fmt"
	"log"
)

// applyStorageOverride aplica o override de engine vindo da flag --storage.
// Implementa ADR-040 EARS-4 (--storage=sqlite mesmo com Postgres global) e
// EARS-5 (--storage=postgres sem URL = erro explícito).
//
// override deve ser "", "sqlite" ou "postgres". Qualquer outro valor retorna erro.
// Quando override=="sqlite", resolvedPG é zerado e o SQLite local será usado.
// Quando override=="postgres", resolvedPG deve estar preenchido (via --postgres
// flag, env var, ou storage.postgres_url no config); erro se ausente.
func applyStorageOverride(override, resolvedDB, resolvedPG string) (string, string, error) {
	switch override {
	case "":
		return resolvedDB, resolvedPG, nil
	case "sqlite":
		if resolvedPG != "" {
			log.Println("ℹ️  SQLite local forçado via --storage=sqlite; Postgres ignorado nesta sessão")
		}
		return resolvedDB, "", nil
	case "postgres":
		if resolvedPG == "" {
			return "", "", fmt.Errorf("--storage=postgres exige --postgres <url> ou storage.postgres_url no config")
		}
		return "", resolvedPG, nil
	default:
		return "", "", fmt.Errorf("--storage inválido: %q (use 'sqlite' ou 'postgres')", override)
	}
}
