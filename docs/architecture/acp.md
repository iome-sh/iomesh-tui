# ACP (Agent Client Protocol)

`iomesh agent stdio` speaks **newline-delimited JSON-RPC 2.0** on stdin/stdout for IDE and automation clients. **Logs go to stderr only.**

## Run

```bash
iomesh agent stdio
iomesh agent -m deepseek-v4-flash --yolo -C /path/to/repo stdio
```

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

## Wire format example

```json
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}
{"jsonrpc":"2.0","id":2,"method":"session/new","params":{"cwd":"/repo"}}
{"jsonrpc":"2.0","id":3,"method":"session/prompt","params":{"sessionId":"acp-1","prompt":[{"type":"text","text":"list files"}]}}
```
