package staleness

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/config"
	"github.com/FelipeMiiller/my-memory/internal/db"
)

func setupTestEnvironment(t *testing.T) (string, *config.Config, func()) {
	t.Helper()
	tempDir := t.TempDir()

	cfg := config.DefaultConfig()
	cfg.Include = []string{"**/*.md"}
	cfg.Exclude = []string{".git/**", "node_modules/**"}

	cleanup := func() {
		// Cleanup happens automatically via t.TempDir()
	}

	return tempDir, &cfg, cleanup
}

func TestDetector_NoDatabase(t *testing.T) {
	tempDir, cfg, cleanup := setupTestEnvironment(t)
	defer cleanup()

	d := NewDetector(tempDir, cfg, nil, nil, "")
	_, err := d.CheckStaleness(context.Background())
	if err == nil {
		t.Fatalf("Esperava erro ao rodar sem banco de dados configurado")
	}
}

func TestDetector_InSync(t *testing.T) {
	tempDir, cfg, cleanup := setupTestEnvironment(t)
	defer cleanup()

	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	// Cria arquivo no disco
	docPath := filepath.Join(tempDir, "note1.md")
	now := time.Now().Truncate(time.Second)
	if err := os.WriteFile(docPath, []byte("# Note 1\nHello"), 0644); err != nil {
		t.Fatalf("WriteFile falhou: %v", err)
	}
	_ = os.Chtimes(docPath, now, now)

	// Registra no banco de dados com mesmo mtime
	ctx := context.Background()
	if err := db.InsertDocument(ctx, database, docPath, "note1.md", "Note 1", now.Unix(), "hash1"); err != nil {
		t.Fatalf("InsertDocument falhou: %v", err)
	}

	d := NewDetector(tempDir, cfg, database, nil, "")
	rep, err := d.CheckStaleness(ctx)
	if err != nil {
		t.Fatalf("CheckStaleness falhou: %v", err)
	}

	if rep.IsStale {
		t.Errorf("Esperava IsStale=false para dados sincronizados, obteve true: %+v", rep)
	}
	if rep.StaleFilesCount != 0 || rep.DeletedFilesCount != 0 {
		t.Errorf("Contagens anômalas: stale=%d, deleted=%d", rep.StaleFilesCount, rep.DeletedFilesCount)
	}
	if rep.NewFilesCount != 0 {
		t.Errorf("NewFilesCount esperado 0 para dados sincronizados, obteve %d", rep.NewFilesCount)
	}
	if rep.IndexedCount != 1 || rep.DiskCount != 1 {
		t.Errorf("Contagens de documentos: indexed=%d, disk=%d", rep.IndexedCount, rep.DiskCount)
	}

	// Banners devem ser vazios
	if banner := FormatMarkdownBanner(rep); banner != "" {
		t.Errorf("FormatMarkdownBanner deveria ser vazio para dados sincronizados, obteve: %q", banner)
	}
	if banner := FormatCLIBanner(rep); banner != "" {
		t.Errorf("FormatCLIBanner deveria ser vazio para dados sincronizados, obteve: %q", banner)
	}
}

