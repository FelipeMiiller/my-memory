package embedder

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ORTVersion é a versão do ONNX Runtime que o provider embutido exige.
//
// A versão não é livre: o binding (github.com/yalue/onnxruntime_go) foi
// escrito contra a API version 29, e é essa versão que a ADR-035 fixa.
const ORTVersion = "1.29.0"

// minRuntimeLibBytes é o tamanho mínimo aceito para a biblioteca extraída do
// pacote do ONNX Runtime. Não é o tamanho exato (que muda entre plataformas):
// o que importa é pegar o caso de membro errado ou extração truncada.
const minRuntimeLibBytes = 8 << 20 // 8 MiB

// Revisões fixadas por commit, não por "main".
//
// O vetor gravado no índice depende do arquivo exato do modelo: se o upstream
// publicasse outro modelo sob o mesmo nome, todo vetor do cofre passaria a
// significar outra coisa sem nenhum sinal. O Epoch (embedder.Epoch) cobre o
// nome e a dimensão, não o conteúdo do arquivo.
const (
	modelRepoRevision     = "2c4055b12046f11709e9df2c122e59ffbdc2f900"
	tokenizerRepoRevision = "e8f8c211226b894fcb81acc59f3b34ba3efd5f42"
)

// URLs do modelo e do tokenizer. O modelo vem do repositório ONNX da
// Xenova; o tokenizer SentencePiece vem do repositório original da
// sentence-transformers, que é o único que publica o sentencepiece.bpe.model.
var (
	modelBaseURL     = "https://huggingface.co/Xenova/" + ModelID + "/resolve/" + modelRepoRevision
	tokenizerBaseURL = "https://huggingface.co/sentence-transformers/" + ModelID + "/resolve/" + tokenizerRepoRevision
)

// Formatos de pacote do ONNX Runtime.
const (
	FormatZip = "zip"
	FormatTgz = "tgz"
)

// ortArchive descreve um asset oficial do ONNX Runtime.
type ortArchive struct {
	url    string
	size   int64
	sha256 string
	member string
	format string
}

// ortArchives mapeia GOOS/GOARCH para o asset oficial do ONNX Runtime 1.29.0.
//
// Os tamanhos e os digests vêm da API do GitHub (o campo `digest` do release),
// não de medições locais: são os valores que a Microsoft publica para o
// asset. A release 1.29.0 não tem build de macOS para x64 (só arm64).
var ortArchives = map[string]ortArchive{
	"windows/amd64": {
		url:    "https://github.com/microsoft/onnxruntime/releases/download/v1.29.0/onnxruntime-win-x64-1.29.0.zip",
		size:   79645520,
		sha256: "c9b4b7086b529ad814f428c1bad028e20a25d7dc0699836775faace4ab5b78b2",
		member: "onnxruntime.dll",
		format: FormatZip,
	},
	"linux/amd64": {
		url:    "https://github.com/microsoft/onnxruntime/releases/download/v1.29.0/onnxruntime-linux-x64-1.29.0.tgz",
		size:   11082880,
		sha256: "c3fddc4f139a045b0c4902c57410f0694f1c2fdf9b6939fbe38b1aeae7cd14ba",
		member: "libonnxruntime.so",
		format: FormatTgz,
	},
	"linux/arm64": {
		url:    "https://github.com/microsoft/onnxruntime/releases/download/v1.29.0/onnxruntime-linux-aarch64-1.29.0.tgz",
		size:   10027600,
		sha256: "e1799098ebc054b370f6176a450f158720f297818c613e5dc99b92e2ec82346f",
		member: "libonnxruntime.so",
		format: FormatTgz,
	},
	"darwin/arm64": {
		url:    "https://github.com/microsoft/onnxruntime/releases/download/v1.29.0/onnxruntime-osx-arm64-1.29.0.tgz",
		size:   41578864,
		sha256: "d0706fc34f315d8c88639d0a8c81f2e09e815f282cabed3493c06a054352cf92",
		member: "libonnxruntime.dylib",
		format: FormatTgz,
	},
}

// ORTLibName é o nome do arquivo da biblioteca do ONNX Runtime no sistema atual.
//
// O nome não é o mesmo em todo sistema: o dlopen do runtime procura a
// biblioteca pelo caminho exato, e no Linux/macOS o arquivo se chama
// libonnxruntime.so / libonnxruntime.dylib.
func ORTLibName() string {
	switch runtime.GOOS {
	case "windows":
		return "onnxruntime.dll"
	case "darwin":
		return "libonnxruntime.dylib"
	default:
		return "libonnxruntime.so"
	}
}

