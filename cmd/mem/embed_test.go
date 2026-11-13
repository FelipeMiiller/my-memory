package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/embedder"
)

// manifestDeTeste monta um manifesto com tudo dentro de um diretório temp.
func manifestDeTeste(t *testing.T) (embedder.Manifest, string) {
	t.Helper()
	dir := t.TempDir()
	modelDir := filepath.Join(dir, "model")
	ortDir := filepath.Join(dir, "onnxruntime")
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		t.Fatalf("criar o diretório do modelo: %v", err)
	}
	if err := os.MkdirAll(ortDir, 0o755); err != nil {
		t.Fatalf("criar o diretório do runtime: %v", err)
	}

	return embedder.Manifest{
		Model: embedder.Artifact{
			URL:  "https://exemplo.invalido/model_int8.onnx",
			Dest: filepath.Join(modelDir, "model_int8.onnx"),
		},
		Tokenizer: embedder.Artifact{
			URL:  "https://exemplo.invalido/sentencepiece.bpe.model",
			Dest: filepath.Join(modelDir, "sentencepiece.bpe.model"),
		},
		Runtime: embedder.RuntimeArchive{
			URL:    "https://exemplo.invalido/onnxruntime.zip",
			Dest:   filepath.Join(ortDir, embedder.ORTLibName()),
			Member: embedder.ORTLibName(),
		},
	}, dir
}

// O doctor tem que relatar o provider ativo, a dimensão, cada caminho e se
// cada artefato existe no disco.
func TestEmbedDoctorRelogaProviderEArtefatos(t *testing.T) {
	manifest, _ := manifestDeTeste(t)

	for _, a := range []embedder.Artifact{manifest.Model, manifest.Tokenizer} {
		if err := os.WriteFile(a.Dest, []byte("conteudo"), 0o644); err != nil {
			t.Fatalf("preparar %s: %v", a.Dest, err)
		}
	}
	if err := os.WriteFile(manifest.Runtime.Dest, []byte("runtime"), 0o644); err != nil {
		t.Fatalf("preparar %s: %v", manifest.Runtime.Dest, err)
	}

	var out bytes.Buffer
	err := runEmbedDoctor(&out, manifest, func() error { return nil })
	if err != nil {
		t.Fatalf("runEmbedDoctor: %v", err)
	}
	relatorio := out.String()

	esperado := []string{
		"builtin",               // o provider em uso
		"384",                   // a dimensão do vetor
		manifest.Model.Dest,     // o caminho do modelo
		manifest.Tokenizer.Dest, // o caminho do tokenizer
		manifest.Runtime.Dest,   // o caminho da biblioteca do ONNX Runtime
		"ATIVO",                 // o status do provider
	}
	for _, want := range esperado {
		if !strings.Contains(relatorio, want) {
			t.Errorf("o relatório não menciona %q:\n%s", want, relatorio)
		}
	}
	for _, a := range []string{manifest.Model.Dest, manifest.Tokenizer.Dest, manifest.Runtime.Dest} {
		if !strings.Contains(relatorio, a+" — presente") {
			t.Errorf("o relatório não diz que %s está presente:\n%s", a, relatorio)
		}
	}
}

// Sem provider o doctor RELATA e não quebra: ele devolve nil e diz o motivo.
func TestEmbedDoctorSemProviderReportaIndisponivel(t *testing.T) {
	manifest, _ := manifestDeTeste(t)

	var out bytes.Buffer
	err := runEmbedDoctor(&out, manifest, func() error {
		return errors.New("provider builtin indisponível: este binário foi compilado sem a tag `onnx`")
	})
	if err != nil {
		t.Fatalf("runEmbedDoctor não deveria falhar sem provider: %v", err)
	}
	relatorio := out.String()

	if !strings.Contains(relatorio, "INDISPONÍVEL") {
		t.Errorf("o relatório não diz que o provider está indisponível:\n%s", relatorio)
	}
	if !strings.Contains(relatorio, "tag `onnx`") {
		t.Errorf("o relatório não traz o motivo da indisponibilidade:\n%s", relatorio)
	}
}

// Um artefato ausente tem que aparecer como ausente.
func TestEmbedDoctorRelogaArtefatoAusente(t *testing.T) {
	manifest, _ := manifestDeTeste(t)

	if err := os.WriteFile(manifest.Model.Dest, []byte("conteudo"), 0o644); err != nil {
		t.Fatalf("preparar %s: %v", manifest.Model.Dest, err)
	}

	var out bytes.Buffer
	if err := runEmbedDoctor(&out, manifest, func() error { return nil }); err != nil {
		t.Fatalf("runEmbedDoctor: %v", err)
	}
	relatorio := out.String()

	if !strings.Contains(relatorio, manifest.Model.Dest+" — presente") {
		t.Errorf("o relatório não diz que o modelo está presente:\n%s", relatorio)
	}
	if !strings.Contains(relatorio, manifest.Tokenizer.Dest+" — ausente") {
		t.Errorf("o relatório não diz que o tokenizer está ausente:\n%s", relatorio)
	}
}

// Falha de download tem que chegar a quem chamou o comando.
func TestEmbedDownloadPropagaErroDoDownloader(t *testing.T) {
	manifest, _ := manifestDeTeste(t)
	manifest.Model.URL = "http://127.0.0.1:1/model_int8.onnx"

	var out bytes.Buffer
	err := runEmbedDownload(context.Background(), &out, manifest)
	if err == nil {
		t.Fatal("runEmbedDownload deveria falhar quando o download falha")
	}
	if !strings.Contains(err.Error(), manifest.Model.URL) {
		t.Errorf("o erro não nomeia a URL tentada: %v", err)
	}
	if !strings.Contains(err.Error(), manifest.Model.Dest) {
		t.Errorf("o erro não nomeia o destino: %v", err)
	}
}
