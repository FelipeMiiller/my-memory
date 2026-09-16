package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/graph"
)

// PathFunc define a assinatura para busca de rotas entre nós
type PathFunc func(ctx context.Context, repo, source, target string, opts graph.PathOptions) (*graph.PathResult, error)

// NewMemoryFindPathHandler cria o executor para a ferramenta MCP memory_find_path
func NewMemoryFindPathHandler(pathFn PathFunc) ToolHandlerFunc {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		var rawMap map[string]json.RawMessage
		if len(args) > 0 {
			if err := json.Unmarshal(args, &rawMap); err != nil {
				return nil, NewError(CodeInvalidParams, "Formato JSON de argumentos inválido", err.Error())
			}
		}

		var source string
		if rawSrc, ok := rawMap["source"]; ok {
			_ = json.Unmarshal(rawSrc, &source)
		}
		source = strings.TrimSpace(source)
		if source == "" {
			return nil, NewError(CodeInvalidParams, "Parâmetro 'source' é obrigatório", nil)
		}

		var target string
		if rawTgt, ok := rawMap["target"]; ok {
			_ = json.Unmarshal(rawTgt, &target)
		}
		target = strings.TrimSpace(target)
		if target == "" {
			return nil, NewError(CodeInvalidParams, "Parâmetro 'target' é obrigatório", nil)
		}

		maxDepth := 6
		if rawDepth, ok := rawMap["max_depth"]; ok {
			var d int
			if err := json.Unmarshal(rawDepth, &d); err == nil && d > 0 {
				maxDepth = d
			}
		}

		directed := true
		if rawDir, ok := rawMap["directed"]; ok {
			var dir bool
			if err := json.Unmarshal(rawDir, &dir); err == nil {
				directed = dir
			}
		}

		costMode := graph.CostModeEpistemic
		if rawMode, ok := rawMap["mode"]; ok {
			var m string
			if err := json.Unmarshal(rawMode, &m); err == nil && strings.ToLower(strings.TrimSpace(m)) == "hops" {
				costMode = graph.CostModeHops
			}
		}

		var repo string
		if rawRepo, ok := rawMap["repository"]; ok {
			_ = json.Unmarshal(rawRepo, &repo)
		}

		opts := graph.PathOptions{
			MaxDepth: maxDepth,
			Directed: directed,
			CostMode: costMode,
		}

		var res *graph.PathResult
		var err error
		if pathFn != nil {
			res, err = pathFn(ctx, repo, source, target, opts)
			if err != nil {
				return nil, NewError(CodeInternalError, "Falha ao calcular rota no grafo", err.Error())
			}
		} else {
			res = &graph.PathResult{
				Found:     false,
				Source:    source,
				Target:    target,
				Directed:  directed,
				CostMode:  costMode,
				Hops:      0,
				TotalCost: 0.0,
				Nodes:     []string{},
				Edges:     []graph.PathEdge{},
				Summary:   "Handler de busca de caminho não configurado",
			}
		}

		var sb strings.Builder
		sb.WriteString("### 🧭 Descoberta de Rotas no Grafo de Conhecimento\n\n")
		sb.WriteString(fmt.Sprintf("- **Origem**: `%s`\n", res.Source))
		sb.WriteString(fmt.Sprintf("- **Destino**: `%s`\n", res.Target))
		dirStr := "Direcionado (A -> B)"
		if !res.Directed {
			dirStr = "Bidirecional (A <-> B)"
		}
		sb.WriteString(fmt.Sprintf("- **Direcionamento**: %s\n", dirStr))
		sb.WriteString(fmt.Sprintf("- **Modo de Custo**: `%s`\n", res.CostMode))
		sb.WriteString(fmt.Sprintf("- **Limite de Profundidade**: %d saltos\n\n", maxDepth))

		if !res.Found {
			sb.WriteString(fmt.Sprintf("❌ **Nenhum caminho encontrado**: %s\n", res.Summary))
			return NewTextResult(sb.String()), nil
		}

		sb.WriteString(fmt.Sprintf("✅ **Caminho Encontrado**: %d salto(s) | Custo Total: `%.3f`\n\n", res.Hops, res.TotalCost))

		if res.Hops == 0 {
			sb.WriteString("*Nó de origem e destino são idênticos.*\n")
			return NewTextResult(sb.String()), nil
		}

		sb.WriteString("#### 🔗 Rota Conectada\n```text\n")
		sb.WriteString(fmt.Sprintf("[%s]\n", res.Nodes[0]))
		for i, edge := range res.Edges {
			arrow := "──>"
			dir := "forward"
			if edge.Direction == "reverse" {
				arrow = "<──"
				dir = "reverse"
			}
			sb.WriteString(fmt.Sprintf(" └──(%s [%s | %s] | custo: %.3f)%s \n[%s]\n",
				edge.Relation, edge.EpistemicStatus, dir, edge.Cost, arrow, res.Nodes[i+1]))
		}
		sb.WriteString("```\n\n")

		sb.WriteString("#### 📊 Segmentos Percorridos\n")
		sb.WriteString("| Salto | De | Para | Relação | Status Epistêmico | Direção | Custo |\n")
		sb.WriteString("|---|---|---|---|---|---|---|\n")
		for i, edge := range res.Edges {
			sb.WriteString(fmt.Sprintf("| %d | `%s` | `%s` | `%s` | `%s` | `%s` | `%.3f` |\n",
				i+1, edge.From, edge.To, edge.Relation, edge.EpistemicStatus, edge.Direction, edge.Cost))
		}

		return NewTextResult(sb.String()), nil
	}
}

// SetPathHandler registra ou atualiza o handler de caminho no servidor MCP
func (s *Server) SetPathHandler(fn PathFunc) {
	s.RegisterTool(ToolMemoryFindPath, NewMemoryFindPathHandler(fn))
}
