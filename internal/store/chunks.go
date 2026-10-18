package store

import "strings"

// JoinVaultChunks recompõe o texto de um documento a partir dos chunks que o
// indexador gravou.
//
// ChunkText (internal/parser/markdown.go) divide o arquivo em blocos de ~200
// palavras com 30 palavras de sobreposição, então concatenar os chunks direto
// repetiria a sobreposição e entregaria ao usuário um texto com parágrafos
// repetidos. A montagem aqui remove o trecho já escrito.
//
// O overlap é descoberto pelo conteúdo (maior sufixo do acumulado que seja
// prefixo do próximo chunk) em vez de ser fixado em 30 palavras: se um dia o
// chunking mudar de parâmetro, a remontagem continua correta.
//
// Limitação que vem do chunker, não desta função: blocos com mais de ~200
// palavras são gravados com os separadores normalizados para espaço simples.
// Documentos que cabem num chunk só (o caso comum de nota) saem byte a byte.
func JoinVaultChunks(chunks []string) string {
	var b strings.Builder
	previous := ""
	for _, chunk := range chunks {
		if chunk == "" {
			continue
		}
		if b.Len() == 0 {
			b.WriteString(chunk)
			previous = chunk
			continue
		}
		b.WriteString(chunk[longestChunkOverlap(previous, chunk):])
		previous = chunk
	}
	return b.String()
}

// longestChunkOverlap devolve o tamanho da maior sobreposição entre a
// sobreposição possível: o trecho final do chunk anterior que é prefixo do
// chunk atual.
//
// A busca é limitada pelo tamanho do chunk anterior porque uma sobreposição
// maior que isso significaria que o chunk atual inteiro já foi escrito — o que
// o chunker nunca produz. Esse limite também mantém o custo linear em vez de
// quadrático num documento grande sem sobreposição reconhecível.
func longestChunkOverlap(previous, next string) int {
	max := len(previous)
	if len(next) < max {
		max = len(next)
	}
	for k := max; k > 0; k-- {
		if previous[len(previous)-k:] == next[:k] {
			return k
		}
	}
	return 0
}
