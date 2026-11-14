package vaultfs

import (
	"os"
	"path/filepath"
	"testing"
)

// buildFixtureVault monta um vault de teste com os casos que a spec exige:
// material bruto em profundidade, frontmatter com title, frontmatter com
// alias, arquivo sem frontmatter, basename duplicado e um title que nao pode
// ser casa por token.
func buildFixtureVault(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	files := map[string]string{
		"notas/fonte.md":                              "---\ntitle: React 19 API Changes\ntags: [react19]\n---\n\ncorpo\n",
		"notas/com-alias.md":                          "---\ntitle: Nota com alias\naliases:\n  - Meu Alias\n  - Outro Alias\n---\n\ncorpo\n",
		"notas/sem-frontmatter.md":                    "corpo sem frontmatter\n",
		"notas/kube.md":                               "---\ntitle: Kubernetes Deployment Patterns\n---\n\ncorpo\n",
		"catalog-skills/skills/x/review-standards.md": "material bruto\n",
		"dup/a/nota.md":                               "primeira\n",
		"dup/b/nota.md":                               "segunda\n",
	}

	for rel, content := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("falha ao criar o diretorio %q: %v", full, err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatalf("falha ao escrever %q: %v", full, err)
		}
	}
	return root
}

func newResolver(t *testing.T, root string) *Resolver {
	t.Helper()
	r, err := NewResolver(root)
	if err != nil {
		t.Fatalf("falha ao criar o resolver do vault de teste: %v", err)
	}
	return r
}

// TestVaultTargetBasenameEmQualquerProfundidade prova que o basename casa
// com o arquivo em qualquer nivel do vault — o caso do material bruto do
// cofre central, que vive em catalog-skills/skills/**.
func TestVaultTargetBasenameEmQualquerProfundidade(t *testing.T) {
	r := newResolver(t, buildFixtureVault(t))

	res := r.Resolve("review-standards")

	if !res.Found {
		t.Fatal("o alvo deveria ter resolvido por basename")
	}
	if res.Path != "catalog-skills/skills/x/review-standards.md" {
		t.Fatalf("caminho esperado %q, obtido %q", "catalog-skills/skills/x/review-standards.md", res.Path)
	}
	if res.Match != MatchBasename {
		t.Fatalf("modo de casamento esperado %q, obtido %q", MatchBasename, res.Match)
	}
}

// TestVaultTargetCaminhoRelativeECaiNoBasename prova as duas metades da
// regra do caminho relativo: com o caminho certain, ele resolve pelo
// caminho; sem ele, cai no basename.
func TestVaultTargetCaminhoRelativeECaiNoBasename(t *testing.T) {
	r := newResolver(t, buildFixtureVault(t))

	comCaminho := r.Resolve("notas/fonte")
	if !comCaminho.Found {
		t.Fatal("o alvo com caminho relativo deveria ter resolvido")
	}
	if comCaminho.Path != "notas/fonte.md" {
		t.Fatalf("caminho esperado %q, obtido %q", "notas/fonte.md", comCaminho.Path)
	}
	if comCaminho.Match != MatchRelPath {
		t.Fatalf("modo de casamento esperado %q, obtido %q", MatchRelPath, comCaminho.Match)
	}

	// Extensao .md no alvo tambem resolve pelo caminho relativo.
	comExtensao := r.Resolve("notas/fonte.md")
	if !comExtensao.Found || comExtensao.Path != "notas/fonte.md" {
		t.Fatalf("alvo com extensao .md nao resolveu pelo caminho: %+v", comExtensao)
	}

	// O caminho nao existe, mas o basename existe em outro lugar.
	semCaminho := r.Resolve("outra-pasta/nota")
	if !semCaminho.Found {
		t.Fatal("o alvo deveria ter caido no basename")
	}
	if semCaminho.Match != MatchBasename {
		t.Fatalf("modo de casamento esperado %q, obtido %q", MatchBasename, semCaminho.Match)
	}
	if semCaminho.Path != "dup/a/nota.md" {
		t.Fatalf("caminho esperado %q, obtido %q", "dup/a/nota.md", semCaminho.Path)
	}
}

// TestVaultTargetResolvePeloTitle prova que o `title:` do frontmatter resolve
// o alvo. No cofre central, 13 dos 125 links que existem em disco so
// resolvem por ele — sem essa regra eles seriam marcados como rompidos e a
// correcao apagaria arestas vivas.
func TestVaultTargetResolvePeloTitle(t *testing.T) {
	r := newResolver(t, buildFixtureVault(t))

	res := r.Resolve("React 19 API Changes")

	if !res.Found {
		t.Fatal("o alvo deveria ter resolvido pelo title do frontmatter")
	}
	if res.Path != "notas/fonte.md" {
		t.Fatalf("caminho esperado %q, obtido %q", "notas/fonte.md", res.Path)
	}
	if res.Match != MatchTitle {
		t.Fatalf("modo de casamento esperado %q, obtido %q", MatchTitle, res.Match)
	}
}

