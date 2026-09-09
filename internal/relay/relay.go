// Package relay provides the HTTP + WebSocket relay server for AgentChat.
// It exposes REST endpoints for message/group management, WebSocket for
// real-time streaming, and A2A protocol support for agent-to-agent communication.
// Messages are persisted via PebbleDB (fgt-sdk L1 cache) and broadcast to
// all WebSocket subscribers in real-time.
package relay

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/biodoia/agentchat/internal/a2aagent"
	"github.com/biodoia/agentchat/internal/chat"
	"github.com/biodoia/agentchat/internal/notify"
	"github.com/biodoia/agentchat/internal/store"
	"github.com/biodoia/agentchat/pkg/types"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Server is the AgentChat relay HTTP server.
type Server struct {
	hub     *chat.Hub
	store   *store.MessageStore
	log     *slog.Logger
	mux     *http.ServeMux
	addr    string
	started time.Time
}

// New creates a relay server. If dataDir is non-empty, enables PebbleDB persistence.
func New(addr string, dataDir string, log *slog.Logger) *Server {
	hub := chat.NewHub()

	// Wire PebbleDB store if dataDir provided
	if dataDir != "" {
		st, err := store.New(dataDir, log)
		if err != nil {
			log.Error("failed to open PebbleDB store", "err", err)
		} else {
			hub.SetStore(st)
			// Load history for default group
			hub.LoadHistory("general", 100)
		}
	}

	s := &Server{
		hub:     hub,
		log:     log,
		mux:     http.NewServeMux(),
		addr:    addr,
		started: time.Now(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	// REST API
	s.mux.HandleFunc("POST /api/message", s.handleSendMessage)
	s.mux.HandleFunc("GET /api/messages", s.handleGetMessages)
	s.mux.HandleFunc("POST /api/group/join", s.handleJoinGroup)
	s.mux.HandleFunc("POST /api/group/leave", s.handleLeaveGroup)
	s.mux.HandleFunc("GET /api/groups", s.handleListGroups)
	s.mux.HandleFunc("POST /api/agent/register", s.handleRegisterAgent)
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/status", s.handleStatus)

	// WebSocket for real-time
	s.mux.HandleFunc("GET /ws", s.handleWebSocket)

	// A2A protocol server
	a2aagent.SetupA2AServer(s.mux, s.hub, s.addr, s.log)
}

// ListenAndServe starts the relay server with graceful shutdown support.
func (s *Server) ListenAndServe() error {
	srv := &http.Server{Addr: s.addr, Handler: s.mux}

	// Graceful shutdown on SIGTERM/SIGINT
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		sig := <-sigCh
		s.log.Info("shutdown signal received", "signal", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			s.log.Error("shutdown error", "err", err)
		}
	}()

	s.log.Info("AgentChat relay starting", "addr", s.addr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}

	// Close PebbleDB store if present
	if s.store != nil {
		s.log.Info("closing PebbleDB store")
		s.store.Close()
	}
	s.log.Info("relay stopped cleanly")
	return nil
}

// --- REST Handlers ---

func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Group    string          `json:"group"`
		Sender   string          `json:"sender"`
		Body     string          `json:"body"`
		Priority types.Priority  `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Group == "" {
		req.Group = "general"
	}
	if req.Priority == "" {
		req.Priority = types.PriorityNormal
	}

	msg := s.hub.SendMessage(req.Group, req.Sender, req.Body, req.Priority)

	// Send desktop notification
	if req.Priority == types.PriorityStealFocus {
		notify.StealFocus(req.Sender, req.Body)
	} else {
		notify.ChatMessage(req.Sender, req.Body, "")
	}

	s.log.Info("message", "group", msg.Group, "sender", msg.Sender, "body", truncate(msg.Body, 60))
	writeJSON(w, msg)
}

func (s *Server) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	group := r.URL.Query().Get("group")
	if group == "" {
		group = "general"
	}
	since := r.URL.Query().Get("since")
	var sinceT time.Time
	if since != "" {
		var err error
		sinceT, err = time.Parse(time.RFC3339, since)
		if err != nil {
			sinceT = time.Now().Add(-24 * time.Hour)
		}
	} else {
		sinceT = time.Now().Add(-24 * time.Hour)
	}

	msgs := s.hub.GetMessages(group, sinceT, 100)
	writeJSON(w, msgs)
}

func (s *Server) handleJoinGroup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Agent string `json:"agent"`
		Group string `json:"group"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.hub.JoinGroup(req.Agent, req.Group)
	// Announce join
	s.hub.SendMessage(req.Group, "system", req.Agent+" joined the chat", types.PriorityNormal)
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleLeaveGroup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Agent string `json:"agent"`
		Group string `json:"group"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.hub.LeaveGroup(req.Agent, req.Group)
	s.hub.SendMessage(req.Group, "system", req.Agent+" left the chat", types.PriorityNormal)
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleListGroups(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.hub.GetGroups())
}

func (s *Server) handleRegisterAgent(w http.ResponseWriter, r *http.Request) {
	var profile types.AgentProfile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.hub.RegisterAgent(profile.Name, profile)
	writeJSON(w, map[string]string{"status": "registered"})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"status": "ok", "service": "agentchat-relay"})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	groups := s.hub.GetGroups()
	groupDetails := make([]map[string]interface{}, 0, len(groups))
	totalMessages := 0
	for _, g := range groups {
		msgs := s.hub.GetMessages(g.Name, time.Time{}, 10000)
		totalMessages += len(msgs)
		groupDetails = append(groupDetails, map[string]interface{}{
			"name":     g.Name,
			"members":  len(g.Members),
			"messages": len(msgs),
		})
	}
	writeJSON(w, map[string]interface{}{
		"service":       "agentchat-relay",
		"version":       "0.2.0",
		"uptime":        time.Since(s.started).String(),
		"groups":        groupDetails,
		"totalMessages": totalMessages,
	})
}

// --- WebSocket ---

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.Error("ws upgrade", "err", err)
		return
	}
	defer conn.Close()

	group := r.URL.Query().Get("group")
	if group == "" {
		group = "general"
	}

	ch := s.hub.Subscribe(group)
	defer s.hub.Unsubscribe(group, ch)

	s.log.Info("ws connected", "group", group)

	// Forward messages to WebSocket
	for msg := range ch {
		wsMsg := types.WSMessage{
			Type:    "message",
			Message: msg,
		}
		if err := conn.WriteJSON(wsMsg); err != nil {
			s.log.Debug("ws write error (client disconnected)", "err", err)
			return
		}
	}
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// GetHub returns the chat hub for external use (TUI, MCP).
func (s *Server) GetHub() *chat.Hub {
	return s.hub
}

// Broadcast sends a message from the system to all groups.
func (s *Server) Broadcast(body string) {
	for _, g := range s.hub.GetGroups() {
		s.hub.SendMessage(g.Name, "system", body, types.PriorityNormal)
	}
}

// ListGroups returns group names as strings.
func (s *Server) ListGroupNames() []string {
	groups := s.hub.GetGroups()
	names := make([]string, len(groups))
	for i, g := range groups {
		names[i] = g.Name
	}
	return names
}

// handleMessagesForTUI returns the message channel for TUI integration.
func (s *Server) SubscribeGroup(group string) <-chan *types.Message {
	return s.hub.Subscribe(group)
}

// SendMessageFromAPI allows external components to send messages.
func (s *Server) SendMessage(group, sender, body string, priority types.Priority) *types.Message {
	msg := s.hub.SendMessage(group, sender, body, priority)
	if priority == types.PriorityStealFocus {
		notify.StealFocus(sender, body)
	} else {
		notify.ChatMessage(sender, body, msg.Color)
	}
	return msg
}