func TestDetector_ModifiedFile(t *testing.T) {
	tempDir, cfg, cleanup := setupTestEnvironment(t)
	defer cleanup()

	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	docPath := filepath.Join(tempDir, "mod.md")
	now := time.Now().Truncate(time.Second)
	if err := os.WriteFile(docPath, []byte("# Original"), 0644); err != nil {
		t.Fatalf("WriteFile falhou: %v", err)
	}
	// mtime no passado
	pastTime := now.Add(-10 * time.Second)
	_ = os.Chtimes(docPath, pastTime, pastTime)

	ctx := context.Background()
	if err := db.InsertDocument(ctx, database, docPath, "mod.md", "Mod", pastTime.Unix(), "hashold"); err != nil {
		t.Fatalf("InsertDocument falhou: %v", err)
	}

	// Modifica arquivo no disco (mtime mais recente que pastTime + 1s)
	futureTime := pastTime.Add(5 * time.Second)
	_ = os.Chtimes(docPath, futureTime, futureTime)

	d := NewDetector(tempDir, cfg, database, nil, "")
	rep, err := d.CheckStaleness(ctx)
	if err != nil {
		t.Fatalf("CheckStaleness falhou: %v", err)
	}

	if !rep.IsStale {
		t.Fatalf("Esperava IsStale=true após modificação de arquivo")
	}
	if rep.StaleFilesCount != 1 {
		t.Errorf("StaleFilesCount esperado 1, obteve %d", rep.StaleFilesCount)
	}
	if len(rep.StaleFiles) == 0 || rep.StaleFiles[0] != "mod.md" {
		t.Errorf("StaleFiles inesperado: %v", rep.StaleFiles)
	}

	mdBanner := FormatMarkdownBanner(rep)
	if !strings.Contains(mdBanner, "Staleness Warning") || !strings.Contains(mdBanner, "mem index") {
		t.Errorf("FormatMarkdownBanner não contém aviso esperado: %s", mdBanner)
	}

	cliBanner := FormatCLIBanner(rep)
	if !strings.Contains(cliBanner, "O índice local está desatualizado") || !strings.Contains(cliBanner, "mem index") {
		t.Errorf("FormatCLIBanner não contém aviso esperado: %s", cliBanner)
	}
}

// Textos literais do banner canônico (ADR-028 §3). São strings hard-coded de
// propósito: afirmar a constante exportada contra si mesma seria verde por
// construção e não provaria nada. Se alguém trocar o aviso, este teste quebra.
const (
	canonicalMarkdownBanner = "> ⚠️ **Staleness Warning**: O índice de memória está desatualizado em relação aos arquivos no disco (1 arquivo(s) pendente(s): 1 modificado(s), 0 novo(s), 0 deletado(s)). As respostas abaixo podem refletir um estado anterior. Execute `mem index` ou inicie `mem watch`."
	canonicalCLIBanner      = "⚠️ [Aviso] O índice local está desatualizado (1 modificado(s), 0 novo(s), 0 removido(s)). Execute `mem index` ou inicie `mem watch`."
)

// TestFormatBanners_CanonicalLiteral: os dois formatadores emitem exatamente o
// texto da ADR-028 §3, com as contagens do relatório.
func TestFormatBanners_CanonicalLiteral(t *testing.T) {
	rep := &StalenessReport{
		IsStale:           true,
		StaleFilesCount:   1,
		NewFilesCount:     0,
		DeletedFilesCount: 0,
	}

	md := FormatMarkdownBanner(rep)
	if strings.TrimRight(md, "\n") != canonicalMarkdownBanner {
		t.Errorf("FormatMarkdownBanner não emitiu o literal canônico.\nobtido: %q\nesperado: %q", md, canonicalMarkdownBanner)
	}

	cli := FormatCLIBanner(rep)
	if strings.TrimRight(cli, "\n") != canonicalCLIBanner {
		t.Errorf("FormatCLIBanner não emitiu o literal canônico.\nobtido: %q\nesperado: %q", cli, canonicalCLIBanner)
	}

	// O literal aposentado não pode sair por nenhum caminho.
	if strings.Contains(md, "AVISO: Memória Desatualizada") || strings.Contains(cli, "AVISO: Memória Desatualizada") {
		t.Errorf("O literal aposentado do banner ainda é emitido: %q / %q", md, cli)
	}
	if strings.Contains(cli, "alterados/novos") {
		t.Errorf("FormatCLIBanner ainda usa o literal aposentado 'alterados/novos': %q", cli)
	}
}

