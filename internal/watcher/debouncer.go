package watcher

import (
	"sync"
	"time"
)

// Debouncer agrupa eventos sucessivos no mesmo arquivo dentro de uma janela de tempo
type Debouncer struct {
	duration time.Duration
	callback func(FileEvent)
	mu       sync.Mutex
	timers   map[string]*time.Timer
	pending  map[string]FileEvent
	stopped  bool
}

// NewDebouncer cria um novo debouncer para eventos de arquivo
func NewDebouncer(duration time.Duration, callback func(FileEvent)) *Debouncer {
	if duration <= 0 {
		duration = 300 * time.Millisecond
	}
	return &Debouncer{
		duration: duration,
		callback: callback,
		timers:   make(map[string]*time.Timer),
		pending:  make(map[string]FileEvent),
	}
}

// Add insere ou renova um evento para o caminho especificado
func (d *Debouncer) Add(event FileEvent) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.stopped {
		return
	}

	key := event.Path

	// Se já existe um evento pendente e recebemos um delete, sobrescreve para delete imediato
	if timer, exists := d.timers[key]; exists {
		timer.Stop()
		delete(d.timers, key)
	}

	// Mantém o tipo mais relevante se um arquivo foi criado e logo em seguida modificado
	prev, exists := d.pending[key]
	if exists && prev.Type == EventCreate && event.Type == EventModify {
		event.Type = EventCreate
	}

	d.pending[key] = event

	// Agenda a emissão do evento consolidado após expirar o debounce
	d.timers[key] = time.AfterFunc(d.duration, func() {
		d.mu.Lock()
		ev, ok := d.pending[key]
		if ok {
			delete(d.pending, key)
			delete(d.timers, key)
		}
		isStopped := d.stopped
		d.mu.Unlock()

		if ok && !isStopped && d.callback != nil {
			d.callback(ev)
		}
	})
}

// Flush dispara imediatamente todos os eventos pendentes e cancela os timers
func (d *Debouncer) Flush() {
	d.mu.Lock()
	defer d.mu.Unlock()

	for key, timer := range d.timers {
		timer.Stop()
		delete(d.timers, key)
	}

	for key, ev := range d.pending {
		delete(d.pending, key)
		if d.callback != nil {
			d.callback(ev)
		}
	}
}

// Stop cancela todos os timers pendentes e encerra o debouncer
func (d *Debouncer) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.stopped = true
	for key, timer := range d.timers {
		timer.Stop()
		delete(d.timers, key)
	}
	d.pending = make(map[string]FileEvent)
}
