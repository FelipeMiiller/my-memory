package turboquant

import (
	"math"
	"math/rand"
)

// Rotator gerencia a rotação ortogonal aleatória determinística.
// Utiliza um produto de reflexões de Householder (H_k * ... * H_1 * x)
// garantindo ortogonalidade exata (R^T * R = I), conservação perfeita de norma
// e eliminação de canais com outliers extremos.
type Rotator struct {
	dim        int
	reflectors [][]float32
}

// NewRotator inicializa um rotador com semente determinística para a dimensão especificada.
func NewRotator(dim int, numReflections int, seed int64) *Rotator {
	if numReflections <= 0 {
		numReflections = 32
	}

	rng := rand.New(rand.NewSource(seed))
	reflectors := make([][]float32, numReflections)

	for i := 0; i < numReflections; i++ {
		v := make([]float32, dim)
		var normSq float64
		for j := 0; j < dim; j++ {
			val := float32(rng.NormFloat64())
			v[j] = val
			normSq += float64(val * val)
		}

		// Normaliza o vetor unitário de Householder ||v|| = 1
		norm := float32(math.Sqrt(normSq))
		if norm > 0 {
			for j := 0; j < dim; j++ {
				v[j] /= norm
			}
		}
		reflectors[i] = v
	}

	return &Rotator{
		dim:        dim,
		reflectors: reflectors,
	}
}

// Rotate aplica a rotação ortogonal: y = R * x
// Cada reflexão é calculada como: x' = x - 2 * (v^T * x) * v
func (r *Rotator) Rotate(x []float32) []float32 {
	if len(x) != r.dim {
		return x
	}

	out := make([]float32, r.dim)
	copy(out, x)

	for _, v := range r.reflectors {
		var dot float32
		for j := 0; j < r.dim; j++ {
			dot += v[j] * out[j]
		}
		twoDot := 2.0 * dot
		for j := 0; j < r.dim; j++ {
			out[j] -= twoDot * v[j]
		}
	}

	return out
}
