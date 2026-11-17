package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/staleness"
)

func TestRunStatusCommand_InSync(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "status_test.db")

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}

	docPath := filepath.Join(tempDir, "sync.md")
	now := time.Now().Truncate(time.Second)
	if err := os.WriteFile(docPath, []byte("# Sincronizado"), 0644); err != nil {
		t.Fatalf("WriteFile falhou: %v", err)
	}
	_ = os.Chtimes(docPath, now, now)

	ctx := context.Background()
	if err := db.InsertDocument(ctx, database, docPath, "sync.md", "Sync", now.Unix(), "hash1"); err != nil {
		t.Fatalf("InsertDocument falhou: %v", err)
	}
	database.Close()

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--dir", tempDir}
	if err := runStatusCommand(ctx, "test-repo", args, &buf); err != nil {
		t.Fatalf("runStatusCommand falhou: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "✅ ATUALIZADO (Fresh)") {
		t.Errorf("Esperava mensagem de atualizado canônica, obteve:\n%s", out)
	}
	if !strings.Contains(out, "Arquivos no Disco:  1") || !strings.Contains(out, "Arquivos Indexados: 1") {
		t.Errorf("Contagens incorretas na saída:\n%s", out)
	}
}

func TestRunStatusCommand_Stale(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "status_test.db")

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}

	docPath := filepath.Join(tempDir, "old.md")
	pastTime := time.Now().Add(-20 * time.Second).Truncate(time.Second)
	if err := os.WriteFile(docPath, []byte("# Velho"), 0644); err != nil {
		t.Fatalf("WriteFile falhou: %v", err)
	}
	_ = os.Chtimes(docPath, pastTime, pastTime)

	ctx := context.Background()
	if err := db.InsertDocument(ctx, database, docPath, "old.md", "Old", pastTime.Unix(), "hashold"); err != nil {
		t.Fatalf("InsertDocument falhou: %v", err)
	}
	database.Close()

	// Modifica o arquivo no disco
	newTime := pastTime.Add(10 * time.Second)
	_ = os.Chtimes(docPath, newTime, newTime)

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--dir", tempDir}
	if err := runStatusCommand(ctx, "test-repo", args, &buf); err != nil {
		t.Fatalf("runStatusCommand falhou: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "⚠️ DESATUALIZADO (Stale)") {
		t.Errorf("Esperava mensagem de desatualizado canônica, obteve:\n%s", out)
	}
	if !strings.Contains(out, "Diferenças:         1 modificado(s), 0 novo(s), 0 removido(s)") {
		t.Errorf("Esperava linha de diferenças com três números, obteve:\n%s", out)
	}
	if !strings.Contains(out, "mem index") {
		t.Errorf("Esperava recomendação de mem index, obteve:\n%s", out)
	}
}

func TestRunStatusCommand_JSON(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "status_test.db")

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	database.Close()

	// Cria arquivo sem indexar
	_ = os.WriteFile(filepath.Join(tempDir, "unindexed.md"), []byte("# Novo"), 0644)

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--dir", tempDir, "--json"}
	ctx := context.Background()
	if err := runStatusCommand(ctx, "test-repo", args, &buf); err != nil {
		t.Fatalf("runStatusCommand --json falhou: %v", err)
	}

	var rep staleness.StalenessReport
	if err := json.Unmarshal(buf.Bytes(), &rep); err != nil {
		t.Fatalf("Falha ao desserializar JSON do status: %v, saída: %s", err, buf.String())
	}

	if !rep.IsStale {
		t.Errorf("Esperava IsStale=true no JSON")
	}
	// O arquivo existe no disco e não está indexado: categoria "novo".
	if rep.NewFilesCount != 1 {
		t.Errorf("NewFilesCount esperado 1, obteve %d", rep.NewFilesCount)
	}
	if rep.StaleFilesCount != 0 {
		t.Errorf("StaleFilesCount esperado 0 (arquivo novo não é modificado), obteve %d", rep.StaleFilesCount)
	}
	if rep.DiskCount != 1 || rep.IndexedCount != 0 {
		t.Errorf("Contagens incorretas no JSON: disk=%d, indexed=%d", rep.DiskCount, rep.IndexedCount)
	}
}

