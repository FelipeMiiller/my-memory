package store

import (
	"testing"
)

func TestCalculateContentHash_Empty(t *testing.T) {
	hash := CalculateContentHash([]byte(""))
	expected := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if hash != expected {
		t.Fatalf("Esperava %s, obteve %s", expected, hash)
	}
	if len(hash) != 64 {
		t.Fatalf("Esperava tamanho 64, obteve %d", len(hash))
	}
}

func TestCalculateContentHash_KnownString(t *testing.T) {
	hash := CalculateContentHash([]byte("hello world"))
	expected := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if hash != expected {
		t.Fatalf("Esperava %s, obteve %s", expected, hash)
	}
}

func TestCalculateContentHash_DeterministicAndDifferent(t *testing.T) {
	docA := []byte("# Nota 1\nConteúdo sobre [[Arquitetura]].")
	docB := []byte("# Nota 1\nConteúdo sobre [[Arquitetura]] alterado.")

	hashA1 := CalculateContentHash(docA)
	hashA2 := CalculateContentHash(docA)
	hashB := CalculateContentHash(docB)

	if hashA1 != hashA2 {
		t.Fatalf("Esperava hashes determinísticos iguais para docA: %s != %s", hashA1, hashA2)
	}
	if hashA1 == hashB {
		t.Fatalf("Esperava hashes diferentes para conteúdos distintos: %s == %s", hashA1, hashB)
	}
	if len(hashA1) != 64 || len(hashB) != 64 {
		t.Fatalf("Hashes devem possuir 64 caracteres hexadecimais")
	}
}