// TestFormatBanners_ThreeCounts: as três contagens entram no banner e N é a
// soma delas (modificados + novos + deletados).
func TestFormatBanners_ThreeCounts(t *testing.T) {
	rep := &StalenessReport{
		IsStale:           true,
		StaleFilesCount:   2,
		NewFilesCount:     3,
		DeletedFilesCount: 4,
	}

	md := FormatMarkdownBanner(rep)
	if !strings.Contains(md, "(9 arquivo(s) pendente(s): 2 modificado(s), 3 novo(s), 4 deletado(s))") {
		t.Errorf("FormatMarkdownBanner não traz as três contagens com N = soma: %q", md)
	}

	cli := FormatCLIBanner(rep)
	if !strings.Contains(cli, "(2 modificado(s), 3 novo(s), 4 removido(s))") {
		t.Errorf("FormatCLIBanner não traz as três contagens: %q", cli)
	}
}

// TestFormatBanners_EmptyOutsideStale: fora do estado stale os dois
// formatadores devolvem string vazia — nenhum banner vaza por acidente.
func TestFormatBanners_EmptyOutsideStale(t *testing.T) {
	if got := FormatMarkdownBanner(nil); got != "" {
		t.Errorf("FormatMarkdownBanner com relatório nulo deveria devolver string vazia, obteve %q", got)
	}
	if got := FormatCLIBanner(nil); got != "" {
		t.Errorf("FormatCLIBanner com relatório nulo deveria devolver string vazia, obteve %q", got)
	}

	inSync := &StalenessReport{IsStale: false, StaleFilesCount: 3, NewFilesCount: 1, DeletedFilesCount: 2}
	if got := FormatMarkdownBanner(inSync); got != "" {
		t.Errorf("FormatMarkdownBanner em sincronia deveria devolver string vazia, obteve %q", got)
	}
	if got := FormatCLIBanner(inSync); got != "" {
		t.Errorf("FormatCLIBanner em sincronia deveria devolver string vazia, obteve %q", got)
	}
}

func TestDetector_NewUnindexedFile(t *testing.T) {
	tempDir, cfg, cleanup := setupTestEnvironment(t)
	defer cleanup()

	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	// Banco de dados vazio, mas cria arquivo novo no disco
	newDoc := filepath.Join(tempDir, "brand_new.md")
	if err := os.WriteFile(newDoc, []byte("# Brand New"), 0644); err != nil {
		t.Fatalf("WriteFile falhou: %v", err)
	}

	d := NewDetector(tempDir, cfg, database, nil, "")
	rep, err := d.CheckStaleness(ctx)
	if err != nil {
		t.Fatalf("CheckStaleness falhou: %v", err)
	}

	if !rep.IsStale {
		t.Fatalf("Esperava IsStale=true para arquivo novo no disco")
	}
	if rep.NewFilesCount != 1 {
		t.Errorf("NewFilesCount esperado 1, obteve %d", rep.NewFilesCount)
	}
	if len(rep.NewFiles) != 1 || rep.NewFiles[0] != "brand_new.md" {
		t.Errorf("NewFiles inesperado: %v", rep.NewFiles)
	}
	// "Novo" é categoria própria: não pode vazar para "modificado".
	if rep.StaleFilesCount != 0 {
		t.Errorf("StaleFilesCount esperado 0 para arquivo novo, obteve %d", rep.StaleFilesCount)
	}
	if len(rep.StaleFiles) != 0 {
		t.Errorf("StaleFiles deveria estar vazio para arquivo novo, obteve: %v", rep.StaleFiles)
	}
	if rep.DiskCount != 1 || rep.IndexedCount != 0 {
		t.Errorf("Contagens incorretas: disk=%d, indexed=%d", rep.DiskCount, rep.IndexedCount)
	}
}

