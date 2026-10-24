package codeast

import (
	"strings"
	"testing"
)

// goSourceFixture é o arquivo Go usado pelos testes do backend (T7).
const goSourceFixture = `package example

import "fmt"

// Config configura o exemplo.
type Config struct {
	Lang string
}

// Runner executa algo.
type Runner interface {
	Run() error
}

// MaxItems é o limite.
const MaxItems = 10

// defaultLang é o idioma padrão.
var defaultLang = "pt-BR"

// Hello greets the world.
func Hello(name string) error {
	return fmt.Println(name)
}

// Run executa o runner.
func (c *Config) Run() error {
	return nil
}
`

func findSymbol(syms []Symbol, kind Kind, name string) (Symbol, bool) {
	for _, s := range syms {
		if s.Kind == kind && s.Name == name {
			return s, true
		}
	}
	return Symbol{}, false
}

// TestGoParserExtraiSimbolosComPosicao cobre T7: o backend Go extrai símbolos
// com kind, nome, qualified name, assinatura, posição (linha/coluna) e doc
// comment corretos.
func TestGoParserExtraiSimbolosComPosicao(t *testing.T) {
	p := NewGoParser()
	if p.Backend() != "go-stdlib" {
		t.Fatalf("Backend() = %q; want go-stdlib", p.Backend())
	}
	if langs := p.Languages(); len(langs) != 1 || langs[0] != "go" {
		t.Fatalf("Languages() = %v; want [go]", langs)
	}

	res, err := p.ParseFile("internal/example/hello.go", []byte(goSourceFixture))
	if err != nil {
		t.Fatalf("ParseFile falhou: %v", err)
	}
	if res.Language != "go" {
		t.Fatalf("Language = %q; want go", res.Language)
	}

	// Função: qualified name, assinatura, posição e doc comment.
	hello, ok := findSymbol(res.Symbols, KindFunction, "Hello")
	if !ok {
		t.Fatal("esperava a função Hello")
	}
	if hello.QualifiedName != "example.Hello" {
		t.Errorf("Hello.QualifiedName = %q; want example.Hello", hello.QualifiedName)
	}
	if !strings.Contains(hello.Signature, "func Hello(name string)") {
		t.Errorf("Hello.Signature = %q; quer 'func Hello(name string)'", hello.Signature)
	}
	if hello.StartLine != 22 {
		t.Errorf("Hello.StartLine = %d; want 22", hello.StartLine)
	}
	if hello.EndLine != 24 {
		t.Errorf("Hello.EndLine = %d; want 24", hello.EndLine)
	}
	if hello.StartCol != 1 {
		t.Errorf("Hello.StartCol = %d; want 1", hello.StartCol)
	}
	if !strings.Contains(hello.DocComment, "greets the world") {
		t.Errorf("Hello.DocComment = %q; quer o doc comment", hello.DocComment)
	}

	// Método: receiver vira ParentName e o qualified name carrega o tipo.
	run, ok := findSymbol(res.Symbols, KindMethod, "Run")
	if !ok {
		t.Fatal("esperava o método Run")
	}
	if run.QualifiedName != "example.Config.Run" {
		t.Errorf("Run.QualifiedName = %q; want example.Config.Run", run.QualifiedName)
	}
	if run.ParentName != "Config" {
		t.Errorf("Run.ParentName = %q; want Config", run.ParentName)
	}
	if !strings.Contains(run.Signature, "*Config") {
		t.Errorf("Run.Signature = %q; quer o receiver *Config", run.Signature)
	}

	// Struct e interface.
	cfg, ok := findSymbol(res.Symbols, KindStruct, "Config")
	if !ok {
		t.Fatal("esperava a struct Config")
	}
	if cfg.QualifiedName != "example.Config" {
		t.Errorf("Config.QualifiedName = %q; want example.Config", cfg.QualifiedName)
	}
	if cfg.StartLine != 6 || cfg.EndLine != 8 {
		t.Errorf("Config posição = %d..%d; want 6..8", cfg.StartLine, cfg.EndLine)
	}
	if _, ok := findSymbol(res.Symbols, KindInterface, "Runner"); !ok {
		t.Fatal("esperava a interface Runner")
	}

	// Constante e variável.
	if c, ok := findSymbol(res.Symbols, KindConstant, "MaxItems"); !ok {
		t.Fatal("esperava a constante MaxItems")
	} else if c.QualifiedName != "example.MaxItems" {
		t.Errorf("MaxItems.QualifiedName = %q; want example.MaxItems", c.QualifiedName)
	}
	if vr, ok := findSymbol(res.Symbols, KindVariable, "defaultLang"); !ok {
		t.Fatal("esperava a variável defaultLang")
	} else if vr.QualifiedName != "example.defaultLang" {
		t.Errorf("defaultLang.QualifiedName = %q; want example.defaultLang", vr.QualifiedName)
	}
}

