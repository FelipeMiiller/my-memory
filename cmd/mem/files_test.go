package main

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/db"
)

// newVaultFixture cria um SQLite de vault com os documentos informados e
// devolve o caminho do arquivo. paths são gravados exatamente como vierem, para
// que os testes possam exercitar tanto caminho relativo ao vault quanto
// absoluto — os dois existem em vaults indexados de jeitos diferentes.
func newVaultFixture(t *testing.T, paths ...string) string {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "vault_fixture.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	for _, path := range paths {
		err := db.InsertDocumentWithMeta(ctx, database, path, path, "titulo", 1700000000, "hash-"+path, "epoch", "", "resource", 1700000000, 128)
		if err != nil {
			t.Fatalf("InsertDocumentWithMeta(%s) falhou: %v", path, err)
		}
	}
	return dbPath
}

// runVaultFixture executa um comando de leitura do vault contra o fixture e
// devolve a saída e o erro.
func runVaultFixture(t *testing.T, dbPath string, args []string) (string, error) {
	t.Helper()

	full := append([]string{"--db", dbPath, "--storage", "sqlite"}, args...)
	var buf bytes.Buffer
	err := runFilesCommand(context.Background(), "fixture-repo", full, &buf)
	return buf.String(), err
}

func TestRunFilesCommand_ListaAlfabetico(t *testing.T) {
	dbPath := newVaultFixture(t, "notes/zeta.md", "docs/alpha.md", "docs/beta.md")

	out, err := runVaultFixture(t, dbPath, nil)
	if err != nil {
		t.Fatalf("runFilesCommand falhou: %v", err)
	}

	got := strings.Fields(strings.TrimSpace(out))
	want := []string{"docs/alpha.md", "docs/beta.md", "notes/zeta.md"}
	if len(got) != len(want) {
		t.Fatalf("esperava %d caminhos, obteve %d:\n%s", len(want), len(got), out)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("posição %d: esperava %q, obteve %q (ordem alfabética por path)", i, want[i], got[i])
		}
	}
}

func TestRunFilesCommand_FiltraExtensao(t *testing.T) {
	dbPath := newVaultFixture(t, "docs/alpha.md", "assets/diagrama.canvas", "notes/beta.md")

	out, err := runVaultFixture(t, dbPath, []string{"--ext", "md"})
	if err != nil {
		t.Fatalf("runFilesCommand --ext md falhou: %v", err)
	}
	if !strings.Contains(out, "docs/alpha.md") || !strings.Contains(out, "notes/beta.md") {
		t.Errorf("esperava as duas notas .md, obteve:\n%s", out)
	}
	if strings.Contains(out, "diagrama.canvas") {
		t.Errorf("filtro --ext md vazou um .canvas:\n%s", out)
	}

	// ".md" é o mesmo pedido que "md".
	outDot, err := runVaultFixture(t, dbPath, []string{"--ext", ".md"})
	if err != nil {
		t.Fatalf("runFilesCommand --ext .md falhou: %v", err)
	}
	if outDot != out {
		t.Errorf("--ext .md e --ext md divergiram:\n%q\nvs\n%q", outDot, out)
	}
}

// TestRunFilesCommand_ExtDesconhecidaNaoErra é a regra do Obsidian: filtro que
// não casa nada é uma lista vazia, não um erro. Um exit non-zero aqui faria
// `mem files --ext $X` num script quebrar por um motivo que não é falha.
func TestRunFilesCommand_ExtDesconhecidaNaoErra(t *testing.T) {
	dbPath := newVaultFixture(t, "docs/alpha.md", "notes/beta.md")

	out, err := runVaultFixture(t, dbPath, []string{"--ext", "xyz"})
	if err != nil {
		t.Fatalf("extensão desconhecida deveria sair zero, obteve erro: %v", err)
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("extensão desconhecida deveria imprimir lista vazia, obteve:\n%s", out)
	}
}

// TestRunFilesCommand_PastaDesconhecidaNaoErra é o mesmo contrato para
// --folder: prefixo que não existe no vault responde vazio e sai zero.
func TestRunFilesCommand_PastaDesconhecidaNaoErra(t *testing.T) {
	dbPath := newVaultFixture(t, "docs/alpha.md", "notes/beta.md")

	out, err := runVaultFixture(t, dbPath, []string{"--folder", "nope"})
	if err != nil {
		t.Fatalf("pasta desconhecida deveria sair zero, obteve erro: %v", err)
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("pasta desconhecida deveria imprimir lista vazia, obteve:\n%s", out)
	}
}

func TestRunFilesCommand_FiltraPastaPorSegmento(t *testing.T) {
	dbPath := newVaultFixture(t, "docs/alpha.md", "docs/notes/deep.md", "docs-antigos/old.md")

	out, err := runVaultFixture(t, dbPath, []string{"--folder", "docs"})
	if err != nil {
		t.Fatalf("runFilesCommand --folder docs falhou: %v", err)
	}
	if !strings.Contains(out, "docs/alpha.md") || !strings.Contains(out, "docs/notes/deep.md") {
		t.Errorf("esperava os documentos dentro de docs/, obteve:\n%s", out)
	}
	if strings.Contains(out, "docs-antigos/old.md") {
		t.Errorf("docs não pode casar docs-antigos (fronteira de segmento), obteve:\n%s", out)
	}
}

// TestMatchesFolder_CaminhoAbsolinoOuRelativo cobre a realidade do índice:
// documents.path pode ter sido gravado absoluto (o walker percorre o caminho
// completo) ou relativo ao vault, e o filtro precisa funcionar nos dois.
func TestMatchesFolder_CaminhoAbsolutoOuRelativo(t *testing.T) {
	cases := []struct {
		path   string
		folder string
		want   bool
	}{
		{"docs/alpha.md", "docs", true},
		{"docs/notes/deep.md", "docs", true},
		{"C:/vault/docs/alpha.md", "docs", true},
		{"C:\\vault\\docs\\alpha.md", "docs", true},
		{"docs-antigos/old.md", "docs", false},
		{"notes/beta.md", "docs", false},
		{"notes/beta.md", "", true},
		{"notes/beta.md", "notes/", true},
	}

	for _, tc := range cases {
		if got := matchesFolder(tc.path, tc.folder); got != tc.want {
			t.Errorf("matchesFolder(%q, %q) = %v, esperava %v", tc.path, tc.folder, got, tc.want)
		}
	}
}

func TestRunFoldersCommand_ListaPastasDistintas(t *testing.T) {
	dbPath := newVaultFixture(t, "docs/alpha.md", "docs/beta.md", "notes/zeta.md", "raiz.md")

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--storage", "sqlite"}
	if err := runFoldersCommand(context.Background(), "fixture-repo", args, &buf); err != nil {
		t.Fatalf("runFoldersCommand falhou: %v", err)
	}

	got := strings.Fields(strings.TrimSpace(buf.String()))
	want := []string{"docs", "notes"}
	if len(got) != len(want) {
		t.Fatalf("esperava %v, obteve %v (documento na raiz do vault não é pasta)", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("posição %d: esperava %q, obteve %q", i, want[i], got[i])
		}
	}
}

func TestRunFoldersCommand_VazioNaoErra(t *testing.T) {
	dbPath := newVaultFixture(t)

	var buf bytes.Buffer
	args := []string{"--db", dbPath, "--storage", "sqlite"}
	if err := runFoldersCommand(context.Background(), "fixture-repo", args, &buf); err != nil {
		t.Fatalf("vault sem documentos deveria sair zero, obteve erro: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "" {
		t.Errorf("vault sem documentos deveria imprimir nada, obteve:\n%s", buf.String())
	}
}
