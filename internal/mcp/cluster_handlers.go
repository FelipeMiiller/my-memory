package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/graph"
)

// ClustersFunc define a assinatura para extração de clusters e comunidades
type ClustersFunc func(ctx context.Context, repo string, minSize int) (graph.CommunityResult, error)

// NewMemoryGetClustersHandler cria o executor para a ferramenta MCP memory_get_clusters
func NewMemoryGetClustersHandler(clustersFn ClustersFunc) ToolHandlerFunc {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		var rawMap map[string]json.RawMessage
		if len(args) > 0 {
			if err := json.Unmarshal(args, &rawMap); err != nil {
				return nil, NewError(CodeInvalidParams, "Formato JSON de argumentos inválido", err.Error())
			}
		}

		minSize := 2
		if rawMin, ok := rawMap["min_size"]; ok {
			var m int
			if err := json.Unmarshal(rawMin, &m); err == nil && m > 0 {
				minSize = m
			}
		}

		var repo string
		if rawRepo, ok := rawMap["repository"]; ok {
			_ = json.Unmarshal(rawRepo, &repo)
		}

		var result graph.CommunityResult
		var err error
		if clustersFn != nil {
			result, err = clustersFn(ctx, repo, minSize)
			if err != nil {
				return nil, NewError(CodeInternalError, "Falha ao calcular clusters da memória", err.Error())
			}
		} else {
			result = graph.CommunityResult{
				Communities: []graph.Community{},
				Modularity:  0.0,
			}
		}

		var filtered []graph.Community
		for _, c := range result.Communities {
			if c.Size >= minSize {
				filtered = append(filtered, c)
			}
		}
		if filtered == nil {
			filtered = []graph.Community{}
		}

		var sb strings.Builder
		sb.WriteString("### 🧩 Clusters e Comunidades Temáticas no Grafo\n\n")
		sb.WriteString(fmt.Sprintf("- **Modularidade Newman-Girvan (Q)**: `%.4f`\n", result.Modularity))
		sb.WriteString(fmt.Sprintf("- **Total de Nós**: %d\n", result.TotalNodes))
		sb.WriteString(fmt.Sprintf("- **Total de Arestas**: %d\n", result.TotalEdges))
		sb.WriteString(fmt.Sprintf("- **Clusters Identificados (tamanho >= %d)**: %d\n\n", minSize, len(filtered)))

		if len(filtered) == 0 {
			sb.WriteString("*Nenhum cluster com o tamanho mínimo solicitado foi encontrado.*\n")
		} else {
			sb.WriteString("| ID | Nó Líder / Hub | Tamanho | Tipo Dominante | Membros Notáveis |\n")
			sb.WriteString("|---|---|---|---|---|\n")
			for _, c := range filtered {
				membersStr := strings.Join(c.Members, ", ")
				if len(membersStr) > 60 {
					membersStr = membersStr[:57] + "..."
				}
				domType := c.DominantType
				if domType == "" {
					domType = "-"
				}
				sb.WriteString(fmt.Sprintf("| %d | `%s` | %d | %s | %s |\n", c.ID, c.LeadNode, c.Size, domType, membersStr))
			}
		}

		return NewTextResult(sb.String()), nil
	}
}

// SetClustersHandler registra ou atualiza o handler de clusters no servidor MCP
func (s *Server) SetClustersHandler(fn ClustersFunc) {
	s.RegisterTool(ToolMemoryGetClusters, NewMemoryGetClustersHandler(fn))
}
