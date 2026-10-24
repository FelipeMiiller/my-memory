// Package codeast implementa a extração estruturada de símbolos de código-fonte
// (ADR-047 — Tree-sitter multi-linguagem).
//
// O pacote expõe uma interface Parser neutra em relação ao backend concreto.
// A implementação real via github.com/tree-sitter/go-tree-sitter é compilada
// apenas sob a build tag `treesitter` (CGO) — ver treesitter_enabled.go e
// treesitter_disabled.go. Testes e consumidores podem usar a implementação
// Mock via NewMockParser quando o backend CGO não está disponível.
package codeast

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ErrTreesitterDisabled é retornado por backends tree-sitter quando o pacote
// foi compilado sem a tag `treesitter` ou quando o toolchain C (CGO) não está
// disponível. O code_pipeline trata este erro como no-op silencioso (CA-03).
var ErrTreesitterDisabled = errors.New("codeast: tree-sitter backend disabled (build without -tags treesitter or CGO toolchain missing)")

// Kind representa o tipo estrutural de um símbolo extraído da AST.
type Kind string

// Kinds estáveis conforme ADR-047 §Decision Outcome.
const (
	KindFunction  Kind = "function"
	KindMethod    Kind = "method"
	KindClass     Kind = "class"
	KindInterface Kind = "interface"
	KindStruct    Kind = "struct"
	KindImport    Kind = "import"
	KindConstant  Kind = "constant"
	KindVariable  Kind = "variable"
)

// ValidKinds lista todos os Kinds reconhecidos pela tabela code_symbols.
func ValidKinds() []Kind {
	return []Kind{
		KindFunction, KindMethod, KindClass, KindInterface,
		KindStruct, KindImport, KindConstant, KindVariable,
	}
}

// ValidKind retorna true se k é um Kind reconhecido.
func ValidKind(k Kind) bool {
	switch k {
	case KindFunction, KindMethod, KindClass, KindInterface,
		KindStruct, KindImport, KindConstant, KindVariable:
		return true
	}
	return false
}

// Symbol representa um nó estrutural extraído de um arquivo de código.
type Symbol struct {
	Kind          Kind   `json:"kind"`
	Name          string `json:"name"`
	QualifiedName string `json:"qualified_name,omitempty"`
	Signature     string `json:"signature,omitempty"`
	StartLine     int    `json:"start_line"`
	EndLine       int    `json:"end_line"`
	StartCol      int    `json:"start_col,omitempty"`
	EndCol        int    `json:"end_col,omitempty"`
	DocComment    string `json:"doc_comment,omitempty"`
	ParentName    string `json:"parent_name,omitempty"` // nome do símbolo-pai (classe/struct para método)
}

// HashKey devolve uma chave estável para deduplicação: kind + qualified_name + linha.
func (s Symbol) HashKey() string {
	qn := s.QualifiedName
	if qn == "" {
		qn = s.Name
	}
	return fmt.Sprintf("%s|%s|%d", s.Kind, qn, s.StartLine)
}

// EdgeKind é o tipo da relação entre dois símbolos do code graph.
type EdgeKind string

const (
	// EdgeCalls: o símbolo origem CHAMA o símbolo destino. A posição gravada
	// é o PONTO DE CHAMADA (call-site), não a definição.
	EdgeCalls EdgeKind = "calls"
	// EdgeImports: o símbolo origem (a declaração de import) IMPORTA o
	// símbolo destino (o pacote importado).
	EdgeImports EdgeKind = "imports"
	// EdgeInherits: o símbolo origem EMBUTE o destino (embedding de struct ou
	// de interface).
	EdgeInherits EdgeKind = "inherits"
	// EdgeUsesType: o símbolo origem USA o tipo destino (campo, parâmetro,
	// resultado, tipo de variável).
	EdgeUsesType EdgeKind = "uses_type"
	// EdgeReferences: referência resolvida por NOME (heurística legada, sem
	// call-site). Sempre nasce INFERRED — nunca EXTRACTED.
	EdgeReferences EdgeKind = "references"
)

// ValidEdgeKinds lista os kinds de aresta aceitos pelo code graph.
//
// A lista é a fonte de verdade do vocabulary de arestas: `calls`, `imports`,
// `inherits`, `uses_type` e `references` (a heurística legada por nome).
func ValidEdgeKinds() []EdgeKind {
	return []EdgeKind{
		EdgeCalls,
		EdgeImports,
		EdgeInherits,
		EdgeUsesType,
		EdgeReferences,
	}
}

// ValidEdgeKind devolve true se k é um kind de aresta reconhecido.
func ValidEdgeKind(k EdgeKind) bool {
	switch k {
	case EdgeCalls, EdgeImports, EdgeInherits, EdgeUsesType, EdgeReferences:
		return true
	}
	return false
}

