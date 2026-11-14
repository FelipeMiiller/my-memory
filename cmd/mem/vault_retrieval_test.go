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

// As provas E2E do cofre central.
//
// Este arquivo NÃO tem `//go:build onnx`: ele compila nas duas builds e
// pula com mensagem explícita quando o provider embutido não está
// disponível — é o que a spec exige ("nunca verde mentiroso"). Sem a tag
// `onnx` o binário nem tem provider, e um teste que fingisse passar ali
// seria verde-mentira; pular com motivo explícito é o comportamento correto.
//
// A prova de prontidão do cofre central (ADR-062 §B-3 e §B-4) são duas:
// recuperar por SENTIDO e reindexar do zero sem perder o vetor.

// providerDoVaultDeTeste devolve o provider embutido real, ou pula o teste
// com mensagem explícita quando ele não está disponível neste ambiente.
func providerDoVaultDeTeste(t *testing.T) (embedder.Embedder, *turboquant.Quantizer) {
	t.Helper()

	cfg := config.DefaultConfig()
	emb, err := newBuiltinEmbedder(&cfg)
	if err != nil {
		t.Skipf("provider embutido indisponível neste ambiente (%v) — a prova do cofre precisa do modelo e da biblioteca do ONNX Runtime (build com -tags onnx)", err)
	}
	if closer, ok := emb.(interface{ Close() error }); ok {
		t.Cleanup(func() { _ = closer.Close() })
	}
	return emb, turboquant.NewQuantizer(embedder.EmbeddingDim)
}

// cofreCentralDeTeste monta um cofre central descartável: um diretório de
// vault, uma configuração global que aponta para ele e um banco SQLite
// próprio.
//
// A configuração global é redirecionada por `MY_MEMORY_GLOBAL_CONFIG_DIR`
// (o mesmo mecanismo que `config.LoadGlobalConfig` usa), então o teste
// nunca toca no cofre real do dono.
func cofreCentralDeTeste(t *testing.T) (vaultPath, dbPath string) {
	t.Helper()

	globalDir := t.TempDir()
	vaultPath = t.TempDir()
	dbPath = filepath.Join(t.TempDir(), "memory.db")

	t.Setenv(config.GlobalConfigDirEnv, globalDir)
	if err := config.SaveGlobalConfig(&config.GlobalConfig{
		Version:      1,
		CentralVault: config.CentralVaultConfig{Path: vaultPath},
		Storage:      config.StorageConfig{Engine: "sqlite"},
	}); err != nil {
		t.Fatalf("SaveGlobalConfig falhou: %v", err)
	}

	return vaultPath, dbPath
}

// TestVaultRecuperacaoPorSentido prova a AC6: a nota escrita pelo caminho da
// lição (`mem lesson`) volta na busca por uma PARÁFRASE — sem nenhuma
// palavra em comum com o texto dela — e o hit vem da fonte `vector`.
//
// É a prova de prontidão do cofre central (ADR-062 §B-3): um cofre que só
// responde à palavra exata não é memória, é um índice de texto.
func TestVaultRecuperacaoPorSentido(t *testing.T) {
	emb, tq := providerDoVaultDeTeste(t)
	ctx := context.Background()
	vaultPath, dbPath := cofreCentralDeTeste(t)

	// A nota: nenhuma palavra da consulta aparece nela (nem no título, nem
	// no corpo, nem no frontmatter que o indexador lê junto).
	titulo := "Por que registrar o motivo"
	conteudo := "Todo produto novo fica guardado junto do que foi descartado, para nao se perder no tempo."
	consulta := "como justificar uma escolha relevante"

	if err := runLessonCLI(ctx, emb, tq, config.CentralRepoID, []string{
		titulo,
		"--content", conteudo,
		"--db", dbPath,
		"--storage", "sqlite",
		"--repo", config.CentralRepoID,
	}); err != nil {
		t.Fatalf("runLessonCLI: %v", err)
	}

	slug := slugifyLessonTitle(titulo)
	arquivo := filepath.Join(vaultPath, "lessons", slug+".md")
	if _, err := os.Stat(arquivo); err != nil {
		t.Fatalf("a lição não foi gravada no cofre (%s): %v", arquivo, err)
	}

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	var out string
	codigo := 0
	out = captureStdout(t, func() {
		codigo = runSearchSQLite(ctx, database, emb, tq, consulta, "hybrid", false, 10, 60,
			store.DefaultDecayOptions(), store.SearchOptions{})
	})

	if codigo != 0 {
		t.Fatalf("a busca híbrida com o provider real deveria responder (código %d):\n%s", codigo, out)
	}

	// Não-vacuous: a nota tem de estar no resultado.
	if !strings.Contains(out, slug) {
		t.Fatalf("a busca por paráfrase não trouxe a nota do cofre (%s):\n%s", slug, out)
	}

	// A origem do hit é a fonte vetorial.
	if !strings.Contains(out, "vector:") {
		t.Errorf("o hit não veio da fonte vetorial — a recuperação por sentido não aconteceu:\n%s", out)
	}

	// E NÃO da léxica: a consulta não tem palavra em comum com a nota.
	if strings.Contains(out, "fts:") {
		t.Errorf("o hit veio também da fonte léxica, mas a consulta não tem palavra em comum com a nota:\n%s", out)
	}
	if strings.Contains(out, "LEXICAL") {
		t.Errorf("a saída declarou fallback léxico com o provider real disponível:\n%s", out)
	}
}

