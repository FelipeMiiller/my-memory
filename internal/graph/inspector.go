package graph

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/deeplink"
)

// InboundLink representa uma aresta de entrada (quem aponta para o nó inspecionado)
type InboundLink struct {
	SourceID     string         `json:"source_id"`
	Title        string         `json:"title"`
	Type         string         `json:"type"`
	Relation     string         `json:"relation"`
	Severity     ImpactSeverity `json:"severity"`
	Weight       float64        `json:"weight"`
	PageRank     float64        `json:"pagerank,omitempty"`
	CommunityID  int            `json:"community_id,omitempty"`
	CommunityTag string         `json:"community_tag,omitempty"`
}

// OutboundLink representa uma aresta de saída (para onde o nó inspecionado aponta)
type OutboundLink struct {
	TargetID     string  `json:"target_id"`
	Title        string  `json:"title"`
	Type         string  `json:"type"`
	Relation     string  `json:"relation"`
	Weight       float64 `json:"weight"`
	PageRank     float64 `json:"pagerank,omitempty"`
	Exists       bool    `json:"exists"` // true se o nó de destino está registrado na base
	CommunityID  int     `json:"community_id,omitempty"`
	CommunityTag string  `json:"community_tag,omitempty"`
}

// NodeSummary sintetiza os atributos canônicos e métricas centrais do nó alvo
type NodeSummary struct {
	ID             string              `json:"id"`
	Title          string              `json:"title"`
	Path           string              `json:"path,omitempty"`
	Type           string              `json:"type"` // "decision", "concept", "guide", etc.
	Tags           []string            `json:"tags,omitempty"`
	PageRank       float64             `json:"pagerank"`
	CommunityID    int                 `json:"community_id,omitempty"`
	CommunityLabel string              `json:"community_label,omitempty"`
	RiskScore      float64             `json:"risk_score"`
	RiskLevel      string              `json:"risk_level"`
	ContentPreview string              `json:"content_preview,omitempty"`
	ContentLength  int                 `json:"content_length"`
	UpdatedAt      time.Time           `json:"updated_at,omitempty"`
	Links          *deeplink.DeepLinks `json:"links,omitempty"`
}

// TriptychView organiza a visualização cirúrgica em 3 colunas espaciais
type TriptychView struct {
	Target             NodeSummary    `json:"target"`
	Inbound            []InboundLink  `json:"inbound"`
	Outbound           []OutboundLink `json:"outbound"`
	TotalInbound       int            `json:"total_inbound"`
	TotalOutbound      int            `json:"total_outbound"`
	CriticalDependents int            `json:"critical_dependents"`
	GeneratedAt        time.Time      `json:"generated_at"`
}

// InspectorOptions define opções configuráveis para montagem do tríptico
type InspectorOptions struct {
	MaxContentLength int                `json:"max_content_length"`
	PageRanks        map[string]float64 `json:"page_ranks,omitempty"`
	Communities      map[string]int     `json:"communities,omitempty"`
	CommunityLabels  map[int]string     `json:"community_labels,omitempty"`
	NodeTypes        map[string]string  `json:"node_types,omitempty"`
	Titles           map[string]string  `json:"titles,omitempty"`
	ExistingNodes    map[string]bool    `json:"existing_nodes,omitempty"`
	RiskResult       *ImpactResult      `json:"risk_result,omitempty"`
	RepoRoot         string             `json:"repo_root,omitempty"`
	VaultName        string             `json:"vault_name,omitempty"`
}

// DefaultInspectorOptions retorna configurações padrões de inspeção
func DefaultInspectorOptions() InspectorOptions {
	return InspectorOptions{
		MaxContentLength: 500,
	}
}

// TruncateContent trunca texto preservando limites de caracteres UTF-8 de forma segura
func TruncateContent(content string, maxLen int) string {
	if maxLen <= 0 {
		return content
	}
	runes := []rune(content)
	if len(runes) <= maxLen {
		return content
	}
	return string(runes[:maxLen]) + " ... [truncado]"
}

