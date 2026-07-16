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

| Mode | Behavior |
|------|----------|
| `none` (default) | Shared workspace root with the parent |
| `worktree` | Detached **`git worktree`** under `<workspace>/.iomesh/worktrees/<id>` |

Requirements for `worktree`:

- Parent path must be a git work tree with at least one commit  
- `git` on `PATH`  
- Safe id characters only (enforced)  

Successful runs **keep** the worktree by default (`worktree_path` in the result) so the parent can inspect or merge. Set `worktree_auto_remove = true` to delete after success. Failures always remove the worktree.

```bash
# inspect
ls .iomesh/worktrees/
# remove when done
git worktree remove --force .iomesh/worktrees/<id>
```

`spawn_subagents` accepts per-task `isolation` or batch `default_isolation`.

### Apply / merge dance

After a child finishes in a worktree, the **parent** merges changes:

| Tool | Mutating | Purpose |
|------|----------|---------|
| `diff_worktree` | no | `git status` + diff stat for id/path |
| `list_worktrees` | no | List `.iomesh/worktrees/*` |
| `apply_worktree` | **yes** | Path-jailed copy of changed files into parent; optional `remove=true` |
| `remove_worktree` | **yes** | Drop worktree without applying |

```text
spawn_subagent(isolation=worktree, subagent_type=general-purpose)
  → { id, worktree_path, summary }
diff_worktree(id)
apply_worktree(id, remove=true)   # requires --yolo / approval
```

Symlinks and directory-only entries are skipped; file creates/updates/deletes under the jail are applied.

## REPL

`/subagents` lists ids, status, type, description for the session.
