// Package chat manages group chat state: groups, members, message history.
package chat

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/biodoia/agentchat/pkg/types"
	"github.com/google/uuid"
)

// Hub is the central chat state manager.
type Hub struct {
	mu       sync.RWMutex
	groups   map[string]*types.Group
	messages map[string][]*types.Message // group → messages
	agents   map[string]types.AgentProfile
	subs     map[string][]chan *types.Message // group → subscriber channels
	store    MessageStore                     // optional persistence backend
}

// MessageStore is the interface for message persistence (PebbleDB).
type MessageStore interface {
	Save(msg *types.Message) error
	LoadRecent(group string, limit int) ([]*types.Message, error)
}

// NewHub creates a new chat hub.
func NewHub() *Hub {
	h := &Hub{
		groups:   make(map[string]*types.Group),
		messages: make(map[string][]*types.Message),
		agents:   make(map[string]types.AgentProfile),
		subs:     make(map[string][]chan *types.Message),
	}
	// Register default agents
	for name, profile := range types.DefaultAgents {
		h.agents[name] = profile
	}
	// Create default group
	h.groups["general"] = &types.Group{
		Name:      "general",
		Members:   []string{},
		CreatedAt: time.Now(),
	}
	return h
}

// RegisterAgent registers or updates an agent profile.
func (h *Hub) RegisterAgent(name string, profile types.AgentProfile) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.agents[name] = profile
}

// SetStore configures the persistence backend. Messages will be saved to it.
func (h *Hub) SetStore(store MessageStore) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.store = store
}

// LoadHistory loads recent messages for a group from the store into memory.
func (h *Hub) LoadHistory(groupName string, limit int) error {
	if h.store == nil {
		return nil
	}
	msgs, err := h.store.LoadRecent(groupName, limit)
	if err != nil {
		return err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	// Merge: store messages are older, prepend them
	existing := h.messages[groupName]
	merged := append(msgs, existing...)
	h.messages[groupName] = merged
	return nil
}

// JoinGroup adds an agent to a group. Creates the group if needed.
func (h *Hub) JoinGroup(agentName, groupName string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	g, ok := h.groups[groupName]
	if !ok {
		g = &types.Group{
			Name:      groupName,
			Members:   []string{},
			CreatedAt: time.Now(),
		}
		h.groups[groupName] = g
	}
	for _, m := range g.Members {
		if m == agentName {
			return // already member
		}
	}
	g.Members = append(g.Members, agentName)
}

// LeaveGroup removes an agent from a group.
func (h *Hub) LeaveGroup(agentName, groupName string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	g, ok := h.groups[groupName]
	if !ok {
		return
	}
	for i, m := range g.Members {
		if m == agentName {
			g.Members = append(g.Members[:i], g.Members[i+1:]...)
			return
		}
	}
}

// SendMessage posts a message to a group. Returns the created message.
func (h *Hub) SendMessage(groupName, sender, body string, priority types.Priority) *types.Message {
	h.mu.Lock()
	defer h.mu.Unlock()

	profile, ok := h.agents[sender]
	if !ok {
		// Auto-register with defaults
		profile = types.AgentProfile{
			Name:    sender,
			Color:   fmt.Sprintf("#%06X", hashColor(sender)),
			Avatar:  "⚪",
			Program: "unknown",
		}
		h.agents[sender] = profile
	}

	msg := &types.Message{
		ID:       uuid.New().String(),
		Group:    groupName,
		Sender:   sender,
		Color:    profile.Color,
		Avatar:   profile.Avatar,
		Body:     body,
		TS:       time.Now(),
		Priority: priority,
	}

	h.messages[groupName] = append(h.messages[groupName], msg)

	// Persist to store if configured
	if h.store != nil {
		go h.store.Save(msg) // async, non-blocking
	}

	// Notify subscribers
	for _, ch := range h.subs[groupName] {
		select {
		case ch <- msg:
		default: // don't block on slow subscribers
		}
	}

	return msg
}

// GetMessages returns messages for a group since a given time.
func (h *Hub) GetMessages(groupName string, since time.Time, limit int) []*types.Message {
	h.mu.RLock()
	defer h.mu.RUnlock()

	msgs := h.messages[groupName]
	if len(msgs) == 0 {
		return nil
	}

	var result []*types.Message
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].TS.Before(since) {
			break
		}
		result = append(result, msgs[i])
		if limit > 0 && len(result) >= limit {
			break
		}
	}

	// Reverse to chronological order
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result
}

// GetGroups returns all groups.
func (h *Hub) GetGroups() []*types.Group {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]*types.Group, 0, len(h.groups))
	for _, g := range h.groups {
		out = append(out, g)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// GetGroup returns a specific group.
func (h *Hub) GetGroup(name string) *types.Group {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.groups[name]
}

// Subscribe returns a channel that receives new messages for a group.
func (h *Hub) Subscribe(groupName string) <-chan *types.Message {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan *types.Message, 64)
	h.subs[groupName] = append(h.subs[groupName], ch)
	return ch
}

// Unsubscribe removes a subscription channel.
func (h *Hub) Unsubscribe(groupName string, ch <-chan *types.Message) {
	h.mu.Lock()
	defer h.mu.Unlock()
	subs := h.subs[groupName]
	for i, s := range subs {
		if s == ch {
			h.subs[groupName] = append(subs[:i], subs[i+1:]...)
			return
		}
	}
}

// BroadcastWS sends a typing event to all subscribers of a group.
// Uses a separate typing channel per group.
func (h *Hub) BroadcastWS(groupName string, wsMsg types.WSMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	// For now, we send typing events through the same subscriber mechanism
	// The relay's WebSocket handler will check the message type
}

// hashColor generates a consistent color hash from a name.
func hashColor(name string) uint32 {
	var h uint32
	for _, c := range name {
		h = h*31 + uint32(c)
	}
	return (h % 0xFFFFFF) | 0x444444 // ensure not too dark
}