// TestVaultReindexDoZeroPreservaOsVetores prova a AC7: o cofre reindexado
// do zero com um provider funcionando fica com ZERO chunks sem vetor real.
//
// O caminho é o do reparo: primeiro o vault é indexado SEM provider (que é o
// que deixa os chunks sem vetor), depois é reindexado do zero COM provider
// (`mem index --force`). O reparo tem de consertar — e não só trocar um
// vetor zero por outro vetor zero.
func TestVaultReindexDoZeroPreservaOsVetores(t *testing.T) {
	// O provider é resolvido aqui para o teste PULAR com motivo explícito
	// quando ele não existe — o `mem index --force` resolve o seu próprio
	// provider, dentro do binário.
	_, tq := providerDoVaultDeTeste(t)
	ctx := context.Background()
	vaultPath, dbPath := cofreCentralDeTeste(t)

	conteudo := "---\ntitle: Regra do cofre\ncategory: memory\n---\n\nA regra do cofre fica escrita aqui.\n"
	if err := os.WriteFile(filepath.Join(vaultPath, "regra.md"), []byte(conteudo), 0o644); err != nil {
		t.Fatalf("criando a nota: %v", err)
	}

	// 1. Indexa SEM provider: os chunks ficam sem vetor real.
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	cfg := config.DefaultConfig()
	captureStdout(t, func() {
		runIndexSQLite(ctx, database, nil, tq, &cfg, vaultPath, true, false)
	})

	docsAntes, chunksAntes := contagemDoVault(t, database)
	semVetorAntes := chunksDesincronizados(t, database)
	if semVetorAntes == 0 {
		t.Fatalf("o teste não exercitou o caminho: indexar sem provider deixou %d chunk(s) sem vetor, mas o doctor não reportou nenhum", chunksAntes)
	}
	database.Close()

	// 2. Reindexa do zero COM o provider real.
	if err := runIndexCLI(ctx, config.CentralRepoID, []string{
		"--storage=sqlite",
		"--db", dbPath,
		"--repo", config.CentralRepoID,
		"--force",
		vaultPath,
	}); err != nil {
		t.Fatalf("runIndexCLI: %v", err)
	}

	// 3. Confere: nenhum chunk sem vetor real, e nada perdido no caminho.
	database2, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database2.Close()

	docsDepois, chunksDepois := contagemDoVault(t, database2)
	if semVetorDepois := chunksDesincronizados(t, database2); semVetorDepois != 0 {
		t.Errorf("depois do reindex do zero, %d chunk(s) continuam sem vetor real (o provider deveria ter resolvido)", semVetorDepois)
	}
	if docsDepois != docsAntes {
		t.Errorf("o reindex mudou a contagem de documentos: %d antes, %d depois", docsAntes, docsDepois)
	}
	if chunksDepois != chunksAntes {
		t.Errorf("o reindex mudou a contagem de chunks: %d antes, %d depois", chunksAntes, chunksDepois)
	}
}

// contagemDoVault devolve quantos documentos e quantos chunks o vault tem.
func contagemDoVault(t *testing.T, database *sql.DB) (documentos, chunks int) {
	t.Helper()

	err := database.QueryRowContext(context.Background(), `
		SELECT (SELECT COUNT(*) FROM documents), (SELECT COUNT(*) FROM chunks)
	`).Scan(&documentos, &chunks)
	if err != nil {
		t.Fatalf("contando o vault: %v", err)
	}
	return documentos, chunks
}

// chunksDesincronizados devolve quantos chunks estão sem vetor real — a
// mesma definição que o doctor usa (ausente OU vetor zero, em qualquer
// caminho de armazenamento).
func chunksDesincronizados(t *testing.T, database *sql.DB) int {
	t.Helper()

	report, err := db.DiagnoseHealth(context.Background(), database, t.TempDir())
	if err != nil {
		t.Fatalf("DiagnoseHealth falhou: %v", err)
	}
	return len(report.DesyncedChunks)
}
