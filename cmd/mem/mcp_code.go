package main

import (
	"database/sql"

	"github.com/FelipeMiiller/my-memory/internal/store"
)

// codeEngineDB devolve a conexão do engine ativo para o code graph:
// a do PostgreSQL quando o vault está no PostgreSQL, a do SQLite local caso
// contrário.
//
// Devolve nil quando não há engine — o servidor MCP segue com as tools de
// código sem executor (resposta "indisponível"), que é o comportamento de
// um ambiente sem code pipeline.
func codeEngineDB(pgStore *store.PostgresStore, database *sql.DB) *sql.DB {
	if pgStore != nil {
		return pgStore.DB()
	}
	return database
}
