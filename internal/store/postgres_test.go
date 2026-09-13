package store

import (
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
