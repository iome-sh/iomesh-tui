# Sessions

Parent agent state (transcript + **subagent registry**) persists under:

```text
<workspace>/.iomesh/sessions/<id>.json
```

## CLI

```bash
iomesh -p "start work"          # creates/saves a session
iomesh -c                       # continue latest session
iomesh --session ses-…          # resume by id
iomesh sessions                 # list (msgs, subagent count, title)
iomesh --no-save …              # disable autosave
```

## REPL

| Command | Action |
|---------|--------|
| `/save` | Persist now |
| `/save compact` | Compact (keep last 8 user turns) then save |
| `/sessions` | List saved sessions |
| `/load <id>` | Restore transcript + subagent catalog |

## Subagent dance

On save, running/pending children are marked **cancelled** (metadata kept for `resume_from` / `apply_worktree`). On load, the registry is restored so the parent can still `diff_worktree` / `apply_worktree` against kept worktrees and `resume_from` completed explore/plan runs.

## Security

- Session files mode `0600`
- IDs path-jailed under `.iomesh/sessions`
- Tool payloads truncated on compact (`MaxStoredToolContent`)
