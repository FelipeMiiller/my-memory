package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/graph"
)

// ImpactFunc define a assinatura para cálculo de análise de impacto (blast radius)
type ImpactFunc func(ctx context.Context, repo string, nodeID string, maxDepth int) (*graph.ImpactResult, error)

// NewMemoryGetImpactHandler cria o executor para a ferramenta MCP memory_get_impact
func NewMemoryGetImpactHandler(impactFn ImpactFunc) ToolHandlerFunc {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		var rawMap map[string]json.RawMessage
		if len(args) > 0 {
			if err := json.Unmarshal(args, &rawMap); err != nil {
				return nil, NewError(CodeInvalidParams, "Formato JSON de argumentos inválido", err.Error())
			}
		}

		var nodeID string
		if rawNode, ok := rawMap["node_id"]; ok {
			_ = json.Unmarshal(rawNode, &nodeID)
		}
		nodeID = strings.TrimSpace(nodeID)
		if nodeID == "" {
			return nil, NewError(CodeInvalidParams, "Parâmetro 'node_id' é obrigatório", nil)
		}

		maxDepth := 2
		if rawDepth, ok := rawMap["max_depth"]; ok {
			var d int
			if err := json.Unmarshal(rawDepth, &d); err == nil && d > 0 {
				maxDepth = d
			}
		}

		var repo string
		if rawRepo, ok := rawMap["repository"]; ok {
			_ = json.Unmarshal(rawRepo, &repo)
		}

		var res *graph.ImpactResult
		var err error
		if impactFn != nil {
			res, err = impactFn(ctx, repo, nodeID, maxDepth)
			if err != nil {
				return nil, NewError(CodeInternalError, "Falha ao calcular análise de impacto", err.Error())
			}
		} else {
			res = &graph.ImpactResult{
				TargetNode:         nodeID,
				MaxDepthReached:    0,
				TotalImpacted:      0,
				DirectDependents:   0,
				IndirectDependents: 0,
				RiskScore:          0.0,
				RiskLevel:          "BAIXO",
				Nodes:              []graph.ImpactedNode{},
				SeverityCounts:     make(map[graph.ImpactSeverity]int),
				AffectedClusters:   []int{},
			}
		}

		var sb strings.Builder
		riskEmoji := "🟢"
		switch strings.ToUpper(res.RiskLevel) {
		case "CRÍTICO", "CRITICAL":
			riskEmoji = "🔴"
		case "ALTO", "HIGH":
			riskEmoji = "🟠"
		case "MODERADO", "MEDIUM":
			riskEmoji = "🟡"
		case "BAIXO", "LOW":
			riskEmoji = "🟢"
		}

		sb.WriteString("### 💥 Análise de Impacto e Raio de Destruição (Blast Radius)\n\n")
		sb.WriteString(fmt.Sprintf("- **Nó Alvo**: `%s`\n", res.TargetNode))
		sb.WriteString(fmt.Sprintf("- **Score de Risco**: %s `%.1f / 100` (`%s`)\n", riskEmoji, res.RiskScore, res.RiskLevel))
		sb.WriteString(fmt.Sprintf("- **Total Afetado**: %d nós (Diretos: %d, Indiretos: %d)\n", res.TotalImpacted, res.DirectDependents, res.IndirectDependents))
		sb.WriteString(fmt.Sprintf("- **Profundidade Máxima Atingida**: %d (Limite Solicitado: %d)\n", res.MaxDepthReached, maxDepth))

		if len(res.AffectedClusters) > 0 {
			clustersStr := make([]string, len(res.AffectedClusters))
			for i, c := range res.AffectedClusters {
				clustersStr[i] = fmt.Sprintf("Cluster %d", c)
			}
			sb.WriteString(fmt.Sprintf("- **Clusters Afetados**: %s\n", strings.Join(clustersStr, ", ")))
		}

		sb.WriteString(fmt.Sprintf("- **Severidades**: 🔴 CRÍTICO: %d | 🟠 ALTO: %d | 🟡 MÉDIO: %d | 🟢 BAIXO: %d\n\n",
			res.SeverityCounts[graph.SeverityCritical],
			res.SeverityCounts[graph.SeverityHigh],
			res.SeverityCounts[graph.SeverityMedium],
			res.SeverityCounts[graph.SeverityLow],
		))

		if len(res.Nodes) == 0 {
			sb.WriteString("*Nenhum nó dependente identificado dentro da profundidade configurada.*\n")
		} else {
			sb.WriteString("| Profundidade | Severidade | Relação | Via Nó | Tipo | ID Afetado |\n")
			sb.WriteString("|---|---|---|---|---|---|\n")
			for _, n := range res.Nodes {
				via := n.ViaNode
				if via == res.TargetNode {
					via = "(direto)"
				}
				sb.WriteString(fmt.Sprintf("| %d | `%s` | %s | `%s` | %s | `%s` |\n",
					n.Depth, n.Severity, n.Relation, via, n.NodeType, n.ID))
			}
		}

		return NewTextResult(sb.String()), nil
	}
}

// SetImpactHandler registra ou atualiza o handler de impacto no servidor MCP
func (s *Server) SetImpactHandler(fn ImpactFunc) {
	s.RegisterTool(ToolMemoryGetImpact, NewMemoryGetImpactHandler(fn))
}
