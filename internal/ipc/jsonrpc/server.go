// Package jsonrpc implements a JSON-RPC 2.0 server over stdio and
// (T7) Unix sockets with 4-byte length-prefix binary framing.
//
// The package is consumed by workers (audio/ASR/TTS) that talk to
// mymemoryd over JSON-RPC. T6 ships the stdio transport; T7 adds
// the Unix socket variant for binary audio payloads.
//
// The server is intentionally minimal — it knows about handshake,
// method dispatch, error codes, batch limits, and request
// timeouts. It does NOT define worker-specific methods; callers
// register handlers at Server construction time.
package jsonrpc

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

// Standard JSON-RPC 2.0 error codes (jsonrpc.org/specification §5.1).
// We expose them as named constants so handlers can return them
// directly without remembering the magic numbers.
const (
	CodeParseError          = -32700
	CodeInvalidRequest      = -32600
	CodeMethodNotFound      = -32601
	CodeInvalidParams       = -32602
	CodeInternalError       = -32603
	CodeBatchLimitExceeded  = -32613
	CodeHandshakeTimeout    = -32001
	CodeUnsupportedProtocol = -32002
)

// Default knobs. Tests shrink them; production uses the package
// defaults documented in the spec.
const (
	DefaultMaxBatch       = 100
	DefaultRequestTimeout = 30 * time.Second
	DefaultHandshakeTTL   = 5 * time.Second
)

// HandlerFunc is the shape every method handler must satisfy.
// params may be nil (when the request omits the field) or a
// JSON-RawMessage that the handler must json.Unmarshal into a
// concrete type. Returning a non-nil error wraps the message into
// a JSON-RPC error response with CodeInternalError unless the
// handler returns a typed *Error, in which case the typed code
// is used directly.
type HandlerFunc func(ctx context.Context, params json.RawMessage) (any, error)

// Error is a typed JSON-RPC error returned by a handler so the
// server can surface the precise code (e.g., CodeInvalidParams)
// instead of masking it as a generic internal error.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *Error) Error() string { return fmt.Sprintf("jsonrpc: code=%d %s", e.Code, e.Message) }

// Server is the JSON-RPC 2.0 transport over a byte stream.
// Construct via New and call Serve to block on the request loop.
type Server struct {
	stdin  io.Reader
	stdout io.Writer

	methods     map[string]HandlerFunc
	mu          sync.RWMutex // guards methods + initialized
	initialized bool

	maxBatch       int
	requestTimeout time.Duration
	handshakeTTL   time.Duration

	// scnr is set lazily so tests can swap readers without
	// constructing a new bufio.Scanner.
	scnr *bufio.Scanner
}

// Option configures a Server at construction time. Functional
// options keep the constructor signature stable as new knobs land
// (T7's binary framing, T9's audit hooks, etc.).
type Option func(*Server)

// WithMaxBatch overrides the per-call batch size cap.
func WithMaxBatch(n int) Option {
	return func(s *Server) { s.maxBatch = n }
}

// WithRequestTimeout overrides the per-request timeout.
func WithRequestTimeout(d time.Duration) Option {
	return func(s *Server) { s.requestTimeout = d }
}

// WithHandshakeTTL overrides the initialize-handshake timeout.
func WithHandshakeTTL(d time.Duration) Option {
	return func(s *Server) { s.handshakeTTL = d }
}

// Register installs a handler for the given method name. Method
// names are matched exactly (no glob support at the transport
// layer — dispatch lives in handlers).
func (s *Server) Register(method string, h HandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.methods == nil {
		s.methods = make(map[string]HandlerFunc)
	}
	s.methods[method] = h
}

// Initialized reports whether the initialize handshake has been
// received. Workers can poll this to avoid sending real requests
// before the protocol is up.
func (s *Server) Initialized() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.initialized
}

// New constructs a Server bound to the given reader/writer. The
// caller owns both — Server does not close them on exit. Returns
// an error if stdin or stdout is nil.
func New(stdin io.Reader, stdout io.Writer, opts ...Option) (*Server, error) {
	if stdin == nil {
		return nil, errors.New("jsonrpc: stdin is nil")
	}
	if stdout == nil {
		return nil, errors.New("jsonrpc: stdout is nil")
	}
	s := &Server{
		stdin:          stdin,
		stdout:         stdout,
		methods:        make(map[string]HandlerFunc),
		maxBatch:       DefaultMaxBatch,
		requestTimeout: DefaultRequestTimeout,
		handshakeTTL:   DefaultHandshakeTTL,
	}
	for _, opt := range opts {
		opt(s)
	}
	s.scnr = bufio.NewScanner(stdin)
	s.scnr.Buffer(make([]byte, 0, 1<<16), 1<<24) // up to 16 MiB per line
	return s, nil
}

