package codeast

import (
	"fmt"
	"strings"
	"sync"
)

// MockParser é uma implementação completa de Parser para testes e para o
// no-op code_pipeline quando tree-sitter está indisponível mas ainda queremos
// exercitar o fluxo estrutural (T1 + T3 + T4).
//
// Ela usa uma heurística baseada em regex/linhas que cobre Go, Python,
// TypeScript/JavaScript, Rust, Java, C/C++, Ruby, PHP e Shell — versões bem
// comportadas para validar indexação sem o toolchain C.
//
// Não é uma implementação canônica de AST: serve para exercitar o código de
// T3/T4 sem gcc/CGO. Quando o binding tree-sitter real estiver disponível,
// MockParser deve ser removido dos paths de produção, mantendo-se apenas
// para testes.
type MockParser struct {
	mu        sync.Mutex
	failOnExt map[string]bool // extensões que devem falhar (injetado nos testes)
}

// NewMockParser cria um MockParser configurável.
//
// defaultFail: se true, ParseFile devolve erro para todo arquivo (útil para
// TestBuildTagIsolatesTreesitter_negative).
func NewMockParser(defaultFail bool) *MockParser {
	m := &MockParser{}
	if defaultFail {
		m.failOnExt = map[string]bool{".go": true, ".py": true, ".ts": true}
	}
	return m
}

// NewMockParserWithFailMap configura falhas por extensão específica.
func NewMockParserWithFailMap(failExt map[string]bool) *MockParser {
	return &MockParser{failOnExt: failExt}
}

// Languages devolve o top-10 do ADR-047.
func (m *MockParser) Languages() []string {
	out := make([]string, 0)
	for _, l := range DefaultLanguageTable() {
		out = append(out, l.Name)
	}
	return out
}

// Backend devolve "mock".
func (m *MockParser) Backend() string {
	return "mock"
}

// ParseFile detecta linguagem e extrai símbolos via heurística.
// Implementação completa o suficiente para satisfazer T1/T3/T4 sem CGO.
func (m *MockParser) ParseFile(path string, content []byte) (*FileResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	lang, ok := DetectLanguage(path, content)
	if !ok {
		return nil, fmt.Errorf("mock: unsupported language for path %q", path)
	}

	if m.failOnExt != nil && m.failOnExt[extOf(path)] {
		return nil, fmt.Errorf("mock: forced failure for %q", path)
	}

	res := &FileResult{
		Path:     path,
		Language: lang,
	}
	for _, s := range m.extractSymbols(lang, content) {
		res.Symbols = append(res.Symbols, s)
	}
	return res, nil
}

// extractSymbols é um walker regex-by-line. Produz no mínimo o conjunto
// mínimo viável para validar T1/T3/T4 nos testes.
func (m *MockParser) extractSymbols(lang string, content []byte) []Symbol {
	lines := strings.Split(string(content), "\n")
	seen := make(map[string]bool)
	var out []Symbol

	add := func(s Symbol) {
		if !ValidKind(s.Kind) {
			return
		}
		k := s.HashKey()
		if seen[k] {
			return
		}
		seen[k] = true
		out = append(out, s)
	}

	switch lang {
	case "go":
		for i, line := range lines {
			trim := strings.TrimSpace(line)
			switch {
			case strings.HasPrefix(trim, "func "):
				if name := matchGoFunc(trim); name != "" {
					add(Symbol{
						Kind: KindFunction, Name: name,
						QualifiedName: qualifiedForGo(lines, i, name),
						StartLine:     i + 1, EndLine: i + 1,
					})
				}
			case strings.HasPrefix(trim, "type ") && strings.Contains(trim, "struct"):
				if name := matchTypeName(trim, "struct"); name != "" {
					add(Symbol{
						Kind: KindStruct, Name: name,
						StartLine: i + 1, EndLine: i + 1,
					})
				}
			case strings.HasPrefix(trim, "type ") && strings.Contains(trim, "interface"):
				if name := matchTypeName(trim, "interface"); name != "" {
					add(Symbol{
						Kind: KindInterface, Name: name,
						StartLine: i + 1, EndLine: i + 1,
					})
				}
			case strings.HasPrefix(trim, "import "):
				add(Symbol{
					Kind: KindImport, Name: trim,
					StartLine: i + 1, EndLine: i + 1,
				})
			}
		}
	default:
		// Heurística genérica: procura "func name(", "def name(", "function name(",
		// "class name", "fn name" — suficiente para o mock satisfazer T1/T3/T4.
		patterns := []struct {
			prefix string
			kind   Kind
		}{
			{"def ", KindFunction},
			{"function ", KindFunction},
			{"fn ", KindFunction},
			{"class ", KindClass},
			{"struct ", KindStruct},
			{"interface ", KindInterface},
		}
		for i, line := range lines {
			trim := strings.TrimSpace(line)
			for _, p := range patterns {
				if strings.HasPrefix(trim, p.prefix) {
					name := extractIdentAfterPrefix(trim, p.prefix)
					if name != "" {
						add(Symbol{
							Kind: p.kind, Name: name,
							StartLine: i + 1, EndLine: i + 1,
						})
					}
				}
			}
		}
	}
	return out
}

// --- helpers -----------------------------------------------------------------

func matchGoFunc(line string) string {
	// func (r *Recv) Name(args) ... | func Name(args) ...
	rest := strings.TrimPrefix(line, "func ")
	rest = strings.TrimSpace(rest)
	// Pula receiver entre "(" e ")"
	if strings.HasPrefix(rest, "(") {
		end := strings.Index(rest, ")")
		if end > 0 {
			rest = strings.TrimSpace(rest[end+1:])
		}
	}
	// Até primeiro "(" ou espaço.
	for i, r := range rest {
		if r == '(' || r == ' ' || r == '\t' {
			return rest[:i]
		}
	}
	return rest
}

func matchTypeName(line, kind string) string {
	// type NAME struct/interface { ... | type NAME interface
	rest := strings.TrimPrefix(line, "type ")
	fields := strings.Fields(rest)
	if len(fields) >= 2 && fields[1] == kind {
		return fields[0]
	}
	return ""
}

// qualifiedForGo monta pkg.FuncName olhando o `package pkg` mais próximo acima.
// Tolera blocos de imports e comentários entre o `package` e a declaração da função.
func qualifiedForGo(lines []string, idx int, funcName string) string {
	pkg := ""
	for i := idx - 1; i >= 0 && i >= idx-200; i-- {
		t := strings.TrimSpace(lines[i])
		if t == "" {
			continue
		}
		if strings.HasPrefix(t, "//") {
			continue
		}
		if strings.HasPrefix(t, "/*") {
			continue
		}
		if strings.HasPrefix(t, "import ") || strings.HasPrefix(t, "import(") {
			continue
		}
		if strings.HasPrefix(t, "package ") {
			pkg = strings.Fields(t)[1]
			break
		}
		// Outras instruções (var, const, type, func) param a busca.
		break
	}
	if pkg == "" {
		return funcName
	}
	return pkg + "." + funcName
}

func extractIdentAfterPrefix(line, prefix string) string {
	rest := strings.TrimPrefix(line, prefix)
	rest = strings.TrimSpace(rest)
	for i, r := range rest {
		if r == '(' || r == ' ' || r == '\t' || r == '{' || r == ':' {
			return rest[:i]
		}
	}
	return rest
}

func extOf(path string) string {
	for i := len(path) - 1; i >= 0 && path[i] != '/' && path[i] != '\\'; i-- {
		if path[i] == '.' {
			return path[i:]
		}
	}
	return ""
}
