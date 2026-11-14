// Package vaultfs resolve a raiz de um vault no disco e os alvos de wikilink
// dentro dele.
//
// O doctor precisa de duas coisas que o índice não tem: ONDE está o vault
// (para checar o disco) e SE o alvo de um link existe como arquivo. Sem a
// checagem em disco, todo link para material fora do escopo de indexação — o
// material bruto do cofre central, por exemplo — vira "link morto" e o Health
// Score mede uma decisão de configuração como se fosse doença.
//
// O pacote é read-only: só lê. Ele não escreve, não cria e não apaga nada.
package vaultfs

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/config"
)

// Origem da raiz resolvida.
const (
	// SourceFlag: a raiz veio da flag explícita da linha de comando.
	SourceFlag = "flag"
	// SourceRepository: a raiz veio do catálogo de repositórios do config global.
	SourceRepository = "repository"
	// SourceCentralVault: a raiz veio de `central_vault.path` do config global.
	SourceCentralVault = "central_vault"
	// SourceCwd: a raiz veio do diretório de trabalho atual.
	SourceCwd = "cwd"
)

// Root é a raiz do vault resolvida e se ela está acessível.
//
// Acessibilidade é a condição para a checagem em disco: sem raiz legível o
// doctor não tem evidência para dizer que um link está rompido, e o classifica
// como não verificável (a suspeita mantém a penalidade, o score fica
// provisório).
type Root struct {
	// Path é a raiz candidata do vault. Vazia quando nada pôde ser resolvido.
	Path string
	// Source diz de onde veio o Path.
	Source string
	// Accessible diz se a raiz existe, é um diretório e pode ser lida.
	Accessible bool
	// Reason explica por que a raiz não está acessível (vazio quando está).
	Reason string
}

// ResolveRoot resolve a raiz do vault que o doctor deve auditar.
//
// Precedência:
//
//  1. flag explícita (--vault)
//  2. entrada do catálogo de repositórios do config global, para o repo auditado
//  3. `central_vault.path` do config global, para o cofre central
//  4. diretório de trabalho atual
//
// O repoID é o repositório que o doctor está auditando. Repo vazio significa
// "vault local" (SQLite), cujo vault é o diretório de trabalho. O repositório
// do catálogo global tem precedência sobre o cofre central: auditar um
// repositório contra os arquivos do cofre central seria medir a coisa errada.
//
// A raiz do cofre central só entra quando o repo auditado é o central — do
// contrário, um `mem doctor --repo repo_xxx` resolveria os links do repo xxx
// contra os arquivos do cofre central e marcaria como rompido link que está
// vivo no repositório.
func ResolveRoot(flagPath, repoID string) Root {
	// 1. Flag explícita: a raiz que o usuário mandou é a que vale.
	if p := strings.TrimSpace(flagPath); p != "" {
		return newRoot(config.ExpandPath(p), SourceFlag)
	}

	// 2 e 3. Config global.
	if global, err := config.LoadGlobalConfig(); err == nil && global != nil {
		if repoID != "" && repoID != config.CentralRepoID {
			for _, entry := range global.Repositories {
				if entry.ID == repoID && strings.TrimSpace(entry.Path) != "" {
					return newRoot(config.ExpandPath(entry.Path), SourceRepository)
				}
			}
		}
		if repoID == config.CentralRepoID && strings.TrimSpace(global.CentralVault.Path) != "" {
			return newRoot(global.CentralVault.Path, SourceCentralVault)
		}
	}

	// 4. Diretório de trabalho atual.
	if cwd, err := os.Getwd(); err == nil {
		return newRoot(cwd, SourceCwd)
	}

	return Root{Reason: "nenhuma raiz de vault disponivel"}
}

// newRoot monta a raiz e verifica se ela está acessível.
func newRoot(path, source string) Root {
	root := Root{Path: path, Source: source}

	if abs, err := filepath.Abs(path); err == nil {
		root.Path = abs
	}

	info, err := os.Stat(root.Path)
	if err != nil {
		root.Reason = "raiz do vault inacessivel: " + err.Error()
		return root
	}
	if !info.IsDir() {
		root.Reason = "a raiz do vault nao e um diretorio: " + root.Path
		return root
	}

	// Abrir o diretorio testa a permissao de leitura sem listar o conteudo.
	f, err := os.Open(root.Path)
	if err != nil {
		root.Reason = "a raiz do vault nao pode ser lida: " + err.Error()
		return root
	}
	_ = f.Close()

	root.Accessible = true
	return root
}
