package mcp

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ServedRootDir é o caminho, relativo à base do vault, da raiz publicada pelo
// transporte HTTP (ADR-059 D-2). Fica fora do escopo de indexação do vault, de
// modo que os artefatos gerados não entram no grafo nem no índice.
//
// A raiz tem duas subáreas:
//
//	app/   — o build estático da viewer (viewer/dist copiado para cá)
//	graph/ — os artefatos de grafo gerados por memory_visualize_graph
const ServedRootDir = ".mymemory/www"

// graphRoutePrefix é o prefixo reservado da rota de artefatos de grafo. Um path
// sob esse prefixo que não corresponde a um arquivo existente devolve 404, e
// nunca o fallback da SPA (ADR-060).
const graphRoutePrefix = "graph"

// viewerNotBuiltMessage é a degradação da ADR-059 D-6: sem build da viewer, a
// resposta diz isso em texto puro, em vez de 404 para cada arquivo.
const viewerNotBuiltMessage = "viewer nao compilado: rode `npm run build` em `viewer/`"

// graphArtifactNotFoundMessage é a resposta da rota reservada de artefatos.
const graphArtifactNotFoundMessage = "artefato de grafo nao encontrado"

// staticSecurityHeaders é a baseline de segurança registrada na ADR-059 (D-10).
// `script-src 'self'` sem `'unsafe-inline'` é viável porque o bundle gerado
// pelo Vite não tem script inline, e `connect-src 'self'` cobre a chamada da
// SPA a `/mcp`, na mesma origem.
const staticSecurityHeaders = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data:; font-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'"

// ServedRoot resolve a raiz servida a partir da base do vault. A base é a
// mesma usada para os artefatos de grafo, de modo que o que a tool escreve é
// exatamente o que o transporte publica.
func ServedRoot(base string) string {
	if base == "" {
		base = "."
	}
	return filepath.Join(base, filepath.FromSlash(ServedRootDir))
}

// GraphArtifactsDir resolve o diretório dos artefatos de grafo dentro da raiz
// servida.
func GraphArtifactsDir(base string) string {
	return filepath.Join(ServedRoot(base), graphRoutePrefix)
}

// StaticHandler publica a viewer compilada e os artefatos de grafo sob a raiz
// dedicada <base>/.mymemory/www/.
//
// O contrato de rotas é o da ADR-059 (D-3) emendado pela ADR-060:
//
//	GET /                       → app/index.html
//	GET /<qualquer>             → app/<qualquer>   (arquivo existente)
//	GET /graph/<arquivo>.html   → graph/<arquivo>  (artefato existente)
//	GET /<qualquer>             → app/index.html   (fallback SPA, só navegação)
//
// A área reservada de grafo é o **conteúdo** de `/graph/`. O path exato
// `/graph` é uma rota da SPA e cai no fallback, como qualquer path de
// navegação.
//
// A ordem de avaliação é não-negociável e é implementada em ServeHTTP:
// (1) resolver e normalizar o path, (2) se escapa da raiz servida → 404,
// (3) se é arquivo existente → servir, (4) se aceita "text/html" e não é
// /graph/ → o entrypoint com 200, (5) senão → 404.
type StaticHandler struct {
	root     string
	appDir   string
	graphDir string
	logger   *log.Logger
}

// NewStaticHandler cria o handler estático para a raiz servida informada. A
// raiz é resolvida para caminho absoluto, para que a comparação de fronteira
// não dependa do diretório de trabalho do processo.
func NewStaticHandler(root string, logger *log.Logger) *StaticHandler {
	if root == "" {
		root = ServedRoot(".")
	}
	if logger == nil {
		logger = log.New(io.Discard, "[mcp-static] ", log.LstdFlags)
	}

	abs, err := filepath.Abs(root)
	if err != nil {
		abs = filepath.Clean(root)
	}

	return &StaticHandler{
		root:     abs,
		appDir:   filepath.Join(abs, "app"),
		graphDir: filepath.Join(abs, graphRoutePrefix),
		logger:   logger,
	}
}

