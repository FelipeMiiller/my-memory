package codeast

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strconv"
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
	v.buildImportMap()
	v.collectSymbols()
	v.collectCalls()

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
	// imports mapeia o nome local usado no arquivo (alias ou base do path)
	// para o path importado.
	imports map[string]string
}

// collectCalls extrai as arestas `calls` — cada uma carrega o PONTO DE
// CHAMADA (a linha/coluna da expressão de chamada), não a definição.
//
// A origem da aresta é o símbolo ENCLOSING (a função/método que contém a
// chamada); o destino é o símbolo chamado.
func (v *goFileVisitor) collectCalls() {
	for _, decl := range v.file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Body == nil {
				continue
			}
			v.walkCalls(d.Body, v.funcQualifiedName(d), v.funcLocals(d))
		case *ast.GenDecl:
			// Inicializadores de package-level (var/const): a origem é o
			// próprio símbolo declarado.
			for _, spec := range d.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, name := range vs.Names {
					if i >= len(vs.Values) {
						continue
					}
					v.walkCalls(vs.Values[i], v.pkg+"."+name.Name, map[string]string{})
				}
			}
		}
	}
}

// walkCalls percorre uma expressão procurando chamadas.
func (v *goFileVisitor) walkCalls(node ast.Node, srcQualified string, locals map[string]string) {
	ast.Inspect(node, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		v.addCallEdge(call, srcQualified, locals)
		return true
	})
}

// addCallEdge registra uma aresta `calls` com o call-site.
func (v *goFileVisitor) addCallEdge(call *ast.CallExpr, srcQualified string, locals map[string]string) {
	name, qualified := v.calleeTarget(call.Fun, locals)
	if name == "" {
		return
	}
	if isGoBuiltin(name) {
		return // builtins não são símbolos do repositório
	}
	pos := v.pos(call.Pos())
	v.edges = append(v.edges, Edge{
		Kind:             EdgeCalls,
		SrcQualifiedName: srcQualified,
		DstName:          name,
		DstQualifiedName: qualified,
		Line:             pos.Line,
		Column:           pos.Column,
	})
}

// calleeTarget devolve o nome e o qualified name do destino de uma chamada.
//
// Quando o alvo é um pacote IMPORTADO, o qualified name é "pacote.Funcao" —
// é assim que os símbolos são nomeados (pkg.Nome) e é o que permite resolver
// por escopo de pacote (EXTRACTED). Quando o alvo é uma função de
// package-level, o qualified name é "pkg.Nome" do pacote do arquivo. Quando o
// alvo é um método chamado sobre um valor local de tipo CONHECIDO
// (ex.: `cfg := &Config{}` → `cfg.Run()`), o qualified name é
// "pkg.Tipo.Metodo" — também resolvível por escopo de pacote.
//
// Quando o tipo do receptor é desconhecido (campo de struct, valor de
// interface, função guardada em variável), só o NOME é devolvido: a resolução
// cai para o nível "por nome" (INFERRED) ou para o bucket de incertas.
func (v *goFileVisitor) calleeTarget(fun ast.Expr, locals map[string]string) (name string, qualified string) {
	switch f := fun.(type) {
	case *ast.Ident:
		if _, isLocal := locals[f.Name]; isLocal {
			// valor local (função guardada em variável): o destino real não é
			// determinável sem type-checking → sem aresta.
			return "", ""
		}
		// função de package-level: o qualified name é pkg.Nome
		return f.Name, v.pkg + "." + f.Name
	case *ast.SelectorExpr:
		if pkgIdent, ok := f.X.(*ast.Ident); ok {
			if base, isImport := v.importPackageBase(pkgIdent.Name); isImport {
				return f.Sel.Name, base + "." + f.Sel.Name
			}
			if typeQN, isLocal := locals[pkgIdent.Name]; isLocal && typeQN != "" {
				return f.Sel.Name, typeQN + "." + f.Sel.Name
			}
		}
		return f.Sel.Name, ""
	case *ast.ParenExpr:
		return v.calleeTarget(f.X, locals)
	case *ast.IndexExpr:
		return v.calleeTarget(f.X, locals)
	case *ast.IndexListExpr:
		return v.calleeTarget(f.X, locals)
	}
	return "", ""
}