// TestVaultTargetResolvePeloAlias prova que um `alias:` do frontmatter resolve
// o alvo.
func TestVaultTargetResolvePeloAlias(t *testing.T) {
	r := newResolver(t, buildFixtureVault(t))

	res := r.Resolve("Meu Alias")

	if !res.Found {
		t.Fatal("o alvo deveria ter resolvido pelo alias do frontmatter")
	}
	if res.Path != "notas/com-alias.md" {
		t.Fatalf("caminho esperado %q, obtido %q", "notas/com-alias.md", res.Path)
	}
	if res.Match != MatchAlias {
		t.Fatalf("modo de casamento esperado %q, obtido %q", MatchAlias, res.Match)
	}
}

// TestVaultTargetNaoFazMatchFuzzy prova que a resolucao e exata: um token do
// title nao casa. O indexador casa por token, mas a checagem em disco decide
// se um link esta rompido — um match aproximado declararia vivo um link
// quebrado.
func TestVaultTargetNaoFazMatchFuzzy(t *testing.T) {
	r := newResolver(t, buildFixtureVault(t))

	// "Kubernetes" e token do title "Kubernetes Deployment Patterns", mas nao
	// e o title inteiro: nao pode casar.
	fuzzy := r.Resolve("Kubernetes")
	if fuzzy.Found {
		t.Fatalf("o match e exato: %q nao deveria ter casado com %q", "Kubernetes", fuzzy.Path)
	}

	// O title completo resolve — a diferenca e o match exato, nao a ausencia
	// do title no indice.
	exato := r.Resolve("Kubernetes Deployment Patterns")
	if !exato.Found {
		t.Fatal("o title completo deveria ter resolvido")
	}
	if exato.Match != MatchTitle {
		t.Fatalf("modo de casamento esperado %q, obtido %q", MatchTitle, exato.Match)
	}
}

// TestVaultTargetAlvoInexistenteNaoERro prova que um alvo que nao existe
// devolve "nao encontrado" em vez de erro — o doctor precisa listar o link
// rompido, nao quebrar.
func TestVaultTargetAlvoInexistenteNaoERro(t *testing.T) {
	r := newResolver(t, buildFixtureVault(t))

	res := r.Resolve("conceito-que-nao-existe")

	if res.Found {
		t.Fatalf("o alvo nao deveria ter resolvido: %+v", res)
	}
	if res.Path != "" {
		t.Fatalf("o caminho deveria estar vazio, obtido %q", res.Path)
	}
	if res.Match != "" {
		t.Fatalf("o modo de casamento deveria estar vazio, obtido %q", res.Match)
	}
}

// TestVaultTargetAmbiguidadeDevolveUmPath prova que dois arquivos com o
// mesmo basename nao quebram a resolucao: o primeiro (ordem lexica da
// varredura) e devolvido, e o resultado e deterministico.
func TestVaultTargetAmbiguidadeDevolveUmPath(t *testing.T) {
	r := newResolver(t, buildFixtureVault(t))

	res := r.Resolve("nota")

	if !res.Found {
		t.Fatal("o alvo deveria ter resolvido")
	}
	if res.Path != "dup/a/nota.md" {
		t.Fatalf("caminho esperado %q, obtido %q", "dup/a/nota.md", res.Path)
	}

	// A resolucao e deterministica: a segunda chamada devolve o mesmo path.
	if again := r.Resolve("nota"); again.Path != res.Path {
		t.Fatalf("a resolucao nao e deterministica: %q e depois %q", res.Path, again.Path)
	}
}

// TestVaultTargetMatchCaseInsensitive prova que o casamento ignora a caixa:
// o sistema de arquivos do Windows nao distingue "Review-Standards.md" de
// "review-standards.md".
func TestVaultTargetMatchCaseInsensitive(t *testing.T) {
	r := newResolver(t, buildFixtureVault(t))

	res := r.Resolve("REVIEW-STANDARDS")

	if !res.Found {
		t.Fatal("o alvo deveria ter resolvido apesar da caixa")
	}
	if res.Path != "catalog-skills/skills/x/review-standards.md" {
		t.Fatalf("caminho esperado %q, obtido %q", "catalog-skills/skills/x/review-standards.md", res.Path)
	}
}

// TestNewResolverFalhaQuandoARaizNaoElegivel prova que uma raiz inutilizavel
// e recusada na construcao: sem raiz legivel o doctor classifica como "nao
// verificavel", nunca como "rompido".
func TestNewResolverFalhaQuandoARaizNaoElegivel(t *testing.T) {
	if _, err := NewResolver(""); err == nil {
		t.Fatal("uma raiz vazia deveria ter sido recusada")
	}

	inexistente := filepath.Join(t.TempDir(), "vault-que-nao-existe")
	if _, err := NewResolver(inexistente); err == nil {
		t.Fatal("uma raiz inexistente deveria ter sido recusada")
	}

	arquivo := filepath.Join(t.TempDir(), "vault.txt")
	if err := os.WriteFile(arquivo, []byte("nao e um vault"), 0644); err != nil {
		t.Fatalf("falha ao escrever o arquivo de teste: %v", err)
	}
	if _, err := NewResolver(arquivo); err == nil {
		t.Fatal("uma raiz que nao e diretorio deveria ter sido recusada")
	}
}
