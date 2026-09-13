//go:build !((darwin || linux) && cgo)

package db

import (
	"encoding/binary"
	"math"
)

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
