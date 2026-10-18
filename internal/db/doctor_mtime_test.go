package db

import (
	"context"
	"path/filepath"
	"testing"
)

// TestDoctorReportsMissingMTimeOnLegacyVault prova a metade visível do ADR-057:
// um vault indexado antes de mtime existir tem todas as linhas em 0, e `mem
// doctor` precisa reportar essa contagem em vez de deixá-la silenciosa.
func TestDoctorReportsMissingMTimeOnLegacyVault(t *testing.T) {
	database, err := InitDB(filepath.Join(t.TempDir(), "doctor.db"))
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	ctx := context.Background()

	// Vault recém-indexado: mtime real em todas as linhas.
	fresh := []struct {
		id, path    string
		mtime, size int64
	}{
		{"doc-novo-1", "notas/nova1.md", 1_700_000_000, 120},
		{"doc-novo-2", "notas/nova2.md", 1_700_000_500, 240},
	}
	for _, d := range fresh {
		if err := InsertDocumentWithMeta(ctx, database, d.id, d.path, d.id, 1_700_000_900, "hash", "", "Resumo", "resource", d.mtime, d.size); err != nil {
			t.Fatalf("insert falhou: %v", err)
		}
	}

	report, err := DiagnoseHealth(ctx, database)
	if err != nil {
		t.Fatalf("DiagnoseHealth falhou: %v", err)
	}
	if report.MissingMTimeCount != 0 {
		t.Errorf("vault recém-indexado deveria reportar 0 documentos sem mtime, obteve %d", report.MissingMTimeCount)
	}
	if report.TotalDocuments != 2 {
		t.Errorf("TotalDocuments = %d, esperado 2", report.TotalDocuments)
	}

	// Simula linhas legadas: mtime e size zerados.
	if _, err := database.ExecContext(ctx, `UPDATE documents SET mtime = 0, size = 0 WHERE id = ?`, "doc-novo-1"); err != nil {
		t.Fatalf("simulação de linha legada falhou: %v", err)
	}

	report, err = DiagnoseHealth(ctx, database)
	if err != nil {
		t.Fatalf("DiagnoseHealth falhou: %v", err)
	}
	if report.MissingMTimeCount != 1 {
		t.Errorf("contagem de documentos sem mtime = %d, esperado 1", report.MissingMTimeCount)
	}
}

// TestDoctorMissingMTimeDoesNotAffectHealthScore fixa a decisão de produto: a
// lacuna de metadados é reportada, mas não penaliza o HealthScore — não é
// corrupção de dado.
func TestDoctorMissingMTimeDoesNotAffectHealthScore(t *testing.T) {
	database, err := InitDB(filepath.Join(t.TempDir(), "doctor_score.db"))
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	if err := InsertDocumentWithMeta(ctx, database, "legado", "notas/legada.md", "Legada", 1_700_000_000, "hash", "", "Resumo", "resource", 0, 0); err != nil {
		t.Fatalf("insert falhou: %v", err)
	}

	report, err := DiagnoseHealth(ctx, database)
	if err != nil {
		t.Fatalf("DiagnoseHealth falhou: %v", err)
	}
	if report.MissingMTimeCount != 1 {
		t.Fatalf("MissingMTimeCount = %d, esperado 1", report.MissingMTimeCount)
	}
	scoreWithGap := report.HealthScore

	// Mesma base, agora com mtime capturado: o HealthScore não pode mudar.
	if _, err := database.ExecContext(ctx, `UPDATE documents SET mtime = ? WHERE id = ?`, 1_700_000_000, "legado"); err != nil {
		t.Fatalf("captura de mtime falhou: %v", err)
	}
	after, err := DiagnoseHealth(ctx, database)
	if err != nil {
		t.Fatalf("DiagnoseHealth falhou: %v", err)
	}
	if after.MissingMTimeCount != 0 {
		t.Errorf("MissingMTimeCount = %d após capturar mtime, esperado 0", after.MissingMTimeCount)
	}
	if after.HealthScore != scoreWithGap {
		t.Errorf("HealthScore mudou de %d para %d por causa do mtime: falta de metadado não é corrupção de dado",
			scoreWithGap, after.HealthScore)
	}
}
