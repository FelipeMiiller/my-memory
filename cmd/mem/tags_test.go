package main

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// tagFixtureDoc é um documento do fixture de tags: o path e as tags que ele
// carrega.
type tagFixtureDoc struct {
	path string
	tags []string
	// resolved mapeia tag -> id do documento que a tag nomeia. Tag sem
	// resolução é a tag órfã (ADR-055): existe no inventário e não nomeia
	// nenhum documento.
	resolved map[string]string
}

// newTagFixture cria um vault SQLite com documentos e tags e devolve o caminho
// do arquivo.
//
// A ordem dos documentos importa: o casing de exibição de uma tag é o da
// linha criada primeiro, então o fixture insere na ordem em que foi declarado.
func newTagFixture(t *testing.T, docs ...tagFixtureDoc) string {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "vault_tags_fixture.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	for _, doc := range docs {
		title := strings.TrimSuffix(filepath.Base(doc.path), filepath.Ext(doc.path))
		err := db.InsertDocumentWithMeta(ctx, database, doc.path, doc.path, title, 1700000000, "hash-"+doc.path, "epoch", "", "resource", 1700000000, 128)
		if err != nil {
			t.Fatalf("InsertDocumentWithMeta(%s) falhou: %v", doc.path, err)
		}

		bindings := make([]store.TagBinding, 0, len(doc.tags))
		for _, tag := range doc.tags {
			bindings = append(bindings, store.TagBinding{Tag: tag, ResolvedDocID: doc.resolved[tag]})
		}
		if err := db.InsertDocumentTags(ctx, database, doc.path, bindings); err != nil {
			t.Fatalf("InsertDocumentTags(%s) falhou: %v", doc.path, err)
		}
	}
	return dbPath
}

// defaultTagFixture é o vault usado pelos testes: quatro documentos, tags
// repetidas (para ter contagem maior que 1), uma tag hierárquica, uma tag
// órfã e o par `#Tag`/`#TAG` que é uma só entrada.
func defaultTagFixture(t *testing.T) string {
	t.Helper()
	return newTagFixture(t,
		tagFixtureDoc{path: "docs/alpha.md", tags: []string{"#Tag", "#projeto/alpha", "#comum"}, resolved: map[string]string{"#projeto/alpha": "docs/alpha.md"}},
		tagFixtureDoc{path: "docs/beta.md", tags: []string{"#TAG", "#projeto/beta", "#comum"}},
		tagFixtureDoc{path: "notes/gamma.md", tags: []string{"#gamma", "#projeto/alpha", "#inbox"}},
		tagFixtureDoc{path: "notes/delta.md", tags: []string{"#inbox/to-read"}},
	)
}

// runTagsFixture executa `mem tags` contra o fixture e devolve saída e erro.
func runTagsFixture(t *testing.T, dbPath string, args []string) (string, error) {
	t.Helper()

	full := append([]string{"--db", dbPath, "--storage", "sqlite"}, args...)
	var buf bytes.Buffer
	err := runTagsCommand(context.Background(), "fixture-repo", full, &buf)
	return buf.String(), err
}

// runTagFixture executa `mem tag` contra o fixture e devolve saída e erro.
func runTagFixture(t *testing.T, dbPath string, args []string) (string, error) {
	t.Helper()

	full := append([]string{"--db", dbPath, "--storage", "sqlite"}, args...)
	var buf bytes.Buffer
	err := runTagCommand(context.Background(), "fixture-repo", full, &buf)
	return buf.String(), err
}

// tagLines devolve as linhas não-vazias da saída.
func tagLines(out string) []string {
	trimmed := strings.TrimSpace(out)
	if trimmed == "" {
		return []string{}
	}
	return strings.Split(trimmed, "\n")
}

func TestRunTagsCommand_ListaUmaTagPorLinha(t *testing.T) {
	dbPath := defaultTagFixture(t)

	out, err := runTagsFixture(t, dbPath, nil)
	if err != nil {
		t.Fatalf("runTagsCommand falhou: %v", err)
	}

	got := tagLines(out)
	want := []string{"comum", "gamma", "inbox", "inbox/to-read", "projeto/alpha", "projeto/beta", "Tag"}
	if len(got) != len(want) {
		t.Fatalf("esperava %d tags, obteve %d:\n%s", len(want), len(got), out)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("posição %d: esperava %q, obteve %q (ordem alfabética por tag_key)", i, want[i], got[i])
		}
	}
}

