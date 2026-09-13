package turboquant

import (
	"math"
	"math/rand"
	"testing"
)

func generateRandomVector(dim int, rng *rand.Rand) []float32 {
	v := make([]float32, dim)
	var normSq float64
	for i := 0; i < dim; i++ {
		val := float32(rng.NormFloat64())
		v[i] = val
		normSq += float64(val * val)
	}
	norm := float32(math.Sqrt(normSq))
	for i := 0; i < dim; i++ {
		v[i] /= norm
	}
	return v
}

func exactDotProduct(a, b []float32) float32 {
	var sum float32
	for i := range a {
		sum += a[i] * b[i]
	}
	return sum
}

func TestTurboQuantFidelity(t *testing.T) {
	dim := 768
	q := NewQuantizer(dim)
	rng := rand.New(rand.NewSource(12345))

	query := generateRandomVector(dim, rng)
	rotatedQuery := q.RotateQuery(query)

	numVectors := 50
	var totalCosSimDiff float64

	for i := 0; i < numVectors; i++ {
		vec := generateRandomVector(dim, rng)
		exact := exactDotProduct(query, vec)

		compressed, err := q.Quantize(vec)
		if err != nil {
			t.Fatalf("erro ao quantizar: %v", err)
		}

		if len(compressed.Data) != dim/2 {
			t.Fatalf("tamanho inesperado: esperado %d bytes, obtido %d bytes", dim/2, len(compressed.Data))
		}

		estimated := q.DotProduct(rotatedQuery, compressed)

		diff := math.Abs(float64(exact - estimated))
		totalCosSimDiff += diff
	}

	avgError := totalCosSimDiff / float64(numVectors)
	t.Logf("Erro médio absoluto no produto escalar: %.5f (dimensão=%d, tamanho=%d bytes)", avgError, dim, dim/2)

	// O erro médio esperado para 4-bits com rotação deve ser menor que 0.05
	if avgError > 0.05 {
		t.Errorf("erro médio muito alto: %.5f", avgError)
	}
}

func TestRotatorNormPreservation(t *testing.T) {
	dim := 768
	r := NewRotator(dim, 32, 999)
	rng := rand.New(rand.NewSource(888))

	vec := generateRandomVector(dim, rng)
	rotated := r.Rotate(vec)

	var normOrig, normRot float64
	for i := 0; i < dim; i++ {
		normOrig += float64(vec[i] * vec[i])
		normRot += float64(rotated[i] * rotated[i])
	}

	diff := math.Abs(normOrig - normRot)
	if diff > 1e-4 {
		t.Errorf("rotação não preservou a norma: original=%.6f, rotacionada=%.6f", normOrig, normRot)
	}
}
