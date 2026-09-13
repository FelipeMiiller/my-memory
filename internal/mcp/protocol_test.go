package mcp

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func TestMessageReader_ValidRequest(t *testing.T) {
	input := `{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"protocolVersion": "2024-11-05"}}` + "\n"
	reader := NewMessageReader(strings.NewReader(input))

	req, err := reader.ReadRequest()
	if err != nil {
		t.Fatalf("esperava sucesso, obteve erro: %v", err)
	}

	if req.Method != "initialize" {
		t.Errorf("esperava method 'initialize', obteve '%s'", req.Method)
	}

	if string(req.ID) != "1" {
		t.Errorf("esperava ID '1', obteve '%s'", string(req.ID))
	}

	if req.IsNotification() {
		t.Errorf("esperava que não fosse notificação")
	}
}

func TestMessageReader_Notification(t *testing.T) {
	input := `{"jsonrpc": "2.0", "method": "notifications/initialized"}` + "\n"
	reader := NewMessageReader(strings.NewReader(input))

	req, err := reader.ReadRequest()
	if err != nil {
		t.Fatalf("esperava sucesso, obteve erro: %v", err)
	}

	if !req.IsNotification() {
		t.Errorf("esperava notificação")
	}
}

func TestMessageReader_ParseError(t *testing.T) {
	input := `{invalid-json` + "\n"
	reader := NewMessageReader(strings.NewReader(input))

	_, err := reader.ReadRequest()
	if err == nil {
		t.Fatalf("esperava erro de parse, obteve nil")
	}

	rpcErr, ok := err.(*Error)
	if !ok || rpcErr.Code != CodeParseError {
		t.Errorf("esperava código ParseError (%d), obteve %v", CodeParseError, err)
	}
}

func TestMessageReader_InvalidRequest(t *testing.T) {
	input := `{"jsonrpc": "1.0", "method": "test"}` + "\n"
	reader := NewMessageReader(strings.NewReader(input))

	_, err := reader.ReadRequest()
	if err == nil {
		t.Fatalf("esperava erro de requisição inválida, obteve nil")
	}

	rpcErr, ok := err.(*Error)
	if !ok || rpcErr.Code != CodeInvalidRequest {
		t.Errorf("esperava código InvalidRequest (%d), obteve %v", CodeInvalidRequest, err)
	}
}

func TestMessageWriter_WriteResponse(t *testing.T) {
	var buf bytes.Buffer
	writer := NewMessageWriter(&buf)

	id := json.RawMessage(`"req-42"`)
	resp := NewSuccessResponse(id, map[string]string{"status": "ok"})

	if err := writer.WriteResponse(resp); err != nil {
		t.Fatalf("erro ao escrever resposta: %v", err)
	}

	output := buf.String()
	if !strings.HasSuffix(output, "\n") {
		t.Errorf("resposta deve terminar com quebra de linha")
	}

	var parsed Response
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("erro ao decodificar JSON gerado: %v", err)
	}

	if parsed.JSONRPC != "2.0" || string(parsed.ID) != `"req-42"` {
		t.Errorf("dados incorretos na resposta: %+v", parsed)
	}
}

func TestMessageReader_EOF(t *testing.T) {
	reader := NewMessageReader(strings.NewReader(""))
	_, err := reader.ReadRequest()
	if err != io.EOF {
		t.Errorf("esperava io.EOF, obteve: %v", err)
	}
}
