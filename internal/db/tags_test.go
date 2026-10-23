package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/store"
)

// newTagsTestDB abre um banco SQLite de verdade (schema aplicado) para os
// testes de tags.
func newTagsTestDB(t *testing.T) *sql.DB {
	t.Helper()

	database, err := InitDB(filepath.Join(t.TempDir(), "tags.db"))
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	return database
}

// insertTagDocument insere um documento mínimo no índice.
func insertTagDocument(t *testing.T, database *sql.DB, id, path, title string) {
	t.Helper()

	_, err := database.Exec(`
		INSERT INTO documents (id, path, title, updated_at, mtime, size)
		VALUES (?, ?, ?, ?, ?, ?)
	`, id, path, title, time.Now().Unix(), int64(100), int64(42))
	if err != nil {
		t.Fatalf("insercao do documento %q falhou: %v", id, err)
	}
}

// countTagRows devolve quantas linhas existem na tabela document_tags.
func countTagRows(t *testing.T, database *sql.DB) int {
	t.Helper()

	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM document_tags`).Scan(&count); err != nil {
		t.Fatalf("contagem de document_tags falhou: %v", err)
	}
	return count
}

// findVaultTag devolve a tag do inventário pela chave canônica.
func findVaultTag(t *testing.T, tags []store.VaultTag, key string) (store.VaultTag, bool) {
	t.Helper()

	for _, tag := range tags {
		if tag.Key == key {
			return tag, true
		}
	}
	return store.VaultTag{}, false
}

func TestInsertDocumentTags_ReplaceInsteadOfAccumulate(t *testing.T) {
	ctx := context.Background()
	database := newTagsTestDB(t)
	insertTagDocument(t, database, "doc-1", "notes/one.md", "Nota Um")

	tags := []store.TagBinding{{Tag: "alfa"}, {Tag: "beta"}}
	if err := InsertDocumentTags(ctx, database, "doc-1", tags); err != nil {
		t.Fatalf("InsertDocumentTags falhou: %v", err)
	}
	if got := countTagRows(t, database); got != 2 {
		t.Fatalf("document_tags = %d linhas; esperava 2", got)
	}

	// Reindexar o mesmo arquivo com as mesmas tags nao pode duplicar.
	if err := InsertDocumentTags(ctx, database, "doc-1", tags); err != nil {
		t.Fatalf("reindexacao falhou: %v", err)
	}
	if got := countTagRows(t, database); got != 2 {
		t.Fatalf("apos reindexar, document_tags = %d linhas; esperava 2", got)
	}

	// Reindexar com outro conjunto de tags substitui: as antigas somem.
	if err := InsertDocumentTags(ctx, database, "doc-1", []store.TagBinding{{Tag: "gama"}}); err != nil {
		t.Fatalf("reindexacao com novas tags falhou: %v", err)
	}
	if got := countTagRows(t, database); got != 1 {
		t.Fatalf("apos trocar as tags, document_tags = %d linhas; esperava 1", got)
	}

	inventory, err := ListVaultTags(ctx, database, store.TagFilter{})
	if err != nil {
		t.Fatalf("ListVaultTags falhou: %v", err)
	}
	if len(inventory) != 1 || inventory[0].Key != "gama" {
		t.Fatalf("inventario = %+v; esperava somente a tag 'gama'", inventory)
	}
}

func TestInsertDocumentTags_CaseInsensitiveKey(t *testing.T) {
	ctx := context.Background()
	database := newTagsTestDB(t)
	insertTagDocument(t, database, "doc-1", "notes/one.md", "Nota Um")

	// `#Tag` e `#TAG` no mesmo lote sao a mesma linha.
	if err := InsertDocumentTags(ctx, database, "doc-1", []store.TagBinding{{Tag: "#Tag"}, {Tag: "#TAG"}}); err != nil {
		t.Fatalf("InsertDocumentTags falhou: %v", err)
	}
	if got := countTagRows(t, database); got != 1 {
		t.Fatalf("document_tags = %d linhas; esperava 1 (#Tag e #TAG sao a mesma linha)", got)
	}

	var tag, tagKey string
	if err := database.QueryRow(`SELECT tag, tag_key FROM document_tags WHERE document_id = 'doc-1'`).Scan(&tag, &tagKey); err != nil {
		t.Fatalf("leitura da linha falhou: %v", err)
	}
	if tagKey != "tag" {
		t.Errorf("tag_key = %q; esperava 'tag' (minusculas)", tagKey)
	}
	if tag != "Tag" {
		t.Errorf("tag = %q; esperava 'Tag' (casing de exibicao do primeiro registro)", tag)
	}

	// Reindexar com o outro casing continua sendo a mesma linha.
	if err := InsertDocumentTags(ctx, database, "doc-1", []store.TagBinding{{Tag: "#TAG"}}); err != nil {
		t.Fatalf("reindexacao com casing diferente falhou: %v", err)
	}
	if got := countTagRows(t, database); got != 1 {
		t.Fatalf("apos reindexar com casing diferente, document_tags = %d linhas; esperava 1", got)
	}
}

