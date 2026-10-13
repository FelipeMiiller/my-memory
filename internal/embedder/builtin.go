//go:build onnx

package embedder

import (
	"fmt"
	"math"
	"os"
	"strings"
	"sync"

	ort "github.com/yalue/onnxruntime_go"
)

// nativeDim é a dimensão de saída do paraphrase-multilingual-MiniLM-L12-v2.
// Não é 768: a projeção para 768 é assunto da T6, e este client devolve o
// vetor NATIVO para servir de base ao benchmark.
const nativeDim = 384

// Nomes de entrada e saída do grafo exportado pelo Xenova. `token_type_ids`
// é obrigatório mesmo em frase única — sem ele o runtime falha em
// /embeddings/token_type_embeddings/Gather com "Missing Input".
var (
	ortInputNames  = []string{"input_ids", "attention_mask", "token_type_ids"}
	ortOutputNames = []string{"last_hidden_state"}
)

// BuiltinConfig aponta para os artefatos locais do provider embutido.
type BuiltinConfig struct {
	ModelPath    string // .onnx
	TokenizerDir string // diretório com sentencepiece.bpe.model
	ORTLibPath   string // onnxruntime.dll / .so / .dylib
}

// BuiltinClient gera embeddings in-process, sem nenhum serviço externo.
type BuiltinClient struct {
	cfg     BuiltinConfig
	tok     *Tokenizer
	sess    *ort.DynamicAdvancedSession
	pool    sync.Pool // reuso de tensores de entrada
	initErr error
}

var (
	ortEnvOnce sync.Once
	ortEnvErr  error
)

// initORT inicializa o ambiente do ONNX Runtime uma vez por processo.
// SetSharedLibraryPath precisa vir antes de InitializeEnvironment e a DLL
// tem de ser a versão 1.29.0 — ver .specs/features/embedder-builtin-onnx-minilm/validation.md.
func initORT(libPath string) error {
	ortEnvOnce.Do(func() {
		if strings.TrimSpace(libPath) == "" {
			ortEnvErr = fmt.Errorf("builtin: ORTLibPath vazio — aponte para onnxruntime.dll (1.29.0)")
			return
		}
		if _, err := os.Stat(libPath); err != nil {
			ortEnvErr = fmt.Errorf("builtin: onnxruntime não encontrado em %q — use `mem embed download`: %w", libPath, err)
			return
		}
		ort.SetSharedLibraryPath(libPath)
		if err := ort.InitializeEnvironment(); err != nil {
			ortEnvErr = fmt.Errorf("builtin: falha ao inicializar ONNX Runtime %q: %w", libPath, err)
		}
	})
	return ortEnvErr
}

// NewBuiltinClient carrega tokenizer e modelo ONNX.
func NewBuiltinClient(cfg BuiltinConfig) (*BuiltinClient, error) {
	if err := initORT(cfg.ORTLibPath); err != nil {
		return nil, err
	}

	tok, err := NewTokenizerFromDir(cfg.TokenizerDir)
	if err != nil {
		return nil, fmt.Errorf("builtin: tokenizer indisponível: %w", err)
	}
	if _, err := os.Stat(cfg.ModelPath); err != nil {
		return nil, fmt.Errorf("builtin: modelo ONNX não encontrado em %q — use `mem embed download`: %w", cfg.ModelPath, err)
	}

	sess, err := ort.NewDynamicAdvancedSession(cfg.ModelPath, ortInputNames, ortOutputNames, nil)
	if err != nil {
		return nil, fmt.Errorf("builtin: falha ao carregar o modelo %q: %w", cfg.ModelPath, err)
	}

	c := &BuiltinClient{cfg: cfg, tok: tok, sess: sess}
	c.pool.New = func() any {
		return new(int64)
	}
	return c, nil
}

// Tokenizer expõe o tokenizer em uso, para diagnóstico.
func (c *BuiltinClient) Tokenizer() *Tokenizer { return c.tok }

