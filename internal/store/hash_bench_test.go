package store

import (
	"bytes"
	"testing"
)

func BenchmarkCalculateContentHash_1KB(b *testing.B) {
	data := bytes.Repeat([]byte("Markdown note line with [[link]] and text.\n"), 25) // ~1 KB
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = CalculateContentHash(data)
	}
}

func BenchmarkCalculateContentHash_64KB(b *testing.B) {
	data := bytes.Repeat([]byte("Markdown note line with [[link]] and text.\n"), 1600) // ~64 KB
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = CalculateContentHash(data)
	}
}

func BenchmarkCalculateContentHash_1MB(b *testing.B) {
	data := bytes.Repeat([]byte("Markdown note line with [[link]] and text.\n"), 25000) // ~1 MB
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = CalculateContentHash(data)
	}
}
