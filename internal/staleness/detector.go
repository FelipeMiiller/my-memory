package staleness

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/config"
	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// DefaultTTL é o tempo de expiração padrão do cache de staleness (3 segundos)
const DefaultTTL = 3 * time.Second

// Fonte única dos literais de staleness (ADR-028 §3).
//
// O aviso é um contrato: o agente que lê o banner precisa reconhecê-lo, e o
// teste precisa poder prová-lo. Escrever o texto no consumidor foi o que
// produziu quatro literais divergentes (spec, ADR, código e guias) — por isso
// o título, a recomendação e os rótulos de status vivem aqui.
const (
	// StalenessWarningTitle é o título canônico do banner de staleness.
	StalenessWarningTitle = "Staleness Warning"

	// StalenessRecommendation é a recomendação canônica: reindexar o vault
	// ou deixar o watcher em pé.
	StalenessRecommendation = "Execute `mem index` ou inicie `mem watch`"

	// StatusStaleLabel é o rótulo de status do vault desatualizado.
	StatusStaleLabel = "⚠️ DESATUALIZADO (Stale)"

	// StatusFreshLabel é o rótulo de status do vault em sincronia.
	StatusFreshLabel = "✅ ATUALIZADO (Fresh)"
)

// StalenessReport agrega as métricas de sincronização entre o disco e a base indexada
//
// As três categorias são disjuntas e exigidas pela ADR-028 §1: "Modificado"
// (mtime mais novo que o persistido), "Novo" (no disco, ausente do índice) e
// "Deletado" (no índice, ausente do disco).
type StalenessReport struct {
	IsStale           bool      `json:"is_stale"`
	StaleFilesCount   int       `json:"stale_files_count"`
	NewFilesCount     int       `json:"new_files_count"`
	DeletedFilesCount int       `json:"deleted_files_count"`
	IndexedCount      int       `json:"indexed_count"`
	DiskCount         int       `json:"disk_count"`
	StaleFiles        []string  `json:"stale_files,omitempty"`
	NewFiles          []string  `json:"new_files,omitempty"`
	DeletedFiles      []string  `json:"deleted_files,omitempty"`
	LastCheck         time.Time `json:"last_check"`
}

// Detector executa a verificação de staleness de forma performática com cache em memória
type Detector struct {
	mu         sync.RWMutex
	cachedRep  *StalenessReport
	cacheUntil time.Time
	ttl        time.Duration
	rootDir    string
	cfg        *config.Config
	database   *sql.DB
	pgStore    *store.PostgresStore
	repo       string
}

// NewDetector cria uma nova instância de Detector para SQLite ou PostgreSQL
func NewDetector(rootDir string, cfg *config.Config, database *sql.DB, pgStore *store.PostgresStore, repo string) *Detector {
	if cfg == nil {
		defaultCfg := config.DefaultConfig()
		cfg = &defaultCfg
	}
	cleanRoot := rootDir
	if cleanRoot == "" || cleanRoot == "." {
		if cwd, err := os.Getwd(); err == nil {
			cleanRoot = cwd
		}
	}
	if abs, err := filepath.Abs(cleanRoot); err == nil {
		cleanRoot = abs
	}

	return &Detector{
		ttl:      DefaultTTL,
		rootDir:  cleanRoot,
		cfg:      cfg,
		database: database,
		pgStore:  pgStore,
		repo:     repo,
	}
}

// SetTTL define um tempo de expiração customizado para o cache
func (d *Detector) SetTTL(ttl time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ttl = ttl
}

// ResetCache limpa o cache forçando a próxima chamada a reavaliar o disco
func (d *Detector) ResetCache() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cachedRep = nil
	d.cacheUntil = time.Time{}
}

