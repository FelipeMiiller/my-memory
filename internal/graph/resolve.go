package graph

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Candidate representa um nó candidato em uma resolução de caminho ambígua.
type Candidate struct {
	ID      string `json:"id"`
	RelPath string `json:"rel_path"`
	Title   string `json:"title"`
}

// AmbiguityReport reporta múltiplos candidatos encontrados para um identificador de nó.
type AmbiguityReport struct {
	Query      string      `json:"query"`
	Candidates []Candidate `json:"candidates"`
}

// AmbiguousNodeError é um alias de tipo para AmbiguityReport para interoperabilidade de erros.
type AmbiguousNodeError = AmbiguityReport

// NewAmbiguityReport cria um relatório de ambiguidade com os candidatos ordenados
// por caminho relativo e, no desempate, por ID.
func NewAmbiguityReport(query string, candidates []Candidate) *AmbiguityReport {
	cands := make([]Candidate, len(candidates))
	copy(cands, candidates)
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].RelPath != cands[j].RelPath {
			return cands[i].RelPath < cands[j].RelPath
		}
		return cands[i].ID < cands[j].ID
	})
	return &AmbiguityReport{
		Query:      query,
		Candidates: cands,
	}
}

// Error implementa a interface error para AmbiguityReport.
func (r *AmbiguityReport) Error() string {
	if r == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("identificador '%s' é ambíguo (%d candidatos encontrados):\n", r.Query, len(r.Candidates)))
	for _, c := range r.Candidates {
		title := c.Title
		if title == "" {
			title = "(sem título)"
		}
		sb.WriteString(fmt.Sprintf("  - %s (título: %s) [desambiguação: caminho relativo com ou sem .md, ou 'title:%s']\n",
			c.RelPath, title, c.Title))
	}
	sb.WriteString("Para desambiguar, utilize o caminho relativo (com sufixo .md opcional) ou o prefixo 'title:<título>'.")
	return sb.String()
}

// FormatCLI formata o relatório de ambiguidade para a CLI.
func (r *AmbiguityReport) FormatCLI() string {
	return r.Error()
}

// IsAmbiguousNode verifica se o erro é ou encapsula um relatório de ambiguidade.
func IsAmbiguousNode(err error) bool {
	var amb *AmbiguityReport
	return errors.As(err, &amb)
}

// AsAmbiguityReport extrai o relatório de ambiguidade de um erro, se presente.
func AsAmbiguityReport(err error) (*AmbiguityReport, bool) {
	var amb *AmbiguityReport
	if errors.As(err, &amb) {
		return amb, true
	}
	return nil, false
}
