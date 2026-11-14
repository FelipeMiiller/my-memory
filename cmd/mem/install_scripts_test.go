package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// repoRootDoTeste sobe a partir do fonte deste arquivo procurando o go.mod.
//
// Não parte do CWD: `go test` roda o binário com o diretório do pacote como
// CWD, o que funciona até o teste ser invocado de outro lugar.
func repoRootDoTeste(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller falhou: não foi possível localizar o fonte deste teste")
	}
	dir := filepath.Dir(thisFile)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("não foi possível encontrar o go.mod subindo a árvore")
		}
		dir = parent
	}
}

// TestInstaladoresImprimemODiretorioDoModeloUsado fixa a AC7: o instalador
// tem que imprimir o diretório que o `mem embed download` REALMENTE usou.
//
// O downloader respeita `MY_MEMORY_MINILM_MODEL_DIR` (o mesmo override que o
// provider embutido lê). Um instalador que imprime um caminho fixo mente
// sobre onde o modelo ficou — e foi exatamente isso que a AC7 reprovou na
// rodada 1.
//
// O instalador não é unit-testável (ele baixa, instala e mexe no PATH), então
// a asserção é sobre o FONTE do script. Ela é dura contra a regressão que
// importa: voltar a imprimir o literal. O caminho default pode — e deve —
// existir como fallback da variável de ambiente; o que não pode é estar
// dentro da linha que imprime.
func TestInstaladoresImprimemODiretorioDoModeloUsado(t *testing.T) {
	root := repoRootDoTeste(t)
	const idModelo = "paraphrase-multilingual-MiniLM-L12-v2"

	for _, nome := range []string{"install.ps1", "install.sh"} {
		caminho := filepath.Join(root, "scripts", nome)
		conteudo, err := os.ReadFile(caminho)
		if err != nil {
			t.Fatalf("lendo %s: %v", caminho, err)
		}
		fonte := string(conteudo)

		if !strings.Contains(fonte, "MY_MEMORY_MINILM_MODEL_DIR") {
			t.Errorf("%s não consulta MY_MEMORY_MINILM_MODEL_DIR: o diretório impresso pode não ser o que o download usou", nome)
		}

		var linhaDaImpressao string
		for _, linha := range strings.Split(fonte, "\n") {
			if strings.Contains(linha, "Diretório do modelo") {
				linhaDaImpressao = linha
				break
			}
		}
		if linhaDaImpressao == "" {
			t.Errorf("%s não imprime o diretório do modelo", nome)
			continue
		}
		if strings.Contains(linhaDaImpressao, idModelo) {
			t.Errorf("%s imprime o caminho do modelo como literal, e não o diretório que o download usou:\n  %s", nome, strings.TrimSpace(linhaDaImpressao))
		}
	}
}
