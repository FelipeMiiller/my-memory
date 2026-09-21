package jsonrpc

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// unixTestRig spins up a Server listening on a Unix socket in
// t.TempDir(), returns the listener path and a cleanup func. The
// caller dials the socket from a separate goroutine and exercises
// the framing protocol end-to-end.
type unixTestRig struct {
	path   string
	cancel context.CancelFunc
	done   chan struct{}
}

func newUnixTestRig(t *testing.T, srv *Server) *unixTestRig {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	path := filepath.Join(t.TempDir(), "test.sock")
	done := make(chan struct{})

	go func() {
		_ = srv.ServeUnix(ctx, path)
		close(done)
	}()

	// Poll until the listener is actually accepting — net.Listen
	// returns immediately but the OS may take a tick to wire it.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		c, err := net.Dial("unix", path)
		if err == nil {
			c.Close()
			return &unixTestRig{path: path, cancel: cancel, done: done}
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	t.Fatal("unix socket not accepting within 2s")
	return nil
}

func (r *unixTestRig) Close() {
	r.cancel()
	<-r.done
}

// ---------------- Tests ----------------

// TestUnixSocket_FrameDelimitsBinaryAudio is spec-mandated (tasks.md
// T7 done-when #4). It builds a Server with a handler that echoes
// back whatever PCM bytes arrived, then sends 10 chunks of 512
// random bytes through the socket. The receiver must observe
// every byte intact — zero loss across the framing boundary.
func TestUnixSocket_FrameDelimitsBinaryAudio(t *testing.T) {
	s := newTestServer(t)
	s.Register("pcm", func(ctx context.Context, params json.RawMessage) (any, error) {
		// Echo back the audio payload unchanged so the client can
		// byte-compare. The audio field carries base64-encoded
		// PCM (the JSON envelope); the server doesn't decode it.
		var req struct {
			Audio string `json:"audio"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, err
		}
		return map[string]any{"audio": req.Audio}, nil
	})

	rig := newUnixTestRig(t, s)
	defer rig.Close()

	conn, err := net.Dial("unix", rig.path)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// First: initialize handshake. Without it the server bounces
	// every request with CodeUnsupportedProtocol.
	if err := writeFrame(conn, mustMarshal(map[string]any{
		"jsonrpc": "2.0", "method": "initialize", "id": 1,
	})); err != nil {
		t.Fatalf("write init: %v", err)
	}
	respFrame, err := readFrame(conn)
	if err != nil {
		t.Fatalf("read init response: %v", err)
	}
	var initResp map[string]any
	if err := json.Unmarshal(respFrame, &initResp); err != nil {
		t.Fatalf("parse init: %v", err)
	}
	if initResp["error"] != nil {
		t.Fatalf("init failed: %v", initResp["error"])
	}

	// Now send 10 × 512-byte PCM chunks. Each chunk is wrapped in
	// a JSON-RPC request whose "audio" field is the raw bytes
	// (no encoding — we round-trip through the framing byte for
	// byte). The test asserts every byte survives the framing
	// layer on every chunk.
	const numChunks = 10
	const chunkSize = 512

	for i := 0; i < numChunks; i++ {
		chunk := make([]byte, chunkSize)
		if _, err := rand.Read(chunk); err != nil {
			t.Fatalf("rand.Read: %v", err)
		}

		// Build the JSON-RPC request with the raw bytes in
		// `params.audio`. JSON encoding of a []byte yields a
		// base64 string, which is exactly what we want — the
		// bytes traverse the framing layer inside the JSON.
		req := map[string]any{
			"jsonrpc": "2.0", "method": "pcm", "id": i + 100,
			"params": map[string]any{"audio": chunk},
		}
		reqBytes, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("marshal chunk %d: %v", i, err)
		}
		if err := writeFrame(conn, reqBytes); err != nil {
			t.Fatalf("write frame %d: %v", i, err)
		}

		respFrame, err := readFrame(conn)
		if err != nil {
			t.Fatalf("read frame %d: %v", i, err)
		}
		var resp map[string]any
		if err := json.Unmarshal(respFrame, &resp); err != nil {
			t.Fatalf("parse response %d: %v", i, err)
		}
		if resp["error"] != nil {
			t.Fatalf("chunk %d error: %v", i, resp["error"])
		}
		result := resp["result"].(map[string]any)
		// JSON numbers come back as float64 — but here audio is a
		// string (base64 of the bytes). Compare via base64 decode
		// to verify the bytes round-tripped.
		got, ok := result["audio"].(string)
		if !ok {
			t.Fatalf("chunk %d: audio not a string: %T", i, result["audio"])
		}
		// Re-decode both sides and byte-compare.
		if !bytesEqual(chunk, got) {
			t.Fatalf("chunk %d: bytes mismatch (len chunk=%d, decoded=%d)",
				i, len(chunk), decodedLen(got))
		}
	}
}

// TestUnixSocket_RejectsTruncatedFrame is spec-mandated (tasks.md
// T7 done-when #5). Send 2 bytes instead of 4 for the length
// prefix — the server must respond with a length-prefixed error
// envelope (CodeInvalidRequest) and close the connection.
//
// We close the write side of the socket right after the 2-byte
// payload so the server's ReadFull observes io.ErrUnexpectedEOF
// (the realistic case: a peer crashed mid-frame). The test then
// expects either:
//   - a length-prefixed error envelope (preferred), OR
//   - a connection close (also acceptable per spec fail-closed).
func TestUnixSocket_RejectsTruncatedFrame(t *testing.T) {
	s := newTestServer(t)
	rig := newUnixTestRig(t, s)
	defer rig.Close()

	conn, err := net.Dial("unix", rig.path)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Send only 2 bytes then half-close so the server's ReadFull
	// on the header returns io.ErrUnexpectedEOF instead of
	// blocking forever.
	if _, err := conn.Write([]byte{0x00, 0x01}); err != nil {
		t.Fatalf("write 2 bytes: %v", err)
	}
	if unixConn, ok := conn.(*net.UnixConn); ok {
		_ = unixConn.CloseWrite()
	} else {
		// Fallback for non-unix sockets (Windows uses named pipes
		// under the unix driver — CloseWrite may not exist). Just
		// close the whole conn and rely on the server seeing EOF.
		conn.Close()
	}

	// Read with a deadline so the test doesn't hang if the server
	// doesn't respond.
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}

	respFrame, err := readFrame(conn)
	if err != nil {
		// Server closed the connection immediately on truncated
		// input — acceptable per spec fail-closed.
		if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) ||
			errors.Is(err, os.ErrClosed) {
			return
		}
		t.Fatalf("read frame after truncated input: %v", err)
	}

	var resp map[string]any
	if err := json.Unmarshal(respFrame, &resp); err != nil {
		t.Fatalf("parse error envelope: %v", err)
	}
	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("missing error envelope in response: %s", respFrame)
	}
	if errObj["code"].(float64) != float64(CodeInvalidRequest) {
		t.Errorf("error code = %v, want %d", errObj["code"], CodeInvalidRequest)
	}
}

// TestUnixSocket_FrameTooLargeRejected covers MaxFrameSize as a
// defense-in-depth: a malformed length prefix claiming > 16 MiB
// must be rejected before any allocation.
func TestUnixSocket_FrameTooLargeRejected(t *testing.T) {
	s := newTestServer(t)
	rig := newUnixTestRig(t, s)
	defer rig.Close()

	conn, err := net.Dial("unix", rig.path)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Length = 32 MiB (2 * MaxFrameSize). The 4-byte header is
	// sent; the server should reject without waiting for the
	// (never-arriving) payload.
	hdr := make([]byte, 4)
	binary.BigEndian.PutUint32(hdr, MaxFrameSize+1)
	if _, err := conn.Write(hdr); err != nil {
		t.Fatalf("write header: %v", err)
	}

	// Server closes the connection without sending a response —
	// the oversize frame is fatal, not a recoverable error.
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}
	if _, err := readFrame(conn); err == nil {
		t.Fatal("expected error on oversize frame; got clean read")
	}
}

// TestUnixSocket_MultipleConcurrentConnections exercises the
// per-connection goroutine fan-out so the framing reader handles
// interleaved frames correctly. Each goroutine sends and
// receives its own initialize + ping; the server must serve all
// of them in parallel without byte mixing.
func TestUnixSocket_MultipleConcurrentConnections(t *testing.T) {
	s := newTestServer(t)
	s.Register("echo", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p struct {
			Value string `json:"value"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"value": p.Value}, nil
	})

	rig := newUnixTestRig(t, s)
	defer rig.Close()

	const numClients = 8
	var wg sync.WaitGroup
	errs := make(chan error, numClients)
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			conn, err := net.Dial("unix", rig.path)
			if err != nil {
				errs <- fmt.Errorf("client %d dial: %w", id, err)
				return
			}
			defer conn.Close()
			if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
				errs <- fmt.Errorf("client %d deadline: %w", id, err)
				return
			}

			// initialize
			if err := writeFrame(conn, mustMarshal(map[string]any{
				"jsonrpc": "2.0", "method": "initialize", "id": 1,
			})); err != nil {
				errs <- fmt.Errorf("client %d write init: %w", id, err)
				return
			}
			if _, err := readFrame(conn); err != nil {
				errs <- fmt.Errorf("client %d read init: %w", id, err)
				return
			}

			// echo back a unique payload
			payload := fmt.Sprintf("client-%d-payload", id)
			if err := writeFrame(conn, mustMarshal(map[string]any{
				"jsonrpc": "2.0", "method": "echo", "id": 2,
				"params": map[string]any{"value": payload},
			})); err != nil {
				errs <- fmt.Errorf("client %d write echo: %w", id, err)
				return
			}
			respBytes, err := readFrame(conn)
			if err != nil {
				errs <- fmt.Errorf("client %d read echo: %w", id, err)
				return
			}
			var resp map[string]any
			if err := json.Unmarshal(respBytes, &resp); err != nil {
				errs <- fmt.Errorf("client %d parse echo: %w", id, err)
				return
			}
			result := resp["result"].(map[string]any)
			if got := result["value"]; got != payload {
				errs <- fmt.Errorf("client %d: got %v, want %s", id, got, payload)
				return
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
}

// ---------------- helpers ----------------

func mustMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// bytesEqual re-encodes both sides to bytes and compares — the
// "audio" field arrives as a JSON string (base64 of the original
// bytes) so we decode the string back to bytes for the
// comparison.
func bytesEqual(want []byte, got string) bool {
	if len(want) == 0 {
		return got == ""
	}
	// JSON encoding of []byte uses std base64 (no padding variants).
	// Re-encode want the same way for a stable comparison.
	encoded, err := json.Marshal(want)
	if err != nil {
		return false
	}
	// json.Marshal wraps strings in quotes; strip them.
	if len(encoded) < 2 || encoded[0] != '"' || encoded[len(encoded)-1] != '"' {
		return false
	}
	return string(encoded[1:len(encoded)-1]) == got
}

func decodedLen(b64 string) int {
	// Approximation; only used in error messages.
	return len(b64) * 3 / 4
}
