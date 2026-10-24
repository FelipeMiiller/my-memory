package codeast

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"
)

// GoParser é o backend de AST em Go puro: usa `go/parser`, `go/ast` e
// `go/token` da stdlib — sem CGO, sem build tag e sem dependência nova no
// go.mod (ADR-047 continua intacto: o plano de binding tree-sitter segue
// deferred, 0/3 gatilhos).
//
// O backend extrai símbolos com posição exata (kind, nome, qualified name,
// assinatura, linha/coluna de início e fim e doc comment) e, a partir da
// Fase 2, também arestas estruturais (calls/imports/inherits/uses_type) com o
// PONTO DE CHAMADA (call-site) — ver Edge.
type GoParser struct{}

// NewGoParser cria o backend Go (stdlib).
func NewGoParser() *GoParser {
	return &GoParser{}
}

// Backend identifica a implementação: "go-stdlib".
func (g *GoParser) Backend() string { return "go-stdlib" }

// Languages devolve as linguagens suportadas pelo backend (lower-case).
// O backend Go entende apenas Go — as demais linguagens do top-10 do
// ADR-047 continuam no MockParser (fallback determinístico).
func (g *GoParser) Languages() []string { return []string{"go"} }

// ParseFile extrai os símbolos estruturais de um arquivo Go.
//
// Erros de sintaxe são fatais (o pipeline conta o arquivo como erro); já
// erros não-fatais do arquivo entram em FileResult.Errors.
func (g *GoParser) ParseFile(path string, content []byte) (*FileResult, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, content, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("go_parser: parse %s: %w", path, err)
	}

	v := &goFileVisitor{fset: fset, file: file, pkg: file.Name.Name}
	v.collectSymbols()

	res := &FileResult{
		Path:     path,
		Language: "go",
		Symbols:  v.symbols,
		Edges:    v.edges,
	}
	return res, nil
}

// goFileVisitor percorre o arquivo Go uma vez, coletando símbolos (e, na
// Fase 2, arestas) com posição exata.
type goFileVisitor struct {
	fset    *token.FileSet
	file    *ast.File
	pkg     string
	symbols []Symbol
	edges   []Edge
}

// pos devolve a posição (linha/coluna) de um token.
func (v *goFileVisitor) pos(p token.Pos) token.Position {
	return v.fset.Position(p)
}

// collectSymbols percorre as declarações de topo do arquivo.
func (v *goFileVisitor) collectSymbols() {
	for _, decl := range v.file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			v.addFuncSymbol(d)
		case *ast.GenDecl:
			v.addGenDeclSymbols(d)
		}
	}
}

// addFuncSymbol registra uma função ou método.
func (v *goFileVisitor) addFuncSymbol(d *ast.FuncDecl) {
	name := d.Name.Name
	kind := KindFunction
	parent := ""
	qualified := v.pkg + "." + name

	if d.Recv != nil && len(d.Recv.List) > 0 {
		kind = KindMethod
		parent = recvTypeName(d.Recv.List[0].Type)
		qualified = v.pkg + "." + parent + "." + name
	}

	start := v.pos(d.Pos())
	end := v.pos(d.End())
	v.symbols = append(v.symbols, Symbol{
		Kind:          kind,
		Name:          name,
		QualifiedName: qualified,
		Signature:     v.funcSignature(d),
		StartLine:     start.Line,
		EndLine:       end.Line,
		StartCol:      start.Column,
		EndCol:        end.Column,
		DocComment:    docText(d.Doc),
		ParentName:    parent,
	})
}

