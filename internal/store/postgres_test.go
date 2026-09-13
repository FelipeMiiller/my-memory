package store

import (
	"strings"
	"testing"
)

func TestFormatVector(t *testing.T) {
	vec := []float32{0.1, -0.5, 1.25}
	got := FormatVector(vec)
	expected := "[0.1,-0.5,1.25]"
	if got != expected {
		t.Errorf("FormatVector(%v) = %s, esperava %s", vec, got, expected)
	}
}

func TestFormatVector_Empty(t *testing.T) {
	vec := []float32{}
	got := FormatVector(vec)
	expected := "[]"
	if got != expected {
		t.Errorf("FormatVector(%v) = %s, esperava %s", vec, got, expected)
	}
}

func TestPostgresStore_InterfaceCompliance(t *testing.T) {
	var _ Store = (*PostgresStore)(nil)
}

func TestPostgresStore_SearchFTS_EmptyQuery(t *testing.T) {
	s := &PostgresStore{}
	res, err := s.SearchFTS(nil, "repo", "   ", 10)
	if err != nil {
		t.Fatalf("Esperava erro nulo para query vazia, obteve: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("Esperava resultado vazio para query vazia, obteve %d itens", len(res))
	}
}

func TestPostgresSchema_ContentHash(t *testing.T) {
	if !strings.Contains(PostgresSchema, "content_hash TEXT") {
		t.Fatalf("PostgresSchema deve conter coluna content_hash TEXT")
	}
	if !strings.Contains(PostgresSchema, "ALTER TABLE documents ADD COLUMN IF NOT EXISTS content_hash TEXT;") {
		t.Fatalf("PostgresSchema deve conter migração retrocompatível para content_hash")
	}
}
