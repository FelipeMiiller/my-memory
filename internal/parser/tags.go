package parser

import (
	"strings"
	"unicode"
)

// Conjunto de caracteres do corpo de uma tag, segundo a norma do Obsidian
// (obsidian.md/help/tags):
//
//   - tagAllowedChars: tudo que pode aparecer no corpo de uma tag.
//   - tagNonNumericChars: o que pode aparecer E não é numérico. A tag precisa
//     ter pelo menos um desses caracteres: `#1984` é inválida, `#y1984` é válida.
//
// As duas classes são espelhadas por rune em isAllowedTagRune, para que a
// validação (NormalizeTag) e a extração (tagRegex) aceitem exatamente o mesmo
// conjunto de caracteres.
//
// A classe NÃO inclui pontuação (`#a.b` é a tag `a`, como no Obsidian) nem
// espaço em branco (a tag termina no espaço).
const (
	tagAllowedChars    = `\p{L}\p{M}\p{S}\p{N}_/-`
	tagNonNumericChars = `\p{L}\p{M}\p{S}_/-`
)

// NormalizeTag devolve a forma canônica de uma tag (minúsculas, sem o '#') e
// diz se ela é válida segundo o formato de tag do Obsidian: letras, números,
// '_', '-', '/' e Unicode; ao menos um caractere não-numérico; sem espaços.
//
// A forma canônica é a **identidade** da tag: `#Tag` e `#TAG` são a mesma tag
// (ADR-055, A.8). A forma de exibição — o casing da primeira criação — é
// problema de quem persiste; o parser entrega a tag crua em conn.Tags.
//
// A hierarquia é preservada na canônica: `#projeto/alpha` → `projeto/alpha`.
//
// Exemplos:
//
//	NormalizeTag("#Tag")    → ("tag", true)
//	NormalizeTag("#TAG")    → ("tag", true)
//	NormalizeTag("#ação")   → ("ação", true)
//	NormalizeTag("#1984")   → ("", false)   // só numérica
//	NormalizeTag("#a b")    → ("", false)   // espaço
//	NormalizeTag("#a.b")    → ("", false)   // pontuação
func NormalizeTag(tag string) (canonical string, ok bool) {
	raw := strings.TrimSpace(tag)
	raw = strings.TrimPrefix(raw, "#")
	if raw == "" {
		return "", false
	}

	hasNonNumeric := false
	for _, r := range raw {
		if !isAllowedTagRune(r) {
			return "", false
		}
		if !unicode.IsNumber(r) {
			hasNonNumeric = true
		}
	}
	if !hasNonNumeric {
		// A tag precisa de ao menos um caractere não-numérico: `#1984` é inválida.
		return "", false
	}

	return strings.ToLower(raw), true
}

// isAllowedTagRune diz se o caractere pode fazer parte do corpo de uma tag.
//
// O conjunto é o da norma do Obsidian: letras de qualquer script (inclusive
// acentuadas), números, marcas de combinação (seletor de variação de emoji),
// símbolos Unicode (emoji) e os separadores '_', '-' e '/'. Pontuação — o
// ponto inclusive — não faz parte: ela delimita a tag na prosa.
func isAllowedTagRune(r rune) bool {
	switch {
	case r == '_' || r == '-' || r == '/':
		return true
	case unicode.IsLetter(r):
		return true
	case unicode.IsNumber(r):
		return true
	case unicode.IsMark(r):
		return true
	case unicode.IsSymbol(r):
		return true
	default:
		return false
	}
}

// ResolveTagTargets resolve cada tag literal para o título do documento que ela
// nomeia, com a mesma estratégia de fuzzy match de ResolveTagConnections.
// A chave do mapa é a tag canônica; tag sem casa devolve "" no mapa.
//
// Regras:
//   - A chave é a forma canônica de NormalizeTag (`#TAG` → "tag"), porque a
//     identidade da tag é a canônica (ADR-055, A.8).
//   - O valor é o **título** do documento, não o id. A conversão título → id é
//     de quem indexa.
//   - Tag sem casa **continua no mapa**, com valor "". A tag órfã é
//     informação, não erro (ADR-055, A.6) — é ela que o inventário lista.
//   - Tag inválida segundo a norma não entra no mapa: ela não tem forma
//     canônica, logo não tem chave.
//
// A lista availableDocs deve conter os títulos dos documentos já indexados
// (o mesmo formato de ListDocumentTitles, `COALESCE(title, id)`).
func ResolveTagTargets(tags []string, availableDocs []string) map[string]string {
	targets := make(map[string]string, len(tags))
	if len(tags) == 0 {
		return targets
	}

	// Mapa normalizado -> título: o mesmo pré-computo de ResolveTagConnections.
	norm := make(map[string]string, len(availableDocs))
	for _, d := range availableDocs {
		key := normalizeForFuzzy(d)
		if key == "" {
			continue
		}
		norm[key] = d
	}

	for _, raw := range tags {
		canonical, ok := NormalizeTag(raw)
		if !ok {
			continue
		}
		if _, exists := targets[canonical]; exists {
			continue
		}
		// fuzzyResolveTag devolve "" quando a tag não casa com nenhum doc —
		// a chave permanece no mapa.
		targets[canonical] = fuzzyResolveTag(canonical, norm, availableDocs)
	}

	return targets
}