func TestInsertDocumentTags_CanonicalizesKey(t *testing.T) {
	ctx := context.Background()
	database := newTagsTestDB(t)
	insertTagDocument(t, database, "doc-1", "notes/one.md", "Nota Um")

	if err := InsertDocumentTags(ctx, database, "doc-1", []store.TagBinding{{Tag: "  #Tag  "}}); err != nil {
		t.Fatalf("InsertDocumentTags falhou: %v", err)
	}

	var tagKey string
	if err := database.QueryRow(`SELECT tag_key FROM document_tags WHERE document_id = 'doc-1'`).Scan(&tagKey); err != nil {
		t.Fatalf("leitura da linha falhou: %v", err)
	}
	if tagKey != "tag" {
		t.Errorf("tag_key = %q; esperava 'tag' (sem # e sem espacos)", tagKey)
	}
}

func TestDeleteDocumentTags(t *testing.T) {
	ctx := context.Background()
	database := newTagsTestDB(t)
	insertTagDocument(t, database, "doc-1", "notes/one.md", "Nota Um")

	if err := InsertDocumentTags(ctx, database, "doc-1", []store.TagBinding{{Tag: "alfa"}}); err != nil {
		t.Fatalf("InsertDocumentTags falhou: %v", err)
	}
	if err := DeleteDocumentTags(ctx, database, "doc-1"); err != nil {
		t.Fatalf("DeleteDocumentTags falhou: %v", err)
	}
	if got := countTagRows(t, database); got != 0 {
		t.Fatalf("document_tags = %d linhas; esperava 0", got)
	}
}

func TestFindVaultDocumentsByTag_MatchesDescendants(t *testing.T) {
	ctx := context.Background()
	database := newTagsTestDB(t)
	insertTagDocument(t, database, "doc-1", "notes/pai.md", "Pai")
	insertTagDocument(t, database, "doc-2", "notes/filho.md", "Filho")
	insertTagDocument(t, database, "doc-3", "outros/r.md", "Outro")

	if err := InsertDocumentTags(ctx, database, "doc-1", []store.TagBinding{{Tag: "a"}}); err != nil {
		t.Fatalf("InsertDocumentTags falhou: %v", err)
	}
	if err := InsertDocumentTags(ctx, database, "doc-2", []store.TagBinding{{Tag: "a/b"}}); err != nil {
		t.Fatalf("InsertDocumentTags falhou: %v", err)
	}
	if err := InsertDocumentTags(ctx, database, "doc-3", []store.TagBinding{{Tag: "c"}}); err != nil {
		t.Fatalf("InsertDocumentTags falhou: %v", err)
	}

	// `a` casa `a` e `a/b`.
	documents, err := FindVaultDocumentsByTag(ctx, database, "a")
	if err != nil {
		t.Fatalf("FindVaultDocumentsByTag falhou: %v", err)
	}
	if len(documents) != 2 {
		t.Fatalf("busca por 'a' devolveu %d documentos; esperava 2 (a e a/b)", len(documents))
	}
	if documents[0].Path != "notes/filho.md" || documents[1].Path != "notes/pai.md" {
		t.Errorf("busca por 'a' devolveu %q e %q; esperava ordenacao por path", documents[0].Path, documents[1].Path)
	}

	// `a/b` casa so a descendente.
	documents, err = FindVaultDocumentsByTag(ctx, database, "a/b")
	if err != nil {
		t.Fatalf("FindVaultDocumentsByTag falhou: %v", err)
	}
	if len(documents) != 1 || documents[0].Path != "notes/filho.md" {
		t.Fatalf("busca por 'a/b' devolveu %+v; esperava so notes/filho.md", documents)
	}
}

