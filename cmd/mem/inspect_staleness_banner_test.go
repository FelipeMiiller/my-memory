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

// newInspectStalenessFixture monta um vault temporário com um nó inspecionável
// e deixa o processo dentro dele — o detector usa o diretório de trabalho
// atual como raiz do vault.
func newInspectStalenessFixture(t *testing.T, stale bool) (string, *sql.DB) {
	t.Helper()

	vaultDir := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "inspect_banner.db")

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	editedAt := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
	docPath := filepath.Join(vaultDir, "docs", "alvo.md")
	if err := os.MkdirAll(filepath.Dir(docPath), 0755); err != nil {
		t.Fatalf("MkdirAll falhou: %v", err)
	}
	if err := os.WriteFile(docPath, []byte("# Alvo\n\nconteúdo do nó inspecionado"), 0644); err != nil {
		t.Fatalf("WriteFile falhou: %v", err)
	}
	_ = os.Chtimes(docPath, editedAt, editedAt)

	ctx := context.Background()
	if err := db.InsertDocument(ctx, database, "alvo", "docs/alvo.md", "Alvo", editedAt.Unix(), "h1"); err != nil {
		t.Fatalf("InsertDocument falhou: %v", err)
	}
	if err := db.InsertChunk(ctx, database, "alvo#0", "alvo", "conteúdo do nó inspecionado", 0, nil); err != nil {
		t.Fatalf("InsertChunk falhou: %v", err)
	}

	if stale {
		// Edição posterior à indexação: o mtime do disco passa o persistido.
		newEdit := time.Now().Truncate(time.Second)
		_ = os.Chtimes(docPath, newEdit, newEdit)
	}

	chdirToVault(t, vaultDir)
	return dbPath, database
}

// TestInspectStalenessBannerVemAntesDoResultado: num vault desatualizado o
// `mem inspect` avisa ANTES do tríptico.
func TestInspectStalenessBannerVemAntesDoResultado(t *testing.T) {
	dbPath, database := newInspectStalenessFixture(t, true)
	defer database.Close()

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--storage=sqlite", "alvo"}
	if err := runInspectCommand(context.Background(), "test-repo", args, &buf); err != nil {
		t.Fatalf("runInspectCommand falhou: %v", err)
	}

	out := buf.String()
	bannerIdx := strings.Index(out, "O índice local está desatualizado")
	if bannerIdx < 0 {
		t.Fatalf("esperava o aviso de desatualização na saída do inspect, obteve:\n%s", out)
	}
	headerIdx := strings.Index(out, "Visualização Cirúrgica de Nó")
	if headerIdx < 0 {
		t.Fatalf("esperava o cabeçalho do inspect na saída, obteve:\n%s", out)
	}
	if bannerIdx > headerIdx {
		t.Errorf("o aviso tem de sair antes do resultado do inspect, obteve:\n%s", out)
	}
}

// TestInspectStalenessBannerSuprimidoNoJSON: com --json o stdout continua
// sendo só o documento JSON.
func TestInspectStalenessBannerSuprimidoNoJSON(t *testing.T) {
	dbPath, database := newInspectStalenessFixture(t, true)
	defer database.Close()

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--storage=sqlite", "--json", "alvo"}
	if err := runInspectCommand(context.Background(), "test-repo", args, &buf); err != nil {
		t.Fatalf("runInspectCommand --json falhou: %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "O índice local está desatualizado") {
		t.Errorf("o aviso não pode aparecer no stdout do --json, obteve:\n%s", out)
	}

	var view graph.TriptychView
	if err := json.Unmarshal(buf.Bytes(), &view); err != nil {
		t.Fatalf("falha ao desserializar JSON do inspect: %v\nJSON: %s", err, out)
	}
	if view.Target.ID != "alvo" {
		t.Errorf("esperava Target.ID 'alvo', obteve '%s'", view.Target.ID)
	}
}

// TestInspectStalenessBannerAusenteEmSync: com o índice em sincronia o
// `mem inspect` não imprime aviso nenhum.
func TestInspectStalenessBannerAusenteEmSync(t *testing.T) {
	dbPath, database := newInspectStalenessFixture(t, false)
	defer database.Close()

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--storage=sqlite", "alvo"}
	if err := runInspectCommand(context.Background(), "test-repo", args, &buf); err != nil {
		t.Fatalf("runInspectCommand falhou: %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "O índice local está desatualizado") {
		t.Errorf("não deveria haver aviso de desatualização com o índice em sincronia, obteve:\n%s", out)
	}
	if !strings.Contains(out, "Visualização Cirúrgica de Nó") {
		t.Errorf("o inspect deveria continuar normalmente, obteve:\n%s", out)
	}
}
