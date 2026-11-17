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

// newImpactStalenessFixture monta um vault temporário com um nó alvo e um
// dependente, e deixa o processo dentro dele — o detector usa o diretório de
// trabalho atual como raiz do vault.
func newImpactStalenessFixture(t *testing.T, stale bool) (string, *sql.DB) {
	t.Helper()

	vaultDir := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "impact_banner.db")

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	editedAt := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
	for _, rel := range []string{"docs/alvo.md", "docs/dependente.md"} {
		full := filepath.Join(vaultDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("MkdirAll falhou: %v", err)
		}
		if err := os.WriteFile(full, []byte("# "+rel+"\n\nnó da análise de impacto"), 0644); err != nil {
			t.Fatalf("WriteFile falhou: %v", err)
		}
		_ = os.Chtimes(full, editedAt, editedAt)
	}
	docPath := filepath.Join(vaultDir, "docs", "alvo.md")

	ctx := context.Background()
	if err := db.InsertDocument(ctx, database, "alvo", "docs/alvo.md", "Alvo", editedAt.Unix(), "h1"); err != nil {
		t.Fatalf("InsertDocument falhou: %v", err)
	}
	if err := db.InsertDocument(ctx, database, "dependente", "docs/dependente.md", "Dependente", editedAt.Unix(), "h2"); err != nil {
		t.Fatalf("InsertDocument falhou: %v", err)
	}
	if err := db.InsertEdgeWithProps(ctx, database, "dependente", "alvo", "implements", "EXTRACTED", 1.0); err != nil {
		t.Fatalf("InsertEdgeWithProps falhou: %v", err)
	}

	if stale {
		// Edição posterior à indexação: o mtime do disco passa o persistido.
		newEdit := time.Now().Truncate(time.Second)
		_ = os.Chtimes(docPath, newEdit, newEdit)
	}

	chdirToVault(t, vaultDir)
	return dbPath, database
}

// TestImpactStalenessBannerVemAntesDoResultado: num vault desatualizado o
// `mem impact` avisa ANTES da tabela de impacto.
func TestImpactStalenessBannerVemAntesDoResultado(t *testing.T) {
	dbPath, database := newImpactStalenessFixture(t, true)
	defer database.Close()

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--storage=sqlite", "alvo"}
	if err := runImpactCommand(context.Background(), "test-repo", args, &buf); err != nil {
		t.Fatalf("runImpactCommand falhou: %v", err)
	}

	out := buf.String()
	bannerIdx := strings.Index(out, "O índice local está desatualizado")
	if bannerIdx < 0 {
		t.Fatalf("esperava o aviso de desatualização na saída do impact, obteve:\n%s", out)
	}
	headerIdx := strings.Index(out, "Análise de Impacto (Blast Radius)")
	if headerIdx < 0 {
		t.Fatalf("esperava o cabeçalho do impact na saída, obteve:\n%s", out)
	}
	if bannerIdx > headerIdx {
		t.Errorf("o aviso tem de sair antes do resultado do impact, obteve:\n%s", out)
	}
}

// TestImpactStalenessBannerSuprimidoNoJSON: com --json o stdout continua
// sendo só o documento JSON.
func TestImpactStalenessBannerSuprimidoNoJSON(t *testing.T) {
	dbPath, database := newImpactStalenessFixture(t, true)
	defer database.Close()

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--storage=sqlite", "--json", "alvo"}
	if err := runImpactCommand(context.Background(), "test-repo", args, &buf); err != nil {
		t.Fatalf("runImpactCommand --json falhou: %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "O índice local está desatualizado") {
		t.Errorf("o aviso não pode aparecer no stdout do --json, obteve:\n%s", out)
	}

	var res graph.ImpactResult
	if err := json.Unmarshal(buf.Bytes(), &res); err != nil {
		t.Fatalf("falha ao desserializar JSON do impact: %v\nJSON: %s", err, out)
	}
	if res.TargetNode != "alvo" {
		t.Errorf("esperava TargetNode 'alvo', obteve '%s'", res.TargetNode)
	}
}

// TestImpactStalenessBannerAusenteEmSync: com o índice em sincronia o
// `mem impact` não imprime aviso nenhum.
func TestImpactStalenessBannerAusenteEmSync(t *testing.T) {
	dbPath, database := newImpactStalenessFixture(t, false)
	defer database.Close()

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--storage=sqlite", "alvo"}
	if err := runImpactCommand(context.Background(), "test-repo", args, &buf); err != nil {
		t.Fatalf("runImpactCommand falhou: %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "O índice local está desatualizado") {
		t.Errorf("não deveria haver aviso de desatualização com o índice em sincronia, obteve:\n%s", out)
	}
	if !strings.Contains(out, "Análise de Impacto (Blast Radius)") {
		t.Errorf("o impact deveria continuar normalmente, obteve:\n%s", out)
	}
}
