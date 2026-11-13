package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/FelipeMiiller/my-memory/internal/embedder"
)

// embedProviderProbe diz se o provider está disponível.
//
// Em produção o probe é o construtor real do provider embutido: ele falha
// quando o binário foi compilado sem a tag `onnx` e quando algum artefato
// falta no disco. Os testes usam um dublê, porque o provider real exige o
// modelo de 118 MB.
type embedProviderProbe func() error

// runEmbedCLI é o entrypoint de `mem embed <download|doctor>`.
func runEmbedCLI(ctx context.Context, args []string) error {
	if len(args) == 0 {
		printEmbedHelp()
		return nil
	}

	switch args[0] {
	case "download":
		provider := "builtin"
		fs := flag.NewFlagSet("embed download", flag.ContinueOnError)
		fs.StringVar(&provider, "provider", provider, "Provider de embedding (`builtin`)")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if provider != "builtin" {
			return fmt.Errorf("provider %q desconhecido: o único provider com artefatos baixáveis é `builtin`", provider)
		}
		manifest, err := embedder.DefaultManifest("", "")
		if err != nil {
			return err
		}
		return runEmbedDownload(ctx, os.Stdout, manifest)

	case "doctor":
		manifest, err := embedder.DefaultManifest("", "")
		if err != nil {
			return err
		}
		return runEmbedDoctor(os.Stdout, manifest, realEmbedProviderProbe)

	default:
		printEmbedHelp()
		return fmt.Errorf("subcomando desconhecido: %q", args[0])
	}
}

// realEmbedProviderProbe tenta construir o provider embutido.
func realEmbedProviderProbe() error {
	client, err := newBuiltinEmbedder(nil)
	if err != nil {
		return err
	}
	if closer, ok := client.(interface{ Close() error }); ok {
		_ = closer.Close()
	}
	return nil
}

// runEmbedDownload baixa os artefatos do provider e diz onde eles ficaram.
func runEmbedDownload(ctx context.Context, out io.Writer, manifest embedder.Manifest) error {
	fmt.Fprintf(out, "📥 Baixando os artefatos do provider builtin...\n")

	for _, a := range []embedder.Artifact{manifest.Model, manifest.Tokenizer} {
		fmt.Fprintf(out, "   ⬇ %s\n", a.Dest)
		if err := embedder.DownloadArtifact(ctx, a); err != nil {
			return err
		}
	}

	fmt.Fprintf(out, "   ⬇ %s\n", manifest.Runtime.Dest)
	if err := embedder.DownloadRuntimeLibrary(ctx, manifest.Runtime); err != nil {
		return err
	}

	fmt.Fprintf(out, "✅ Provider pronto.\n")
	fmt.Fprintf(out, "   Modelo:    %s\n", manifest.Model.Dest)
	fmt.Fprintf(out, "   Tokenizer: %s\n", manifest.Tokenizer.Dest)
	fmt.Fprintf(out, "   Runtime:   %s\n", manifest.Runtime.Dest)
	return nil
}

// runEmbedDoctor diagnostica o provider no ambiente.
//
// O diagnóstico é do ambiente, não do conteúdo do vault: ele roda igual em um
// vault vazio. Quando o provider não está disponível, o comando RELATA e
// termina com sucesso -- quem chama (o instalador, um script) decide o que
// fazer com um provider ausente.
func runEmbedDoctor(out io.Writer, manifest embedder.Manifest, probe embedProviderProbe) error {
	fmt.Fprintf(out, "🔍 Provider de embedding\n\n")
	fmt.Fprintf(out, "   Provider:         builtin (ONNX MiniLM)\n")
	fmt.Fprintf(out, "   Dimensão:         %d\n", embedder.EmbeddingDim)
	fmt.Fprintf(out, "   Modelo:           %s\n", manifest.Model.Dest)
	fmt.Fprintf(out, "   Tokenizer:        %s\n", manifest.Tokenizer.Dest)
	fmt.Fprintf(out, "   Biblioteca ORT:   %s\n", manifest.Runtime.Dest)
	fmt.Fprintf(out, "\n   Artefatos:\n")

	for _, a := range []embedder.Artifact{manifest.Model, manifest.Tokenizer} {
		writeEmbedArtifact(out, a.Dest)
	}
	writeEmbedArtifact(out, manifest.Runtime.Dest)

	if err := probe(); err != nil {
		fmt.Fprintf(out, "\n   Status: INDISPONÍVEL — %v\n", err)
		fmt.Fprintf(out, "   (o diagnóstico é do ambiente: sem provider a indexação cai para FTS léxico)\n")
		return nil
	}

	fmt.Fprintf(out, "\n   Status: ATIVO\n")
	return nil
}

// writeEmbedArtifact reporta a presença de um artefato no disco.
func writeEmbedArtifact(out io.Writer, path string) {
	info, err := os.Stat(path)
	if err != nil {
		fmt.Fprintf(out, "     ✘ %s — ausente\n", path)
		return
	}
	fmt.Fprintf(out, "     ✔ %s — presente (%d bytes)\n", path, info.Size())
}

// printEmbedHelp explica o uso de `mem embed`.
func printEmbedHelp() {
	fmt.Println("  mem embed download [--provider builtin]")
	fmt.Println("      Baixa o modelo, o tokenizer e a biblioteca do ONNX Runtime para ~/.memory/models/")
	fmt.Println("  mem embed doctor")
	fmt.Println("      Diagnostica o provider: caminhos, dimensão e presença de cada artefato no disco")
}
