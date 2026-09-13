package store

import (
	"strings"
	"unicode"
)

// SanitizeFTS5Query limpa e formata uma consulta de texto para evitar erros de sintaxe no SQLite FTS5.
// Remove caracteres especiais de controle do FTS5 e encapsula cada termo entre aspas duplas,
// garantindo que palavras reservadas (AND, OR, NOT) ou pontuações não quebrem a query SQL.
func SanitizeFTS5Query(rawQuery string) string {
	if strings.TrimSpace(rawQuery) == "" {
		return ""
	}

	// Substitui caracteres que possuem significado sintático no FTS5 por espaços
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			return r
		}
		return ' '
	}, rawQuery)

	fields := strings.Fields(cleaned)
	if len(fields) == 0 {
		return ""
	}

	// Envolve cada termo em aspas duplas para neutralizar palavras reservadas do FTS5 (ex: "NOT", "AND")
	var quotedTerms []string
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f != "" {
			quotedTerms = append(quotedTerms, `"`+f+`"`)
		}
	}

	return strings.Join(quotedTerms, " ")
}