// TestRunTagsCommand_Counts imprime "tag<TAB>contagem" — a mesma forma do
// `obsidian tags` com contagem.
func TestRunTagsCommand_Counts(t *testing.T) {
	dbPath := defaultTagFixture(t)

	out, err := runTagsFixture(t, dbPath, []string{"--counts"})
	if err != nil {
		t.Fatalf("runTagsCommand --counts falhou: %v", err)
	}

	want := []string{"comum\t2", "gamma\t1", "inbox\t1", "inbox/to-read\t1", "projeto/alpha\t2", "projeto/beta\t1", "Tag\t2"}
	got := tagLines(out)
	if len(got) != len(want) {
		t.Fatalf("esperava %d linhas, obteve %d:\n%s", len(want), len(got), out)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("posição %d: esperava %q, obteve %q", i, want[i], got[i])
		}
	}
}

// TestRunTagsCommand_SortCount ordena por contagem decrescente, com desempate
// pela tag ascendente — a saída não pode mudar entre execuções.
func TestRunTagsCommand_SortCount(t *testing.T) {
	dbPath := defaultTagFixture(t)

	out, err := runTagsFixture(t, dbPath, []string{"--sort", "count"})
	if err != nil {
		t.Fatalf("runTagsCommand --sort count falhou: %v", err)
	}

	want := []string{"Tag", "comum", "projeto/alpha", "gamma", "inbox", "inbox/to-read", "projeto/beta"}
	got := tagLines(out)
	if len(got) != len(want) {
		t.Fatalf("esperava %d tags, obteve %d:\n%s", len(want), len(got), out)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("posição %d: esperava %q, obteve %q (contagem DESC, desempate tag ASC)", i, want[i], got[i])
		}
	}
}

func TestRunTagsCommand_FiltraPorArquivo(t *testing.T) {
	dbPath := defaultTagFixture(t)

	out, err := runTagsFixture(t, dbPath, []string{"--file", "docs/alpha.md"})
	if err != nil {
		t.Fatalf("runTagsCommand --file falhou: %v", err)
	}

	got := tagLines(out)
	want := []string{"comum", "projeto/alpha", "Tag"}
	if len(got) != len(want) {
		t.Fatalf("esperava %v, obteve %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("posição %d: esperava %q, obteve %q", i, want[i], got[i])
		}
	}
}

func TestRunTagsCommand_FiltraPorPasta(t *testing.T) {
	dbPath := defaultTagFixture(t)

	out, err := runTagsFixture(t, dbPath, []string{"--folder", "docs"})
	if err != nil {
		t.Fatalf("runTagsCommand --folder falhou: %v", err)
	}

	got := tagLines(out)
	want := []string{"comum", "projeto/alpha", "projeto/beta", "Tag"}
	if len(got) != len(want) {
		t.Fatalf("esperava %v, obteve %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("posição %d: esperava %q, obteve %q", i, want[i], got[i])
		}
	}
}

func TestRunTagsCommand_Total(t *testing.T) {
	dbPath := defaultTagFixture(t)

	out, err := runTagsFixture(t, dbPath, []string{"--total"})
	if err != nil {
		t.Fatalf("runTagsCommand --total falhou: %v", err)
	}
	if strings.TrimSpace(out) != "7" {
		t.Errorf("esperava só a contagem de tags distintas (7), obteve:\n%s", out)
	}
}

