package turboquant

import (
	"math/rand"
	"testing"
)

func BenchmarkQuantize_4Bit(b *testing.B) {
	dim := 768
	q := NewQuantizer(dim)
	rng := rand.New(rand.NewSource(42))
	vec := generateRandomVector(dim, rng)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := q.Quantize(vec)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDotProduct_4Bit(b *testing.B) {
	dim := 768
	q := NewQuantizer(dim)
	rng := rand.New(rand.NewSource(42))
	query := generateRandomVector(dim, rng)
	rotatedQuery := q.RotateQuery(query)
	vec := generateRandomVector(dim, rng)
	compressed, err := q.Quantize(vec)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = q.DotProduct(rotatedQuery, compressed)
	}
}

func BenchmarkDotProduct_Float32(b *testing.B) {
	dim := 768
	rng := rand.New(rand.NewSource(42))
	v1 := generateRandomVector(dim, rng)
	v2 := generateRandomVector(dim, rng)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = exactDotProduct(v1, v2)
	}
}

func BenchmarkDequantize_4Bit(b *testing.B) {
	dim := 768
	q := NewQuantizer(dim)
	rng := rand.New(rand.NewSource(42))
	vec := generateRandomVector(dim, rng)
	compressed, err := q.Quantize(vec)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = q.Dequantize(compressed)
	}
}
