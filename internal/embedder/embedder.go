package embedder

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

// Compila: ambos os providers implementam o contrato.
var (
	_ Embedder = (*OllamaClient)(nil)
)
