package parser

import (
	"strings"
)

// ResolveTagConnections reescreve os targets das edges `tagged_as` em conn
// para apontar pra notas existentes quando possível, usando a mesma estratégia
// de fuzzy match do Obsidian:
//
//  1. Match exato case-insensitive
//  2. Substring normalizado (sem hífens/underscores, sem acentos)
//
// Se nenhum match for encontrado, o target original é preservado (não perdemos
// informação: a tag continua existindo como nó conceitual). Se múltiplos matches
// existirem, usa o primeiro encontrado (ordem determinística da lista de entrada).
//
// A lista availableDocs deve conter os títulos (ou IDs) das notas já
// indexadas. Se vazia ou nil, a função é no-op.
func ResolveTagConnections(conn *ExtractedConnections, availableDocs []string) {
	if conn == nil || len(availableDocs) == 0 {
		return
	}

	// Pre-compute mapa de normalizado -> original para lookup O(1)
	norm := make(map[string]string, len(availableDocs))
	for _, d := range availableDocs {
		key := normalizeForFuzzy(d)
		if key == "" {
			continue
		}
		norm[key] = d
	}

	// Mutate edges in place (range por índice, não valor)
	for i := range conn.Edges {
		if conn.Edges[i].Relation != "tagged_as" {
			continue
		}
		original := conn.Edges[i].Target
		resolved := fuzzyResolveTag(original, norm, availableDocs)
		if resolved != "" && resolved != original {
			conn.Edges[i].Target = resolved
		}
	}
}

// fuzzyResolveTag resolve uma tag contra o mapa normalizado e a lista original.
// Retorna "" se não houver match (o caller mantém o target original).
func fuzzyResolveTag(tag string, norm map[string]string, originals []string) string {
	if tag == "" {
		return ""
	}

	// 1. Match exato (case-insensitive, normalizado)
	key := normalizeForFuzzy(tag)
	if doc, ok := norm[key]; ok {
		return doc
	}

	// 2. Substring match (case-insensitive)
	for _, d := range originals {
		nd := normalizeForFuzzy(d)
		if nd == "" {
			continue
		}
		// tag é substring do doc OU doc é substring da tag
		if strings.Contains(nd, key) || strings.Contains(key, nd) {
			return d
		}
	}

	return "" // sem match, caller mantém original
}

// accentMap mapeia caracteres acentuados comuns em PT-BR para suas versões
// sem acento. Suficiente para os títulos do vault (Wikilinks, Memória, etc.).
var accentMap = map[rune]rune{
	'á': 'a', 'à': 'a', 'â': 'a', 'ã': 'a', 'ä': 'a',
	'é': 'e', 'ê': 'e', 'è': 'e', 'ë': 'e',
	'í': 'i', 'î': 'i', 'ì': 'i', 'ï': 'i',
	'ó': 'o', 'ô': 'o', 'õ': 'o', 'ò': 'o', 'ö': 'o',
	'ú': 'u', 'û': 'u', 'ù': 'u', 'ü': 'u',
	'ç': 'c',
	'ñ': 'n',
}

// normalizeForFuzzy normaliza uma string para comparação fuzzy:
//   - lowercase
//   - remove hífens, underscores e espaços
//   - remove acentos comuns (PT-BR)
//
// "001-uso-de-sqlite" → "usodesqlite"
// "Memória"           → "memoria"
// "Busca Híbrida"     → "buscahibrida"
func normalizeForFuzzy(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))

	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '-' || r == '_' || r == ' ' || r == '\t' {
			continue
		}
		if plain, ok := accentMap[r]; ok {
			b.WriteRune(plain)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
