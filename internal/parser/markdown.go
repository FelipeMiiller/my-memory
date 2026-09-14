package parser

import (
	"regexp"
	"strings"
)

var (
	reWikilinkAlias   = regexp.MustCompile(`\[\[(?:[^|\]]+\|)?([^\]]+)\]\]`)
	reMarkdownLink    = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	reMarkdownImage   = regexp.MustCompile(`!\[([^\]]*)\]\([^)]+\)`)
	reBoldItalicStar  = regexp.MustCompile(`\*{1,3}([^*]+)\*{1,3}`)
	reBoldItalicUnder = regexp.MustCompile(`_{1,3}([^_]+)_{1,3}`)
	reStrikethrough   = regexp.MustCompile(`~~([^~]+)~~`)
	reInlineCode      = regexp.MustCompile("`([^`]+)`")
	reHTMLTags        = regexp.MustCompile(`<[^>]+>`)
)

// ExtractMicroAbstract extrai o primeiro parágrafo narrativo descritivo sem marcações markdown (L0)
func ExtractMicroAbstract(body string, maxLen int) string {
	if maxLen <= 0 {
		maxLen = 160
	}

	lines := strings.Split(body, "\n")
	var paragraphLines []string
	inCodeBlock := false

	for _, rawLine := range lines {
		line := strings.TrimRight(rawLine, "\r")
		trimmed := strings.TrimSpace(line)

		// Toggle blocos de código fenced (```)
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			if len(paragraphLines) > 0 {
				clean := CleanMarkdownText(strings.Join(paragraphLines, " "))
				if clean != "" {
					return truncateText(clean, maxLen)
				}
				paragraphLines = nil
			}
			continue
		}
		if inCodeBlock {
			continue
		}

		// Linha em branco
		if trimmed == "" {
			if len(paragraphLines) > 0 {
				clean := CleanMarkdownText(strings.Join(paragraphLines, " "))
				if clean != "" {
					return truncateText(clean, maxLen)
				}
				paragraphLines = nil
			}
			continue
		}

		// Ignora cabeçalhos (# ...), divisórias (---, ***, ___), tabelas (|...|)
		if strings.HasPrefix(trimmed, "#") ||
			trimmed == "---" || trimmed == "***" || trimmed == "___" ||
			(strings.HasPrefix(trimmed, "|") && strings.HasSuffix(trimmed, "|")) {
			if len(paragraphLines) > 0 {
				clean := CleanMarkdownText(strings.Join(paragraphLines, " "))
				if clean != "" {
					return truncateText(clean, maxLen)
				}
				paragraphLines = nil
			}
			continue
		}

		// Remove marcadores de citação ou listas no início da primeira linha
		cleanedLine := trimmed
		if len(paragraphLines) == 0 {
			cleanedLine = strings.TrimLeft(cleanedLine, ">- *+0123456789.")
			cleanedLine = strings.TrimSpace(cleanedLine)
			if cleanedLine == "" {
				continue
			}
		}

		paragraphLines = append(paragraphLines, cleanedLine)
	}

	if len(paragraphLines) > 0 {
		clean := CleanMarkdownText(strings.Join(paragraphLines, " "))
		if clean != "" {
			return truncateText(clean, maxLen)
		}
	}

	return ""
}

// CleanMarkdownText remove tags, wikilinks, links e ênfases de markdown
func CleanMarkdownText(text string) string {
	s := text
	s = reMarkdownImage.ReplaceAllString(s, "$1")
	s = reWikilinkAlias.ReplaceAllStringFunc(s, func(match string) string {
		sub := reWikilinkAlias.FindStringSubmatch(match)
		if len(sub) > 1 {
			target := sub[1]
			if idx := strings.IndexByte(target, '#'); idx != -1 {
				target = target[:idx]
			}
			return strings.TrimSpace(target)
		}
		return match
	})
	s = reMarkdownLink.ReplaceAllString(s, "$1")
	s = reBoldItalicStar.ReplaceAllString(s, "$1")
	s = reBoldItalicStar.ReplaceAllString(s, "$1")
	s = reBoldItalicUnder.ReplaceAllString(s, "$1")
	s = reBoldItalicUnder.ReplaceAllString(s, "$1")
	s = reStrikethrough.ReplaceAllString(s, "$1")
	s = reInlineCode.ReplaceAllString(s, "$1")
	s = reHTMLTags.ReplaceAllString(s, "")

	fields := strings.Fields(s)
	return strings.TrimSpace(strings.Join(fields, " "))
}

func truncateText(text string, maxLen int) string {
	runes := []rune(text)
	if len(runes) <= maxLen {
		return text
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	cutoff := maxLen - 3
	lastSpace := -1
	for i := cutoff; i >= cutoff/2; i-- {
		if runes[i] == ' ' {
			lastSpace = i
			break
		}
	}
	if lastSpace != -1 {
		return strings.TrimRight(string(runes[:lastSpace]), " ,.;:-") + "..."
	}
	return string(runes[:cutoff]) + "..."
}

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