// TestGoParserRejeitaGoInvalido: erro de sintaxe é fatal (o pipeline conta o
// arquivo como erro) — o backend não devolve FileResult parcial.
func TestGoParserRejeitaGoInvalido(t *testing.T) {
	p := NewGoParser()
	if _, err := p.ParseFile("broken.go", []byte("package x\nfunc (")); err == nil {
		t.Fatal("esperava erro de sintaxe")
	}
}

// TestGoParserExtraiCallsComCallSite cobre T9: o backend extrai as chamadas do
// AST e cada aresta `calls` carrega o PONTO DE CHAMADA (a linha/coluna da
// expressão de chamada) — e não a posição da definição.
func TestGoParserExtraiCallsComCallSite(t *testing.T) {
	src := `package example

import "fmt"

// Helper ajuda.
func Helper(n int) int {
	return n + 1
}

// Config configura.
type Config struct{}

// Run executa.
func (c *Config) Run() error {
	return nil
}

// Caller chama Helper.
func Caller(n int) int {
	helper := Helper(n)
	fmt.Println(helper)
	return helper
}

// Caller2 usa um valor local tipado.
func Caller2() error {
	cfg := &Config{}
	return cfg.Run()
}
`
	p := NewGoParser()
	res, err := p.ParseFile("internal/example/calls.go", []byte(src))
	if err != nil {
		t.Fatalf("ParseFile falhou: %v", err)
	}

	var calls []Edge
	for _, e := range res.Edges {
		if e.Kind == EdgeCalls {
			calls = append(calls, e)
		}
	}
	if len(calls) != 3 {
		t.Fatalf("arestas calls = %d; want 3 (Helper, fmt.Println, Config.Run); got %+v", len(calls), calls)
	}

	// A chamada a Helper (linha 24) é a PRIMEIRA: o destino é resolvido pelo
	// qualified name (mesmo pacote) e o call-site é a linha da chamada.
	first := calls[0]
	if first.SrcQualifiedName != "example.Caller" {
		t.Errorf("origem = %q; want example.Caller (o símbolo ENCLOSING)", first.SrcQualifiedName)
	}
	if first.DstName != "Helper" {
		t.Errorf("dst name = %q; want Helper", first.DstName)
	}
	if first.DstQualifiedName != "example.Helper" {
		t.Errorf("dst qualified = %q; want example.Helper", first.DstQualifiedName)
	}
	if first.Line != 20 {
		t.Errorf("call-site line = %d; want 20 (a linha da chamada, não da definição)", first.Line)
	}
	if first.Column != 12 {
		t.Errorf("call-site column = %d; want 12", first.Column)
	}

	// A DEFINIÇÃO de Helper é a linha 6 — o call-site (20) é diferente.
	def, ok := findSymbol(res.Symbols, KindFunction, "Helper")
	if !ok {
		t.Fatal("esperava a função Helper")
	}
	if def.StartLine == first.Line {
		t.Errorf("o call-site (%d) não pode ser a linha da definição (%d)", first.Line, def.StartLine)
	}

	// A chamada a um pacote importado carrega o qualified name do pacote.
	var pkgCall *Edge
	for i := range calls {
		if calls[i].DstName == "Println" {
			pkgCall = &calls[i]
			break
		}
	}
	if pkgCall == nil {
		t.Fatal("esperava a chamada fmt.Println")
	}
	if pkgCall.DstQualifiedName != "fmt.Println" {
		t.Errorf("dst qualified = %q; want fmt.Println", pkgCall.DstQualifiedName)
	}
	if pkgCall.Line != 21 {
		t.Errorf("call-site line = %d; want 21", pkgCall.Line)
	}

	// Método chamado sobre um valor local de tipo CONHECIDO: o qualified name
	// é o do método (pkg.Tipo.Metodo) — resolvível por escopo de pacote.
	var methodCall *Edge
	for i := range calls {
		if calls[i].DstName == "Run" {
			methodCall = &calls[i]
			break
		}
	}
	if methodCall == nil {
		t.Fatal("esperava a chamada de método cfg.Run()")
	}
	if methodCall.DstQualifiedName != "example.Config.Run" {
		t.Errorf("dst qualified = %q; want example.Config.Run", methodCall.DstQualifiedName)
	}
	if methodCall.SrcQualifiedName != "example.Caller2" {
		t.Errorf("origem = %q; want example.Caller2", methodCall.SrcQualifiedName)
	}
	if methodCall.Line != 28 {
		t.Errorf("call-site line = %d; want 28", methodCall.Line)
	}
}

