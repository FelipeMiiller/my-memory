package db

import (
	"context"
	"path/filepath"
	"testing"
)

// TestFindVaultDocumentsByPath_IgnoraCuringasNoNome prova que o basename é
// comparado literalmente: `_` e `%` no nome não viram curingas do LIKE.
// Sem o escape, `mem read my_note.md` casaria com `myXnote.md` e o comando
// reportaria ambiguidade falsa.
func TestFindVaultDocumentsByPath_IgnoraCuringasNoNome(t *testing.T) {
	database, err := InitDB(filepath.Join(t.TempDir(), "vault_like.db"))
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	docs := []string{"notas/my_note.md", "notas/myXnote.md", "outro/my_note.md", "notas/100%.md", "notas/100x.md"}
	for _, p := range docs {
		if err := InsertDocumentWithMeta(ctx, database, p, p, p, 1_700_000_000, "h-"+p, "", "", "resource", 1_700_000_000, 1); err != nil {
			t.Fatalf("InsertDocumentWithMeta(%s) falhou: %v", p, err)
		}
	}

	cases := []struct {
		query string
		want  []string
	}{
		// `_` casa qualquer caractere: sem escape também traria myXnote.md.
		{"my_note.md", []string{"notas/my_note.md", "outro/my_note.md"}},
		// `%` casa qualquer sequência: sem escape também traria 100x.md.
		{"100%.md", []string{"notas/100%.md"}},
		// Um basename simples continua casando em qualquer pasta.
		{"myXnote.md", []string{"notas/myXnote.md"}},
	}

	for _, c := range cases {
		got, err := FindVaultDocumentsByPath(ctx, database, c.query)
		if err != nil {
			t.Fatalf("FindVaultDocumentsByPath(%q) falhou: %v", c.query, err)
		}
		if len(got) != len(c.want) {
			var names []string
			for _, d := range got {
				names = append(names, d.Path)
			}
			t.Errorf("busca %q: esperava %v, obteve %v", c.query, c.want, names)
			continue
		}
		for i := range c.want {
			if got[i].Path != c.want[i] {
				t.Errorf("busca %q posição %d: esperava %q, obteve %q", c.query, i, c.want[i], got[i].Path)
			}
		}
	}
}
