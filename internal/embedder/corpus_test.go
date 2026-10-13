package embedder

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

// CorpusPair é um par de frases do corpus de benchmark semântico.
type CorpusPair struct {
	Class string // "sin" (paráfrase) ou "dis" (topicamente distante)
	A     string
	B     string
}

const (
	classSynonym     = "sin"
	classDissimilar  = "dis"
	corpusFile       = "benchmark_corpus_testdata.txt"
	minPairsPerClass = 25
)

// loadCorpus lê o corpus de benchmark. Formato de cada linha útil:
//
//	<classe>\t<frase A>\t<frase B>
//
// Linhas vazias e comentários (#) são ignoradas.
func loadCorpus(t *testing.T) []CorpusPair {
	t.Helper()
	f, err := os.Open(corpusFile)
	if err != nil {
		t.Fatalf("nao foi possivel abrir %s: %v", corpusFile, err)
	}
	defer f.Close()

	var pairs []CorpusPair
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	line := 0
	for sc.Scan() {
		line++
		raw := strings.TrimSpace(sc.Text())
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		parts := strings.Split(raw, "\t")
		if len(parts) != 3 {
			t.Fatalf("linha %d: esperado 3 campos separados por TAB, veio %d", line, len(parts))
		}
		class := strings.TrimSpace(parts[0])
		if class != classSynonym && class != classDissimilar {
			t.Fatalf("linha %d: classe invalida %q (esperado %q ou %q)", line, class, classSynonym, classDissimilar)
		}
		a := strings.TrimSpace(parts[1])
		b := strings.TrimSpace(parts[2])
		if a == "" || b == "" {
			t.Fatalf("linha %d: frase vazia", line)
		}
		if a == b {
			t.Fatalf("linha %d: as duas frases sao identicas (%q) — isso nao mede nada", line, a)
		}
		pairs = append(pairs, CorpusPair{Class: class, A: a, B: b})
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("erro de leitura: %v", err)
	}
	return pairs
}

func TestCorpusLoads(t *testing.T) {
	pairs := loadCorpus(t)

	if len(pairs) == 0 {
		t.Fatal("corpus vazio")
	}

	counts := map[string]int{}
	seen := map[string]string{}
	for i, p := range pairs {
		counts[p.Class]++
		key := p.A + "\x00" + p.B
		if prev, dup := seen[key]; dup {
			t.Errorf("par duplicado (pos %d): %q | %q  (ja estava em %q)", i, p.A, p.B, prev)
		}
		seen[key] = p.A
	}

	for _, c := range []string{classSynonym, classDissimilar} {
		if counts[c] < minPairsPerClass {
			t.Errorf("classe %q tem %d pares, minimo exigido %d", c, counts[c], minPairsPerClass)
		}
	}
	t.Logf("corpus carregado: %d sin + %d dis = %d pares", counts[classSynonym], counts[classDissimilar], len(pairs))
}

// O corpus só vale se exercitar português de verdade. Um corpus todo-ASCII
// pode passar por engano e a medição perde o sentido.
//
// A exigência é de nível CORPUS, não de frase: português correto tem frases
// legítimas sem acento nenhum ("Como redefinir uma senha que esqueci?"). O que
// não pode é um corpus onde os diacríticos são exceção.
func TestCorpusExercitaPortugues(t *testing.T) {
	pairs := loadCorpus(t)
	if len(pairs) == 0 {
		t.Fatal("corpus vazio")
	}

	const minRatioComDiacritico = 0.80
	const minParesComDiacritico = 40

	var comDiacritico int
	for _, p := range pairs {
		frase := p.A + " " + p.B
		if strings.ContainsAny(frase, "áàâãéêíóôõúüçÁÀÂÃÉÊÍÓÔÕÚÇ") {
			comDiacritico++
		}
	}

	ratio := float64(comDiacritico) / float64(len(pairs))
	t.Logf("%d/%d pares com diacriticos (%.0f%%)", comDiacritico, len(pairs), ratio*100)

	if comDiacritico < minParesComDiacritico {
		t.Errorf("corpus com poucos diacriticos: %d pares (minimo %d). "+
			"Um corpus ASCII nao mede semantica em portugues.", comDiacritico, minParesComDiacritico)
	}
	if ratio < minRatioComDiacritico {
		t.Errorf("apenas %.0f%% dos pares tem diacriticos (minimo %.0f%%)", ratio*100, minRatioComDiacritico*100)
	}
}

// Guarda contra o erro que invalida a medicao inteira: um par marcado como
// "dissimilar" que compartilha vocabulario com o outro lado nao mede nada.
func TestCorpusDissimilarNaoCompartilhaVocabulario(t *testing.T) {
	pairs := loadCorpus(t)

	// Palavras funcionais do português: carregam gramática, não tópico.
	// Um par "Como faço X?" × "Como faço Y?" não é dissimilar por natureza.
	stop := map[string]bool{
		// artigos, preposições, conjunções, pronomes
		"a": true, "o": true, "as": true, "os": true, "um": true, "uma": true,
		"de": true, "da": true, "do": true, "das": true, "dos": true,
		"em": true, "no": true, "na": true, "nos": true, "nas": true,
		"para": true, "por": true, "com": true, "sem": true, "ate": true, "apos": true,
		"que": true, "e": true, "ou": true, "mas": true, "se": true, "como": true,
		"quando": true, "onde": true, "qual": true, "quais": true, "porque": true, "porem": true,
		"eu": true, "meu": true, "minha": true, "me": true, "seu": true, "sua": true,
		"mesma": true, "mesmo": true, "outro": true, "outra": true,
		"cada": true, "todo": true, "toda": true, "todos": true, "todas": true,
		"ao": true, "aos": true, "pela": true, "pelo": true, "dele": true, "dela": true,
		// advérbios e quantificadores sem carga temática
		"nao": true, "sim": true, "mais": true, "menos": true, "muito": true,
		"tambem": true, "entao": true, "depois": true, "antes": true, "ja": true,
		"sempre": true, "nunca": true, "agora": true, "bem": true, "mal": true,
		"so": true, "aqui": true, "la": true, "ali": true,
	}

	para := func(s string) map[string]bool {
		out := map[string]bool{}
		for _, w := range strings.Fields(strings.ToLower(s)) {
			w = strings.Trim(w, ".,;:!?()")
			if len(w) > 3 && !stop[w] {
				out[w] = true
			}
		}
		return out
	}

	for i, p := range pairs {
		if p.Class != classDissimilar {
			continue
		}
		va, vb := para(p.A), para(p.B)
		for w := range va {
			if vb[w] {
				t.Errorf("par dissimilar %d compartilha a palavra %q entre os lados: %q | %q", i, w, p.A, p.B)
			}
		}
	}
}
