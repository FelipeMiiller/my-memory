package store

import (
	"strings"
	"testing"
)

// fixesByTerm indexa os achados por termo em caixa baixa, para asserções legíveis.
func fixesByTerm(fixes []TermFix) map[string]string {
	out := make(map[string]string, len(fixes))
	for _, f := range fixes {
		out[strings.ToLower(f.Term)] = f.Expected
	}
	return out
}

func requireNoFixes(t *testing.T, content string) []TermFix {
	t.Helper()
	fixes := FindUnaccentedTerms(content)
	if len(fixes) != 0 {
		t.Fatalf("esperava nenhum termo sinalizado em %q, obteve %+v", content, fixes)
	}
	return fixes
}

func requireFixes(t *testing.T, content string, want map[string]string) []TermFix {
	t.Helper()
	fixes := FindUnaccentedTerms(content)
	got := fixesByTerm(fixes)
	for term, expected := range want {
		actual, ok := got[term]
		if !ok {
			t.Errorf("termo %q não foi detectado em %q (obtive %+v)", term, content, fixes)
			continue
		}
		if actual != expected {
			t.Errorf("termo %q: esperava %q, obteve %q", term, expected, actual)
		}
	}
	if len(fixes) != len(want) {
		t.Errorf("esperava exatamente %d termos em %q, obteve %d: %+v", len(want), content, len(fixes), fixes)
	}
	return fixes
}

// ---------------------------------------------------------------------------
// POSITIVOS: o defeito real precisa ser encontrado
// ---------------------------------------------------------------------------

// TestFindUnaccentedTerms_DetectsCuratedList amarra a lista curada inteira.
// Se alguém corrigir uma entrada, o erro de digitação volta a passar pelo gate.
func TestFindUnaccentedTerms_DetectsCuratedList(t *testing.T) {
	cases := []struct{ term, expected string }{
		{"tres", "três"},
		{"sao", "são"},
		{"nao", "não"},
		{"apos", "após"},
		{"configuracao", "configuração"},
		{"variaveis", "variáveis"},
		{"funcoes", "funções"},
		{"usuario", "usuário"},
		{"indice", "índice"},
		{"tecnico", "técnico"},
		{"proposito", "propósito"},
		{"unico", "único"},
		{"ultimo", "último"},
		{"analise", "análise"},
		{"referencia", "referência"},
		{"introducao", "introdução"},
		{"validacao", "validação"},
		{"definicao", "definição"},
		{"implementacao", "implementação"},
		{"versao", "versão"},
	}

	for _, tc := range cases {
		t.Run(tc.term, func(t *testing.T) {
			// A prosa precisa conter o termo como palavra inteira, cercada por
			// pontuação — não dentro de outro termo.
			content := "O " + tc.term + " foi corrigido.\n"
			requireFixes(t, content, map[string]string{tc.term: tc.expected})
		})
	}
}

// TestFindUnaccentedTerms_ProseDeteccaoDeVariosTermos cobre o caso real do vault:
// vários termos sem acento na mesma frase de prosa.
func TestFindUnaccentedTerms_DetectsSeveralTermsInProse(t *testing.T) {
	content := "Tres vezes a configuracao falhou: nao ha caminho, apos o deploy.\n"
	requireFixes(t, content, map[string]string{
		"tres":         "Três",
		"configuracao": "configuração",
		"nao":          "não",
		"apos":         "após",
	})
}

// TestFindUnaccentedTerms_PreservesCase garante que o campo Expected é utilizável
// numa substituição automática, respeitando a capitalização do documento.
func TestFindUnaccentedTerms_PreservesCase(t *testing.T) {
	cases := []struct {
		content  string
		term     string
		expected string
	}{
		{"Tres sao dois.\n", "Tres", "Três"},
		{"Tres sao dois.\n", "sao", "são"},
		{"SAO tambem.\n", "SAO", "SÃO"},
		{"tres pontos.\n", "tres", "três"},
	}
	for _, tc := range cases {
		fixes := FindUnaccentedTerms(tc.content)
		found := false
		for _, f := range fixes {
			if f.Term == tc.term {
				found = true
				if f.Expected != tc.expected {
					t.Errorf("em %q, termo %q: esperava %q, obteve %q", tc.content, tc.term, tc.expected, f.Expected)
				}
			}
		}
		if !found {
			t.Errorf("termo %q não foi detectado em %q (obtive %+v)", tc.term, tc.content, fixes)
		}
	}
}

