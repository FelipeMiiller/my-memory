package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
)

// Standard JSON-RPC 2.0 Error Codes
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
)

// Request representa uma mensagem JSON-RPC 2.0 de requisição ou notificação
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// IsNotification verifica se a requisição não espera resposta (id omitido/nulo)
func (r *Request) IsNotification() bool {
	return len(r.ID) == 0 || string(r.ID) == "null"
}

// Response representa a resposta a uma requisição JSON-RPC 2.0
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Error representa a estrutura de erro do JSON-RPC 2.0
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("jsonrpc error %d: %s", e.Code, e.Message)
}

// NewError cria um objeto de erro padronizado
func NewError(code int, message string, data any) *Error {
	return &Error{Code: code, Message: message, Data: data}
}

// NewSuccessResponse cria uma resposta de sucesso
func NewSuccessResponse(id json.RawMessage, result any) *Response {
	return &Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}

// NewErrorResponse cria uma resposta de erro
func NewErrorResponse(id json.RawMessage, code int, message string, data any) *Response {
	return &Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   NewError(code, message, data),
	}
}

// MessageWriter garante escrita segura e atômica linha a linha
type MessageWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func NewMessageWriter(w io.Writer) *MessageWriter {
	return &MessageWriter{w: w}
}

func (mw *MessageWriter) WriteResponse(resp *Response) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	mw.mu.Lock()
	defer mw.mu.Unlock()

	if _, err := mw.w.Write(data); err != nil {
		return err
	}
	_, err = mw.w.Write([]byte("\n"))
	return err
}

// MessageReader lê requisições JSON-RPC linha a linha
type MessageReader struct {
	scanner *bufio.Scanner
}

func NewMessageReader(r io.Reader) *MessageReader {
	return &MessageReader{scanner: bufio.NewScanner(r)}
}

func (mr *MessageReader) ReadRequest() (*Request, error) {
	for mr.scanner.Scan() {
		line := strings.TrimSpace(mr.scanner.Text())
		if line == "" {
			continue
		}

		var req Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			return nil, NewError(CodeParseError, "Parse error: invalid JSON", err.Error())
		}

		if req.JSONRPC != "2.0" || req.Method == "" {
			return nil, NewError(CodeInvalidRequest, "Invalid Request: missing jsonrpc version or method", nil)
		}

		return &req, nil
	}

	if err := mr.scanner.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}
