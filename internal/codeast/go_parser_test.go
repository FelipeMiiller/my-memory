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

// TestGoParserNaoDevolveArestasNaFase1 seria o placeholder da Fase 2 — a
// partir de T9 o backend passa a devolver arestas `calls`.
func TestGoParserNaoDevolveArestasNaFase1(t *testing.T) {
	p := NewGoParser()
	res, err := p.ParseFile("a.go", []byte(goSourceFixture))
	if err != nil {
		t.Fatalf("ParseFile falhou: %v", err)
	}
	if len(res.Edges) != 0 {
		t.Fatalf("Edges = %d; want 0 (fase de símbolos)", len(res.Edges))
	}
}
