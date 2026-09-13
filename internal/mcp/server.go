package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"
)

// ProtocolVersion versÃ£o do protocolo MCP suportada
const ProtocolVersion = "2024-11-05"

// HandlerFunc processa uma requisiÃ§Ã£o JSON-RPC genÃ©rica
type HandlerFunc func(ctx context.Context, params json.RawMessage) (any, error)

// Tool representa uma ferramenta MCP declarada ao cliente
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"inputSchema"`
}

// ToolHandlerFunc executa uma ferramenta especÃ­fica
type ToolHandlerFunc func(ctx context.Context, args json.RawMessage) (any, error)

// Implementation descreve o nome e versÃ£o do servidor ou cliente
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ServerCapabilities declara o que o servidor suporta
type ServerCapabilities struct {
	Tools *ToolsCapability `json:"tools,omitempty"`
}

type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// InitializeParams parÃ¢metros recebidos no handshake
type InitializeParams struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities"`
	ClientInfo      Implementation `json:"clientInfo"`
}

// InitializeResult resultado enviado de volta no handshake
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      Implementation     `json:"serverInfo"`
}

// Server gerencia o loop de mensagens e ciclo de vida do servidor MCP
type Server struct {
	name         string
	version      string
	reader       *MessageReader
	writer       *MessageWriter
	logger       *log.Logger
	handlers     map[string]HandlerFunc
	tools        []Tool
	toolHandlers map[string]ToolHandlerFunc
	mu           sync.RWMutex
	initialized  bool
}

// NewServer cria um servidor MCP configurado
func NewServer(name, version string, in io.Reader, out io.Writer, errLog io.Writer) *Server {
	if errLog == nil {
		errLog = io.Discard
	}

	s := &Server{
		name:         name,
		version:      version,
		reader:       NewMessageReader(in),
		writer:       NewMessageWriter(out),
		logger:       log.New(errLog, "[mcp] ", log.LstdFlags),
		handlers:     make(map[string]HandlerFunc),
		tools:        make([]Tool, 0),
		toolHandlers: make(map[string]ToolHandlerFunc),
	}

	s.RegisterHandler("initialize", s.handleInitialize)
	s.RegisterHandler("notifications/initialized", s.handleInitialized)
	s.RegisterHandler("ping", s.handlePing)

	return s
}

// RegisterHandler registra um mÃ©todo JSON-RPC customizado
func (s *Server) RegisterHandler(method string, h HandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[method] = h
}

// Run executa o loop de leitura e despacho de mensagens atÃ© EOF ou cancelamento do contexto
func (s *Server) Run(ctx context.Context) error {
	s.logger.Printf("Servidor MCP '%s' v%s iniciado", s.name, s.version)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, err := s.reader.ReadRequest()
		if err != nil {
			if err == io.EOF {
				s.logger.Println("Cliente desconectou (EOF)")
				return nil
			}

			if rpcErr, ok := err.(*Error); ok {
				_ = s.writer.WriteResponse(NewErrorResponse(nil, rpcErr.Code, rpcErr.Message, rpcErr.Data))
				continue
			}

			s.logger.Printf("Erro de leitura: %v", err)
			return err
		}

		s.dispatch(ctx, req)
	}
}

func (s *Server) dispatch(ctx context.Context, req *Request) {
	s.mu.RLock()
	handler, exists := s.handlers[req.Method]
	s.mu.RUnlock()

	if !exists {
		if !req.IsNotification() {
			resp := NewErrorResponse(req.ID, CodeMethodNotFound, fmt.Sprintf("MÃ©todo '%s' nÃ£o encontrado", req.Method), nil)
			_ = s.writer.WriteResponse(resp)
		}
		return
	}

	result, err := handler(ctx, req.Params)
	if req.IsNotification() {
		return
	}

	if err != nil {
		var resp *Response
		if rpcErr, ok := err.(*Error); ok {
			resp = NewErrorResponse(req.ID, rpcErr.Code, rpcErr.Message, rpcErr.Data)
		} else {
			resp = NewErrorResponse(req.ID, CodeInternalError, err.Error(), nil)
		}
		_ = s.writer.WriteResponse(resp)
		return
	}

	resp := NewSuccessResponse(req.ID, result)
	_ = s.writer.WriteResponse(resp)
}

func (s *Server) handleInitialize(ctx context.Context, params json.RawMessage) (any, error) {
	s.mu.Lock()
	s.initialized = true
	s.mu.Unlock()

	return InitializeResult{
		ProtocolVersion: ProtocolVersion,
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{ListChanged: false},
		},
		ServerInfo: Implementation{
			Name:    s.name,
			Version: s.version,
		},
	}, nil
}

func (s *Server) handleInitialized(ctx context.Context, params json.RawMessage) (any, error) {
	s.logger.Println("Cliente confirmou inicializaÃ§Ã£o")
	return nil, nil
}

func (s *Server) handlePing(ctx context.Context, params json.RawMessage) (any, error) {
	return map[string]any{}, nil
}