// DefaultModelsDir é o diretório onde o modelo e o tokenizer são instalados.
func DefaultModelsDir() string {
	if d := os.Getenv("MY_MEMORY_MINILM_MODEL_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".memory", "models", ModelID)
	}
	return filepath.Join(home, ".memory", "models", ModelID)
}

// DefaultORTDir é o diretório onde a biblioteca do ONNX Runtime é instalada.
func DefaultORTDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".memory", "models", "onnxruntime")
	}
	return filepath.Join(home, ".memory", "models", "onnxruntime")
}

// DefaultORTLibPath é o caminho da biblioteca do ONNX Runtime.
func DefaultORTLibPath() string {
	if p := os.Getenv("MY_MEMORY_ORT_LIB"); p != "" {
		return p
	}
	return filepath.Join(DefaultORTDir(), ORTLibName())
}

// Artifact é um arquivo que o provider precisa no disco.
type Artifact struct {
	// URL é a origem do download.
	URL string
	// Dest é o caminho final do arquivo.
	Dest string
	// Size é o tamanho exato esperado em bytes. Zero desliga a verificação.
	Size int64
	// SHA256 é o digest esperado. Vazio desliga a verificação.
	SHA256 string
}

// RuntimeArchive é o pacote oficial do ONNX Runtime.
//
// A biblioteca não é publicada como arquivo solto: vem dentro de um zip
// (Windows) ou de um tgz (Linux/macOS), e é extraída do pacote.
type RuntimeArchive struct {
	// URL é a origem do pacote.
	URL string
	// Dest é o caminho final da biblioteca extraída.
	Dest string
	// Member é o nome base do arquivo procurado dentro do pacote.
	Member string
	// Format é o formato do pacote: FormatZip ou FormatTgz.
	//
	// O formato é declarado, não deduzido do nome do arquivo: o pacote é
	// baixado para um temporário, e um temporário não tem extensão.
	Format string
	// Size é o tamanho exato esperado do pacote.
	Size int64
	// SHA256 é o digest esperado do pacote.
	SHA256 string
	// MinBytes é o tamanho mínimo aceito da biblioteca extraída.
	MinBytes int64
}

// Manifest é o conjunto de artefatos que o provider embutido precisa.
type Manifest struct {
	Model     Artifact
	Tokenizer Artifact
	Runtime   RuntimeArchive
}

// DefaultManifest devolve o manifesto do provider para o sistema atual.
//
// modelDir e ortLibPath podem ser vazios: nesse caso valem os caminhos padrão
// (DefaultModelsDir e DefaultORTLibPath).
func DefaultManifest(modelDir, ortLibPath string) (Manifest, error) {
	if modelDir == "" {
		modelDir = DefaultModelsDir()
	}
	if ortLibPath == "" {
		ortLibPath = DefaultORTLibPath()
	}

	key := runtime.GOOS + "/" + runtime.GOARCH
	archive, ok := ortArchives[key]
	if !ok {
		return Manifest{}, fmt.Errorf(
			"não há build oficial do ONNX Runtime %s para %s: o provider embutido não pode ser instalado nesta plataforma",
			ORTVersion, key)
	}

	return Manifest{
		Model: Artifact{
			URL:  modelBaseURL + "/onnx/model_int8.onnx",
			Dest: filepath.Join(modelDir, "model_int8.onnx"),
			// Tamanho publicado pelo repositório do modelo.
			Size: 118054609,
		},
		Tokenizer: Artifact{
			URL:  tokenizerBaseURL + "/sentencepiece.bpe.model",
			Dest: filepath.Join(modelDir, "sentencepiece.bpe.model"),
			// Tamanho publicado pelo repositório do modelo.
			Size: 5069051,
		},
		Runtime: RuntimeArchive{
			URL:      archive.url,
			Dest:     ortLibPath,
			Member:   archive.member,
			Format:   archive.format,
			Size:     archive.size,
			SHA256:   archive.sha256,
			MinBytes: minRuntimeLibBytes,
		},
	}, nil
}

// DownloadProviderArtifacts baixa os três artefatos do provider: modelo,
// tokenizer e a biblioteca do ONNX Runtime.
//
// Cada artefato é baixado para um arquivo temporário no diretório de destino,
// tem o tamanho (e o digest, quando conhecido) verificado, e só então é
// movido para o caminho final por rename. Uma falha não deixa arquivo
// parcial no disco.
func DownloadProviderArtifacts(ctx context.Context, m Manifest) error {
	for _, a := range []Artifact{m.Model, m.Tokenizer} {
		if err := DownloadArtifact(ctx, a); err != nil {
			return err
		}
	}
	return DownloadRuntimeLibrary(ctx, m.Runtime)
}

