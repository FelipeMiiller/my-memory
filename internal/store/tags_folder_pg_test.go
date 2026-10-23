package store

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

// TestPostgresListVaultTags_FolderAceitaPathWindows prova, contra o Postgres
// real, que o filtro por pasta do inventário de tags casa o path com o
// separador do Windows.
//
// O path indexado no Windows usa "\", e a comparação é feita sobre a forma
// com "/" — sem a normalização, `mem tags --folder` devolvia lista vazia em
// Windows e funcionava em Linux. O teste de integração existe porque o
// literal `E'\\'` e o `REPLACE` só valem contra o Postgres de verdade.
func TestPostgresListVaultTags_FolderAceitaPathWindows(t *testing.T) {
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

	repo := fmt.Sprintf("test-tag-folder-%d", time.Now().UnixNano())
	now := time.Now().Unix()
	docs := []string{`vault\notas\sub\um.md`, `vault\docs-outro\x.md`, `outros\y.md`}
	for _, p := range docs {
		if err := s.InsertDocumentWithMeta(ctx, repo, p, p, p, now, "h-"+p, "", "", "resource", now, 1); err != nil {
			t.Fatalf("InsertDocumentWithMeta(%s) falhou: %v", p, err)
		}
	}
	for i, tag := range []string{"tag-um", "tag-docs", "tag-y"} {
		if err := s.InsertDocumentTags(ctx, repo, docs[i], []TagBinding{{Tag: tag}}); err != nil {
			t.Fatalf("InsertDocumentTags(%s) falhou: %v", docs[i], err)
		}
	}

	// A pasta "notas" casa o path com backslash.
	inventory, err := s.ListVaultTags(ctx, repo, TagFilter{Folder: "notas"})
	if err != nil {
		t.Fatalf("ListVaultTags falhou: %v", err)
	}
	if len(inventory) != 1 || inventory[0].Key != "tag-um" {
		t.Fatalf("filtro por pasta 'notas' = %+v; esperava so 'tag-um'", inventory)
	}

	// O filtro também aceita a pasta escrita com o separador do Windows.
	inventory, err = s.ListVaultTags(ctx, repo, TagFilter{Folder: `notas\sub`})
	if err != nil {
		t.Fatalf("ListVaultTags falhou: %v", err)
	}
	if len(inventory) != 1 || inventory[0].Key != "tag-um" {
		t.Fatalf("filtro por pasta 'notas\\sub' = %+v; esperava so 'tag-um'", inventory)
	}

	// "docs" é um segmento inteiro: não casa "docs-outro".
	inventory, err = s.ListVaultTags(ctx, repo, TagFilter{Folder: "docs"})
	if err != nil {
		t.Fatalf("ListVaultTags falhou: %v", err)
	}
	if len(inventory) != 0 {
		t.Fatalf("filtro por pasta 'docs' = %+v; esperava vazio ('docs' não casa 'docs-outro')", inventory)
	}
}
