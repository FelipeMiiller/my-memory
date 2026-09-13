package watcher

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/config"
)

// EventType representa a natureza da alteração no arquivo
type EventType int

const (
	EventCreate EventType = iota
	EventModify
	EventDelete
)

func (e EventType) String() string {
	switch e {
	case EventCreate:
		return "CREATE"
	case EventModify:
		return "MODIFY"
	case EventDelete:
		return "DELETE"
	default:
		return "UNKNOWN"
	}
}

// FileEvent encapsula os dados de uma alteração detectada no vault
type FileEvent struct {
	Type    EventType
	Path    string    // Caminho completo no disco
	RelPath string    // Caminho relativo à raiz do vault
	ModTime time.Time // Horário da última modificação
	Size    int64     // Tamanho do arquivo em bytes
}

type fileSnapshot struct {
	ModTime time.Time
	Size    int64
}

// Watcher monitora continuamente alterações em arquivos dentro de um vault
type Watcher struct {
	rootDir   string
	cfg       *config.Config
	interval  time.Duration
	debounce  time.Duration
	debouncer *Debouncer
	events    chan FileEvent
	stopCh    chan struct{}
	doneCh    chan struct{}
	mu        sync.RWMutex
	state     map[string]fileSnapshot
	started   bool
}

// NewWatcher instancia um novo monitor para o diretório alvo
func NewWatcher(rootDir string, cfg *config.Config, interval, debounce time.Duration) *Watcher {
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	if debounce <= 0 {
		debounce = 300 * time.Millisecond
	}
	if cfg == nil {
		def := config.DefaultConfig()
		cfg = &def
	}

	absRoot, err := filepath.Abs(rootDir)
	if err == nil {
		rootDir = absRoot
	}

	w := &Watcher{
		rootDir:  rootDir,
		cfg:      cfg,
		interval: interval,
		debounce: debounce,
		events:   make(chan FileEvent, 100),
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
		state:    make(map[string]fileSnapshot),
	}

	w.debouncer = NewDebouncer(debounce, func(ev FileEvent) {
		select {
		case w.events <- ev:
		default:
			// Canal cheio, descarta ou bufferiza
		}
	})

	return w
}

// Start inicia o loop de monitoramento em segundo plano e retorna o canal de eventos consolidados
func (w *Watcher) Start(ctx context.Context) <-chan FileEvent {
	w.mu.Lock()
	if w.started {
		w.mu.Unlock()
		return w.events
	}
	w.started = true
	w.mu.Unlock()

	// 1. Varredura inicial sem emitir eventos (apenas registra o estado de baseline)
	w.scan(true)

	// 2. Loop de polling periódico
	go func() {
		defer close(w.doneCh)
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				w.debouncer.Flush()
				return
			case <-w.stopCh:
				w.debouncer.Flush()
				return
			case <-ticker.C:
				w.scan(false)
			}
		}
	}()

	return w.events
}

// Stop finaliza graciosamente a execução do Watcher
func (w *Watcher) Stop() {
	w.mu.Lock()
	if !w.started {
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()

	select {
	case <-w.stopCh:
	default:
		close(w.stopCh)
	}

	<-w.doneCh
	w.debouncer.Stop()
	close(w.events)
}

// scan percorre o diretório comparando o estado atual dos arquivos com o snapshot
func (w *Watcher) scan(initial bool) {
	visited := make(map[string]bool)

	_ = filepath.WalkDir(w.rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		relPath, relErr := filepath.Rel(w.rootDir, path)
		if relErr != nil {
			relPath = path
		}

		if d.IsDir() {
			if relPath != "." {
				base := d.Name()
				// Ignora diretórios de sistema padrão
				for _, defEx := range config.DefaultExcludedDirs {
					if base == defEx {
						return filepath.SkipDir
					}
				}
				// Avalia excludes da configuração
				for _, ex := range w.cfg.Exclude {
					cleanEx := strings.TrimSuffix(ex, "/**")
					if relPath == cleanEx || strings.HasSuffix(relPath, "/"+cleanEx) {
						return filepath.SkipDir
					}
				}
			}
			return nil
		}

		// Checa se o arquivo atende às regras de ShouldIndex
		if !w.cfg.ShouldIndex(relPath) {
			return nil
		}

		visited[path] = true
		info, err := d.Info()
		if err != nil {
			return nil
		}

		w.mu.Lock()
		prev, exists := w.state[path]
		modTime := info.ModTime()
		size := info.Size()

		if !exists {
			w.state[path] = fileSnapshot{ModTime: modTime, Size: size}
			w.mu.Unlock()

			if !initial {
				w.debouncer.Add(FileEvent{
					Type:    EventCreate,
					Path:    path,
					RelPath: relPath,
					ModTime: modTime,
					Size:    size,
				})
			}
		} else {
			// Verifica se houve modificação real em ModTime ou Size
			if !prev.ModTime.Equal(modTime) || prev.Size != size {
				w.state[path] = fileSnapshot{ModTime: modTime, Size: size}
				w.mu.Unlock()

				if !initial {
					w.debouncer.Add(FileEvent{
						Type:    EventModify,
						Path:    path,
						RelPath: relPath,
						ModTime: modTime,
						Size:    size,
					})
				}
			} else {
				w.mu.Unlock()
			}
		}

		return nil
	})

	// Detecta exclusões comparando o que estava no estado mas não foi visitado
	w.mu.Lock()
	var deletedPaths []string
	for p := range w.state {
		if !visited[p] {
			deletedPaths = append(deletedPaths, p)
		}
	}

	for _, p := range deletedPaths {
		delete(w.state, p)
		relPath, _ := filepath.Rel(w.rootDir, p)

		if !initial {
			w.debouncer.Add(FileEvent{
				Type:    EventDelete,
				Path:    p,
				RelPath: relPath,
			})
		}
	}
	w.mu.Unlock()
}
