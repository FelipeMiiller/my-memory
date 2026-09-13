package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestServer_InitializeHandshake(t *testing.T) {
	in := `{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"protocolVersion": "2024-11-05"}}` + "\n"
	var out bytes.Buffer
	var errLog bytes.Buffer

	srv := NewServer("test-server", "1.0.0", strings.NewReader(in), &out, &errLog)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = srv.Run(ctx)

	var resp Response
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("erro ao decodificar resposta do initialize: %v\nSaída: %s", err, out.String())
	}

	if resp.Error != nil {
		t.Fatalf("esperava sucesso, obteve erro: %+v", resp.Error)
	}

	resultBytes, _ := json.Marshal(resp.Result)
	var initResult InitializeResult
	_ = json.Unmarshal(resultBytes, &initResult)

	if initResult.ProtocolVersion != ProtocolVersion {
		t.Errorf("esperava versão %s, obteve %s", ProtocolVersion, initResult.ProtocolVersion)
	}

	if initResult.ServerInfo.Name != "test-server" {
		t.Errorf("esperava nome 'test-server', obteve %s", initResult.ServerInfo.Name)
	}

	if initResult.Capabilities.Tools == nil {
		t.Errorf("esperava capacidade Tools declarada")
	}
}

func TestServer_Ping(t *testing.T) {
	in := `{"jsonrpc": "2.0", "id": 2, "method": "ping"}` + "\n"
	var out bytes.Buffer

	srv := NewServer("test-server", "1.0.0", strings.NewReader(in), &out, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = srv.Run(ctx)

	var resp Response
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("erro ao decodificar ping: %v", err)
	}

	if string(resp.ID) != "2" || resp.Error != nil {
		t.Errorf("resposta inesperada para ping: %+v", resp)
	}
}

func TestServer_UnknownMethod(t *testing.T) {
	in := `{"jsonrpc": "2.0", "id": 3, "method": "non_existent_method"}` + "\n"
	var out bytes.Buffer

	srv := NewServer("test-server", "1.0.0", strings.NewReader(in), &out, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = srv.Run(ctx)

	var resp Response
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("erro ao decodificar resposta: %v", err)
	}

	if resp.Error == nil || resp.Error.Code != CodeMethodNotFound {
		t.Errorf("esperava erro CodeMethodNotFound (-32601), obteve: %+v", resp.Error)
	}
}
