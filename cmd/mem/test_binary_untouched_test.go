package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// As provas da ISSUE-041: a suíte de `cmd/mem` NÃO pode sobrescrever o
// binário que o dono usa.
//
// O defeito: dois helpers compilavam por cima de `<repo>/bin/mem.exe` com um
// build SEM a tag `onnx`. A suíte derrubava o artefato taggeado, e o próximo
// `mem index` gravava chunk sem vetor no índice do cofre — o Health Score
// caía de 100 para 96 (medido em 2026-11-14).
//
// A correção: o helper compila em `t.TempDir()`. Estes testes são a prova de
// que ela não volta.

// TestMemBinaryNaoCompilaEmBin prova o AC2: o helper devolve um caminho que
// está FORA de `<repo>/bin/`.
func TestMemBinaryNaoCompilaEmBin(t *testing.T) {
	repoRoot := findRepoRoot(t, workingDir(t))
	binDir := filepath.Join(repoRoot, "bin")

	compilado := memBinary(t)

	if dentroDe(compilado, binDir) {
		t.Errorf("o helper compilou dentro de bin/: %s (o binario do usuario nao pode ser efeito colateral da suite)", compilado)
	}
	if !filepath.IsAbs(compilado) {
		t.Errorf("o helper devolveu um caminho relativo: %s", compilado)
	}
	if _, err := os.Stat(compilado); err != nil {
		t.Errorf("o binario compilado nao existe: %v", err)
	}
}

// TestSuitePreservaOBinarioDoUsuario prova o AC3: o `bin/mem.exe` do dono
// fica byte a byte igual depois de a suíte compilar o binário de teste.
//
// Se o arquivo não existir (repo recém-clonado), o teste não tem o que
// comparar — e diz isso, em vez de passar em silêncio.
func TestSuitePreservaOBinarioDoUsuario(t *testing.T) {
	repoRoot := findRepoRoot(t, workingDir(t))
	binPath := filepath.Join(repoRoot, "bin", nomeDoExecutavel())

	hashAntes, existe := hashDoArquivo(t, binPath)
	if !existe {
		t.Logf("%s nao existe (repo recem-clonado): nada a preservar", binPath)
		// Ainda assim o helper tem que funcionar e não escrever em bin/.
		if dentroDe(memBinary(t), filepath.Join(repoRoot, "bin")) {
			t.Fatal("o helper compilou dentro de bin/ mesmo sem binario pre-existente")
		}
		return
	}

	// Compila o binário de teste — exatamente o que a suíte fazia.
	_ = memBinary(t)

	hashDepois, existeDepois := hashDoArquivo(t, binPath)
	if !existeDepois {
		t.Fatalf("a suíte APAGOU o binario do usuario: %s", binPath)
	}
	if hashAntes != hashDepois {
		t.Errorf("a suíte alterou o binario do usuario (%s): %s -> %s", binPath, hashAntes, hashDepois)
	}
}

// TestNenhumFonteDeTesteCompilaParaBin prova o AC1 no longo prazo: nenhum
// arquivo de teste do repositório compila para dentro de `<repo>/bin/`.
//
// É um guard estático sobre os fontes — ele pega o defeito voltando, mesmo que
// alguém introduza um helper novo. O guard não se acusa a si mesmo (por isso
// ele se exclui).
func TestNenhumFonteDeTesteCompilaParaBin(t *testing.T) {
	repoRoot := findRepoRoot(t, workingDir(t))

	var achados []string
	err := filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			base := info.Name()
			if base == ".git" || base == "node_modules" || base == "viewer" || base == "bin" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if strings.EqualFold(filepath.Base(path), "test_binary_untouched_test.go") {
			return nil // este guard procura o padrao; nao se acusa
		}
		conteudo, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		linhas := strings.Split(string(conteudo), "\n")
		for i, linha := range linhas {
			if !strings.Contains(linha, `"go", "build"`) {
				continue
			}
			// Procura o destino do build na janela seguinte.
			fim := i + 12
			if fim > len(linhas) {
				fim = len(linhas)
			}
			janela := strings.Join(linhas[i:fim], "\n")
			if strings.Contains(janela, `"bin"`) {
				rel, _ := filepath.Rel(repoRoot, path)
				achados = append(achados, fmt.Sprintf("%s:%d compila para dentro de bin/", rel, i+1))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("varrendo os fontes de teste: %v", err)
	}

	for _, achado := range achados {
		t.Errorf("%s", achado)
	}
	if len(achados) == 0 {
		t.Log("nenhum fonte de teste compila para dentro de bin/")
	}
}

// nomeDoExecutavel devolve o nome do executável do mem na plataforma.
func nomeDoExecutavel() string {
	if runtime.GOOS == "windows" {
		return "mem.exe"
	}
	return "mem"
}

// workingDir devolve o diretório de trabalho atual do teste.
func workingDir(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("cwd: %v", err)
	}
	return cwd
}

// hashDoArquivo devolve o SHA-256 do arquivo e se ele existe.
func hashDoArquivo(t *testing.T, path string) (string, bool) {
	t.Helper()
	conteudo, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false
		}
		t.Fatalf("lendo %s: %v", path, err)
	}
	sum := sha256.Sum256(conteudo)
	return hex.EncodeToString(sum[:]), true
}

// dentroDe diz se o caminho está dentro do diretório (ou é ele mesmo).
func dentroDe(caminho, dir string) bool {
	rel, err := filepath.Rel(dir, caminho)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
