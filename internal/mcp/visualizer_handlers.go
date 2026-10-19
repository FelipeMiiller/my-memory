package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/graphview"
)

// GraphViewFunc define a assinatura para construção do grafo visual
type GraphViewFunc func(ctx context.Context, repo, rootNode string, maxDepth int) (*graphview.GraphView, error)

// VisualizeGraphResult é o structuredContent de memory_visualize_graph.
//
// structuredContent: {url?: string, path: string, total_nodes: number, total_edges: number, hub_count: number, density: number}
//
// O campo "url" existe apenas quando o transporte HTTP está ativo: sem ele,
// a tool devolve só o caminho, sem erro (ADR-059 D-5).
type VisualizeGraphResult struct {
	URL        string  `json:"url,omitempty"`
	Path       string  `json:"path"`
	TotalNodes int     `json:"total_nodes"`
	TotalEdges int     `json:"total_edges"`
	HubCount   int     `json:"hub_count"`
	Density    float64 `json:"density"`
}

// NewMemoryVisualizeGraphHandler cria o handler MCP para a ferramenta memory_visualize_graph
//
// baseURL é o provedor do endereço base do transporte HTTP ("http://host:port").
// Quando devolve string vazia — modo stdio, sem transporte HTTP — a ferramenta
// devolve apenas o caminho do arquivo, sem erro (ADR-059 D-5 e nota 4).
func NewMemoryVisualizeGraphHandler(gvFn GraphViewFunc, defaultRoot string, baseURL func() string) ToolHandlerFunc {
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

		// Nome do artefato. Um output_path explícito define o nome do arquivo;
		// a escrita sempre acontece dentro da raiz servida, que é a única área
		// publicada pelo transporte HTTP.
		fileName := "graph.html"
		switch {
		case outputPath != "":
			fileName = artifactFileName(outputPath)
		case rootNode != "":
			safe := strings.ReplaceAll(rootNode, "/", "_")
			safe = strings.ReplaceAll(safe, "\\", "_")
			safe = strings.TrimSuffix(safe, ".md")
			fileName = fmt.Sprintf("graph_%s.html", safe)
		}

		// O diretório é criado aqui porque graphview.ExportHTML usa os.WriteFile
		// sem os.MkdirAll, e a falha precisa virar mensagem, não panic
		// (ADR-059 D-6).
		graphDir := GraphArtifactsDir(defaultRoot)
		if err := os.MkdirAll(graphDir, 0o755); err != nil {
			return nil, NewError(CodeInternalError, "Falha ao preparar o diretório de artefatos do grafo", err.Error())
		}
		outputPath = filepath.Join(graphDir, fileName)

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

		// A URL usa o host e a porta que o transporte realmente vinculou — nunca
		// um valor fixo (ADR-059 D-5).
		servedURL := ""
		if baseURL != nil {
			if b := strings.TrimRight(baseURL(), "/"); b != "" {
				servedURL = b + "/graph/" + url.PathEscape(fileName)
			}
		}

		var sb strings.Builder
		sb.WriteString("### 🌐 Visualização Interativa do Grafo Gerada com Sucesso\n\n")
		sb.WriteString(fmt.Sprintf("- **Arquivo HTML**: `%s`\n", absPath))
		if servedURL != "" {
			sb.WriteString(fmt.Sprintf("- **URL Servida**: %s\n", servedURL))
		}
		sb.WriteString(fmt.Sprintf("- **Total de Nós**: %d\n", gv.Stats.TotalNodes))
		sb.WriteString(fmt.Sprintf("- **Total de Arestas**: %d\n", gv.Stats.TotalEdges))
		sb.WriteString(fmt.Sprintf("- **Nós Centrais (Hubs)**: %d\n", gv.Stats.HubCount))
		sb.WriteString(fmt.Sprintf("- **Densidade**: %.4f\n", gv.Stats.Density))
		if rootNode != "" {
			sb.WriteString(fmt.Sprintf("- **Nó Raiz Focado**: `%s` (Profundidade: %d)\n", rootNode, maxDepth))
		}
		sb.WriteString("\n*Abra o arquivo acima em qualquer navegador web para explorar a topologia de conexões, com simulação de física de forças, busca em tempo real, filtros por tipo e painel de detalhes.*")

		structured := VisualizeGraphResult{
			Path:       absPath,
			TotalNodes: gv.Stats.TotalNodes,
			TotalEdges: gv.Stats.TotalEdges,
			HubCount:   gv.Stats.HubCount,
			Density:    gv.Stats.Density,
		}
		if servedURL != "" {
			structured.URL = servedURL
		}

		return NewStructuredResult(sb.String(), structured), nil
	}
}

// artifactFileName extrai o nome do arquivo de um output_path pedido,
// descartando diretórios: o artefato é sempre gravado dentro do diretório de
// artefatos servidos.
func artifactFileName(outputPath string) string {
	name := filepath.Base(filepath.FromSlash(outputPath))
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "/", "_")
	if name == "" || name == "." || name == ".." {
		return "graph.html"
	}
	return name
}

// SetGraphViewHandler registra ou atualiza o handler de visualização de grafo no servidor MCP
func (s *Server) SetGraphViewHandler(gvFn GraphViewFunc, defaultRoot string) {
	s.RegisterTool(ToolMemoryVisualizeGraph, NewMemoryVisualizeGraphHandler(gvFn, defaultRoot, s.BaseURL))
}
