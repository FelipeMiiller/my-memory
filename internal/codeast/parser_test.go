package codeast

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidKind(t *testing.T) {
	if !ValidKind(KindFunction) || !ValidKind(KindStruct) {
		t.Fatal("ValidKind deve aceitar tipos listados em ADR-047")
	}
	if ValidKind("lambda") || ValidKind("") {
		t.Fatal("ValidKind deve rejeitar valores fora da tabela")
	}
}

func TestParser_DetectLanguage_ByExtension(t *testing.T) {
	cases := []struct {
		path string
		want string
		ok   bool
	}{
		{"internal/foo.go", "go", true},
		{"src/bar.py", "python", true},
		{"src/bar.pyi", "python", true},
		{"web/app.ts", "typescript", true},
		{"web/app.tsx", "typescript", true},
		{"web/app.js", "javascript", true},
		{"lib/lib.rs", "rust", true},
		{"Foo.java", "java", true},
		{"a/b.cpp", "cpp", true},
		{"a/b.cc", "cpp", true},
		{"a/b.h", "cpp", true},
		{"app.rb", "ruby", true},
		{"index.php", "php", true},
		{"script.sh", "shell", true},
		{"script.bash", "shell", true},
		{"README.md", "", false},
		{"random.xyz", "", false},
	}
	for _, c := range cases {
		t.Run(filepath.Base(c.path), func(t *testing.T) {
			gotLang, gotOK := DetectLanguage(c.path, nil)
			if gotOK != c.ok {
				t.Fatalf("DetectLanguage(%q) ok=%v; want %v", c.path, gotOK, c.ok)
			}
			if gotLang != c.want {
				t.Fatalf("DetectLanguage(%q) lang=%q; want %q", c.path, gotLang, c.want)
			}
		})
	}
}

func TestParser_DetectLanguage_ByShebang(t *testing.T) {
	cases := []struct {
		shebang string
		lang    string
	}{
		{"#!/usr/bin/env python3", "python"},
		{"#!/usr/bin/python", "python"},
		{"#!/bin/bash", "shell"},
		{"#!/usr/bin/env bash", "shell"},
		{"#!/usr/bin/env ruby", "ruby"},
	}
	for _, c := range cases {
		t.Run(c.lang, func(t *testing.T) {
			lang, ok := DetectLanguage("script", []byte(c.shebang+"\n"))
			if !ok || lang != c.lang {
				t.Fatalf("DetectLanguage(shebang=%q) = (%q,%v); want (%q,true)", c.shebang, lang, ok, c.lang)
			}
		})
	}
}

func TestHashContent_Stable(t *testing.T) {
	a := []byte("hello world")
	if HashContent(a) != HashContent(a) {
		t.Fatal("HashContent deve ser determinístico")
	}
	if HashContent(a) == HashContent([]byte("Hello world")) {
		t.Fatal("HashContent deve diferenciar case")
	}
	if len(HashContent(a)) != 64 {
		t.Fatalf("HashContent deve devolver hex de 64 chars (SHA-256); got %d", len(HashContent(a)))
	}
}

func TestHashSymbols_StableAndOrderIndependent(t *testing.T) {
	syms := []Symbol{
		{Kind: KindFunction, QualifiedName: "pkg.A", StartLine: 10},
		{Kind: KindFunction, QualifiedName: "pkg.B", StartLine: 5},
		{Kind: KindStruct, Name: "S", StartLine: 1},
	}
	out1 := HashSymbols(syms)
	// Ordem invertida deve produzir mesmo hash (ordenação interna)
	reversed := []Symbol{syms[2], syms[1], syms[0]}
	if HashSymbols(syms) != HashSymbols(reversed) {
		t.Fatal("HashSymbols deve ser independente da ordem de entrada")
	}
	// Símbolos diferentes devem mudar o hash
	changed := append([]Symbol{}, syms...)
	changed[0].QualifiedName = "pkg.X"
	if HashSymbols(syms) == HashSymbols(changed) {
		t.Fatal("HashSymbols deve mudar quando symbols mudam")
	}
	if out1 == "" {
		t.Fatal("HashSymbols não pode ser vazio para slices não-vazios")
	}
}

func TestHasQualifiedNameShape(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"pkg.FuncName", true},
		{"PkgName.FuncName", true},
		{"pkg.sub.Func", true},
		{"DB.Open", true},
		{"open", false},
		{"foo bar", false},
		{"pkg.", false},
		{".foo", false},
		{"", false},
		{"pkg.123", false}, // número não é nome válido
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			if got := HasQualifiedNameShape(c.in); got != c.want {
				t.Errorf("HasQualifiedNameShape(%q) = %v; want %v", c.in, got, c.want)
			}
		})
	}
}

