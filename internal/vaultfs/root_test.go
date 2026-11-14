package vaultfs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/config"
)

// writeGlobalConfig isola o config global num diretorio temporario e escreve
// o conteudo declarado. Devolve o diretorio do config.
func writeGlobalConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv(config.GlobalConfigDirEnv, dir)
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatalf("falha ao escrever o config global de teste: %v", err)
	}
	return dir
}

// samePath compara dois caminhos ignorando separador e barra final.
func samePath(a, b string) bool {
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	return filepath.Clean(absA) == filepath.Clean(absB)
}

// TestVaultRootFlagVenceConfigGlobal prova a primeira regra da precedencia:
// a flag explicita manda na raiz, mesmo com o config global declarando outro
// vault.
func TestVaultRootFlagVenceConfigGlobal(t *testing.T) {
	centralDir := t.TempDir()
	writeGlobalConfig(t, "central_vault:\n  path: '"+filepath.ToSlash(centralDir)+"'\n")
	flagDir := t.TempDir()

	root := ResolveRoot(flagDir, config.CentralRepoID)

	if root.Source != SourceFlag {
		t.Fatalf("origem esperada %q, obtida %q (a flag tem precedencia sobre o config global)", SourceFlag, root.Source)
	}
	if !samePath(root.Path, flagDir) {
		t.Fatalf("raiz esperada %q, obtida %q", flagDir, root.Path)
	}
	if !root.Accessible {
		t.Fatalf("a raiz %q existe e deveria estar acessivel: %s", root.Path, root.Reason)
	}
}

// TestVaultRootConfigGlobalVenceCwd prova a segunda regra: sem flag, o
// `central_vault.path` do config global vale para o cofre central.
func TestVaultRootConfigGlobalVenceCwd(t *testing.T) {
	centralDir := t.TempDir()
	writeGlobalConfig(t, "central_vault:\n  path: '"+filepath.ToSlash(centralDir)+"'\n")

	root := ResolveRoot("", config.CentralRepoID)

	if root.Source != SourceCentralVault {
		t.Fatalf("origem esperada %q, obtida %q", SourceCentralVault, root.Source)
	}
	if !samePath(root.Path, centralDir) {
		t.Fatalf("raiz esperada %q, obtida %q", centralDir, root.Path)
	}
}

// TestVaultRootCwdQuandoNaoHaConfigGlobal prova a ultima regra: sem flag e
// sem config global, a raiz e o diretorio de trabalho atual.
func TestVaultRootCwdQuandoNaoHaConfigGlobal(t *testing.T) {
	writeGlobalConfig(t, "")
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("nao foi possivel ler o diretorio de trabalho: %v", err)
	}

	root := ResolveRoot("", "")

	if root.Source != SourceCwd {
		t.Fatalf("origem esperada %q, obtida %q", SourceCwd, root.Source)
	}
	if !samePath(root.Path, cwd) {
		t.Fatalf("raiz esperada %q, obtida %q", cwd, root.Path)
	}
}

// TestVaultRootUsaCatalogoDoRepositorio prova que auditar um repositorio do
// catalogo resolve a raiz DELE, e nao a do cofre central: auditar um repo
// contra os arquivos de outro vault mede a coisa errada.
func TestVaultRootUsaCatalogoDoRepositorio(t *testing.T) {
	repoDir := t.TempDir()
	centralDir := t.TempDir()
	writeGlobalConfig(t,
		"repositories:\n  - id: repo_abc123456789\n    path: '"+filepath.ToSlash(repoDir)+"'\n"+
			"central_vault:\n  path: '"+filepath.ToSlash(centralDir)+"'\n")

	root := ResolveRoot("", "repo_abc123456789")

	if root.Source != SourceRepository {
		t.Fatalf("origem esperada %q, obtida %q", SourceRepository, root.Source)
	}
	if !samePath(root.Path, repoDir) {
		t.Fatalf("raiz esperada %q, obtida %q", repoDir, root.Path)
	}
}

// TestVaultRootCofreCentralVenceCatalogo prova que o cofre central nao e
// contaminado pelo catalogo: o `central_vault.path` responde pelo central.
func TestVaultRootCofreCentralVenceCatalogo(t *testing.T) {
	repoDir := t.TempDir()
	centralDir := t.TempDir()
	writeGlobalConfig(t,
		"repositories:\n  - id: repo_abc123456789\n    path: '"+filepath.ToSlash(repoDir)+"'\n"+
			"central_vault:\n  path: '"+filepath.ToSlash(centralDir)+"'\n")

	root := ResolveRoot("", config.CentralRepoID)

	if root.Source != SourceCentralVault {
		t.Fatalf("origem esperada %q, obtida %q", SourceCentralVault, root.Source)
	}
	if !samePath(root.Path, centralDir) {
		t.Fatalf("raiz esperada %q, obtida %q", centralDir, root.Path)
	}
}

// TestVaultRootInacessivelQuandoNaoExiste prova que uma raiz inexistente e
// reportada como nao acessivel, com razao — sem ela o doctor nao saberia
// distinguir "nao ha vault" de "o vault esta vazio".
func TestVaultRootInacessivelQuandoNaoExiste(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "vault-que-nao-existe")

	root := ResolveRoot(missing, "")

	if root.Accessible {
		t.Fatalf("a raiz %q nao existe e nao deveria estar acessivel", root.Path)
	}
	if root.Reason == "" {
		t.Fatal("a razao da inacessibilidade e obrigatoria")
	}
}

// TestVaultRootInacessivelQuandoNaoEDiretorio prova que um arquivo no lugar
// da raiz tambem e inacessivel.
func TestVaultRootInacessivelQuandoNaoEDiretorio(t *testing.T) {
	file := filepath.Join(t.TempDir(), "vault.txt")
	if err := os.WriteFile(file, []byte("nao e um vault"), 0644); err != nil {
		t.Fatalf("falha ao escrever o arquivo de teste: %v", err)
	}

	root := ResolveRoot(file, "")

	if root.Accessible {
		t.Fatalf("a raiz %q e um arquivo e nao deveria estar acessivel", root.Path)
	}
	if root.Reason == "" {
		t.Fatal("a razao da inacessibilidade e obrigatoria")
	}
}

// TestVaultRootNaoEscreveNaRaiz prova que a resolucao e read-only: nada e
// criado dentro do vault resolvido.
func TestVaultRootNaoEscreveNaRaiz(t *testing.T) {
	dir := t.TempDir()
	before, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("falha ao listar o vault de teste: %v", err)
	}

	ResolveRoot(dir, "")

	after, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("falha ao listar o vault de teste: %v", err)
	}
	if len(after) != len(before) {
		t.Fatalf("a resolucao da raiz escreveu no vault: %d entradas antes, %d depois", len(before), len(after))
	}
}
