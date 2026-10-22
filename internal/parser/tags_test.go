package parser

import (
	"strings"
	"testing"
)

// TestNormalizeTag_ValidTags valida as tags que a norma do Obsidian aceita
// (obsidian.md/help/tags): letras, números, '_', '-', '/' e Unicode, com ao
// menos um caractere não-numérico.
func TestNormalizeTag_ValidTags(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"#a", "a"},
		{"#a_b", "a_b"},
		{"#a-b", "a-b"},
		{"#a/b", "a/b"},
		{"#ação", "ação"},
		{"#🚀", "🚀"},
		{"#y1984", "y1984"},
		{"#1a", "1a"},
		{"#_tag", "_tag"},
		{"#Ω", "ω"}, // a canônica é minúscula (comparação case-insensitive)
		{"a", "a"},  // sem '#' — o frontmatter entrega a tag sem o '#'
	}

	for _, tc := range cases {
		got, ok := NormalizeTag(tc.in)
		if !ok {
			t.Errorf("NormalizeTag(%q) = %q, false; esperava tag válida", tc.in, got)
			continue
		}
		if got != tc.want {
			t.Errorf("NormalizeTag(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestNormalizeTag_InvalidTags valida as tags que a norma rejeita.
func TestNormalizeTag_InvalidTags(t *testing.T) {
	cases := []string{
		"#1984", // só numérica
		"#a b",  // espaço em branco
		"#a.b",  // pontuação
		"#",     // vazia
		"",      // vazia
		"#a\tb", // espaço em branco
		"##a",   // '#' não é caractere de tag
	}

	for _, in := range cases {
		got, ok := NormalizeTag(in)
		if ok {
			t.Errorf("NormalizeTag(%q) = %q, true; esperava tag inválida", in, got)
		}
		if got != "" {
			t.Errorf("NormalizeTag(%q) devolveu %q para tag inválida; esperava string vazia", in, got)
		}
	}
}

// TestNormalizeTag_CaseInsensitive valida que `#Tag` e `#TAG` são a mesma tag:
// a canônica é a identidade da tag (ADR-055, A.8).
func TestNormalizeTag_CaseInsensitive(t *testing.T) {
	c1, ok1 := NormalizeTag("#Tag")
	c2, ok2 := NormalizeTag("#TAG")

	if !ok1 || !ok2 {
		t.Fatalf("esperava as duas tags válidas: ok1=%v ok2=%v", ok1, ok2)
	}
	if c1 != c2 {
		t.Errorf("canônicas divergem: %q vs %q; esperava a mesma", c1, c2)
	}
	if c1 != "tag" {
		t.Errorf("canônica = %q, want %q", c1, "tag")
	}
}

// TestNormalizeTag_KeepsHierarchy valida que a hierarquia sobrevive à
// canonicalização: `#projeto/alpha` → `projeto/alpha`.
func TestNormalizeTag_KeepsHierarchy(t *testing.T) {
	got, ok := NormalizeTag("#Projeto/Alpha")
	if !ok {
		t.Fatalf("esperava tag hierárquica válida")
	}
	if got != "projeto/alpha" {
		t.Errorf("canônica = %q, want %q", got, "projeto/alpha")
	}
}

// TestTagRegex_ObsidianCharset valida que o regex de tag aceita o conjunto de
// caracteres da norma e NÃO trunca `#ação` (que antes virava a tag espúria "a").
func TestTagRegex_ObsidianCharset(t *testing.T) {
	cases := []struct {
		content string
		want    string
	}{
		{"#1a", "1a"},
		{"#_tag", "_tag"},
		{"#ação", "ação"},
		{"#🚀", "🚀"},
		{"#Ω", "Ω"},
		{"#projeto/alpha", "projeto/alpha"},
		{"#v1/release", "v1/release"},
	}

	for _, tc := range cases {
		m := tagRegex.FindStringSubmatch(tc.content)
		if m == nil {
			t.Errorf("tagRegex não capturou %q", tc.content)
			continue
		}
		if m[1] != tc.want {
			t.Errorf("tagRegex capturou %q em %q, want %q", m[1], tc.content, tc.want)
		}
	}
}

// TestTagRegex_RejectsNumericOnlyAndSpaces valida as duas recusas do regex:
// tag só-numérica e espaço em branco.
func TestTagRegex_RejectsNumericOnlyAndSpaces(t *testing.T) {
	if m := tagRegex.FindStringSubmatch("#1984"); m != nil {
		t.Errorf("tagRegex capturou tag só-numérica: %q", m[1])
	}

	// O espaço delimita a tag: o que é capturado é "a", nunca "a b".
	m := tagRegex.FindStringSubmatch("#a b")
	if m == nil {
		t.Fatalf("esperava a tag \"a\" em \"#a b\"")
	}
	if strings.Contains(m[1], " ") {
		t.Errorf("tagRegex capturou tag com espaço: %q", m[1])
	}
	if m[1] != "a" {
		t.Errorf("tagRegex capturou %q em \"#a b\", want %q", m[1], "a")
	}
}

// TestExtractConnections_TagContexts valida os contextos em que a tag NÃO é
// tag (blocos de código, comentários do Obsidian) e o contexto em que ela É
// tag (célula de tabela).
func TestExtractConnections_TagContexts(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "fenced-code-triple-backtick",
			content: "```markdown\n#nao-e-tag\n```\n",
			want:    nil,
		},
		{
			name:    "fenced-code-triple-tilde",
			content: "~~~markdown\n#nao-e-tag\n~~~\n",
			want:    nil,
		},
		{
			name:    "inline-code",
			content: "Exemplo `#nao-e-tag` inline.",
			want:    nil,
		},
		{
			name:    "obsidian-comment-inline",
			content: "%% oculto #tag-oculta %% e #tag-visivel",
			want:    []string{"tag-visivel"},
		},
		{
			name:    "obsidian-comment-block",
			content: "%%\n#tag-oculta\n%%\n#tag-visivel\n",
			want:    []string{"tag-visivel"},
		},
		{
			// O '%' de porcentagem não abre comentário: o delimiter só abre
			// comentário quando não é precedido por caractere de palavra.
			name:    "percentual-nao-e-comentario",
			content: "Cobertura de 50%% com #tag-1 e depois 90%% com #tag-2\n",
			want:    []string{"tag-1", "tag-2"},
		},
		{
			name:    "markdown-table-cell",
			content: "| Nota | Tags |\n|---|---|\n| alpha | #backlog #projeto/alpha |\n",
			want:    []string{"backlog", "projeto/alpha"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertTags(t, tc.content, tc.want)
		})
	}
}

// TestExtractConnections_TableRowWikilinkStillStripped valida que a defesa
// contra link fantasma não regrediu: wikilink em linha de tabela continua não
// virando aresta, mesmo com o novo contexto de tag.
func TestExtractConnections_TableRowWikilinkStillStripped(t *testing.T) {
	content := `| Sintaxe de Relações Tipadas | [[rel:target]] e [[Target]] | Default |
|------------------------------|-----------------------------------|---------|
| Wikilink comum               | [[Target]]                        | Default |
`

	conn := ExtractConnections(content)
	if len(conn.Links) != 0 {
		t.Errorf("esperava 0 links, obteve %d: %+v", len(conn.Links), conn.Links)
	}
	if len(conn.Edges) != 0 {
		t.Errorf("esperava 0 edges, obteve %d: %+v", len(conn.Edges), conn.Edges)
	}
	if len(conn.OutgoingLinks) != 0 {
		t.Errorf("esperava 0 outgoing links, obteve %v", conn.OutgoingLinks)
	}
}

// TestExtractConnections_TagKeepsRawForm valida que conn.Tags guarda a tag
// CRUA (a forma de exibição, com o casing original). A canonicalização é
// responsabilidade de quem chama NormalizeTag.
func TestExtractConnections_TagKeepsRawForm(t *testing.T) {
	conn := ExtractConnections("Tags do documento: #Backend e #projeto/Alpha")

	found := false
	for _, tag := range conn.Tags {
		if tag == "Backend" {
			found = true
		}
	}
	if !found {
		t.Errorf("esperava a tag crua \"Backend\" em conn.Tags, obteve %v", conn.Tags)
	}
}

// TestResolveTagTargets valida a resolução tag → documento, com a mesma
// estratégia de fuzzy match das arestas (exato, substring, token).
func TestResolveTagTargets(t *testing.T) {
	available := []string{
		"001-uso-de-sqlite-como-camada-unificada-de-dados",
		"002-adocao-de-go-como-linguagem-principal",
		"Busca Híbrida",
		"Memória",
	}

	t.Run("match exato", func(t *testing.T) {
		got := ResolveTagTargets([]string{"memória"}, available)
		if got["memória"] != "Memória" {
			t.Errorf("resolução = %q, want %q", got["memória"], "Memória")
		}
	})

	t.Run("match por substring", func(t *testing.T) {
		got := ResolveTagTargets([]string{"sqlite"}, available)
		want := "001-uso-de-sqlite-como-camada-unificada-de-dados"
		if got["sqlite"] != want {
			t.Errorf("resolução = %q, want %q", got["sqlite"], want)
		}
	})

	t.Run("match por token", func(t *testing.T) {
		got := ResolveTagTargets([]string{"progressive-loading"}, []string{
			"025-progressive-context-loading-e-taxonomia-de-memoria",
		})
		want := "025-progressive-context-loading-e-taxonomia-de-memoria"
		if got["progressive-loading"] != want {
			t.Errorf("resolução = %q, want %q", got["progressive-loading"], want)
		}
	})

	t.Run("tag sem casa fica no mapa com valor vazio", func(t *testing.T) {
		got := ResolveTagTargets([]string{"nonexistent-concept"}, available)

		value, ok := got["nonexistent-concept"]
		if !ok {
			t.Fatalf("a tag órfã sumiu do mapa: %v", got)
		}
		if value != "" {
			t.Errorf("resolução = %q, want %q (tag órfã)", value, "")
		}
	})

	t.Run("a chave é a canônica", func(t *testing.T) {
		got := ResolveTagTargets([]string{"#TAG", "Memória"}, available)

		if _, ok := got["tag"]; !ok {
			t.Errorf("esperava a chave canônica \"tag\", obteve %v", got)
		}
		if _, ok := got["#TAG"]; ok {
			t.Errorf("a chave não pode ser a tag crua: %v", got)
		}
		if got["memória"] != "Memória" {
			t.Errorf("resolução = %q, want %q", got["memória"], "Memória")
		}
	})

	t.Run("tag inválida não entra no mapa", func(t *testing.T) {
		got := ResolveTagTargets([]string{"#1984", "#a b"}, available)
		if len(got) != 0 {
			t.Errorf("esperava mapa vazio, obteve %v", got)
		}
	})

	t.Run("lista vazia devolve mapa vazio", func(t *testing.T) {
		got := ResolveTagTargets(nil, available)
		if len(got) != 0 {
			t.Errorf("esperava mapa vazio, obteve %v", got)
		}
	})
}

// assertTags falha o teste se as tags extraídas não baterem com as esperadas.
func assertTags(t *testing.T, content string, want []string) {
	t.Helper()

	got := ExtractConnections(content).Tags
	if len(got) != len(want) {
		t.Fatalf("tags = %v, want %v\ncontent:\n%s", got, want, content)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("tags[%d] = %q, want %q\ncontent:\n%s", i, got[i], want[i], content)
		}
	}
}
