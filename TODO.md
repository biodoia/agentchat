# TODO — AgentChat

Tracking dei lavori da fare. Aggiornato ad ogni loop.

## LEGGENDA

- `[ ]` = da fare
- `[~]` = in corso
- `[x]` = fatto
- `[#]` = cancellato (non necessario)

---

## CRITICA (blocca l'uso in produzione)

- [ ] **TUI: join group + load history on startup**
  Il TUI si connette al WebSocket ma non fa POST /api/group/join e non
  carica la cronologia. Senza questo, i messaggi precedenti sono persi.
  Sforzo: 30 min

- [ ] **Graceful shutdown nel relay**
  ListenAndServe() non gestisce SIGTERM/SIGINT. Ctrl+C non chiude
  PebbleDB pulitamente. Rischio corruzione dati.
  Sforzo: 15 min

- [ ] **Installare skill negli agenti**
  Copiare skills/agentchat/SKILL.md in .mimocode/skills/agentchat/
  e .agents/skills/agentchat/ per ogni agente del fleet.
  Sforzo: 5 min

---

## IMPORTANTE (migliora significativamente l'UX)

- [ ] **Vero MCP server (JSON-RPC tools)**
  cmd/mcp e' un CLI wrapper. Dovrebbe esporre tool MCP nativi:
  chat_send, chat_poll, chat_join, chat_groups. Un agente MCP potrebbe
  usarlo senza shell.
  Sforzo: 2 ore

- [ ] **Auto-discovery via env**
  Gli agenti devono conoscere l'URL del relay hardcoded. Servono:
  - Variabile AGENTCHAT_RELAY (default: http://127.0.0.1:18950)
  - Auto-detection del programma agent (Claude, Codex, ecc.)
  Sforzo: 15 min

- [ ] **Retry/reconnect nel TUI**
  Se la connessione WebSocket cade, il TUI mostra errore e basta.
  Dovrebbe riconnettersi automaticamente con backoff.
  Sforzo: 30 min

- [ ] **Tray menu funzionanti**
  Le funzioni showInputDialog e showInfo sono stub. "New Group" non
  apre un dialog, "List Agents" non mostra nulla.
  Sforzo: 30 min

- [ ] **Sender auto-detection nel TUI**
  Invece di AGENTCHAT_SENDER, rilevare il processo parent per
  determinare se siamo Claude, Codex, Kiro, ecc.
  Sforzo: 20 min

---

## NICE-TO-HAVE (migliorano ma non bloccano)

- [ ] **Integration test e2e**
  Start relay → send message → verify persistence → restart →
  verify load from PebbleDB.
  Sforzo: 1 ora

- [ ] **Rate limiting**
  Un agente impazzito potrebbe spammarne migliaia di messaggi.
  Max 10 messaggi/secondo per agent.
  Sforzo: 30 min

- [ ] **Message edit/delete**
  Una volta inviato, un messaggio e' permanente. Servirebbe soft-delete.
  Sforzo: 1 ora

- [ ] **Typing indicator**
  Vedere quando un agente sta scrivendo (WebSocket event "typing").
  Sforzo: 30 min

- [ ] **File/allegati supportati**
  Solo testo al momento. Servirebbe supporto per immagini e file.
  Sforzo: 2 ore

- [ ] **Message threading**
  Rispondere a un messaggio specifico (reply-to).
  Sforzo: 1 ora

- [ ] **Search messaggi**
  Cercare nella cronologia per keyword.
  Sforzo: 30 min

- [ ] **Web dashboard (HTMX)**
  Visualizzare gruppi e messaggi nel browser.
  Sforzo: 3 ore

- [ ] **Makefile**
  Build automatizzato con make build, make test, make install.
  Sforzo: 10 min

---

## COMPLETATO

- [x] **Core relay server** — HTTP + WebSocket + A2A (loop 1)
- [x] **Chat hub** — groups, messages, subscribe (loop 1)
- [x] **PebbleDB persistence** — save on send, load on start (loop 2)
- [x] **SNI D-Bus tray icon** — waybar + wofi menu (loop 3)
- [x] **HTTP handler tests** — 12 test (loop 4)
- [x] **Systemd services** — relay + tui + tray (loop 5)
- [x] **A2A executor tests** — 8 test (loop 6)
- [x] **Notify tests + health dashboard** — /api/status (loop 7)
- [x] **Types tests + README** — 4 test + docs (loop 8)
- [x] **Retro README + AGENT_GUIDE + SKILL + TODO** — (loop 9)

---

## STATISTICHE

| Metrica | Valore |
|---------|--------|
| LOC | ~2600 |
| Test | 55 PASS, 0 FAIL |
| Coverage | chat 90%, store 84%, a2aagent 76%, relay 53% |
| Binari | 4 (relay, tui, mcp, tray) |
| Commit | 13 |
| Packages con test | 7/11 |
| Repo | github.com/biodoia/agentchat |
