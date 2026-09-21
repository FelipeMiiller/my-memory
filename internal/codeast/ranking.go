package codeast

import (
	"sort"
	"strings"
)

// BoostFactor é o multiplicador aplicado ao score RRF de um code_symbol
// quando o query match exato em qualified_name (CA-05 / ADR-047).
//
// Constante em vez de parâmetro para tornar trivial o override futuro
// (por ADR / config / flag), sem alterar a assinatura dos helpers.
const BoostFactor = 2.0

// CodeSearchHit representa um code_symbol candidato a resultado de busca,
// enriquecido com score final já com boost aplicado quando aplicável.
type CodeSearchHit struct {
	SymbolID      int64   `json:"symbol_id"`
	QualifiedName string  `json:"qualified_name"`
	Name          string  `json:"name"`
	Kind          string  `json:"kind"`
	Language      string  `json:"language"`
	FilePath      string  `json:"file_path"`
	StartLine     int     `json:"start_line"`
	EndLine       int     `json:"end_line"`
	Signature     string  `json:"signature,omitempty"`
	DocComment    string  `json:"doc_comment,omitempty"`
	RRFScore      float64 `json:"rrf_score"`
	Boosted       bool    `json:"boosted"`
}

// IsQualifiedNameMatchExact devolve true quando `query` é igual ao
// qualified_name do símbolo (case-sensitive após TrimSpace). É a forma
// canônica do "boost 2x" — match exato, não substring (CA-05).
func IsQualifiedNameMatchExact(query, qualifiedName string) bool {
	q := strings.TrimSpace(query)
	qn := strings.TrimSpace(qualifiedName)
	if q == "" || qn == "" {
		return false
	}
	return q == qn
}

// ShouldApplyBoost decide se o score de uma hit deve receber o boost CA-05.
//
// Regras (ADR-047 §CA-05):
//   - query deve casar com HasQualifiedNameShape (contém '.' e cada
//     segmento é identificador válido).
//   - query deve casar EXATAMENTE com qualified_name do símbolo
//     (case-sensitive).
//   - noBoost deve ser false.
//
// Retorna também o motivo pelo qual o boost foi aplicado (útil para logging).
func ShouldApplyBoost(query, qualifiedName string, noBoost bool) (apply bool, reason string) {
	if noBoost {
		return false, "no-boost flag"
	}
	if !HasQualifiedNameShape(query) {
		return false, "query sem formato qualified_name"
	}
	if !IsQualifiedNameMatchExact(query, qualifiedName) {
		return false, "qualified_name difere do query"
	}
	return true, "qualified_name match exato"
}

// ApplyBoost multiplica o RRF score quando ShouldApplyBoost retornar true.
// Quando não aplica, devolve o score inalterado.
func ApplyBoost(score float64, query, qualifiedName string, noBoost bool) (newScore float64, boosted bool) {
	if score < 0 {
		score = 0
	}
	if apply, _ := ShouldApplyBoost(query, qualifiedName, noBoost); apply {
		return score * BoostFactor, true
	}
	return score, false
}

// ApplyBoostToHits aplica o boost em uma fatia de hits e a devolve ordenada
// por score decrescente (estável: desempate por SymbolID).
//
// A ordenação é feita in-place para reduzir alocações; a função também
// marca o campo `Boosted` em cada hit afetada para serialização downstream.
func ApplyBoostToHits(hits []CodeSearchHit, query string, noBoost bool) []CodeSearchHit {
	if len(hits) == 0 {
		return hits
	}
	for i := range hits {
		newScore, boosted := ApplyBoost(hits[i].RRFScore, query, hits[i].QualifiedName, noBoost)
		hits[i].RRFScore = newScore
		hits[i].Boosted = boosted
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].RRFScore == hits[j].RRFScore {
			return hits[i].SymbolID < hits[j].SymbolID
		}
		return hits[i].RRFScore > hits[j].RRFScore
	})
	return hits
}

// ComputeRRFScore combina N ranked lists em um único score via Reciprocal
// Rank Fusion (ADR-009) — utilitário exposto para testes e consumidores
// que precisem reproduzir o score sem reimplementar.
//
// k é a constante de suavização canônica do RRF (padrão 60 em ADR-009).
// Para cada hit, `score += 1 / (k + rank)` onde `rank` é 1-based.
func ComputeRRFScore(ranks map[int64]int, k int) float64 {
	if k <= 0 {
		k = 60
	}
	var s float64
	for _, rank := range ranks {
		if rank <= 0 {
			continue
		}
		s += 1.0 / float64(k+rank)
	}
	return s
}
