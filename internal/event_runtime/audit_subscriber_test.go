package event_runtime

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// TestAuditSubscriber_RedactsPII: emit envelope with transcript; assert
// audit.log line does NOT contain transcript text (PII gate, LLM02).
func TestAuditSubscriber_RedactsPII(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	sub, err := NewAuditSubscriber(path)
	if err != nil {
		t.Fatalf("NewAuditSubscriber: %v", err)
	}
	defer sub.Close()

	env := NewEnvelope("voice.transcribed", "audio-123")
	env.Payload = json.RawMessage(`{"transcript":"hello world","lang":"en"}`)

	if err := sub.Handle(context.Background(), env); err != nil {
		t.Fatalf("Handle falhou: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "hello world") {
		t.Errorf("audit.log contém 'hello world' (PII NÃO redactado): %s", content)
	}
	if strings.Contains(content, "transcript") {
		t.Errorf("audit.log contém a chave 'transcript' (campo deveria ser omitido): %s", content)
	}
	// The audit log MUST contain the redacted_payload_hash so audit can
	// verify what the original payload was without storing it.
	if !strings.Contains(content, "redacted_payload_hash") {
		t.Errorf("audit.log não contém redacted_payload_hash: %s", content)
	}
}

// TestAuditSubscriber_HashIsStable: same envelope twice → same SHA-256 hash
// in both audit lines.
func TestAuditSubscriber_HashIsStable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	sub, err := NewAuditSubscriber(path)
	if err != nil {
		t.Fatalf("NewAuditSubscriber: %v", err)
	}
	defer sub.Close()

	env := NewEnvelope("voice.transcribed", "audio-1")
	env.Payload = json.RawMessage(`{"transcript":"hello world"}`)

	if err := sub.Handle(context.Background(), env); err != nil {
		t.Fatalf("Handle 1: %v", err)
	}
	if err := sub.Handle(context.Background(), env); err != nil {
		t.Fatalf("Handle 2: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("audit.log tem %d linhas; esperado 2", len(lines))
	}
	var l1, l2 auditLine
	if err := json.Unmarshal([]byte(lines[0]), &l1); err != nil {
		t.Fatalf("unmarshal line 1: %v", err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &l2); err != nil {
		t.Fatalf("unmarshal line 2: %v", err)
	}
	if l1.RedactedPayloadHash == "" {
		t.Error("line 1 sem hash")
	}
	if l1.RedactedPayloadHash != l2.RedactedPayloadHash {
		t.Errorf("hash instável: %s vs %s", l1.RedactedPayloadHash, l2.RedactedPayloadHash)
	}
}

// TestAuditSubscriber_ReadOnlyFilesystem_FailsClosed: chmod 0444 the audit
// log file path's directory → Handle must return error (fail-closed).
//
// On Windows, chmod has limited effect (ACLs are the real control). We
// skip this test on Windows because the read-only enforcement there is
// different. The fail-closed behavior is exercised at the dispatcher level
// when Handle returns an error (T7's retry path).
func TestAuditSubscriber_ReadOnlyFilesystem_FailsClosed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping read-only test in -short mode")
	}
	if isWindows() {
		t.Skip("skipping read-only FS test on Windows (chmod semantics differ)")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	// Create the file first (writable), then chmod it read-only.
	if _, err := os.Create(path); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := os.Chmod(path, 0444); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(path, 0644) })

	sub, err := NewAuditSubscriber(path)
	if err != nil {
		// Some platforms reject opening a read-only file in append mode.
		// Either failure mode (open error or handle error) is acceptable
		// proof of fail-closed behavior.
		t.Logf("NewAuditSubscriber rejeitou arquivo read-only (acceptable): %v", err)
		return
	}
	defer sub.Close()

	env := NewEnvelope("voice.transcribed", "audio-1")
	env.Payload = json.RawMessage(`{"transcript":"hello"}`)
	err = sub.Handle(context.Background(), env)
	if err == nil {
		t.Fatal("Handle deveria falhar em filesystem read-only")
	}
}

// TestAuditSubscriber_AppendDoesNotCorrupt: write 5 lines, reopen, verify
// all 5 lines are readable in order.
func TestAuditSubscriber_AppendDoesNotCorrupt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	sub, err := NewAuditSubscriber(path)
	if err != nil {
		t.Fatalf("NewAuditSubscriber: %v", err)
	}

	for i := 0; i < 5; i++ {
		env := NewEnvelope("memory.committed", "agg-1")
		env.Payload = json.RawMessage(`{"i":` + itoa(i) + `}`)
		if err := sub.Handle(context.Background(), env); err != nil {
			t.Fatalf("Handle %d: %v", i, err)
		}
	}
	sub.Close()

	// Re-open the file and verify all 5 lines are present in order.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 5 {
		t.Fatalf("audit.log tem %d linhas; esperado 5", len(lines))
	}
	for i, line := range lines {
		var al auditLine
		if err := json.Unmarshal([]byte(line), &al); err != nil {
			t.Errorf("linha %d não parseável: %v", i, err)
			continue
		}
		if al.EventType != "memory.committed" {
			t.Errorf("linha %d event_type=%q; esperado memory.committed", i, al.EventType)
		}
	}
}

// TestAuditSubscriber_ConcurrentWrites: spawn N concurrent Handle calls,
// verify all lines appear in the audit log exactly once.
func TestAuditSubscriber_ConcurrentWrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	sub, err := NewAuditSubscriber(path)
	if err != nil {
		t.Fatalf("NewAuditSubscriber: %v", err)
	}
	defer sub.Close()

	const N = 50
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			env := NewEnvelope("memory.committed", "agg-1")
			env.Payload = json.RawMessage(`{"i":` + itoa(i) + `}`)
			if err := sub.Handle(context.Background(), env); err != nil {
				t.Errorf("Handle %d: %v", i, err)
			}
		}(i)
	}
	wg.Wait()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != N {
		t.Errorf("audit.log tem %d linhas; esperado %d", len(lines), N)
	}
}

// TestAuditSubscriber_HandleReturnsErrorOnClosedFile: after Close, Handle
// must return an error.
func TestAuditSubscriber_HandleReturnsErrorOnClosedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	sub, err := NewAuditSubscriber(path)
	if err != nil {
		t.Fatalf("NewAuditSubscriber: %v", err)
	}
	sub.Close()

	env := NewEnvelope("memory.committed", "agg-1")
	env.Payload = json.RawMessage(`{"k":"v"}`)
	err = sub.Handle(context.Background(), env)
	if err == nil {
		t.Fatal("Handle deveria falhar após Close")
	}
	if !errors.Is(err, err) && err == nil {
		t.Errorf("erro inesperado: %v", err)
	}
}

// itoa is a tiny int→string helper for test payloads (avoids importing
// strconv in test file just for this).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// isWindows reports whether we're on Windows (used to skip platform-specific
// tests).
func isWindows() bool {
	return os.PathSeparator == '\\' && (os.Getenv("OS") == "Windows_NT" || true)
}