func TestRunStatusCommand_ListagemFormatacao(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "status_format_test.db")

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}

	ctx := context.Background()
	pastTime := time.Now().Add(-20 * time.Second).Truncate(time.Second)

	// 1 arquivo modificado
	modPath := filepath.Join(tempDir, "modified.md")
	if err := os.WriteFile(modPath, []byte("# Modificado"), 0644); err != nil {
		t.Fatalf("WriteFile falhou: %v", err)
	}
	_ = os.Chtimes(modPath, pastTime, pastTime)
	if err := db.InsertDocument(ctx, database, modPath, "modified.md", "Modified", pastTime.Unix(), "hashmod"); err != nil {
		t.Fatalf("InsertDocument falhou: %v", err)
	}
	// Edita no disco após indexação
	newTime := pastTime.Add(10 * time.Second)
	_ = os.Chtimes(modPath, newTime, newTime)

	// 1 arquivo deletado (indexado mas não existe no disco)
	deletedPath := filepath.Join(tempDir, "deleted.md")
	if err := db.InsertDocument(ctx, database, deletedPath, "deleted.md", "Deleted", pastTime.Unix(), "hashdel"); err != nil {
		t.Fatalf("InsertDocument falhou: %v", err)
	}

	// 12 arquivos novos (existem no disco mas não estão indexados)
	for i := 1; i <= 12; i++ {
		fname := fmt.Sprintf("new_%02d.md", i)
		p := filepath.Join(tempDir, fname)
		if err := os.WriteFile(p, []byte("# "+fname), 0644); err != nil {
			t.Fatalf("WriteFile falhou: %v", err)
		}
	}
	database.Close()

	// 1. Executa saída humana
	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--dir", tempDir}
	if err := runStatusCommand(ctx, "test-repo", args, &buf); err != nil {
		t.Fatalf("runStatusCommand falhou: %v", err)
	}

	out := buf.String()

	// Checa headings
	if !strings.Contains(out, "Arquivos Modificados (1):") {
		t.Errorf("Esperava heading de modificados, obteve:\n%s", out)
	}
	if !strings.Contains(out, "Arquivos Novos (12):") {
		t.Errorf("Esperava heading de novos, obteve:\n%s", out)
	}
	if !strings.Contains(out, "Arquivos Removidos do Disco (1):") {
		t.Errorf("Esperava heading de removidos, obteve:\n%s", out)
	}

	// Checa que o caminho é relativo e ordenado
	if !strings.Contains(out, "  • modified.md") {
		t.Errorf("Esperava caminho relativo modified.md, obteve:\n%s", out)
	}
	if !strings.Contains(out, "  • deleted.md") {
		t.Errorf("Esperava caminho relativo deleted.md, obteve:\n%s", out)
	}
	if !strings.Contains(out, "  • new_01.md") || !strings.Contains(out, "  • new_10.md") {
		t.Errorf("Esperava caminhos relativos para os 10 primeiros novos, obteve:\n%s", out)
	}

	// Truncamento: 12 arquivos -> 10 primeiros impressos + "... e mais 2 arquivo(s)"
	if strings.Contains(out, "  • new_11.md") || strings.Contains(out, "  • new_12.md") {
		t.Errorf("Não deveria imprimir o 11º ou 12º arquivo novo, obteve:\n%s", out)
	}
	if !strings.Contains(out, "  ... e mais 2 arquivo(s)") {
		t.Errorf("Esperava truncamento '  ... e mais 2 arquivo(s)', obteve:\n%s", out)
	}

	// 2. Executa com --json e confere paridade
	var jsonBuf bytes.Buffer
	jsonArgs := []string{"--db", dbPath, "--dir", tempDir, "--json"}
	if err := runStatusCommand(ctx, "test-repo", jsonArgs, &jsonBuf); err != nil {
		t.Fatalf("runStatusCommand --json falhou: %v", err)
	}

	var rep staleness.StalenessReport
	if err := json.Unmarshal(jsonBuf.Bytes(), &rep); err != nil {
		t.Fatalf("Falha ao desserializar JSON: %v", err)
	}

	if rep.NewFilesCount != 12 || len(rep.NewFiles) != 12 {
		t.Errorf("JSON new_files divergente: count=%d, len=%d", rep.NewFilesCount, len(rep.NewFiles))
	}
	if rep.StaleFilesCount != 1 || len(rep.StaleFiles) != 1 {
		t.Errorf("JSON stale_files divergente: count=%d, len=%d", rep.StaleFilesCount, len(rep.StaleFiles))
	}
	if rep.DeletedFilesCount != 1 || len(rep.DeletedFiles) != 1 {
		t.Errorf("JSON deleted_files divergente: count=%d, len=%d", rep.DeletedFilesCount, len(rep.DeletedFiles))
	}
}