// Edge é uma relação estrutural extraída pelo backend.
//
// FORMA ESCOLHIDA (contrato aditivo do Parser — decision 2 da Fase 2):
//
//	Kind             o tipo da relação (calls/imports/inherits/uses_type/references)
//	SrcQualifiedName o qualified name do símbolo ORIGEM (o símbolo ENCLOSING:
//	                 a função/método que contém a chamada, ou a declaração de
//	                 import no caso de `imports`)
//	DstName          o nome simples do destino
//	DstQualifiedName o qualified name do destino quando o backend consegue
//	                 inferi-lo ("pkg.Nome"); vazio quando só o nome é known
//	                 (ex.: chamada de método — o tipo do receptor exige
//	                 type-checking, fora do escopo do backend AST)
//	Line/Column      a posição do CALL-SITE: a linha e a coluna da EXPRESSÃO
//	                 que cria a relação (não a da definição do símbolo)
//	Reason           a razão do caso INCERTO (destino ambíguo ou não
//	                 resolvido); vazia para arestas resolvidas
//
// A aresta é resolvida em três níveis (decision 5 da Fase 2):
// qualified name bate dentro do pacote → EXTRACTED; só o nome bate (único no
// repositório) → INFERRED; mais de um candidato ou nada → vai para
// code_edges_uncertain com confidence + reason e NUNCA vira aresta resolvida.
type Edge struct {
	Kind             EdgeKind `json:"kind"`
	SrcQualifiedName string   `json:"src_qualified_name"`
	DstName          string   `json:"dst_name"`
	DstQualifiedName string   `json:"dst_qualified_name,omitempty"`
	Line             int      `json:"line"`
	Column           int      `json:"column,omitempty"`
	Reason           string   `json:"reason,omitempty"`
}

// FileResult é o resultado de ParseFile: metadados do arquivo + símbolos extraídos.
type FileResult struct {
	Path     string   `json:"path"`
	Language string   `json:"language"`
	Symbols  []Symbol `json:"symbols"`
	// Edges são as relações estruturais extraídas pelo backend. O campo é
	// ADITIVO: backends que não devolvem arestas (linguagens sem suporte
	// estrutural) simplesmente o deixam vazio — o pipeline mantém o
	// comportamento legado (arestas `references` por nome) nesse caso.
	Edges []Edge `json:"edges,omitempty"`
	// Errors são problemas não-fatais detectados durante o parse (não retornam erro).
	Errors []string `json:"errors,omitempty"`
}

// Parser é o contrato neutro de backend para extração de símbolos de código.
//
// Toda implementação (tree-sitter real, mock para testes) deve satisfazer esta
// interface. Consumidores devem usar NewParser() para obter a implementação
// adequada ao build em execução (ver treesitter_enabled.go / treesitter_disabled.go).
type Parser interface {
	// ParseFile extrai os símbolos estruturais de um único arquivo.
	// O parâmetro content é o conteúdo bruto (UTF-8) do arquivo; path é
	// usado somente para detecção de linguagem e logging.
	ParseFile(path string, content []byte) (*FileResult, error)

	// Languages devolve a lista de linguagens suportadas pelo backend (lower-case).
	Languages() []string

	// Backend devolve o identificador do backend ("treesitter", "mock", ...).
	Backend() string
}

// HashContent devolve o SHA-256 hex de content (mesmo formato ADR-010).
func HashContent(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// HashSymbols devolve um SHA-256 estável sobre o conjunto de símbolos em symbols.
//
// A ordem é determinística: símbolos são ordenados por HashKey() antes de
// serializar para "{kind}|{qualified_name}|{start_line}". Símbolos com mesmo
// HashKey são colapsados.
func HashSymbols(symbols []Symbol) string {
	if len(symbols) == 0 {
		return ""
	}
	seen := make(map[string]bool)
	keys := make([]string, 0, len(symbols))
	for _, s := range symbols {
		k := s.HashKey()
		if seen[k] {
			continue
		}
		seen[k] = true
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write([]byte{'\n'})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// IsCamelOrPascalSegment retorna true se s contém letras maiúsculas/minúsculas
// (padrão camelCase ou PascalCase). Útil para o detector de qualified_name em T5,
// exposto aqui porque é lógica estrutural compartilhada.
func IsCamelOrPascalSegment(s string) bool {
	if s == "" {
		return false
	}
	hasUpper, hasLower := false, false
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			hasLower = true
		}
	}
	return hasUpper && hasLower
}

// isIdentifierSegment devolve true se s bate com `[A-Za-z_][A-Za-z0-9_]*`,
// o formato canônico de identificador Go/JS/Python/etc (sem espaços).
func isIdentifierSegment(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		switch {
		case r == '_':
			// OK
		case r >= 'A' && r <= 'Z':
			// OK
		case r >= 'a' && r <= 'z':
			// OK
		case r >= '0' && r <= '9' && i > 0:
			// OK (não no primeiro char)
		default:
			return false
		}
	}
	return true
}

// HasQualifiedNameShape devolve true se query parece um qualified_name
// (contém '.' e cada segmento é um identificador válido tipo `pkg`,
// `Pkg`, `PkgName` ou `pkg_name` — sem espaços).
// Detalhes em T5 (CA-05).
func HasQualifiedNameShape(query string) bool {
	query = strings.TrimSpace(query)
	if query == "" || !strings.Contains(query, ".") {
		return false
	}
	for _, seg := range strings.Split(query, ".") {
		seg = strings.TrimSpace(seg)
		if !isIdentifierSegment(seg) {
			return false
		}
	}
	return true
}

// SupportedExtensions devolve o conjunto de extensões reconhecidas pelo backend
// de produção (tree-sitter). É a união canônica de LanguagesToExtension.
func SupportedExtensions() []string {
	langs := DefaultLanguageTable()
	seen := make(map[string]bool)
	out := make([]string, 0)
	for _, l := range langs {
		for _, ext := range l.Extensions {
			ext = strings.TrimPrefix(ext, ".")
			if ext != "" && !seen[ext] {
				seen[ext] = true
				out = append(out, ext)
			}
		}
	}
	sort.Strings(out)
	return out
}
