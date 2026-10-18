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
