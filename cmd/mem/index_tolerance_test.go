package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/config"
	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/embedder"
	"github.com/FelipeMiiller/my-memory/internal/store"
	"github.com/FelipeMiiller/my-memory/internal/turboquant"
)

// TestMemIndexPostgresToleranteAFalhaPorDocumento prova o contrato novo do
// `mem index` (ISSUE-032, opção C): a falha de gravação de UM documento não
// aborta a varredura, e a falha aparece no relatório.
//
// A prova é hermética: o store é falso (nenhum Postgres, nenhuma rede, o
// binário `mem` não é chamado) e o vault é um diretório temporário.
func TestMemIndexPostgresToleranteAFalhaPorDocumento(t *testing.T) {
	vault := t.TempDir()
	primeiro := filepath.Join(vault, "a-primeiro.md")
	segundo := filepath.Join(vault, "b-segundo.md")
	terceiro := filepath.Join(vault, "c-terceiro.md")
	for _, p := range []string{primeiro, segundo, terceiro} {
		if err := os.WriteFile(p, []byte("# Nota\n\nconteudo da nota\n"), 0o644); err != nil {
			t.Fatalf("criando nota %s: %v", p, err)
		}
	}

	// O walk é lexical: a-primeiro → b-segundo → c-terceiro. O segundo falha.
	fake := &fakePostgresStore{failDocID: segundo}
	cfg := config.DefaultConfig()

	out := captureStdout(t, func() {
		runIndexPostgres(context.Background(), fake, nil, &cfg, "repo_teste", vault, true, true)
	})

	// (a) O documento DEPOIS da falha foi processado: a varredura não abortou.
	if !contem(fake.inserted, terceiro) {
		t.Errorf("a varredura abortou na falha do documento %q; documentos gravados = %v", segundo, fake.inserted)
	}
	if !contem(fake.inserted, primeiro) {
		t.Errorf("o documento anterior à falha deveria ter sido gravado; documentos gravados = %v", fake.inserted)
	}

	// (c) O documento que falhou NÃO conta como indexado.
	if contem(fake.inserted, segundo) {
		t.Errorf("o documento que falhou foi contado como indexado: %v", fake.inserted)
	}

	// (4) O documento que falhou NÃO pode ser podado: ele tem que estar em
	// activeDocIDs (que é montado antes da gravação).
	if !contem(fake.activeIDs, segundo) {
		t.Errorf("o documento que falhou saiu de activeDocIDs e pode ser podado: %v", fake.activeIDs)
	}

	// (b) O aviso nomeia o arquivo e traz o erro original.
	if !strings.Contains(out, "falha ao gravar documento") {
		t.Errorf("não houve aviso de falha por documento:\n%s", out)
	}
	if !strings.Contains(out, segundo) {
		t.Errorf("o aviso não nomeia o arquivo que falhou:\n%s", out)
	}
	if !strings.Contains(out, "26000") {
		t.Errorf("o aviso não traz o erro original:\n%s", out)
	}

	// (b) O resumo final quantifica a falha...
	resumo := linhaConclusao(out)
	if resumo == "" {
		t.Fatalf("o comando não emitiu o resumo final:\n%s", out)
	}
	if !strings.Contains(resumo, "2 indexados") {
		t.Errorf("o resumo contou o documento falho como indexado: %q", resumo)
	}
	if !strings.Contains(resumo, "1 falhos") {
		t.Errorf("o resumo não quantifica a falha: %q", resumo)
	}
	// ...e o relatório nomeia o documento que ficou de fora.
	if !strings.Contains(out, "ficaram de fora do índice") {
		t.Errorf("o relatório final não nomeia o documento que ficou de fora:\n%s", out)
	}
}

