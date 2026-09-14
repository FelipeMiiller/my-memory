package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/graph"
)

// PackFunc define a assinatura da função de empacotamento de subgrafo
type PackFunc func(ctx context.Context, repo, rootQuery string, opts graph.PackOptions) (*graph.PackResult, error)

// NewMemoryPackContextHandler cria o executor para a ferramenta MCP memory_pack_context
func NewMemoryPackContextHandler(packFn PackFunc) ToolHandlerFunc {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		var rawMap map[string]json.RawMessage
		if len(args) > 0 {
			if err := json.Unmarshal(args, &rawMap); err != nil {
				return nil, NewError(CodeInvalidParams, "Formato JSON de argumentos inválido", err.Error())
			}
		}

		var rootNode string
		if rawRoot, ok := rawMap["root_node"]; ok {
			_ = json.Unmarshal(rawRoot, &rootNode)
		}
		rootNode = strings.TrimSpace(rootNode)
		if rootNode == "" {
			return nil, NewError(CodeInvalidParams, "Parâmetro 'root_node' é obrigatório", nil)
		}

		maxDepth := 2
		if rawDepth, ok := rawMap["max_depth"]; ok {
			var d int
			if err := json.Unmarshal(rawDepth, &d); err == nil && d > 0 {
				maxDepth = d
			}
		}

		maxTokens := 4000
		if rawTokens, ok := rawMap["max_tokens"]; ok {
			var t int
			if err := json.Unmarshal(rawTokens, &t); err == nil && t > 0 {
				maxTokens = t
			}
		}

		direction := "both"
		if rawDir, ok := rawMap["direction"]; ok {
			var dir string
			if err := json.Unmarshal(rawDir, &dir); err == nil && dir != "" {
				direction = strings.ToLower(strings.TrimSpace(dir))
			}
		}

		var repo string
		if rawRepo, ok := rawMap["repository"]; ok {
			_ = json.Unmarshal(rawRepo, &repo)
		}

		if packFn == nil {
			return nil, NewError(CodeInternalError, "Handler de empacotamento de subgrafo não configurado no servidor", nil)
		}

		opts := graph.PackOptions{
			MaxDepth:               maxDepth,
			MaxTokens:              maxTokens,
			Direction:              direction,
			IncludeFringeAbstracts: true,
		}

		res, err := packFn(ctx, repo, rootNode, opts)
		if err != nil {
			return nil, NewError(CodeInternalError, fmt.Sprintf("Erro ao empacotar contexto para nó '%s': %v", rootNode, err), nil)
		}

		return NewTextResult(res.Markdown), nil
	}
}

// SetPackHandler registra ou atualiza o handler de empacotamento de contexto no servidor MCP
func (s *Server) SetPackHandler(fn PackFunc) {
	s.RegisterTool(ToolMemoryPackContext, NewMemoryPackContextHandler(fn))
}