// ---------------------------------------------------------------------------
// NEGATIVOS: o teste que impede a regressão de dano.
// Um termo correto listado aqui NÃO pode ser tocado. Se um dia for, o doctor
// vai sugerir "corrigir" 400 palavras válidas de um vault.
// ---------------------------------------------------------------------------

func TestFindUnaccentedTerms_IgnoresCorrectUnaccentedPortuguese(t *testing.T) {
	content := "A media esta no limite de qualidade da entrada ate o fim; sim, ha mais.\n"
	requireNoFixes(t, content)
}

func TestFindUnaccentedTerms_IgnoresAlreadyAccentedPortuguese(t *testing.T) {
	content := "NÃO e ENTÃO são válidos: três, já, código, informação, está, são, não.\n"
	requireNoFixes(t, content)
}

func TestFindUnaccentedTerms_IgnoresEnglishTechnicalTerms(t *testing.T) {
	content := "O cache usa target fixo; index, default, build, runtime, commit, slug e timeout valem.\n"
	requireNoFixes(t, content)
}

func TestFindUnaccentedTerms_IgnoresSubstrings(t *testing.T) {
	// "indices" e "notas" se assemelham a termos do mapa, mas são palavras
	// diferentes e não podem ser sinalizadas.
	content := "Os indices do documento e as notas foram revisados.\n"
	requireNoFixes(t, content)
}

func TestFindUnaccentedTerms_IgnoresTermsInsideLongerWords(t *testing.T) {
	// Termo do mapa colado a letra: não é palavra inteira, não é erro.
	content := "indicacoes Tresnao aposicao CONFIGUREI IndiceBuild\n"
	requireNoFixes(t, content)
}

// ---------------------------------------------------------------------------
// Zonas que não são prosa humana
// ---------------------------------------------------------------------------

func TestFindUnaccentedTerms_IgnoresCodeFences(t *testing.T) {
	content := "Antes: tres.\n" +
		"```go\n" +
		"sao := tres(configuracao)\n" +
		"nao := funcoes(variaveis)\n" +
		"```\n" +
		"Depois: tres nao sao.\n"
	fixes := requireFixes(t, content, map[string]string{
		"tres": "três",
		"nao":  "não",
		"sao":  "são",
	})
	// Confirma que nada veio de dentro do bloco de código.
	for _, f := range fixes {
		if f.Term == "configuracao" || f.Term == "funcoes" || f.Term == "variaveis" {
			t.Errorf("termo %q veio de dentro do bloco de código", f.Term)
		}
	}
}

func TestFindUnaccentedTerms_IgnoresIndentedCodeAndCodeSpans(t *testing.T) {
	content := "Texto: tres.\n" +
		"\n" +
		"    nao := configuracao\n" +
		"\tapos := funcoes\n" +
		"Use `sao` e `referencia` inline.\n"
	requireFixes(t, content, map[string]string{"tres": "três"})
}

func TestFindUnaccentedTerms_IgnoresWikilinksAndURLs(t *testing.T) {
	// O rótulo do link é texto humano (e seria detectado); o alvo da URL, não.
	content := "Veja [[configuracao]] e [[tres-camadas|alias]].\n" +
		"Docs em https://exemplo.com/docs/configuracao e www.exemplo.com/tres.\n" +
		"[nota](https://exemplo.com/sao) e <https://exemplo.com/nao>\n"
	requireNoFixes(t, content)
}

