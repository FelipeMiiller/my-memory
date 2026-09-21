package codeast

import (
	"path/filepath"
	"strings"
)

// LanguageDef descreve uma linguagem suportada pelo code_pipeline (ADR-047).
type LanguageDef struct {
	Name        string   `json:"name"`         // Lower-case (canonical): "go", "python", ...
	DisplayName string   `json:"display_name"` // Nome amigável ("Go", "Python")
	Extensions  []string `json:"extensions"`   // Extensões com ponto (".go", ".py")
	Shebangs    []string `json:"shebangs"`     // Sinais shebang na primeira linha (sem '#!')
}

// DefaultLanguageTable cobre o top-10 de linguagens do ADR-047 §Decision Outcome.
// Cobertura ≥90% dos repos reais; demais via --lang=all (download on-demand).
func DefaultLanguageTable() []LanguageDef {
	return []LanguageDef{
		{
			Name:        "go",
			DisplayName: "Go",
			Extensions:  []string{".go"},
		},
		{
			Name:        "python",
			DisplayName: "Python",
			Extensions:  []string{".py", ".pyi"},
			Shebangs:    []string{"python", "python3", "/usr/bin/env python", "/usr/bin/python", "/usr/bin/python3"},
		},
		{
			Name:        "typescript",
			DisplayName: "TypeScript",
			Extensions:  []string{".ts", ".tsx"},
		},
		{
			Name:        "javascript",
			DisplayName: "JavaScript",
			Extensions:  []string{".js", ".jsx", ".mjs", ".cjs"},
		},
		{
			Name:        "rust",
			DisplayName: "Rust",
			Extensions:  []string{".rs"},
		},
		{
			Name:        "java",
			DisplayName: "Java",
			Extensions:  []string{".java"},
		},
		{
			Name:        "cpp",
			DisplayName: "C/C++",
			Extensions:  []string{".c", ".h", ".cpp", ".hpp", ".cc", ".hh", ".cxx"},
		},
		{
			Name:        "ruby",
			DisplayName: "Ruby",
			Extensions:  []string{".rb"},
			Shebangs:    []string{"ruby", "/usr/bin/ruby", "/usr/local/bin/ruby"},
		},
		{
			Name:        "php",
			DisplayName: "PHP",
			Extensions:  []string{".php"},
		},
		{
			Name:        "shell",
			DisplayName: "Shell",
			Extensions:  []string{".sh", ".bash"},
			Shebangs:    []string{"sh", "bash", "/bin/sh", "/bin/bash", "/usr/bin/env bash", "/usr/bin/env sh", "/usr/bin/bash"},
		},
	}
}

// languageByName indexa a tabela por nome canônico (lower-case).
func languageByName(name string) *LanguageDef {
	for i, l := range DefaultLanguageTable() {
		if l.Name == name {
			return &DefaultLanguageTable()[i]
		}
	}
	return nil
}

// languageByExt indexa a tabela por extensão (case-insensitive).
func languageByExt(ext string) *LanguageDef {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	if ext == "" {
		return nil
	}
	for i, l := range DefaultLanguageTable() {
		for _, e := range l.Extensions {
			if strings.EqualFold(strings.TrimPrefix(e, "."), ext) {
				return &DefaultLanguageTable()[i]
			}
		}
	}
	return nil
}

// DetectLanguage identifica a linguagem de um arquivo a partir do caminho
// (extensão) e, se necessário, do shebang presente na primeira linha de content.
//
// Retorna ("", false) quando nenhuma heurística casa — chamador deve
// descartar o arquivo (não é erro fatal).
func DetectLanguage(path string, content []byte) (string, bool) {
	if ext := filepath.Ext(path); ext != "" {
		if lang := languageByExt(ext); lang != nil {
			return lang.Name, true
		}
	}
	if len(content) > 0 && (content[0] == '#') {
		// Lê apenas a primeira linha (shebang); bounded por 512 bytes.
		limit := 512
		if len(content) < limit {
			limit = len(content)
		}
		firstLine := string(content[:limit])
		if nl := strings.IndexByte(firstLine, '\n'); nl >= 0 {
			firstLine = firstLine[:nl]
		}
		// shebang formato: "#!..." opcional com argumentos
		if strings.HasPrefix(firstLine, "#!") {
			prog := strings.TrimSpace(strings.TrimPrefix(firstLine, "#!"))
			prog = strings.TrimPrefix(prog, "/usr/bin/env ")
			prog = strings.TrimPrefix(prog, "/usr/local/bin/")
			prog = strings.TrimPrefix(prog, "/bin/")
			prog = strings.TrimSpace(prog)
			// Também aceita "/usr/bin/python" → base "python"
			baseProg := prog
			if idx := strings.LastIndex(prog, "/"); idx >= 0 {
				baseProg = prog[idx+1:]
			}
			for _, lang := range DefaultLanguageTable() {
				for _, s := range lang.Shebangs {
					// Cadê `s` é "/usr/bin/env python" — comparar contra
					// prog já com o env prefix removido OU pelo base name.
					sProg := strings.TrimPrefix(s, "/usr/bin/env ")
					sProg = strings.TrimPrefix(sProg, "/usr/local/bin/")
					sProg = strings.TrimPrefix(sProg, "/bin/")
					sProg = strings.TrimSpace(sProg)
					if sProg == prog || sProg == baseProg {
						return lang.Name, true
					}
				}
			}
		}
	}
	return "", false
}

// IsSupportedLanguage devolve true se lang está entre as top-10 suportadas.
func IsSupportedLanguage(lang string) bool {
	return languageByName(strings.ToLower(lang)) != nil
}

// LanguageSupportedExtensions retorna extensões canônicas (com ".") de lang.
func LanguageSupportedExtensions(lang string) []string {
	l := languageByName(strings.ToLower(lang))
	if l == nil {
		return nil
	}
	out := make([]string, len(l.Extensions))
	copy(out, l.Extensions)
	return out
}

// FilterLanguages retorna os subsets das definições DefaultLanguageTable
// correspondentes aos nomes canônicos fornecidos. Útil para `--lang=csv` (CA-06).
// Nomes desconhecidos são descartados silenciosamente.
func FilterLanguages(names []string) []LanguageDef {
	seen := make(map[string]bool)
	var out []LanguageDef
	for _, n := range names {
		n = strings.ToLower(strings.TrimSpace(n))
		if n == "" || seen[n] {
			continue
		}
		if l := languageByName(n); l != nil {
			seen[n] = true
			out = append(out, *l)
		}
	}
	return out
}
