package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/drift"
)

// DriftAnalysisFunc define a assinatura para execução da análise de drift
type DriftAnalysisFunc func(ctx context.Context, rangeStr string, threshold float64, includeUncovered bool, repoSlug string) (*drift.DriftReport, error)

// MemoryGetDriftParams representa os parâmetros de entrada para a ferramenta memory_get_drift
type MemoryGetDriftParams struct {
	Since            string   `json:"since,omitempty"`
	Threshold        *float64 `json:"threshold,omitempty"`
	IncludeUncovered *bool    `json:"include_uncovered,omitempty"`
	Repository       string   `json:"repository,omitempty"`
}

// NewMemoryGetDriftHandler cria um executor para a ferramenta MCP memory_get_drift
func NewMemoryGetDriftHandler(analyzer DriftAnalysisFunc) ToolHandlerFunc {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		var params MemoryGetDriftParams
		if len(args) > 0 && string(args) != "{}" {
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, fmt.Errorf("parâmetros inválidos para memory_get_drift: %w", err)
			}
		}

		since := params.Since
		if since == "" {
			since = "HEAD~5..HEAD"
		}

		threshold := 0.20
		if params.Threshold != nil {
			threshold = *params.Threshold
		}

		includeUncovered := true
		if params.IncludeUncovered != nil {
			includeUncovered = *params.IncludeUncovered
		}

		if analyzer == nil {
			return NewTextResult("Nenhum motor de drift configurado no servidor."), nil
		}

		report, err := analyzer(ctx, since, threshold, includeUncovered, params.Repository)
		if err != nil {
			return nil, fmt.Errorf("erro ao analisar desvio semântico: %w", err)
		}

		var sb strings.Builder
		sb.WriteString("### 🧭 Diagnóstico de Desvio Código-Memória (Semantic Drift)\n\n")
		sb.WriteString(fmt.Sprintf("- **Faixa Git Analisada:** `%s`\n", report.RangeStr))
		sb.WriteString(fmt.Sprintf("- **Commits no Período:** %d\n", report.TotalCommits))
		sb.WriteString(fmt.Sprintf("- **Arquivos Modificados:** %d\n", report.TotalChanged))
		sb.WriteString(fmt.Sprintf("- **Score Geral de Drift:** `%.1f/100`\n", report.OverallScore))
		sb.WriteString(fmt.Sprintf("- **Severidades:** 🔴 Crítico: %d | 🟠 Alto: %d | 🟡 Médio: %d | 🟢 Baixo: %d\n\n",
			report.CriticalCount, report.HighCount, report.MediumCount, report.LowCount))

		if len(report.DriftedNotes) == 0 {
			sb.WriteString("✅ **Nenhuma nota ou ADR com desvio significativo identificado.** A documentação reflete fielmente as mudanças recentes de código.\n\n")
		} else {
			sb.WriteString("#### 📑 Notas e Decisões Defasadas em Relação ao Código\n\n")
			for i, n := range report.DriftedNotes {
				badge := "🟢 `[LOW]`"
				switch n.Severity {
				case drift.SeverityCritical:
					badge = "🔴 `[CRITICAL]`"
				case drift.SeverityHigh:
					badge = "🟠 `[HIGH]`"
				case drift.SeverityMedium:
					badge = "🟡 `[MEDIUM]`"
				}

				sb.WriteString(fmt.Sprintf("%d. %s **%s** (Score: `%.1f/100`)\n", i+1, badge, n.Title, n.DriftScore))
				sb.WriteString(fmt.Sprintf("   - **Caminho:** `%s`\n", n.Path))
				sb.WriteString(fmt.Sprintf("   - **Diagnóstico:** %s\n", n.Reason))
				if len(n.AffectedFiles) > 0 {
					sb.WriteString(fmt.Sprintf("   - **Arquivos Alterados:** `%s`\n", strings.Join(n.AffectedFiles, "`, `")))
				}
				if n.Links != nil {
					sb.WriteString(fmt.Sprintf("   - **Links Rápidos:** [Obsidian](%s) | [VS Code](%s)\n", n.Links.Obsidian, n.Links.VSCode))
				}
				sb.WriteString("\n")
			}
		}

		if len(report.UncoveredCode) > 0 {
			sb.WriteString("#### ⚠️ Código Órfão de Decisões (Sem Notas Vinculadas)\n\n")
			sb.WriteString("| Arquivo de Código | Status | Adições/Deleções | Ação Recomendada |\n")
			sb.WriteString("| :--- | :---: | :---: | :--- |\n")
			for _, uc := range report.UncoveredCode {
				sb.WriteString(fmt.Sprintf("| `%s` | `%s` | `+%d/-%d` | %s |\n",
					uc.FilePath, uc.Status, uc.Additions, uc.Deletions, uc.SuggestedAction))
			}
			sb.WriteString("\n")
		}

		return NewTextResult(sb.String()), nil
	}
}

// SetDriftHandler registra ou atualiza o executor de análise de drift no servidor MCP
func (s *Server) SetDriftHandler(analyzer DriftAnalysisFunc) {
	s.RegisterToolHandler(ToolMemoryGetDrift.Name, NewMemoryGetDriftHandler(analyzer))
}
