package watcher

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// novoBancoTags cria um banco SQLite temporário para os testes de tags.
// O ambiente sem CGO (sem sqlite-vec) não suporta o banco: o teste é pulado,
// não falhado.
func novoBancoTags(t *testing.T) (*sql.DB, string) {
	t.Helper()

	tmpDir := t.TempDir()
	database, err := db.InitDB(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		if strings.Contains(err.Error(), "CGO_ENABLED=0") || strings.Contains(err.Error(), "binary was compiled without cgo") {
			t.Skip("Pulando teste SQLite: ambiente sem CGO")
		}
		t.Fatalf("erro ao iniciar banco SQLite: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	return database, tmpDir
}

// tagsDoDocumento devolve as linhas de document_tags de um documento.
func tagsDoDocumento(t *testing.T, database *sql.DB, docID string) map[string]sql.NullString {
	t.Helper()

	rows, err := database.QueryContext(context.Background(), `
		SELECT tag_key, resolved_doc_id FROM document_tags WHERE document_id = ?
	`, docID)
	if err != nil {
		t.Fatalf("erro ao ler document_tags: %v", err)
	}
	defer rows.Close()

	tags := make(map[string]sql.NullString)
	for rows.Next() {
		var key string
		var resolved sql.NullString
		if err := rows.Scan(&key, &resolved); err != nil {
			t.Fatalf("erro ao escanear linha de document_tags: %v", err)
		}
		tags[key] = resolved
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("erro ao iterar document_tags: %v", err)
	}
	return tags
}

// contaTags conta as linhas de document_tags de um documento.
func contaTags(t *testing.T, database *sql.DB, docID string) int {
	t.Helper()
	return len(tagsDoDocumento(t, database, docID))
}

// TestIndexSingleFileSQLite_GravaTagOrfa cobre o requisito central da T3: a
// tag órfã (que não nomeia nenhum documento) é gravada mesmo assim, com
// resolved_doc_id nulo.
func TestIndexSingleFileSQLite_GravaTagOrfa(t *testing.T) {
	database, tmpDir := novoBancoTags(t)
	ctx := context.Background()

	notePath := filepath.Join(tmpDir, "nota.md")
	content := "---\ntitle: Nota\ncategory: resource\n---\n\nCorpo com #homenagem-inexistente.\n"
	if err := os.WriteFile(notePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := IndexSingleFileSQLite(ctx, database, nil, nil, notePath, false); err != nil {
		t.Fatalf("erro ao indexar nota: %v", err)
	}

	tags := tagsDoDocumento(t, database, notePath)
	resolved, ok := tags["homenagem-inexistente"]
	if !ok {
		t.Fatalf("tag órfã não foi gravada; document_tags = %v", tags)
	}
	if resolved.Valid {
		t.Errorf("tag órfã tem resolved_doc_id = %q; esperava NULL", resolved.String)
	}
}

// TestIndexSingleFileSQLite_ResolveTagParaDocumento cobre a resolução: a tag
// que nomeia um documento aponta para o documents.id desse documento — não para
// o documento que carrega a tag.
func TestIndexSingleFileSQLite_ResolveTagParaDocumento(t *testing.T) {
	database, tmpDir := novoBancoTags(t)
	ctx := context.Background()

	alvoPath := filepath.Join(tmpDir, "alvo.md")
	if err := os.WriteFile(alvoPath, []byte("# Alvo\n\nDocumento alvo.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := IndexSingleFileSQLite(ctx, database, nil, nil, alvoPath, false); err != nil {
		t.Fatalf("erro ao indexar alvo: %v", err)
	}

	notePath := filepath.Join(tmpDir, "nota.md")
	content := "---\ntitle: Nota\ncategory: resource\n---\n\nCarrega a tag #alvo.\n"
	if err := os.WriteFile(notePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := IndexSingleFileSQLite(ctx, database, nil, nil, notePath, false); err != nil {
		t.Fatalf("erro ao indexar nota: %v", err)
	}

	tags := tagsDoDocumento(t, database, notePath)
	resolved, ok := tags["alvo"]
	if !ok {
		t.Fatalf("tag 'alvo' não foi gravada; document_tags = %v", tags)
	}
	if !resolved.Valid {
		t.Fatal("tag 'alvo' ficou sem resolved_doc_id; esperava o id do documento alvo")
	}
	if resolved.String != alvoPath {
		t.Errorf("resolved_doc_id = %q; esperava %q (o documents.id do documento nomeado)", resolved.String, alvoPath)
	}
}

// TestIndexSingleFileSQLite_TagsComCasingDiferenteViramUmaLinha cobre a
// identidade da linha: #Tag e #TAG são a mesma tag no Obsidian.
func TestIndexSingleFileSQLite_TagsComCasingDiferenteViramUmaLinha(t *testing.T) {
	database, tmpDir := novoBancoTags(t)
	ctx := context.Background()

	notePath := filepath.Join(tmpDir, "casing.md")
	content := "---\ntitle: Casing\ncategory: resource\n---\n\nCores #Tag e #TAG no mesmo documento.\n"
	if err := os.WriteFile(notePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := IndexSingleFileSQLite(ctx, database, nil, nil, notePath, false); err != nil {
		t.Fatalf("erro ao indexar nota: %v", err)
	}

	if got := contaTags(t, database, notePath); got != 1 {
		t.Fatalf("document_tags do documento = %d linhas; esperava 1 (#Tag e #TAG são a mesma linha)", got)
	}

	var tag string
	if err := database.QueryRowContext(ctx, `SELECT tag FROM document_tags WHERE document_id = ?`, notePath).Scan(&tag); err != nil {
		t.Fatalf("erro ao ler a linha de document_tags: %v", err)
	}
	if tag != "Tag" {
		t.Errorf("tag = %q; esperava 'Tag' (casing da primeira ocorrência)", tag)
	}
}

// TestIndexSingleFileSQLite_ReindexNaoDuplica cobre o reindex: reindexar o
// mesmo arquivo não duplica as tags.
func TestIndexSingleFileSQLite_ReindexNaoDuplica(t *testing.T) {
	database, tmpDir := novoBancoTags(t)
	ctx := context.Background()

	notePath := filepath.Join(tmpDir, "nota.md")
	content := "---\ntitle: Nota\ncategory: resource\n---\n\nTags #alpha e #beta.\n"
	if err := os.WriteFile(notePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		if _, err := IndexSingleFileSQLite(ctx, database, nil, nil, notePath, true); err != nil {
			t.Fatalf("erro ao indexar nota (tentativa %d): %v", i+1, err)
		}
	}

	if got := contaTags(t, database, notePath); got != 2 {
		t.Fatalf("document_tags do documento = %d linhas após 3 indexações; esperava 2", got)
	}
}

// TestPurgeSingleFileSQLite_RemoveTags cobre o caminho de remoção por purge: o
// arquivo sumiu do disco, então as tags dele têm que sumir do índice.
func TestPurgeSingleFileSQLite_RemoveTags(t *testing.T) {
	database, tmpDir := novoBancoTags(t)
	ctx := context.Background()

	notePath := filepath.Join(tmpDir, "nota.md")
	content := "---\ntitle: Nota\ncategory: resource\n---\n\nTags #alpha.\n"
	if err := os.WriteFile(notePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := IndexSingleFileSQLite(ctx, database, nil, nil, notePath, false); err != nil {
		t.Fatalf("erro ao indexar nota: %v", err)
	}
	if got := contaTags(t, database, notePath); got != 1 {
		t.Fatalf("document_tags do documento = %d linhas antes da purga; esperava 1", got)
	}

	if _, err := PurgeSingleFileSQLite(ctx, database, notePath); err != nil {
		t.Fatalf("erro ao purgar nota: %v", err)
	}

	if got := contaTags(t, database, notePath); got != 0 {
		t.Errorf("document_tags do documento = %d linhas após a purga; esperava 0", got)
	}
}

// TestPruneDeletedDocuments_RemoveTags cobre o caminho de remoção por prune: o
// documento saiu do índice porque o arquivo não está mais no disco.
func TestPruneDeletedDocuments_RemoveTags(t *testing.T) {
	database, tmpDir := novoBancoTags(t)
	ctx := context.Background()

	notePath := filepath.Join(tmpDir, "nota.md")
	content := "---\ntitle: Nota\ncategory: resource\n---\n\nTags #alpha.\n"
	if err := os.WriteFile(notePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := IndexSingleFileSQLite(ctx, database, nil, nil, notePath, false); err != nil {
		t.Fatalf("erro ao indexar nota: %v", err)
	}

	// O arquivo não está mais na pasta: o prune o remove do índice.
	pruned, err := db.PruneDeletedDocuments(ctx, database, tmpDir, []string{})
	if err != nil {
		t.Fatalf("erro ao podar documentos: %v", err)
	}
	if len(pruned) != 1 || pruned[0] != notePath {
		t.Fatalf("prune devolveu %v; esperava [%s]", pruned, notePath)
	}

	if got := contaTags(t, database, notePath); got != 0 {
		t.Errorf("document_tags do documento = %d linhas após o prune; esperava 0", got)
	}
}

// TestBuildTagBindings cobre o núcleo do cálculo, sem passar pelo índice:
// tag órfã (sem resolved_doc_id), tag que nomeia um documento (com o id) e
// casing diferente na mesma linha.
func TestBuildTagBindings(t *testing.T) {
	titleToID := map[string]string{
		"alvo": "docs/alvo.md",
		"nota": "docs/nota.md",
	}

	bindings := buildTagBindings([]string{"#alvo", "#Orfa", "#ORFA"}, titleToID)

	byKey := make(map[string]store.TagBinding, len(bindings))
	for _, binding := range bindings {
		byKey[binding.Key] = binding
	}

	if len(byKey) != 2 {
		t.Fatalf("vínculos = %d; esperava 2 (alvo e orfa)", len(byKey))
	}

	alvo, ok := byKey["alvo"]
	if !ok {
		t.Fatal("tag 'alvo' não foi montada")
	}
	if alvo.ResolvedDocID != "docs/alvo.md" {
		t.Errorf("resolved_doc_id de 'alvo' = %q; esperava 'docs/alvo.md'", alvo.ResolvedDocID)
	}
	if alvo.Tag != "alvo" {
		t.Errorf("tag de exibição = %q; esperava 'alvo'", alvo.Tag)
	}

	orfa, ok := byKey["orfa"]
	if !ok {
		t.Fatal("tag órfã não foi montada")
	}
	if orfa.ResolvedDocID != "" {
		t.Errorf("resolved_doc_id da tag órfã = %q; esperava vazio", orfa.ResolvedDocID)
	}
}
