package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/config"
	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/embedder"
	"github.com/FelipeMiiller/my-memory/internal/turboquant"
)

// avisoDegradacao é o texto do aviso consolidado que o `mem index` emite
// quando não há provider vetorial.
const avisoDegradacao = "Provider vetorial indisponível"

// provaAvisoConsolidado confere o contrato do aviso (AC9, AC10, AC11):
//
//   - aparece UMA vez por execução, não uma vez por documento;
//   - nomeia a tag que falta (`onnx`);
//   - diz a consequência (documentos sem vetor) e quantos foram;
//   - a linha antiga, por documento, não existe mais.
func provaAvisoConsolidado(t *testing.T, out string, docsEsperados int) {
	t.Helper()

	if n := strings.Count(out, avisoDegradacao); n != 1 {
		t.Errorf("o aviso de degradação apareceu %d vezes, deveria aparecer 1:\n%s", n, out)
	}
	if !strings.Contains(out, "tag `onnx`") {
		t.Errorf("o aviso não nomeia a tag ausente:\n%s", out)
	}
	if !strings.Contains(out, "SEM VETOR") {
		t.Errorf("o aviso não diz a consequência (documentos sem vetor):\n%s", out)
	}
	if !strings.Contains(out, "3 documento(s)") {
		t.Errorf("o aviso não quantifica os documentos sem vetor:\n%s", out)
	}
	if docsEsperados != 3 {
		t.Errorf("o teste esperava 3 documentos, veio %d", docsEsperados)
	}
	if strings.Contains(out, "sem provider de embedding;") {
		t.Errorf("o aviso por documento continua existindo:\n%s", out)
	}
	if linhaConclusao(out) == "" {
		t.Errorf("o comando não emitiu o resumo final:\n%s", out)
	}
}

// TestProviderDegradationIndexPostgresAvisaUmaVez prova o aviso consolidado
// no caminho Postgres, com o store falso (sem rede).
func TestProviderDegradationIndexPostgresAvisaUmaVez(t *testing.T) {
	vault := t.TempDir()
	for _, nome := range []string{"a-um.md", "b-dois.md", "c-tres.md"} {
		if err := os.WriteFile(filepath.Join(vault, nome), []byte("# Nota\n\nconteudo\n"), 0o644); err != nil {
			t.Fatalf("criando nota %s: %v", nome, err)
		}
	}

	fake := &fakePostgresStore{}
	cfg := config.DefaultConfig()

	out := captureStdout(t, func() {
		runIndexPostgres(context.Background(), fake, nil, &cfg, "repo_teste", vault, true, true)
	})

	provaAvisoConsolidado(t, out, 3)
}

// TestProviderDegradationIndexSQLiteAvisaUmaVez prova o MESMO contrato no
// caminho SQLite, com o driver real.
func TestProviderDegradationIndexSQLiteAvisaUmaVez(t *testing.T) {
	vault := t.TempDir()
	for _, nome := range []string{"a-um.md", "b-dois.md", "c-tres.md"} {
		if err := os.WriteFile(filepath.Join(vault, nome), []byte("# Nota\n\nconteudo\n"), 0o644); err != nil {
			t.Fatalf("criando nota %s: %v", nome, err)
		}
	}

	database, err := db.InitDB(filepath.Join(t.TempDir(), "vault.db"))
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()
	tq := turboquant.NewQuantizer(embedder.EmbeddingDim)

	out := captureStdout(t, func() {
		runIndexSQLite(context.Background(), database, nil, tq, &cfg, vault, true, false)
	})

	provaAvisoConsolidado(t, out, 3)
}

// TestProviderDegradationIndexTerminaComCodigoZero prova o AC11: indexar sem
// provider é um modo válido, então o comando NÃO pode falhar.
//
// O `mem index` só chama os.Exit(1) quando runIndexCLI devolve erro. O teste
// roda o comando inteiro contra um vault temporário, com o storage forçado
// para SQLite: nenhum Postgres, nenhuma rede, nenhum arquivo fora do diretório
// temporário do teste.
func TestProviderDegradationIndexTerminaComCodigoZero(t *testing.T) {
	vault := t.TempDir()
	for _, nome := range []string{"a-um.md", "b-dois.md", "c-tres.md"} {
		if err := os.WriteFile(filepath.Join(vault, nome), []byte("# Nota\n\nconteudo\n"), 0o644); err != nil {
			t.Fatalf("criando nota %s: %v", nome, err)
		}
	}
	dbPath := filepath.Join(t.TempDir(), "memory.db")

	// O provider tem que estar indisponível por CONTA DO TESTE, e não por
	// acaso da máquina: numa build com a tag `onnx` e com os artefatos
	// instalados, o comando indexaria com vetor de verdade e o teste não
	// exercitaria nada.
	t.Setenv("MY_MEMORY_MINILM_MODEL_DIR", filepath.Join(t.TempDir(), "sem-modelo"))
	t.Setenv("MY_MEMORY_ORT_LIB", filepath.Join(t.TempDir(), "sem-runtime", "onnxruntime.dll"))

	var out string
	err := func() error {
		out = captureStdout(t, func() {
			runIndexCLI(context.Background(), "repo_teste", []string{
				"--storage=sqlite",
				"--db", dbPath,
				"--repo", "repo_teste",
				vault,
			})
		})
		return nil
	}()
	if err != nil {
		t.Fatalf("runIndexCLI: %v", err)
	}

	provaAvisoConsolidado(t, out, 3)

	if _, err := os.Stat(dbPath); err != nil {
		t.Errorf("o comando não gravou no banco indicado: %v", err)
	}
}
