package embedder

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// assertNoLeftovers falha quando o diretório de destino ficou com algum
// arquivo. O contrato do AC6 é que uma falha de download não deixe parcial
// no disco -- um temporário esquecido conta como parcial.
func assertNoLeftovers(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("não foi possível ler o diretório de destino %q: %v", dir, err)
	}
	if len(entries) != 0 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("o diretório de destino deveria estar vazio, tem %v", names)
	}
}

func TestDownloadArtifactGravaNoDestino(t *testing.T) {
	content := []byte("conteudo do artefato")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer srv.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "model_int8.onnx")

	err := DownloadArtifact(context.Background(), Artifact{
		URL:  srv.URL + "/model_int8.onnx",
		Dest: dest,
		Size: int64(len(content)),
	})
	if err != nil {
		t.Fatalf("DownloadArtifact: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("o arquivo não foi criado no destino: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("conteúdo gravado = %q, esperado %q", got, content)
	}
}

func TestDownloadArtifactTamanhoDivergenteNaoDeixaParcial(t *testing.T) {
	content := []byte("conteudo truncado")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer srv.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "model_int8.onnx")
	url := srv.URL + "/model_int8.onnx"

	err := DownloadArtifact(context.Background(), Artifact{
		URL:  url,
		Dest: dest,
		Size: int64(len(content)) + 100, // o servidor entrega menos do que o esperado
	})
	if err == nil {
		t.Fatal("DownloadArtifact deveria falhar quando o tamanho diverge do esperado")
	}
	if !strings.Contains(err.Error(), url) {
		t.Errorf("o erro não nomeia a URL tentada: %v", err)
	}
	if !strings.Contains(err.Error(), dest) {
		t.Errorf("o erro não nomeia o destino: %v", err)
	}
	assertNoLeftovers(t, dir)
}

func TestDownloadArtifactSHA256DivergenteNaoDeixaParcial(t *testing.T) {
	content := []byte("conteudo adulterado")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer srv.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "onnxruntime.zip")
	url := srv.URL + "/onnxruntime.zip"
	wrong := strings.Repeat("0", 64)

	err := DownloadArtifact(context.Background(), Artifact{
		URL:    url,
		Dest:   dest,
		Size:   int64(len(content)),
		SHA256: wrong,
	})
	if err == nil {
		t.Fatal("DownloadArtifact deveria falhar quando o sha256 diverge do esperado")
	}
	if !strings.Contains(err.Error(), url) {
		t.Errorf("o erro não nomeia a URL tentada: %v", err)
	}
	if !strings.Contains(err.Error(), dest) {
		t.Errorf("o erro não nomeia o destino: %v", err)
	}
	assertNoLeftovers(t, dir)
}

func TestDownloadArtifactHTTPFalhaNaoDeixaParcial(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "model_int8.onnx")
	url := srv.URL + "/model_int8.onnx"

	err := DownloadArtifact(context.Background(), Artifact{
		URL:  url,
		Dest: dest,
		Size: 10,
	})
	if err == nil {
		t.Fatal("DownloadArtifact deveria falhar com HTTP 404")
	}
	if !strings.Contains(err.Error(), url) {
		t.Errorf("o erro não nomeia a URL tentada: %v", err)
	}
	if !strings.Contains(err.Error(), dest) {
		t.Errorf("o erro não nomeia o destino: %v", err)
	}
	assertNoLeftovers(t, dir)
}

// O destino já correto não é baixado de novo: o instalador chama
// `mem embed download` numa máquina que pode já ter os artefatos.
func TestDownloadArtifactPulaArquivoJaValido(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Write([]byte("conteudo"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "model_int8.onnx")
	content := []byte("conteudo")
	if err := os.WriteFile(dest, content, 0o644); err != nil {
		t.Fatalf("preparar o destino: %v", err)
	}

	err := DownloadArtifact(context.Background(), Artifact{
		URL:  srv.URL + "/model_int8.onnx",
		Dest: dest,
		Size: int64(len(content)),
	})
	if err != nil {
		t.Fatalf("DownloadArtifact: %v", err)
	}
	if called {
		t.Error("o arquivo já válido no destino não deveria ser baixado de novo")
	}
}

// zip com a biblioteca do ONNX Runtime dentro, como o asset oficial do Windows.
func TestDownloadRuntimeLibraryExtraiMembroDoZip(t *testing.T) {
	lib := bytes.Repeat([]byte("onnxruntime-dll"), 1024)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("onnxruntime/lib/onnxruntime.dll")
	if err != nil {
		t.Fatalf("criar o zip: %v", err)
	}
	if _, err := w.Write(lib); err != nil {
		t.Fatalf("escrever no zip: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("fechar o zip: %v", err)
	}
	archive := buf.Bytes()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(archive)
	}))
	defer srv.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "onnxruntime.dll")
	sum := sha256.Sum256(archive)

	err = DownloadRuntimeLibrary(context.Background(), RuntimeArchive{
		URL:      srv.URL + "/onnxruntime-win-x64.zip",
		Dest:     dest,
		Member:   "onnxruntime.dll",
		Format:   FormatZip,
		Size:     int64(len(archive)),
		SHA256:   hex.EncodeToString(sum[:]),
		MinBytes: 1024,
	})
	if err != nil {
		t.Fatalf("DownloadRuntimeLibrary: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("a biblioteca não foi extraída para o destino: %v", err)
	}
	if !bytes.Equal(got, lib) {
		t.Errorf("a biblioteca extraída tem %d bytes, esperado %d", len(got), len(lib))
	}
}

