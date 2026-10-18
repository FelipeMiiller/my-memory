package store

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Ortografia do português em documentos indexados (ISSUE-017).
//
// Por que existe: o vault `central-memory` é gerado por agentes de IA. Em 2026-10-18
// foram encontrados 158 termos sem acento espalhados por 14 pastas. Isso não é
// cosmético: "sao" e "são" tokenizam diferente no FTS5 e no embedding, então uma
// busca pelo termo correto não encontra o termo digitado.
//
// A lista de correções abaixo é CURADA, nunca derivada por regex. Um `\w+` genérico
// corromperia texto correto ("esta", "ate", "ha" e "media" são palavras válidas sem
// acento). Só entra palavra que, em português, é sempre escrita acentuada.

// TermFix é o par canônico de uma correção: como o termo está e como deveria estar.
type TermFix struct {
	Term     string // como está no documento
	Expected string // como deveria estar
}

// PortugueseOrthographyIssue descreve um termo que deveria estar acentuado em um documento.
type PortugueseOrthographyIssue struct {
	DocumentID string
	Term       string // como está no documento
	Expected   string // como deveria estar
	// Count é o número de trechos (chunks) do documento em que o termo aparece.
	// Serve para ranquear os piores casos; não é contagem de caracteres.
	Count int
}

// OrthographyIssueLimit é quantos casos o relatório do doctor detalha.
const OrthographyIssueLimit = 20

// portugueseAccentFixes mapeia palavra COMPLETA sem acento -> palavra acentuada.
// A chave é sempre minúscula e ASCII. A comparação em FindUnaccentedTerms é
// case-insensitive e respeita limites de palavra Unicode (uma letra acentuada
// conta como letra, então "três" nunca casa com "tres").
//
// Termos deliberadamente AUSENTES deste mapa, porque detectá-los é falso positivo:
//   - esta    pronome ("esta nota"); o verbo correto é "está", já acentuado
//   - ate     preposição correta sem acento
//   - ha      verbo "há" correto sem acento
//   - sim, na, por, sem, solo, limite, entrada, qualidade, media: corretos sem acento
//   - so      ambíguo ("só" x aporto)
//   - indices plural de "índice" usado como identificador técnico
//   - cache, index, target, default, build, runtime, commit, slug, timeout: inglês
var portugueseAccentFixes = map[string]string{
	// --- Verbos, advérbios e numerais ---
	"tres":    "três",
	"sao":     "são",
	"nao":     "não",
	"apos":    "após",
	"atraves": "através",
	"tambem":  "também",
	"porem":   "porém",
	"voce":    "você",
	"voces":   "vocês",
	"estao":   "estão",
	"serao":   "serão",
	"alem":    "além",

	// --- Família -ção / -ções ---
	"acao":           "ação",
	"acoes":          "ações",
	"agencia":        "agência",
	"analise":        "análise",
	"analises":       "análises",
	"atualizacao":    "atualização",
	"autenticacao":   "autenticação",
	"automacao":      "automação",
	"avaliacao":      "avaliação",
	"avaliacoes":     "avaliações",
	"ciencia":        "ciência",
	"compilacao":     "compilação",
	"configuracao":   "configuração",
	"configuracoes":  "configurações",
	"conexao":        "conexão",
	"conexoes":       "conexões",
	"decisao":        "decisão",
	"decisoes":       "decisões",
	"definicao":      "definição",
	"definicoes":     "definições",
	"documentacao":   "documentação",
	"evidencia":      "evidência",
	"execucao":       "execução",
	"experiencia":    "experiência",
	"implementacao":  "implementação",
	"implementacoes": "implementações",
	"informacao":     "informação",
	"informacoes":    "informações",
	"instalacao":     "instalação",
	"integracao":     "integração",
	"introducao":     "introdução",
	"introducoes":    "introduções",
	"iteracao":       "iteração",
	"logica":         "lógica",
	"manutencao":     "manutenção",
	"migracao":       "migração",
	"navegacao":      "navegação",
	"observacao":     "observação",
	"observacoes":    "observações",
	"opcao":          "opção",
	"opcoes":         "opções",
	"permissao":      "permissão",
	"persistencia":   "persistência",
	"politica":       "política",
	"politicas":      "políticas",
	"referencia":     "referência",
	"referencias":    "referências",
	"repeticao":      "repetição",
	"revisao":        "revisão",
	"sessao":         "sessão",
	"sessoes":        "sessões",
	"tolerancia":     "tolerância",
	"validacao":      "validação",
	"validacoes":     "validações",
	"versao":         "versão",
	"versoes":        "versões",
	"visualizacao":   "visualização",

	// --- Funções, variáveis e substantivos técnicos ---
	"cenario":      "cenário",
	"cenarios":     "cenários",
	"comentario":   "comentário",
	"dicionario":   "dicionário",
	"diretorio":    "diretório",
	"estatistica":  "estatística",
	"estatisticas": "estatísticas",
	"funcao":       "função",
	"funcoes":      "funções",
	"indice":       "índice",
	"laboratorio":  "laboratório",
	"matematica":   "matemática",
	"padrao":       "padrão",
	"padroes":      "padrões",
	"proposito":    "propósito",
	"publico":      "público",
	"relatorio":    "relatório",
	"relatorios":   "relatórios",
	"sequencia":    "sequência",
	"tecnico":      "técnico",
	"tecnicos":     "técnicos",
	"ultimo":       "último",
	"ultimos":      "últimos",
	"unico":        "único",
	"usuario":      "usuário",
	"usuarios":     "usuários",
	"variavel":     "variável",
	"variaveis":    "variáveis",
}

