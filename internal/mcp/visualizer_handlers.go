package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/graphview"
)

// GraphViewFunc define a assinatura para construção do grafo visual
type GraphViewFunc func(ctx context.Context, repo, rootNode string, maxDepth int) (*graphview.GraphView, error)

// NewMemoryVisualizeGraphHandler cria o handler MCP para a ferramenta memory_visualize_graph
func NewMemoryVisualizeGraphHandler(gvFn GraphViewFunc, defaultRoot string) ToolHandlerFunc {
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

		maxDepth := 2
		if rawDepth, ok := rawMap["max_depth"]; ok {
			var d int
			if err := json.Unmarshal(rawDepth, &d); err == nil && d > 0 {
				maxDepth = d
			}
		}

		var outputPath string
		if rawOut, ok := rawMap["output_path"]; ok {
			_ = json.Unmarshal(rawOut, &outputPath)
		}

		var repo string
		if rawRepo, ok := rawMap["repository"]; ok {
			_ = json.Unmarshal(rawRepo, &repo)
		}

		if outputPath == "" {
			if rootNode != "" {
				safe := strings.ReplaceAll(rootNode, "/", "_")
				safe = strings.ReplaceAll(safe, "\\", "_")
				safe = strings.TrimSuffix(safe, ".md")
				outputPath = fmt.Sprintf("graph_%s.html", safe)
			} else {
				outputPath = "graph.html"
			}
		}

		if defaultRoot != "" && !filepath.IsAbs(outputPath) {
			outputPath = filepath.Join(defaultRoot, outputPath)
		}

		var gv *graphview.GraphView
		var err error

		if gvFn != nil {
			gv, err = gvFn(ctx, repo, rootNode, maxDepth)
			if err != nil {
				return nil, NewError(CodeInternalError, "Falha ao construir grafo da memória", err.Error())
			}
		} else {
			// Fallback vazio seguro
			gv = graphview.BuildGraphView(nil, nil, rootNode, maxDepth, repo)
		}

		if err := graphview.ExportHTML(gv, outputPath); err != nil {
			return nil, NewError(CodeInternalError, "Falha ao exportar página HTML do grafo", err.Error())
		}

		absPath, _ := filepath.Abs(outputPath)
		var sb strings.Builder
		sb.WriteString("### 🌐 Visualização Interativa do Grafo Gerada com Sucesso\n\n")
		sb.WriteString(fmt.Sprintf("- **Arquivo HTML**: `%s`\n", absPath))
		sb.WriteString(fmt.Sprintf("- **Total de Nós**: %d\n", gv.Stats.TotalNodes))
		sb.WriteString(fmt.Sprintf("- **Total de Arestas**: %d\n", gv.Stats.TotalEdges))
		sb.WriteString(fmt.Sprintf("- **Nós Centrais (Hubs)**: %d\n", gv.Stats.HubCount))
		sb.WriteString(fmt.Sprintf("- **Densidade**: %.4f\n", gv.Stats.Density))
		if rootNode != "" {
			sb.WriteString(fmt.Sprintf("- **Nó Raiz Focado**: `%s` (Profundidade: %d)\n", rootNode, maxDepth))
		}
		sb.WriteString("\n*Abra o arquivo acima em qualquer navegador web para explorar a topologia de conexões, com simulação de física de forças, busca em tempo real, filtros por tipo e painel de detalhes.*")

		return NewTextResult(sb.String()), nil
	}
}

// SetGraphViewHandler registra ou atualiza o handler de visualização de grafo no servidor MCP
func (s *Server) SetGraphViewHandler(gvFn GraphViewFunc, defaultRoot string) {
	s.RegisterTool(ToolMemoryVisualizeGraph, NewMemoryVisualizeGraphHandler(gvFn, defaultRoot))
}
