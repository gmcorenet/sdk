package gmcore_messenger

import (
	"errors"
	"testing"
)

func TestInMemoryTransport_AckLifecycle(t *testing.T) {
	tx := NewInMemoryTransport()

	err := tx.Send([]interface{}{"m1", "m2"})
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}

	m1, err := tx.Receive()
	if err != nil {
		t.Fatalf("receive failed: %v", err)
	}
	if m1 != "m1" {
		t.Fatalf("expected m1, got %v", m1)
	}

	if err := tx.Ack(m1); err != nil {
		t.Fatalf("ack failed: %v", err)
	}

	m2, err := tx.Receive()
	if err != nil {
		t.Fatalf("receive failed: %v", err)
	}
	if m2 != "m2" {
		t.Fatalf("expected m2, got %v", m2)
	}
}

func TestInMemoryTransport_RejectRequeues(t *testing.T) {
	tx := NewInMemoryTransport()

	err := tx.Send([]interface{}{"m1"})
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}

	m1, err := tx.Receive()
	if err != nil {
		t.Fatalf("receive failed: %v", err)
	}
	if m1 != "m1" {
		t.Fatalf("expected m1, got %v", m1)
	}

	if err := tx.Reject(m1); err != nil {
		t.Fatalf("reject failed: %v", err)
	}

	m1Again, err := tx.Receive()
	if err != nil {
		t.Fatalf("receive failed: %v", err)
	}
	if m1Again != "m1" {
		t.Fatalf("expected rejected message to be requeued, got %v", m1Again)
	}
}

func TestInMemoryTransport_AckUnknownMessage(t *testing.T) {
	tx := NewInMemoryTransport()
	err := tx.Ack("missing")
	if !errors.Is(err, ErrMessageNotInFlight) {
		t.Fatalf("expected ErrMessageNotInFlight, got %v", err)
	}
}

func TestInMemoryTransport_RejectUnknownMessage(t *testing.T) {
	tx := NewInMemoryTransport()
	err := tx.Reject("missing")
	if !errors.Is(err, ErrMessageNotInFlight) {
		t.Fatalf("expected ErrMessageNotInFlight, got %v", err)
	}
}