// DownloadArtifact baixa um arquivo e o instala no destino.
//
// O arquivo é escrito em um temporário no mesmo diretório do destino (para o
// rename final ser atômico), tem o tamanho e o digest verificados, e só
// depois entra no lugar. Se qualquer verificação falhar, o temporário é
// removido e o diretório de destino fica como estava.
func DownloadArtifact(ctx context.Context, a Artifact) error {
	if strings.TrimSpace(a.URL) == "" {
		return fmt.Errorf("download sem URL para o destino %s", a.Dest)
	}
	if strings.TrimSpace(a.Dest) == "" {
		return fmt.Errorf("download sem destino para a URL %s", a.URL)
	}
	if ok, err := artifactAlreadyValid(a); err != nil {
		return err
	} else if ok {
		return nil
	}

	dir := filepath.Dir(a.Dest)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("download falhou para %s (destino %s): não foi possível criar o diretório: %w", a.URL, a.Dest, err)
	}

	tmp, err := os.CreateTemp(dir, "."+filepath.Base(a.Dest)+".partial")
	if err != nil {
		return fmt.Errorf("download falhou para %s (destino %s): %w", a.URL, a.Dest, err)
	}
	tmpName := tmp.Name()

	fail := func(format string, args ...any) error {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("download falhou para %s (destino %s): %s", a.URL, a.Dest, fmt.Sprintf(format, args...))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.URL, nil)
	if err != nil {
		return fail("%v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fail("%v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fail("HTTP %d", resp.StatusCode)
	}

	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(tmp, hasher), resp.Body)
	if err != nil {
		return fail("%v", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("download falhou para %s (destino %s): %v", a.URL, a.Dest, err)
	}

	if a.Size > 0 && written != a.Size {
		os.Remove(tmpName)
		return fmt.Errorf("download falhou para %s (destino %s): tamanho %d bytes, esperado %d", a.URL, a.Dest, written, a.Size)
	}
	if a.SHA256 != "" {
		got := hex.EncodeToString(hasher.Sum(nil))
		if !strings.EqualFold(got, a.SHA256) {
			os.Remove(tmpName)
			return fmt.Errorf("download falhou para %s (destino %s): sha256 %s, esperado %s", a.URL, a.Dest, got, a.SHA256)
		}
	}

	if err := os.Rename(tmpName, a.Dest); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("download falhou para %s (destino %s): %v", a.URL, a.Dest, err)
	}
	return nil
}

// artifactAlreadyValid diz se o destino já tem o artefato correto.
//
// Sem isso, cada `mem embed download` baixaria de novo 197 MB mesmo com tudo
// no lugar. A checagem é por tamanho (e digest, quando conhecido): um arquivo
// do tamanho errado é exatamente o caso do download truncado.
func artifactAlreadyValid(a Artifact) (bool, error) {
	info, err := os.Stat(a.Dest)
	if err != nil {
		return false, nil
	}
	if a.Size > 0 && info.Size() != a.Size {
		return false, nil
	}
	if a.SHA256 == "" {
		return true, nil
	}
	data, err := os.ReadFile(a.Dest)
	if err != nil {
		return false, nil
	}
	sum := sha256.Sum256(data)
	return strings.EqualFold(hex.EncodeToString(sum[:]), a.SHA256), nil
}

// DownloadRuntimeLibrary baixa o pacote do ONNX Runtime e extrai a biblioteca.
func DownloadRuntimeLibrary(ctx context.Context, a RuntimeArchive) error {
	if strings.TrimSpace(a.URL) == "" {
		return fmt.Errorf("download sem URL para o destino %s", a.Dest)
	}
	if strings.TrimSpace(a.Dest) == "" {
		return fmt.Errorf("download sem destino para a URL %s", a.URL)
	}
	if ok, err := runtimeAlreadyValid(a); err != nil {
		return err
	} else if ok {
		return nil
	}

	dir := filepath.Dir(a.Dest)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("download falhou para %s (destino %s): não foi possível criar o diretório: %w", a.URL, a.Dest, err)
	}

	tmp, err := os.CreateTemp(dir, ".onnxruntime-archive-*.partial")
	if err != nil {
		return fmt.Errorf("download falhou para %s (destino %s): %w", a.URL, a.Dest, err)
	}
	archivePath := tmp.Name()
	tmp.Close()

	// O pacote sai do lugar depois da extração, inclusive quando a extração
	// falha: nenhum arquivo parcial fica no diretório de destino.
	defer os.Remove(archivePath)

	if err := downloadToFile(ctx, a.URL, archivePath, a.Size, a.SHA256); err != nil {
		return err
	}

	if err := extractArchiveMember(archivePath, a.Member, a.Format, a.Dest, a.MinBytes); err != nil {
		return fmt.Errorf("download falhou para %s (destino %s): %v", a.URL, a.Dest, err)
	}
	return nil
}

