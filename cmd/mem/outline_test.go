package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestParseVaultHeadings_LeveEProfundidade(t *testing.T) {
	content := "# Raiz\ntexto\n## Intermediate\n### Fundo\n#### Mais fundo\n"

	headings := parseVaultHeadings(content)
	if len(headings) != 4 {
		t.Fatalf("esperava 4 títulos, obteve %d: %+v", len(headings), headings)
	}

	want := []vaultHeading{
		{Level: 1, Text: "Raiz"},
		{Level: 2, Text: "Intermediate"},
		{Level: 3, Text: "Fundo"},
		{Level: 4, Text: "Mais fundo"},
	}
	for i := range want {
		if headings[i] != want[i] {
			t.Errorf("título %d: esperava %+v, obteve %+v", i, want[i], headings[i])
		}
	}
}

// TestParseVaultHeadings_IgnoraBlocoDeCodigo fecha o caso que faria o outline
// inventar seções: um "# comment" dentro de um exemplo de shell é código, não
// estrutura da nota.
func TestParseVaultHeadings_IgnoraBlocoDeCodigo(t *testing.T) {
	content := "# Real\n\n```bash\n# isto e um comentario do shell\n## tambem nao e titulo\n```\n\n## Depois do bloco\n"

	headings := parseVaultHeadings(content)
	if len(headings) != 2 {
		t.Fatalf("esperava 2 títulos (fora do fence), obteve %d: %+v", len(headings), headings)
	}
	if headings[0].Text != "Real" || headings[1].Text != "Depois do bloco" {
		t.Errorf("títulos incorretos: %+v", headings)
	}
}

func TestParseVaultHeadings_NaoConfundeHashSolto(t *testing.T) {
	content := "###sem espaco\n####### sete hashes\n####  \n"

	if headings := parseVaultHeadings(content); len(headings) != 0 {
		t.Errorf("linhas que não são título ATX foram aceitas: %+v", headings)
	}
}

func TestRunOutlineCommand_ImprimeEstrutura(t *testing.T) {
	content := "# Nota\n\n## Seção\n\n### Subseção\n"
	dbPath := newVaultNoteFixture(t, "docs/nota.md", content)

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--storage", "sqlite", "docs/nota.md"}
	if err := runOutlineCommand(context.Background(), "fixture-repo", args, &buf); err != nil {
		t.Fatalf("runOutlineCommand falhou: %v", err)
	}

	want := "Nota\n  Seção\n    Subseção\n"
	if buf.String() != want {
		t.Errorf("outline incorreto.\nEsperado:\n%q\nObtido:\n%q", want, buf.String())
	}
}

// TestRunOutlineCommand_NotaSemTitulosImprimeNada é o critério da spec: nota
// sem headings sai zero e em silêncio. Imprimir erro aqui faria o comando
// falhar em notas curtas, que são a maioria de um vault.
func TestRunOutlineCommand_NotaSemTitulosImprimeNada(t *testing.T) {
	dbPath := newVaultNoteFixture(t, "docs/nota.md", "Só um parágrafo, sem título nenhum.\n")

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--storage", "sqlite", "docs/nota.md"}
	if err := runOutlineCommand(context.Background(), "fixture-repo", args, &buf); err != nil {
		t.Fatalf("nota sem títulos deveria sair zero, obteve erro: %v", err)
	}
	if buf.String() != "" {
		t.Errorf("nota sem títulos deveria imprimir nada, obteve:\n%q", buf.String())
	}
}

func TestRunOutlineCommand_SemArgumentoFalha(t *testing.T) {
	dbPath := newVaultNoteFixture(t, "docs/nota.md", "# Nota\n")

	var buf bytes.Buffer
	err := runOutlineCommand(context.Background(), "fixture-repo", []string{"--db", dbPath, "--storage", "sqlite"}, &buf)
	if err == nil {
		t.Fatal("mem outline sem caminho deveria falhar")
	}
}

func TestRunWordcountCommand_ContaPalavrasECaracteres(t *testing.T) {
	content := "Uma nota com acentuação.\n"
	dbPath := newVaultNoteFixture(t, "docs/nota.md", content)

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--storage", "sqlite", "docs/nota.md"}
	if err := runWordcountCommand(context.Background(), "fixture-repo", args, &buf); err != nil {
		t.Fatalf("runWordcountCommand falhou: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Palavras:    4") {
		t.Errorf("contagem de palavras incorreta:\n%s", out)
	}
	// 25 pontos de código (o "ã" e o "ç" contam 1 cada), não bytes.
	if !strings.Contains(out, "Caracteres:  25") {
		t.Errorf("contagem de caracteres deveria contar runes:\n%s", out)
	}
}

func TestRunWordcountCommand_SemArgumentoFalha(t *testing.T) {
	dbPath := newVaultNoteFixture(t, "docs/nota.md", "# Nota\n")

	var buf bytes.Buffer
	err := runWordcountCommand(context.Background(), "fixture-repo", []string{"--db", dbPath, "--storage", "sqlite"}, &buf)
	if err == nil {
		t.Fatal("mem wordcount sem caminho deveria falhar")
	}
}