// tgz com a biblioteca do ONNX Runtime dentro, como o asset oficial do Linux.
func TestDownloadRuntimeLibraryExtraiMembroDoTgz(t *testing.T) {
	lib := bytes.Repeat([]byte("onnxruntime-so"), 1024)

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	if err := tw.WriteHeader(&tar.Header{
		Name: "onnxruntime/lib/libonnxruntime.so",
		Mode: 0o644,
		Size: int64(len(lib)),
	}); err != nil {
		t.Fatalf("escrever o header do tar: %v", err)
	}
	if _, err := tw.Write(lib); err != nil {
		t.Fatalf("escrever no tar: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("fechar o tar: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("fechar o gzip: %v", err)
	}
	archive := buf.Bytes()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(archive)
	}))
	defer srv.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "libonnxruntime.so")

	err := DownloadRuntimeLibrary(context.Background(), RuntimeArchive{
		URL:      srv.URL + "/onnxruntime-linux-x64.tgz",
		Dest:     dest,
		Member:   "libonnxruntime.so",
		Format:   FormatTgz,
		Size:     int64(len(archive)),
		MinBytes: 1024,
	})
	if err != nil {
		t.Fatalf("DownloadRuntimeLibrary: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("a biblioteca não foi extraída para o destino: %v", err)
	}
	if !bytes.Equal(got, lib) {
		t.Errorf("a biblioteca extraída tem %d bytes, esperado %d", len(got), len(lib))
	}
}

// Uma biblioteca extraída menor que o mínimo é recusada: o destino fica
// intacto e o erro diz que o tamanho não serve.
func TestDownloadRuntimeLibraryBibliotecaAbaixoDoMinimoNaoDeixaParcial(t *testing.T) {
	lib := bytes.Repeat([]byte("pequeno"), 16)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("onnxruntime/lib/onnxruntime.dll")
	if err != nil {
		t.Fatalf("criar o zip: %v", err)
	}
	if _, err := w.Write(lib); err != nil {
		t.Fatalf("escrever no zip: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("fechar o zip: %v", err)
	}
	archive := buf.Bytes()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(archive)
	}))
	defer srv.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "onnxruntime.dll")

	err = DownloadRuntimeLibrary(context.Background(), RuntimeArchive{
		URL:      srv.URL + "/onnxruntime-win-x64.zip",
		Dest:     dest,
		Member:   "onnxruntime.dll",
		Format:   FormatZip,
		Size:     int64(len(archive)),
		MinBytes: 8 << 20,
	})
	if err == nil {
		t.Fatal("DownloadRuntimeLibrary deveria recusar uma biblioteca abaixo do tamanho mínimo")
	}
	assertNoLeftovers(t, dir)
}

// Um pacote sem o membro esperado não pode deixar biblioteca no destino.
func TestDownloadRuntimeLibraryMembroAusenteNaoDeixaParcial(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("onnxruntime/lib/onnxruntime.dll")
	if err != nil {
		t.Fatalf("criar o zip: %v", err)
	}
	if _, err := w.Write([]byte("outra coisa")); err != nil {
		t.Fatalf("escrever no zip: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("fechar o zip: %v", err)
	}
	archive := buf.Bytes()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(archive)
	}))
	defer srv.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "libonnxruntime.so")
	url := srv.URL + "/onnxruntime-linux-x64.zip"

	err = DownloadRuntimeLibrary(context.Background(), RuntimeArchive{
		URL:      url,
		Dest:     dest,
		Member:   "libonnxruntime.so",
		Format:   FormatTgz,
		Size:     int64(len(archive)),
		MinBytes: 1024,
	})
	if err == nil {
		t.Fatal("DownloadRuntimeLibrary deveria falhar quando o membro não existe no pacote")
	}
	if !strings.Contains(err.Error(), url) {
		t.Errorf("o erro não nomeia a URL tentada: %v", err)
	}
	if !strings.Contains(err.Error(), dest) {
		t.Errorf("o erro não nomeia o destino: %v", err)
	}
	assertNoLeftovers(t, dir)
}
