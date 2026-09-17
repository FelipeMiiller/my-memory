package parser

import "testing"

// TestResolveTagConnections valida a resolução fuzzy de tags contra uma
// lista de docs disponíveis (estratégia igual ao Obsidian).
func TestResolveTagConnections(t *testing.T) {
	available := []string{
		"001-uso-de-sqlite-como-camada-unificada-de-dados",
		"002-adocao-de-go-como-linguagem-principal",
		"003-compressao-vetorial-de-4-bit-via-turboquant",
		"Busca Híbrida", // com acento
		"Memória",       // com acento
	}

	t.Run("exact match", func(t *testing.T) {
		conn := mkTaggedConn([]string{"memória"})
		ResolveTagConnections(conn, available)
		assertEdges(t, conn, []string{"Memória"})
	})

	t.Run("fuzzy substring: tag inside doc slug", func(t *testing.T) {
		conn := mkTaggedConn([]string{"sqlite"})
		ResolveTagConnections(conn, available)
		assertEdges(t, conn, []string{"001-uso-de-sqlite-como-camada-unificada-de-dados"})
	})

	t.Run("fuzzy substring: tag inside doc title", func(t *testing.T) {
		conn := mkTaggedConn([]string{"busca"})
		ResolveTagConnections(conn, available)
		assertEdges(t, conn, []string{"Busca Híbrida"})
	})

	t.Run("accent normalization", func(t *testing.T) {
		conn := mkTaggedConn([]string{"memoria"}) // sem acento, mas doc tem "Memória"
		ResolveTagConnections(conn, available)
		assertEdges(t, conn, []string{"Memória"})
	})

	t.Run("no match keeps original", func(t *testing.T) {
		conn := mkTaggedConn([]string{"nonexistent-concept"})
		ResolveTagConnections(conn, available)
		assertEdges(t, conn, []string{"nonexistent-concept"})
	})

	t.Run("nil connection is no-op", func(t *testing.T) {
		ResolveTagConnections(nil, available) // should not panic
	})

	t.Run("empty docs is no-op", func(t *testing.T) {
		conn := mkTaggedConn([]string{"sqlite"})
		ResolveTagConnections(conn, nil)
		assertEdges(t, conn, []string{"sqlite"})
	})

	t.Run("non-tagged edges untouched", func(t *testing.T) {
		conn := &ExtractedConnections{
			Edges: []EdgeConnection{
				{Target: "wikilink-target", Relation: "links_to"},
				{Target: "sqlite", Relation: "tagged_as"},
			},
		}
		ResolveTagConnections(conn, available)
		if conn.Edges[0].Target != "wikilink-target" {
			t.Errorf("non-tagged edge was modified: got %q", conn.Edges[0].Target)
		}
		if conn.Edges[1].Target != "001-uso-de-sqlite-como-camada-unificada-de-dados" {
			t.Errorf("tagged edge was not resolved: got %q", conn.Edges[1].Target)
		}
	})
}

// TestNormalizeForFuzzy valida a normalização.
func TestNormalizeForFuzzy(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Memória", "memoria"},
		{"Busca Híbrida", "buscahibrida"},
		{"001-uso-de-sqlite", "001usodesqlite"},
		{"Café_com_leite", "cafecomleite"},
		{"áéíóú ç", "aeiouc"},
	}
	for _, c := range cases {
		if got := normalizeForFuzzy(c.in); got != c.want {
			t.Errorf("normalizeForFuzzy(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// mkTaggedConn é um helper que cria uma ExtractedConnections com 1 edge tagged_as
// por tag. Útil para testes do fuzzy resolver.
func mkTaggedConn(tags []string) *ExtractedConnections {
	conn := &ExtractedConnections{}
	for _, t := range tags {
		conn.Edges = append(conn.Edges, EdgeConnection{
			Target:   t,
			Relation: "tagged_as",
		})
	}
	return conn
}

// assertEdges falha o teste se as edges resultantes não baterem com o esperado.
func assertEdges(t *testing.T, conn *ExtractedConnections, want []string) {
	t.Helper()
	if len(conn.Edges) != len(want) {
		t.Fatalf("edges count: got %d, want %d", len(conn.Edges), len(want))
	}
	for i, w := range want {
		if conn.Edges[i].Target != w {
			t.Errorf("edges[%d].Target = %q, want %q", i, conn.Edges[i].Target, w)
		}
	}
}
