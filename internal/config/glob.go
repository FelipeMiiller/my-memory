package config

import (
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// DefaultExcludedDirs lista diretórios de sistema que sempre são ignorados por padrão
var DefaultExcludedDirs = []string{
	".git",
	"node_modules",
	"vendor",
	".obsidian",
	".trash",
	".memory",
}

var (
	regexCache   = make(map[string]*regexp.Regexp)
	regexCacheMu sync.RWMutex
)

// GlobToRegex converte um padrão glob com suporte a '*' e '**' em uma expressão regular compilada
func GlobToRegex(pattern string) (*regexp.Regexp, error) {
	regexCacheMu.RLock()
	if re, ok := regexCache[pattern]; ok {
		regexCacheMu.RUnlock()
		return re, nil
	}
	regexCacheMu.RUnlock()

	norm := filepath.ToSlash(pattern)
	norm = strings.TrimPrefix(norm, "./")

	var sb strings.Builder
	sb.WriteString("^")

	hasLeadingSlash := strings.HasPrefix(norm, "/")
	norm = strings.TrimPrefix(norm, "/")

	// Se não tiver barra no início e não começar com "**", pode casar na raiz ou em qualquer subdiretório
	if !hasLeadingSlash && !strings.HasPrefix(norm, "**/") {
		sb.WriteString("(?:.*/)?")
	}

	i := 0
	n := len(norm)
	for i < n {
		c := norm[i]
		if c == '*' {
			if i+1 < n && norm[i+1] == '*' {
				// "**"
				i += 2
				if i < n && norm[i] == '/' {
					i++
					sb.WriteString("(?:.*/)?")
				} else {
					sb.WriteString(".*")
				}
			} else {
				// "*"
				i++
				sb.WriteString("[^/]*")
			}
		} else if c == '?' {
			i++
			sb.WriteString("[^/]")
		} else if strings.ContainsRune(`.+()[]{}^$|\`, rune(c)) {
			sb.WriteByte('\\')
			sb.WriteByte(c)
			i++
		} else {
			sb.WriteByte(c)
			i++
		}
	}
	sb.WriteString("$")

	re, err := regexp.Compile(sb.String())
	if err != nil {
		return nil, err
	}

	regexCacheMu.Lock()
	regexCache[pattern] = re
	regexCacheMu.Unlock()

	return re, nil
}

// MatchGlob testa se um caminho relativo atende a um determinado padrão glob
func MatchGlob(pattern, relPath string) bool {
	normPath := filepath.ToSlash(relPath)
	normPath = strings.TrimPrefix(normPath, "./")
	normPath = strings.Trim(normPath, "/")

	// Tratamento especial para patterns terminando em /** (ex: .git/**, node_modules/**)
	cleanPattern := filepath.ToSlash(pattern)
	cleanPattern = strings.TrimPrefix(cleanPattern, "./")
	if strings.HasSuffix(cleanPattern, "/**") {
		baseDir := strings.TrimSuffix(cleanPattern, "/**")
		baseDir = strings.TrimPrefix(baseDir, "/")
		// Casa o próprio diretório ou qualquer filho
		if normPath == baseDir || strings.HasPrefix(normPath, baseDir+"/") {
			return true
		}
		// Também casa se o diretório base for um segmento intermediário em qualquer profundidade
		if strings.Contains(normPath, "/"+baseDir+"/") || strings.HasSuffix(normPath, "/"+baseDir) {
			return true
		}
	}

	re, err := GlobToRegex(pattern)
	if err != nil {
		return false
	}

	return re.MatchString(normPath)
}

// ShouldIndex avalia se um caminho relativo deve ser indexado de acordo com as regras de Exclude e Include
func (c *Config) ShouldIndex(relPath string) bool {
	if c == nil {
		def := DefaultConfig()
		c = &def
	}

	normPath := filepath.ToSlash(relPath)
	normPath = strings.TrimPrefix(normPath, "./")
	normPath = strings.Trim(normPath, "/")

	if normPath == "" || normPath == "." {
		return false
	}

	// 1. Checagem rápida de diretórios de sistema padrão por segmentos do caminho
	segments := strings.Split(normPath, "/")
	for _, seg := range segments {
		for _, defEx := range DefaultExcludedDirs {
			if seg == defEx {
				return false
			}
		}
	}

	// 2. Avaliação de Excludes configurados
	for _, pattern := range c.Exclude {
		if MatchGlob(pattern, normPath) {
			return false
		}
	}

	// 3. Avaliação de Includes configurados
	if len(c.Include) == 0 {
		return true
	}

	for _, pattern := range c.Include {
		if MatchGlob(pattern, normPath) {
			return true
		}
	}

	return false
}
