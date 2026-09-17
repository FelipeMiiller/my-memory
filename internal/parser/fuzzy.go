package parser

import (
	"strings"
)

// ResolveTagConnections reescreve os targets das edges `tagged_as` E `links_to`
// em conn para apontar pra notas existentes quando possível, usando a mesma
// estratégia de fuzzy match do Obsidian:
//
//  1. Match exato case-insensitive
//  2. Substring normalizado (sem hífens/underscores, sem acentos)
//
// Aplica-se a:
//   - `tagged_as`: tags de frontmatter e inline `#tag` (ex: `tags: [sqlite]` → ADR-001)
//   - `links_to`: wikilinks `[[Wikilinks]]` apontando pra uma seção/nota existente
//
// Não aplica a relações tipadas (implements, depends_on, extends, etc.) —
// essas têm semântica explícita e não devem ser reescritas.
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
		rel := conn.Edges[i].Relation
		if rel != "tagged_as" && rel != "links_to" {
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
//
// Estratégia de match (em ordem de prioridade):
//  1. Match exato (case-insensitive, normalizado)
//  2. Substring bidirecional normalizado
//  3. Token match: tag "progressive-loading" tem tokens ["progressive", "loading"];
//     se QUALQUER token (len >= 4) aparece em QUALQUER doc normalizado, match.
//     Pega casos onde a tag descreve o conceito mas o slug não é literal.
func fuzzyResolveTag(tag string, norm map[string]string, originals []string) string {
	if tag == "" {
		return ""
	}

	key := normalizeForFuzzy(tag)

	// 1. Match exato
	if doc, ok := norm[key]; ok {
		return doc
	}

	// 2. Substring match
	for _, d := range originals {
		nd := normalizeForFuzzy(d)
		if nd == "" {
			continue
		}
		if strings.Contains(nd, key) || strings.Contains(key, nd) {
			return d
		}
	}

	// 3. Token match: tokeniza a tag ORIGINAL (antes do normalize) pra preservar
	// separadores como '-' e '_'. Cada token (len >= 4) é procurado em cada doc
	// normalizado. Pega tags compostas que descrevem o conceito sem ser literal.
	tokens := tokenizeForFuzzy(tag)
	for _, d := range originals {
		nd := normalizeForFuzzy(d)
		if nd == "" {
			continue
		}
		for _, t := range tokens {
			tLower := strings.ToLower(t)
			if len(tLower) < 4 {
				continue
			}
			if strings.Contains(nd, tLower) {
				return d
			}
		}
	}

	return "" // sem match
}

// tokenizeForFuzzy divide uma string em tokens (palavras) usando '-' e '_' como
// separadores. Case preserved (normalização é feita no caller).
func tokenizeForFuzzy(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_' || r == ' ' || r == '.'
	})
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