// runtimeAlreadyValid diz se a biblioteca já está no destino com o tamanho
// mínimo esperado.
func runtimeAlreadyValid(a RuntimeArchive) (bool, error) {
	info, err := os.Stat(a.Dest)
	if err != nil {
		return false, nil
	}
	if a.MinBytes > 0 && info.Size() < a.MinBytes {
		return false, nil
	}
	return true, nil
}

// downloadToFile baixa uma URL para um caminho, verificando tamanho e digest.
//
// O destino já é um temporário criado pelo chamador: aqui o arquivo é
// escrito no lugar e apagado se a verificação falhar.
func downloadToFile(ctx context.Context, url, dest string, size int64, sha256hex string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("download falhou para %s (destino %s): %v", url, dest, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download falhou para %s (destino %s): %v", url, dest, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download falhou para %s (destino %s): HTTP %d", url, dest, resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("download falhou para %s (destino %s): %v", url, dest, err)
	}
	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(f, hasher), resp.Body)
	if err != nil {
		f.Close()
		os.Remove(dest)
		return fmt.Errorf("download falhou para %s (destino %s): %v", url, dest, err)
	}
	if err := f.Close(); err != nil {
		os.Remove(dest)
		return fmt.Errorf("download falhou para %s (destino %s): %v", url, dest, err)
	}

	if size > 0 && written != size {
		os.Remove(dest)
		return fmt.Errorf("download falhou para %s (destino %s): tamanho %d bytes, esperado %d", url, dest, written, size)
	}
	if sha256hex != "" {
		got := hex.EncodeToString(hasher.Sum(nil))
		if !strings.EqualFold(got, sha256hex) {
			os.Remove(dest)
			return fmt.Errorf("download falhou para %s (destino %s): sha256 %s, esperado %s", url, dest, got, sha256hex)
		}
	}
	return nil
}

// extractArchiveMember copia o membro pedido do pacote para destPath.
//
// O membro é procurado pelo nome base, porque o caminho interno do pacote
// (onnxruntime/lib/...) não é contrato da Microsoft.
func extractArchiveMember(archivePath, member, format, destPath string, minBytes int64) error {
	reader, cleanup, err := openArchiveMember(archivePath, member, format)
	if err != nil {
		return err
	}
	defer cleanup()

	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("não foi possível criar o diretório %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, "."+filepath.Base(destPath)+".partial")
	if err != nil {
		return fmt.Errorf("não foi possível criar o arquivo temporário em %s: %w", dir, err)
	}
	tmpName := tmp.Name()

	written, err := io.Copy(tmp, reader)
	if err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("falha ao extrair %s: %w", member, err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("falha ao extrair %s: %w", member, err)
	}
	if minBytes > 0 && written < minBytes {
		os.Remove(tmpName)
		return fmt.Errorf("a biblioteca extraída tem %d bytes, abaixo do mínimo %d", written, minBytes)
	}
	if err := os.Rename(tmpName, destPath); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("não foi possível mover a biblioteca extraída para %s: %w", destPath, err)
	}
	return nil
}

// openArchiveMember abre o membro pedido dentro de um zip ou de um tgz.
func openArchiveMember(archivePath, member, format string) (io.Reader, func(), error) {
	if format == FormatZip {
		zr, err := zip.OpenReader(archivePath)
		if err != nil {
			return nil, nil, fmt.Errorf("não foi possível abrir o pacote %s: %w", archivePath, err)
		}
		for _, f := range zr.File {
			if filepath.Base(f.Name) != member {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				zr.Close()
				return nil, nil, fmt.Errorf("não foi possível ler %s do pacote %s: %w", f.Name, archivePath, err)
			}
			return rc, func() { rc.Close(); zr.Close() }, nil
		}
		zr.Close()
		return nil, nil, fmt.Errorf("o membro %s não existe no pacote %s", member, archivePath)
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return nil, nil, fmt.Errorf("não foi possível abrir o pacote %s: %w", archivePath, err)
	}
	gz, err := gzip.NewReader(f)
	if err != nil {
		f.Close()
		return nil, nil, fmt.Errorf("não foi possível abrir o pacote %s: %w", archivePath, err)
	}
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			f.Close()
			gz.Close()
			return nil, nil, fmt.Errorf("o membro %s não existe no pacote %s", member, archivePath)
		}
		if err != nil {
			f.Close()
			gz.Close()
			return nil, nil, fmt.Errorf("não foi possível ler o pacote %s: %w", archivePath, err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if filepath.Base(hdr.Name) != member {
			continue
		}
		return tr, func() { gz.Close(); f.Close() }, nil
	}
}
