package main

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// searchTagFixtureDoc é um documento do fixture de busca por tag: o path, o
// conteúdo indexado, as tags que ele carrega e a taxonomia.
type searchTagFixtureDoc struct {
	path     string
	content  string
	tags     []string
	category string
}

// newSearchTagFixture cria um vault SQLite com documentos, chunks e tags, e
// devolve o caminho do arquivo e a conexão aberta (a busca precisa do índice
// vivo).
func newSearchTagFixture(t *testing.T, docs ...searchTagFixtureDoc) (string, *sql.DB) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "vault_search_tag_fixture.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	ctx := context.Background()
	for _, doc := range docs {
		category := doc.category
		if category == "" {
			category = "resource"
		}
		title := strings.TrimSuffix(filepath.Base(doc.path), filepath.Ext(doc.path))
		err := db.InsertDocumentWithMeta(ctx, database, doc.path, doc.path, title, 1700000000, "hash-"+doc.path, "epoch", "", category, 1700000000, 128)
		if err != nil {
			t.Fatalf("InsertDocumentWithMeta(%s) falhou: %v", doc.path, err)
		}

		chunkID := doc.path + "#0"
		if err := db.InsertChunk(ctx, database, chunkID, doc.path, doc.content, 0, nil); err != nil {
			t.Fatalf("InsertChunk(%s) falhou: %v", chunkID, err)
		}

		bindings := make([]store.TagBinding, 0, len(doc.tags))
		for _, tag := range doc.tags {
			bindings = append(bindings, store.TagBinding{Tag: tag})
		}
		if err := db.InsertDocumentTags(ctx, database, doc.path, bindings); err != nil {
			t.Fatalf("InsertDocumentTags(%s) falhou: %v", doc.path, err)
		}
	}

	return dbPath, database
}

// defaultSearchTagFixture é o vault usado pelos testes: três documentos que
// falam de "arquitetura", com tags distintas (inclusive uma hierárquica) e
// duas taxonomias.
func defaultSearchTagFixture(t *testing.T) (string, *sql.DB) {
	t.Helper()
	return newSearchTagFixture(t,
		searchTagFixtureDoc{path: "docs/alpha.md", content: "arquitetura do sistema alpha", tags: []string{"#inbox", "#projeto/alpha"}},
		searchTagFixtureDoc{path: "docs/beta.md", content: "arquitetura do sistema beta", tags: []string{"#comum"}, category: "memory"},
		searchTagFixtureDoc{path: "docs/gamma.md", content: "arquitetura do sistema gamma", tags: []string{"#inbox/to-read"}},
	)
}

// runSearchFixture executa a busca por FTS contra o fixture e devolve a saída
// impressa na tela.
func runSearchFixture(t *testing.T, database *sql.DB, searchOpts store.SearchOptions, query string) string {
	t.Helper()

	return captureStdout(t, func() {
		runSearchSQLite(context.Background(), database, nil, nil, query, "fts", false, 10, 60, store.DefaultDecayOptions(), searchOpts)
	})
}

// TestSearchComTagFiltraDocumentos: só os documentos que carregam a tag
// (e as descendentes) entram no resultado.
func TestSearchComTagFiltraDocumentos(t *testing.T) {
	_, database := defaultSearchTagFixture(t)

	out := runSearchFixture(t, database, store.SearchOptions{Level: "l1", Tag: "inbox"}, "arquitetura")

	if !strings.Contains(out, "docs/alpha.md") {
		t.Errorf("esperava o documento com a tag #inbox, obteve:\n%s", out)
	}
	if strings.Contains(out, "docs/beta.md") {
		t.Errorf("o filtro por tag trouxe um documento sem a tag:\n%s", out)
	}
}

// TestSearchComTagCasaDescendentes: a tag casa as descendentes, como no
// Obsidian (`inbox` casa `inbox/to-read`).
func TestSearchComTagCasaDescendentes(t *testing.T) {
	_, database := defaultSearchTagFixture(t)

	out := runSearchFixture(t, database, store.SearchOptions{Level: "l1", Tag: "inbox"}, "arquitetura")

	if !strings.Contains(out, "docs/gamma.md") {
		t.Errorf("esperava o documento com a tag descendente #inbox/to-read, obteve:\n%s", out)
	}
}