func TestFindVaultDocumentsByTag_EscapesLikeWildcards(t *testing.T) {
	ctx := context.Background()
	database := newTagsTestDB(t)
	insertTagDocument(t, database, "doc-1", "notes/underscore.md", "Underscore")
	insertTagDocument(t, database, "doc-2", "notes/wildcard.md", "Wildcard")
	insertTagDocument(t, database, "doc-3", "notes/percent.md", "Percent")

	if err := InsertDocumentTags(ctx, database, "doc-1", []store.TagBinding{{Tag: "a_b"}}); err != nil {
		t.Fatalf("InsertDocumentTags falhou: %v", err)
	}
	if err := InsertDocumentTags(ctx, database, "doc-2", []store.TagBinding{{Tag: "aXb"}}); err != nil {
		t.Fatalf("InsertDocumentTags falhou: %v", err)
	}
	if err := InsertDocumentTags(ctx, database, "doc-3", []store.TagBinding{{Tag: "p%q"}}); err != nil {
		t.Fatalf("InsertDocumentTags falhou: %v", err)
	}

	// `_` e curinga do LIKE: `a_b` nao pode casar `aXb`.
	documents, err := FindVaultDocumentsByTag(ctx, database, "a_b")
	if err != nil {
		t.Fatalf("FindVaultDocumentsByTag falhou: %v", err)
	}
	if len(documents) != 1 || documents[0].Path != "notes/underscore.md" {
		t.Fatalf("busca por 'a_b' devolveu %+v; esperava so notes/underscore.md", documents)
	}

	// `%` tambem e curinga do LIKE.
	documents, err = FindVaultDocumentsByTag(ctx, database, "p%q")
	if err != nil {
		t.Fatalf("FindVaultDocumentsByTag falhou: %v", err)
	}
	if len(documents) != 1 || documents[0].Path != "notes/percent.md" {
		t.Fatalf("busca por 'p%%q' devolveu %+v; esperava so notes/percent.md", documents)
	}
}

func TestFindVaultDocumentsByTag_CanonicalizesInput(t *testing.T) {
	ctx := context.Background()
	database := newTagsTestDB(t)
	insertTagDocument(t, database, "doc-1", "notes/one.md", "Nota Um")

	if err := InsertDocumentTags(ctx, database, "doc-1", []store.TagBinding{{Tag: "#Tag"}}); err != nil {
		t.Fatalf("InsertDocumentTags falhou: %v", err)
	}

	for _, input := range []string{"Tag", "#Tag", "tag", " tag ", "#TAG"} {
		documents, err := FindVaultDocumentsByTag(ctx, database, input)
		if err != nil {
			t.Fatalf("FindVaultDocumentsByTag(%q) falhou: %v", input, err)
		}
		if len(documents) != 1 {
			t.Fatalf("FindVaultDocumentsByTag(%q) devolveu %d documentos; esperava 1", input, len(documents))
		}
	}
}