// Regexps de filtragem de trechos que não são prosa humana.
var (
	reCodeSpanDouble = regexp.MustCompile("``[^`]*``")
	reCodeSpanSingle = regexp.MustCompile("`[^`]*`")
	reHTMLComment    = regexp.MustCompile(`<!--.*?-->`)
	reWikilink       = regexp.MustCompile(`!?\[\[[^\]]*\]\]`)
	reAutoLink       = regexp.MustCompile(`<(?:https?|ftp|mailto):[^>]*>`)
	reURL            = regexp.MustCompile(`(?i)\b(?:https?|ftp)://\S+|\bwww\.\S+`)
	reMarkdownTarget = regexp.MustCompile(`\]\([^)]*\)`)
)

// reUnaccentedTerms casa um termo curado precedido de início de linha ou de um
// caractere que não é letra/número/underscore. O limite direito é conferido em
// código porque RE2 não tem lookahead — sem ele "media" casaria dentro de
// "mediado" e "tres" casaria no meio de "indicaTres".
var reUnaccentedTerms = regexp.MustCompile(`(?i)(^|[^\pL\pN_])(` + unaccentedTermsAlternation() + `)`)

// unaccentedTermsAlternation monta a alternância dos termos curados, do mais longo
// para o mais curto (determinístico, independente da ordem de iteração do map).
func unaccentedTermsAlternation() string {
	keys := make([]string, 0, len(portugueseAccentFixes))
	for k := range portugueseAccentFixes {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if len(keys[i]) != len(keys[j]) {
			return len(keys[i]) > len(keys[j])
		}
		return keys[i] < keys[j]
	})
	quoted := make([]string, len(keys))
	for i, k := range keys {
		quoted[i] = regexp.QuoteMeta(k)
	}
	return strings.Join(quoted, "|")
}

// FindUnaccentedTerms devolve os termos sem acento de um documento, na ordem de
// primeira aparição e sem repetir o mesmo termo.
//
// Ignora de propósito (não é prosa humana): blocos de código (```), linhas
// indentadas com 4 espaços ou tab, code spans, URLs, `[[wikilinks]]` e chaves de
// frontmatter. As exceções dentro do frontmatter são os VALORES de `title` e
// `summary`, que são texto escrito para humanos e por isso podem ter o defeito.
func FindUnaccentedTerms(content string) []TermFix {
	if strings.TrimSpace(content) == "" {
		return nil
	}

	var found []TermFix
	seen := make(map[string]bool)

	inFence := false
	inFrontmatter := false
	frontmatterChecked := false

	lines := strings.Split(content, "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], "\r")
		if i == 0 {
			line = strings.TrimPrefix(line, "\ufeff")
		}
		trimmed := strings.TrimSpace(line)

		// O frontmatter só abre na primeira linha não vazia do documento.
		if !frontmatterChecked && trimmed != "" {
			frontmatterChecked = true
			if trimmed == "---" {
				inFrontmatter = true
				continue
			}
		}

		if inFrontmatter {
			if trimmed == "---" || trimmed == "..." {
				inFrontmatter = false
				continue
			}
			if value, ok := humanFrontmatterValue(line); ok {
				collectTermFixes(&found, seen, value)
			}
			continue
		}

		if isFenceLine(trimmed) {
			inFence = !inFence
			continue
		}
		if inFence || isIndentedCode(line) || trimmed == "" {
			continue
		}

		collectTermFixes(&found, seen, stripNonProse(line))
	}
	return found
}

// isFenceLine identifica abertura/fechamento de bloco de código cercado.
func isFenceLine(trimmed string) bool {
	return strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")
}

// isIndentedCode identifica linha de código indentada (4 espaços ou tab).
func isIndentedCode(line string) bool {
	return strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t")
}

// stripNonProse remove code spans, wikilinks, URLs e alvos de link Markdown.
func stripNonProse(line string) string {
	out := reCodeSpanDouble.ReplaceAllString(line, " ")
	out = reCodeSpanSingle.ReplaceAllString(out, " ")
	out = reHTMLComment.ReplaceAllString(out, " ")
	out = reWikilink.ReplaceAllString(out, " ")
	out = reAutoLink.ReplaceAllString(out, " ")
	out = reURL.ReplaceAllString(out, " ")
	out = reMarkdownTarget.ReplaceAllString(out, "]")
	return out
}

