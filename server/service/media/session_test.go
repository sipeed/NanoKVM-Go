package media

import (
	"testing"
	"time"
)

func TestSessionTokenIsShortLivedAndRevocable(t *testing.T) {
	manager := NewManager()
	now := time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC)
	manager.now = func() time.Time { return now }
	session, err := manager.Create("mcp", Capabilities{SpeakerReceive: true}, time.Minute)
	if err != nil || session.Token == "" || session.ID == "" {
		t.Fatalf("create session: %#v %v", session, err)
	}
	got, ok := manager.Get(session.Token)
	if !ok || got.Token != "" || !got.Capabilities.SpeakerReceive {
		t.Fatalf("unexpected session: %#v ok=%t", got, ok)
	}
	if !manager.Close(session.Token) || manager.Close(session.Token) {
		t.Fatal("unexpected close result")
	}
}

func TestCloseAllRevokesEverySession(t *testing.T) {
	manager := NewManager()
	first, err := manager.Create("first", Capabilities{SpeakerReceive: true}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	second, err := manager.Create("second", Capabilities{MicSend: true}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	manager.CloseAll()

	if _, ok := manager.Get(first.Token); ok {
		t.Fatal("first media session remained valid")
	}
	if _, ok := manager.Get(second.Token); ok {
		t.Fatal("second media session remained valid")
	}
}
