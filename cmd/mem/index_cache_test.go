package main

import "testing"

// indexCacheHit é a decisão que evita trabalho. Os casos abaixo são a diferença
// entre "pular com segurança" e "destruir o índice sem querer".

func TestIndexCacheHit_HashIgualEEpochIgual_Pula(t *testing.T) {
	if !indexCacheHit(true, "h1", "h1", "minilm:384:v1", "minilm:384:v1") {
		t.Error("hash e epoch iguais devem pular a reindexação")
	}
}

func TestIndexCacheHit_HashDiferente_NaoPula(t *testing.T) {
	if indexCacheHit(true, "h1", "h2", "e1", "e1") {
		t.Error("hash diferente significa arquivo mudou: precisa reindexar")
	}
}

func TestIndexCacheHit_HashVazio_NuncaPula(t *testing.T) {
	// Documento novo no banco: storedHash vazio não pode ser treated como match.
	if indexCacheHit(true, "", "h1", "", "e1") {
		t.Error("hash vazio significa documento novo: precisa reindexar")
	}
}

// A regressão que o embed_epoch podia ter criado: sem provider, currentEpoch é
// "" e nunca bateria com o epoch gravado por um embedder real. Sem o epochGate,
// TODO documento seria reindexado e seus vetores reais trocados por zeros.
func TestIndexCacheHit_SemProvider_NaoDestroiVetores(t *testing.T) {
	casos := []struct {
		nome        string
		storedEpoch string
	}{
		{"epoch gravado por embedder real", "minilm:384:v1"},
		{"epoch gravado por outro embedder", "outro:768:v2"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if !indexCacheHit(false, "h1", "h1", c.storedEpoch, "") {
				t.Errorf("sem provider o epoch não deve invalidar: sem embedder "+
					"não há como regenerar o vetor, e reindexar trocaria vetores por zeros (stored=%q)", c.storedEpoch)
			}
		})
	}
}

func TestIndexCacheHit_SemProvider_AindaDetectaArquivoAlterado(t *testing.T) {
	if indexCacheHit(false, "h1", "h2", "minilm:384:v1", "") {
		t.Error("sem provider, hash diferente ainda precisa reindexar")
	}
}

func TestIndexCacheHit_ComProvider_InvalidacaoPorTrocaDeEmbedder(t *testing.T) {
	// O caso que o embed_epoch veio resolver: arquivo intacto, embedder trocado.
	if indexCacheHit(true, "h1", "h1", "minilm:384:v1", "outro:768:v2") {
		t.Error("epoch diferente significa embedding de outra geração: precisa reindexar")
	}
}

func TestIndexCacheHit_ComProvider_HerdaLinhaLegadaSemEpoch(t *testing.T) {
	// Linha gravada antes da coluna existir: precisa reindexar uma vez para
	// curar os vetores do embedder antigo.
	if indexCacheHit(true, "h1", "h1", "", "minilm:384:v1") {
		t.Error("linha legada sem epoch deve ser reindexada uma vez")
	}
}
