---
name: hello-iome
description: Residual-honest operator welcome and mesh orientation playbook (sample plugin skill; not install APPLY · not Memory GA)
---

# Hello IOME (sample plugin skill)

Short **skills-only** playbook for operators dogfooding Agent Plugins package discover + opt-in `[plugins]` wire in iomesh-tui.

This skill is a **playbook only** — guidance text the agent may load when plugins are enabled. It does not install software, mint secrets, or auto-send outbound messages.

## Orientation

1. **What this package is** — a portable sample under `examples/agent-plugins/hello-iome` with a closed `plugin.json` and this skill. Skills-first; no MCP server ships in this sample.
2. **How it loads** — only when the operator **opts in** via TOML `[plugins] enabled = true` and a `dirs` entry pointing at this package root (or a parent of package roots). Default is **disabled**.
3. **Discover vs green** — `agentplugins.Discover` / map success means the package validated. It is **not** install Connected, marketplace install, or Agent Plugins product GA.

## Operator checklist (dogfood)

1. Point `[plugins].dirs` at this package (see package `README.md`).
2. Set `enabled = true` (opt-in only).
3. Restart / reload the TUI session so skills catalog merge can see `skills/hello-iome`.
4. Confirm the skill name appears in the skills catalog path your build uses — fail-open if the dir is missing.

## Explicit non-claims

| Do **not** claim | Truth |
|------------------|--------|
| Agent Plugins GA | Sample package + client candidacy only |
| Memory GA | Orthogonal surface; this skill is not memory product green |
| dual_write ON | dual_write remains **OFF** (unchanged default) |
| install APPLY / Connected green | Discover/load ≠ install success |
| Auto-send outbound | Skills are playbooks only; no auto-send |
| Secrets in plugin.json | Portable package fields must not carry secrets |
| MCP tools always available | This sample is **skills-only**; if MCP appears later, tools stay **approval-gated** (mutating default true) |

## Related

- Package README: `examples/agent-plugins/hello-iome/README.md`
- Architecture: `docs/architecture/agent-plugins.md` (s1326 discover · s1331 opt-in wire · s1337 sample)
- Spec peer: [Agent Plugins 1.0.0](https://agent-plugins.org/specification)
