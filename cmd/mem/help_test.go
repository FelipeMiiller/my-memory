package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// caseRE casa um `case "x":` (ou `case "x", "y":`) de primeiro nível.
var caseRE = regexp.MustCompile(`^` + "\t" + `case "([^"]+)"`)

// dispatchedSubcommands extrai, do próprio código-fonte de main.go, os
// subcomandos registrados no switch de dispatch `switch os.Args[1]`.
//
// Por que parsear o fonte em vez de manter uma lista à mão no teste: a lista
// manual passariaverde mesmo quando alguém adiciona `case "novo":` e esquece
// de documentar. O objetivo deste teste é exactamente esse — travar a
// sincronia entre o dispatch e a ajuda. A fonte é a única verdade, então é
// ela que o teste lê.
//
// O scan para na primeira linha que seja só `}` com um tab de indentação, que
// é o fecha-chaves do switch. Assim os `case "fts":` / `case "vector":` que
// existem mais abaixo (modos de busca, dentro de outras funções) ficam fora.
func dispatchedSubcommands(t *testing.T) []string {
	t.Helper()

	path := filepath.Join("main.go")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("não consegui ler %s: %v", path, err)
	}

	lines := strings.Split(string(src), "\n")

	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "switch os.Args[1] {" {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatal("não achei `switch os.Args[1]` em main.go — o teste precisa ser atualizado")
	}

	var subs []string
	seen := map[string]bool{}
	for _, raw := range lines[start+1:] {
		line := strings.TrimRight(raw, "\r")
		// Só o fecha-chaves do próprio switch interrompe o scan — UM tab.
		// Comparar com TrimSpace não serviria: ele apaga a indentação e faria
		// o primeiro `}` aninhado dentro de um case (indentado com dois tabs)
		// encerrar a varredura depois do primeiro comando.
		if line == "\t}" {
			break
		}
		m := caseRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name := m[1]
		// `case "version", "--version", "-v":` documenta o subcomando
		// `version`; as formas com traço são aliases do mesmo comando.
		if seen[name] {
			continue
		}
		seen[name] = true
		subs = append(subs, name)
	}

	if len(subs) == 0 {
		t.Fatal("nenhum subcomando extraído do switch de dispatch — o parser quebrou")
	}
	return subs
}

// capturePrintHelp roda printHelp() com os stdout capturados e devolve o texto.
func capturePrintHelp(t *testing.T) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe(): %v", err)
	}

	orig := os.Stdout
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	printHelp()

	os.Stdout = orig
	_ = w.Close()
	out := <-done
	_ = r.Close()

	return out
}

// TestPrintHelp_CobreTodoSubcommandDoDispatch é o teste que transforma a
// afirmação "a ajuda lista todos os comandos" em algo verificado pelo CI.
//
// Sem ele, `printHelp()` encolheu por anos enquanto o switch crescia
// (`up`, `down`, `logs`, `profiles`, `uninstall` chegaram a ser
// invisíveis para quem chamava `mem --help`).
func TestPrintHelp_CobreTodoSubcommandDoDispatch(t *testing.T) {
	help := capturePrintHelp(t)
	subs := dispatchedSubcommands(t)

	if len(subs) < 30 {
		t.Fatalf("esperava dozens de subcomandos no dispatch, obtive %d — o parser de main.go está errado", len(subs))
	}

	var missing []string
	for _, sub := range subs {
		if !strings.Contains(help, "mem "+sub+" ") && !strings.Contains(help, "mem "+sub+"\n") {
			missing = append(missing, sub)
		}
	}

	if len(missing) > 0 {
		t.Errorf("subcomandos registrados no dispatch mas ausentes de printHelp(): %v\n\n"+
			"Adicione uma linha `mem <comando> [flags]` e sua descrição em printHelp() (cmd/mem/main.go).",
			missing)
	}
}

// specParityCommands são os nove comandos de leitura que a spec
// .specs/features/obsidian-parity-foundation entregou.
var specParityCommands = []string{
	"files", "folders", "read", "outline", "wordcount",
	"random", "deadends", "file", "recents",
}

// TestPrintHelp_CobreOsComandosDaSpecObsidian fecha o outro lado da guarda.
//
// O teste de sincronia acima prova que o que está no dispatch está na ajuda. O
// inverso — apagar a linha do help E o case do dispatch juntos — passa nele sem
// que ninguém perceba: os dois caminhos desaparecem juntos e o vault perde um
// comando inteiro. Aqui a lista vem da spec, não do código, então remover um
// desses nove comandos quebra o teste.
func TestPrintHelp_CobreOsComandosDaSpecObsidian(t *testing.T) {
	help := capturePrintHelp(t)
	dispatched := map[string]bool{}
	for _, sub := range dispatchedSubcommands(t) {
		dispatched[sub] = true
	}

	for _, sub := range specParityCommands {
		if !dispatched[sub] {
			t.Errorf("comando %q da spec de paridade não está no switch de dispatch de main.go", sub)
		}
		if !strings.Contains(help, "mem "+sub+" ") && !strings.Contains(help, "mem "+sub+"\n") {
			t.Errorf("comando %q da spec de paridade não está em printHelp()", sub)
		}
	}
}

// TestPrintHelp_NaoListaComandoInexistente é o contrapeso: a ajuda não pode
// inventar comando. Pega o erro simétrico — documentar algo que o dispatch
// não roteia — que o teste anterior, por ser só de "faltando", deixaria passar.
func TestPrintHelp_NaoListaComandoInexistente(t *testing.T) {
	help := capturePrintHelp(t)
	dispatched := map[string]bool{}
	for _, sub := range dispatchedSubcommands(t) {
		dispatched[sub] = true
	}

	re := regexp.MustCompile(`(?m)^\s*mem ([a-z][a-z-]*)`)
	var ghosts []string
	for _, m := range re.FindAllStringSubmatch(help, -1) {
		name := m[1]
		if !dispatched[name] {
			ghosts = append(ghosts, name)
		}
	}

	if len(ghosts) > 0 {
		t.Errorf("printHelp() documenta comandos que o switch de dispatch não roteia: %v", ghosts)
	}
}
