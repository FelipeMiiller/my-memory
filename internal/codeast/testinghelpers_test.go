package codeast

import (
	"database/sql"
	"errors"
	"os"
	"testing"
)

// toSQLDB extrai o *sql.DB subjacente do handle criado em cache_test.go.
// Como InitDB devolve *sql.DB, usamos type assertion e retornamos um handle
// que satisfaz a interface mínima.
func toSQLDB(h *dbHandle) *sql.DB {
	d, ok := h.db.(*sql.DB)
	if !ok {
		panic("toSQLDB: handle não contém *sql.DB")
	}
	return d
}

// writeFile encapsula os.WriteFile para evitar import direto nos testes
// (deixar código de teste focado no codeast).
func writeFile(path string, content []byte) error {
	if path == "" {
		return errors.New("writeFile: path vazio")
	}
	return os.WriteFile(path, content, 0o644)
}

// Compile-time guards: garante que os helpers acima ficam em sync com
// outras alterações de tipo no projeto.
var _ = (*sql.DB)(nil)
var _ = (*testing.T)(nil)
