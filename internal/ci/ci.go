// Package ci protege uma unica invariante: todo pacote Go deste repositorio e
// de fato executado pelo workflow do GitHub Actions em
// .github/workflows/ci.yml.
//
// A invariante esta escrita em prosa dentro do proprio workflow (bloco
// "Cobertura de pacotes"): todo pacote devolvido por `go list ./...` precisa
// aparecer em algum step de teste do arquivo, ou em uma tabela de exclusoes
// explicita. Prosa nao e execucao. Removendo o step de um pacote do workflow,
// `go build`, `go vet` e a suite inteira de testes continuavam verdes -- a
// lacuna de cobertura voltava em silencio, que e exatamente a regressao que a
// feature de realinhamento do repositorio existe para impedir.
//
// Este pacote transforma a prosa em verificacao executavel. A lista de pacotes
// e obtida chamando `go list ./...` de verdade (nunca parseando o workflow ou
// varrendo o filesystem), e cada caminho de pacote precisa aparecer no workflow
// como alvo de `go test`, seja exato ("./internal/db") ou como padrao de
// subarvore ("./internal/graph/...").
//
// Pacotes de dependencia que por acaso moram dentro da arvore de trabalho --
// hoje, codigo Go distribuido por um pacote npm sob `viewer/node_modules/` --
// nao sao pacotes deste repositorio e sao filtrados antes da comparacao (ver
// isThirdPartyPkg).
//
// Um pacote que deliberadamente nao roda na CI precisa estar listado em
// excludedPackages, com o motivo declarado. Hoje a tabela e vazia de proposito:
// nenhum pacote do repositorio fica de fora da CI.
package ci

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// workflowPath is the CI workflow this package guards, relative to repoRoot.
const workflowPath = ".github/workflows/ci.yml"

// excludedPackages is the explicit exclusion table for the CI invariant: a
// package listed here is allowed to have no test step in the workflow.
//
// Format: caminho do pacote relativo a raiz do modulo -> motivo.
//
// A tabela e vazia hoje, e continua declarada no fonte de proposito: a
// proxima exclusao tem que ser um ato explicito e revisavel, nao uma omissao.
// Pacotes como internal/policy, internal/asr, internal/ipc/jsonrpc,
// internal/writer e internal/writer/projection foram removidos na limpeza do
// repositorio; eles nao entram aqui, porque nao aparecem mais em
// `go list ./...`. Se algum voltar a existir, precisa de step de teste de
// volta, nao de exclusao.
var excludedPackages = map[string]string{}

// repoRoot walks up from the directory of this source file looking for go.mod.
//
// It deliberately does not start from the process CWD: `go test` happens to run
// the binary with the package directory as CWD, which works until the test is
// invoked from elsewhere (`go test -run ... ./...` a partir de um diretorio
// acima, um binario de teste executado a mao, um IDE runner).
func repoRoot() (string, error) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("runtime.Caller falhou: nao foi possivel localizar o fonte deste pacote")
	}
	dir := filepath.Dir(thisFile)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod nao encontrado a partir de %s", dir)
		}
		dir = parent
	}
}

// goOutput runs `go` with args at dir and returns stdout trimmed.
func goOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("go %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(out)), nil
}

// Packages returns the repository package paths as they appear in the workflow,
// i.e. relative to the module root and slash-separated ("internal/parser",
// "cmd/mem"), in the order reported by `go list ./...`.
//
// The list comes from `go list ./...`, never from the filesystem or from the
// workflow: the whole point of this package is to compare two independent
// sources of truth.
//
// Packages living under a `node_modules` tree are filtered out. See
// isThirdPartyPkg for why.
func Packages(root string) ([]string, error) {
	module, err := goOutput(root, "list", "-m")
	if err != nil {
		return nil, err
	}
	out, err := goOutput(root, "list", "./...")
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var pkgs []string
	for _, line := range strings.Split(out, "\n") {
		imp := strings.TrimSpace(line)
		if imp == "" {
			continue
		}
		if imp != module {
			if !strings.HasPrefix(imp, module+"/") {
				return nil, fmt.Errorf("pacote %q fora do modulo %q", imp, module)
			}
			imp = strings.TrimPrefix(imp, module+"/")
		}
		if isThirdPartyPkg(imp) {
			continue
		}
		if seen[imp] {
			continue
		}
		seen[imp] = true
		pkgs = append(pkgs, imp)
	}
	return pkgs, nil
}

// isThirdPartyPkg reports whether a package path points inside a
// `node_modules` tree.
//
// `go list ./...` walks the filesystem, so any dependency that ships Go files
// is reported as a package of this module. The npm package `flatted` does
// exactly that: it distributes a Go port under `golang/pkg/flatted`, and it
// lands in `go list ./...` the moment someone runs `npm install` inside
// `viewer/`.
//
// Those packages are third-party code that merely sits inside the working
// tree. They are not part of this repository, the repository does not build
// them on purpose, and whether they exist depends on the npm lockfile rather
// than on the Go source tree.
//
// The invariant guarded by this package is about first-party code ("todo
// pacote Go deste repositorio"), so a `node_modules` path is dropped before
// the comparison. Filtering here rather than listing the package in
// excludedPackages keeps the exclusion table explicit and reviewable: an
// entry there would have to be rewritten whenever the npm tree changes, and
// StaleExclusions would then report it as obsolete.
func isThirdPartyPkg(pkg string) bool {
	for _, segment := range strings.Split(pkg, "/") {
		if segment == "node_modules" {
			return true
		}
	}
	return false
}

// ReadWorkflow returns the content of the CI workflow file.
func ReadWorkflow(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(workflowPath)))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// workflowReferences reports whether workflow passes pkg to a `go test`
// invocation, either as an exact package ("./internal/db") or as a subtree
// pattern ("./internal/graph/...").
//
// A bare prefix match is not enough: "./internal/db" also matches
// "./internal/dbx", which runs a different package.
func workflowReferences(workflow, pkg string) bool {
	needle := "./" + pkg
	for i := 0; i+len(needle) <= len(workflow); {
		j := strings.Index(workflow[i:], needle)
		if j < 0 {
			return false
		}
		i += j + len(needle)
		if i == len(workflow) {
			return true
		}
		switch workflow[i] {
		case ' ', '\t', '\r', '\n', '"', '\'', ',', ':':
			return true
		}
		if strings.HasPrefix(workflow[i:], "/...") {
			return true
		}
	}
	return false
}

// MissingPackages returns the packages of pkgs that no step of workflow runs,
// skipping the ones declared in excludedPackages.
func MissingPackages(workflow string, pkgs []string) []string {
	var missing []string
	for _, p := range pkgs {
		if _, excluded := excludedPackages[p]; excluded {
			continue
		}
		if !workflowReferences(workflow, p) {
			missing = append(missing, p)
		}
	}
	return missing
}

// StaleExclusions returns the entries of excludedPackages that are not in
// pkgs, i.e. exclusions kept for packages that no longer exist.
func StaleExclusions(pkgs []string) []string {
	existing := map[string]bool{}
	for _, p := range pkgs {
		existing[p] = true
	}
	var stale []string
	for p := range excludedPackages {
		if !existing[p] {
			stale = append(stale, p)
		}
	}
	return stale
}
