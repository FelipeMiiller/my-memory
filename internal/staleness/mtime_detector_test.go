package staleness

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/db"
)

// TestDetector_UsesPersistedMTimeNotIndexTime é o teste que fixa a mudança de
// referência: o arquivo é indexado com updated_at = agora, mas o mtime gravado é
// de 60s atrás. Depois o arquivo é editado para 30s atrás — ainda ANTERIOR ao
// updated_at. A comparação antiga (contra updated_at) não encontraria nada; a
// nova (contra documents.mtime) encontra a edição.
func TestDetector_UsesPersistedMTimeNotIndexTime(t *testing.T) {
	tempDir, cfg, cleanup := setupTestEnvironment(t)
	defer cleanup()

	database, err := db.InitDB(filepath.Join(tempDir, "test.db"))
	if err != nil {
		if strings.Contains(err.Error(), "CGO_ENABLED=0") || strings.Contains(err.Error(), "binary was compiled without cgo") {
			t.Skip("Pulando teste SQLite: ambiente sem CGO")
		}
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	docPath := filepath.Join(tempDir, "nota.md")
	if err := os.WriteFile(docPath, []byte("# Nota\n\nconteudo original"), 0644); err != nil {
		t.Fatalf("WriteFile falhou: %v", err)
	}

	indexedAt := time.Now().Truncate(time.Second)
	firstEdit := indexedAt.Add(-60 * time.Second)
	_ = os.Chtimes(docPath, firstEdit, firstEdit)

	ctx := context.Background()
	// updated_at = instante da indexação; mtime = instante real da edição.
	if err := db.InsertDocumentWithMeta(ctx, database, docPath, "nota.md", "Nota", indexedAt.Unix(), "hash1", "", "Resumo", "resource", firstEdit.Unix(), 41); err != nil {
		t.Fatalf("InsertDocumentWithMeta falhou: %v", err)
	}

	// Segunda edição: 30s atrás, ou seja, POSTERIOR ao mtime persistido mas
	// ANTERIOR ao updated_at. Só a comparação por mtime a enxerga.
	secondEdit := indexedAt.Add(-30 * time.Second)
	if err := os.WriteFile(docPath, []byte("# Nota\n\nconteudo editado novamente"), 0644); err != nil {
		t.Fatalf("WriteFile falhou: %v", err)
	}
	_ = os.Chtimes(docPath, secondEdit, secondEdit)

	d := NewDetector(tempDir, cfg, database, nil, "")
	rep, err := d.ForceCheck(ctx)
	if err != nil {
		t.Fatalf("ForceCheck falhou: %v", err)
	}

	if !rep.IsStale || rep.StaleFilesCount != 1 {
		t.Fatalf("esperava 1 arquivo stale detectado via mtime, obteve stale=%d deleted=%d", rep.StaleFilesCount, rep.DeletedFilesCount)
	}
	if rep.StaleFiles[0] != "nota.md" {
		t.Errorf("arquivo stale reportado = %q, esperado %q", rep.StaleFiles[0], "nota.md")
	}
}

// TestDetector_UnchangedFileIsNotStale cobre o caso negativo: mtime persistido
// igual ao do disco não pode gerar alerta.
func TestDetector_UnchangedFileIsNotStale(t *testing.T) {
	tempDir, cfg, cleanup := setupTestEnvironment(t)
	defer cleanup()

	database, err := db.InitDB(filepath.Join(tempDir, "test.db"))
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	docPath := filepath.Join(tempDir, "estavel.md")
	content := []byte("# Estavel\n\nsem edicoes")
	if err := os.WriteFile(docPath, content, 0644); err != nil {
		t.Fatalf("WriteFile falhou: %v", err)
	}
	editedAt := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
	_ = os.Chtimes(docPath, editedAt, editedAt)

	ctx := context.Background()
	indexedAt := time.Now().Truncate(time.Second)
	if err := db.InsertDocumentWithMeta(ctx, database, docPath, "estavel.md", "Estavel", indexedAt.Unix(), "hash1", "", "Resumo", "resource", editedAt.Unix(), int64(len(content))); err != nil {
		t.Fatalf("InsertDocumentWithMeta falhou: %v", err)
	}

	d := NewDetector(tempDir, cfg, database, nil, "")
	rep, err := d.ForceCheck(ctx)
	if err != nil {
		t.Fatalf("ForceCheck falhou: %v", err)
	}

	if rep.IsStale || rep.StaleFilesCount != 0 {
		t.Errorf("arquivo inalterado não deveria ser stale: %+v", rep)
	}
}

// TestDetector_LegacyRowWithZeroMTimeFallsBack prova que uma linha indexada
// antes de mtime existir não quebra o detector: com mtime = 0 ele volta ao
// comportamento anterior, comparando com updated_at.
func TestDetector_LegacyRowWithZeroMTimeFallsBack(t *testing.T) {
	tempDir, cfg, cleanup := setupTestEnvironment(t)
	defer cleanup()

	database, err := db.InitDB(filepath.Join(tempDir, "test.db"))
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	docPath := filepath.Join(tempDir, "legada.md")
	if err := os.WriteFile(docPath, []byte("# Legada\n\nconteudo"), 0644); err != nil {
		t.Fatalf("WriteFile falhou: %v", err)
	}

	indexedAt := time.Now().Truncate(time.Second)
	oldEdit := indexedAt.Add(-10 * time.Minute)
	_ = os.Chtimes(docPath, oldEdit, oldEdit)

	ctx := context.Background()
	// mtime = 0: linha legada, anterior ao ADR-057.
	if err := db.InsertDocumentWithMeta(ctx, database, docPath, "legada.md", "Legada", indexedAt.Unix(), "hash1", "", "Resumo", "resource", 0, 0); err != nil {
		t.Fatalf("InsertDocumentWithMeta falhou: %v", err)
	}

	d := NewDetector(tempDir, cfg, database, nil, "")

	// 1) Sem mudança: não pode reportar stale (fallback = updated_at = agora,
	//    e o mtime do disco é anterior a ele).
	rep, err := d.ForceCheck(ctx)
	if err != nil {
		t.Fatalf("ForceCheck falhou: %v", err)
	}
	if rep.StaleFilesCount != 0 {
		t.Errorf("linha legada intacta não deveria ser stale: %+v", rep)
	}

	// 2) Editada depois da indexação: o fallback precisa pegar.
	newEdit := indexedAt.Add(5 * time.Minute)
	if err := os.WriteFile(docPath, []byte("# Legada\n\neditada"), 0644); err != nil {
		t.Fatalf("WriteFile falhou: %v", err)
	}
	_ = os.Chtimes(docPath, newEdit, newEdit)

	rep, err = d.ForceCheck(ctx)
	if err != nil {
		t.Fatalf("ForceCheck falhou: %v", err)
	}
	if rep.StaleFilesCount != 1 {
		t.Errorf("linha legada editada deveria ser stale pelo fallback, obteve %d", rep.StaleFilesCount)
	}
}