// addGenDeclSymbols registra os símbolos de type/const/var.
func (v *goFileVisitor) addGenDeclSymbols(d *ast.GenDecl) {
	switch d.Tok {
	case token.TYPE:
		for _, spec := range d.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			kind := KindClass
			switch ts.Type.(type) {
			case *ast.StructType:
				kind = KindStruct
			case *ast.InterfaceType:
				kind = KindInterface
			}
			start := v.pos(ts.Pos())
			end := v.pos(ts.End())
			doc := docText(ts.Doc)
			if doc == "" {
				doc = docText(d.Doc)
			}
			v.symbols = append(v.symbols, Symbol{
				Kind:          kind,
				Name:          ts.Name.Name,
				QualifiedName: v.pkg + "." + ts.Name.Name,
				Signature:     v.typeSignature(ts),
				StartLine:     start.Line,
				EndLine:       end.Line,
				StartCol:      start.Column,
				EndCol:        end.Column,
				DocComment:    doc,
			})
		}
	case token.CONST:
		for _, spec := range d.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range vs.Names {
				start := v.pos(name.Pos())
				end := v.pos(name.End())
				v.symbols = append(v.symbols, Symbol{
					Kind:          KindConstant,
					Name:          name.Name,
					QualifiedName: v.pkg + "." + name.Name,
					Signature:     "const " + name.Name,
					StartLine:     start.Line,
					EndLine:       end.Line,
					StartCol:      start.Column,
					EndCol:        end.Column,
					DocComment:    docText(vs.Doc),
				})
			}
		}
	case token.VAR:
		for _, spec := range d.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range vs.Names {
				start := v.pos(name.Pos())
				end := v.pos(name.End())
				signature := "var " + name.Name
				if vs.Type != nil {
					signature += " " + v.exprString(vs.Type)
				}
				v.symbols = append(v.symbols, Symbol{
					Kind:          KindVariable,
					Name:          name.Name,
					QualifiedName: v.pkg + "." + name.Name,
					Signature:     signature,
					StartLine:     start.Line,
					EndLine:       end.Line,
					StartCol:      start.Column,
					EndCol:        end.Column,
					DocComment:    docText(vs.Doc),
				})
			}
		}
	}
}

// funcSignature monta a assinatura textual da função (sem o corpo).
func (v *goFileVisitor) funcSignature(d *ast.FuncDecl) string {
	var b strings.Builder
	b.WriteString("func ")
	if d.Recv != nil && len(d.Recv.List) > 0 {
		b.WriteString("(")
		parts := make([]string, 0, len(d.Recv.List))
		for _, f := range d.Recv.List {
			typ := v.exprString(f.Type)
			if len(f.Names) > 0 {
				typ += " " + f.Names[0].Name
			}
			parts = append(parts, typ)
		}
		b.WriteString(strings.Join(parts, ", "))
		b.WriteString(") ")
	}
	b.WriteString(d.Name.Name)
	b.WriteString(v.fieldListString(d.Type.Params))
	b.WriteString(v.fieldListString(d.Type.Results))
	return b.String()
}

// typeSignature monta a assinatura textual de um type.
func (v *goFileVisitor) typeSignature(ts *ast.TypeSpec) string {
	sig := "type " + ts.Name.Name
	if ts.TypeParams != nil {
		sig += v.fieldListString(ts.TypeParams)
	}
	if ts.Type != nil {
		sig += " " + v.exprString(ts.Type)
	}
	return sig
}

// fieldListString renderiza uma lista de campos/parâmetros.
func (v *goFileVisitor) fieldListString(fl *ast.FieldList) string {
	if fl == nil || len(fl.List) == 0 {
		return ""
	}
	parts := make([]string, 0, len(fl.List))
	for _, f := range fl.List {
		typ := v.exprString(f.Type)
		if len(f.Names) == 0 {
			parts = append(parts, typ)
			continue
		}
		names := make([]string, 0, len(f.Names))
		for _, n := range f.Names {
			names = append(names, n.Name)
		}
		parts = append(parts, strings.Join(names, ", ")+" "+typ)
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

// exprString renderiza uma expressão Go usando go/printer.
func (v *goFileVisitor) exprString(e ast.Expr) string {
	if e == nil {
		return ""
	}
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, v.fset, e); err != nil {
		return ""
	}
	return buf.String()
}

// recvTypeName devolve o nome do tipo receptor de um método (sem `*`).
func recvTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return recvTypeName(t.X)
	case *ast.IndexExpr:
		return recvTypeName(t.X)
	case *ast.IndexListExpr:
		return recvTypeName(t.X)
	case *ast.SelectorExpr:
		return t.Sel.Name
	}
	return ""
}

// docText extrai o texto de um doc comment (string vazia quando não há).
func docText(group *ast.CommentGroup) string {
	if group == nil {
		return ""
	}
	return strings.TrimSpace(group.Text())
}
