//go:build onnx

package embedder

import (
	"math"
	"math/rand"
	"strings"
)

// schemaDim é a dimensão que o schema do banco exige.
//
// Após a migração de 2026-10-13 (T7) o schema é 384 e este código de projeção
// NÃO é mais usado no caminho de indexação: o BuiltinClient grava o vetor
// nativo. Fica aqui apenas para o benchmark histórico que comparou as três
// estratégias e para reproduzir a medição.
const schemaDim = EmbeddingDim

// projectKind identifica a estratégia de projeção 384d → 768d.
type projectKind int

const (
	projectJL      projectKind = iota // Johnson-Lindenstrauss
	projectPadZero                    // padding com zeros
)

// ProjectJL projeta um vetor nativo para a dimensão do schema usando um
// projetao aleatoria de Johnson-Lindenstrauss.
//
// A ideia: um mapa linear aleatório R preserva distâncias até distorção
// O(log n / sqrt(k)) — ou seja, o cosseno é praticamente mantido. E
// determinístico quando a seed é fixa, o que mantém o cache SHA-256 do
// índice consistente.
func ProjectJL(v []float32, outDim int, seed int64) []float32 {
	in := len(v)
	if in == 0 || outDim <= in {
		cp := make([]float32, in)
		copy(cp, v)
		return cp
	}
	rng := rand.New(rand.NewSource(seed))
	scale := float32(1.0 / math.Sqrt(float64(in)))

	// R é in×out com N(0,1). Gerado e aplicado em bloco para não segurar
	// in*out floats na memória de uma vez.
	out := make([]float32, outDim)
	col := make([]float32, in)
	for j := 0; j < outDim; j++ {
		for i := range col {
			col[i] = float32(rng.NormFloat64())
		}
		var acc float64
		for i := 0; i < in; i++ {
			acc += float64(v[i]) * float64(col[i])
		}
		out[j] = float32(acc) * scale
	}
	l2Normalize(out)
	return out
}

// ProjectPadZero preenche com zeros até a dimensão do schema.
//
// Só existe para a medição: não preserva similaridade alguma, já que metade
// das dimensões ficam zeradas e o cosseno só enxerga o subspace original
// (com norma reduzida). Serve de contraponto ao JL.
func ProjectPadZero(v []float32, outDim int) []float32 {
	out := make([]float32, outDim)
	copy(out, v)
	return out
}

// stopWordsLexical são palavras funcionais do português: carregam gramática,
// não tópico. Um par "Como faço X?" × "Como faço Y?" não é dissimilar por
// natureza, e o tokenSet precisa descartá-las.
var stopWordsLexical = map[string]bool{
	"a": true, "o": true, "as": true, "os": true, "um": true, "uma": true,
	"de": true, "da": true, "do": true, "das": true, "dos": true,
	"em": true, "no": true, "na": true, "nos": true, "nas": true,
	"para": true, "por": true, "com": true, "sem": true, "ate": true, "apos": true,
	"que": true, "e": true, "ou": true, "mas": true, "se": true, "como": true,
	"quando": true, "onde": true, "qual": true, "quais": true, "porque": true,
	"eu": true, "meu": true, "minha": true, "me": true, "seu": true, "sua": true,
	"ao": true, "aos": true, "pela": true, "pelo": true,
	"nao": true, "sim": true, "mais": true, "menos": true, "muito": true,
	"tambem": true, "entao": true, "depois": true, "antes": true, "ja": true,
	"sempre": true, "nunca": true, "agora": true, "bem": true, "mal": true,
	"so": true, "aqui": true, "la": true, "ali": true,
}

// tokenSet extrai o conjunto de palavras de conteúdo (sem função, sem acento
// tratado como separador). Serve de base para LexicalSimilarity.
func tokenSet(s string) map[string]bool {
	out := map[string]bool{}
	fields := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !('a' <= r && r <= 'z' || 'à' <= r && r <= 'ü' || 'A' <= r && r <= 'Z')
	})
	for _, w := range fields {
		if len(w) < 3 || stopWordsLexical[w] {
			continue
		}
		out[w] = true
	}
	return out
}

// LexicalSimilarity é um proxy de similaridade puramente léxica (Jaccard
// sobre tokens), que é o que um FTS5/BM25 consegue enxergar: sobreposição
// de termos, zero semântica.
//
// Existe para dar um piso honesto ao benchmark — um provider vetorial só
// vale se o gap sin-vs-dis for maior que o que a sobreposição de palavras já
// entrega sozinha.
func LexicalSimilarity(a, b string) float64 {
	ta := tokenSet(a)
	tb := tokenSet(b)
	if len(ta) == 0 || len(tb) == 0 {
		return 0
	}
	inter := 0
	for w := range ta {
		if tb[w] {
			inter++
		}
	}
	union := len(ta) + len(tb) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}
