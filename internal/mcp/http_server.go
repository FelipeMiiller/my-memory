package mcp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

// HTTPServer implementa o transporte de rede HTTP e Server-Sent Events (SSE) para o protocolo MCP
type HTTPServer struct {
	server      *Server
	addr        string
	corsEnabled bool
	httpServer  *http.Server
	logger      *log.Logger
	startedAt   time.Time
	defaultRepo string

	mu       sync.RWMutex
	sessions map[string]chan []byte
}

// HTTPServerOptions define as configurações para o servidor HTTP MCP
type HTTPServerOptions struct {
	Addr        string
	CORSEnabled bool
	DefaultRepo string
	Logger      *log.Logger
}

// HealthResponse estrutura os dados retornados pelo endpoint GET /health
type HealthResponse struct {
	Status          string  `json:"status"`
	Name            string  `json:"name"`
	Version         string  `json:"version"`
	ProtocolVersion string  `json:"protocol_version"`
	UptimeSeconds   float64 `json:"uptime_seconds"`
	DefaultRepo     string  `json:"default_repo,omitempty"`
	ToolsCount      int     `json:"tools_count"`
	ActiveSessions  int     `json:"active_sessions"`
}

// NewHTTPServer cria e configura um servidor HTTP para o protocolo MCP
func NewHTTPServer(srv *Server, opts HTTPServerOptions) *HTTPServer {
	if opts.Addr == "" {
		opts.Addr = ":38400"
	}
	logger := opts.Logger
	if logger == nil {
		logger = log.New(io.Discard, "[mcp-http] ", log.LstdFlags)
	}

	h := &HTTPServer{
		server:      srv,
		addr:        opts.Addr,
		corsEnabled: opts.CORSEnabled,
		logger:      logger,
		startedAt:   time.Now(),
		defaultRepo: opts.DefaultRepo,
		sessions:    make(map[string]chan []byte),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/sse", h.handleSSE)
	mux.HandleFunc("/message", h.handleMessage)
	mux.HandleFunc("/mcp", h.handleDirectRPC)

	h.httpServer = &http.Server{
		Addr:         opts.Addr,
		Handler:      h.withCORS(mux),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 0, // Essencial para manter conexões SSE abertas sem timeout prematuro
		IdleTimeout:  120 * time.Second,
	}

	return h
}

func (h *HTTPServer) createSession() (string, chan []byte) {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	sessionID := hex.EncodeToString(b)

	ch := make(chan []byte, 64)

	h.mu.Lock()
	defer h.mu.Unlock()
	h.sessions[sessionID] = ch
	return sessionID, ch
}

func (h *HTTPServer) getSession(sessionID string) (chan []byte, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ch, exists := h.sessions[sessionID]
	return ch, exists
}

func (h *HTTPServer) removeSession(sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if ch, exists := h.sessions[sessionID]; exists {
		delete(h.sessions, sessionID)
		close(ch)
	}
}

func (h *HTTPServer) activeSessionsCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.sessions)
}

// handleSSE gerencia o ciclo de vida do stream de eventos Server-Sent Events (SSE)
func (h *HTTPServer) handleSSE(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido (esperado GET)", http.StatusMethodNotAllowed)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming de Server-Sent Events não suportado pelo cliente ou servidor", http.StatusInternalServerError)
		return
	}

	sessionID, ch := h.createSession()
	defer h.removeSession(sessionID)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Emite o evento inicial "endpoint" conforme exigido pela especificação oficial do MCP
	endpointURI := fmt.Sprintf("/message?sessionId=%s", sessionID)
	fmt.Fprintf(w, "event: endpoint\r\ndata: %s\r\n\r\n", endpointURI)
	flusher.Flush()

	h.logger.Printf("Nova sessão SSE estabelecida: %s", sessionID)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			h.logger.Printf("Sessão SSE encerrada pelo cliente: %s", sessionID)
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: message\r\ndata: %s\r\n\r\n", string(msg))
			flusher.Flush()
		}
	}
}

// handleMessage processa requisições JSON-RPC de clientes associados a uma sessão SSE ativa
func (h *HTTPServer) handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido (esperado POST)", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		http.Error(w, "Parâmetro 'sessionId' obrigatório na query string", http.StatusBadRequest)
		return
	}

	ch, exists := h.getSession(sessionID)
	if !exists {
		http.Error(w, fmt.Sprintf("Sessão '%s' não encontrada ou já encerrada", sessionID), http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 10<<20)) // 10 MB limite
	if err != nil {
		http.Error(w, fmt.Sprintf("Erro ao ler corpo da requisição: %v", err), http.StatusBadRequest)
		return
	}

	var req Request
	if err := json.Unmarshal(body, &req); err != nil {
		errResp := NewErrorResponse(nil, CodeParseError, "JSON malformado", err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	resp := h.server.HandleRequest(r.Context(), &req)

	if resp != nil {
		respBytes, err := json.Marshal(resp)
		if err == nil {
			select {
			case ch <- respBytes:
			default:
				h.logger.Printf("Buffer de canal SSE cheio para sessão %s", sessionID)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted) // 202 Accepted conforme especificação oficial MCP
			_, _ = w.Write(respBytes)
			return
		}
	}

	w.WriteHeader(http.StatusAccepted)
}

// handleHealth expõe métricas de prontidão, uptime e metadados operacionais
func (h *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	res := HealthResponse{
		Status:          "healthy",
		Name:            h.server.Name(),
		Version:         h.server.Version(),
		ProtocolVersion: ProtocolVersion,
		UptimeSeconds:   time.Since(h.startedAt).Seconds(),
		DefaultRepo:     h.defaultRepo,
		ToolsCount:      len(h.server.GetTools()),
		ActiveSessions:  h.activeSessionsCount(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

// handleDirectRPC provê um endpoint RPC direto e sem estado para clientes HTTP simples
func (h *HTTPServer) handleDirectRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido (esperado POST)", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 10<<20))
	if err != nil {
		http.Error(w, "Erro ao ler corpo", http.StatusBadRequest)
		return
	}

	var req Request
	if err := json.Unmarshal(body, &req); err != nil {
		errResp := NewErrorResponse(nil, CodeParseError, "JSON malformado", err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	resp := h.server.HandleRequest(r.Context(), &req)
	w.Header().Set("Content-Type", "application/json")
	if resp == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// withCORS adiciona cabeçalhos CORS e responde preflight OPTIONS
func (h *HTTPServer) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.corsEnabled {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// Serve inicia o servidor aceitando conexões a partir de um net.Listener já vinculado
func (h *HTTPServer) Serve(l net.Listener) error {
	h.logger.Printf("Servidor MCP escutando em %s", l.Addr().String())
	return h.httpServer.Serve(l)
}

// ListenAndServe inicia a escuta no endereço configurado
func (h *HTTPServer) ListenAndServe() error {
	h.logger.Printf("Servidor MCP iniciando em %s", h.addr)
	return h.httpServer.ListenAndServe()
}

// Shutdown encerra graciosamente o servidor HTTP e fecha as sessões ativas
func (h *HTTPServer) Shutdown(ctx context.Context) error {
	h.mu.Lock()
	for id, ch := range h.sessions {
		delete(h.sessions, id)
		close(ch)
	}
	h.mu.Unlock()

	return h.httpServer.Shutdown(ctx)
}
