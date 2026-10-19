package mcp

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// newStaticFixture cria uma raiz servida com a viewer compilada, um asset do
// bundle e um artefato de grafo.
func newStaticFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "app", "assets"), 0o755); err != nil {
		t.Fatalf("falha ao criar app/assets: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "graph"), 0o755); err != nil {
		t.Fatalf("falha ao criar graph/: %v", err)
	}

	files := map[string]string{
		filepath.Join("app", "index.html"):         "<!doctype html><title>viewer</title>",
		filepath.Join("app", "assets", "index.js"): "console.log('bundle')",
		filepath.Join("graph", "graph.html"):       "<html><body>grafo</body></html>",
	}
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("falha ao escrever %s: %v", name, err)
		}
	}

	return root
}

// doRequest executa uma requisição no handler estático e devolve o recorder.
func doRequest(handler http.Handler, method, target string, accept string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestStaticHandler_ServesExistingFile(t *testing.T) {
	root := newStaticFixture(t)
	handler := NewStaticHandler(root, nil)

	rec := doRequest(handler, http.MethodGet, "/assets/index.js", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("esperado 200 para arquivo existente, obteve %d", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "console.log('bundle')") {
		t.Errorf("conteúdo inesperado: %q", body)
	}
}

func TestStaticHandler_ServesHTMLWithContentType(t *testing.T) {
	root := newStaticFixture(t)
	handler := NewStaticHandler(root, nil)

	rec := doRequest(handler, http.MethodGet, "/index.html", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("esperado 200, obteve %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("Content-Type inesperado: %s", ct)
	}
}

func TestStaticHandler_RejectsPathTraversal(t *testing.T) {
	root := newStaticFixture(t)

	outside := filepath.Join(filepath.Dir(root), "fora-da-raiz.txt")
	if err := os.WriteFile(outside, []byte("conteudo secreto"), 0o644); err != nil {
		t.Fatalf("falha ao criar arquivo fora da raiz: %v", err)
	}
	defer func() { _ = os.Remove(outside) }()

	handler := NewStaticHandler(root, nil)

	// O Accept de navegação é proposital: se o fallback fosse avaliado antes
	// da barreira, a resposta seria o entrypoint com 200 em vez de 404.
	targets := []string{
		"/../../fora-da-raiz.txt",
		"/assets/../../fora-da-raiz.txt",
		"/..%2f..%2ffora-da-raiz.txt",
		"/graph/../../fora-da-raiz.txt",
		"/%2e%2e/%2e%2e/fora-da-raiz.txt",
	}

	for _, target := range targets {
		rec := doRequest(handler, http.MethodGet, target, "text/html")

		if rec.Code != http.StatusNotFound {
			t.Errorf("%s: esperado 404, obteve %d", target, rec.Code)
		}
		body := rec.Body.String()
		if strings.Contains(body, "conteudo secreto") {
			t.Errorf("%s: a resposta devolveu o arquivo fora da raiz", target)
		}
		if strings.Contains(body, "<title>viewer</title>") {
			t.Errorf("%s: a resposta devolveu o entrypoint da SPA", target)
		}
	}
}

func TestStaticHandler_RootWithoutBuildReturns503(t *testing.T) {
	handler := NewStaticHandler(t.TempDir(), nil)

	rec := doRequest(handler, http.MethodGet, "/", "")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("esperado 503 sem build, obteve %d", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "npm run build") {
		t.Errorf("mensagem de degradação ausente: %q", body)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/plain") {
		t.Errorf("a degradação deveria responder em texto puro, obteve %s", ct)
	}
}

func TestStaticHandler_FallbackServesEntrypoint(t *testing.T) {
	root := newStaticFixture(t)
	handler := NewStaticHandler(root, nil)

	rec := doRequest(handler, http.MethodGet, "/clusters", "text/html,application/xhtml+xml")
	if rec.Code != http.StatusOK {
		t.Fatalf("esperado 200 no fallback SPA, obteve %d", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "<title>viewer</title>") {
		t.Errorf("o fallback não devolveu o entrypoint: %q", body)
	}
}

func TestStaticHandler_FallbackNotAppliedToMissingAsset(t *testing.T) {
	root := newStaticFixture(t)
	handler := NewStaticHandler(root, nil)

	rec := doRequest(handler, http.MethodGet, "/assets/nao-existe.js", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("esperado 404 para asset faltando, obteve %d", rec.Code)
	}
	if body := rec.Body.String(); strings.Contains(body, "<title>viewer</title>") {
		t.Errorf("o entrypoint não pode substituir um asset faltando: %q", body)
	}
}

func TestStaticHandler_GraphRouteServesArtifactsOnly(t *testing.T) {
	root := newStaticFixture(t)
	handler := NewStaticHandler(root, nil)

	rec := doRequest(handler, http.MethodGet, "/graph/graph.html", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("esperado 200 para artefato existente, obteve %d", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "<body>grafo</body>") {
		t.Errorf("conteúdo inesperado: %q", body)
	}

	// A rota /graph/ é reservada: o fallback da SPA nunca a alcança.
	missing := doRequest(handler, http.MethodGet, "/graph/nao-existe.html", "text/html")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("esperado 404 para artefato ausente, obteve %d", missing.Code)
	}
	if body := missing.Body.String(); strings.Contains(body, "<title>viewer</title>") {
		t.Errorf("a rota /graph/ não pode devolver o entrypoint: %q", body)
	}
}

// TestStaticHandler_GraphPathIsASpaRoute fixa a fronteira da área reservada de
// grafo: o **conteúdo** de `/graph/` é dos artefatos, mas o path exato `/graph`
// é uma rota da SPA.
//
// Sem essa distinção, a rota do grafo da viewer respondia 404 em texto puro
// quando o usuário abria ou recarregava a URL direto — a navegação por link
// funcionava, porque o roteador do cliente não passa pelo servidor.
func TestStaticHandler_GraphPathIsASpaRoute(t *testing.T) {
	root := newStaticFixture(t)
	handler := NewStaticHandler(root, nil)

	for _, target := range []string{"/graph", "/graph/"} {
		rec := doRequest(handler, http.MethodGet, target, "text/html")
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: esperado 200 (rota da SPA), obteve %d", target, rec.Code)
		}
		if body := rec.Body.String(); !strings.Contains(body, "<title>viewer</title>") {
			t.Errorf("%s: esperado o entrypoint da SPA, obteve %q", target, body)
		}
	}
}

func TestStaticHandler_AppliesSecurityHeaders(t *testing.T) {
	root := newStaticFixture(t)
	handler := NewStaticHandler(root, nil)

	for _, target := range []string{"/index.html", "/clusters"} {
		rec := doRequest(handler, http.MethodGet, target, "text/html")

		csp := rec.Header().Get("Content-Security-Policy")
		if !strings.Contains(csp, "default-src 'self'") || !strings.Contains(csp, "frame-ancestors 'none'") {
			t.Errorf("%s: CSP ausente ou incorreta: %q", target, csp)
		}
		if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Errorf("%s: X-Content-Type-Options ausente", target)
		}
		if rec.Header().Get("Referrer-Policy") != "no-referrer" {
			t.Errorf("%s: Referrer-Policy ausente", target)
		}
	}
}

func TestStaticHandler_RejectsNonGetMethods(t *testing.T) {
	root := newStaticFixture(t)
	handler := NewStaticHandler(root, nil)

	rec := doRequest(handler, http.MethodPost, "/index.html", "")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("esperado 405, obteve %d", rec.Code)
	}
}

func TestHTTPServer_ServesStaticRoutesAndKeepsProtocolRoutes(t *testing.T) {
	root := newStaticFixture(t)
	srv := newTestServer()
	httpSrv := NewHTTPServer(srv, HTTPServerOptions{
		CORSEnabled: true,
		ServedRoot:  root,
	})
	ts := httptest.NewServer(httpSrv.httpServer.Handler)
	defer ts.Close()

	// A rota do protocolo continua respondendo.
	health, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("falha ao consultar /health: %v", err)
	}
	defer health.Body.Close()
	if health.StatusCode != http.StatusOK {
		t.Errorf("esperado 200 em /health, obteve %d", health.StatusCode)
	}

	// A rota estática da SPA responde.
	spa, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("falha ao consultar /: %v", err)
	}
	defer spa.Body.Close()
	if spa.StatusCode != http.StatusOK {
		t.Errorf("esperado 200 em /, obteve %d", spa.StatusCode)
	}

	// A rota estática herda o CORS já habilitado.
	if spa.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("as rotas estáticas devem herdar o CORS do transporte")
	}
}

func TestHTTPServer_PublishesBoundAddress(t *testing.T) {
	srv := newTestServer()
	httpSrv := NewHTTPServer(srv, HTTPServerOptions{})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("falha ao abrir listener tcp: %v", err)
	}

	go func() {
		_ = httpSrv.Serve(ln)
	}()
	defer func() { _ = httpSrv.Shutdown(context.Background()) }()

	deadline := time.Now().Add(2 * time.Second)
	for srv.BaseURL() == "" && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	base := srv.BaseURL()
	if base == "" {
		t.Fatal("o endereço do transporte não foi publicado")
	}
	if !strings.HasPrefix(base, "http://127.0.0.1:") {
		t.Errorf("endereço inesperado: %s", base)
	}
	if httpSrv.BaseURL() != base {
		t.Errorf("o HTTPServer e o Server MCP publicaram endereços diferentes: %s vs %s", httpSrv.BaseURL(), base)
	}
}
