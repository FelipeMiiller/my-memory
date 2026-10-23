package db

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"sync"
)

// Dialect identifica o SQL do engine configurado.
type Dialect string

const (
	// DialectSQLite é o dialeto do memory.db local (modernc.org/sqlite).
	DialectSQLite Dialect = "sqlite"
	// DialectPostgres é o dialeto do PostgreSQL com pgvector (lib/pq).
	DialectPostgres Dialect = "postgres"
)

// dialectCache memoiza o dialeto por conexão (*sql.DB): a detecção custa uma
// query e o code pipeline reusa a mesma conexão em cada arquivo indexado.
var dialectCache sync.Map

// DetectDialect descobre o dialeto do banco conectado.
//
// A detecção usa `SELECT version()`: o SQLite devolve a versão da biblioteca
// e o PostgreSQL devolve "PostgreSQL <versão> ...". O resultado é memoizado
// por *sql.DB.
func DetectDialect(ctx context.Context, database *sql.DB) Dialect {
	if database == nil {
		return DialectSQLite
	}
	if cached, ok := dialectCache.Load(database); ok {
		if d, ok := cached.(Dialect); ok {
			return d
		}
	}
	dialect := DialectSQLite
	var version string
	if err := database.QueryRowContext(ctx, "SELECT version()").Scan(&version); err == nil {
		if strings.Contains(strings.ToLower(version), "postgresql") {
			dialect = DialectPostgres
		}
	}
	dialectCache.Store(database, dialect)
	return dialect
}

// IsPostgres devolve true quando o dialeto é o do PostgreSQL.
func (d Dialect) IsPostgres() bool {
	return d == DialectPostgres
}

// Bind converte os placeholders "?" no marcador ordinal do dialeto.
//
// O SQLite aceita "?"; o lib/pq exige "$1", "$2", ... — por isso o mesmo
// código de escrita serve aos dois engines sem duplicar as queries.
func (d Dialect) Bind(query string) string {
	if !d.IsPostgres() {
		return query
	}
	return bindOrdinal(query)
}

// bindOrdinal troca cada "?" por "$N". Literais de string ('...') e
// identificadores ("...") são preservados: um "?" dentro deles não é
// placeholder.
func bindOrdinal(query string) string {
	var sb strings.Builder
	sb.Grow(len(query) + 8)
	n := 0
	inString := false
	inQuoted := false
	for i := 0; i < len(query); i++ {
		ch := query[i]
		switch {
		case inString:
			sb.WriteByte(ch)
			if ch == '\'' {
				// '' é escape de aspas simples em SQL.
				if i+1 < len(query) && query[i+1] == '\'' {
					sb.WriteByte('\'')
					i++
					continue
				}
				inString = false
			}
		case inQuoted:
			sb.WriteByte(ch)
			if ch == '"' {
				inQuoted = false
			}
		case ch == '\'':
			inString = true
			sb.WriteByte(ch)
		case ch == '"':
			inQuoted = true
			sb.WriteByte(ch)
		case ch == '?':
			n++
			sb.WriteByte('$')
			sb.WriteString(strconv.Itoa(n))
		default:
			sb.WriteByte(ch)
		}
	}
	return sb.String()
}
