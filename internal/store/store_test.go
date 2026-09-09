package store

import (
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/biodoia/agentchat/pkg/types"
)

func newTestStore(t *testing.T) (*MessageStore, func()) {
	t.Helper()
	dir := t.TempDir()
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	s, err := New(dir, log)
	if err != nil {
		t.Fatal(err)
	}
	return s, func() { s.Close() }
}

func TestSaveAndLoad(t *testing.T) {
	s, cleanup := newTestStore(t)
	defer cleanup()

	msg := &types.Message{
		ID:    "test-1",
		Group: "general",
		Sender: "Alice",
		Body:  "hello",
		TS:    time.Now(),
	}
	if err := s.Save(msg); err != nil {
		t.Fatal(err)
	}

	msgs, err := s.LoadRecent("general", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Body != "hello" {
		t.Errorf("expected 'hello', got %s", msgs[0].Body)
	}
}

func TestSaveBatch(t *testing.T) {
	s, cleanup := newTestStore(t)
	defer cleanup()

	now := time.Now()
	msgs := []*types.Message{
		{ID: "b1", Group: "g", Sender: "A", Body: "msg1", TS: now},
		{ID: "b2", Group: "g", Sender: "B", Body: "msg2", TS: now.Add(time.Second)},
		{ID: "b3", Group: "g", Sender: "C", Body: "msg3", TS: now.Add(2 * time.Second)},
	}
	if err := s.SaveBatch(msgs); err != nil {
		t.Fatal(err)
	}

	loaded, err := s.LoadRecent("g", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 3 {
		t.Fatalf("expected 3, got %d", len(loaded))
	}
}

func TestLoadRecentLimit(t *testing.T) {
	s, cleanup := newTestStore(t)
	defer cleanup()

	for i := 0; i < 10; i++ {
		s.Save(&types.Message{
			ID:    fmt.Sprintf("m%d", i),
			Group: "g",
			Body:  fmt.Sprintf("msg%d", i),
			TS:    time.Now().Add(time.Duration(i) * time.Second),
		})
	}

	msgs, err := s.LoadRecent("g", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 3 {
		t.Fatalf("expected 3, got %d", len(msgs))
	}
	if msgs[0].Body != "msg9" {
		t.Errorf("expected newest first 'msg9', got %s", msgs[0].Body)
	}
}

func TestLoadSince(t *testing.T) {
	s, cleanup := newTestStore(t)
	defer cleanup()

	s.Save(&types.Message{ID: "old", Group: "g", Body: "old", TS: time.Now().Add(-time.Hour)})
	cutoff := time.Now()
	s.Save(&types.Message{ID: "new", Group: "g", Body: "new", TS: time.Now().Add(time.Second)})

	msgs, err := s.LoadSince("g", cutoff)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 || msgs[0].Body != "new" {
		t.Errorf("expected only 'new', got %v", msgs)
	}
}

func TestIsolationBetweenGroups(t *testing.T) {
	s, cleanup := newTestStore(t)
	defer cleanup()

	s.Save(&types.Message{ID: "a", Group: "alpha", Body: "a", TS: time.Now()})
	s.Save(&types.Message{ID: "b", Group: "beta", Body: "b", TS: time.Now()})

	alpha, _ := s.LoadRecent("alpha", 10)
	beta, _ := s.LoadRecent("beta", 10)
	if len(alpha) != 1 || len(beta) != 1 {
		t.Errorf("expected isolation: alpha=%d beta=%d", len(alpha), len(beta))
	}
}