func TestFindUnaccentedTerms_FrontmatterIgnoraChavesMasLeTitleESummary(t *testing.T) {
	content := "---\n" +
		"title: Configuracao do vault\n" +
		"summary: Tres notas para revisao\n" +
		"tags: [sao, configuracao, tres]\n" +
		"category: resource\n" +
		"aliases: [sao-nota]\n" +
		"id: 012-configuracao-do-vault\n" +
		"---\n" +
		"Corpo limpo.\n"
	fixes := requireFixes(t, content, map[string]string{
		"configuracao": "Configuração",
		"tres":         "Três",
		"revisao":      "revisão",
	})
	got := fixesByTerm(fixes)
	if _, ok := got["sao"]; ok {
		t.Error("tag de frontmatter não pode ser sinalizada")
	}
}

func TestFindUnaccentedTerms_FrontmatterJSON(t *testing.T) {
	// O valor JSON precisa ser capturado entre aspas: senão as tags da mesma
	// linha (`"tags": ["sao"]`) seriam varridas como prosa.
	content := "---\n" +
		"{\"title\": \"Configuracao do vault\", \"tags\": [\"sao\"], \"category\": \"resource\"}\n" +
		"---\n"
	requireFixes(t, content, map[string]string{"configuracao": "Configuração"})
}

// ---------------------------------------------------------------------------
// Bordas
// ---------------------------------------------------------------------------

func TestFindUnaccentedTerms_EmptyInput(t *testing.T) {
	for _, content := range []string{"", "   ", "\n\n", "\r\n"} {
		fixes := FindUnaccentedTerms(content)
		if len(fixes) != 0 {
			t.Errorf("entrada %q deveria devolver lista vazia, obteve %+v", content, fixes)
		}
	}
}

func TestFindUnaccentedTerms_DeduplicaPorTermo(t *testing.T) {
	content := "Tres vezes tres, e tres de novo.\n"
	fixes := requireFixes(t, content, map[string]string{"tres": "Três"})
	if len(fixes) != 1 {
		t.Fatalf("esperava um único achado, obteve %+v", fixes)
	}
}

// ---------------------------------------------------------------------------
// Acumulador (usado pelo mem doctor)
// ---------------------------------------------------------------------------

func TestOrthographyAccumulator_ZeroValueAndRanking(t *testing.T) {
	var acc OrthographyAccumulator // valor zero tem de servir

	acc.Add("doc-a", "Tres sao tres.\n")
	acc.Add("doc-a", "Tres de novo.\n") // 2º trecho com o mesmo termo
	acc.Add("doc-a", "Nao.\n")
	acc.Add("doc-b", "configuracao e versao\n")
	acc.Add("doc-c", "texto limpo\n")

	affected, issues := acc.Report(10)
	if affected != 2 {
		t.Fatalf("esperava 2 documentos afetados, obteve %d", affected)
	}
	if len(issues) != 5 {
		t.Fatalf("esperava 5 casos (doc-a: tres x2, sao, nao; doc-b: 2), obteve %d: %+v", len(issues), issues)
	}
	// Pior caso primeiro: "tres" aparece em 2 trechos do doc-a.
	if issues[0].Term != "Tres" || issues[0].DocumentID != "doc-a" || issues[0].Count != 2 {
		t.Errorf("esperava 'tres' do doc-a com Count 2 em primeiro, obteve %+v", issues[0])
	}
	if issues[0].Expected != "Três" {
		t.Errorf("esperava Expected %q, obteve %q", "Três", issues[0].Expected)
	}
}

func TestOrthographyAccumulator_ReportLimitaCasos(t *testing.T) {
	var acc OrthographyAccumulator
	for _, doc := range []string{"a", "b", "c", "d"} {
		acc.Add(doc, "sao e nao e tres\n")
	}
	affected, issues := acc.Report(2)
	if affected != 4 {
		t.Fatalf("esperava 4 documentos afetados, obteve %d", affected)
	}
	if len(issues) != 2 {
		t.Fatalf("esperava no máximo 2 casos, obteve %d", len(issues))
	}
}

func TestOrthographyAccumulator_ZeroValueReport(t *testing.T) {
	var acc OrthographyAccumulator
	affected, issues := acc.Report(10)
	if affected != 0 || len(issues) != 0 {
		t.Fatalf("acumulador vazio deveria devolver (0, []), obteve (%d, %+v)", affected, issues)
	}
}