// TestSearchComTagCombinaComCategory: --tag e --category se combinam — o
// documento precisa passar pelos dois filtros.
func TestSearchComTagCombinaComCategory(t *testing.T) {
	_, database := defaultSearchTagFixture(t)

	out := runSearchFixture(t, database, store.SearchOptions{Level: "l1", Tag: "inbox", Category: "memory"}, "arquitetura")

	if strings.Contains(out, "docs/alpha.md") {
		t.Errorf("o filtro por categoria deixou passar um documento de outra taxonomia:\n%s", out)
	}
	if strings.Contains(out, "docs/gamma.md") {
		t.Errorf("o filtro por taxonomia deixou passar um documento de outra taxonomia:\n%s", out)
	}
}

// TestSearchComTagSemNotaDevolveVazio: tag que ninguém carrega é resultado
// vazio — e não erro.
func TestSearchComTagSemNotaDevolveVazio(t *testing.T) {
	_, database := defaultSearchTagFixture(t)

	out := runSearchFixture(t, database, store.SearchOptions{Level: "l1", Tag: "inexistente"}, "arquitetura")

	if strings.Contains(out, "docs/alpha.md") || strings.Contains(out, "docs/beta.md") || strings.Contains(out, "docs/gamma.md") {
		t.Errorf("tag sem nota não deveria trazer documento algum, obteve:\n%s", out)
	}
	if strings.Contains(out, "Erro") {
		t.Errorf("tag sem nota não é erro, obteve:\n%s", out)
	}
}

// TestSearchComTagSemValorEhErroDeUso: `--tag` sem valor imprime o uso e sai
// com código diferente de zero.
func TestSearchComTagSemValorEhErroDeUso(t *testing.T) {
	searchCmd, _ := newSearchFlagSet()
	searchCmd.SetOutput(io.Discard)

	var buf bytes.Buffer
	code := parseSearchFlags(searchCmd, []string{"--tag"}, &buf)

	if code == 0 {
		t.Errorf("--tag sem valor deveria sair com código diferente de zero")
	}
	if !strings.Contains(buf.String(), "Uso: mem search") {
		t.Errorf("esperava o uso na saída de erro, obteve:\n%s", buf.String())
	}
}

// TestSearchComTagValidoNaoEhErroDeUso: o parse de --tag com valor funciona e
// devolve código zero.
func TestSearchComTagValidoNaoEhErroDeUso(t *testing.T) {
	searchCmd, sf := newSearchFlagSet()
	searchCmd.SetOutput(io.Discard)

	var buf bytes.Buffer
	code := parseSearchFlags(searchCmd, []string{"--tag", "inbox", "--category", "memory", "arquitetura"}, &buf)

	if code != 0 {
		t.Errorf("esperava código zero, obteve %d (%s)", code, buf.String())
	}
	if *sf.tag != "inbox" {
		t.Errorf("esperava a tag 'inbox', obteve '%s'", *sf.tag)
	}
	if *sf.category != "memory" {
		t.Errorf("esperava a categoria 'memory', obteve '%s'", *sf.category)
	}
	if len(searchCmd.Args()) != 1 || searchCmd.Args()[0] != "arquitetura" {
		t.Errorf("esperava a query como argumento posicional, obteve %v", searchCmd.Args())
	}
}

// TestSearchComTagDepoisDaQuery: a flag pode vir depois do termo — o
// rearranjador de argumentos precisa tratar `--tag` como flag com valor.
func TestSearchComTagDepoisDaQuery(t *testing.T) {
	searchCmd, sf := newSearchFlagSet()
	searchCmd.SetOutput(io.Discard)

	var buf bytes.Buffer
	code := parseSearchFlags(searchCmd, []string{"arquitetura", "--tag", "inbox"}, &buf)

	if code != 0 {
		t.Errorf("esperava código zero, obteve %d (%s)", code, buf.String())
	}
	if *sf.tag != "inbox" {
		t.Errorf("esperava a tag 'inbox', obteve '%s'", *sf.tag)
	}
	if len(searchCmd.Args()) != 1 || searchCmd.Args()[0] != "arquitetura" {
		t.Errorf("esperava a query como argumento posicional, obteve %v", searchCmd.Args())
	}
}

// TestPrintHelp_MencionaFiltroDeTag: a ajuda da CLI precisa Teach a flag.
func TestPrintHelp_MencionaFiltroDeTag(t *testing.T) {
	help := capturePrintHelp(t)
	if !strings.Contains(help, "--tag") {
		t.Errorf("a ajuda de `mem search` não menciona a flag --tag:\n%s", help)
	}
}

// captureStdout executa a função com o stdout redirecionado e devolve a saída.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe falhou: %v", err)
	}

	original := os.Stdout
	os.Stdout = writer

	captured := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, reader)
		captured <- buf.String()
	}()

	fn()

	os.Stdout = original
	writer.Close()
	out := <-captured
	reader.Close()

	return out
}