// BuildTriptychView compila a visualização cirúrgica em 3 colunas para um nó alvo
func BuildTriptychView(target NodeSummary, edges []WeightedEdge, opts InspectorOptions) (*TriptychView, error) {
	cleanID := strings.TrimSpace(target.ID)
	if cleanID == "" {
		return nil, fmt.Errorf("identificador do nó alvo não pode ser vazio")
	}

	target.ID = cleanID
	if target.Title == "" {
		if opts.Titles != nil && opts.Titles[cleanID] != "" {
			target.Title = opts.Titles[cleanID]
		} else {
			target.Title = cleanID
		}
	}

	if target.Type == "" {
		if opts.NodeTypes != nil && opts.NodeTypes[cleanID] != "" {
			target.Type = opts.NodeTypes[cleanID]
		} else {
			target.Type = "note"
		}
	}

	if target.PageRank == 0 && opts.PageRanks != nil {
		target.PageRank = opts.PageRanks[cleanID]
	}

	if target.CommunityID == 0 && opts.Communities != nil {
		target.CommunityID = opts.Communities[cleanID]
	}

	if target.CommunityLabel == "" && target.CommunityID > 0 && opts.CommunityLabels != nil {
		target.CommunityLabel = opts.CommunityLabels[target.CommunityID]
	}

	if opts.RiskResult != nil {
		target.RiskScore = opts.RiskResult.RiskScore
		target.RiskLevel = opts.RiskResult.RiskLevel
	}

	if target.ContentLength == 0 && target.ContentPreview != "" {
		target.ContentLength = len([]rune(target.ContentPreview))
	}

	if opts.MaxContentLength > 0 && target.ContentPreview != "" {
		target.ContentPreview = TruncateContent(target.ContentPreview, opts.MaxContentLength)
	}

	if target.Links == nil {
		pathForLinks := target.Path
		if pathForLinks == "" {
			pathForLinks = target.ID
		}
		if pathForLinks != "" {
			l := deeplink.GenerateLinks(opts.RepoRoot, opts.VaultName, pathForLinks, 0)
			target.Links = &l
		}
	}

	var inbound []InboundLink
	var outbound []OutboundLink
	criticalCount := 0

	for _, e := range edges {
		// Inbound link: quem aponta para o alvo
		if e.Target == cleanID && e.Source != cleanID {
			rel := e.Type
			if strings.TrimSpace(rel) == "" {
				rel = "links_to"
			}
			sev := DetermineSeverity(rel, 1)

			srcTitle := e.Source
			if opts.Titles != nil && opts.Titles[e.Source] != "" {
				srcTitle = opts.Titles[e.Source]
			}

			srcType := "note"
			if opts.NodeTypes != nil && opts.NodeTypes[e.Source] != "" {
				srcType = opts.NodeTypes[e.Source]
			}

			pr := 0.0
			if opts.PageRanks != nil {
				pr = opts.PageRanks[e.Source]
			}

			commID := 0
			commTag := ""
			if opts.Communities != nil {
				commID = opts.Communities[e.Source]
				if opts.CommunityLabels != nil {
					commTag = opts.CommunityLabels[commID]
				}
			}

			if sev == SeverityCritical || sev == SeverityHigh {
				criticalCount++
			}

			inbound = append(inbound, InboundLink{
				SourceID:     e.Source,
				Title:        srcTitle,
				Type:         srcType,
				Relation:     rel,
				Severity:     sev,
				Weight:       e.Weight,
				PageRank:     pr,
				CommunityID:  commID,
				CommunityTag: commTag,
			})
		}

		// Outbound link: para onde o alvo aponta
		if e.Source == cleanID && e.Target != cleanID {
			rel := e.Type
			if strings.TrimSpace(rel) == "" {
				rel = "links_to"
			}

			tgtTitle := e.Target
			if opts.Titles != nil && opts.Titles[e.Target] != "" {
				tgtTitle = opts.Titles[e.Target]
			}

			tgtType := "note"
			if opts.NodeTypes != nil && opts.NodeTypes[e.Target] != "" {
				tgtType = opts.NodeTypes[e.Target]
			}

			pr := 0.0
			if opts.PageRanks != nil {
				pr = opts.PageRanks[e.Target]
			}

			exists := true
			if opts.ExistingNodes != nil {
				exists = opts.ExistingNodes[e.Target]
			}

			commID := 0
			commTag := ""
			if opts.Communities != nil {
				commID = opts.Communities[e.Target]
				if opts.CommunityLabels != nil {
					commTag = opts.CommunityLabels[commID]
				}
			}

			outbound = append(outbound, OutboundLink{
				TargetID:     e.Target,
				Title:        tgtTitle,
				Type:         tgtType,
				Relation:     rel,
				Weight:       e.Weight,
				PageRank:     pr,
				Exists:       exists,
				CommunityID:  commID,
				CommunityTag: commTag,
			})
		}
	}

	// Ordenação determinística de Inbound: Severidade (CRITICAL primeiro) -> PageRank DESC -> SourceID ASC
	severityRank := func(s ImpactSeverity) int {
		switch s {
		case SeverityCritical:
			return 0
		case SeverityHigh:
			return 1
		case SeverityMedium:
			return 2
		case SeverityLow:
			return 3
		default:
			return 4
		}
	}

	sort.Slice(inbound, func(i, j int) bool {
		rI := severityRank(inbound[i].Severity)
		rJ := severityRank(inbound[j].Severity)
		if rI != rJ {
			return rI < rJ
		}
		if inbound[i].PageRank != inbound[j].PageRank {
			return inbound[i].PageRank > inbound[j].PageRank
		}
		return inbound[i].SourceID < inbound[j].SourceID
	})

	// Ordenação determinística de Outbound: Exists DESC (links válidos primeiro) -> Relation ASC -> TargetID ASC
	sort.Slice(outbound, func(i, j int) bool {
		if outbound[i].Exists != outbound[j].Exists {
			return outbound[i].Exists // true antes de false
		}
		if outbound[i].Relation != outbound[j].Relation {
			return outbound[i].Relation < outbound[j].Relation
		}
		return outbound[i].TargetID < outbound[j].TargetID
	})

	return &TriptychView{
		Target:             target,
		Inbound:            inbound,
		Outbound:           outbound,
		TotalInbound:       len(inbound),
		TotalOutbound:      len(outbound),
		CriticalDependents: criticalCount,
		GeneratedAt:        time.Now(),
	}, nil
}
