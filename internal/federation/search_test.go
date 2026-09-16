package federation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFederatedSearcher_BothSuccess(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	centralDir := filepath.Join(tmpDir, "central-vault")
	if err := os.MkdirAll(centralDir, 0755); err != nil {
		t.Fatal(err)
	}

	localFn := func(ctx context.Context, params SearchParams) ([]SearchResult, error) {
		return []SearchResult{
			{
				ChunkID:    "chunk_loc_1",
				DocumentID: "docs/architecture.md",
				Content:    "Arquitetura local do microsserviço",
			},
			{
				ChunkID:    "chunk_loc_2",
				DocumentID: "docs/api.md",
				Content:    "Endpoints da API local",
			},
		}, nil
	}

	centralFn := func(ctx context.Context, params SearchParams) ([]SearchResult, error) {
		return []SearchResult{
			{
				ChunkID:    "chunk_cen_1",
				DocumentID: "standards/api-standards.md",
				Content:    "Padrão corporativo de REST e HTTP",
			},
			{
				ChunkID:    "chunk_cen_2",
				DocumentID: "security/oauth.md",
				Content:    "Políticas corporativas de OAuth2",
			},
		}, nil
	}

	fs := NewFederatedSearcher(localFn, centralFn, centralDir, true)
	results, err := fs.Search(ctx, SearchParams{
		Query: "arquitetura e api",
		Limit: 10,
		K:     60,
	})
	if err != nil {
		t.Fatalf("erro inesperado na busca federada: %v", err)
	}

	if len(results) != 4 {
		t.Fatalf("esperava 4 resultados combinados, obteve %d", len(results))
	}

	hasLocal := false
	hasCentral := false
	for _, r := range results {
		if strings.HasPrefix(r.DocumentID, "[local] ") {
			hasLocal = true
		}
		if strings.HasPrefix(r.DocumentID, "[central] ") {
			hasCentral = true
		}
		if r.Score <= 0 {
			t.Errorf("esperava Score RRF positivo, obteve %f para %s", r.Score, r.DocumentID)
		}
	}

	if !hasLocal {
		t.Errorf("não encontrou resultados com tag [local]")
	}
	if !hasCentral {
		t.Errorf("não encontrou resultados com tag [central]")
	}
}

func TestFederatedSearcher_CentralFailsFallback(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	centralDir := filepath.Join(tmpDir, "central-vault")
	if err := os.MkdirAll(centralDir, 0755); err != nil {
		t.Fatal(err)
	}

	localFn := func(ctx context.Context, params SearchParams) ([]SearchResult, error) {
		return []SearchResult{
			{
				ChunkID:    "loc_1",
				DocumentID: "local_doc.md",
				Content:    "Conteúdo local preservado",
			},
		}, nil
	}

	centralFn := func(ctx context.Context, params SearchParams) ([]SearchResult, error) {
		return nil, errors.New("Google Drive unmounted / connection timeout")
	}

	fs := NewFederatedSearcher(localFn, centralFn, centralDir, true)
	var loggedMsg string
	fs.SetLogger(func(format string, args ...any) {
		loggedMsg = format
	})

	results, err := fs.Search(ctx, SearchParams{
		Query: "teste",
	})
	if err != nil {
		t.Fatalf("esperava fallback suave sem erro, obteve: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("esperava 1 resultado local retornado no fallback, obteve %d", len(results))
	}
	if loggedMsg == "" {
		t.Errorf("esperava aviso registrado pelo logger na falha do cofre central")
	}
}

func TestFederatedSearcher_LocalFailsReturnsCentral(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	centralDir := filepath.Join(tmpDir, "central-vault")
	if err := os.MkdirAll(centralDir, 0755); err != nil {
		t.Fatal(err)
	}

	localFn := func(ctx context.Context, params SearchParams) ([]SearchResult, error) {
		return nil, errors.New("fora de repositório git")
	}

	centralFn := func(ctx context.Context, params SearchParams) ([]SearchResult, error) {
		return []SearchResult{
			{
				ChunkID:    "cen_1",
				DocumentID: "standards/overview.md",
				Content:    "Normas globais",
			},
		}, nil
	}

	fs := NewFederatedSearcher(localFn, centralFn, centralDir, true)
	results, err := fs.Search(ctx, SearchParams{
		Query: "normas",
	})
	if err != nil {
		t.Fatalf("esperava sucesso servindo central, obteve: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("esperava 1 resultado do central, obteve %d", len(results))
	}
	if !strings.HasPrefix(results[0].DocumentID, "[central] ") {
		t.Errorf("esperava tag [central] no resultado, obteve: %s", results[0].DocumentID)
	}
}

func TestMergeFederatedResults_RankingAndDeduplication(t *testing.T) {
	local := []SearchResult{
		{
			ChunkID:    "shared_chunk",
			DocumentID: "docs/readme.md",
			Content:    "Documentação comum",
		},
		{
			ChunkID:    "local_only",
			DocumentID: "docs/internal.md",
			Content:    "Documento interno",
		},
	}

	central := []SearchResult{
		{
			ChunkID:    "shared_chunk",
			DocumentID: "standards/readme.md",
			Content:    "Documentação comum central",
		},
		{
			ChunkID:    "central_only",
			DocumentID: "standards/rfc.md",
			Content:    "RFC corporativa",
		},
	}

	// Com k=60
	merged := MergeFederatedResults(local, central, 60, 10)

	// O chunk compartilhado apareceu em ambas as listas, seu score deve ser maior que os outros
	if len(merged) != 3 {
		t.Fatalf("esperava 3 itens após dedup por chunk_id, obteve %d", len(merged))
	}

	if merged[0].ChunkID != "shared_chunk" {
		t.Errorf("esperava item compartilhado em 1º lugar com score somado, obteve %s", merged[0].ChunkID)
	}

	// Testa limite de truncamento
	limited := MergeFederatedResults(local, central, 60, 2)
	if len(limited) != 2 {
		t.Errorf("esperava limite de 2 itens, obteve %d", len(limited))
	}
}

func TestResolveCentralSQLitePath(t *testing.T) {
	tmpDir := t.TempDir()
	storageDir := filepath.Join(tmpDir, ".memory", "storage")
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		t.Fatal(err)
	}
	dbFile := filepath.Join(storageDir, "memory.db")
	if err := os.WriteFile(dbFile, []byte("sqlite data"), 0644); err != nil {
		t.Fatal(err)
	}

	resolved := ResolveCentralSQLitePath(tmpDir)
	if resolved != dbFile {
		t.Errorf("esperava %s, obteve %s", dbFile, resolved)
	}
}
