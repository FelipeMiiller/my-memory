package ci

import (
	"strings"
	"testing"
)

// TestWorkflowCoversEveryPackage e o guard do bloco "Cobertura de pacotes" de
// .github/workflows/ci.yml: nenhum pacote de `go list ./...` pode ficar sem
// step de teste na CI sem estar na tabela de exclusoes.
//
// Sem este teste, remover um step do workflow passava `go build`, `go vet` e a
// suite inteira (mutante M5 do relatorio de validacao da feature
// repo-realignment).
func TestWorkflowCoversEveryPackage(t *testing.T) {
	root, err := repoRoot()
	if err != nil {
		t.Fatalf("localizar raiz do repositorio: %v", err)
	}

	workflow, err := ReadWorkflow(root)
	if err != nil {
		t.Fatalf("ler %s: %v", workflowPath, err)
	}

	pkgs, err := Packages(root)
	if err != nil {
		t.Fatalf("enumerar pacotes: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatalf("`go list ./...` em %s devolveu zero pacotes; o guard nao teria o que checar", root)
	}

	if missing := MissingPackages(workflow, pkgs); len(missing) > 0 {
		t.Errorf("%d pacote(s) de `go list ./...` nao aparece em nenhum step de teste de %s:\n  %s\n"+
			"Adicione um step `go test ./<pacote>/...` ao workflow, ou, se o pacote nao deve rodar na CI, "+
			"declare-o em excludedPackages (internal/ci/ci.go) com o motivo — a tabela hoje e vazia, "+
			"entao qualquer exclusao nova precisa ser explicita.",
			len(missing), workflowPath, strings.Join(missing, "\n  "))
	}
}

// TestExclusionsAreJustifiedAndFresh impede que a tabela de exclusoes vire um
// buraco negro: toda entrada precisa ter motivo e toda entrada precisa
// continuar correspondendo a um pacote que existe.
func TestExclusionsAreJustifiedAndFresh(t *testing.T) {
	for pkg, reason := range excludedPackages {
		if strings.TrimSpace(reason) == "" {
			t.Errorf("exclusao de %q sem motivo declarado em %s", pkg, "internal/ci/ci.go")
		}
	}

	root, err := repoRoot()
	if err != nil {
		t.Fatalf("localizar raiz do repositorio: %v", err)
	}
	pkgs, err := Packages(root)
	if err != nil {
		t.Fatalf("enumerar pacotes: %v", err)
	}
	if stale := StaleExclusions(pkgs); len(stale) > 0 {
		t.Errorf("exclusao obsoleta: %s nao existe mais em `go list ./...`; remova a entrada de excludedPackages",
			strings.Join(stale, ", "))
	}
}

// TestThirdPartyPackagesAreFilteredOut fixa a semantica do filtro de
// `node_modules`: codigo de dependencia npm que distribui arquivos Go nao entra
// na lista de pacotes do repositorio.
//
// O caso que motivou o filtro e real: o pacote npm `flatted` distribui um port
// Go em `golang/pkg/flatted`, e `go list ./...` o reportava como pacote deste
// modulo assim que alguem rodava `npm install` dentro de `viewer/`. Sem o
// filtro, o invariant do CI quebrava para quem developing localmente, e a
// causa nao tinha nada a ver com o workflow.
func TestThirdPartyPackagesAreFilteredOut(t *testing.T) {
	cases := []struct {
		name string
		pkg  string
		want bool
	}{
		{"first-party package", "internal/ci", false},
		{"first-party package at the root of a cmd", "cmd/mem", false},
		{"npm dependency at the root of node_modules", "node_modules/flatted/golang/pkg/flatted", true},
		{"npm dependency nested under a project folder", "viewer/node_modules/flatted/golang/pkg/flatted", true},
		{"similar name that is not a node_modules segment", "internal/node_modules_helper", false},
		{"node_modules as a substring, not a segment", "internal/mynode_modules", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isThirdPartyPkg(tc.pkg); got != tc.want {
				t.Errorf("isThirdPartyPkg(%q) = %v, quer %v", tc.pkg, got, tc.want)
			}
		})
	}
}

// TestPackagesExcludeNodeModules verifica, contra a arvore real, que a lista
// de pacotes devolvida nao contem nenhum caminho sob `node_modules`.
//
// Sem este teste, o filtro poderia regredir em silencio: o invariant do
// workflow continuaria verde enquanto `go list ./...` voltaria a devolver o
// pacote Go da dependencia npm. Em uma maquina sem `npm install` executado, o
// caso nao se manifesta -- por isso o teste acima fixa a semantica e este
// fixa o efeito.
func TestPackagesExcludeNodeModules(t *testing.T) {
	root, err := repoRoot()
	if err != nil {
		t.Fatalf("localizar raiz do repositorio: %v", err)
	}

	pkgs, err := Packages(root)
	if err != nil {
		t.Fatalf("enumerar pacotes: %v", err)
	}
	for _, p := range pkgs {
		if isThirdPartyPkg(p) {
			t.Errorf("pacote de dependencia sob node_modules escapou do filtro: %q", p)
		}
	}
}

// TestWorkflowReferencesIsNotABarePrefixMatch fixa a semantica do matcher: o
// pacote precisa aparecer como alvo de `go test` (exato ou subarvore), e um
// pacote vizinho com prefixo textual nao conta.
func TestWorkflowReferencesIsNotABarePrefixMatch(t *testing.T) {
	cases := []struct {
		name     string
		workflow string
		pkg      string
		want     bool
	}{
		{"exact package", "run: go test -v ./internal/db\n", "internal/db", true},
		{"subtree pattern", "run: go test -v ./internal/graph/...\n", "internal/graph", true},
		{"quoted in coverage step", `  go test ./internal/store/... ./internal/db/...` + "\n", "internal/db", true},
		{"absent", "run: go test -v ./internal/graph/...\n", "internal/supervisor", false},
		{"prefix of a different package only", "run: go test -v ./internal/dbx/...\n", "internal/db", false},
		{"subpath is not the parent", "run: go test -v ./internal/supervisor/child/...\n", "internal/supervisor", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := workflowReferences(tc.workflow, tc.pkg); got != tc.want {
				t.Errorf("workflowReferences(%q, %q) = %v, quer %v", tc.workflow, tc.pkg, got, tc.want)
			}
		})
	}
}