// ServeHTTP atende as rotas estáticas do transporte MCP.
func (s *StaticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.setSecurityHeaders(w)

	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Método não permitido (esperado GET)", http.StatusMethodNotAllowed)
		return
	}

	// (1) Resolver e normalizar o path. Um path que escape da raiz é recusado
	// aqui, antes de qualquer outra decisão.
	segments, ok := splitURLPath(r.URL.Path)
	if !ok {
		s.logger.Printf("path traversal bloqueado: %q", r.URL.Path)
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// A rota /graph/ é reservada para os artefatos de grafo. A área reservada é
	// o **conteúdo** do diretório: o path exato `/graph` é uma rota da SPA, e
	// a ADR-060 exige que qualquer path de navegação devolva o entrypoint — sem
	// isso, recarregar a rota do grafo na viewer devolveria 404 em vez do
	// painel.
	isGraph := len(segments) > 1 && segments[0] == graphRoutePrefix
	if isGraph {
		segments = segments[1:]
	}

	base := s.appDir
	if isGraph {
		base = s.graphDir
	}
	target := filepath.Join(append([]string{base}, segments...)...)

	// Fronteira da raiz servida: o caminho resolvido tem de continuar dentro
	// dela. O padrão segue internal/compiler/note.go:181-184.
	if target != s.root && !strings.HasPrefix(target, s.root+string(filepath.Separator)) {
		s.logger.Printf("caminho fora da raiz servida recusado: %q", r.URL.Path)
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// (3) Arquivo existente: servir o arquivo.
	if info, err := os.Stat(target); err == nil && !info.IsDir() {
		s.serveFile(w, r, target)
		return
	}

	// (4) /graph/ é reservado: nunca cai no fallback da SPA.
	if isGraph {
		http.Error(w, graphArtifactNotFoundMessage, http.StatusNotFound)
		return
	}

	entry := filepath.Join(s.appDir, "index.html")
	entryExists := fileExists(entry)

	// A raiz da URL é o entrypoint da SPA. Sem build, a degridação é 503 com
	// mensagem simples, e não 404 para cada arquivo.
	if len(segments) == 0 || (len(segments) == 1 && segments[0] == "index.html") {
		if !entryExists {
			http.Error(w, viewerNotBuiltMessage, http.StatusServiceUnavailable)
			return
		}
		s.serveFile(w, r, entry)
		return
	}

	// (5) O fallback SPA vale só para requisições de navegação. Um asset
	// faltando continua sendo 404, para que o browser não receba HTML onde
	// esperava JavaScript.
	if !acceptsHTML(r) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if !entryExists {
		http.Error(w, viewerNotBuiltMessage, http.StatusServiceUnavailable)
		return
	}

	s.serveFile(w, r, entry)
}

// splitURLPath decompõe o path pedido em segmentos e recusa qualquer path que
// escape da raiz servida. O path chega decodificado do net/http, então "%2e%2e"
// já chega como "..".
func splitURLPath(urlPath string) ([]string, bool) {
	normalized := strings.ReplaceAll(urlPath, "\\", "/")
	parts := strings.Split(normalized, "/")

	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		switch part {
		case "", ".":
			continue
		case "..":
			if len(segments) == 0 {
				return nil, false
			}
			segments = segments[:len(segments)-1]
		default:
			// Segmento com nome de volume ("C:", "C:foo") é recusado: em
			// Windows ele escapa da raiz, e em nenhum dos casos é um path
			// servível dentro da raiz.
			if filepath.VolumeName(part) != "" {
				return nil, false
			}
			segments = append(segments, part)
		}
	}

	return segments, true
}

// serveFile escreve o arquivo pedido com o Content-Type derivado da
// extensão. http.ServeContent cuida de Range, Last-Modified e do tamanho,
// sem o redirect de "index.html" que http.ServeFile faz.
func (s *StaticHandler) serveFile(w http.ResponseWriter, r *http.Request, path string) {
	f, err := os.Open(path)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil || info.IsDir() {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	http.ServeContent(w, r, info.Name(), info.ModTime(), f)
}

// setSecurityHeaders aplica a baseline da ADR-059 (D-10) em todas as respostas
// do handler.
func (s *StaticHandler) setSecurityHeaders(w http.ResponseWriter) {
	applySecurityHeaders(w)
}

// applySecurityHeaders aplica a baseline da ADR-059 (D-10). É uma função de
// pacote, e não um método, porque o preflight CORS responde antes do handler
// estático — e a baseline vale para ele também.
func applySecurityHeaders(w http.ResponseWriter) {
	header := w.Header()
	header.Set("Content-Security-Policy", staticSecurityHeaders)
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Referrer-Policy", "no-referrer")
}

// acceptsHTML informa se a requisição é de navegação. É o que separa o
// fallback SPA de um asset faltante (ADR-060).
func acceptsHTML(r *http.Request) bool {
	return strings.Contains(strings.ToLower(r.Header.Get("Accept")), "text/html")
}

// fileExists informa se o caminho é um arquivo existente.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
