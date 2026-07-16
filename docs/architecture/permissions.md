# Permissions & tool approval

Mutating tools require operator consent unless `--yolo` / ACP `--always-approve` is set.

## Mutating tools (examples)

| Tool | Risk |
|------|------|
| `write_file` | Overwrites workspace files |
| `run_shell` | Executes shell (policy-checked) |
| `apply_worktree` / `apply_worktrees` | Merges subagent isolation worktrees into parent |
| `remove_worktree` | Deletes isolation worktrees |

Read-only tools (`read_file`, `list_dir`, `grep`, `diff_worktree`, `list_worktrees`, `spawn_subagent*`, `wait_subagents`) do not prompt.

## TUI / REPL

When a mutating tool is requested (full-screen overlay or classic REPL prompt):

```text
⚠ approve tool apply_worktree?
  {"id":"sa-…","remove":true}
[y]es / [n]o / [a]lways this session:
```

| Key | Effect |
|-----|--------|
| `y` | Allow once |
| `n` | Deny (tool result explains denial) |
| `a` | Allow this tool name for the rest of the session |

`/permissions` lists session always-allow tools. Full-screen layout: [tui.md](tui.md).

## Headless / ACP

Without an `Approver`, mutators are **denied** unless Yolo is enabled. That keeps CI and unattended runs fail-closed.

## Subagent dance

Typical isolated-edit flow with approval:

1. `spawn_subagent` / `spawn_subagents` with `isolation=worktree` (no prompt)
2. `diff_worktree` (no prompt)
3. `apply_worktree` / `apply_worktrees` → **approval prompt**
4. Optional `remove_worktree` → **approval prompt**