func TestListVaultTags_CountsDistinctDocuments(t *testing.T) {
	ctx := context.Background()
	database := newTagsTestDB(t)
	insertTagDocument(t, database, "doc-1", "notes/one.md", "Nota Um")
	insertTagDocument(t, database, "doc-2", "notes/two.md", "Nota Dois")
	insertTagDocument(t, database, "doc-3", "notes/three.md", "Nota Tres")

	if err := InsertDocumentTags(ctx, database, "doc-1", []store.TagBinding{{Tag: "comum"}, {Tag: "so-um"}}); err != nil {
		t.Fatalf("InsertDocumentTags falhou: %v", err)
	}
	if err := InsertDocumentTags(ctx, database, "doc-2", []store.TagBinding{{Tag: "comum"}}); err != nil {
		t.Fatalf("InsertDocumentTags falhou: %v", err)
	}
	if err := InsertDocumentTags(ctx, database, "doc-3", []store.TagBinding{{Tag: "comum"}}); err != nil {
		t.Fatalf("InsertDocumentTags falhou: %v", err)
	}

	inventory, err := ListVaultTags(ctx, database, store.TagFilter{})
	if err != nil {
		t.Fatalf("ListVaultTags falhou: %v", err)
	}
	if len(inventory) != 2 {
		t.Fatalf("inventario = %+v; esperava 2 tags", inventory)
	}

	comum, ok := findVaultTag(t, inventory, "comum")
	if !ok {
		t.Fatalf("inventario = %+v; esperava a tag 'comum'", inventory)
	}
	if comum.Count != 3 {
		t.Errorf("count de 'comum' = %d; esperava 3", comum.Count)
	}

	soUm, ok := findVaultTag(t, inventory, "so-um")
	if !ok {
		t.Fatalf("inventario = %+v; esperava a tag 'so-um'", inventory)
	}
	if soUm.Count != 1 {
		t.Errorf("count de 'so-um' = %d; esperava 1", soUm.Count)
	}
}

func TestListVaultTags_OrderedByKey(t *testing.T) {
	ctx := context.Background()
	database := newTagsTestDB(t)
	insertTagDocument(t, database, "doc-1", "notes/one.md", "Nota Um")

	if err := InsertDocumentTags(ctx, database, "doc-1", []store.TagBinding{{Tag: "zeta"}, {Tag: "alfa"}, {Tag: "meio"}}); err != nil {
		t.Fatalf("InsertDocumentTags falhou: %v", err)
	}

	inventory, err := ListVaultTags(ctx, database, store.TagFilter{})
	if err != nil {
		t.Fatalf("ListVaultTags falhou: %v", err)
	}
	if len(inventory) != 3 {
		t.Fatalf("inventario = %+v; esperava 3 tags", inventory)
	}
	if inventory[0].Key != "alfa" || inventory[1].Key != "meio" || inventory[2].Key != "zeta" {
		t.Errorf("ordem do inventario = %q, %q, %q; esperava alfa, meio, zeta", inventory[0].Key, inventory[1].Key, inventory[2].Key)
	}
}

func TestListVaultTags_Filters(t *testing.T) {
	ctx := context.Background()
	database := newTagsTestDB(t)
	insertTagDocument(t, database, "doc-1", "notes/one.md", "Nota Um")
	insertTagDocument(t, database, "doc-2", "notes/sub/two.md", "Nota Dois")
	insertTagDocument(t, database, "doc-3", "outros/three.md", "Nota Tres")

	if err := InsertDocumentTags(ctx, database, "doc-1", []store.TagBinding{{Tag: "tag-um"}}); err != nil {
		t.Fatalf("InsertDocumentTags falhou: %v", err)
	}
	if err := InsertDocumentTags(ctx, database, "doc-2", []store.TagBinding{{Tag: "tag-dois"}}); err != nil {
		t.Fatalf("InsertDocumentTags falhou: %v", err)
	}
	if err := InsertDocumentTags(ctx, database, "doc-3", []store.TagBinding{{Tag: "tag-tres"}}); err != nil {
		t.Fatalf("InsertDocumentTags falhou: %v", err)
	}

	// Filtro por documento.
	inventory, err := ListVaultTags(ctx, database, store.TagFilter{DocumentID: "doc-2"})
	if err != nil {
		t.Fatalf("ListVaultTags falhou: %v", err)
	}
	if len(inventory) != 1 || inventory[0].Key != "tag-dois" {
		t.Fatalf("filtro por documento = %+v; esperava so 'tag-dois'", inventory)
	}

	// Filtro por pasta: "notes" inclui "notes/sub".
	inventory, err = ListVaultTags(ctx, database, store.TagFilter{Folder: "notes"})
	if err != nil {
		t.Fatalf("ListVaultTags falhou: %v", err)
	}
	if len(inventory) != 2 {
		t.Fatalf("filtro por pasta = %+v; esperava 2 tags", inventory)
	}
	if _, ok := findVaultTag(t, inventory, "tag-tres"); ok {
		t.Errorf("filtro por pasta incluiu 'tag-tres', que esta fora de 'notes'")
	}

	// Filtro por pasta com curinga no nome: "outros" nao casa "out%ros".
	inventory, err = ListVaultTags(ctx, database, store.TagFilter{Folder: "outros"})
	if err != nil {
		t.Fatalf("ListVaultTags falhou: %v", err)
	}
	if len(inventory) != 1 || inventory[0].Key != "tag-tres" {
		t.Fatalf("filtro por pasta 'outros' = %+v; esperava so 'tag-tres'", inventory)
	}
}

