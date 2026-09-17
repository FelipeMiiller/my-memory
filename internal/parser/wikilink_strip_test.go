package parser

import (
	"testing"
)

// TestExtractConnections_StripsWikilinkContexts valida que wikilinks em
// contextos onde devem ser tratados como texto literal NÃO viram edges.
//
// Cobertura (defesa em profundidade contra falsos positivos):
//   - Inline code (backticks simples)
//   - Fenced code (```)
//   - HTML comments (<!-- ... -->)
//   - Linhas de tabela Markdown (| ... |)
//
// Caso de controle: wikilink em prosa EARS continua sendo extraído quando
// não está em nenhum desses contextos (sintaxe legítima).
func TestExtractConnections_StripsWikilinkContexts(t *testing.T) {
	cases := []struct {
		name      string
		content   string
		wantLinks int
	}{
		{
			name:      "inline-code-wikilink-stripped",
			content:   "Quero poder escrever `[[memory://central/standards/oauth2|Padrão de Autenticação]]` ou `[[implements:memory://central/standards/oauth2]]`.",
			wantLinks: 0,
		},
		{
			name:      "fenced-code-wikilink-stripped",
			content:   "```markdown\nSee [[memory://central/standards/oauth2]] for auth.\n```\n",
			wantLinks: 0,
		},
		{
			name:      "html-comment-wikilink-stripped",
			content:   "<!-- TODO: link [[Target]] here once the spec stabilizes -->\n",
			wantLinks: 0,
		},
		{
			name: "markdown-table-wikilink-stripped",
			content: `| Sintaxe de Relações Tipadas | [[rel:target]] e [[target|rel:tipo]] | Intuitivo |
|------------------------------|----------------------------------------|-----------|
| Wikilink comum               | [[Target]]                              | Default    |
`,
			wantLinks: 0,
		},
		{
			name:      "unwrapped-wikilink-in-ears-prose-extracted",
			content:   "When the user references [[memory://central/standards/oauth2]] in a note, the system SHALL resolve the URI.",
			wantLinks: 1,
		},
		{
			name: "inline-code-and-real-wikilink-on-separate-lines",
			content: "Example `[[Target]]` is a placeholder.\n" +
				"But `[[Real Note]]` would also be stripped (backticks always win).\n" +
				"This is the realistic case: prose with real wikilink + backticked example.\n",
			wantLinks: 0, // Both `[[Target]]` and `[[Real Note]]` are in backticks → both stripped
		},
		{
			name:      "prose-with-real-wikilink-after-backticked-example",
			content:   "Example `[[Target]]` is a placeholder. The real link is [[memory://central/standards/oauth2]].",
			wantLinks: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			conn := ExtractConnections(tc.content)
			if got := len(conn.Links); got != tc.wantLinks {
				t.Errorf("Links count = %d, want %d\ncontent:\n%s", got, tc.wantLinks, tc.content)
				for i, lt := range conn.Links {
					t.Logf("  link[%d]: target=%q raw=%q", i, lt.Target, lt.Raw)
				}
			}
			if len(conn.OutgoingLinks) != tc.wantLinks {
				t.Errorf("OutgoingLinks count = %d, want %d", len(conn.OutgoingLinks), tc.wantLinks)
			}
		})
	}
}
