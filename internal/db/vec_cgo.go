//go:build (darwin || linux) && cgo

package db

import (
	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
)

func initSqliteVec() {
	sqlite_vec.Auto()
}

func serializeFloat32(vec []float32) ([]byte, error) {
	return sqlite_vec.SerializeFloat32(vec)
}