// Dimension é a dimensão do vetor devolvido (384, o nativo do modelo).
func (c *BuiltinClient) Dimension() int { return nativeDim }

// GenerateEmbedding devolve o vetor nativo 384d, com mean pooling sobre os
// tokens válidos e normalização L2.
//
// Texto vazio é erro explícito, não vetor zero: um vetor zero entra no índice
// e degrada a busca sem nenhuma pista de por quê.
func (c *BuiltinClient) GenerateEmbedding(text string) ([]float32, error) {
	if c == nil || c.sess == nil {
		return nil, fmt.Errorf("builtin: client não inicializado")
	}
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("builtin: texto vazio não gera embedding")
	}

	tk, err := c.tok.Encode(text)
	if err != nil {
		return nil, err
	}
	seqLen := len(tk.IDs)
	shape := ort.NewShape(1, int64(seqLen))

	idsTensor, err := ort.NewTensor[int64](shape, tk.IDs)
	if err != nil {
		return nil, fmt.Errorf("builtin: falha ao criar input_ids: %w", err)
	}
	defer idsTensor.Destroy()

	maskTensor, err := ort.NewTensor[int64](shape, tk.Mask)
	if err != nil {
		return nil, fmt.Errorf("builtin: falha ao criar attention_mask: %w", err)
	}
	defer maskTensor.Destroy()

	// Frase única: token_type_ids é tudo zero (0 = sentence A).
	types := make([]int64, seqLen)
	typeTensor, err := ort.NewTensor[int64](shape, types)
	if err != nil {
		return nil, fmt.Errorf("builtin: falha ao criar token_type_ids: %w", err)
	}
	defer typeTensor.Destroy()

	outputs := []ort.Value{nil}
	if err := c.sess.Run([]ort.Value{idsTensor, maskTensor, typeTensor}, outputs); err != nil {
		return nil, fmt.Errorf("builtin: inferência falhou: %w", err)
	}
	defer outputs[0].Destroy()

	hidden, ok := outputs[0].(*ort.Tensor[float32])
	if !ok {
		return nil, fmt.Errorf("builtin: saída do modelo não é tensor float32")
	}
	shapeOut := hidden.GetShape()
	// Esperado: [1, seqLen, 384]
	if len(shapeOut) != 3 || shapeOut[2] != nativeDim {
		return nil, fmt.Errorf("builtin: shape de saída inesperado %v (esperado [1, %d, %d])", shapeOut, seqLen, nativeDim)
	}

	data := hidden.GetData()
	// Mean pooling: soma os vetores dos tokens válidos e divide pela máscara.
	vec := make([]float32, nativeDim)
	var valid int64
	for t := 0; t < seqLen; t++ {
		if t < len(tk.Mask) && tk.Mask[t] == 0 {
			continue
		}
		base := t * nativeDim
		if base+nativeDim > len(data) {
			return nil, fmt.Errorf("builtin: tensor de saída truncado (%d floats para %d tokens)", len(data), seqLen)
		}
		for d := 0; d < nativeDim; d++ {
			vec[d] += data[base+d]
		}
		valid++
	}
	if valid == 0 {
		return nil, fmt.Errorf("builtin: attention_mask sem tokens válidos")
	}
	inv := float32(1) / float32(valid)
	for d := range vec {
		vec[d] *= inv
	}

	l2Normalize(vec)
	return vec, nil
}

// Close libera a sessão ONNX.
func (c *BuiltinClient) Close() error {
	if c != nil && c.sess != nil {
		c.sess.Destroy()
		c.sess = nil
	}
	return nil
}

// l2Normalize normaliza o vetor para norma 1. Vetor zero fica intacto.
func l2Normalize(v []float32) {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	norm := float32(math.Sqrt(sum))
	if norm == 0 {
		return
	}
	for i := range v {
		v[i] /= norm
	}
}

// CosineSimilarity é o cosseno entre dois vetores já normalizados.
func CosineSimilarity(a, b []float32) float64 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	var dot, na, nb float64
	for i := 0; i < n; i++ {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
