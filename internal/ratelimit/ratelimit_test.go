package ratelimit

import (
	"testing"
	"time"
)

func TestNewLimiter(t *testing.T) {
	l := New(10, 20)
	if l == nil {
		t.Fatal("expected non-nil limiter")
	}
}

func TestAllowWithinBurst(t *testing.T) {
	l := New(10, 5)

	// First 5 should be allowed (burst capacity)
	for i := 0; i < 5; i++ {
		if !l.Allow("agent1") {
			t.Errorf("request %d should be allowed", i)
		}
	}
}

func TestDenyOverBurst(t *testing.T) {
	l := New(1, 3)

	// Use up burst
	for i := 0; i < 3; i++ {
		l.Allow("agent1")
	}

	// Next should be denied (no time passed for refill)
	if l.Allow("agent1") {
		t.Error("should be denied after burst exhausted")
	}
}

func TestRefillOverTime(t *testing.T) {
	l := New(100, 1) // 100/sec, burst 1

	l.Allow("agent1") // exhaust burst
	if l.Allow("agent1") {
		t.Error("should be denied immediately after burst")
	}

	// Wait for refill
	time.Sleep(20 * time.Millisecond) // 100/sec * 0.02s = 2 tokens
	if !l.Allow("agent1") {
		t.Error("should be allowed after refill")
	}
}

func TestIsolationBetweenAgents(t *testing.T) {
	l := New(1, 1)

	l.Allow("agent1") // exhaust agent1's burst
	l.Allow("agent2") // use agent2's burst

	// agent1 should be denied, agent2 should also be denied
	if l.Allow("agent1") {
		t.Error("agent1 should be denied")
	}

	// But a fresh agent3 should be allowed
	if !l.Allow("agent3") {
		t.Error("agent3 should be allowed (fresh bucket)")
	}
}

func TestReset(t *testing.T) {
	l := New(1, 1)

	l.Allow("agent1")
	l.Allow("agent1") // denied

	l.Reset()

	// After reset, should be allowed again
	if !l.Allow("agent1") {
		t.Error("should be allowed after reset")
	}
}

func TestStats(t *testing.T) {
	l := New(10, 5)

	tokens, max := l.Stats("unknown")
	if max != 5 {
		t.Errorf("expected max 5, got %d", max)
	}
	if tokens != 5 {
		t.Errorf("expected 5 tokens for unknown agent, got %f", tokens)
	}

	l.Allow("agent1")
	tokens, _ = l.Stats("agent1")
	if tokens != 4 {
		t.Errorf("expected 4 tokens after one use, got %f", tokens)
	}
}
