package vaultfs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/parser"
)

// Como o alvo de um wikilink casou com um arquivo do vault.
const (
	// MatchBasename: casou pelo nome do arquivo, em qualquer profundidade.
	MatchBasename = "basename"
	// MatchRelPath: casou pelo caminho relativo dentro do vault.
	MatchRelPath = "relpath"
	// MatchTitle: casou pelo `title:` do frontmatter.
	MatchTitle = "title"
	// MatchAlias: casou por um `alias:` do frontmatter.
	MatchAlias = "alias"
)

// Resolution é o resultado da resolução de um alvo contra o disco do vault.
type Resolution struct {
	// Found diz se o alvo existe no vault.
	Found bool
	// Path é o caminho do arquivo encontrado, relativo à raiz do vault.
	Path string
	// Match diz como o alvo casou (MatchBasename, MatchRelPath, MatchTitle ou
	// MatchAlias).
	Match string
}

// Resolver resolve alvos de wikilink contra o disco de um vault.
//
// A identidade de um link no my-memory é o TÍTULO do documento, não o nome do
// arquivo: `documents.title` é `COALESCE(frontmatter.title, basename)`
// (ISSUE-010) e o indexador reescreve wikilink e tag contra os títulos
// indexados (`parser.ResolveTagConnections`). Por isso a resolução inclui o
// `title:` do frontmatter — no cofre central, 13 dos 125 links que existem em
// disco só resolvem por ele.
type Resolver struct {
	root    string
	byBase  map[string][]string
	byTitle map[string][]string
	byAlias map[string][]string
	byRel   map[string]string
}

// NewResolver varre o vault uma vez e monta o índice de resolução.
//
// Falha quando a raiz não é informada, não existe, não é um diretório ou não
// pode ser lida. Sem raiz legível o doctor não tem evidência para classificar
// um link: o resultado honesto é "não verificável", nunca "rompido".
func NewResolver(root string) (*Resolver, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("raiz do vault nao informada")
	}

	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("raiz do vault inacessivel: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("a raiz do vault nao e um diretorio: %s", root)
	}
	f, err := os.Open(root)
	if err != nil {
		return nil, fmt.Errorf("a raiz do vault nao pode ser lida: %w", err)
	}
	_ = f.Close()

	r := &Resolver{
		root:    root,
		byBase:  map[string][]string{},
		byTitle: map[string][]string{},
		byAlias: map[string][]string{},
		byRel:   map[string]string{},
	}

	walkErr := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			// Um arquivo ilegível no meio do vault não invalida o índice:
			// o alvo só não vai ser encontrado por ele.
			return nil
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(p), ".md") {
			return nil
		}

		rel, err := filepath.Rel(root, p)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)

		r.byRel[strings.ToLower(stripMdExt(rel))] = rel
		r.byRel[strings.ToLower(rel)] = rel
		r.add(r.byBase, stripMdExt(filepath.Base(p)), rel)

		data, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		fm, _ := parser.ExtractFrontmatter(string(data))
		if fm == nil {
			return nil
		}
		if fm.Title != "" {
			r.add(r.byTitle, fm.Title, rel)
		}
		for _, alias := range fm.Aliases {
			r.add(r.byAlias, alias, rel)
		}
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("falha ao varrer o vault: %w", walkErr)
	}

	return r, nil
}

// Resolve procura o alvo de um wikilink dentro do vault.
//
// A ordem é a do Obsidian: caminho relativo primeiro (com ou sem `.md`),
// depois o basename em qualquer profundidade, depois o `title:` e o `alias:`
// do frontmatter. O match é exato (case-insensitive) e nunca fuzzy: esta
// checagem decide se um link está rompido, e um match aproximado declararia
// vivo um link quebrado.
func (r *Resolver) Resolve(target string) Resolution {
	t := strings.TrimSpace(target)
	if t == "" {
		return Resolution{}
	}
	t = strings.ReplaceAll(t, "\\", "/")

	// 1. Caminho relativo dentro do vault, com ou sem a extensao .md.
	if p, ok := r.byRel[strings.ToLower(stripMdExt(t))]; ok {
		return Resolution{Found: true, Path: p, Match: MatchRelPath}
	}

	// 2. Basename em qualquer profundidade.
	base := stripMdExt(t)
	if i := strings.LastIndex(base, "/"); i >= 0 {
		base = base[i+1:]
	}
	if hits := r.byBase[strings.ToLower(base)]; len(hits) > 0 {
		return Resolution{Found: true, Path: hits[0], Match: MatchBasename}
	}

	// 3. title e alias do frontmatter — o alvo completo e, em seguida, o
	//    ultimo elemento do caminho.
	for _, candidate := range []string{t, base} {
		if hits := r.byTitle[strings.ToLower(candidate)]; len(hits) > 0 {
			return Resolution{Found: true, Path: hits[0], Match: MatchTitle}
		}
		if hits := r.byAlias[strings.ToLower(candidate)]; len(hits) > 0 {
			return Resolution{Found: true, Path: hits[0], Match: MatchAlias}
		}
	}

	return Resolution{}
}

// add registra um caminho sob uma chave, preservando a ordem de varredura
// (que e lexical) para que a ambiguidade seja resolvida de forma
// determinística.
func (r *Resolver) add(index map[string][]string, key, path string) {
	k := strings.ToLower(strings.TrimSpace(key))
	if k == "" {
		return
	}
	index[k] = append(index[k], path)
}

// stripMdExt remove a extensao .md (qualquer caixa) de um caminho.
func stripMdExt(p string) string {
	if len(p) > 3 && strings.EqualFold(p[len(p)-3:], ".md") {
		return p[:len(p)-3]
	}
	return p
}
