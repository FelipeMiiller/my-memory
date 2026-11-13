package main

import (
	"context"
	"database/sql"
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

	// O retorno REAL de runIndexCLI é capturado: é ele que o `main` traduz em
	// os.Exit(1). Uma closure que devolvesse nil fixo deixaria a asserção
	// vacuamente verdadeira — o mutante "indexa tudo e devolve erro" passaria
	// pelo teste.
	var out string
	var errIndex error
	out = captureStdout(t, func() {
		errIndex = runIndexCLI(context.Background(), "repo_teste", []string{
			"--storage=sqlite",
			"--db", dbPath,
			"--repo", "repo_teste",
			vault,
		})
	})
	if errIndex != nil {
		t.Fatalf("indexar sem provider é um modo válido e não pode falhar, mas runIndexCLI devolveu: %v", errIndex)
	}

	provaAvisoConsolidado(t, out, 3)

	if _, err := os.Stat(dbPath); err != nil {
		t.Errorf("o comando não gravou no banco indicado: %v", err)
	}
}

// vetorFake é um vetor normalizado para o fixture: a busca vetorial precisa de
// um vetor com norma 1 para a distância de cosseno fazer sentido.
func vetorFake() []float32 {
	v := make([]float32, embedder.EmbeddingDim)
	v[0] = 1
	return v
}

// embedderFake é um provider de mentira que devolve sempre o mesmo vetor.
type embedderFake struct{}

func (embedderFake) GenerateEmbedding(text string) ([]float32, error) {
	return vetorFake(), nil
}

// newFixtureComVetores cria um vault SQLite com documentos, chunks COM vetor
// real e as linhas TurboQuant — para que a busca tenha o que encontrar tanto
// pelo FTS quanto pelo caminho vetorial.
func newFixtureComVetores(t *testing.T) *sql.DB {
	t.Helper()

	database, err := db.InitDB(filepath.Join(t.TempDir(), "vault_fallback.db"))
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	ctx := context.Background()
	tq := turboquant.NewQuantizer(embedder.EmbeddingDim)
	for _, path := range []string{"nota-arquitetura-alpha.md", "nota-arquitetura-beta.md"} {
		title := strings.TrimSuffix(path, ".md")
		if err := db.InsertDocumentWithMeta(ctx, database, path, path, title, 1700000000, "hash-"+path, "epoch", "", "resource", 1700000000, 128); err != nil {
			t.Fatalf("InsertDocumentWithMeta(%s) falhou: %v", path, err)
		}
		vec := vetorFake()
		chunkID := path + "#0"
		if err := db.InsertChunk(ctx, database, chunkID, path, "arquitetura do sistema "+title, 0, vec); err != nil {
			t.Fatalf("InsertChunk(%s) falhou: %v", chunkID, err)
		}
		cv, err := tq.Quantize(vec)
		if err != nil {
			t.Fatalf("Quantize(%s) falhou: %v", chunkID, err)
		}
		if err := db.InsertTurboQuantChunk(ctx, database, chunkID, cv.Scale, cv.Data); err != nil {
			t.Fatalf("InsertTurboQuantChunk(%s) falhou: %v", chunkID, err)
		}
	}
	return database
}

// TestProviderDegradationBuscaHibridaDeclaraFallbackLexico prova o AC10: sem
// provider, a saída da busca declara que o resultado é APENAS léxico.
//
// A busca já respondia antes (o aviso existia), mas o aviso dizia o que a
// busca ESTAVA FAZENDO — não que o RESULTADO era só textual. A diferença é
// o que o usuário lê: um resultado que parece semântico e não é.
func TestProviderDegradationBuscaHibridaDeclaraFallbackLexico(t *testing.T) {
	database := newFixtureComVetores(t)
	tq := turboquant.NewQuantizer(embedder.EmbeddingDim)

	var out string
	codigo := 0
	out = captureStdout(t, func() {
		codigo = runSearchSQLite(context.Background(), database, nil, tq, "arquitetura", "hybrid", false, 10, 60,
			store.DefaultDecayOptions(), store.SearchOptions{})
	})

	if codigo != 0 {
		t.Errorf("a busca híbrida sem provider deveria responder (código %d):\n%s", codigo, out)
	}
	if !strings.Contains(out, "LEXICAL") {
		t.Errorf("a saída não declara que o resultado é apenas léxico:\n%s", out)
	}
	if !strings.Contains(out, "NÃO usou similaridade vetorial") {
		t.Errorf("a saída não diz que a similaridade vetorial não foi usada:\n%s", out)
	}
	// Não-vacuous: a busca encontrou os documentos (pelo FTS). Sem isto, o
	// teste passaria mesmo com a busca quebrada.
	if !strings.Contains(out, "nota-arquitetura-alpha") {
		t.Errorf("a busca sem provider não trouxe o documento; o teste não exercitou o fallback:\n%s", out)
	}
}

// TestProviderDegradationBuscaComProviderNaoDeclaraFallback prova o outro
// lado do contrato: com provider disponível, a busca NÃO pode dizer que o
// resultado é lexical.
func TestProviderDegradationBuscaComProviderNaoDeclaraFallback(t *testing.T) {
	database := newFixtureComVetores(t)
	tq := turboquant.NewQuantizer(embedder.EmbeddingDim)

	var out string
	codigo := 0
	out = captureStdout(t, func() {
		codigo = runSearchSQLite(context.Background(), database, embedderFake{}, tq, "arquitetura", "hybrid", false, 10, 60,
			store.DefaultDecayOptions(), store.SearchOptions{})
	})

	if codigo != 0 {
		t.Errorf("a busca híbrida com provider deveria responder (código %d):\n%s", codigo, out)
	}
	if strings.Contains(out, "LEXICAL") {
		t.Errorf("a saída declarou fallback léxico com o provider disponível:\n%s", out)
	}
	// Não-vacuous: a busca traz resultado E o caminho vetorial foi exercitado.
	if !strings.Contains(out, "nota-arquitetura-alpha") {
		t.Errorf("a busca com provider não trouxe o documento:\n%s", out)
	}
}

// TestProviderDegradationBuscaVectorSemProviderFalha prova que o modo
// vetorial sem provider continua sendo erro de uso: código diferente de zero.
func TestProviderDegradationBuscaVectorSemProviderFalha(t *testing.T) {
	database := newFixtureComVetores(t)
	tq := turboquant.NewQuantizer(embedder.EmbeddingDim)

	codigo := 0
	captureStdout(t, func() {
		codigo = runSearchSQLite(context.Background(), database, nil, tq, "arquitetura", "vector", false, 10, 60,
			store.DefaultDecayOptions(), store.SearchOptions{})
	})

	if codigo == 0 {
		t.Error("o modo vetorial sem provider deveria sair com código diferente de zero")
	}
}