// TestListVaultTags_FolderAceitaPathWindows cobre o path indexado no Windows.
//
// O walker grava o caminho com o separador da plataforma, e no Windows esse
// separador é "\". A comparação por pasta é feita sobre a forma com "/", então
// um path com backslash precisa casar do mesmo jeito — e a pasta precisa ser
// um segmento inteiro: "docs" não casa "docs-outro".
//
// Os fixtures da suíte usam paths com "/" (o caminho relativo que o Linux
// produz), então sem este teste o filtro por pasta nunca era exercitado com
// o separador do Windows — que é justamente o SO em que o bug aparecia.
func TestListVaultTags_FolderAceitaPathWindows(t *testing.T) {
	ctx := context.Background()
	database := newTagsTestDB(t)
	insertTagDocument(t, database, "doc-1", `vault\notas\sub\um.md`, "Nota Um")
	insertTagDocument(t, database, "doc-2", `vault\docs-outro\x.md`, "Outro")
	insertTagDocument(t, database, "doc-3", `outros\y.md`, "Y")

	if err := InsertDocumentTags(ctx, database, "doc-1", []store.TagBinding{{Tag: "tag-um"}}); err != nil {
		t.Fatalf("InsertDocumentTags falhou: %v", err)
	}
	if err := InsertDocumentTags(ctx, database, "doc-2", []store.TagBinding{{Tag: "tag-docs"}}); err != nil {
		t.Fatalf("InsertDocumentTags falhou: %v", err)
	}
	if err := InsertDocumentTags(ctx, database, "doc-3", []store.TagBinding{{Tag: "tag-y"}}); err != nil {
		t.Fatalf("InsertDocumentTags falhou: %v", err)
	}

	// A pasta "notas" casa o path com backslash.
	inventory, err := ListVaultTags(ctx, database, store.TagFilter{Folder: "notas"})
	if err != nil {
		t.Fatalf("ListVaultTags falhou: %v", err)
	}
	if len(inventory) != 1 || inventory[0].Key != "tag-um" {
		t.Fatalf("filtro por pasta 'notas' = %+v; esperava so 'tag-um'", inventory)
	}

	// O filtro também aceita a pasta escrita com o separador do Windows.
	inventory, err = ListVaultTags(ctx, database, store.TagFilter{Folder: `notas\sub`})
	if err != nil {
		t.Fatalf("ListVaultTags falhou: %v", err)
	}
	if len(inventory) != 1 || inventory[0].Key != "tag-um" {
		t.Fatalf("filtro por pasta 'notas\\sub' = %+v; esperava so 'tag-um'", inventory)
	}

	// "docs" é um segmento inteiro: não casa "docs-outro".
	inventory, err = ListVaultTags(ctx, database, store.TagFilter{Folder: "docs"})
	if err != nil {
		t.Fatalf("ListVaultTags falhou: %v", err)
	}
	if len(inventory) != 0 {
		t.Fatalf("filtro por pasta 'docs' = %+v; esperava vazio ('docs' não casa 'docs-outro')", inventory)
	}
}

