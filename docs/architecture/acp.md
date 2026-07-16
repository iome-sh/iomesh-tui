# ACP (Agent Client Protocol)

`iomesh agent` speaks **JSON-RPC 2.0** for IDE and automation clients.

| Mode | Command | Transport |
|------|---------|-----------|
| **stdio** | `iomesh agent stdio` | NDJSON on stdin/stdout (logs → stderr) |
| **WebSocket** | `iomesh agent serve` | Text frames on `ws://127.0.0.1:7400/acp` |

Protocol methods and `session/update` shapes are identical in both modes.

## Run

```bash
# Stdio (pipes / IDE process spawn)
iomesh agent stdio
iomesh agent -m deepseek-v4-flash --yolo -C /path/to/repo stdio

# WebSocket (default loopback)
iomesh agent serve
iomesh agent serve --listen 127.0.0.1:7400 --path /acp
iomesh agent serve --token "$SECRET" --listen 0.0.0.0:7400   # token required off-loopback
```

Health endpoints (serve mode):

- `GET /healthz` → `ok`
- `GET /readyz` → `ready`

## Lifecycle

1. `initialize` → server capabilities + `serverInfo`
2. `session/new` `{ "cwd": "..." }` → `{ "sessionId": "acp-N" }`
3. `session/prompt` `{ "sessionId", "prompt": [{ "type":"text", "text":"..." }] }`
4. Server emits `session/update` notifications while the agent runs
5. Prompt result `{ "stopReason": "end_turn" }`

Also supported:

- `session/load` — load a persisted `.iomesh/sessions/<id>.json` into a new ACP session
- `session/cancel`, `shutdown`, `exit`

## Streaming updates (`session/update`)

| `sessionUpdate` | Meaning |
|-----------------|---------|
| `agent_message_chunk` | Assistant text delta |
| `agent_thought_chunk` | Thinking / mesh notes |
| `tool_call` | Tool started (includes **subagent** tools) |
| `tool_call_update` | Tool finished / failed |
| `model_selected` | Cascade pick (iomesh extension) |
| `usage` | Tokens / cost / duration |

Subagent orchestration tools (`spawn_subagent`, `spawn_subagents`, `wait_subagents`, `apply_worktree`, …) appear as tool calls so IDE clients can show the subagent dance live.

## Permissions

Mutating tools respect agent approval rules. Pass `--yolo` / `--always-approve` for unattended apply/shell.

## WebSocket security

- **Default bind** is `127.0.0.1:7400` (loopback only).
- **`--token`**: require `Authorization: Bearer <token>` or `?token=` on upgrade.
- Binding non-loopback without `--token` prints a warning.
- Each WebSocket connection gets an **isolated** ACP session map (not shared across clients).

## Wire format example (stdio NDJSON / WS text frames)

```json
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}
{"jsonrpc":"2.0","id":2,"method":"session/new","params":{"cwd":"/repo"}}
{"jsonrpc":"2.0","id":3,"method":"session/prompt","params":{"sessionId":"acp-1","prompt":[{"type":"text","text":"list files"}]}}
```

## Package

- `internal/acp/server.go` — JSON-RPC handlers + stdio `Run`
- `internal/acp/ws.go` — `ListenAndServe` + `RunWebSocket`
