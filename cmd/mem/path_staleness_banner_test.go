package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/graph"
)

// newPathStalenessFixture monta um vault temporário com dois nós ligados
// (o `mem path` só renderiza quando a rota existe) e deixa o processo dentro
// dele — o detector usa o diretório de trabalho atual como raiz do vault.
func newPathStalenessFixture(t *testing.T, stale bool) (string, *sql.DB) {
	t.Helper()

	vaultDir := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "path_banner.db")

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	editedAt := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
	for _, rel := range []string{"services/auth.md", "infra/redis.md"} {
		full := filepath.Join(vaultDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("MkdirAll falhou: %v", err)
		}
		if err := os.WriteFile(full, []byte("# "+rel), 0644); err != nil {
			t.Fatalf("WriteFile falhou: %v", err)
		}
		_ = os.Chtimes(full, editedAt, editedAt)
	}

	ctx := context.Background()
	if err := db.InsertDocument(ctx, database, "auth", "services/auth.md", "Auth Service", editedAt.Unix(), "h1"); err != nil {
		t.Fatalf("InsertDocument falhou: %v", err)
	}
	if err := db.InsertDocument(ctx, database, "redis", "infra/redis.md", "Redis Cache", editedAt.Unix(), "h2"); err != nil {
		t.Fatalf("InsertDocument falhou: %v", err)
	}
	if err := db.InsertEdgeWithProps(ctx, database, "auth", "redis", "implements", "EXTRACTED", 1.0); err != nil {
		t.Fatalf("InsertEdgeWithProps falhou: %v", err)
	}

	if stale {
		// Edição posterior à indexação: o mtime do disco passa o persistido.
		edited := filepath.Join(vaultDir, "services", "auth.md")
		newEdit := time.Now().Truncate(time.Second)
		_ = os.Chtimes(edited, newEdit, newEdit)
	}

	chdirToVault(t, vaultDir)
	return dbPath, database
}

// TestPathStalenessBannerVemAntesDoResultado: num vault desatualizado o
// `mem path` avisa ANTES do bloco de resultado.
func TestPathStalenessBannerVemAntesDoResultado(t *testing.T) {
	dbPath, database := newPathStalenessFixture(t, true)
	defer database.Close()

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--storage=sqlite", "auth", "redis"}
	if err := runPathCommand(context.Background(), "test-repo", args, &buf); err != nil {
		t.Fatalf("runPathCommand falhou: %v", err)
	}

	out := buf.String()
	bannerIdx := strings.Index(out, "O índice local está desatualizado")
	if bannerIdx < 0 {
		t.Fatalf("esperava o aviso de desatualização na saída do path, obteve:\n%s", out)
	}
	headerIdx := strings.Index(out, "=== Descoberta de Rotas")
	if headerIdx < 0 {
		t.Fatalf("esperava o cabeçalho do path na saída, obteve:\n%s", out)
	}
	if bannerIdx > headerIdx {
		t.Errorf("o aviso tem de sair antes do resultado do path, obteve:\n%s", out)
	}
}

// TestPathStalenessBannerSuprimidoNoJSON: com --json o stdout continua sendo
// só o documento JSON.
func TestPathStalenessBannerSuprimidoNoJSON(t *testing.T) {
	dbPath, database := newPathStalenessFixture(t, true)
	defer database.Close()

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--storage=sqlite", "--json", "auth", "redis"}
	if err := runPathCommand(context.Background(), "test-repo", args, &buf); err != nil {
		t.Fatalf("runPathCommand --json falhou: %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "O índice local está desatualizado") {
		t.Errorf("o aviso não pode aparecer no stdout do --json, obteve:\n%s", out)
	}

	var res graph.PathResult
	if err := json.Unmarshal(buf.Bytes(), &res); err != nil {
		t.Fatalf("falha ao desserializar JSON do path: %v\nJSON: %s", err, out)
	}
	if !res.Found {
		t.Errorf("esperava a rota encontrada no JSON do path")
	}
}

// TestPathStalenessBannerAusenteEmSync: com o índice em sincronia o
// `mem path` não imprime aviso nenhum.
func TestPathStalenessBannerAusenteEmSync(t *testing.T) {
	dbPath, database := newPathStalenessFixture(t, false)
	defer database.Close()

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--storage=sqlite", "auth", "redis"}
	if err := runPathCommand(context.Background(), "test-repo", args, &buf); err != nil {
		t.Fatalf("runPathCommand falhou: %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "O índice local está desatualizado") {
		t.Errorf("não deveria haver aviso de desatualização com o índice em sincronia, obteve:\n%s", out)
	}
	if !strings.Contains(out, "=== Descoberta de Rotas") {
		t.Errorf("o path deveria continuar normalmente, obteve:\n%s", out)
	}
}