func TestListVaultTags_DisplayCasingIsDeterministic(t *testing.T) {
	ctx := context.Background()
	database := newTagsTestDB(t)
	insertTagDocument(t, database, "doc-a", "notes/a.md", "Nota A")
	insertTagDocument(t, database, "doc-b", "notes/b.md", "Nota B")

	if err := InsertDocumentTags(ctx, database, "doc-a", []store.TagBinding{{Tag: "#Zeta"}}); err != nil {
		t.Fatalf("InsertDocumentTags falhou: %v", err)
	}
	if err := InsertDocumentTags(ctx, database, "doc-b", []store.TagBinding{{Tag: "#zeta"}}); err != nil {
		t.Fatalf("InsertDocumentTags falhou: %v", err)
	}

	// created_at fixado: o teste nao depende do relogio.
	if _, err := database.Exec(`UPDATE document_tags SET created_at = 100 WHERE document_id = 'doc-a'`); err != nil {
		t.Fatalf("fixacao de created_at falhou: %v", err)
	}
	if _, err := database.Exec(`UPDATE document_tags SET created_at = 200 WHERE document_id = 'doc-b'`); err != nil {
		t.Fatalf("fixacao de created_at falhou: %v", err)
	}

	inventory, err := ListVaultTags(ctx, database, store.TagFilter{})
	if err != nil {
		t.Fatalf("ListVaultTags falhou: %v", err)
	}
	if len(inventory) != 1 {
		t.Fatalf("inventario = %+v; esperava 1 tag", inventory)
	}
	if inventory[0].Tag != "Zeta" {
		t.Errorf("tag de exibicao = %q; esperava 'Zeta' (created_at menor)", inventory[0].Tag)
	}
	if inventory[0].Count != 2 {
		t.Errorf("count = %d; esperava 2", inventory[0].Count)
	}

	// Empate de created_at: desempate pelo document_id.
	if _, err := database.Exec(`UPDATE document_tags SET created_at = 100`); err != nil {
		t.Fatalf("fixacao de created_at falhou: %v", err)
	}
	inventory, err = ListVaultTags(ctx, database, store.TagFilter{})
	if err != nil {
		t.Fatalf("ListVaultTags falhou: %v", err)
	}
	if inventory[0].Tag != "Zeta" {
		t.Errorf("tag de exibicao = %q; esperava 'Zeta' (document_id menor no empate)", inventory[0].Tag)
	}
}

func TestListVaultTags_ResolvedDocument(t *testing.T) {
	ctx := context.Background()
	database := newTagsTestDB(t)
	insertTagDocument(t, database, "doc-1", "notes/one.md", "Nota Um")
	insertTagDocument(t, database, "adr-1", "docs/adr-001.md", "ADR 001")

	if err := InsertDocumentTags(ctx, database, "doc-1", []store.TagBinding{{Tag: "ADR-001", ResolvedDocID: "adr-1"}}); err != nil {
		t.Fatalf("InsertDocumentTags falhou: %v", err)
	}

	inventory, err := ListVaultTags(ctx, database, store.TagFilter{})
	if err != nil {
		t.Fatalf("ListVaultTags falhou: %v", err)
	}
	if len(inventory) != 1 {
		t.Fatalf("inventario = %+v; esperava 1 tag", inventory)
	}
	if inventory[0].ResolvedDocID != "adr-1" {
		t.Errorf("resolved_doc_id = %q; esperava 'adr-1'", inventory[0].ResolvedDocID)
	}
	if inventory[0].ResolvedTitle != "ADR 001" {
		t.Errorf("resolved_title = %q; esperava 'ADR 001'", inventory[0].ResolvedTitle)
	}
}

func TestListDocumentIDTitles(t *testing.T) {
	ctx := context.Background()
	database := newTagsTestDB(t)
	insertTagDocument(t, database, "doc-1", "notes/one.md", "Nota Um")

	// Documento sem título: o fallback é o próprio id.
	if _, err := database.Exec(`
		INSERT INTO documents (id, path, updated_at, mtime, size)
		VALUES ('doc-2', 'notes/two.md', ?, ?, ?)
	`, time.Now().Unix(), int64(100), int64(42)); err != nil {
		t.Fatalf("insercao do documento sem titulo falhou: %v", err)
	}

	titles, err := ListDocumentIDTitles(ctx, database)
	if err != nil {
		t.Fatalf("ListDocumentIDTitles falhou: %v", err)
	}
	if len(titles) != 2 {
		t.Fatalf("mapa = %+v; esperava 2 entradas", titles)
	}
	if titles["Nota Um"] != "doc-1" {
		t.Errorf("mapa['Nota Um'] = %q; esperava 'doc-1'", titles["Nota Um"])
	}
	// Sem título, o fallback é o próprio id.
	if titles["doc-2"] != "doc-2" {
		t.Errorf("mapa['doc-2'] = %q; esperava 'doc-2'", titles["doc-2"])
	}
}