// reFrontmatterHuman* localizam as duas únicas chaves de frontmatter que são prosa
// humana: `title` e `summary`. Duas formas, porque o valor JSON precisa ser
// capturado entre aspas — senão a linha inteira (`{"title": "X", "tags": ["sao"]}`)
// varreria também as tags.
var (
	reFrontmatterJSONHuman = regexp.MustCompile(`(?i)"\s*(?:title|summary)\s*"\s*:\s*"([^"]*)"`)
	reFrontmatterYAMLHuman = regexp.MustCompile(`(?i)^\s*["']?\s*(?:title|summary)\s*["']?\s*:\s*(.+)$`)
)

// humanFrontmatterValue devolve o valor de `title`/`summary` de uma linha de
// frontmatter (YAML ou JSON). Qualquer outra chave — tags, category, aliases,
// urls, ids — é configuração, não prosa, e por isso é ignorada.
func humanFrontmatterValue(line string) (string, bool) {
	if strings.TrimSpace(line) == "" {
		return "", false
	}
	if m := reFrontmatterJSONHuman.FindStringSubmatch(line); m != nil {
		return m[1], m[1] != ""
	}
	m := reFrontmatterYAMLHuman.FindStringSubmatch(line)
	if m == nil {
		return "", false
	}
	value := strings.Trim(strings.TrimSpace(m[1]), `"'`)
	return value, value != ""
}

// collectTermFixes anexa à lista os termos curados encontrados em text.
func collectTermFixes(found *[]TermFix, seen map[string]bool, text string) {
	for _, m := range reUnaccentedTerms.FindAllStringSubmatchIndex(text, -1) {
		// m[0:2] é o match completo (inclui o boundary inicial), m[2:4] é o grupo
		// do boundary e m[4:6] é o grupo do termo propriamente dito.
		start, end := m[4], m[5]
		if !endsAtWordBoundary(text, end) {
			continue
		}
		term := text[start:end]
		key := strings.ToLower(term)
		if seen[key] {
			continue
		}
		expected, ok := portugueseAccentFixes[key]
		if !ok || expected == "" {
			continue
		}
		seen[key] = true
		*found = append(*found, TermFix{Term: term, Expected: matchCase(term, expected)})
	}
}

// endsAtWordBoundary confere o limite direito do termo usando categorias Unicode.
func endsAtWordBoundary(s string, end int) bool {
	if end >= len(s) {
		return true
	}
	r, _ := utf8.DecodeRuneInString(s[end:])
	return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_'
}

// matchCase preserva a capitalização do documento no campo Expected:
// "Tres" -> "Três", "SAO" -> "SÃO", "tres" -> "três".
func matchCase(found, expected string) string {
	if found == strings.ToUpper(found) && found != strings.ToLower(found) {
		return strings.ToUpper(expected)
	}
	foundRunes := []rune(found)
	expectedRunes := []rune(expected)
	if len(foundRunes) > 0 && len(expectedRunes) > 0 && unicode.IsUpper(foundRunes[0]) {
		expectedRunes[0] = unicode.ToUpper(expectedRunes[0])
	}
	return string(expectedRunes)
}

// OrthographyAccumulator agrega termos sem acento documento a documento sem
// manter o texto indexado em memória. O valor zero é utilizável.
type OrthographyAccumulator struct {
	perDoc map[string]map[string]*PortugueseOrthographyIssue
	docs   map[string]struct{}
}

// Add varre um trecho (título, abstract ou chunk) de um documento.
func (a *OrthographyAccumulator) Add(documentID, content string) {
	fixes := FindUnaccentedTerms(content)
	if len(fixes) == 0 {
		return
	}
	if a.perDoc == nil {
		a.perDoc = make(map[string]map[string]*PortugueseOrthographyIssue)
		a.docs = make(map[string]struct{})
	}
	byTerm, ok := a.perDoc[documentID]
	if !ok {
		byTerm = make(map[string]*PortugueseOrthographyIssue)
		a.perDoc[documentID] = byTerm
		a.docs[documentID] = struct{}{}
	}
	for _, fix := range fixes {
		key := strings.ToLower(fix.Term)
		issue, ok := byTerm[key]
		if !ok {
			issue = &PortugueseOrthographyIssue{
				DocumentID: documentID,
				Term:       fix.Term,
				Expected:   fix.Expected,
			}
			byTerm[key] = issue
		}
		issue.Count++
	}
}

// Report devolve quantos documentos foram afetados e os piores casos (maior
// contagem primeiro), truncados em limit. É um AVISO: nada aqui altera o Health Score.
func (a *OrthographyAccumulator) Report(limit int) (int, []PortugueseOrthographyIssue) {
	issues := make([]PortugueseOrthographyIssue, 0, 16)
	for _, byTerm := range a.perDoc {
		for _, issue := range byTerm {
			issues = append(issues, *issue)
		}
	}
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Count != issues[j].Count {
			return issues[i].Count > issues[j].Count
		}
		if issues[i].Term != issues[j].Term {
			return issues[i].Term < issues[j].Term
		}
		return issues[i].DocumentID < issues[j].DocumentID
	})
	if limit > 0 && len(issues) > limit {
		issues = issues[:limit]
	}
	return len(a.docs), issues
}