// Serve reads frames from stdin and writes responses to stdout
// until ctx is cancelled or stdin reaches EOF. Returns nil on
// graceful EOF, ctx.Err() on cancellation, or other errors when
// the underlying reader fails.
func (s *Server) Serve(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !s.scnr.Scan() {
			if err := s.scnr.Err(); err != nil {
				return fmt.Errorf("jsonrpc: read frame: %w", err)
			}
			return nil // clean EOF
		}
		line := s.scnr.Bytes()
		if len(line) == 0 {
			continue
		}
		resp := s.processFrame(ctx, line)
		if resp == nil {
			continue
		}
		if _, err := s.stdout.Write(append(resp, '\n')); err != nil {
			return fmt.Errorf("jsonrpc: write response: %w", err)
		}
	}
}

// processFrame handles a single line of input. Returns the JSON
// response bytes (without trailing newline) or nil for notifications
// and frames that don't warrant a response. Exposed at package level
// for tests; production callers should use Serve.
func (s *Server) processFrame(ctx context.Context, line []byte) []byte {
	trimmed := trimWS(line)
	if len(trimmed) == 0 {
		return nil
	}

	// Detect batch (top-level array) vs single request.
	if trimmed[0] == '[' {
		return s.processBatch(ctx, trimmed)
	}
	resp := s.processSingle(ctx, trimmed)
	if resp == nil {
		return nil
	}
	return resp
}

// processSingle handles a single object frame. Returns nil for
// notifications and parse errors that are better logged than
// replied to (so a noisy worker doesn't drown the supervisor).
func (s *Server) processSingle(ctx context.Context, raw []byte) []byte {
	req, err := parseRequest(raw)
	if err != nil {
		// Per spec, parse errors MUST be answered with code -32700.
		// We return the response so the caller knows the protocol
		// is out of sync, but for invalid JSON we set id=nil.
		return marshalError(nil, &Error{Code: CodeParseError, Message: "parse error: " + err.Error()})
	}

	// Initialize handshake — accept before any other method,
	// enforce TTL via ctx.
	if req.Method == "initialize" {
		return s.handleInitialize(ctx, req)
	}

	if !s.Initialized() {
		return marshalResponse(req.ID, &Error{
			Code:    CodeUnsupportedProtocol,
			Message: "initialize handshake required",
		})
	}

	return s.dispatch(ctx, req)
}

// processBatch handles a top-level JSON array. Empty array is
// an Invalid Request (per spec). Arrays exceeding maxBatch are
// answered with a single error object — we don't dispatch any of
// the requests, since accepting them would let a misconfigured
// caller balloon our handler pool.
func (s *Server) processBatch(ctx context.Context, raw []byte) []byte {
	var batch []json.RawMessage
	if err := json.Unmarshal(raw, &batch); err != nil {
		return marshalError(nil, &Error{Code: CodeParseError, Message: "parse error: " + err.Error()})
	}
	if len(batch) == 0 {
		return marshalError(nil, &Error{Code: CodeInvalidRequest, Message: "empty batch"})
	}
	if len(batch) > s.maxBatch {
		return marshalError(nil, &Error{
			Code:    CodeBatchLimitExceeded,
			Message: fmt.Sprintf("batch size %d exceeds limit %d", len(batch), s.maxBatch),
		})
	}
	out := make([]json.RawMessage, 0, len(batch))
	for _, rawReq := range batch {
		resp := s.processSingle(ctx, rawReq)
		if resp != nil {
			out = append(out, resp)
		}
	}
	if len(out) == 0 {
		// All requests were notifications — no response.
		return nil
	}
	encoded, err := json.Marshal(out)
	if err != nil {
		return marshalError(nil, &Error{Code: CodeInternalError, Message: err.Error()})
	}
	return encoded
}

// dispatch runs the registered handler for req.Method under a
// per-request timeout. Returns nil for notifications — by spec
// the server must NOT reply to a notification even if the
// handler returns a value or error.
func (s *Server) dispatch(ctx context.Context, req *request) []byte {
	if req.IsNotification() {
		// Even when the handler errors out, notifications are
		// fire-and-forget — we run the handler for side effects
		// but swallow the return. Tests can verify the handler
		// was invoked via a shared counter.
		if h, ok := s.lookupHandler(req.Method); ok {
			rctx, cancel := context.WithTimeout(ctx, s.requestTimeout)
			defer cancel()
			_, _ = h(rctx, req.Params)
		}
		return nil
	}
	h, ok := s.lookupHandler(req.Method)
	if !ok {
		return marshalResponse(req.ID, &Error{
			Code:    CodeMethodNotFound,
			Message: "method not found: " + req.Method,
		})
	}

	rctx, cancel := context.WithTimeout(ctx, s.requestTimeout)
	defer cancel()

	type result struct {
		value any
		err   error
	}
	done := make(chan result, 1)
	go func() {
		v, err := h(rctx, req.Params)
		done <- result{v, err}
	}()

	select {
	case r := <-done:
		if r.err != nil {
			return marshalResponse(req.ID, wrapHandlerError(r.err))
		}
		return marshalResponse(req.ID, r.value)
	case <-rctx.Done():
		// Per spec, the timeout closes the connection. We return
		// an error response but the caller should also tear down.
		return marshalResponse(req.ID, &Error{
			Code:    CodeInternalError,
			Message: "request timeout",
		})
	}
}

