package embedder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Os testes de tokenizer exigem o arquivo do modelo. Sem ele no CI, os testes
// pulam com aviso explícito em vez de passar em silêncio.
func modeloPath(t *testing.T) string {
	t.Helper()
	candidates := []string{
		os.Getenv("MY_MEMORY_MINILM_MODEL_DIR"),
		filepath.Join(os.Getenv("USERPROFILE"), ".memory", "models", "paraphrase-multilingual-MiniLM-L12-v2"),
		filepath.Join(os.Getenv("HOME"), ".memory", "models", "paraphrase-multilingual-MiniLM-L12-v2"),
	}
	for _, dir := range candidates {
		if dir == "" {
			continue
		}
		p := filepath.Join(dir, "sentencepiece.bpe.model")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	t.Skip("sentencepiece.bpe.model não encontrado — baixe com `mem embed download` ou defina MY_MEMORY_MINILM_MODEL_DIR")
	return ""
}

func novoTokenizerDeTeste(t *testing.T) *Tokenizer {
	t.Helper()
	tk, err := NewTokenizer(modeloPath(t))
	if err != nil {
		t.Fatalf("NewTokenizer: %v", err)
	}
	return tk
}

func TestTokenizerVocabSize(t *testing.T) {
	tk := novoTokenizerDeTeste(t)
	// A lib fixa 250002 (= 250000 peças + offset fairseq + <mask>), que é o
	// len(tokenizer) do HuggingFace para este modelo.
	if got := tk.VocabSize(); got != 250002 {
		t.Errorf("VocabSize = %d, quer 250002", got)
	}
}

func TestTokenizerEncodeTextoSimples(t *testing.T) {
	tk := novoTokenizerDeTeste(t)
	out, err := tk.Encode("olá mundo")
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if len(out.IDs) == 0 {
		t.Fatal("Encode devolveu zero tokens")
	}
	if len(out.IDs) != len(out.Mask) {
		t.Errorf("ids (%d) e mask (%d) com tamanhos diferentes", len(out.IDs), len(out.Mask))
	}
	// <s> ... </s>
	if len(out.IDs) < 3 {
		t.Errorf("esperava no mínimo 3 tokens (CLS + conteúdo + SEP), veio %d", len(out.IDs))
	}
	t.Logf("olá mundo -> %d tokens: %v", len(out.IDs), tk.Pieces("olá mundo"))
}

// A normalização tem de preservar o texto português com acento e cedilha —
// é o que diferencia um tokenizer usable de um que quebra o idioma do vault.
func TestTokenizerPreservaAcentosECedilha(t *testing.T) {
	tk := novoTokenizerDeTeste(t)
	casos := []string{
		"configuração do sistema",
		"operação bancária",
		"impressão de relatório",
		"código fonte",
		"informação técnica",
	}
	for _, texto := range casos {
		out, err := tk.Encode(texto)
		if err != nil {
			t.Errorf("Encode(%q): %v", texto, err)
			continue
		}
		if len(out.IDs) == 0 {
			t.Errorf("Encode(%q) devolveu zero tokens", texto)
			continue
		}
		// O texto normalizado não pode virar vazio nem perder todos os acentos.
		norm := tk.tk.Normalize(texto)
		if strings.TrimSpace(norm) == "" {
			t.Errorf("normalização de %q ficou vazia", texto)
		}
	}
}

// Determinismo: o mesmo texto tem que virar os mesmos ids sempre. É o que
// garante que o cache SHA-256 do índice não fica inconsistente.
func TestTokenizerDeterminismo(t *testing.T) {
	tk := novoTokenizerDeTeste(t)
	const texto = "o índice de banco de dados acelera a consulta"

	primeira, err := tk.Encode(texto)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	for i := 0; i < 5; i++ {
		outra, err := tk.Encode(texto)
		if err != nil {
			t.Fatalf("Encode rodada %d: %v", i, err)
		}
		if len(primeira.IDs) != len(outra.IDs) {
			t.Fatalf("rodada %d: %d tokens, esperava %d", i, len(outra.IDs), len(primeira.IDs))
		}
		for j := range primeira.IDs {
			if primeira.IDs[j] != outra.IDs[j] {
				t.Fatalf("rodada %d, posição %d: id %d != %d", i, j, outra.IDs[j], primeira.IDs[j])
			}
		}
	}
}

// Texto longo tem que ser truncado no limite do modelo, não estourar.
func TestTokenizerTruncaEm512(t *testing.T) {
	tk := novoTokenizerDeTeste(t)
	longo := strings.Repeat("a phrase about indexing and vector search. ", 200)

	out, err := tk.Encode(longo)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if len(out.IDs) > maxSeqLen {
		t.Errorf("produziu %d tokens, acima do limite de %d", len(out.IDs), maxSeqLen)
	}
	if len(out.IDs) != len(out.Mask) {
		t.Errorf("ids (%d) e mask (%d) com tamanhos diferentes após truncamento", len(out.IDs), len(out.Mask))
	}
	t.Logf("texto de %d runes -> %d tokens", len([]rune(longo)), len(out.IDs))
}

// Caracteres fora do vocabulário não podem fazer o encoder explodir nem
// devolver zero tokens silenciosamente.
func TestTokenizerCaractereDesconhecido(t *testing.T) {
	tk := novoTokenizerDeTeste(t)
	casos := []string{
		"texto com emoji 🚀 no meio",
		"caractere raro: ꧁꧂",
		"Zero Width Space: invisível",
	}
	for _, texto := range casos {
		out, err := tk.Encode(texto)
		if err != nil {
			t.Errorf("Encode(%q): %v", texto, err)
			continue
		}
		if len(out.IDs) == 0 {
			t.Errorf("Encode(%q) devolveu zero tokens", texto)
		}
	}
}

func TestTokenizerTextoVazio(t *testing.T) {
	tk := novoTokenizerDeTeste(t)
	for _, texto := range []string{"", "   ", "\n\t"} {
		out, err := tk.Encode(texto)
		if err != nil {
			t.Errorf("Encode(%q) deveria devolver zero tokens sem erro: %v", texto, err)
		}
		if len(out.IDs) != 0 {
			t.Errorf("Encode(%q) = %d tokens, esperava 0", texto, len(out.IDs))
		}
	}
}

// Erro tem que ser explícito e nomear o arquivo: tokenizer ausente degrada
// todo embedding em silêncio se a mensagem não for clara.
func TestTokenizerModeloAusente(t *testing.T) {
	_, err := NewTokenizer(filepath.Join(t.TempDir(), "nao-existe.model"))
	if err == nil {
		t.Fatal("NewTokenizer deveria falhar com arquivo inexistente")
	}
	if !strings.Contains(err.Error(), "sentencepiece.bpe.model") {
		t.Errorf("erro não nomeia o arquivo esperado: %v", err)
	}
	if !strings.Contains(err.Error(), "mem embed download") {
		t.Errorf("erro não sugere como resolver: %v", err)
	}
}

func TestTokenizerPathVazio(t *testing.T) {
	if _, err := NewTokenizer("  "); err == nil {
		t.Fatal("NewTokenizer deveria falhar com path vazio")
	}
}
