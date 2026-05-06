package gmcore_notifier

import (
	"errors"
	"testing"
)

type mockChannel struct {
	name    string
	sendErr error
	sent    bool
}

func (m *mockChannel) Send(n *Notification) error {
	m.sent = true
	return m.sendErr
}

func TestNewNotifier(t *testing.T) {
	n := NewNotifier()
	if n == nil {
		t.Fatal("NewNotifier returned nil")
	}
	if n.channels == nil {
		t.Fatal("channels map should be initialized")
	}
}

func TestNewNotification(t *testing.T) {
	notif := NewNotification("Test Subject", "Test Content")
	if notif == nil {
		t.Fatal("NewNotification returned nil")
	}
	if notif.Subject != "Test Subject" {
		t.Fatalf("unexpected subject: %s", notif.Subject)
	}
	if notif.Content != "Test Content" {
		t.Fatalf("unexpected content: %s", notif.Content)
	}
	if notif.Importance != ImportanceMedium {
		t.Fatalf("expected default ImportanceMedium, got %d", notif.Importance)
	}
}

func TestNotification_SetImportance(t *testing.T) {
	n := NewNotification("subj", "content")
	n.SetImportance(ImportanceHigh)
	if n.Importance != ImportanceHigh {
		t.Fatalf("expected ImportanceHigh, got %d", n.Importance)
	}
}

func TestNotification_AddChannel(t *testing.T) {
	n := NewNotification("subj", "content")
	n.AddChannel("email")
	n.AddChannel("slack")
	if len(n.Channels) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(n.Channels))
	}
}

func TestNotifier_AddChannel(t *testing.T) {
	n := NewNotifier()
	mock := &mockChannel{name: "email"}
	n.AddChannel("email", mock)

	notif := NewNotification("Test", "Body")
	results := n.Send(notif)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !mock.sent {
		t.Fatal("mock channel should have been sent")
	}
}

func TestNotifier_Send_SpecificChannels(t *testing.T) {
	n := NewNotifier()
	emailMock := &mockChannel{name: "email"}
	slackMock := &mockChannel{name: "slack"}
	n.AddChannel("email", emailMock)
	n.AddChannel("slack", slackMock)

	notif := NewNotification("Test", "Body")
	results := n.Send(notif, "email")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !emailMock.sent {
		t.Fatal("email channel should have been sent")
	}
	if slackMock.sent {
		t.Fatal("slack channel should not have been sent")
	}
}

func TestNotifier_Send_NonExistentChannel(t *testing.T) {
	n := NewNotifier()
	notif := NewNotification("Test", "Body")
	results := n.Send(notif, "nonexistent")
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestSentMessage(t *testing.T) {
	n := NewNotifier()
	mock := &mockChannel{name: "test", sendErr: errors.New("failed")}
	n.AddChannel("test", mock)

	notif := NewNotification("Error Test", "Body")
	results := n.Send(notif)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	result := results[0]
	if result.Sent {
		t.Fatal("sent should be false when channel errors")
	}
	if result.Error == nil {
		t.Fatal("error should not be nil")
	}
	if result.Channel != "test" {
		t.Fatalf("expected channel 'test', got %s", result.Channel)
	}
}

func TestImportanceConstants(t *testing.T) {
	if ImportanceLow != 0 {
		t.Fatal("ImportanceLow should be 0")
	}
	if ImportanceMedium != 1 {
		t.Fatal("ImportanceMedium should be 1")
	}
	if ImportanceHigh != 2 {
		t.Fatal("ImportanceHigh should be 2")
	}
}

func TestNewEmailChannel(t *testing.T) {
	ec := NewEmailChannel(nil)
	if ec == nil {
		t.Fatal("NewEmailChannel returned nil")
	}
}

func TestNewSlackChannel(t *testing.T) {
	sc := NewSlackChannel("test-token", "#general")
	if sc == nil {
		t.Fatal("NewSlackChannel returned nil")
	}
	if sc.token != "test-token" {
		t.Fatalf("unexpected token: %s", sc.token)
	}
	if sc.channel != "#general" {
		t.Fatalf("unexpected channel: %s", sc.channel)
	}
}

func TestSlackChannel_Send_NoToken(t *testing.T) {
	sc := NewSlackChannel("", "#general")
	err := sc.Send(NewNotification("Test", "Content"))
	if err != ErrSlackTokenMissing {
		t.Fatalf("expected ErrSlackTokenMissing, got %v", err)
	}
}

func TestEmailChannel_Send_NoRecipients(t *testing.T) {
	ec := NewEmailChannel(nil)
	notif := &Notification{Subject: "Test", Content: "Body"}
	err := ec.Send(notif)
	if err == nil {
		t.Fatal("expected error for no recipients")
	}
}

func TestErrorVariables(t *testing.T) {
	if ErrSlackTokenMissing.Error() != "slack token is missing" {
		t.Fatal("unexpected ErrSlackTokenMissing message")
	}
	if ErrSlackSendFailed.Error() != "failed to send slack message" {
		t.Fatal("unexpected ErrSlackSendFailed message")
	}
}
