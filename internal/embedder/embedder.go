package embedder

import "strconv"

// Embedder é o contrato mínimo de um provider de embeddings.
//
// Existe para que a cadeia de fallback seja testável e para que adicionar ou
// remover um provider não force mudar as assinaturas de ~15 funções do CLI.
//
// Schema: o vetor devolvido tem `EmbeddingDim` dimensões. Após a migração do
// schema (2026-10-13) isso é **384**, a dimensão nativa do
// paraphrase-multilingual-MiniLM-L12-v2.
//
// Ver ADR-035 e .specs/features/embedder-builtin-onnx-minilm/.
type Embedder interface {
	// GenerateEmbedding devolve o vetor do texto, já normalizado em L2.
	// Texto vazio é erro, nunca vetor zero: um vetor zero entra no índice
	// e degrada a busca sem deixar pista de por quê.
	GenerateEmbedding(text string) ([]float32, error)
}

// EmbeddingDim é a dimensão do vetor no schema do banco.
//
// 384 = dimensão nativa do paraphrase-multilingual-MiniLM-L12-v2 (ADR-035).
// Foi 768 enquanto o provider era o `nomic-embed-text` do Ollama; o provider
// builtin permitiu guardar o vetor nativo e evitar a projeção 384→768, que o
// benchmark mostrou perder separação par a par.
const EmbeddingDim = 384

// ModelID identifica o modelo de embedding embutido. Entra no Epoch: trocar de
// modelo sem trocar de dimensão ainda assim invalida todos os vetores gravados.
const ModelID = "paraphrase-multilingual-MiniLM-L12-v2"

// vectorRecipe versiona **como** o vetor é calculado a partir do modelo, para o
// caso de a conta mudar sem o modelo ou a dimensão mudarem (mean pooling → CLS,
// L2 → cosseno, normalização de acentos no texto de entrada...).
//
// Regra: bumpar esta constante quando o cálculo do vetor mudar. Sem isso, um
// documento continuaria "cacheado" com o content_hash igual e o índice passaria
// a misturar vetores de receitas diferentes sem nenhum sinal.
const vectorRecipe = "v1"

// Epoch devolve a impressão digital de tudo que altera o *significado* de um
// vetor já gravado no banco: modelo, dimensão e receita de cálculo.
//
// É o par do `content_hash` (ADR-010). O hash responde "o arquivo mudou?"; o
// epoch responde "o que produziu o embedding ainda é a mesma coisa?". Sem ele,
// trocar o embedder deixa o cache decidir que nada mudou — foi exatamente o que
// aconteceu na migração 768 → 384 e só não deixou vetores velhos porque a
// base foi apagada à mão na occasion.
func Epoch() string {
	return ModelID + ":" + strconv.Itoa(EmbeddingDim) + ":" + vectorRecipe
}

// EpochFor devolve o Epoch corrente, ou "" quando não há provider de embedding
// (indexação em modo léxico FTS). Nesse caso o gate compara "" com "" e o cache
// por hash continua valendo — corretamente, já que nada vetorial foi gravado.
func EpochFor(e Embedder) string {
	if e == nil {
		return ""
	}
	return Epoch()
}

// Compila: ambos os providers implementam o contrato.
var (
	_ Embedder = (*OllamaClient)(nil)
)