// TestMemIndexSQLiteToleranteAFalhaPorDocumento prova que o MESMO contrato
// vale no caminho SQLite: a opção C é do comando, não do driver.
//
// Aqui não há store falso — o erro é do driver real, induzido por um trigger
// que aborta a gravação de um documento.
func TestMemIndexSQLiteToleranteAFalhaPorDocumento(t *testing.T) {
	vault := t.TempDir()
	primeiro := filepath.Join(vault, "a-primeiro.md")
	segundo := filepath.Join(vault, "b-segundo.md")
	terceiro := filepath.Join(vault, "c-terceiro.md")
	for _, p := range []string{primeiro, segundo, terceiro} {
		if err := os.WriteFile(p, []byte("# Nota\n\nconteudo da nota\n"), 0o644); err != nil {
			t.Fatalf("criando nota %s: %v", p, err)
		}
	}

	dbFile := filepath.Join(t.TempDir(), "vault.db")
	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	// Trigger que faz a gravação do 2º documento falhar.
	if _, err := database.Exec(`
		CREATE TRIGGER falha_gravacao BEFORE INSERT ON documents
		WHEN NEW.path LIKE '%b-segundo.md'
		BEGIN
			SELECT RAISE(ABORT, 'falha simulada de gravacao');
		END;
	`); err != nil {
		t.Fatalf("criando trigger de falha: %v", err)
	}

	cfg := config.DefaultConfig()
	tq := turboquant.NewQuantizer(embedder.EmbeddingDim)

	out := captureStdout(t, func() {
		runIndexSQLite(context.Background(), database, nil, tq, &cfg, vault, true, false)
	})

	// O documento DEPOIS da falha entrou no índice.
	var depois int
	if err := database.QueryRow("SELECT COUNT(*) FROM documents WHERE path = ?", terceiro).Scan(&depois); err != nil {
		t.Fatalf("contando documento posterior: %v", err)
	}
	if depois != 1 {
		t.Errorf("o documento posterior à falha não foi indexado (documents = %d):\n%s", depois, out)
	}

	// O documento que falhou NÃO entrou no índice.
	var falho int
	if err := database.QueryRow("SELECT COUNT(*) FROM documents WHERE path = ?", segundo).Scan(&falho); err != nil {
		t.Fatalf("contando documento que falhou: %v", err)
	}
	if falho != 0 {
		t.Errorf("o documento que falhou entrou no índice: %d linha(s)", falho)
	}

	// O aviso nomeia o arquivo e o resumo quantifica a falha.
	if !strings.Contains(out, "falha ao gravar documento") || !strings.Contains(out, segundo) {
		t.Errorf("não houve aviso de falha por documento com o nome do arquivo:\n%s", out)
	}
	resumo := linhaConclusao(out)
	if resumo == "" {
		t.Fatalf("o comando não emitiu o resumo final:\n%s", out)
	}
	if !strings.Contains(resumo, "2 indexados") {
		t.Errorf("o resumo contou o documento falho como indexado: %q", resumo)
	}
	if !strings.Contains(resumo, "1 falhos") {
		t.Errorf("o resumo não quantifica a falha: %q", resumo)
	}
}

// fakePostgresStore implementa a costura `postgresIndexStore` e falha a
// gravação de um documento escolhido.
//
// É o que permite testar a tolerância a falha por documento sem Postgres:
// nada sai daqui para a rede e nada é gravado fora do diretório temporário
// do teste.
type fakePostgresStore struct {
	failDocID string

	inserted  []string // documentos gravados com sucesso
	activeIDs []string // activeDocIDs recebidos pelo prune
}

func (f *fakePostgresStore) ListDocumentTitles(context.Context, string) ([]string, error) {
	return nil, nil
}

func (f *fakePostgresStore) ListDocumentIDTitles(context.Context, string) (map[string]string, error) {
	return map[string]string{}, nil
}

func (f *fakePostgresStore) GetDocumentIndexState(context.Context, string, string) (string, string, error) {
	return "", "", nil
}

func (f *fakePostgresStore) DeleteDocumentData(context.Context, string, string) error {
	return nil
}

func (f *fakePostgresStore) InsertDocumentWithMeta(_ context.Context, _ string, id, _ string, _ string, _ int64, _ string, _ string, _ string, _ string, _ int64, _ int64) error {
	if id == f.failDocID {
		return errors.New("pq: unnamed prepared statement does not exist (26000)")
	}
	f.inserted = append(f.inserted, id)
	return nil
}

func (f *fakePostgresStore) InsertDocumentTags(context.Context, string, string, []store.TagBinding) error {
	return nil
}

func (f *fakePostgresStore) InsertEdgeWithProps(context.Context, string, string, string, string, string, float64) error {
	return nil
}

func (f *fakePostgresStore) InsertChunk(context.Context, string, string, string, string, int, []float32) error {
	return nil
}

func (f *fakePostgresStore) PruneDeletedDocuments(_ context.Context, _ string, _ string, activeDocIDs []string) ([]string, error) {
	f.activeIDs = append([]string(nil), activeDocIDs...)
	return nil, nil
}

func (f *fakePostgresStore) DeleteDocumentTags(context.Context, string, string) error {
	return nil
}

// linhaConclusao devolve a linha de resumo final do `mem index`.
func linhaConclusao(out string) string {
	for _, ln := range strings.Split(out, "\n") {
		if strings.Contains(ln, "Concluído!") {
			return ln
		}
	}
	return ""
}

// contem diz se a lista tem o item.
func contem(lista []string, item string) bool {
	for _, v := range lista {
		if v == item {
			return true
		}
	}
	return false
}
