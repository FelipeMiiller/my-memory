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

// StalenessReport agrega as métricas de sincronização entre o disco e a base indexada
type StalenessReport struct {
	IsStale           bool      `json:"is_stale"`
	StaleFilesCount   int       `json:"stale_files_count"`
	DeletedFilesCount int       `json:"deleted_files_count"`
	IndexedCount      int       `json:"indexed_count"`
	DiskCount         int       `json:"disk_count"`
	StaleFiles        []string  `json:"stale_files,omitempty"`
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
			// Arquivo novo presente no disco mas não indexado
			staleFiles = append(staleFiles, displayPath)
		} else {
			seenDocs[docMeta.ID] = true
			seenPaths[filepath.Clean(docMeta.Path)] = true
			seenPaths[filepath.ToSlash(docMeta.Path)] = true

			// Tolerância de 1 segundo para precisão de filesystem (FAT32/NTFS vs Unix timestamp)
			if mtime > docMeta.UpdatedAt+1 {
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
	sort.Strings(deletedFiles)

	return &StalenessReport{
		IsStale:           len(staleFiles) > 0 || len(deletedFiles) > 0,
		StaleFilesCount:   len(staleFiles),
		DeletedFilesCount: len(deletedFiles),
		IndexedCount:      len(metaMap),
		DiskCount:         diskCount,
		StaleFiles:        staleFiles,
		DeletedFiles:      deletedFiles,
		LastCheck:         time.Now(),
	}, nil
}

// FormatMarkdownBanner gera o banner amigável em Markdown para ferramentas MCP
func FormatMarkdownBanner(report *StalenessReport) string {
	if report == nil || !report.IsStale {
		return ""
	}
	return fmt.Sprintf("> ⚠️ **AVISO: Memória Desatualizada (Stale Data)**\n"+
		"> O índice local está defasado em relação aos arquivos no disco (%d modificado(s)/novo(s), %d removido(s)).\n"+
		"> Execute `mem index` no terminal para sincronizar o grafo e embeddings.\n\n",
		report.StaleFilesCount, report.DeletedFilesCount)
}

// FormatCLIBanner gera a linha de aviso para comandos CLI
func FormatCLIBanner(report *StalenessReport) string {
	if report == nil || !report.IsStale {
		return ""
	}
	return fmt.Sprintf("⚠️  [Aviso] O índice local está desatualizado (%d alterados/novos, %d removidos). Execute 'mem index'.\n",
		report.StaleFilesCount, report.DeletedFilesCount)
}