// TestGoParserExtraiImportsEInherits cobre T10: o backend emite `imports`
// (arquivo importador → pacote importado, com os símbolos sintéticos que dão
// corpo ao grafo) e `inherits`/`uses_type` (embedding de struct/interface e
// tipos usados).
func TestGoParserExtraiImportsEInherits(t *testing.T) {
	src := `package example

import (
	"fmt"
	"strings"
)

// Base é a base.
type Base struct {
	ID   int
	Next *Base
}

// Derived embuta Base.
type Derived struct {
	Base
	Name string
}

// Reader é uma interface.
type Reader interface {
	Read() error
}

// ReadCloser embuta Reader.
type ReadCloser interface {
	Reader
	Close() error
}

// Use usa Base.
func Use(b Base) *Base {
	return &b
}

// Format formata.
func Format(s string) string {
	return strings.TrimSpace(s)
}
`
	p := NewGoParser()
	res, err := p.ParseFile("internal/example/types.go", []byte(src))
	if err != nil {
		t.Fatalf("ParseFile falhou: %v", err)
	}

	byKind := map[EdgeKind][]Edge{}
	for _, e := range res.Edges {
		byKind[e.Kind] = append(byKind[e.Kind], e)
	}

	// imports: dois imports, cada um com a declaração como origem e o
	// pacote como destino.
	imports := byKind[EdgeImports]
	if len(imports) != 2 {
		t.Fatalf("arestas imports = %d; want 2; got %+v", len(imports), imports)
	}
	if imports[0].SrcQualifiedName != "import:fmt" || imports[0].DstQualifiedName != "fmt" {
		t.Errorf("imports[0] = %s -> %s; want import:fmt -> fmt", imports[0].SrcQualifiedName, imports[0].DstQualifiedName)
	}
	if imports[0].Line != 4 {
		t.Errorf("imports[0].Line = %d; want 4 (a posição do import)", imports[0].Line)
	}
	if imports[1].SrcQualifiedName != "import:strings" || imports[1].DstQualifiedName != "strings" {
		t.Errorf("imports[1] = %s -> %s; want import:strings -> strings", imports[1].SrcQualifiedName, imports[1].DstQualifiedName)
	}

	// Os símbolos sintéticos do import existem (a origem e o destino).
	var hasDecl, hasPkg bool
	for _, s := range res.Symbols {
		if s.Kind != KindImport {
			continue
		}
		if s.QualifiedName == "import:fmt" {
			hasDecl = true
		}
		if s.QualifiedName == "fmt" {
			hasPkg = true
		}
	}
	if !hasDecl || !hasPkg {
		t.Errorf("faltam os símbolos sintéticos do import (decl=%v pkg=%v)", hasDecl, hasPkg)
	}

	// inherits: embedding de struct e de interface.
	inherits := byKind[EdgeInherits]
	if len(inherits) != 2 {
		t.Fatalf("arestas inherits = %d; want 2 (Derived→Base, ReadCloser→Reader); got %+v", len(inherits), inherits)
	}
	found := map[string]bool{}
	for _, e := range inherits {
		found[e.SrcQualifiedName+"->"+e.DstQualifiedName] = true
	}
	if !found["example.Derived->example.Base"] {
		t.Errorf("faltou a herança example.Derived->example.Base: %+v", inherits)
	}
	if !found["example.ReadCloser->example.Reader"] {
		t.Errorf("faltou a herança example.ReadCloser->example.Reader: %+v", inherits)
	}

	// uses_type: campo de struct e parâmetros/resultados.
	usesType := byKind[EdgeUsesType]
	if len(usesType) < 3 {
		t.Fatalf("arestas uses_type = %d; want ≥3 (campo Next, param e resultado de Use)", len(usesType))
	}
	uses := map[string]bool{}
	for _, e := range usesType {
		uses[e.SrcQualifiedName+"->"+e.DstQualifiedName] = true
	}
	if !uses["example.Base->example.Base"] {
		t.Errorf("faltou o uso de tipo example.Base->example.Base (campo Next): %+v", usesType)
	}
	if !uses["example.Use->example.Base"] {
		t.Errorf("faltou o uso de tipo example.Use->example.Base (parâmetro): %+v", usesType)
	}

	// Tipos da linguagem (string, int) não geram aresta: não são símbolos do
	// repositório.
	for _, e := range usesType {
		if e.DstName == "string" || e.DstName == "int" {
			t.Errorf("tipo builtin não deve gerar aresta: %+v", e)
		}
	}
}

// TestGoParserNaoCriaArestaParaBuiltin: builtins não são símbolos do
// repositório e não geram aresta `calls`.
func TestGoParserNaoCriaArestaParaBuiltin(t *testing.T) {
	p := NewGoParser()
	res, err := p.ParseFile("a.go", []byte("package x\n\nfunc A(s []int) int {\n\treturn len(s)\n}\n"))
	if err != nil {
		t.Fatalf("ParseFile falhou: %v", err)
	}
	for _, e := range res.Edges {
		if e.DstName == "len" {
			t.Fatalf("builtin len não deve gerar aresta: %+v", e)
		}
	}
}
