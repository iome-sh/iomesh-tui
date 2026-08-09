# Setup lifecycle (agent-native wizard foundation)

**Serial:** free eng **s1525** P1–P2 · residual-honest  
**Status:** foundation shipped — config write + `iomesh setup init|preflight`  
**Not yet:** in-session continuous pull, hot MCP reattach, analyze ticks (later PRs)

## Goal

Enable the map story without inventing Connected / Memory GA:

```text
setup init → write managed config → start memory host → preflight probe
  → restart TUI → optional portal HITL integrations
  → iomesh memory pull (continuous) → /memory analyze
```

## CLI

```bash
# Local memory plane (default profile)
iomesh setup init local-memory --config ~/.iomesh/config.toml

# Multiple planes
iomesh setup init --profiles local-memory,plugins,mesh --plugins-dir /path/to/examples/agent-plugins

# Print fragment only
iomesh setup init all --print-only

# Preflight (probe · not invent green)
iomesh setup preflight --json
```

### Profiles

| Profile | Writes |
|---------|--------|
| `local-memory` | `[mcp]` + memory server URL/stdio + `[memory]` dual_write=false |
| `plugins` | `[plugins] enabled` + dirs |
| `mesh` | `[iomesh]` endpoint/tenant placeholders + `api_key_env` |
| `platform-mcp` | platform `[[mcp.servers]]` + `oauth_token_env` |
| `all` | all of the above |

Managed block markers:

```text
# BEGIN iomesh-setup-managed
…
# END iomesh-setup-managed
```

User edits **outside** the block are preserved on re-init.

## Honesty locks

| Lock | Rule |
|------|------|
| dual_write OFF | Managed fragment refuses `dual_write = true` |
| not Memory GA | Preflight never stamps Memory GA |
| catalog ≠ Connected | Setup PASS ≠ invent install green |
| secrets | Env **names** only (`api_key_env`, `oauth_token_env`) |
| portal HITL | OAuth/install still browser |
| continuous pull | Still CLI `iomesh memory pull` until in-session PR |

## Preflight states

| State | Meaning |
|-------|---------|
| `not_started` | No config file |
| `config_present` / `config_written` | File exists |
| `awaiting_memory_host` | Memory configured but healthz/PATH fail |
| `local_memory_probe_ok` | Memory host healthz OK (or stdio binary on PATH) |

## Next phases (plan)

- `/setup` slash + `setup-lifecycle-agent` skill  
- `ReplaceMCP` / reload after apply  
- In-session continuous pull + analyze ticks  
- Maintenance drift repair  

See product plan: agent-native MCP/plugin setup wizard + continuous pull/analyze.

## Related

- [memory-advanced-install.md](./memory-advanced-install.md)  
- [memory-edge-usage-demo.md](./memory-edge-usage-demo.md)  
- [agent-plugins.md](./agent-plugins.md)  