// funcLocals monta o mapa de variáveis locais com o tipo CONHECIDO (quando
// dá para saber): parâmetros, receiver e declarações com tipo explícito.
//
// É o que permite resolver `cfg.Run()` pelo tipo do receiver — sem isso toda
// chamada de método cairia na resolução "por nome" (INFERRED/incerta).
func (v *goFileVisitor) funcLocals(d *ast.FuncDecl) map[string]string {
	locals := make(map[string]string)
	if d.Recv != nil {
		for _, f := range d.Recv.List {
			qn := v.typeQualifiedName(f.Type)
			for _, n := range f.Names {
				if qn != "" {
					locals[n.Name] = qn
				}
			}
		}
	}
	if d.Type.Params != nil {
		for _, f := range d.Type.Params.List {
			qn := v.typeQualifiedName(f.Type)
			for _, n := range f.Names {
				if qn != "" {
					locals[n.Name] = qn
				}
			}
		}
	}
	if d.Body == nil {
		return locals
	}
	ast.Inspect(d.Body, func(n ast.Node) bool {
		switch decl := n.(type) {
		case *ast.AssignStmt:
			if decl.Tok != token.DEFINE {
				return true
			}
			for i, lhs := range decl.Lhs {
				ident, ok := lhs.(*ast.Ident)
				if !ok || i >= len(decl.Rhs) {
					continue
				}
				if qn := v.compositeTypeQualified(decl.Rhs[i]); qn != "" {
					locals[ident.Name] = qn
				}
			}
		case *ast.ValueSpec:
			for i, name := range decl.Names {
				if decl.Type != nil {
					if qn := v.typeQualifiedName(decl.Type); qn != "" {
						locals[name.Name] = qn
						continue
					}
				}
				if i < len(decl.Values) {
					if qn := v.compositeTypeQualified(decl.Values[i]); qn != "" {
						locals[name.Name] = qn
					}
				}
			}
		}
		return true
	})
	return locals
}

// typeQualifiedName devolve o qualified name do tipo de uma expressão
// ("pkg.Tipo"), quando o tipo é determinável sem type-checking.
func (v *goFileVisitor) typeQualifiedName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		if _, isImport := v.importPackageBase(t.Name); isImport {
			return "" // é o nome de um pacote, não de um tipo
		}
		return v.pkg + "." + t.Name
	case *ast.StarExpr:
		return v.typeQualifiedName(t.X)
	case *ast.SelectorExpr:
		if pkgIdent, ok := t.X.(*ast.Ident); ok {
			if base, isImport := v.importPackageBase(pkgIdent.Name); isImport {
				return base + "." + t.Sel.Name
			}
		}
		return ""
	case *ast.ArrayType:
		return v.typeQualifiedName(t.Elt)
	case *ast.MapType:
		return v.typeQualifiedName(t.Value)
	}
	return ""
}

// compositeTypeQualified devolve o qualified name do tipo de um literal
// composto (`Config{}`, `&Config{}`, `new(Config)`).
func (v *goFileVisitor) compositeTypeQualified(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.CompositeLit:
		return v.typeQualifiedName(e.Type)
	case *ast.UnaryExpr:
		if e.Op == token.AND {
			return v.compositeTypeQualified(e.X)
		}
	case *ast.CallExpr:
		if ident, ok := e.Fun.(*ast.Ident); ok && ident.Name == "new" && len(e.Args) == 1 {
			return v.typeQualifiedName(e.Args[0])
		}
	}
	return ""
}

// importPackageBase devolve o nome do pacote importado quando o identificador
// local é um pacote importado (alias ou base do path).
func (v *goFileVisitor) importPackageBase(local string) (string, bool) {
	path, ok := v.imports[local]
	if !ok {
		return "", false
	}
	return pathBase(path), true
}

// funcQualifiedName devolve o qualified name de uma função/método.
func (v *goFileVisitor) funcQualifiedName(d *ast.FuncDecl) string {
	if d.Recv != nil && len(d.Recv.List) > 0 {
		return v.pkg + "." + recvTypeName(d.Recv.List[0].Type) + "." + d.Name.Name
	}
	return v.pkg + "." + d.Name.Name
}

// buildImportMap mapeia o nome local usado no arquivo para o path importado.
func (v *goFileVisitor) buildImportMap() {
	v.imports = make(map[string]string)
	for _, spec := range v.file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		local := ""
		if spec.Name != nil {
			local = spec.Name.Name
		}
		if local == "" {
			local = pathBase(path)
		}
		v.imports[local] = path
	}
}

// pathBase devolve o nome do pacote a partir do path de import
// ("gopkg.in/yaml.v3" → "yaml").
func pathBase(path string) string {
	path = strings.TrimSuffix(path, "/")
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		path = path[idx+1:]
	}
	if idx := strings.LastIndex(path, "."); idx > 0 {
		path = path[:idx]
	}
	return path
}

// goBuiltins são os builtins da linguagem: eles não são símbolos do
// repositório, então não geram aresta `calls`.
var goBuiltins = map[string]bool{
	"append": true, "cap": true, "clear": true, "close": true, "complex": true,
	"copy": true, "delete": true, "imag": true, "len": true, "make": true,
	"max": true, "min": true, "new": true, "panic": true, "print": true,
	"println": true, "real": true, "recover": true,
}

// isGoBuiltin devolve true se o nome é um builtin da linguagem.
func isGoBuiltin(name string) bool {
	return goBuiltins[name]
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
