# Subagents

Child sessions with independent context windows, matching Grok Build’s explore / plan / general-purpose roles.

## Built-in types

| Type | Tools | Purpose |
|------|--------|---------|
| `explore` | read, list, grep, shell | Codebase investigation |
| `plan` | read, list, grep, shell | Structured implementation plan |
| `general-purpose` | full toolset | Delegated implement/research |

Nested `spawn_subagent` is **disabled** on children (`max_depth` still enforced at manager).

## Capability modes

| Mode | Read | Write | Shell |
|------|------|-------|-------|
| `read-only` | ✓ | | |
| `read-write` | ✓ | ✓ | |
| `execute` | ✓ | | ✓ |
| `all` | ✓ | ✓ | ✓ |

Type defaults: explore/plan → `execute`; general-purpose → `all`.

## Parent tools

| Tool | Purpose |
|------|---------|
| **`spawn_subagent`** | One child: `prompt`, `description`, `subagent_type`, `background`, `capability_mode`, `isolation`, `resume_from`, `cwd` |
| **`spawn_subagents`** | **Maximum parallel fan-out**: `tasks[]` + optional `wait`, `default_subagent_type` |
| **`get_subagent_output`** | One id; optional `wait=true` |
| **`wait_subagents`** | Join many ids from a non-waiting batch |

### Parallel fan-out (preferred)

```json
{
  "tasks": [
    {"prompt": "Map cmd/ package entrypoints", "description": "scan cmd"},
    {"prompt": "Map internal/router cascade", "description": "scan router"},
    {"prompt": "List security controls", "description": "scan security"}
  ],
  "default_subagent_type": "explore",
  "wait": true
}
```

- Tasks start as **background** so the semaphore fills up to `max_concurrent`.
- `wait=true` joins all; `wait=false` returns ids for later `wait_subagents`.
- Batch size capped by `max_batch` (default **32**).

## Cost routing

Children force `TaskSubagent` + `ComplexityRoutine` so the cascade prefers **DeepSeek V4 Flash**.

## Config

```toml
[subagents]
enabled = true
max_concurrent = 16   # parallel running children
max_batch = 32        # max tasks per spawn_subagents
max_depth = 2
```

Env: `IOMESH_SUBAGENTS=0` disables.

## Isolation

- `none` (default) — shared workspace root  
- `worktree` — reserved; returns clear error until git worktree backend lands  

## REPL

`/subagents` lists ids, status, type, description for the session.
