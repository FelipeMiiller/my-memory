package turboquant

import (
	"fmt"
	"math"
)

// CompressedVector representa o vetor quantizado em 4-bits com fator de escala
type CompressedVector struct {
	Dim   int     `json:"dim"`
	Scale float32 `json:"scale"`
	Data  []byte  `json:"data"` // 2 dimensões por byte (4 bits cada)
}

// Quantizer gerencia a rotação e quantização de 4 bits inspirada no TurboQuant
type Quantizer struct {
	rotator *Rotator
	dim     int
}

// NewQuantizer cria um quantizador para a dimensão especificada
func NewQuantizer(dim int) *Quantizer {
	// Semente determinística padrão para que todos os nós e índices usem a mesma base
	const defaultSeed = 421337
	return &Quantizer{
		rotator: NewRotator(dim, 32, defaultSeed),
		dim:     dim,
	}
}

// RotateQuery rotaciona o vetor de consulta para o espaço ortogonal
func (q *Quantizer) RotateQuery(query []float32) []float32 {
	return q.rotator.Rotate(query)
}

// Quantize comprime um vetor float32 para 4-bits empacotados
func (q *Quantizer) Quantize(vec []float32) (*CompressedVector, error) {
	if len(vec) != q.dim {
		return nil, fmt.Errorf("dimensão inválida: esperado %d, recebido %d", q.dim, len(vec))
	}

	// 1. Aplica a rotação ortogonal aleatória para dispersar outliers
	rotated := q.rotator.Rotate(vec)

	// 2. Determina o valor máximo absoluto (fator de escala)
	var maxAbs float32
	for _, val := range rotated {
		abs := float32(math.Abs(float64(val)))
		if abs > maxAbs {
			maxAbs = abs
		}
	}
	if maxAbs == 0 {
		maxAbs = 1e-6
	}

	// 3. Quantização escalar de 4-bits com 15 níveis simétricos (-7 a +7)
	// Bins de -7 a +7 mapeados para [0..14] somando 7 (cabe em 4 bits: 0x00 a 0x0F)
	numBytes := (q.dim + 1) / 2
	packed := make([]byte, numBytes)

	for i := 0; i < q.dim; i += 2 {
		// Primeiro valor (nibble inferior)
		norm0 := (rotated[i] / maxAbs) * 7.0
		val0 := int(math.Round(float64(norm0)))
		if val0 < -7 {
			val0 = -7
		} else if val0 > 7 {
			val0 = 7
		}
		u0 := byte(val0 + 7) // 0..14

		// Segundo valor (nibble superior)
		var u1 byte
		if i+1 < q.dim {
			norm1 := (rotated[i+1] / maxAbs) * 7.0
			val1 := int(math.Round(float64(norm1)))
			if val1 < -7 {
				val1 = -7
			} else if val1 > 7 {
				val1 = 7
			}
			u1 = byte(val1 + 7)
		}

		packed[i/2] = (u0 & 0x0F) | ((u1 & 0x0F) << 4)
	}

	return &CompressedVector{
		Dim:   q.dim,
		Scale: maxAbs,
		Data:  packed,
	}, nil
}

// DotProduct estima o produto escalar não-viesado entre a query rotacionada e o vetor comprimido
func (q *Quantizer) DotProduct(rotatedQuery []float32, cv *CompressedVector) float32 {
	if len(rotatedQuery) != cv.Dim {
		return 0
	}

	var sum float32
	dim := cv.Dim
	data := cv.Data

	for i := 0; i < dim; i += 2 {
		b := data[i/2]
		u0 := int(b & 0x0F)
		v0 := float32(u0 - 7)
		sum += rotatedQuery[i] * v0

		if i+1 < dim {
			u1 := int((b >> 4) & 0x0F)
			v1 := float32(u1 - 7)
			sum += rotatedQuery[i+1] * v1
		}
	}

	// Reconverte usando o fator de escala original: (Scale / 7.0) * sum
	return (cv.Scale / 7.0) * sum
}

// Dequantize descompacta o vetor de 4-bits de volta para o espaço original float32
func (q *Quantizer) Dequantize(cv *CompressedVector) []float32 {
	if cv == nil || cv.Dim == 0 {
		return nil
	}
	dim := cv.Dim
	data := cv.Data
	rotated := make([]float32, dim)

	step := cv.Scale / 7.0
	for i := 0; i < dim; i += 2 {
		b := data[i/2]
		u0 := int(b & 0x0F)
		rotated[i] = step * float32(u0-7)

		if i+1 < dim {
			u1 := int((b >> 4) & 0x0F)
			rotated[i+1] = step * float32(u1-7)
		}
	}

	// Inverte a rotação ortogonal: R^T * rotated (como R é ortogonal, a inversa é a transposta)
	return q.rotator.RotateInverse(rotated)
}
