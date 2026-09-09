// agentchat-tui is the WhatsApp-style group chat viewer for AgentChat.
// Built with Bubble Tea for Sway/Wayland.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gorilla/websocket"
)

// Styles
var (
	systemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true).
			Padding(0, 2)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("63")).
			Padding(0, 3).
			MarginBottom(1)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Padding(0, 1)
)

type chatMsg struct {
	Sender   string
	Body     string
	Color    string
	Avatar   string
	TS       time.Time
	Priority string
}

type model struct {
	messages []chatMsg
	input    string
	group    string
	relayURL string
	sender   string
	wsConn   *websocket.Conn
	err      error
	width    int
	height   int
	ready    bool
}

func initialModel(relayURL, group, sender string) model {
	return model{
		relayURL: relayURL,
		group:    group,
		sender:   sender,
		width:    80,
		height:   24,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		joinGroup(m.relayURL, m.sender, m.group),
		loadHistory(m.relayURL, m.group),
		tea.EnterAltScreen,
	)
}

// --- Message types for Bubble Tea ---

type wsMessage chatMsg
type wsConnectedMsg struct{ conn *websocket.Conn }
type wsErrorMsg struct{ err error }
type sentMsg struct{}
type historyLoadedMsg struct{ msgs []chatMsg }
type joinedGroupMsg struct{}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			if m.wsConn != nil {
				m.wsConn.Close()
			}
			return m, tea.Quit
		case tea.KeyEnter:
			if m.input != "" {
				cmd := sendMessage(m.relayURL, m.group, m.sender, m.input)
				m.input = ""
				return m, cmd
			}
		case tea.KeyBackspace:
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
			m.input += msg.String()
		}

	case historyLoadedMsg:
		m.messages = append(msg.msgs, m.messages...)
		// After history loads, connect WebSocket
		return m, connectWS(m.relayURL, m.group)

	case joinedGroupMsg:
		// Group joined, history will load next
		return m, nil

	case wsMessage:
		m.messages = append(m.messages, chatMsg(msg))
		return m, listenWS(m.wsConn)

	case wsConnectedMsg:
		m.wsConn = msg.conn
		return m, listenWS(m.wsConn)

	case wsErrorMsg:
		m.err = msg.err
		return m, nil

	case sentMsg:
		return m, nil
	}

	return m, nil
}

func (m model) View() string {
	if !m.ready {
		return "Connecting to AgentChat..."
	}

	header := titleStyle.Render(fmt.Sprintf(" AgentChat — #%s", m.group))

	var msgLines []string
	maxBodyWidth := m.width - 12

	for _, msg := range m.messages {
		if msg.Sender == "system" {
			msgLines = append(msgLines, systemStyle.Width(m.width).Render("── "+msg.Body+" ──"))
			continue
		}

		senderLine := fmt.Sprintf("%s %s", msg.Avatar, msg.Sender)
		senderStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(msg.Color))
		bubbleContent := senderStyle.Render(senderLine) + "\n" + wrapText(msg.Body, maxBodyWidth-4)

		bubble := lipgloss.NewStyle().
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(msg.Color)).
			Width(maxBodyWidth).
			Render(bubbleContent)

		msgLines = append(msgLines, bubble)
	}

	msgAreaHeight := m.height - 6
	if len(msgLines) > msgAreaHeight {
		msgLines = msgLines[len(msgLines)-msgAreaHeight:]
	}
	messagesView := strings.Join(msgLines, "\n")

	inputBar := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(lipgloss.Color("63")).
		Width(m.width).
		Render(fmt.Sprintf("▶ %s_", m.input))

	status := statusStyle.Render(fmt.Sprintf("  %s | %d messages | Ctrl+C to quit", m.relayURL, len(m.messages)))

	return fmt.Sprintf("%s\n%s\n%s\n%s", header, messagesView, inputBar, status)
}

// --- WebSocket commands ---

