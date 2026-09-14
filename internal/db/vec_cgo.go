//go:build (darwin || linux) && cgo

package db

import (
	"database/sql"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	_ "github.com/mattn/go-sqlite3"
)

const HasSqliteVec = true

func initSqliteVec() {
	sqlite_vec.Auto()
}

func serializeFloat32(vec []float32) ([]byte, error) {
	return sqlite_vec.SerializeFloat32(vec)
}

func openDB(dbPath string) (*sql.DB, error) {
	return sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL")
}
