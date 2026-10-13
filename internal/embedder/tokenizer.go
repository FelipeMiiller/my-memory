package embedder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	sentencepiece "github.com/rostamlabs/rembed/tokenizer/sentencepiece"
)

// maxSeqLen é o limite de tokens do modelo (tokenizer_config.json: model_max_length).
const maxSeqLen = 512

// metaspace é o símbolo U+2581 que o SentencePiece usa como marcador de
// início de palavra.
const metaspace = "▁"

// Tokenizer encapsula o tokenizer SentencePiece Unigram do modelo
// paraphrase-multilingual-MiniLM-L12-v2 (família XLM-R).
//
// A implementação de Viterbi, o normalizador NMT-NFKC com o charsmap
// pré-compilado e o remapeamento de ids para o espaço do fairseq vêm de
// github.com/rostamlabs/rembed — reimplementar isso à mão era o caminho
// óbvio para um normalizer silenciosamente errado, que degradaria todo
// embedding sem produzir erro nenhum.
//
// Ver ADR-035 e .specs/features/embedder-builtin-onnx-minilm/.
type Tokenizer struct {
	tk *sentencepiece.Tokenizer
}

// NewTokenizer carrega o tokenizer a partir do arquivo sentencepiece.bpe.model.
//
// O caminho pode ser absoluto ou relativo a modelsDir. Erros são explícitos e
// nomeiam o arquivo esperado — um tokenizer ausente degrada todo embedding
// silenciosamente se não for tratado aqui.
func NewTokenizer(modelPath string) (*Tokenizer, error) {
	path := strings.TrimSpace(modelPath)
	if path == "" {
		return nil, fmt.Errorf("tokenizer: model_path vazio")
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf(
				"tokenizer: modelo SentencePiece não encontrado em %q — "+
					"baixe sentencepiece.bpe.model com `mem embed download --provider builtin` "+
					"ou aponte embedding.builtin.model_path para o arquivo existente: %w",
				path, err)
		}
		return nil, fmt.Errorf("tokenizer: não foi possível ler o modelo em %q: %w", path, err)
	}

	tk, err := sentencepiece.New(path)
	if err != nil {
		return nil, fmt.Errorf("tokenizer: falha ao carregar %q: %w", path, err)
	}
	return &Tokenizer{tk: tk}, nil
}

// NewTokenizerFromDir carrega o tokenizer a partir do diretório do modelo,
// usando o nome de arquivo padrão dos repos XLM-R.
func NewTokenizerFromDir(dir string) (*Tokenizer, error) {
	return NewTokenizer(filepath.Join(dir, "sentencepiece.bpe.model"))
}

// Tokenized é a saída pronta para inferência: ids com o mapeamento do modelo
// e a máscara de atenção que exclui o padding.
type Tokenized struct {
	IDs  []int64
	Mask []int64
}

// Encode tokeniza o texto respeitando o limite de 512 tokens do modelo.
//
// O texto vazio devolve zero tokens (sem CLS/SEN) porque não há o que
// codificar; a BuiltinClient trata esse caso separadamente.
func (tk *Tokenizer) Encode(text string) (Tokenized, error) {
	if tk == nil || tk.tk == nil {
		return Tokenized{}, fmt.Errorf("tokenizer: não inicializado")
	}
	if strings.TrimSpace(text) == "" {
		return Tokenized{}, nil
	}

	ids, mask := tk.tk.Encode(text, maxSeqLen)
	if len(ids) == 0 {
		return Tokenized{}, fmt.Errorf("tokenizer: texto produziu zero tokens: %q", truncate(text, 80))
	}
	if len(ids) > maxSeqLen {
		return Tokenized{}, fmt.Errorf("tokenizer: %d tokens excede o limite de %d", len(ids), maxSeqLen)
	}
	return Tokenized{IDs: ids, Mask: mask}, nil
}

// Pieces devolve as peças textuais — usado nos testes e no diagnóstico de
// `mem embed doctor`, onde é útil ver como o texto foi segmentado.
func (tk *Tokenizer) Pieces(text string) []string {
	if tk == nil || tk.tk == nil {
		return nil
	}
	return tk.tk.Pieces(text)
}

// VocabSize é o tamanho do vocabulário no espaço de ids do modelo.
func (tk *Tokenizer) VocabSize() int {
	if tk == nil || tk.tk == nil {
		return 0
	}
	return tk.tk.VocabSize()
}

// Close libera o tokenizer. O objeto é seguro para uso concorrente depois de
// New, mas liberá-lo cedo ajuda o GC em processos de vida curta.
func (tk *Tokenizer) Close() error {
	tk.tk = nil
	return nil
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// builtinPool reusa um Tokenizer entre chamadas. O carregamento do modelo
// SentencePiece custa ~100 ms e o warm-up é explicitamente citado no ADR-035
// como mitigação para o cold start de 1-2 s da primeira inferência.
var (
	builtinPoolMu sync.Mutex
	builtinPool   *Tokenizer
)