// CheckStaleness retorna o relatório de staleness, utilizando o cache se válido
func (d *Detector) CheckStaleness(ctx context.Context) (*StalenessReport, error) {
	d.mu.RLock()
	if d.cachedRep != nil && time.Now().Before(d.cacheUntil) {
		rep := *d.cachedRep
		d.mu.RUnlock()
		return &rep, nil
	}
	d.mu.RUnlock()

	d.mu.Lock()
	defer d.mu.Unlock()

	// Double check após adquirir Lock exclusivo
	if d.cachedRep != nil && time.Now().Before(d.cacheUntil) {
		rep := *d.cachedRep
		return &rep, nil
	}

	rep, err := d.computeStaleness(ctx)
	if err != nil {
		return nil, err
	}

	d.cachedRep = rep
	ttl := d.ttl
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	d.cacheUntil = time.Now().Add(ttl)

	repCopy := *rep
	return &repCopy, nil
}

// ForceCheck força a execução imediata ignorando o cache em memória
func (d *Detector) ForceCheck(ctx context.Context) (*StalenessReport, error) {
	d.ResetCache()
	return d.CheckStaleness(ctx)
}

func (d *Detector) computeStaleness(ctx context.Context) (*StalenessReport, error) {
	var metaMap map[string]store.DocumentMeta
	var err error

	if d.database != nil {
		metaMap, err = db.GetDocumentsMetadata(ctx, d.database)
	} else if d.pgStore != nil {
		metaMap, err = d.pgStore.GetDocumentsMetadata(ctx, d.repo)
	} else {
		return nil, errors.New("nenhuma base de dados (SQLite ou PostgreSQL) configurada no detector")
	}

	if err != nil {
		return nil, fmt.Errorf("falha ao consultar metadados para staleness: %w", err)
	}

	// Normaliza caminhos do banco para buscas rápidas
	lookupMap := make(map[string]store.DocumentMeta, len(metaMap)*3)
	for rawPath, meta := range metaMap {
		clean := filepath.Clean(rawPath)
		lookupMap[clean] = meta
		lookupMap[filepath.ToSlash(clean)] = meta

		if filepath.IsAbs(clean) {
			if rel, relErr := filepath.Rel(d.rootDir, clean); relErr == nil {
				lookupMap[filepath.Clean(rel)] = meta
				lookupMap[filepath.ToSlash(rel)] = meta
			}
		}
	}

	seenDocs := make(map[string]bool)
	seenPaths := make(map[string]bool)
	var staleFiles []string
	var newFiles []string
	diskCount := 0

	// Varre o disco a partir de rootDir
	walkErr := filepath.WalkDir(d.rootDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		relPath, relErr := filepath.Rel(d.rootDir, path)
		if relErr != nil {
			relPath = path
		}

		if entry.IsDir() {
			if relPath != "." {
				base := entry.Name()
				for _, defEx := range config.DefaultExcludedDirs {
					if base == defEx {
						return filepath.SkipDir
					}
				}
				for _, ex := range d.cfg.Exclude {
					cleanEx := strings.TrimSuffix(ex, "/**")
					if relPath == cleanEx || strings.HasSuffix(relPath, "/"+cleanEx) {
						return filepath.SkipDir
					}
				}
			}
			return nil
		}

		if !d.cfg.ShouldIndex(relPath) {
			return nil
		}

		diskCount++

		info, infoErr := entry.Info()
		if infoErr != nil {
			return nil
		}
		mtime := info.ModTime().Unix()

		// Busca na tabela de metadados
		cleanRel := filepath.Clean(relPath)
		slashRel := filepath.ToSlash(cleanRel)
		cleanAbs := filepath.Clean(path)

		var docMeta store.DocumentMeta
		var found bool

		if m, ok := lookupMap[cleanRel]; ok {
			docMeta, found = m, true
		} else if m, ok := lookupMap[slashRel]; ok {
			docMeta, found = m, true
		} else if m, ok := lookupMap[cleanAbs]; ok {
			docMeta, found = m, true
		}

		displayPath := filepath.ToSlash(relPath)
		if !found {
			// Arquivo novo presente no disco mas não indexado (ADR-028 §1,
			// categoria "Novo"). Antes ele caía na lista de modificados e o
			// relatório não distinguia "novo" de "modificado".
			newFiles = append(newFiles, displayPath)
		} else {
			seenDocs[docMeta.ID] = true
			seenPaths[filepath.Clean(docMeta.Path)] = true
			seenPaths[filepath.ToSlash(docMeta.Path)] = true

			// ADR-057: a referência é o mtime persistido na indexação, não
			// updated_at. updated_at é o instante em que NÓS indexamos — um
			// arquivo editado antes de ser indexado sempre parece "no futuro",
			// e qualquer edição posterior ficava invisível. Linha legada
			// (mtime = 0) cai para updated_at em vez de virar falso positivo.
			persistedMTime := docMeta.MTime
			if persistedMTime <= 0 {
				persistedMTime = docMeta.UpdatedAt
			}

			// Tolerância de 1 segundo para precisão de filesystem (FAT32/NTFS vs Unix timestamp)
			if mtime > persistedMTime+1 {
				staleFiles = append(staleFiles, displayPath)
			}
		}

		return nil
	})

	if walkErr != nil {
		return nil, fmt.Errorf("falha ao varrer diretório para staleness: %w", walkErr)
	}

	// Identifica arquivos deletados que ainda constam no índice
	var deletedFiles []string
	for _, meta := range metaMap {
		cleanP := filepath.Clean(meta.Path)
		slashP := filepath.ToSlash(meta.Path)

		if seenDocs[meta.ID] || seenPaths[cleanP] || seenPaths[slashP] {
			continue
		}

		// Checa se realmente não existe no disco
		fullP := meta.Path
		if !filepath.IsAbs(fullP) {
			fullP = filepath.Join(d.rootDir, fullP)
		}

		if _, statErr := os.Stat(fullP); os.IsNotExist(statErr) {
			relDeleted, relErr := filepath.Rel(d.rootDir, fullP)
			if relErr != nil {
				deletedFiles = append(deletedFiles, filepath.ToSlash(meta.Path))
			} else {
				deletedFiles = append(deletedFiles, filepath.ToSlash(relDeleted))
			}
		}
	}

	sort.Strings(staleFiles)
	sort.Strings(newFiles)
	sort.Strings(deletedFiles)

	return &StalenessReport{
		IsStale:           len(staleFiles) > 0 || len(newFiles) > 0 || len(deletedFiles) > 0,
		StaleFilesCount:   len(staleFiles),
		NewFilesCount:     len(newFiles),
		DeletedFilesCount: len(deletedFiles),
		IndexedCount:      len(metaMap),
		DiskCount:         diskCount,
		StaleFiles:        staleFiles,
		NewFiles:          newFiles,
		DeletedFiles:      deletedFiles,
		LastCheck:         time.Now(),
	}, nil
}

