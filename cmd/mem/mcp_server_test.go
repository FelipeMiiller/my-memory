package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/mcp"
)

func getFreePort(t *testing.T) int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("falha ao obter porta livre: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func TestMCPServer_CLI_Network(t *testing.T) {
	port := getFreePort(t)
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		runMCPServer(ctx, nil, nil, nil, "test-repo", addr, true)
	}()

	// Aguarda o servidor estar pronto (até 2 segundos)
	ready := false
	for i := 0; i < 20; i++ {
		resp, err := http.Get(fmt.Sprintf("http://%s/health", addr))
		if err == nil {
			resp.Body.Close()
			ready = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if !ready {
		t.Fatalf("servidor MCP HTTP não iniciou em %s dentro do tempo esperado", addr)
	}

	// Valida /health
	resp, err := http.Get(fmt.Sprintf("http://%s/health", addr))
	if err != nil {
		t.Fatalf("falha ao consultar /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("esperado status 200 em /health, obteve %d", resp.StatusCode)
	}

	var health mcp.HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("falha decodificando json de /health: %v", err)
	}
	if health.Status != "healthy" {
		t.Errorf("status inesperado: %s", health.Status)
	}
	if health.DefaultRepo != "test-repo" {
		t.Errorf("repo inesperado: %s", health.DefaultRepo)
	}

	// Cancela contexto para testar encerramento gracioso
	cancel()

	select {
	case <-serverDone:
		// Sucesso no encerramento gracioso
	case <-time.After(3 * time.Second):
		t.Errorf("servidor MCP não encerrou graciosamente dentro do tempo limite")
	}
}
