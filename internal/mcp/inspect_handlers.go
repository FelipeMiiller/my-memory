package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/graph"
)

// InspectFunc define a assinatura para inspeção cirúrgica de nós (triptych view)
type InspectFunc func(ctx context.Context, repo string, nodeID string, maxContentLen int) (*graph.TriptychView, error)

// NewMemoryInspectNodeHandler cria o executor para a ferramenta MCP memory_inspect_node
func NewMemoryInspectNodeHandler(inspectFn InspectFunc) ToolHandlerFunc {
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

		maxContentLen := 500
		if rawLen, ok := rawMap["max_content_length"]; ok {
			var l int
			if err := json.Unmarshal(rawLen, &l); err == nil && l >= 0 {
				maxContentLen = l
			}
		}

		var repo string
		if rawRepo, ok := rawMap["repository"]; ok {
			_ = json.Unmarshal(rawRepo, &repo)
		}

		var res *graph.TriptychView
		var err error
		if inspectFn != nil {
			res, err = inspectFn(ctx, repo, nodeID, maxContentLen)
			if err != nil {
				return nil, NewError(CodeInternalError, "Falha na inspeção do nó", err.Error())
			}
		} else {
			res = &graph.TriptychView{
				Target: graph.NodeSummary{
					ID:    nodeID,
					Title: nodeID,
					Type:  "note",
				},
				Inbound:  []graph.InboundLink{},
				Outbound: []graph.OutboundLink{},
			}
		}

		var sb strings.Builder
		riskEmoji := "🟢"
		switch strings.ToUpper(res.Target.RiskLevel) {
		case "CRÍTICO", "CRITICAL":
			riskEmoji = "🔴"
		case "ALTO", "HIGH":
			riskEmoji = "🟠"
		case "MODERADO", "MEDIUM":
			riskEmoji = "🟡"
		}

		sb.WriteString("### 🔬 Visualização Cirúrgica de Nó (Triptych Node Inspector)\n\n")

		// 1. Nó Central
		sb.WriteString(fmt.Sprintf("#### 🎯 Nó Central: `%s`\n\n", res.Target.Title))
		sb.WriteString(fmt.Sprintf("- **ID Canônico**: `%s`\n", res.Target.ID))
		if res.Target.Path != "" {
			sb.WriteString(fmt.Sprintf("- **Caminho do Arquivo**: `%s`\n", res.Target.Path))
		}
		sb.WriteString(fmt.Sprintf("- **Tipo**: `%s` | **PageRank**: `%.4f`\n", res.Target.Type, res.Target.PageRank))
		if res.Target.CommunityID > 0 {
			label := res.Target.CommunityLabel
			if label == "" {
				label = "Geral"
			}
			sb.WriteString(fmt.Sprintf("- **Comunidade Temática**: `Cluster #%d` (`%s`)\n", res.Target.CommunityID, label))
		}
		sb.WriteString(fmt.Sprintf("- **Risco de Quebra (Blast Radius)**: %s `%.1f / 100` (`%s`)\n", riskEmoji, res.Target.RiskScore, res.Target.RiskLevel))
		if len(res.Target.Tags) > 0 {
			sb.WriteString(fmt.Sprintf("- **Tags**: `%s`\n", strings.Join(res.Target.Tags, ", ")))
		}

		if res.Target.ContentPreview != "" {
			sb.WriteString(fmt.Sprintf("\n##### 📄 Pré-visualização de Conteúdo (%d caracteres):\n\n", res.Target.ContentLength))
			sb.WriteString("```markdown\n")
			sb.WriteString(res.Target.ContentPreview)
			sb.WriteString("\n```\n")
		}

		sb.WriteString("\n---\n\n")

		// 2. Chamadores e Dependentes (Inbound)
		sb.WriteString(fmt.Sprintf("#### ⬅️ Chamadores & Dependentes Reversos (Inbound: %d)\n\n", res.TotalInbound))
		if res.CriticalDependents > 0 {
			sb.WriteString(fmt.Sprintf("> ⚠️ **Alerta de Risco**: Identificado(s) **%d** dependente(s) com severidade crítica ou alta!\n\n", res.CriticalDependents))
		}

		if len(res.Inbound) == 0 {
			sb.WriteString("*Nenhum nó dependente apontando para este documento.*\n\n")
		} else {
			sb.WriteString("| Severidade | Tipo | Relação | PageRank | ID / Documento Chamador |\n")
			sb.WriteString("|---|---|---|---|---|\n")
			for _, in := range res.Inbound {
				badge := "🟢 `LOW`"
				switch in.Severity {
				case graph.SeverityCritical:
					badge = "🔴 `CRITICAL`"
				case graph.SeverityHigh:
					badge = "🟠 `HIGH`"
				case graph.SeverityMedium:
					badge = "🟡 `MEDIUM`"
				}
				sb.WriteString(fmt.Sprintf("| %s | `%s` | `%s` | `%.4f` | `%s` (%s) |\n",
					badge, in.Type, in.Relation, in.PageRank, in.SourceID, in.Title))
			}
			sb.WriteString("\n")
		}

		sb.WriteString("---\n\n")

		// 3. Referências e Saída (Outbound)
		sb.WriteString(fmt.Sprintf("#### ➡️ Referências & Saída (Outbound: %d)\n\n", res.TotalOutbound))
		if len(res.Outbound) == 0 {
			sb.WriteString("*Este documento não aponta para nenhum outro nó no grafo.*\n\n")
		} else {
			sb.WriteString("| Status | Tipo | Relação | PageRank | ID / Documento Alvo |\n")
			sb.WriteString("|---|---|---|---|---|\n")
			for _, out := range res.Outbound {
				status := "✓ Válido"
				if out.IsFederated || strings.HasPrefix(out.TargetID, "memory://") {
					status = "🌐 Federado (is_federated: true)"
				} else if !out.Exists {
					status = "⚠️ DEAD LINK"
				}
				sb.WriteString(fmt.Sprintf("| %s | `%s` | `%s` | `%.4f` | `%s` (%s) |\n",
					status, out.Type, out.Relation, out.PageRank, out.TargetID, out.Title))
			}
			sb.WriteString("\n")
		}

		return NewTextResult(sb.String()), nil
	}
}

// SetInspectHandler registra ou atualiza o executor de inspeção no servidor MCP
func (s *Server) SetInspectHandler(fn InspectFunc) {
	s.RegisterTool(ToolMemoryInspectNode, NewMemoryInspectNodeHandler(fn))
}
