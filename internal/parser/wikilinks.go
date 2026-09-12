package parser

import (
	"regexp"
	"strings"
)

var (
	wikilinkRegex = regexp.MustCompile(`\[\[([^\]\|]+)(?:\|[^\]]+)?\]\]`)
	tagRegex      = regexp.MustCompile(`\B#([a-zA-Z0-9_\-\/]+)`)
)

type ExtractedConnections struct {
	OutgoingLinks []string
	Tags          []string
}

// ExtractConnections extrai os [[wikilinks]] e as #tags estilo Obsidian
func ExtractConnections(content string) ExtractedConnections {
	var conn ExtractedConnections

	matches := wikilinkRegex.FindAllStringSubmatch(content, -1)
	for _, m := range matches {
		if len(m) > 1 {
			target := strings.TrimSpace(m[1])
			if target != "" {
				conn.OutgoingLinks = append(conn.OutgoingLinks, target)
			}
		}
	}

	tagMatches := tagRegex.FindAllStringSubmatch(content, -1)
	for _, m := range tagMatches {
		if len(m) > 1 {
			tag := strings.TrimSpace(m[1])
			if tag != "" {
				conn.Tags = append(conn.Tags, tag)
			}
		}
	}

	return conn
}
