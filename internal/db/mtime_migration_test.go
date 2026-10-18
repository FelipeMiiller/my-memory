package db

import (
	"strings"
	"testing"
)

// legacyDocumentsDDL reproduz o schema de documents ANTES do ADR-057, sem
// as colunas mtime e size. E o fixture que prova que a migracao adiciona as
// colunas sem perder dados.
const legacyDocumentsDDL = `
CREATE TABLE IF NOT EXISTS documents (
    id TEXT PRIMARY KEY,
    path TEXT NOT NULL UNIQUE,
    title TEXT,
    updated_at INTEGER NOT NULL,
    content_hash TEXT,
    embed_epoch TEXT,
    abstract TEXT,
    category TEXT DEFAULT 'resource'
);
`

// TestMigrationAddsMTimeAndSizeWithoutDataLoss abre um banco legado (sem mtime
// nem size) e roda a migracao de producao, provando que:
//   - as duas colunas existem depois;
//   - a linha preexistente sobrevive intacta;
//   - mtime e size terminam em 0 (sinal explicito de "indexado antes de mtime
//     existir"), nunca num mtime inventado.
func TestMigrationAddsMTimeAndSizeWithoutDataLoss(t *testing.T) {
	dbPath := t.TempDir() + "/legacy.db"

	legacy, err := openDB(dbPath)
	if err != nil {
		t.Fatalf("erro ao abrir banco legado: %v", err)
	}
	if _, err := legacy.Exec(legacyDocumentsDDL); err != nil {
		t.Fatalf("erro ao criar schema legado: %v", err)
	}
	const legacyUpdatedAt = 1700000000
	if _, err := legacy.Exec(
		`INSERT INTO documents (id, path, title, updated_at, content_hash, category) VALUES (?, ?, ?, ?, ?, ?)`,
		"legacy-doc", "notas/antiga.md", "Nota Antiga", legacyUpdatedAt, "hash-antigo", "memory",
	); err != nil {
		t.Fatalf("erro ao inserir documento legado: %v", err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatalf("erro ao fechar banco legado: %v", err)
	}

	database, err := InitDB(dbPath)
	if err != nil {
		if strings.Contains(err.Error(), "CGO_ENABLED=0") || strings.Contains(err.Error(), "binary was compiled without cgo") {
			t.Skip("Pulando teste SQLite: ambiente sem CGO")
		}
		t.Fatalf("InitDB falhou na migracao: %v", err)
	}
	defer database.Close()

	for _, col := range []string{"mtime", "size"} {
		var cnt int
		if err := database.QueryRow(
			"SELECT COUNT(*) FROM pragma_table_info('documents') WHERE name = ?", col,
		).Scan(&cnt); err != nil {
			t.Fatalf("erro ao inspecionar coluna %s: %v", col, err)
		}
		if cnt != 1 {
			t.Fatalf("coluna %s ausente apos migracao (count=%d)", col, cnt)
		}
	}

	var (
		id, path, title, contentHash, category string
		updatedAt, mtime, size                 int64
	)
	err = database.QueryRow(
		`SELECT id, path, title, updated_at, content_hash, category, mtime, size
		 FROM documents WHERE id = ?`, "legacy-doc",
	).Scan(&id, &path, &title, &updatedAt, &contentHash, &category, &mtime, &size)
	if err != nil {
		t.Fatalf("documento legado sumiu na migracao: %v", err)
	}

	if id != "legacy-doc" || path != "notas/antiga.md" || title != "Nota Antiga" {
		t.Errorf("dados da linha mudaram: id=%q path=%q title=%q", id, path, title)
	}
	if contentHash != "hash-antigo" || category != "memory" {
		t.Errorf("metadados da linha mudaram: content_hash=%q category=%q", contentHash, category)
	}
	if updatedAt != legacyUpdatedAt {
		t.Errorf("updated_at deve preservar o valor original: esperado %d, obteve %d", legacyUpdatedAt, updatedAt)
	}
	if mtime != 0 {
		t.Errorf("mtime de linha legada deveria ser 0, obteve %d", mtime)
	}
	if size != 0 {
		t.Errorf("size de linha legada deveria ser 0, obteve %d", size)
	}
}

// TestMigrationIsIdempotent prova que rodar a migracao duas vezes nao falha
// nem duplica colunas.
func TestMigrationIsIdempotent(t *testing.T) {
	dbPath := t.TempDir() + "/twice.db"

	first, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("primeiro InitDB falhou: %v", err)
	}
	first.Close()

	second, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("segundo InitDB falhou (migracao nao idempotente): %v", err)
	}
	defer second.Close()

	var cnt int
	if err := second.QueryRow(
		"SELECT COUNT(*) FROM pragma_table_info('documents') WHERE name = 'mtime'",
	).Scan(&cnt); err != nil {
		t.Fatalf("erro ao inspecionar coluna mtime: %v", err)
	}
	if cnt != 1 {
		t.Fatalf("mtime deveria existir exatamente uma vez, obteve %d", cnt)
	}
}
