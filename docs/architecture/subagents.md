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

- **`spawn_subagent`** — `prompt` (required), `description`, `subagent_type`, `background`, `capability_mode`, `isolation`, `resume_from`, `cwd`
- **`get_subagent_output`** — `id`, optional `wait=true`

## Cost routing

Children force `TaskSubagent` + `ComplexityRoutine` so the cascade prefers **DeepSeek V4 Flash**.

## Config

```toml
[subagents]
enabled = true
max_concurrent = 4
max_depth = 2
```

Env: `IOMESH_SUBAGENTS=0` disables.

## Isolation

- `none` (default) — shared workspace root  
- `worktree` — reserved; returns clear error until git worktree backend lands  

## REPL

`/subagents` lists ids, status, type, description for the session.