func TestRunTagsCommand_Json(t *testing.T) {
	dbPath := defaultTagFixture(t)

	out, err := runTagsFixture(t, dbPath, []string{"--json"})
	if err != nil {
		t.Fatalf("runTagsCommand --json falhou: %v", err)
	}

	var tags []store.VaultTag
	if err := json.Unmarshal([]byte(out), &tags); err != nil {
		t.Fatalf("JSON inválido: %v\n%s", err, out)
	}
	if len(tags) != 7 {
		t.Fatalf("esperava 7 tags no JSON, obteve %d:\n%s", len(tags), out)
	}

	first := tags[0]
	if first.Tag != "comum" || first.Key != "comum" || first.Count != 2 {
		t.Errorf("primeira tag inesperada: %+v", first)
	}

	// A tag resolvida carrega o documento que ela nomeia (ADR-055).
	var resolved *store.VaultTag
	for i := range tags {
		if tags[i].Key == "projeto/alpha" {
			resolved = &tags[i]
			break
		}
	}
	if resolved == nil {
		t.Fatalf("a tag projeto/alpha não apareceu no JSON:\n%s", out)
	}
	if resolved.ResolvedDocID != "docs/alpha.md" {
		t.Errorf("esperava resolved_doc_id docs/alpha.md, obteve %q", resolved.ResolvedDocID)
	}
	if resolved.ResolvedTitle != "alpha" {
		t.Errorf("esperava resolved_title alpha, obteve %q", resolved.ResolvedTitle)
	}
}

// TestRunTagsCommand_VaultSemTags responde vazio e sai zero: um vault sem
// tag nenhuma é uma resposta válida, não uma falha.
func TestRunTagsCommand_VaultSemTags(t *testing.T) {
	dbPath := newTagFixture(t, tagFixtureDoc{path: "docs/alpha.md"})

	out, err := runTagsFixture(t, dbPath, nil)
	if err != nil {
		t.Fatalf("vault sem tags deveria sair zero, obteve erro: %v", err)
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("vault sem tags deveria imprimir lista vazia, obteve:\n%s", out)
	}

	outJSON, err := runTagsFixture(t, dbPath, []string{"--json"})
	if err != nil {
		t.Fatalf("runTagsCommand --json falhou: %v", err)
	}
	if strings.TrimSpace(outJSON) != "[]" {
		t.Errorf("inventário vazio em JSON deveria ser [], obteve:\n%s", outJSON)
	}
}

// TestRunTagsCommand_TagOrfanaAparece cobre a ADR-055: a tag que não nomeia
// nenhum documento continua no inventário.
func TestRunTagsCommand_TagOrfanaAparece(t *testing.T) {
	dbPath := defaultTagFixture(t)

	out, err := runTagsFixture(t, dbPath, []string{"--json"})
	if err != nil {
		t.Fatalf("runTagsCommand --json falhou: %v", err)
	}

	var tags []store.VaultTag
	if err := json.Unmarshal([]byte(out), &tags); err != nil {
		t.Fatalf("JSON inválido: %v\n%s", err, out)
	}

	found := false
	for _, tag := range tags {
		if tag.Key != "inbox" {
			continue
		}
		found = true
		if tag.ResolvedDocID != "" {
			t.Errorf("a tag órfã não deveria ter resolução, obteve %q", tag.ResolvedDocID)
		}
	}
	if !found {
		t.Errorf("a tag órfã #inbox não apareceu no inventário:\n%s", out)
	}
}

// TestRunTagsCommand_TagComCasingDuplicado cobre a identidade da tag: `#Tag`
// e `#TAG` são uma só entrada, exibida com o casing da primeira criação.
func TestRunTagsCommand_TagComCasingDuplicado(t *testing.T) {
	dbPath := defaultTagFixture(t)

	out, err := runTagsFixture(t, dbPath, nil)
	if err != nil {
		t.Fatalf("runTagsCommand falhou: %v", err)
	}

	got := tagLines(out)
	for _, line := range got {
		if line == "TAG" {
			t.Errorf("`#TAG` não pode virar uma entrada separada de `#Tag`:\n%s", out)
		}
	}

	count := 0
	for _, line := range got {
		if line == "Tag" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("esperava exatamente uma entrada `Tag`, obteve %d:\n%s", count, out)
	}
}

func TestRunTagCommand_ContaDescendentes(t *testing.T) {
	dbPath := defaultTagFixture(t)

	out, err := runTagFixture(t, dbPath, []string{"inbox"})
	if err != nil {
		t.Fatalf("runTagCommand falhou: %v", err)
	}
	if strings.TrimSpace(out) != "2" {
		t.Errorf("`mem tag inbox` deveria contar a tag e as descendentes (2), obteve:\n%s", out)
	}
}

