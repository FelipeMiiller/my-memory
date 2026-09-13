package parser

import (
	"reflect"
	"testing"
)

func TestExtractConnections(t *testing.T) {
	content := `
# Arquitetura do Sistema

Este documento descreve como o módulo [[Autenticacao]] se conecta com [[Banco de Dados|PostgreSQL]].
Também fazemos referência à nota [[Cache Redis]].

Tags associadas: #backend #arquitetura #v1/release

Não deve pegar hashtags coladas em textoabc#teste.
`

	conn := ExtractConnections(content)

	expectedLinks := []string{"Autenticacao", "Banco de Dados", "Cache Redis"}
	if !reflect.DeepEqual(conn.OutgoingLinks, expectedLinks) {
		t.Errorf("Links incorretos. Esperado: %v, Obtido: %v", expectedLinks, conn.OutgoingLinks)
	}

	expectedTags := []string{"backend", "arquitetura", "v1/release"}
	if !reflect.DeepEqual(conn.Tags, expectedTags) {
		t.Errorf("Tags incorretas. Esperado: %v, Obtido: %v", expectedTags, conn.Tags)
	}
}

func TestChunkText(t *testing.T) {
	text := "palavra1 palavra2 palavra3 palavra4 palavra5 palavra6 palavra7 palavra8 palavra9 palavra10"

	// Chunk size 4, overlap 2
	chunks := ChunkText(text, 4, 2)

	if len(chunks) == 0 {
		t.Fatalf("Esperava múltiplos chunks, obteve 0")
	}

	// Primeiro chunk deve ter 4 palavras
	expectedFirst := "palavra1 palavra2 palavra3 palavra4"
	if chunks[0] != expectedFirst {
		t.Errorf("Primeiro chunk incorreto. Esperado: %q, Obtido: %q", expectedFirst, chunks[0])
	}

	// Segundo chunk com overlap de 2 palavras (palavra3 palavra4 palavra5 palavra6)
	expectedSecond := "palavra3 palavra4 palavra5 palavra6"
	if chunks[1] != expectedSecond {
		t.Errorf("Segundo chunk incorreto. Esperado: %q, Obtido: %q", expectedSecond, chunks[1])
	}
}

func TestChunkTextShort(t *testing.T) {
	text := "apenas tres palavras"
	chunks := ChunkText(text, 10, 2)

	if len(chunks) != 1 {
		t.Fatalf("Esperava 1 chunk para texto curto, obteve %d", len(chunks))
	}
	if chunks[0] != text {
		t.Errorf("Esperava %q, obteve %q", text, chunks[0])
	}
}
