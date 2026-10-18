package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/FelipeMiiller/my-memory/internal/mcp"
)

// vaultHeading é um título ATX extraído do Markdown da nota.
type vaultHeading struct {
	Level int
	Text  string
}

// parseVaultHeadings extrai os títulos de um Markdown na ordem do documento.
//
// Só o estilo ATX (`# Título`) é lido: é o que o outline do Obsidian mostra
// por padrão, e o estilo underline (`Título\n=======`) é ambíguo demais para
// inferir sem parsear bloco por bloco. Blocos cercados por ``` são ignorados —
// um `# comentário` dentro de um exemplo de código não é título da nota.
func parseVaultHeadings(content string) []vaultHeading {
	var headings []vaultHeading
	inFence := false
	fenceMarker := ""

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimRight(line, "\r")

		if marker, isFence := fenceLine(trimmed); isFence {
			if !inFence {
				inFence = true
				fenceMarker = marker
			} else if strings.HasPrefix(marker, fenceMarker) {
				inFence = false
				fenceMarker = ""
			}
			continue
		}
		if inFence {
			continue
		}

		level, text, ok := atxHeading(trimmed)
		if !ok {
			continue
		}
		headings = append(headings, vaultHeading{Level: level, Text: text})
	}
	return headings
}

// fenceLine reconhece a abertura e o fechamento de um bloco de código.
func fenceLine(line string) (marker string, isFence bool) {
	trimmed := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(trimmed, "```") && !strings.HasPrefix(trimmed, "~~~") {
		return "", false
	}
	return trimmed[:3], true
}

// atxHeading interpreta uma linha como título ATX.
func atxHeading(line string) (level int, text string, ok bool) {
	hashes := 0
	for _, r := range line {
		if r != '#' {
			break
		}
		hashes++
	}
	if hashes == 0 || hashes > 6 {
		return 0, "", false
	}
	rest := line[hashes:]
	// ATX exige espaço depois dos # (ou fim de linha): "###texto" é parágrafo.
	if rest != "" && !strings.HasPrefix(rest, " ") && !strings.HasPrefix(rest, "\t") {
		return 0, "", false
	}
	text = strings.TrimSpace(strings.TrimRight(strings.TrimSpace(rest), "#"))
	if text == "" {
		return 0, "", false
	}
	return hashes, text, true
}

// vaultOutlineHeadings adapta o parser de títulos para a assinatura que o
// servidor MCP consome. `mem outline` e memory_note_outline precisam concordar
// sobre o que é um título: dois parsers divergiriam só para o agente e o humano
// lerem estruturas diferentes do mesmo arquivo.
func vaultOutlineHeadings(content string) []mcp.OutlineHeading {
	parsed := parseVaultHeadings(content)
	headings := make([]mcp.OutlineHeading, 0, len(parsed))
	for _, heading := range parsed {
		headings = append(headings, mcp.OutlineHeading{Level: heading.Level, Text: heading.Text})
	}
	return headings
}

// runOutlineCLI executa o subcomando mem outline direcionando para stdout
func runOutlineCLI(ctx context.Context, defaultRepo string, args []string) error {
	return runOutlineCommand(ctx, defaultRepo, args, os.Stdout)
}

// runOutlineCommand imprime a estrutura de títulos da nota, um por linha,
// indentada dois espaços por nível abaixo do primeiro.
//
// Nota sem título nenhum imprime nada e sai zero: é uma resposta válida sobre
// a nota, não uma falha do comando.
func runOutlineCommand(ctx context.Context, defaultRepo string, args []string, out io.Writer) error {
	outlineCmd := flag.NewFlagSet("outline", flag.ContinueOnError)
	outlineCmd.SetOutput(out)

	flags := registerVaultFlags(outlineCmd)

	if err := outlineCmd.Parse(args); err != nil {
		return err
	}

	if outlineCmd.NArg() == 0 {
		return fmt.Errorf("mem outline exige o caminho da nota (ex: mem outline docs/adr/001.md)")
	}

	query := vaultSlash(outlineCmd.Arg(0))

	reader, err := flags.open(defaultRepo)
	if err != nil {
		return err
	}
	defer reader.Close()

	doc, err := resolveVaultDocument(ctx, reader, query)
	if err != nil {
		return err
	}

	content, err := reader.ReadContent(ctx, doc.ID)
	if err != nil {
		return fmt.Errorf("falha ao ler o conteúdo de '%s': %w", doc.Path, err)
	}

	for _, heading := range parseVaultHeadings(content) {
		fmt.Fprintf(out, "%s%s\n", strings.Repeat("  ", heading.Level-1), heading.Text)
	}
	return nil
}

// runWordcountCLI executa o subcomando mem wordcount direcionando para stdout
func runWordcountCLI(ctx context.Context, defaultRepo string, args []string) error {
	return runWordcountCommand(ctx, defaultRepo, args, os.Stdout)
}

// runWordcountCommand imprime o tamanho da nota em palavras e caracteres.
//
// Caractere é ponto de código, não byte: uma nota em português tem acentos e
// contar byte daria um número que não corresponde ao que o editor mostra.
func runWordcountCommand(ctx context.Context, defaultRepo string, args []string, out io.Writer) error {
	wordcountCmd := flag.NewFlagSet("wordcount", flag.ContinueOnError)
	wordcountCmd.SetOutput(out)

	flags := registerVaultFlags(wordcountCmd)

	if err := wordcountCmd.Parse(args); err != nil {
		return err
	}

	if wordcountCmd.NArg() == 0 {
		return fmt.Errorf("mem wordcount exige o caminho da nota (ex: mem wordcount docs/adr/001.md)")
	}

	query := vaultSlash(wordcountCmd.Arg(0))

	reader, err := flags.open(defaultRepo)
	if err != nil {
		return err
	}
	defer reader.Close()

	doc, err := resolveVaultDocument(ctx, reader, query)
	if err != nil {
		return err
	}

	content, err := reader.ReadContent(ctx, doc.ID)
	if err != nil {
		return fmt.Errorf("falha ao ler o conteúdo de '%s': %w", doc.Path, err)
	}

	fmt.Fprintf(out, "Palavras:    %d\n", len(strings.Fields(content)))
	fmt.Fprintf(out, "Caracteres:  %d\n", utf8.RuneCountInString(content))
	return nil
}
