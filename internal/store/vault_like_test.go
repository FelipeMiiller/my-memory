package store

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

// TestEscapeLikePattern é o teste puro do escaper: `_` e `%` deixavam de ser
// curingas do LIKE.
func TestEscapeLikePattern(t *testing.T) {
	cases := []struct{ in, want string }{
		{"my_note.md", `my\_note.md`},
		{"100%.md", `100\%.md`},
		{`back\slash.md`, `back\\slash.md`},
		{"a_b%c\\d.md", `a\_b\%c\\d.md`},
		{"sem-curinga.md", "sem-curinga.md"},
		{"", ""},
	}
	for _, c := range cases {
		if got := EscapeLikePattern(c.in); got != c.want {
			t.Errorf("EscapeLikePattern(%q) = %q, esperava %q", c.in, got, c.want)
		}
	}
}

// TestPostgresFindVaultDocumentsByPath_IgnoraCuringasNoNome prova, contra o
// Postgres real, que um nome com `_` não casa com um arquivo de nome diferente.
// Sem o escape, `mem read my_note.md` casaria com `myXnote.md` e o comando
// reportaria ambiguidade onde não existe.
func TestPostgresFindVaultDocumentsByPath_IgnoraCuringasNoNome(t *testing.T) {
	pgURL := os.Getenv("MY_MEMORY_PG_URL")
	if pgURL == "" {
		t.Skip("Pulando integração PostgreSQL: MY_MEMORY_PG_URL não configurada")
	}

	ctx := context.Background()
	s, err := NewPostgresStore(pgURL)
	if err != nil {
		t.Fatalf("conexão com o PostgreSQL falhou: %v", err)
	}
	defer s.Close()

	repo := fmt.Sprintf("test-like-%d", time.Now().UnixNano())
	now := time.Now().Unix()
	docs := []string{"notas/my_note.md", "notas/myXnote.md", "outro/my_note.md", "notas/100%.md", "notas/100x.md"}
	for _, p := range docs {
		if err := s.InsertDocumentWithMeta(ctx, repo, p, p, p, now, "h-"+p, "", "", "resource", now, 1); err != nil {
			t.Fatalf("InsertDocumentWithMeta(%s) falhou: %v", p, err)
		}
	}

	// `my_note.md` casa com as duas pastas, mas nunca com myXnote.md.
	assertFoundPaths(t, s, repo, "my_note.md", []string{"notas/my_note.md", "outro/my_note.md"})
	// `%` é curinga puro: sem escape ele puxaria 100x.md também.
	assertFoundPaths(t, s, repo, "100%.md", []string{"notas/100%.md"})
}

func assertFoundPaths(t *testing.T, s *PostgresStore, repo, query string, want []string) {
	t.Helper()
	got, err := s.FindVaultDocumentsByPath(context.Background(), repo, query)
	if err != nil {
		t.Fatalf("FindVaultDocumentsByPath(%q) falhou: %v", query, err)
	}
	if len(got) != len(want) {
		var names []string
		for _, d := range got {
			names = append(names, d.Path)
		}
		t.Errorf("busca %q: esperava %v, obteve %v", query, want, names)
		return
	}
	for i := range want {
		if got[i].Path != want[i] {
			t.Errorf("busca %q posição %d: esperava %q, obteve %q", query, i, want[i], got[i].Path)
		}
	}
}
