package embedder

import (
	"strconv"
	"strings"
	"testing"
)

// O Epoch é o que impede o cache por content_hash de mentir quando o embedder
// muda. Estes testes travam o contrato do qual o gate depende: mesma string
// quando nada muda, string diferente quando algo muda.

func TestEpoch_EstavelECompostoPorModeloEDimensao(t *testing.T) {
	got := Epoch()
	if got != Epoch() {
		t.Error("Epoch() deve ser determinístico entre chamadas")
	}
	if !strings.HasPrefix(got, ModelID+":") {
		t.Errorf("Epoch deve começar com o ModelID %q; got %q", ModelID, got)
	}
	if !strings.Contains(got, ":"+strconv.Itoa(EmbeddingDim)+":") {
		t.Errorf("Epoch deve conter a dimensão %d; got %q", EmbeddingDim, got)
	}
}

func TestEpoch_TrocaDeModeloDimensaoOuReceitaNaoColide(t *testing.T) {
	dim := strconv.Itoa(EmbeddingDim)
	base := Epoch()

	for _, cenario := range []struct {
		nome   string
		epoch  string
		explica string
	}{
		{"dimensão diferente", ModelID + ":768:" + vectorRecipe, "EmbeddingDim mudou"},
		{"modelo diferente", "outro-modelo:" + dim + ":" + vectorRecipe, "ModelID mudou"},
		{"receita diferente", ModelID + ":" + dim + ":" + vectorRecipe + "-x", "vectorRecipe mudou"},
	} {
		if cenario.epoch == base {
			t.Errorf("Epoch com %s colide com o corrente: %s", cenario.nome, cenario.explica)
		}
	}
}

func TestEpochFor_SemProviderDevolveVazio(t *testing.T) {
	// Sem provider a indexação cai para FTS: nada vetorial foi gravado, então o
	// gate precisa comparar "" com "" e não invalidar a base a cada `mem index`.
	if got := EpochFor(nil); got != "" {
		t.Errorf("EpochFor(nil) deve devolver string vazia; got %q", got)
	}
}

func TestEpochFor_ComProviderDevolveEpochCorrente(t *testing.T) {
	if got := EpochFor(&OllamaClient{}); got != Epoch() {
		t.Errorf("EpochFor(provider) deve devolver Epoch(); got %q quer %q", got, Epoch())
	}
}