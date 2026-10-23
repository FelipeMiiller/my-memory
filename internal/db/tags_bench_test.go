package db

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/store"
)

// newTagsBenchDB monta um vault de teste com `documents` documentos, cada um
// com `tagsPerDocument` tags, e devolve o banco pronto para medir.
//
// As tags são escolhidas para exercitar os dois caminhos quentes da
// superfície de tags: a tag compartilhada (o inventário agrega contagem) e a
// hierarquia (a consulta por tag casa as descendentes via `LIKE`).
func newTagsBenchDB(b *testing.B, documents, tagsPerDocument int) *sql.DB {
	b.Helper()

	database, err := InitDB(filepath.Join(b.TempDir(), "bench.db"))
	if err != nil {
		b.Fatalf("InitDB falhou: %v", err)
	}
	b.Cleanup(func() { database.Close() })

	ctx := context.Background()
	now := time.Now().Unix()
	for i := 0; i < documents; i++ {
		docID := fmt.Sprintf("doc-%04d", i)
		path := fmt.Sprintf("notes/nota-%04d.md", i)
		if _, err := database.ExecContext(ctx, `
			INSERT INTO documents (id, path, title, updated_at, mtime, size)
			VALUES (?, ?, ?, ?, ?, ?)
		`, docID, path, docID, now, int64(100), int64(42)); err != nil {
			b.Fatalf("insercao do documento %q falhou: %v", docID, err)
		}

		bindings := make([]store.TagBinding, 0, tagsPerDocument)
		for j := 0; j < tagsPerDocument; j++ {
			names := []string{"comum", "projeto/alfa", fmt.Sprintf("equipe/e%02d", i%10)}
			bindings = append(bindings, store.TagBinding{Tag: names[j%len(names)]})
		}
		if err := InsertDocumentTags(ctx, database, docID, bindings); err != nil {
			b.Fatalf("InsertDocumentTags do documento %q falhou: %v", docID, err)
		}
	}

	return database
}

// BenchmarkListVaultTags mede o inventário de tags do vault — a consulta que
// roda por trás de `mem tags` e de `memory_get_tags`.
func BenchmarkListVaultTags(b *testing.B) {
	database := newTagsBenchDB(b, 200, 3)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tags, err := ListVaultTags(ctx, database, store.TagFilter{})
		if err != nil {
			b.Fatalf("ListVaultTags falhou: %v", err)
		}
		if len(tags) == 0 {
			b.Fatal("inventario vazio")
		}
	}
}

// BenchmarkFindVaultDocumentsByTag mede a consulta de documentos por tag,
// incluindo o caminho hierárquico: `equipe` casa `equipe/e00` … `equipe/e09`
// pelo `LIKE` escapado.
func BenchmarkFindVaultDocumentsByTag(b *testing.B) {
	database := newTagsBenchDB(b, 200, 3)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		documents, err := FindVaultDocumentsByTag(ctx, database, "equipe")
		if err != nil {
			b.Fatalf("FindVaultDocumentsByTag falhou: %v", err)
		}
		if len(documents) == 0 {
			b.Fatal("nenhum documento encontrado")
		}
	}
}

// BenchmarkInsertDocumentTags mede a substituição das tags de um documento — o
// que roda a cada reindexação de arquivo.
func BenchmarkInsertDocumentTags(b *testing.B) {
	database := newTagsBenchDB(b, 1, 3)
	ctx := context.Background()

	bindings := []store.TagBinding{{Tag: "comum"}, {Tag: "projeto/alfa"}, {Tag: "equipe/e00"}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := InsertDocumentTags(ctx, database, "doc-0000", bindings); err != nil {
			b.Fatalf("InsertDocumentTags falhou: %v", err)
		}
	}
}
