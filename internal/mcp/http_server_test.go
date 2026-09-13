package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func newTestServer() *Server {
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}
	srv := NewServer("test-mcp-server", "1.0.0", in, out, io.Discard)
	return srv
}

func TestHTTPServer_SSE_HandshakeAndMessage(t *testing.T) {
	srv := newTestServer()
	httpSrv := NewHTTPServer(srv, HTTPServerOptions{
		Addr:        ":0",
		CORSEnabled: true,
		DefaultRepo: "test/repo",
	})

	ts := httptest.NewServer(httpSrv.httpServer.Handler)
	defer ts.Close()

	// 1. Abre conexão SSE
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/sse", nil)
	if err != nil {
		t.Fatalf("falha ao criar requisição SSE: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req = req.WithContext(ctx)

	client := &http.Client{Timeout: 0}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("falha ao conectar no endpoint SSE: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("esperado status 200 para SSE, obteve %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Errorf("esperado Content-Type text/event-stream, obteve %s", ct)
	}

	reader := bufio.NewReader(resp.Body)

	// 2. Lê evento inicial "endpoint"
	line1, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("erro lendo linha 1 do SSE: %v", err)
	}
	line2, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("erro lendo linha 2 do SSE: %v", err)
	}
	_, _ = reader.ReadString('\n') // linha vazia delimitadora

	if !strings.Contains(line1, "event: endpoint") {
		t.Errorf("esperado 'event: endpoint', obteve %q", line1)
	}
	if !strings.Contains(line2, "data: /message?sessionId=") {
		t.Errorf("esperado 'data: /message?sessionId=...', obteve %q", line2)
	}

	// Extrai sessionId
	dataPart := strings.TrimSpace(strings.TrimPrefix(line2, "data:"))
	parsedURL, err := url.Parse(dataPart)
	if err != nil {
		t.Fatalf("falha ao parsear endpoint retornado: %v", err)
	}
	sessionID := parsedURL.Query().Get("sessionId")
	if sessionID == "" {
		t.Fatalf("sessionId extraído é vazio: %s", dataPart)
	}

	// 3. Envia requisição POST /message usando o sessionId obtido
	msgBody := `{"jsonrpc":"2.0","id":1,"method":"ping","params":{}}`
	postResp, err := http.Post(ts.URL+dataPart, "application/json", strings.NewReader(msgBody))
	if err != nil {
		t.Fatalf("falha ao enviar POST para /message: %v", err)
	}
	defer postResp.Body.Close()

	if postResp.StatusCode != http.StatusAccepted {
		t.Errorf("esperado status 202 Accepted no POST /message, obteve %d", postResp.StatusCode)
	}

	// 4. Lê o evento SSE "message" transmitido pelo servidor
	mLine1, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("erro lendo linha de evento message: %v", err)
	}
	mLine2, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("erro lendo data de mensagem SSE: %v", err)
	}

	if !strings.Contains(mLine1, "event: message") {
		t.Errorf("esperado 'event: message', obteve %q", mLine1)
	}
	if !strings.Contains(mLine2, `"id":1`) {
		t.Errorf("resposta SSE não contém id correspondente: %q", mLine2)
	}
}

func TestHTTPServer_SSE_Errors(t *testing.T) {
	srv := newTestServer()
	httpSrv := NewHTTPServer(srv, HTTPServerOptions{})
	ts := httptest.NewServer(httpSrv.httpServer.Handler)
	defer ts.Close()

	// POST no /sse (inválido)
	resp, _ := http.Post(ts.URL+"/sse", "application/json", nil)
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("POST /sse deve retornar 405, obteve %d", resp.StatusCode)
	}

	// GET no /message (inválido)
	resp, _ = http.Get(ts.URL + "/message")
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("GET /message deve retornar 405, obteve %d", resp.StatusCode)
	}

	// POST /message sem sessionId
	resp, _ = http.Post(ts.URL+"/message", "application/json", strings.NewReader("{}"))
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("POST /message sem sessionId deve retornar 400, obteve %d", resp.StatusCode)
	}

	// POST /message com sessionId inexistente
	resp, _ = http.Post(ts.URL+"/message?sessionId=inexistente123", "application/json", strings.NewReader("{}"))
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("POST /message com sessionId inválido deve retornar 404, obteve %d", resp.StatusCode)
	}
}

