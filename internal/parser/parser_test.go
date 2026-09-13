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

func TestObsidianMarkdownFeatures(t *testing.T) {
	markdown := `---
title: Projeto Alpha
tags:
  - projeto
  - golang
aliases:
  - Alpha
  - Project Alpha
status: ativo
---

# Introdução

Consulte a [[Arquitetura#Visao Geral|Visão Geral da Arquitetura]] e o [[Database#^c182|Schema]].
Também veja o link local [[#Conclusao]].
E a imagem ![[diagrama.png|500]].

Tags no texto: #dev/backend #golang
`

	conn := ExtractConnections(markdown)

	// Verifica Frontmatter
	if conn.Frontmatter == nil {
		t.Fatalf("Esperava frontmatter extraído")
	}
	if conn.Frontmatter.Title != "Projeto Alpha" {
		t.Errorf("Esperava title 'Projeto Alpha', obteve '%s'", conn.Frontmatter.Title)
	}

	expectedAliases := []string{"Alpha", "Project Alpha"}
	if !reflect.DeepEqual(conn.Aliases, expectedAliases) {
		t.Errorf("Aliases incorretos. Esperado: %v, Obtido: %v", expectedAliases, conn.Aliases)
	}

	// OutgoingLinks deve normalizar targets externos sem âncoras e ignorar [[#Conclusao]]
	expectedOutgoing := []string{"Arquitetura", "Database", "diagrama.png"}
	if !reflect.DeepEqual(conn.OutgoingLinks, expectedOutgoing) {
		t.Errorf("OutgoingLinks incorretos. Esperado: %v, Obtido: %v", expectedOutgoing, conn.OutgoingLinks)
	}

	// Tags devem unir frontmatter e inline sem duplicatas
	expectedTags := []string{"projeto", "golang", "dev/backend"}
	if !reflect.DeepEqual(conn.Tags, expectedTags) {
		t.Errorf("Tags incorretas. Esperado: %v, Obtido: %v", expectedTags, conn.Tags)
	}

	// Verifica detalhes estruturados dos Links
	if len(conn.Links) != 4 {
		t.Fatalf("Esperava 4 links, obteve %d", len(conn.Links))
	}

	// Link 1: Arquitetura com âncora e alias
	l1 := conn.Links[0]
	if l1.Target != "Arquitetura" || l1.Anchor != "Visao Geral" || l1.Alias != "Visão Geral da Arquitetura" {
		t.Errorf("Link 1 incorreto: %+v", l1)
	}

	// Link 2: Database com block reference
	l2 := conn.Links[1]
	if l2.Target != "Database" || l2.BlockID != "c182" || l2.Alias != "Schema" {
		t.Errorf("Link 2 incorreto: %+v", l2)
	}

	// Link 3: Link local mesma nota
	l3 := conn.Links[2]
	if !l3.IsSameDoc || l3.Anchor != "Conclusao" {
		t.Errorf("Link 3 incorreto: %+v", l3)
	}

	// Link 4: Embed
	l4 := conn.Links[3]
	if !l4.IsEmbed || l4.Target != "diagrama.png" || l4.Alias != "500" {
		t.Errorf("Link 4 incorreto: %+v", l4)
	}
}

func TestChunkText(t *testing.T) {
	text := "palavra1 palavra2 palavra3 palavra4 palavra5 palavra6 palavra7 palavra8 palavra9 palavra10"

	chunks := ChunkText(text, 4, 2)

	if len(chunks) == 0 {
		t.Fatalf("Esperava múltiplos chunks, obteve 0")
	}

	expectedFirst := "palavra1 palavra2 palavra3 palavra4"
	if chunks[0] != expectedFirst {
		t.Errorf("Primeiro chunk incorreto. Esperado: %q, Obtido: %q", expectedFirst, chunks[0])
	}

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

func TestExtractConnections_TypedEdges(t *testing.T) {
	markdown := `---
title: Nota com Arestas Epistêmicas
relations:
  supports:
    - TeoriaGeral
  refutes:
    - HipoteseLegada
---

# Desenvolvimento

Este componente [[implements:Arquitetura]] e possui dependência em [[BancoDados|rel:depends_on]].
Também conecta com [[NotaSimples]] e estende [[extends:ModuloBase]].
Tags do documento: #arquitetura #mvp
`

	conn := ExtractConnections(markdown)

	// 1. Verifica OutgoingLinks (retrocompatibilidade)
	expectedOutgoing := []string{"Arquitetura", "BancoDados", "NotaSimples", "ModuloBase", "TeoriaGeral", "HipoteseLegada"}
	for _, exp := range expectedOutgoing {
		found := false
		for _, out := range conn.OutgoingLinks {
			if out == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("OutgoingLinks deve conter %s, obteve %v", exp, conn.OutgoingLinks)
		}
	}

	// 2. Verifica Edges tipadas e epistêmicas
	findEdge := func(target, relation string) *EdgeConnection {
		for _, e := range conn.Edges {
			if e.Target == target && e.Relation == relation {
				return &e
			}
		}
		return nil
	}

	// Verifica prefixo implements
	if e := findEdge("Arquitetura", "implements"); e == nil || e.EpistemicStatus != "EXTRACTED" || e.Weight != 1.0 {
		t.Errorf("Aresta 'implements:Arquitetura' não encontrada ou inválida: %+v", e)
	}

	// Verifica alias rel:depends_on
	if e := findEdge("BancoDados", "depends_on"); e == nil || e.EpistemicStatus != "EXTRACTED" {
		t.Errorf("Aresta 'depends_on:BancoDados' não encontrada: %+v", e)
	}

	// Verifica link padrão links_to
	if e := findEdge("NotaSimples", "links_to"); e == nil || e.EpistemicStatus != "EXTRACTED" {
		t.Errorf("Aresta 'links_to:NotaSimples' não encontrada: %+v", e)
	}

	// Verifica prefixo extends
	if e := findEdge("ModuloBase", "extends"); e == nil {
		t.Errorf("Aresta 'extends:ModuloBase' não encontrada: %+v", e)
	}

	// Verifica tags como tagged_as
	if e := findEdge("arquitetura", "tagged_as"); e == nil {
		t.Errorf("Aresta de tag 'tagged_as:arquitetura' não encontrada: %+v", e)
	}
	if e := findEdge("mvp", "tagged_as"); e == nil {
		t.Errorf("Aresta de tag 'tagged_as:mvp' não encontrada: %+v", e)
	}

	// Verifica frontmatter relations
	if e := findEdge("TeoriaGeral", "supports"); e == nil {
		t.Errorf("Aresta frontmatter 'supports:TeoriaGeral' não encontrada: %+v", e)
	}
	if e := findEdge("HipoteseLegada", "refutes"); e == nil {
		t.Errorf("Aresta frontmatter 'refutes:HipoteseLegada' não encontrada: %+v", e)
	}
}
