package store

import (
	"github.com/FelipeMiiller/my-memory/internal/vaultfs"
)

// LinkCandidate é um link que a query de dead link suspeitou: o alvo não
// bate com documento indexado, título, tag conhecida nem símbolo de código.
//
// A suspeita não é veredito. O candidato só é link rompido depois de a
// checagem em disco dizer que o alvo não existe.
type LinkCandidate struct {
	SourceID string
	TargetID string
	Relation string
}

// LinkClassification é a classificação dos candidatos de dead link.
type LinkClassification struct {
	// Broken são os links comprovadamente rompidos: o alvo não existe em
	// disco nem como documento indexado. São os únicos que penalizam o
	// Health Score e os únicos que a correção apaga.
	Broken []DeadLink
	// OutOfIndex são os links cujo alvo existe em disco mas está fora do
	// índice por decisão. Não penalizam o score e a correção os preserva.
	OutOfIndex []OutOfIndexLink
	// Unverifiable são os links que não puderam ser verificados porque a raiz
	// do vault não estava acessível. Penalizam como link morto (o doctor não
	// tem prova de que o alvo existe) e tornam o score provisório.
	Unverifiable []DeadLink
}

// ClassifyLinks classifica os candidatos de dead link contra o disco do vault.
//
// Um resolver nil significa "sem raiz legível": todos os candidatos ficam não
// verificáveis. Sem prova, o doctor não declara um link rompido — a suspeita
// mantém a penalidade e o score fica provisório. É o "verde-mentira" que a
// ISSUE-033 verificou pelo avesso.
func ClassifyLinks(candidates []LinkCandidate, resolver *vaultfs.Resolver) LinkClassification {
	out := LinkClassification{
		Broken:       []DeadLink{},
		OutOfIndex:   []OutOfIndexLink{},
		Unverifiable: []DeadLink{},
	}

	for _, c := range candidates {
		link := DeadLink{SourceID: c.SourceID, TargetID: c.TargetID, Relation: c.Relation}

		if resolver == nil {
			out.Unverifiable = append(out.Unverifiable, link)
			continue
		}

		res := resolver.Resolve(c.TargetID)
		if !res.Found {
			out.Broken = append(out.Broken, link)
			continue
		}

		out.OutOfIndex = append(out.OutOfIndex, OutOfIndexLink{
			SourceID:     c.SourceID,
			TargetID:     c.TargetID,
			Relation:     c.Relation,
			ResolvedPath: res.Path,
			Match:        res.Match,
		})
	}

	return out
}

// ApplyLinkClassification preenche as listas de link do relatório e marca o
// score como provisório quando houve link não verificável.
//
// É o único ponto que decide a lista de dead links do relatório: a rotina de
// correção e o relatório passam a ler a mesma classificação (a lição da
// ISSUE-038).
func (r *DoctorReport) ApplyLinkClassification(c LinkClassification) {
	r.DeadLinks = c.Broken
	r.OutOfIndexLinks = c.OutOfIndex
	r.UnverifiableLinks = c.Unverifiable
	r.HealthScoreProvisional = len(c.Unverifiable) > 0
}