func TestDocumentTagsTable_CreatedOnLegacyDatabase(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy.db")

	rawDB, err := openDB(dbPath)
	if err != nil {
		t.Fatalf("openDB falhou: %v", err)
	}

	legacySchema := `
	CREATE TABLE documents (
		id TEXT PRIMARY KEY,
		path TEXT NOT NULL UNIQUE,
		title TEXT,
		updated_at INTEGER NOT NULL,
		content_hash TEXT
	);
	`
	if _, err := rawDB.Exec(legacySchema); err != nil {
		t.Fatalf("Exec legacySchema falhou: %v", err)
	}
	if _, err := rawDB.Exec(`
		INSERT INTO documents (id, path, title, updated_at, content_hash)
		VALUES ('legacy-doc', 'docs/legacy.md', 'Legacy Title', 1000, 'hash-legacy')
	`); err != nil {
		t.Fatalf("Insert documento legado falhou: %v", err)
	}
	rawDB.Close()

	database, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB sobre banco legado falhou: %v", err)
	}
	defer database.Close()

	var count int
	if err := database.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'document_tags'
	`).Scan(&count); err != nil {
		t.Fatalf("consulta em sqlite_master falhou: %v", err)
	}
	if count != 1 {
		t.Fatalf("tabela document_tags nao foi criada ao abrir o banco legado")
	}

	// E a tabela funciona: nao basta existir.
	if err := InsertDocumentTags(context.Background(), database, "legacy-doc", []store.TagBinding{{Tag: "legado"}}); err != nil {
		t.Fatalf("InsertDocumentTags no banco legado falhou: %v", err)
	}
	inventory, err := ListVaultTags(context.Background(), database, store.TagFilter{})
	if err != nil {
		t.Fatalf("ListVaultTags no banco legado falhou: %v", err)
	}
	if len(inventory) != 1 || inventory[0].Key != "legado" {
		t.Fatalf("inventario = %+v; esperava a tag 'legado'", inventory)
	}
}

// TestListVaultTags_DanglingResolvedDocIsCleared cobre o caso em que o
// documento resolvido foi removido depois que a tag foi gravada.
//
// A chave de `document_tags` nao e uma foreign key (a identidade do documento no
// Postgres e `(repository, id)`), entao a linha da tag sobrevive ao documento.
// O id pendurado nao pode vazar para a saida: o consumidor abriria um
// documento que nao existe mais.
func TestListVaultTags_DanglingResolvedDocIsCleared(t *testing.T) {
	ctx := context.Background()
	database := newTagsTestDB(t)
	insertTagDocument(t, database, "doc-1", "notes/one.md", "Nota Um")
	insertTagDocument(t, database, "adr-1", "docs/adr-001.md", "ADR 001")

	if err := InsertDocumentTags(ctx, database, "doc-1", []store.TagBinding{{Tag: "ADR-001", ResolvedDocID: "adr-1"}}); err != nil {
		t.Fatalf("InsertDocumentTags falhou: %v", err)
	}

	// O documento resolvido some do indice; a linha da tag fica.
	if _, err := database.Exec(`DELETE FROM documents WHERE id = ?`, "adr-1"); err != nil {
		t.Fatalf("remocao do documento falhou: %v", err)
	}

	inventory, err := ListVaultTags(ctx, database, store.TagFilter{})
	if err != nil {
		t.Fatalf("ListVaultTags falhou: %v", err)
	}
	if len(inventory) != 1 {
		t.Fatalf("inventario = %+v; esperava 1 tag", inventory)
	}
	if inventory[0].ResolvedDocID != "" {
		t.Errorf("resolved_doc_id = %q; esperava vazio (o documento foi removido)", inventory[0].ResolvedDocID)
	}
	if inventory[0].ResolvedTitle != "" {
		t.Errorf("resolved_title = %q; esperava vazio (o documento foi removido)", inventory[0].ResolvedTitle)
	}
}
