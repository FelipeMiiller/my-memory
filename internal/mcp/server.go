package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/FelipeMiiller/my-memory/internal/staleness"
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
	name              string
	version           string
	reader            *MessageReader
	writer            *MessageWriter
	logger            *log.Logger
	handlers          map[string]HandlerFunc
	tools             []Tool
	toolHandlers      map[string]ToolHandlerFunc
	stalenessDetector *staleness.Detector
	mu                sync.RWMutex
	initialized       bool
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
	s.RegisterHandler("tools/list", s.handleToolsList)
	s.RegisterHandler("tools/call", s.handleToolsCall)

	s.RegisterTool(ToolMemorySearch, NewMemorySearchHandler(nil))
	s.RegisterTool(ToolMemoryGetNeighbors, NewMemoryNeighborsHandler(nil))
	s.RegisterTool(ToolMemoryExportCanvas, NewMemoryExportCanvasHandler(nil))
	s.RegisterTool(ToolMemoryGetHubs, NewMemoryGetHubsHandler(nil))
	s.RegisterTool(ToolMemoryGetInsights, NewMemoryGetInsightsHandler(nil))
	s.RegisterTool(ToolMemoryDoctor, NewMemoryDoctorHandler(nil, nil))
	s.RegisterTool(ToolMemoryWriteNote, NewMemoryWriteNoteHandler(nil, "."))
	s.RegisterTool(ToolMemoryAppendSection, NewMemoryAppendSectionHandler(nil, "."))
	s.RegisterTool(ToolMemoryCompileNote, NewMemoryCompileNoteHandler(nil, nil, "."))
	s.RegisterTool(ToolMemoryVisualizeGraph, NewMemoryVisualizeGraphHandler(nil, "."))
	s.RegisterTool(ToolMemoryGetClusters, NewMemoryGetClustersHandler(nil))
	s.RegisterTool(ToolMemoryGetImpact, NewMemoryGetImpactHandler(nil))
	s.RegisterTool(ToolMemoryInspectNode, NewMemoryInspectNodeHandler(nil))
	s.RegisterTool(ToolMemoryFindPath, NewMemoryFindPathHandler(nil))
	s.RegisterTool(ToolMemoryPackContext, NewMemoryPackContextHandler(nil))
	s.RegisterTool(ToolMemoryOpenNode, NewMemoryOpenNodeHandler(nil, "", "", nil))
	s.RegisterTool(ToolMemoryGetDrift, NewMemoryGetDriftHandler(nil))
	s.RegisterTool(ToolMemoryCodeSearch, NewMemoryCodeSearchHandler(nil))
	s.RegisterTool(ToolMemoryCodeNeighbors, NewMemoryCodeNeighborsHandler(nil))

	return s
}

// SetStalenessDetector configura o detector de integridade e staleness do vault
func (s *Server) SetStalenessDetector(d *staleness.Detector) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stalenessDetector = d
}

// StalenessDetector retorna o detector configurado (se houver)
func (s *Server) StalenessDetector() *staleness.Detector {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stalenessDetector
}

// RegisterHandler registra um método JSON-RPC customizado
func (s *Server) RegisterHandler(method string, h HandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[method] = h
}

// Run executa o loop de leitura e despacho de mensagens até EOF ou cancelamento do contexto
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

// HandleRequest processa uma requisição JSON-RPC e retorna a resposta MCP correspondente
func (s *Server) HandleRequest(ctx context.Context, req *Request) *Response {
	s.mu.RLock()
	handler, exists := s.handlers[req.Method]
	s.mu.RUnlock()

	if !exists {
		if req.IsNotification() {
			return nil
		}
		return NewErrorResponse(req.ID, CodeMethodNotFound, fmt.Sprintf("Método '%s' não encontrado", req.Method), nil)
	}

	result, err := handler(ctx, req.Params)
	if req.IsNotification() {
		return nil
	}

	if err != nil {
		if mcpErr, ok := err.(*Error); ok {
			return NewErrorResponse(req.ID, mcpErr.Code, mcpErr.Message, mcpErr.Data)
		}
		return NewErrorResponse(req.ID, CodeInternalError, err.Error(), nil)
	}

	return NewSuccessResponse(req.ID, result)
}

// Name retorna o nome configurado do servidor MCP
func (s *Server) Name() string {
	return s.name
}

// Version retorna a versão do servidor MCP
func (s *Server) Version() string {
	return s.version
}

func (s *Server) dispatch(ctx context.Context, req *Request) {
	resp := s.HandleRequest(ctx, req)
	if resp != nil {
		_ = s.writer.WriteResponse(resp)
	}
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
	s.logger.Println("Cliente confirmou inicialização")
	return nil, nil
}

func (s *Server) handlePing(ctx context.Context, params json.RawMessage) (any, error) {
	return map[string]any{}, nil
}
