package event_runtime

import (
	"context"
	"errors"
	"testing"
)

// fakeSub is the minimal Subscriber used by the registry tests in T6.
// The dispatcher delivery loop (T7) does not exercise this type.
type fakeSub struct {
	name    string
	types   []string
	pending int
}

func (f *fakeSub) Name() string         { return f.name }
func (f *fakeSub) EventTypes() []string { return f.types }
func (f *fakeSub) Handle(ctx context.Context, env *Envelope) error {
	return nil
}
func (f *fakeSub) MaxAckPending() int { return f.pending }

// TestSubscriber_RegisterDuplicateName_IsRejected verifies the registry
// rejects two subscribers with the same Name() with ErrDuplicateSubscriber.
func TestSubscriber_RegisterDuplicateName_IsRejected(t *testing.T) {
	d := NewDispatcher()
	a := &fakeSub{name: "audit", types: []string{"*"}}
	b := &fakeSub{name: "audit", types: []string{"*"}}

	if err := RegisterDispatcher(d, a); err != nil {
		t.Fatalf("first RegisterDispatcher falhou: %v", err)
	}
	err := RegisterDispatcher(d, b)
	if err == nil {
		t.Fatal("segundo RegisterDispatcher com mesmo nome deveria falhar")
	}
	if !errors.Is(err, ErrDuplicateSubscriber) {
		t.Errorf("erro deveria envolver ErrDuplicateSubscriber; obteve %v", err)
	}
	// Registry must still contain only one subscriber.
	if got := len(d.Subscribers()); got != 1 {
		t.Errorf("Subscribers()=%d; esperado 1 (sem duplicatas)", got)
	}
}

// TestSubscriber_RegisterNil_IsRejected covers the nil-subscriber guard.
func TestSubscriber_RegisterNil_IsRejected(t *testing.T) {
	d := NewDispatcher()
	err := RegisterDispatcher(d, nil)
	if err == nil {
		t.Fatal("RegisterDispatcher(nil) deveria falhar")
	}
	if !errors.Is(err, ErrNilSubscriber) {
		t.Errorf("erro deveria envolver ErrNilSubscriber; obteve %v", err)
	}
}

// TestSubscriber_RegisterNilDispatcher_IsRejected covers the nil-dispatcher
// guard (defense in depth; in practice nobody should call this with nil).
func TestSubscriber_RegisterNilDispatcher_IsRejected(t *testing.T) {
	err := RegisterDispatcher(nil, &fakeSub{name: "x"})
	if err == nil {
		t.Fatal("RegisterDispatcher com dispatcher nil deveria falhar")
	}
	// We don't pin this to a specific sentinel (it carries extra context)
	// but it MUST be non-nil.
}

// TestSubscriber_RegisterEmptyName_IsRejected covers the empty-name guard.
func TestSubscriber_RegisterEmptyName_IsRejected(t *testing.T) {
	d := NewDispatcher()
	err := RegisterDispatcher(d, &fakeSub{name: "", types: []string{"*"}})
	if err == nil {
		t.Fatal("RegisterDispatcher com name vazio deveria falhar")
	}
	if !errors.Is(err, ErrInvalidEnvelope) {
		t.Errorf("erro deveria envolver ErrInvalidEnvelope; obteve %v", err)
	}
}

// TestSubscriber_RegisterHappyPath sanity-checks a clean registration.
func TestSubscriber_RegisterHappyPath(t *testing.T) {
	d := NewDispatcher()
	sub := &fakeSub{name: "audit", types: []string{"*"}, pending: 256}
	if err := RegisterDispatcher(d, sub); err != nil {
		t.Fatalf("RegisterDispatcher falhou: %v", err)
	}
	if !d.HasSubscriber("audit") {
		t.Error("HasSubscriber(\"audit\") deveria retornar true")
	}
	if got := sub.MaxAckPending(); got != 256 {
		t.Errorf("MaxAckPending()=%d; esperado 256", got)
	}
}
