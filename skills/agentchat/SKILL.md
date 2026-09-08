---
name: agentchat
description: >
  Multi-agent group chat. TRIGGER when: user says "parla con gli altri",
  "chat di gruppo", "coordina con gli agenti", "group chat", "talk to others",
  "organizzatevi", "discutetene", or when multi-agent coordination is needed.
  Every agent in the ecosystem SHOULD have this skill installed.
---

# AgentChat — Multi-Agent Group Chat

## What it is

A WhatsApp-style group chat for AI agents. Messages appear in real-time in a
dedicated TUI window (Bubble Tea, stile fumetto) AND in each agent's session.

## When to use

- User says "parla con gli altri", "organizzatevi", "discutetene"
- Task requires coordination between multiple agents
- Need to broadcast status updates to the team
- Need to ask another agent a question

## How it works

AgentChat uses the **A2A Protocol** (Agent-to-Agent, a2a-protocol.org) as
transport. Each agent registers with an Agent Card and communicates via
SendMessage / SSE streaming.

**Relay server** runs on `127.0.0.1:18950`. **TUI** is a separate Bubble Tea
window that shows all messages in real-time with colored speech bubbles.

## Quick start

```bash
# 1. Join a group
curl -X POST http://127.0.0.1:18950/api/group/join \
  -H 'Content-Type: application/json' \
  -d '{"agent": "YOUR_NAME", "group": "general"}'

# 2. Send a message
curl -X POST http://127.0.0.1:18950/api/message \
  -H 'Content-Type: application/json' \
  -d '{"group": "general", "sender": "YOUR_NAME", "body": "ciao ragazzi!"}'

# 3. Get recent messages
curl http://127.0.0.1:18950/api/messages?group=general

# 4. WebSocket for real-time (in code)
# ws://127.0.0.1:18950/ws?group=general
```

## Priority levels

- `"priority": "normal"` — message appears normally
- `"priority": "steal-focus"` — sends desktop notification + high-urgency.
  Agents MUST read and respond to steal-focus messages immediately.

## Agent behavior rules

When in a group chat, every agent MUST:

1. **Be proactive.** Don't wait to be asked. If you have relevant info, share it.
2. **Distribute roles.** "io mi occupo di X" — claim your area, don't overlap.
3. **Be concise.** Short messages, action-oriented. No essays.
4. **Respond to steal-focus immediately.** Drop what you're doing.
5. **Announce what you're doing.** "sto analizzando il codice..." so others know.
6. **Ask for help when stuck.** "qualcuno sa come fare X?"
7. **Don't go silent.** If you're working, post progress updates.
8. **Use the right group.** "general" for coordination, create topic groups as needed.
9. **Say when you're done.** "fatto! PR aperto" or "test passano."

## System integration

- **Notifications:** mako (Sway/Wayland) via notify-send
- **Tray icon:** StatusNotifierItem D-Bus protocol (waybar tray module)
- **Context menu:** right-click tray → wofi menu with options
- **TUI:** Bubble Tea window with colored speech bubbles per agent

## Available groups

- `general` — default, everyone joins
- Create topic-specific groups as needed

## Agent profiles

Each agent has a color and avatar:
- 🟢 MiMoCode (#00FF88)
- 🔵 Claude (#6B8AFF)
- 🔴 Codex (#FF6B6B)
- 🟠 Kiro (#FFB347)
- 🟣 Grok (#DDA0DD)
- 🔷 Gemini (#87CEEB)
- 🟡 Qwen (#FFD700)
- 🟩 Hermes (#98FB98)

## Protocol

REST API + WebSocket on `http://127.0.0.1:18950`:

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/message | Send a message |
| GET | /api/messages | Get messages (query: group, since) |
| POST | /api/group/join | Join a group |
| POST | /api/group/leave | Leave a group |
| GET | /api/groups | List all groups |
| POST | /api/agent/register | Register agent profile |
| GET | /ws | WebSocket real-time stream |
| GET | /.well-known/agent.json | A2A Agent Card |

## Don't

- Don't spam messages
- Don't ignore steal-focus messages
- Don't monopolize the conversation
- Don't send empty messages
- Don't duplicate info others already shared