// FormatMarkdownBanner gera o banner amigável em Markdown para ferramentas MCP.
//
// A linha é a da ADR-028 §3 (Accepted): o texto é o contrato que o agente
// reconhece, e N é o total pendente (modificados + novos + deletados).
func FormatMarkdownBanner(report *StalenessReport) string {
	if report == nil || !report.IsStale {
		return ""
	}
	return fmt.Sprintf("> ⚠️ **%s**: O índice de memória está desatualizado em relação aos arquivos no disco (%d arquivo(s) pendente(s): %d modificado(s), %d novo(s), %d deletado(s)). As respostas abaixo podem refletir um estado anterior. %s.\n",
		StalenessWarningTitle,
		report.StaleFilesCount+report.NewFilesCount+report.DeletedFilesCount,
		report.StaleFilesCount,
		report.NewFilesCount,
		report.DeletedFilesCount,
		StalenessRecommendation)
}

// FormatCLIBanner gera a linha de aviso para comandos CLI.
//
// A ADR-028 especifica apenas o banner Markdown (§3); a linha da CLI não tem
// norma. Ela carrega a mesma recomendação e as mesmas três contagens, numa
// linha só que cabe no topo do stdout com resultados.
func FormatCLIBanner(report *StalenessReport) string {
	if report == nil || !report.IsStale {
		return ""
	}
	return fmt.Sprintf("⚠️ [Aviso] O índice local está desatualizado (%d modificado(s), %d novo(s), %d removido(s)). %s.\n",
		report.StaleFilesCount,
		report.NewFilesCount,
		report.DeletedFilesCount,
		StalenessRecommendation)
}
