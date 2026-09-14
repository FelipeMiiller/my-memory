//go:build !((darwin || linux) && cgo && sqlite_fts5)

package db

import (
	"database/sql"
	"encoding/binary"
	"math"

	_ "modernc.org/sqlite"
)

const HasSqliteVec = false

func initSqliteVec() {
	// sqlite-vec cgo bindings não disponíveis nesta plataforma/configuração
}

func serializeFloat32(vec []float32) ([]byte, error) {
	b := make([]byte, len(vec)*4)
	for i, v := range vec {
		binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(v))
	}
	return b, nil
}

func openDB(dbPath string) (*sql.DB, error) {
	return sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
}
