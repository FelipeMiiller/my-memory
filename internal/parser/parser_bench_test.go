package parser

import (
	"strings"
	"testing"
)

const sampleMarkdown = `---
title: "Documento de Arquitetura e Engenharia"
tags:
  - architecture
  - backend
  - go
relations:
  - implements: CoreStore
  - depends_on: Database
---

# Visão Geral do Sistema

Este módulo documenta o funcionamento do [[implements:StorageEngine]] e suas conexões com o [[Database|rel:depends_on]].
Também fazemos referência cruzada para a [[Segurança]] e analisamos o impacto em [[CacheManager]].

## Detalhes Técnicos

A infraestrutura utiliza #microservices e protocolos de comunicação orientados a eventos.
Conforme discutido em [[supports:Observability]], o pipeline de telemetria deve capturar métricas.
Para mais informações sobre o roadmap, consulte [[Roadmap2026]].

### Recomendações

1. Avaliar os links entre [[cites:RFC9000]] e os clientes HTTP/3.
2. Garantir que as tags como #performance e #reliability sejam indexadas.
`

func BenchmarkExtractConnections_ComplexMarkdown(b *testing.B) {
	doc := strings.Repeat(sampleMarkdown, 5) // ~3.5 KB
	b.SetBytes(int64(len(doc)))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ExtractConnections(doc)
	}
}

func BenchmarkChunkText(b *testing.B) {
	doc := strings.Repeat(sampleMarkdown, 10) // ~7 KB
	b.SetBytes(int64(len(doc)))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ChunkText(doc, 200, 30)
	}
}