// TestDetector_SeparatesNewModifiedAndDeleted fixa a separação das três
// categorias da ADR-028 §1: um arquivo novo, um modificado e um deletado
// ocupam uma lista cada — e as contagens são reportadas separadamente.
func TestDetector_SeparatesNewModifiedAndDeleted(t *testing.T) {
	tempDir, cfg, cleanup := setupTestEnvironment(t)
	defer cleanup()

	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	ctx := context.Background()

	// 1. Modificado: indexado com mtime antigo, editado no disco depois.
	modPath := filepath.Join(tempDir, "mod.md")
	pastTime := time.Now().Add(-30 * time.Second).Truncate(time.Second)
	if err := os.WriteFile(modPath, []byte("# Mod"), 0644); err != nil {
		t.Fatalf("WriteFile falhou: %v", err)
	}
	_ = os.Chtimes(modPath, pastTime, pastTime)
	if err := db.InsertDocument(ctx, database, modPath, "mod.md", "Mod", pastTime.Unix(), "hashmod"); err != nil {
		t.Fatalf("InsertDocument falhou: %v", err)
	}
	_ = os.Chtimes(modPath, pastTime.Add(10*time.Second), pastTime.Add(10*time.Second))

	// 2. Novo: existe no disco, nunca foi indexado.
	if err := os.WriteFile(filepath.Join(tempDir, "novo.md"), []byte("# Novo"), 0644); err != nil {
		t.Fatalf("WriteFile falhou: %v", err)
	}

	// 3. Deletado: indexado, mas não existe mais no disco.
	if err := db.InsertDocument(ctx, database, "apagado.md", "apagado.md", "Apagado", pastTime.Unix(), "hashdel"); err != nil {
		t.Fatalf("InsertDocument falhou: %v", err)
	}

	d := NewDetector(tempDir, cfg, database, nil, "")
	rep, err := d.CheckStaleness(ctx)
	if err != nil {
		t.Fatalf("CheckStaleness falhou: %v", err)
	}

	if !rep.IsStale {
		t.Fatalf("Esperava IsStale=true com as três categorias presentes")
	}
	if rep.StaleFilesCount != 1 {
		t.Errorf("StaleFilesCount esperado 1, obteve %d", rep.StaleFilesCount)
	}
	if rep.NewFilesCount != 1 {
		t.Errorf("NewFilesCount esperado 1, obteve %d", rep.NewFilesCount)
	}
	if rep.DeletedFilesCount != 1 {
		t.Errorf("DeletedFilesCount esperado 1, obteve %d", rep.DeletedFilesCount)
	}
	if len(rep.StaleFiles) != 1 || rep.StaleFiles[0] != "mod.md" {
		t.Errorf("StaleFiles inesperado: %v", rep.StaleFiles)
	}
	if len(rep.NewFiles) != 1 || rep.NewFiles[0] != "novo.md" {
		t.Errorf("NewFiles inesperado: %v", rep.NewFiles)
	}
	if len(rep.DeletedFiles) != 1 || rep.DeletedFiles[0] != "apagado.md" {
		t.Errorf("DeletedFiles inesperado: %v", rep.DeletedFiles)
	}
}

// TestDetector_OnlyNewFilesIsStale: um vault com só arquivos novos é stale — a
// separação das listas não pode produzir regressão para "in sync".
func TestDetector_OnlyNewFilesIsStale(t *testing.T) {
	tempDir, cfg, cleanup := setupTestEnvironment(t)
	defer cleanup()

	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	for _, name := range []string{"a.md", "b.md"} {
		if err := os.WriteFile(filepath.Join(tempDir, name), []byte("# "+name), 0644); err != nil {
			t.Fatalf("WriteFile falhou: %v", err)
		}
	}

	d := NewDetector(tempDir, cfg, database, nil, "")
	rep, err := d.CheckStaleness(context.Background())
	if err != nil {
		t.Fatalf("CheckStaleness falhou: %v", err)
	}

	if !rep.IsStale {
		t.Fatalf("Esperava IsStale=true quando só há arquivos novos")
	}
	if rep.NewFilesCount != 2 {
		t.Errorf("NewFilesCount esperado 2, obteve %d", rep.NewFilesCount)
	}
	if rep.StaleFilesCount != 0 || rep.DeletedFilesCount != 0 {
		t.Errorf("Contagens anômalas: stale=%d, deleted=%d", rep.StaleFilesCount, rep.DeletedFilesCount)
	}
}

