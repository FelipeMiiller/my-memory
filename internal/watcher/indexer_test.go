package watcher

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/db"
)

func TestIndexAndPurgeSingleFileSQLite(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	database, err := db.InitDB(dbPath)
	if err != nil {
		if strings.Contains(err.Error(), "CGO_ENABLED=0") || strings.Contains(err.Error(), "binary was compiled without cgo") {
			t.Skip("Pulando teste SQLite: ambiente sem CGO")
		}
		t.Fatalf("erro ao iniciar banco SQLite: %v", err)
	}
	defer database.Close()

	ctx := context.Background()

	// 1. Criar nota Markdown de teste
	notePath := filepath.Join(tmpDir, "nota.md")
	content := "# Minha Nota\n\nConecta com [[OutraNota]] e [[TerceiraNota]].\n"
	if err := os.WriteFile(notePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// 2. Indexar nota pela primeira vez: deve ser Action == "indexed"
	res, err := IndexSingleFileSQLite(ctx, database, nil, nil, notePath, false)
	if err != nil {
		t.Fatalf("erro ao indexar nota cirurgicamente: %v", err)
	}
	if res.Action != "indexed" {
		t.Errorf("esperava action 'indexed', obteve '%s'", res.Action)
	}
	if res.EdgesCount != 2 {
		t.Errorf("esperava 2 arestas extraídas, obteve %d", res.EdgesCount)
	}
	if res.Title != "nota" {
		t.Errorf("esperava title 'nota', obteve '%s'", res.Title)
	}

	// 3. Indexar novamente sem alterações: deve ser Action == "cached"
	resCached, err := IndexSingleFileSQLite(ctx, database, nil, nil, notePath, false)
	if err != nil {
		t.Fatalf("erro ao reindexar nota em cache: %v", err)
	}
	if resCached.Action != "cached" {
		t.Errorf("esperava action 'cached', obteve '%s'", resCached.Action)
	}

	// 4. Modificar nota: deve reindexar com nova contagem de arestas
	newContent := "# Minha Nota Atualizada\n\nConecta apenas com [[NotaUnica]].\n"
	if err := os.WriteFile(notePath, []byte(newContent), 0644); err != nil {
		t.Fatal(err)
	}
	resMod, err := IndexSingleFileSQLite(ctx, database, nil, nil, notePath, false)
	if err != nil {
		t.Fatalf("erro ao indexar nota modificada: %v", err)
	}
	if resMod.Action != "indexed" {
		t.Errorf("esperava action 'indexed', obteve '%s'", resMod.Action)
	}
	if resMod.EdgesCount != 1 {
		t.Errorf("esperava 1 aresta extraída, obteve %d", resMod.EdgesCount)
	}

	// 5. Purgar nota: deve limpar do banco
	purgeRes, err := PurgeSingleFileSQLite(ctx, database, notePath)
	if err != nil {
		t.Fatalf("erro ao purgar nota: %v", err)
	}
	if purgeRes.Action != "purged" {
		t.Errorf("esperava action 'purged', obteve '%s'", purgeRes.Action)
	}

	// Verifica se hash sumiu após a purga
	hash, _ := db.GetDocumentHash(ctx, database, notePath)
	if hash != "" {
		t.Errorf("esperava hash vazio após purga, obteve '%s'", hash)
	}
}