func TestHTTPServer_Health(t *testing.T) {
	srv := newTestServer()
	httpSrv := NewHTTPServer(srv, HTTPServerOptions{
		DefaultRepo: "org/repo-alfa",
	})
	ts := httptest.NewServer(httpSrv.httpServer.Handler)
	defer ts.Close()

	// GET /health
	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("falha ao consultar /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("esperado status 200 em /health, obteve %d", resp.StatusCode)
	}

	var hr HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&hr); err != nil {
		t.Fatalf("erro decodificando json de /health: %v", err)
	}

	if hr.Status != "healthy" {
		t.Errorf("status inesperado: %s", hr.Status)
	}
	if hr.Name != "test-mcp-server" {
		t.Errorf("nome inesperado: %s", hr.Name)
	}
	if hr.DefaultRepo != "org/repo-alfa" {
		t.Errorf("repo default inesperado: %s", hr.DefaultRepo)
	}
	if hr.ToolsCount == 0 {
		t.Errorf("tools_count deve ser maior que zero")
	}

	// POST /health (não permitido)
	respPost, _ := http.Post(ts.URL+"/health", "application/json", nil)
	if respPost.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("POST /health deve retornar 405, obteve %d", respPost.StatusCode)
	}
}

func TestHTTPServer_DirectRPC(t *testing.T) {
	srv := newTestServer()
	httpSrv := NewHTTPServer(srv, HTTPServerOptions{})
	ts := httptest.NewServer(httpSrv.httpServer.Handler)
	defer ts.Close()

	// 1. Sucesso chamando /mcp com ping
	payload := `{"jsonrpc":"2.0","id":42,"method":"ping","params":{}}`
	resp, err := http.Post(ts.URL+"/mcp", "application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatalf("falha em POST /mcp: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("esperado status 200 em POST /mcp, obteve %d", resp.StatusCode)
	}

	var res Response
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("erro ao decodificar resposta JSON-RPC: %v", err)
	}
	if string(res.ID) != "42" {
		t.Errorf("esperado ID '42' na resposta, obteve %s", string(res.ID))
	}

	// 2. JSON malformado em /mcp
	badResp, err := http.Post(ts.URL+"/mcp", "application/json", strings.NewReader("{invalid-json"))
	if err != nil {
		t.Fatalf("falha no POST com json inválido: %v", err)
	}
	defer badResp.Body.Close()

	if badResp.StatusCode != http.StatusBadRequest {
		t.Errorf("esperado status 400 para JSON inválido, obteve %d", badResp.StatusCode)
	}

	// 3. GET em /mcp (não permitido)
	getResp, _ := http.Get(ts.URL + "/mcp")
	if getResp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("GET /mcp deve retornar 405, obteve %d", getResp.StatusCode)
	}
}

func TestHTTPServer_CORS(t *testing.T) {
	srv := newTestServer()
	httpSrv := NewHTTPServer(srv, HTTPServerOptions{
		CORSEnabled: true,
	})
	ts := httptest.NewServer(httpSrv.httpServer.Handler)
	defer ts.Close()

	// Preflight OPTIONS
	req, _ := http.NewRequest(http.MethodOptions, ts.URL+"/mcp", nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("falha no preflight OPTIONS: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("esperado status 204 No Content para preflight, obteve %d", resp.StatusCode)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("cabeçalho Access-Control-Allow-Origin ausente ou incorreto")
	}
}

func TestHTTPServer_ServeAndShutdown(t *testing.T) {
	srv := newTestServer()
	httpSrv := NewHTTPServer(srv, HTTPServerOptions{})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("falha ao abrir listener tcp: %v", err)
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- httpSrv.Serve(ln)
	}()

	// Garante que o servidor subiu
	time.Sleep(50 * time.Millisecond)

	// Testa requisição rápida de health
	resp, err := http.Get("http://" + ln.Addr().String() + "/health")
	if err != nil {
		t.Fatalf("falha chamando health no listener real: %v", err)
	}
	resp.Body.Close()

	// Executa shutdown gracioso
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := httpSrv.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown retornou erro: %v", err)
	}

	err = <-serverErr
	if err != nil && err != http.ErrServerClosed {
		t.Errorf("erro inesperado no encerramento: %v", err)
	}
}
