//go:build onnx

package embedder

import (
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func modelosDir(t *testing.T) string {
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
		if _, err := os.Stat(filepath.Join(dir, "model_int8.onnx")); err == nil {
			return dir
		}
	}
	t.Skip("model_int8.onnx não encontrado — baixe com `mem embed download` ou defina MY_MEMORY_MINILM_MODEL_DIR")
	return ""
}

func ortLib(t *testing.T) string {
	t.Helper()
	candidates := []string{
		os.Getenv("MY_MEMORY_ORT_LIB"),
		filepath.Join(os.Getenv("USERPROFILE"), ".memory", "models", "onnxruntime", "onnxruntime.dll"),
	}
	for _, p := range candidates {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	t.Skip("onnxruntime.dll não encontrado — defina MY_MEMORY_ORT_LIB")
	return ""
}

func novoBuiltinDeTeste(t *testing.T) *BuiltinClient {
	t.Helper()
	dir := modelosDir(t)
	c, err := NewBuiltinClient(BuiltinConfig{
		ModelPath:    filepath.Join(dir, "model_int8.onnx"),
		TokenizerDir: dir,
		ORTLibPath:   ortLib(t),
	})
	if err != nil {
		t.Fatalf("NewBuiltinClient: %v", err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

func TestBuiltinGera384Dim(t *testing.T) {
	c := novoBuiltinDeTeste(t)
	vec, err := c.GenerateEmbedding("o índice do banco de dados acelera consultas repetidas")
	if err != nil {
		t.Fatalf("GenerateEmbedding: %v", err)
	}
	if len(vec) != nativeDim {
		t.Fatalf("vetor com %d dimensões, esperava %d (nativo)", len(vec), nativeDim)
	}
	if c.Dimension() != 384 {
		t.Errorf("Dimension() = %d, quer 384", c.Dimension())
	}
}

func TestBuiltinVetorNormalizado(t *testing.T) {
	c := novoBuiltinDeTeste(t)
	vec, err := c.GenerateEmbedding("balanceamento de carga em servidores distribuídos")
	if err != nil {
		t.Fatalf("GenerateEmbedding: %v", err)
	}
	var soma float64
	for _, v := range vec {
		soma += float64(v) * float64(v)
	}
	norma := math.Sqrt(soma)
	if math.Abs(norma-1.0) > 1e-4 {
		t.Errorf("norma L2 = %.6f, quer 1.0", norma)
	}
}

func TestBuiltinDeterminismo(t *testing.T) {
	c := novoBuiltinDeTeste(t)
	const texto = "configuração do cache da aplicação"

	primeiro, err := c.GenerateEmbedding(texto)
	if err != nil {
		t.Fatalf("GenerateEmbedding: %v", err)
	}
	for i := 0; i < 3; i++ {
		outro, err := c.GenerateEmbedding(texto)
		if err != nil {
			t.Fatalf("rodada %d: %v", i, err)
		}
		for j := range primeiro {
			if primeiro[j] != outro[j] {
				t.Fatalf("rodada %d posição %d: %.9f != %.9f (não determinístico)", i, j, outro[j], primeiro[j])
			}
		}
	}
}

// O provider embutido não pode tocar a rede: a promessa do ADR-035 é
// inferência 100% in-process. Substituímos o transport HTTP global por um que
// falha — qualquer tentativa de rede estoura aqui em vez de passar em silêncio.
type transportQueFalha struct{}

func (transportQueFalha) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("rede bloqueada pelo teste: o provider embutido nao pode fazer requisicao HTTP")
}

func TestBuiltinSemRede(t *testing.T) {
	c := novoBuiltinDeTeste(t)

	orig := http.DefaultTransport
	http.DefaultTransport = transportQueFalha{}
	defer func() { http.DefaultTransport = orig }()

	if _, err := c.GenerateEmbedding("teste de inferência local sem rede"); err != nil {
		t.Fatalf("GenerateEmbedding falhou com o transporte HTTP bloqueado: %v", err)
	}
}

func TestBuiltinTextoVazioERrro(t *testing.T) {
	c := novoBuiltinDeTeste(t)
	for _, texto := range []string{"", "   "} {
		vec, err := c.GenerateEmbedding(texto)
		if err == nil {
			t.Errorf("GenerateEmbedding(%q) deveria falhar, devolveu vetor de %d floats", texto, len(vec))
		}
		if vec != nil {
			t.Errorf("GenerateEmbedding(%q) devolveu vetor junto com erro", texto)
		}
	}
}

// Vetor zero é o pior resultado possível: entra no índice e degrada a busca
// sem deixar pista. Textos diferentes têm que produzir vetores diferentes.
func TestBuiltinVetoresDiferentes(t *testing.T) {
	c := novoBuiltinDeTeste(t)
	a, err := c.GenerateEmbedding("uma receita de bolo de chocolate")
	if err != nil {
		t.Fatalf("a: %v", err)
	}
	b, err := c.GenerateEmbedding("a arquitetura das catedrais medievais")
	if err != nil {
		t.Fatalf("b: %v", err)
	}
	if sim := CosineSimilarity(a, b); sim > 0.99 {
		t.Errorf("textos sem relação deram similaridade %.4f", sim)
	}
	for i := range a {
		if a[i] != 0 {
			return
		}
	}
	t.Error("vetor de texto real saiu totalmente zerado")
}

func TestBuiltinModeloAusenteErroNomeado(t *testing.T) {
	_, err := NewBuiltinClient(BuiltinConfig{
		ModelPath:    filepath.Join(t.TempDir(), "nao-existe.onnx"),
		TokenizerDir: modelosDir(t),
		ORTLibPath:   ortLib(t),
	})
	if err == nil {
		t.Fatal("NewBuiltinClient deveria falhar com modelo ausente")
	}
	if !strings.Contains(err.Error(), "mem embed download") {
		t.Errorf("erro não sugere como resolver: %v", err)
	}
}

func TestBuiltinOrtLibAusenteErroNomeado(t *testing.T) {
	// O ambiente do ORT é inicializado uma vez por processo (sync.Once), então
	// este teste só é meaningfully o primeiro quando roda isolado. Aceita
	// qualquer um dos dois desfechos e exige que, havendo erro, ele nomeie o
	// caminho da biblioteca.
	_, err := NewBuiltinClient(BuiltinConfig{
		ModelPath:    filepath.Join(modelosDir(t), "model_int8.onnx"),
		TokenizerDir: modelosDir(t),
		ORTLibPath:   filepath.Join(t.TempDir(), "onnxruntime.dll"),
	})
	if err != nil {
		if !strings.Contains(err.Error(), "onnxruntime") {
			t.Errorf("erro não nomeia a biblioteca: %v", err)
		}
	}
}