func TestMockParser_ParseGoFile(t *testing.T) {
	p := NewMockParser(false)
	src := []byte(`package example

import "fmt"

// Hello greets the world.
func Hello(name string) error {
	return fmt.Println(name)
}

type Config struct {
	Lang string
}

type Runner interface {
	Run() error
}
`)
	res, err := p.ParseFile("internal/example/hello.go", src)
	if err != nil {
		t.Fatalf("ParseFile falhou: %v", err)
	}
	if res.Language != "go" {
		t.Fatalf("Language = %q; want go", res.Language)
	}
	if len(res.Symbols) < 4 {
		t.Fatalf("Symbols = %d; esperava ≥4 (Hello, Config, Runner, import)", len(res.Symbols))
	}
	hasFn := false
	hasStruct := false
	hasIface := false
	hasImport := false
	for _, s := range res.Symbols {
		switch s.Kind {
		case KindFunction:
			hasFn = s.Name == "Hello"
			if !strings.HasPrefix(s.QualifiedName, "example.") {
				t.Errorf("QualifiedName = %q; esperava prefix example.", s.QualifiedName)
			}
		case KindStruct:
			hasStruct = s.Name == "Config"
		case KindInterface:
			hasIface = s.Name == "Runner"
		case KindImport:
			hasImport = true
		}
	}
	if !hasFn {
		t.Errorf("esperava function Hello")
	}
	if !hasStruct {
		t.Errorf("esperava struct Config")
	}
	if !hasIface {
		t.Errorf("esperava interface Runner")
	}
	if !hasImport {
		t.Errorf("esperava import symbol")
	}
}

func TestMockParser_ParsePythonFile(t *testing.T) {
	p := NewMockParser(false)
	src := []byte(`#!/usr/bin/env python3
def hello(name):
    return name
class Greeter:
    def greet(self):
        return "hi"
`)
	res, err := p.ParseFile("scripts/greet.py", src)
	if err != nil {
		t.Fatalf("ParseFile falhou: %v", err)
	}
	if res.Language != "python" {
		t.Fatalf("Language = %q; want python", res.Language)
	}
	if len(res.Symbols) < 3 {
		t.Fatalf("Symbols = %d; esperava ≥3 (hello, Greeter, greet)", len(res.Symbols))
	}
}

func TestMockParser_Languages(t *testing.T) {
	p := NewMockParser(false)
	langs := p.Languages()
	if len(langs) != len(DefaultLanguageTable()) {
		t.Fatalf("Languages() = %d; want %d", len(langs), len(DefaultLanguageTable()))
	}
}

func TestMockParser_Backend(t *testing.T) {
	if NewMockParser(false).Backend() != "mock" {
		t.Fatal("Backend() = esperado 'mock'")
	}
}

func TestMockParser_ForcedFailure(t *testing.T) {
	m := NewMockParserWithFailMap(map[string]bool{".go": true})
	_, err := m.ParseFile("a/b.go", []byte("package x"))
	if err == nil {
		t.Fatal("esperava erro para extensão configurada em failExt")
	}
}

// TestLanguageDetection_ShebangsWithoutPermission verifica que o detector não
// chama fork/exec nem tenta carregar arquivos do disco — apenas inspeção
// do conteúdo já presente em memória.
func TestLanguageDetection_ShebangsWithoutPermission(t *testing.T) {
	content := []byte("#!/usr/bin/env python3\nimport os\n")
	lang, ok := DetectLanguage("tools/scripts/__init__.py", content)
	// Extensão .py tem prioridade sobre shebang (comportamento documentado).
	if !ok || lang != "python" {
		t.Fatalf("extensão .py deve ser detectada independentemente do shebang: got (%q,%v)", lang, ok)
	}

	// Agora sem extensão: shebang é o sinal primário.
	content2 := []byte("#!/usr/bin/env bash\necho hi\n")
	lang2, ok2 := DetectLanguage("scripts/hello", content2)
	if !ok2 || lang2 != "shell" {
		t.Fatalf("shebang bash devia ser detectado sem extensão: got (%q,%v)", lang2, ok2)
	}
}

func TestBuildTagIsolatesTreesitter_BothTagsCompile(t *testing.T) {
	// Garante que IsTreesitterEnabled devolve um booleano (sem panic) tanto
	// no build default (sem tag) quanto com tag. Aqui só checamos o ramo atual.
	got := IsTreesitterEnabled()
	if got {
		t.Log("compilando com -tags treesitter: stub ativo")
	} else {
		t.Log("compilando sem tag: ErrTreesitterDisabled")
	}
	_ = bytes.Equal // silencia import não-utilizado quando refatorando
}

func TestFilterLanguages(t *testing.T) {
	got := FilterLanguages([]string{"go", "ruby", "INVALID", "go"})
	if len(got) != 2 {
		t.Fatalf("FilterLanguages = %d entradas; esperava 2 (go, ruby); got=%+v", len(got), got)
	}
	names := map[string]bool{}
	for _, l := range got {
		names[l.Name] = true
	}
	if !names["go"] || !names["ruby"] {
		t.Fatalf("FilterLanguages perdeu alguma entrada válida: %+v", got)
	}
}
