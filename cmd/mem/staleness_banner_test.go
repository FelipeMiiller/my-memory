package main

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/config"
	"github.com/FelipeMiiller/my-memory/internal/db"
)

// newStalenessBannerFixture monta um vault temporário com um arquivo
// indexado e editado depois da indexação (o vault fica stale) e devolve as
// opções do helper prontas para uso.
func newStalenessBannerFixture(t *testing.T, stale bool) (stalenessBannerOptions, *sql.DB) {
	t.Helper()

	vaultDir := t.TempDir()
	dbDir := t.TempDir()
	dbPath := filepath.Join(dbDir, "banner.db")

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	docPath := filepath.Join(vaultDir, "nota.md")
	if err := os.WriteFile(docPath, []byte("# Nota\nconteúdo"), 0644); err != nil {
		t.Fatalf("WriteFile falhou: %v", err)
	}

	editedAt := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
	_ = os.Chtimes(docPath, editedAt, editedAt)

	ctx := context.Background()
	if err := db.InsertDocument(ctx, database, "nota", "nota.md", "Nota", editedAt.Unix(), "hash1"); err != nil {
		t.Fatalf("InsertDocument falhou: %v", err)
	}

	if stale {
		// Edição posterior à indexação: o mtime do disco passa o persistido.
		newEdit := time.Now().Truncate(time.Second)
		_ = os.Chtimes(docPath, newEdit, newEdit)
	}

	cfg := config.DefaultConfig()
	cfg.Include = []string{"**/*.md"}
	cfg.Exclude = []string{".git/**"}

	opts := stalenessBannerOptions{
		RootDir:  vaultDir,
		Config:   &cfg,
		Database: database,
	}
	return opts, database
}

// TestStalenessBanner_StaleVaultWritesBanner: num vault desatualizado o
// helper escreve a linha canônica do aviso.
func TestStalenessBanner_StaleVaultWritesBanner(t *testing.T) {
	opts, _ := newStalenessBannerFixture(t, true)

	var buf bytes.Buffer
	printStalenessBanner(context.Background(), &buf, opts)

	out := buf.String()
	if !strings.Contains(out, "⚠️ [Aviso] O índice local está desatualizado") {
		t.Errorf("esperava o banner de staleness na saída, obteve:\n%s", out)
	}
	if !strings.Contains(out, "1 modificado(s)") {
		t.Errorf("esperava a contagem de modificados no banner, obteve:\n%s", out)
	}
	if !strings.Contains(out, "mem index") {
		t.Errorf("esperava a recomendação 'mem index' no banner, obteve:\n%s", out)
	}
}

// TestStalenessBanner_InSyncWritesNothing: com o índice em sincronia nenhum
// banner sai.
func TestStalenessBanner_InSyncWritesNothing(t *testing.T) {
	opts, _ := newStalenessBannerFixture(t, false)

	var buf bytes.Buffer
	printStalenessBanner(context.Background(), &buf, opts)

	if buf.Len() != 0 {
		t.Errorf("esperava stdout vazio com o índice em sincronia, obteve:\n%s", buf.String())
	}
}

// TestStalenessBanner_UsesCachedCheck: a segunda checagem dentro do TTL vem
// do cache do detector — o diretório do vault some e o banner continua, com as
// MESMAS contagens. Com ForceCheck a varredura refeita veria o arquivo como
// deletado (0 modificado(s), 1 removido(s)) e a asserção quebraria.
func TestStalenessBanner_UsesCachedCheck(t *testing.T) {
	opts, _ := newStalenessBannerFixture(t, true)

	// TTL longo: a segunda chamada tem de cair no cache, não no disco.
	sharedStalenessDetector(opts).SetTTL(time.Hour)

	var first bytes.Buffer
	printStalenessBanner(context.Background(), &first, opts)
	if !strings.Contains(first.String(), "1 modificado(s)") {
		t.Fatalf("esperava o banner com 1 modificado(s) na primeira chamada, obteve:\n%s", first.String())
	}

	if err := os.RemoveAll(opts.RootDir); err != nil {
		t.Fatalf("RemoveAll falhou: %v", err)
	}

	var second bytes.Buffer
	printStalenessBanner(context.Background(), &second, opts)

	out := second.String()
	if !strings.Contains(out, "O índice local está desatualizado") {
		t.Errorf("a checagem cacheada deveria continuar emitindo o banner, obteve:\n%s", out)
	}
	if !strings.Contains(out, "1 modificado(s)") {
		t.Errorf("a checagem cacheada deveria repetir as contagens do relatório em cache, obteve:\n%s", out)
	}
	if strings.Contains(out, "1 removido(s)") {
		t.Errorf("o relatório veio do disco (o vault sumiu) e não do cache, obteve:\n%s", out)
	}
}

// TestStalenessBanner_SilentDegradation: uma checagem que falha não escreve
// nada e não derruba o comando.
func TestStalenessBanner_SilentDegradation(t *testing.T) {
	cfg := config.DefaultConfig()
	opts := stalenessBannerOptions{
		RootDir: t.TempDir(),
		Config:  &cfg,
		// Sem Database e sem PgStore: o detector não tem de onde ler.
	}

	var buf bytes.Buffer
	printStalenessBanner(context.Background(), &buf, opts)

	if buf.Len() != 0 {
		t.Errorf("uma checagem que falha não pode escrever nada, obteve:\n%s", buf.String())
	}
}
