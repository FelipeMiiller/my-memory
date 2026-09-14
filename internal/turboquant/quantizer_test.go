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

func TestRotatorInverse(t *testing.T) {
	dim := 768
	r := NewRotator(dim, 32, 999)
	rng := rand.New(rand.NewSource(777))

	vec := generateRandomVector(dim, rng)
	rotated := r.Rotate(vec)
	restored := r.RotateInverse(rotated)

	if len(restored) != dim {
		t.Fatalf("tamanho inesperado: esperado %d, obtido %d", dim, len(restored))
	}

	for i := 0; i < dim; i++ {
		diff := math.Abs(float64(vec[i] - restored[i]))
		if diff > 1e-4 {
			t.Errorf("posição %d: restauração da inversa divergiu: original=%.6f, restaurado=%.6f", i, vec[i], restored[i])
			break
		}
	}

	// Testa dimensão inválida
	invalidVec := []float32{1.0, 2.0}
	invRes := r.RotateInverse(invalidVec)
	if len(invRes) != len(invalidVec) {
		t.Errorf("deve retornar vetor original para dimensão divergente")
	}
}

func TestDequantize(t *testing.T) {
	dim := 768
	q := NewQuantizer(dim)
	rng := rand.New(rand.NewSource(555))

	// Casos de borda
	if q.Dequantize(nil) != nil {
		t.Error("Dequantize(nil) deve retornar nil")
	}
	if q.Dequantize(&CompressedVector{Dim: 0}) != nil {
		t.Error("Dequantize com Dim=0 deve retornar nil")
	}

	vec := generateRandomVector(dim, rng)
	compressed, err := q.Quantize(vec)
	if err != nil {
		t.Fatalf("Quantize falhou: %v", err)
	}

	dequantized := q.Dequantize(compressed)
	if len(dequantized) != dim {
		t.Fatalf("esperado tamanho %d, obtido %d", dim, len(dequantized))
	}

	// O vetor reconstruído deve ter alta similaridade com o original (> 0.95)
	sim := exactDotProduct(vec, dequantized)
	if sim < 0.95 {
		t.Errorf("similaridade do vetor dequantizado baixa: %.4f", sim)
	}
}

func TestQuantize_InvalidDimension(t *testing.T) {
	dim := 768
	q := NewQuantizer(dim)

	_, err := q.Quantize([]float32{0.1, 0.2})
	if err == nil {
		t.Error("esperava erro ao quantizar vetor com dimensão diferente da configurada")
	}
}
