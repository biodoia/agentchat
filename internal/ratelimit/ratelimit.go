// Package ratelimit provides per-agent rate limiting for AgentChat.
// Uses a token bucket algorithm: each agent gets N tokens per second,
// each message consumes one token. Excess messages are rejected with 429.
package ratelimit

import (
	"sync"
	"time"
)

// Limiter enforces per-agent rate limits.
type Limiter struct {
	mu       sync.Mutex
	agents   map[string]*bucket
	rate     float64 // tokens per second
	capacity int     // max burst
}

// bucket is a token bucket for one agent.
type bucket struct {
	tokens   float64
	lastTime time.Time
}

// New creates a rate limiter with the given rate (tokens/sec) and burst capacity.
func New(rate float64, capacity int) *Limiter {
	return &Limiter{
		agents:   make(map[string]*bucket),
		rate:     rate,
		capacity: capacity,
	}
}

// Allow returns true if the agent is allowed to send a message.
// Returns false if the agent has exceeded the rate limit.
func (l *Limiter) Allow(agent string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.agents[agent]
	now := time.Now()

	if !ok {
		l.agents[agent] = &bucket{
			tokens:   float64(l.capacity) - 1,
			lastTime: now,
		}
		return true
	}

	// Refill tokens based on elapsed time
	elapsed := now.Sub(b.lastTime).Seconds()
	b.tokens += elapsed * l.rate
	if b.tokens > float64(l.capacity) {
		b.tokens = float64(l.capacity)
	}
	b.lastTime = now

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// Reset clears all rate limit state.
func (l *Limiter) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.agents = make(map[string]*bucket)
}

// Stats returns current state for an agent.
func (l *Limiter) Stats(agent string) (tokens float64, max int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	b, ok := l.agents[agent]
	if !ok {
		return float64(l.capacity), l.capacity
	}
	return b.tokens, l.capacity
}
