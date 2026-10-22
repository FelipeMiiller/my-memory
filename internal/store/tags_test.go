package store

import (
	"context"
	"strings"
	"testing"
)

// As assinaturas da API de tags do Postgres são fixadas aqui: se alguém
// mudá-las, o teste não compila.
var (
	_ func(*PostgresStore, context.Context, string, string, []TagBinding) error      = (*PostgresStore).InsertDocumentTags
	_ func(*PostgresStore, context.Context, string, string) error                    = (*PostgresStore).DeleteDocumentTags
	_ func(*PostgresStore, context.Context, string, TagFilter) ([]VaultTag, error)   = (*PostgresStore).ListVaultTags
	_ func(*PostgresStore, context.Context, string, string) ([]VaultDocument, error) = (*PostgresStore).FindVaultDocumentsByTag
	_ func(*PostgresStore, context.Context, string) (map[string]string, error)       = (*PostgresStore).ListDocumentIDTitles
)

func TestPostgresSchema_DocumentTags(t *testing.T) {
	if !strings.Contains(PostgresSchema, "CREATE TABLE IF NOT EXISTS document_tags") {
		t.Fatalf("PostgresSchema deve conter a tabela document_tags")
	}
	if !strings.Contains(PostgresSchema, "PRIMARY KEY (repository, document_id, tag_key)") {
		t.Fatalf("PostgresSchema deve conter a primary key (repository, document_id, tag_key)")
	}
	if !strings.Contains(PostgresSchema, "CREATE INDEX IF NOT EXISTS idx_document_tags_key ON document_tags(repository, tag_key)") {
		t.Fatalf("PostgresSchema deve conter o indice por (repository, tag_key)")
	}
	if !strings.Contains(PostgresSchema, "CREATE INDEX IF NOT EXISTS idx_document_tags_document ON document_tags(repository, document_id)") {
		t.Fatalf("PostgresSchema deve conter o indice por (repository, document_id)")
	}
	if !strings.Contains(PostgresSchema, "created_at    BIGINT NOT NULL DEFAULT 0") {
		t.Fatalf("PostgresSchema deve conter created_at BIGINT NOT NULL DEFAULT 0")
	}
	if !strings.Contains(PostgresSchema, "resolved_doc_id TEXT") {
		t.Fatalf("PostgresSchema deve conter resolved_doc_id TEXT")
	}
}

func TestCanonicalTagKey(t *testing.T) {
	cases := map[string]string{
		"Tag":      "tag",
		"#Tag":     "tag",
		"#TAG":     "tag",
		"tag":      "tag",
		"  #Tag  ": "tag",
		"a/b":      "a/b",
		"#a/b":     "a/b",
		"":         "",
		"#":        "",
	}

	for input, expected := range cases {
		if got := CanonicalTagKey(input); got != expected {
			t.Errorf("CanonicalTagKey(%q) = %q; esperava %q", input, got, expected)
		}
	}
}

func TestDisplayTag(t *testing.T) {
	cases := map[string]string{
		"Tag":      "Tag",
		"#Tag":     "Tag",
		"#TAG":     "TAG",
		"  #Tag  ": "Tag",
		"":         "",
	}

	for input, expected := range cases {
		if got := DisplayTag(input); got != expected {
			t.Errorf("DisplayTag(%q) = %q; esperava %q", input, got, expected)
		}
	}
}

func TestBuildTagInsertBatch_DeduplicatesByKey(t *testing.T) {
	batch := buildTagInsertBatch([]TagBinding{
		{Tag: "#Tag"},
		{Tag: "#TAG"},
		{Tag: "Outro"},
		{Tag: "   "},
	})

	if len(batch.rows) != 2 {
		t.Fatalf("lote = %+v; esperava 2 linhas", batch.rows)
	}
	if batch.rows[0].Key != "tag" || batch.rows[0].Tag != "Tag" {
		t.Errorf("primeira linha = %+v; esperava key 'tag' e display 'Tag'", batch.rows[0])
	}
	if batch.rows[1].Key != "outro" {
		t.Errorf("segunda linha = %+v; esperava key 'outro'", batch.rows[1])
	}
}

func TestEscapeLikePattern_EscapesWildcards(t *testing.T) {
	if got := EscapeLikePattern("a_b"); got != `a\_b` {
		t.Errorf("EscapeLikePattern(\"a_b\") = %q; esperava %q", got, `a\_b`)
	}
	if got := EscapeLikePattern("p%q"); got != `p\%q` {
		t.Errorf("EscapeLikePattern(\"p%%q\") = %q; esperava %q", got, `p\%q`)
	}
	if got := EscapeLikePattern(`a\b`); got != `a\\b` {
		t.Errorf("EscapeLikePattern(\"a\\b\") = %q; esperava %q", got, `a\\b`)
	}
}

func TestVaultTagAndTagFilterStructs(t *testing.T) {
	// A forma de exibicao e a chave canonica sao campos distintos.
	tag := VaultTag{Tag: "Tag", Key: "tag", Count: 2, ResolvedDocID: "doc-1", ResolvedTitle: "Nota"}
	if tag.Tag != "Tag" || tag.Key != "tag" {
		t.Errorf("VaultTag = %+v; esperava display 'Tag' e key 'tag'", tag)
	}

	filter := TagFilter{DocumentID: "doc-1", Folder: "notes"}
	if filter.DocumentID != "doc-1" || filter.Folder != "notes" {
		t.Errorf("TagFilter = %+v; esperava doc-1/notes", filter)
	}

	binding := TagBinding{Tag: "Tag", Key: "tag", ResolvedDocID: "doc-1"}
	if binding.ResolvedDocID != "doc-1" {
		t.Errorf("TagBinding = %+v; esperava resolved_doc_id 'doc-1'", binding)
	}
}