func TestDetector_DeletedFile(t *testing.T) {
	tempDir, cfg, cleanup := setupTestEnvironment(t)
	defer cleanup()

	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	// Registra documento no banco que NÃO existe no disco
	if err := db.InsertDocument(ctx, database, "deleted.md", "deleted.md", "Deleted", time.Now().Unix(), "hashdel"); err != nil {
		t.Fatalf("InsertDocument falhou: %v", err)
	}

	d := NewDetector(tempDir, cfg, database, nil, "")
	rep, err := d.CheckStaleness(ctx)
	if err != nil {
		t.Fatalf("CheckStaleness falhou: %v", err)
	}

	if !rep.IsStale {
		t.Fatalf("Esperava IsStale=true para arquivo deletado")
	}
	if rep.DeletedFilesCount != 1 {
		t.Errorf("DeletedFilesCount esperado 1, obteve %d", rep.DeletedFilesCount)
	}
	if rep.IndexedCount != 1 || rep.DiskCount != 0 {
		t.Errorf("Contagens incorretas: indexed=%d, disk=%d", rep.IndexedCount, rep.DiskCount)
	}
}

func TestDetector_CacheTTLAndReset(t *testing.T) {
	tempDir, cfg, cleanup := setupTestEnvironment(t)
	defer cleanup()

	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	d := NewDetector(tempDir, cfg, database, nil, "")
	d.SetTTL(1 * time.Second)

	rep1, err := d.CheckStaleness(ctx)
	if err != nil {
		t.Fatalf("Primeira checagem falhou: %v", err)
	}

	// Adiciona arquivo no disco
	newDoc := filepath.Join(tempDir, "cached_test.md")
	if err := os.WriteFile(newDoc, []byte("# Cached"), 0644); err != nil {
		t.Fatalf("WriteFile falhou: %v", err)
	}

	// Segunda chamada imediata deve reutilizar cache (DiskCount ainda 0)
	rep2, err := d.CheckStaleness(ctx)
	if err != nil {
		t.Fatalf("Segunda checagem falhou: %v", err)
	}
	if rep2.DiskCount != rep1.DiskCount {
		t.Errorf("Cache deveria ter retornado mesmo resultado imediato")
	}

	// ForceCheck deve ignorar o cache e detectar o novo arquivo
	rep3, err := d.ForceCheck(ctx)
	if err != nil {
		t.Fatalf("ForceCheck falhou: %v", err)
	}
	if rep3.DiskCount != 1 {
		t.Errorf("ForceCheck deveria ter detectado o arquivo novo no disco (DiskCount=1, obteve %d)", rep3.DiskCount)
	}
}

func TestDetector_ExcludedDirs(t *testing.T) {
	tempDir, cfg, cleanup := setupTestEnvironment(t)
	defer cleanup()

	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	// Arquivo dentro de .git não deve ser contado
	gitDir := filepath.Join(tempDir, ".git")
	_ = os.MkdirAll(gitDir, 0755)
	_ = os.WriteFile(filepath.Join(gitDir, "COMMIT_EDITMSG.md"), []byte("git log"), 0644)

	ctx := context.Background()
	d := NewDetector(tempDir, cfg, database, nil, "")
	rep, err := d.CheckStaleness(ctx)
	if err != nil {
		t.Fatalf("CheckStaleness falhou: %v", err)
	}

	if rep.DiskCount != 0 {
		t.Errorf("Arquivo dentro de .git não deveria ser contado como disco elegível, obteve: %d", rep.DiskCount)
	}
}
