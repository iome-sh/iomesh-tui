---
name: aion-memory-local
description: Residual-honest local-primary memory edge playbook (sample plugin skill; not Memory GA · dual_write OFF · package load ≠ Connected)
---

# Aion memory local edge (sample plugin skill)

Short playbook for operators dogfooding the **aion-memory-mcp** Agent Plugins sample package — stdio **map only** against the local-primary Memory MCP binary.

This skill is a **playbook only** — guidance text the agent may load when plugins are enabled. It does not install software, mint secrets, auto dual-write, or invent Memory product green.

## Orientation

1. **What this package is** — portable sample under `examples/agent-plugins/aion-memory-mcp` with closed `plugin.json`, `mcp.json` (stdio `memory` → `aion-memory-mcp`), and this skill. Map-only; binary not shipped.
2. **How it loads** — only when the operator **opts in** via TOML `[plugins] enabled = true` and a `dirs` entry pointing at this package root (or a parent of package roots). Default is **disabled**.
3. **Discover / map vs Connected** — `agentplugins.Discover` and `MCPServersFromPlugins` success means the package validated and mapped. It is **not** process Connected, install APPLY green, marketplace install, Agent Plugins product GA, or Memory GA.
4. **Connect requirement** — the operator must install `aion-memory-mcp` and put it on **PATH**. Attach is **fail-open** if the binary is missing (no invent green).

## Operator checklist (dogfood)

1. Install `aion-memory-mcp` and confirm `which aion-memory-mcp` (or equivalent) succeeds in the TUI environment.
2. Point `[plugins].dirs` at this package (see package `README.md`).
3. Set `enabled = true` (opt-in only).
4. Restart / reload the TUI session so skills catalog merge and plugin MCP map can see this package.
5. Prefer dual_write **OFF** (default). Local-primary palace; not freemium hosted palace.
6. Mutating tools (including `memory_trigger_compact` / `/memory trigger-compact`) remain **HITL / approval-gated** — refuse without operator confirm.

## Explicit non-claims

| Do **not** claim | Truth |
|------------------|--------|
| Agent Plugins GA | Sample package + client candidacy only |
| Memory GA | Local-primary edge residual · not product Memory green |
| dual_write ON | dual_write remains **OFF** (unchanged default) |
| freemium hosted palace | Hosted Palace sunset until scale · local-primary only |
| install APPLY / Connected green | Discover/load/map ≠ install or process Connected |
| Secrets in plugin.json / mcp.json | Portable package fields must not carry secrets |
| Map success = tools always available | Connect needs binary on PATH; MCP tools stay **approval-gated** (mutating default true) |
| Auto-send / auto compact green | Skills are playbooks only; `trigger_compact` is HITL if used |

## Related

- Package README: `examples/agent-plugins/aion-memory-mcp/README.md`
- Architecture: `docs/architecture/agent-plugins.md` (s1326 discover · s1331 opt-in wire · s1346 sample)
- Memory MCP: `docs/architecture/memory-mcp.md` (local-primary · dual_write OFF · not Memory GA)
- Spec peer: [Agent Plugins 1.0.0](https://agent-plugins.org/specification)
