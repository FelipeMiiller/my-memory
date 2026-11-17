package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"sync"

	"github.com/FelipeMiiller/my-memory/internal/config"
	"github.com/FelipeMiiller/my-memory/internal/staleness"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// stalenessBannerOptions descreve a origem do relatório de staleness que o
// banner da CLI usa.
//
// O comando de consulta já tem a conexão aberta quando chama o helper: o
// detector reaproveita essa mesma conexão em vez de abrir outra.
type stalenessBannerOptions struct {
	// RootDir é a raiz do vault auditado. Vazio = diretório de trabalho atual
	// (mesmo default do `mem status`, sem inventar flag nova).
	RootDir string

	// Config é a configuração de escopo (include/exclude). Nil = resolveConfig().
	Config *config.Config

	// Database é a conexão SQLite já aberta pelo comando.
	Database *sql.DB

	// PgStore é a conexão Postgres já aberta pelo comando.
	PgStore *store.PostgresStore

	// Repo é o slug do repositório (usado apenas no Postgres).
	Repo string
}

// sameSource diz se duas opções apontam para a MESMA origem de dados.
//
// A comparação é por ponteiro de conexão: o detector guarda o handle do banco,
// e um handle novo (ou fechado) invalida o cache do detector anterior.
func (o stalenessBannerOptions) sameSource(other stalenessBannerOptions) bool {
	return o.RootDir == other.RootDir &&
		o.Config == other.Config &&
		o.Database == other.Database &&
		o.PgStore == other.PgStore &&
		o.Repo == other.Repo
}

// stalenessBannerSession é o detector compartilhado do processo.
type stalenessBannerSession struct {
	opts     stalenessBannerOptions
	detector *staleness.Detector
}

var (
	stalenessBannerMu    sync.Mutex
	stalenessBannerCache *stalenessBannerSession
)

// sharedStalenessDetector devolve o detector de staleness do processo.
//
// O detector é compartilhado de propósito: o cache de 3 s vive dentro dele, e
// um detector novo a cada chamada transformaria a checagem cacheada em varredura
// do disco a cada consulta — exatamente o custo que a ADR-028 §2 quer evitar.
func sharedStalenessDetector(opts stalenessBannerOptions) *staleness.Detector {
	stalenessBannerMu.Lock()
	defer stalenessBannerMu.Unlock()

	if s := stalenessBannerCache; s != nil && s.opts.sameSource(opts) {
		return s.detector
	}

	cfg := opts.Config
	if cfg == nil {
		cfg = resolveConfig()
	}

	detector := staleness.NewDetector(opts.RootDir, cfg, opts.Database, opts.PgStore, opts.Repo)
	stalenessBannerCache = &stalenessBannerSession{opts: opts, detector: detector}
	return detector
}

// printStalenessBanner escreve o aviso de desatualização em out quando o índice
// local está stale.
//
// O aviso é um sinal, não uma barreira: a checagem vem do cache do detector
// (TTL 3 s, nunca uma varredura forçada) e qualquer falha é engolida — sem
// banner, sem erro novo, com o mesmo código de saída do comando. Um aviso de
// staleness não pode ser a razão de uma consulta falhar.
func printStalenessBanner(ctx context.Context, out io.Writer, opts stalenessBannerOptions) {
	if out == nil {
		return
	}

	detector := sharedStalenessDetector(opts)
	if detector == nil {
		return
	}

	report, err := detector.CheckStaleness(ctx)
	if err != nil || report == nil || !report.IsStale {
		return
	}

	banner := staleness.FormatCLIBanner(report)
	if banner == "" {
		return
	}

	fmt.Fprint(out, banner)
}
