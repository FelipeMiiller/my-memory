package watcher

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/db"
)

// TestIndexPersistsFileMTimeAndSize indexa um arquivo com mtime artificially
// antigo e prova que documents.mtime e documents.size recebem os valores REAIS
// do disco, e nao o relogio da indexacao (que e o que updated_at guarda).
func TestIndexPersistsFileMTimeAndSize(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.InitDB(filepath.Join(tmpDir, "mtime.db"))
	if err != nil {
		if strings.Contains(err.Error(), "CGO_ENABLED=0") || strings.Contains(err.Error(), "binary was compiled without cgo") {
			t.Skip("Pulando teste SQLite: ambiente sem CGO")
		}
		t.Fatalf("erro ao iniciar banco SQLite: %v", err)
	}
	defer database.Close()

	notePath := filepath.Join(tmpDir, "nota.md")
	content := "# Nota com mtime antigo\n\nConteudo suficiente para gerar chunk e aresta com [[OutraNota]].\n"
	if err := os.WriteFile(notePath, []byte(content), 0644); err != nil {
		t.Fatalf("erro ao escrever nota: %v", err)
	}

	editedAt := time.Now().Add(-90 * 24 * time.Hour).Truncate(time.Second)
	if err := os.Chtimes(notePath, editedAt, editedAt); err != nil {
		t.Fatalf("erro ao ajustar mtime da nota: %v", err)
	}

	res, err := IndexSingleFileSQLite(context.Background(), database, nil, nil, notePath, false)
	if err != nil {
		t.Fatalf("erro ao indexar nota: %v", err)
	}
	if res.Action != "indexed" {
		t.Fatalf("esperava action 'indexed', obteve '%s'", res.Action)
	}

	info, err := os.Stat(notePath)
	if err != nil {
		t.Fatalf("erro ao stat da nota: %v", err)
	}
	wantMTime := info.ModTime().Unix()
	wantSize := info.Size()

	var gotMTime, gotSize, gotUpdatedAt int64
	if err := database.QueryRow(
		`SELECT mtime, size, updated_at FROM documents WHERE id = ?`, notePath,
	).Scan(&gotMTime, &gotSize, &gotUpdatedAt); err != nil {
		t.Fatalf("erro ao ler documento indexado: %v", err)
	}

	if gotMTime != wantMTime {
		t.Errorf("mtime persistido = %d, esperado %d (mtime real do arquivo)", gotMTime, wantMTime)
	}
	if gotSize != wantSize {
		t.Errorf("size persistido = %d, esperado %d", gotSize, wantSize)
	}
	// A prova de que mtime nao e updated_at: os dois valores precisam divergir,
	// porque a nota foi editada meses antes desta indexacao.
	if gotUpdatedAt == gotMTime {
		t.Errorf("mtime e updated_at iguais (%d): o teste nao discrimina index-time de edit-time", gotMTime)
	}
	if gotUpdatedAt < wantMTime {
		t.Errorf("updated_at (%d) nao pode ser anterior ao mtime do arquivo (%d)", gotUpdatedAt, wantMTime)
	}
}
