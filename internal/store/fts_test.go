package store

import (
	"testing"
)

func TestSanitizeFTS5Query(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "consulta simples",
			input:    "turboquant",
			expected: `"turboquant"`,
		},
		{
			name:     "múltiplas palavras com pontuação",
			input:    `O que é "TurboQuant"? (4-bit)`,
			expected: `"O" "que" "é" "TurboQuant" "4" "bit"`,
		},
		{
			name:     "palavras reservadas FTS5",
			input:    "NOT AND OR NEAR",
			expected: `"NOT" "AND" "OR" "NEAR"`,
		},
		{
			name:     "caracteres especiais e operadores",
			input:    `+term1 -term2 *term3: "term4" ^term5`,
			expected: `"term1" "term2" "term3" "term4" "term5"`,
		},
		{
			name:     "string vazia e apenas espaços",
			input:    "   \t\n  ",
			expected: "",
		},
		{
			name:     "apenas pontuações",
			input:    "??? !!! ::: --- ***",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeFTS5Query(tt.input)
			if got != tt.expected {
				t.Errorf("SanitizeFTS5Query(%q) = %q, esperado %q", tt.input, got, tt.expected)
			}
		})
	}
}
