package jsonrpc

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

// FrameHeaderSize is the fixed length of the binary framing prefix.
// Every frame on the wire is `FrameHeaderSize bytes big-endian length
// || payload of exactly that length`. We use a fixed-width prefix
// (not varint) because audio payloads are large and predictable, so
// the 4-byte per-frame overhead is dominated by the saved copy cost
// on hot paths.
const FrameHeaderSize = 4

// MaxFrameSize caps any single frame. A malformed length prefix
// claiming 1 GiB is rejected before we attempt the allocation so
// an attacker can't OOM the server with a single byte of input.
const MaxFrameSize = 16 << 20 // 16 MiB

// SocketPath builds the canonical path for the unix socket on this
// platform. On POSIX the path is "/tmp/mymemoryd-<pid>.sock" per the
// spec. On Windows net.Listen("unix", ...) maps that to a named
// pipe — the literal "tmp" segment is harmless but uninformative,
// so we route through the OS temp dir.
func SocketPath(pid int) string {
	return filepath.Join(os.TempDir(), "mymemoryd-"+strconv.Itoa(pid)+".sock")
}

// ServeUnix binds a Unix socket at path and serves JSON-RPC frames
// using 4-byte big-endian length-prefix framing. Each accepted
// connection runs in its own goroutine; the function returns when
// ctx is cancelled or the listener fails.
//
// The Server is reused across connections — handler registry and
// initialized flag are shared so workers that connect mid-session
// see the same capabilities the first worker negotiated.
func (s *Server) ServeUnix(ctx context.Context, path string) error {
	if path == "" {
		return errors.New("jsonrpc: unix socket path is empty")
	}
	ln, err := net.Listen("unix", path)
	if err != nil {
		return fmt.Errorf("jsonrpc: listen unix %s: %w", path, err)
	}
	defer ln.Close()

	var wg sync.WaitGroup
	defer wg.Wait() // drain in-flight connections before returning

	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("jsonrpc: accept: %w", err)
		}
		wg.Add(1)
		go func(c net.Conn) {
			defer wg.Done()
			s.serveConn(ctx, c)
		}(conn)
	}
}

// serveConn handles a single connection. Frames are read using
// 4-byte big-endian length prefix; responses are written with the
// same framing. The loop exits on EOF, ctx cancellation, or a
// fatal read/write error.
func (s *Server) serveConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		frame, err := readFrame(conn)
		if err != nil {
			// EOF and ctx cancellation are normal shutdown signals.
			if errors.Is(err, io.EOF) || ctx.Err() != nil {
				return
			}
			// Truncated frame: reply with a single length-prefixed
			// error envelope so the caller can distinguish "partial
			// write" from "connection lost". Per spec AC 4 the
			// server should fail-closed.
			if isTruncatedFrame(err) {
				resp := marshalError(nil, &Error{
					Code:    CodeInvalidRequest,
					Message: "truncated frame: " + err.Error(),
				})
				_ = writeFrame(conn, resp)
				return
			}
			// Anything else (oversize, IO error) — close.
			return
		}
		resp := s.processFrame(ctx, frame)
		if resp == nil {
			continue
		}
		if err := writeFrame(conn, resp); err != nil {
			return
		}
	}
}

// readFrame reads exactly one frame from r: 4 bytes big-endian
// length followed by that many payload bytes. Returns the payload
// (without the header). Errors:
//   - io.EOF when the reader is closed cleanly.
//   - ErrTruncatedFrame when the connection closes mid-frame.
//   - ErrFrameTooLarge when the claimed length exceeds MaxFrameSize.
func readFrame(r io.Reader) ([]byte, error) {
	var hdr [FrameHeaderSize]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(hdr[:])
	if length > MaxFrameSize {
		return nil, fmt.Errorf("jsonrpc: frame too large: %d > %d", length, MaxFrameSize)
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, fmt.Errorf("%w: read %d bytes: %v", ErrTruncatedFrame, length, err)
	}
	return payload, nil
}

// writeFrame writes length-prefixed payload to w. Used for both
// response frames and (rare) error envelopes emitted when a
// malformed frame arrives.
func writeFrame(w io.Writer, payload []byte) error {
	if len(payload) > MaxFrameSize {
		return fmt.Errorf("jsonrpc: outgoing frame too large: %d > %d", len(payload), MaxFrameSize)
	}
	var hdr [FrameHeaderSize]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(payload)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

// ErrTruncatedFrame is returned by readFrame when the connection
// closes before the payload bytes arrive. Detected by
// isTruncatedFrame below so serveConn can answer with a length-
// prefixed error envelope and close.
var ErrTruncatedFrame = errors.New("jsonrpc: truncated frame")

// isTruncatedFrame reports whether err originates from a partial
// frame (the caller closed mid-payload). We use errors.Is so
// wrapped errors from io.ReadFull still match.
func isTruncatedFrame(err error) bool {
	return errors.Is(err, ErrTruncatedFrame) || errors.Is(err, io.ErrUnexpectedEOF)
}