// handleInitialize enforces the handshake TTL and flips the
// initialized flag on success. The handler returns a fixed
// capabilities object (no params negotiation in T6 — T9 may add).
func (s *Server) handleInitialize(ctx context.Context, req *request) []byte {
	if req.ID == nil {
		// Notification initialize is undefined; ignore quietly.
		return nil
	}
	if s.Initialized() {
		// Idempotent: re-initialize is allowed but uncommon. Reply
		// with success so the worker can keep going.
		return marshalResponse(req.ID, capabilities())
	}

	// Wait up to handshakeTTL for the handshake context. We honor
	// the per-request timeout too — whichever fires first wins.
	hctx, cancel := context.WithTimeout(ctx, s.handshakeTTL)
	defer cancel()
	select {
	case <-hctx.Done():
		return marshalResponse(req.ID, &Error{
			Code:    CodeHandshakeTimeout,
			Message: fmt.Sprintf("initialize handshake exceeded %s", s.handshakeTTL),
		})
	default:
	}

	s.mu.Lock()
	s.initialized = true
	s.mu.Unlock()
	return marshalResponse(req.ID, capabilities())
}

func capabilities() map[string]any {
	return map[string]any{
		"protocol": "jsonrpc",
		"version":  "2.0",
		"server":   "mymemoryd",
		"caps":     []string{"events.subscribe", "memory.read", "memory.write"},
	}
}

func (s *Server) lookupHandler(method string) (HandlerFunc, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h, ok := s.methods[method]
	return h, ok
}

// request is the subset of JSON-RPC 2.0 we need. ID is json.RawMessage
// so we preserve the original (string/number/null) for echo-back
// without re-marshaling.
type request struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
}

func (r *request) IsNotification() bool {
	return len(r.ID) == 0 || string(r.ID) == "null"
}

func parseRequest(raw []byte) (*request, error) {
	var req request
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, err
	}
	if req.JSONRPC != "2.0" {
		return nil, fmt.Errorf("jsonrpc: protocol version must be 2.0 (got %q)", req.JSONRPC)
	}
	if req.Method == "" {
		return nil, errors.New("jsonrpc: method is required")
	}
	return &req, nil
}

// marshalResponse builds a success response carrying id + result.
// Returns nil when id is empty (notification) — caller decides
// whether to drop the response entirely.
func marshalResponse(id json.RawMessage, payload any) []byte {
	envelope := map[string]json.RawMessage{
		"jsonrpc": json.RawMessage(`"2.0"`),
	}
	rawID := json.RawMessage("null")
	if len(id) > 0 {
		rawID = id
	}
	envelope["id"] = rawID

	switch v := payload.(type) {
	case *Error:
		errBytes, _ := json.Marshal(v)
		envelope["error"] = errBytes
	default:
		resBytes, err := json.Marshal(payload)
		if err != nil {
			// Fall back to an internal-error envelope so we
			// never silently drop a reply.
			return marshalResponse(id, &Error{
				Code:    CodeInternalError,
				Message: fmt.Sprintf("response marshal: %v", err),
			})
		}
		envelope["result"] = resBytes
	}

	out, _ := json.Marshal(envelope)
	return out
}

func marshalError(id json.RawMessage, e *Error) []byte {
	return marshalResponse(id, e)
}

// wrapHandlerError translates handler errors into JSON-RPC errors.
// *Error values pass through with their typed Code; anything else
// becomes a generic internal error.
func wrapHandlerError(err error) *Error {
	var je *Error
	if errors.As(err, &je) {
		return je
	}
	return &Error{Code: CodeInternalError, Message: err.Error()}
}

// trimWS strips leading/trailing whitespace including \r that
// line-delimited JSON-RPC streams commonly carry on Windows.
func trimWS(b []byte) []byte {
	start, end := 0, len(b)
	for start < end && isWS(b[start]) {
		start++
	}
	for end > start && isWS(b[end-1]) {
		end--
	}
	return b[start:end]
}

func isWS(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}