func connectWS(relayURL, group string) tea.Cmd {
	return func() tea.Msg {
		u := url.URL{
			Scheme:   "ws",
			Host:     strings.TrimPrefix(strings.TrimPrefix(relayURL, "http://"), "https://"),
			Path:     "/ws",
			RawQuery: "group=" + group,
		}
		conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			return wsErrorMsg{err}
		}
		return wsConnectedMsg{conn}
	}
}

func listenWS(conn *websocket.Conn) tea.Cmd {
	if conn == nil {
		return nil
	}
	return func() tea.Msg {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return wsErrorMsg{err}
		}
		var ws struct {
			Type    string `json:"type"`
			Message struct {
				Sender   string `json:"sender"`
				Body     string `json:"body"`
				Color    string `json:"color"`
				Avatar   string `json:"avatar"`
				TS       string `json:"ts"`
				Priority string `json:"priority"`
			} `json:"message"`
		}
		json.Unmarshal(data, &ws)
		ts, _ := time.Parse(time.RFC3339, ws.Message.TS)
		return wsMessage{
			Sender:   ws.Message.Sender,
			Body:     ws.Message.Body,
			Color:    ws.Message.Color,
			Avatar:   ws.Message.Avatar,
			TS:       ts,
			Priority: ws.Message.Priority,
		}
	}
}

func sendMessage(relayURL, group, sender, body string) tea.Cmd {
	return func() tea.Msg {
		payload := map[string]string{
			"group":  group,
			"sender": sender,
			"body":   body,
		}
		data, _ := json.Marshal(payload)
		resp, err := http.Post(relayURL+"/api/message", "application/json", bytes.NewReader(data))
		if err != nil {
			return wsErrorMsg{err}
		}
		defer resp.Body.Close()
		io.ReadAll(resp.Body)
		return sentMsg{}
	}
}

func joinGroup(relayURL, agent, group string) tea.Cmd {
	return func() tea.Msg {
		payload := map[string]string{"agent": agent, "group": group}
		data, _ := json.Marshal(payload)
		http.Post(relayURL+"/api/group/join", "application/json", bytes.NewReader(data))
		return joinedGroupMsg{}
	}
}

func loadHistory(relayURL, group string) tea.Cmd {
	return func() tea.Msg {
		resp, err := http.Get(fmt.Sprintf("%s/api/messages?group=%s&limit=50", relayURL, group))
		if err != nil {
			return historyLoadedMsg{}
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

		var msgs []struct {
			Sender   string `json:"sender"`
			Body     string `json:"body"`
			Color    string `json:"color"`
			Avatar   string `json:"avatar"`
			TS       string `json:"ts"`
			Priority string `json:"priority"`
		}
		json.Unmarshal(body, &msgs)

		result := make([]chatMsg, 0, len(msgs))
		for _, m := range msgs {
			ts, _ := time.Parse(time.RFC3339, m.TS)
			result = append(result, chatMsg{
				Sender: m.Sender, Body: m.Body, Color: m.Color,
				Avatar: m.Avatar, TS: ts, Priority: m.Priority,
			})
		}
		return historyLoadedMsg{msgs: result}
	}
}

func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}
	var lines []string
	for len(text) > width {
		cut := width
		for cut > 0 && text[cut] != ' ' {
			cut--
		}
		if cut == 0 {
			cut = width
		}
		lines = append(lines, text[:cut])
		text = strings.TrimLeft(text[cut:], " ")
	}
	lines = append(lines, text)
	return strings.Join(lines, "\n")
}

func main() {
	relayURL := "http://127.0.0.1:18950"
	group := "general"
	sender := os.Getenv("AGENTCHAT_SENDER")
	if sender == "" {
		sender = "MiMoCode"
	}

	if len(os.Args) > 1 {
		relayURL = os.Args[1]
	}
	if len(os.Args) > 2 {
		group = os.Args[2]
	}
	if len(os.Args) > 3 {
		sender = os.Args[3]
	}

	// Register agent on connect
	go func() {
		profile := map[string]string{"name": sender}
		data, _ := json.Marshal(profile)
		http.Post(relayURL+"/api/agent/register", "application/json", bytes.NewReader(data))
	}()

	p := tea.NewProgram(initialModel(relayURL, group, sender), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