func TestRunTagCommand_Verbose(t *testing.T) {
	dbPath := defaultTagFixture(t)

	out, err := runTagFixture(t, dbPath, []string{"inbox", "--verbose"})
	if err != nil {
		t.Fatalf("runTagCommand --verbose falhou: %v", err)
	}

	got := tagLines(out)
	want := []string{"2", "notes/delta.md", "notes/gamma.md"}
	if len(got) != len(want) {
		t.Fatalf("esperava %v, obteve %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("posição %d: esperava %q, obteve %q", i, want[i], got[i])
		}
	}
}

func TestRunTagCommand_Total(t *testing.T) {
	dbPath := defaultTagFixture(t)

	out, err := runTagFixture(t, dbPath, []string{"inbox", "--total"})
	if err != nil {
		t.Fatalf("runTagCommand --total falhou: %v", err)
	}
	if strings.TrimSpace(out) != "2" {
		t.Errorf("esperava só a contagem (2), obteve:\n%s", out)
	}
}

func TestRunTagCommand_Json(t *testing.T) {
	dbPath := defaultTagFixture(t)

	out, err := runTagFixture(t, dbPath, []string{"projeto/alpha", "--json"})
	if err != nil {
		t.Fatalf("runTagCommand --json falhou: %v", err)
	}

	var result tagQueryResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("JSON inválido: %v\n%s", err, out)
	}
	if result.Tag != "projeto/alpha" {
		t.Errorf("esperava tag projeto/alpha, obteve %q", result.Tag)
	}
	if result.Count != 2 {
		t.Errorf("esperava contagem 2, obteve %d", result.Count)
	}
	if len(result.Documents) != 2 {
		t.Errorf("esperava 2 documentos, obteve %d:\n%s", len(result.Documents), out)
	}
	if result.ResolvedDocID != "docs/alpha.md" {
		t.Errorf("esperava o documento resolvido docs/alpha.md, obteve %q", result.ResolvedDocID)
	}
}

// TestRunTagCommand_SemNome sai com código diferente de zero: sem o nome da
// tag o comando não tem o que fazer.
func TestRunTagCommand_SemNome(t *testing.T) {
	dbPath := defaultTagFixture(t)

	out, err := runTagFixture(t, dbPath, nil)
	if err == nil {
		t.Fatalf("`mem tag` sem nome deveria falhar, obteve saída zero:\n%s", out)
	}
	if !strings.Contains(out, "Uso: mem tag") {
		t.Errorf("esperava a linha de uso na saída, obteve:\n%s", out)
	}
}

// TestRunTagCommand_TagInexistente não é erro: contagem 0 e saída zero.
func TestRunTagCommand_TagInexistente(t *testing.T) {
	dbPath := defaultTagFixture(t)

	out, err := runTagFixture(t, dbPath, []string{"nao-existe"})
	if err != nil {
		t.Fatalf("tag inexistente deveria sair zero, obteve erro: %v", err)
	}
	if strings.TrimSpace(out) != "0" {
		t.Errorf("esperava contagem 0, obteve:\n%s", out)
	}
}

// TestRunTagsCommand_AjudaNaoEhErro: `mem tags --help` imprime a ajuda e sai
// com código zero. Pedir ajuda não é erro de uso.
func TestRunTagsCommand_AjudaNaoEhErro(t *testing.T) {
	dbPath := defaultTagFixture(t)

	out, err := runTagsFixture(t, dbPath, []string{"--help"})
	if err != nil {
		t.Fatalf("`mem tags --help` deveria sair zero, obteve erro: %v", err)
	}
	if !strings.Contains(out, "Usage of tags") {
		t.Errorf("esperava a ajuda na saída, obteve:\n%s", out)
	}
}

// TestRunTagCommand_AjudaNaoEhErro: idem para `mem tag --help`.
func TestRunTagCommand_AjudaNaoEhErro(t *testing.T) {
	dbPath := defaultTagFixture(t)

	out, err := runTagFixture(t, dbPath, []string{"--help"})
	if err != nil {
		t.Fatalf("`mem tag --help` deveria sair zero, obteve erro: %v", err)
	}
	if !strings.Contains(out, "Usage of tag") {
		t.Errorf("esperava a ajuda na saída, obteve:\n%s", out)
	}
}
