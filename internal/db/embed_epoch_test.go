package db

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

// Regressão do problema que o embed_epoch resolve: uma linha gravada por um
// embedder antigo tem content_hash exatamente igual ao do arquivo em disco (o
// arquivo não mudou), mas os vetores guardados vêm de outro modelo. Se o gate
// só comparar hash, ele pula a reindexação e mistura vetores de gerações
// diferentes no mesmo índice sem nenhum sinal.

func TestEmbedEpoch_Roundtrip(t *testing.T) {
	ctx := context.Background()
	database, err := InitDB(filepath.Join(t.TempDir(), "epoch_test.db"))
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	const epoch = "paraphrase-multilingual-MiniLM-L12-v2:384:v1"
	if err := InsertDocumentWithMeta(ctx, database, "doc-1", "notes/a.md", "A", time.Now().Unix(), "hash-1", epoch, "Resumo A", "resource"); err != nil {
		t.Fatalf("InsertDocumentWithMeta falhou: %v", err)
	}

	hash, ep, err := GetDocumentIndexState(ctx, database, "doc-1")
	if err != nil {
		t.Fatalf("GetDocumentIndexState falhou: %v", err)
	}
	if hash != "hash-1" {
		t.Errorf("content_hash: got %q quer %q", hash, "hash-1")
	}
	if ep != epoch {
		t.Errorf("embed_epoch: got %q quer %q", ep, epoch)
	}

	// Update precisa propagar o epoch, senão toda reindexação limpa deixaria NULL.
	if err := InsertDocumentWithMeta(ctx, database, "doc-1", "notes/a.md", "A2", time.Now().Unix(), "hash-2", epoch+"2", "Resumo A2", "resource"); err != nil {
		t.Fatalf("update falhou: %v", err)
	}
	_, ep, err = GetDocumentIndexState(ctx, database, "doc-1")
	if err != nil {
		t.Fatalf("GetDocumentIndexState pós-update falhou: %v", err)
	}
	if ep != epoch+"2" {
		t.Errorf("embed_epoch após update: got %q quer %q", ep, epoch+"2")
	}
}

func TestEmbedEpoch_LinhaLegadaNaoPassaNoGate(t *testing.T) {
	ctx := context.Background()
	database, err := InitDB(filepath.Join(t.TempDir(), "legacy_test.db"))
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	// Simula o estado herdado: documento gravado antes do embed_epoch existir,
	// com o vetor vindo do modelo antigo (nomic 768).
	if _, err := database.ExecContext(ctx, `
		INSERT INTO documents (id, path, title, updated_at, content_hash, abstract, category)
		VALUES ('legacy', 'notes/legacy.md', 'Legacy', 1, 'hash-igual', 'Resumo', 'resource')
	`); err != nil {
		t.Fatalf("insert legado falhou: %v", err)
	}

	hash, epoch, err := GetDocumentIndexState(ctx, database, "legacy")
	if err != nil {
		t.Fatalf("GetDocumentIndexState falhou: %v", err)
	}
	if hash != "hash-igual" {
		t.Fatalf("premissa do teste: hash deveria bater, got %q", hash)
	}
	if epoch != "" {
		t.Fatalf("premissa do teste: linha legada deveria ter epoch vazio, got %q", epoch)
	}

	// A condição exata do gate em runIndexSQLite/runIndexPostgres.
	currentHash := "hash-igual"
	currentEpoch := "paraphrase-multilingual-MiniLM-L12-v2:384:v1"
	passariaNoGate := hash != "" && hash == currentHash && epoch == currentEpoch
	if passariaNoGate {
		t.Error("BUG: hash igual com epoch vazio passou no gate — vetores do embedder antigo sobreviveram à troca de modelo")
	}
}

func TestEmbedEpoch_SemProviderNaoInvalidaBase(t *testing.T) {
	ctx := context.Background()
	database, err := InitDB(filepath.Join(t.TempDir(), "nolex_test.db"))
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	// Modo léxico: nada vetorial é gravado, então epoch "" nos dois lados e o
	// cache por hash continua valendo — sem reindexar a base a cada execução.
	if err := InsertDocumentWithMeta(ctx, database, "fts", "notes/f.md", "F", time.Now().Unix(), "h", "", "Resumo", "resource"); err != nil {
		t.Fatalf("insert falhou: %v", err)
	}

	hash, epoch, err := GetDocumentIndexState(ctx, database, "fts")
	if err != nil {
		t.Fatalf("GetDocumentIndexState falhou: %v", err)
	}
	if !(hash != "" && hash == "h" && epoch == "") {
		t.Errorf("modo léxico deveria passar no gate; hash=%q epoch=%q", hash, epoch)
	}
}

func TestGetDocumentHash_DelegaParaIndexState(t *testing.T) {
	ctx := context.Background()
	database, err := InitDB(filepath.Join(t.TempDir(), "compat_test.db"))
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	if err := InsertDocumentWithMeta(ctx, database, "d", "notes/d.md", "D", time.Now().Unix(), "hh", "ee", "R", "resource"); err != nil {
		t.Fatalf("insert falhou: %v", err)
	}

	hash, err := GetDocumentHash(ctx, database, "d")
	if err != nil {
		t.Fatalf("GetDocumentHash falhou: %v", err)
	}
	if hash != "hh" {
		t.Errorf("GetDocumentHash: got %q quer %q", hash, "hh")
	}
}
