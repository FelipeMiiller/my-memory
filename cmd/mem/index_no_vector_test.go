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
	"github.com/FelipeMiiller/my-memory/internal/pgtest"
	"github.com/FelipeMiiller/my-memory/internal/store"
	"github.com/FelipeMiiller/my-memory/internal/turboquant"
)

// TestIndexNoVectorPostgresGravaSemEmbedding prova a AC1 no caminho Postgres:
// indexar SEM provider grava os chunks com `embedding` NULL — e não com um
// vetor zero.
//
// O vetor zero é o defeito: ele "existe" na coluna, então o doctor — que
// procura `embedding IS NULL` — via um vault sem um único vetor como
// saudável. `COUNT(embedding)` conta só os não-NULL, então a asserção é
// exata: sem provider, nenhum chunk pode ter embedding.
//
// O banco é descartável (pgtest) — o Postgres do dono nunca é alvo.
func TestIndexNoVectorPostgresGravaSemEmbedding(t *testing.T) {
	h := pgtest.New(t)

	vault := t.TempDir()
	if err := os.WriteFile(filepath.Join(vault, "nota.md"), []byte("# Nota\n\nconteudo do cofre\n"), 0o644); err != nil {
		t.Fatalf("criando nota: %v", err)
	}

	pgStore, err := store.NewPostgresStore(h.DSN)
	if err != nil {
		t.Fatalf("conectar no PostgreSQL descartável (%s): %v", h.DBName, err)
	}
	defer pgStore.Close()

	cfg := config.DefaultConfig()
	ctx := context.Background()

	out := captureStdout(t, func() {
		runIndexPostgres(ctx, pgStore, nil, &cfg, "repo_teste", vault, true, true)
	})

	var total, comEmbedding int
	err = pgStore.DB().QueryRowContext(ctx,
		`SELECT COUNT(*), COUNT(embedding) FROM chunks WHERE repository = $1`,
		"repo_teste").Scan(&total, &comEmbedding)
	if err != nil {
		t.Fatalf("contando os chunks: %v", err)
	}

	if total == 0 {
		t.Fatalf("nenhum chunk foi indexado; o teste não exercitou o caminho:\n%s", out)
	}
	if comEmbedding != 0 {
		t.Errorf("%d de %d chunks têm embedding gravado, mas o indexador rodou sem provider: o vetor zero voltou a substituir a ausência", comEmbedding, total)
	}

	// A degradação continua visível: um aviso consolidado por execução.
	if !strings.Contains(out, "Provider vetorial indisponível") {
		t.Errorf("o indexador sem provider não avisou a degradação:\n%s", out)
	}
}

// TestIndexNoVectorSQLiteNaoGravaVetorNemTurboQuant prova a AC1 no caminho
// SQLite: indexar SEM provider não deixa linha em `chunks_vec` NEM em
// `chunks_turboquant`.
//
// O TurboQuant importa tanto quanto o vetor bruto: no Windows `db.HasSqliteVec`
// é false, então a busca vetorial lê `chunks_turboquant`. Uma linha
// comprimida com todos os coeficientes zerados é o mesmo defeito — e por ela
// que o chunk continuaria voltando na busca.
func TestIndexNoVectorSQLiteNaoGravaVetorNemTurboQuant(t *testing.T) {
	vault := t.TempDir()
	if err := os.WriteFile(filepath.Join(vault, "nota.md"), []byte("# Nota\n\nconteudo do cofre\n"), 0o644); err != nil {
		t.Fatalf("criando nota: %v", err)
	}

	database, err := db.InitDB(filepath.Join(t.TempDir(), "vault.db"))
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()
	tq := turboquant.NewQuantizer(embedder.EmbeddingDim)
	ctx := context.Background()

	out := captureStdout(t, func() {
		runIndexSQLite(ctx, database, nil, tq, &cfg, vault, true, false)
	})

	var total, comVec, comTurbo int
	err = database.QueryRowContext(ctx, `
		SELECT (SELECT COUNT(*) FROM chunks),
		       (SELECT COUNT(*) FROM chunks_vec),
		       (SELECT COUNT(*) FROM chunks_turboquant)
	`).Scan(&total, &comVec, &comTurbo)
	if err != nil {
		t.Fatalf("contando as linhas de vetor: %v", err)
	}

	if total == 0 {
		t.Fatalf("nenhum chunk foi indexado; o teste não exercitou o caminho:\n%s", out)
	}
	if comVec != 0 {
		t.Errorf("%d de %d chunks têm linha em `chunks_vec`, mas o indexador rodou sem provider: o vetor zero voltou a substituir a ausência", comVec, total)
	}
	if comTurbo != 0 {
		t.Errorf("%d de %d chunks têm linha em `chunks_turboquant`, mas o indexador rodou sem provider: o caminho comprimido ainda guarda o vetor zero", comTurbo, total)
	}

	if !strings.Contains(out, "Provider vetorial indisponível") {
		t.Errorf("o indexador sem provider não avisou a degradação:\n%s", out)
	}
}
