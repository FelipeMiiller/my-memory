package parser

import (
	"strings"
)

// ChunkText divide o texto do Markdown em blocos de tamanho aproximado com sobreposição
func ChunkText(content string, chunkSize int, overlap int) []string {
	words := strings.Fields(content)
	if len(words) == 0 {
		return nil
	}

	if len(words) <= chunkSize {
		return []string{content}
	}

	var chunks []string
	step := chunkSize - overlap
	if step <= 0 {
		step = chunkSize / 2
	}

	for i := 0; i < len(words); i += step {
		end := i + chunkSize
		if end > len(words) {
			end = len(words)
		}

		chunk := strings.Join(words[i:end], " ")
		chunks = append(chunks, chunk)

		if end == len(words) {
			break
		}
	}

	return chunks
}
