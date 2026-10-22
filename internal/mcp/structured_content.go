package mcp

import (
	"github.com/FelipeMiiller/my-memory/internal/drift"
	"github.com/FelipeMiiller/my-memory/internal/graph"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// normalizeStructuredContent garante que nenhum payload de `structuredContent`
// serialize um array como `null`.
//
// O `encoding/json` transforma slice nil em `null`, e slice nil é a forma
// natural de o Go representar "nenhum item". O schema `zod/mini` da viewer
// aceita `undefined` em `.optional()`, mas **não** `null`: um array vazio
// chegando como `null` derrubava a validação do schema, o cliente caía no
// fallback textual e o painel `/drift` renderizava markdown cru em vez da UI
// estruturada.
//
// A normalização fica aqui, na fronteira do wire, e não em cada produtor: o
// contrato do payload é "array vazio é `[]`", e ele vale para toda tool que
// um dia passe a devolver um array — não só para as que já existem hoje.
//
// A função é pura: os structs são copiados antes de normalizados, então o
// relatório devolvido pelo produtor não é alterado por quem o consome.
func normalizeStructuredContent(structured any) any {
	switch payload := structured.(type) {
	case []store.VaultDocument:
		return nonNilSlice(payload)
	case []store.VaultTag:
		return nonNilSlice(payload)
	case []OutlineHeading:
		return nonNilSlice(payload)
	case []GodNode:
		return nonNilSlice(payload)
	case []PageRankNode:
		return nonNilSlice(payload)
	case []SurprisingConnection:
		return nonNilSlice(payload)

	case graph.CommunityResult:
		normalized := payload
		communities := make([]graph.Community, len(payload.Communities))
		for i, community := range payload.Communities {
			community.Members = nonNilSlice(community.Members)
			communities[i] = community
		}
		normalized.Communities = communities
		return normalized

	case *drift.DriftReport:
		if payload == nil {
			return payload
		}
		normalized := *payload
		normalized.DriftedNotes = nonNilSlice(payload.DriftedNotes)
		normalized.UncoveredCode = nonNilSlice(payload.UncoveredCode)
		return &normalized

	case *DoctorReport:
		if payload == nil {
			return payload
		}
		normalized := *payload
		normalized.DeadLinks = nonNilSlice(payload.DeadLinks)
		normalized.OrphanNotes = nonNilSlice(payload.OrphanNotes)
		normalized.SelfLoops = nonNilSlice(payload.SelfLoops)
		normalized.DesyncedChunks = nonNilSlice(payload.DesyncedChunks)
		return &normalized

	default:
		return structured
	}
}

// nonNilSlice devolve o slice recebido, ou um slice vazio do mesmo tipo
// quando o original é nil.
func nonNilSlice[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
