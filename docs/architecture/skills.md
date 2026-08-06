# Skills loader

Skills are markdown playbooks (`SKILL.md`) the agent can discover and read via tools.

## Discovery

When `[skills] enabled = true` (default):

| Path | Role |
|------|------|
| **Builtin** (`go:embed` `internal/skills/builtin/`) | Shipped skills (always merged) |
| `<workspace>/.iomesh/skills/<name>/SKILL.md` | Project skills |
| `~/.iomesh/skills/<name>/SKILL.md` | User skills |
| `[skills].dirs` | Extra roots |

Load uses `skills.LoadWithBuiltin(dirs...)`: **builtin first**, then workspace/user dirs (user overrides builtin on name collision). Missing directories are skipped. Builtin skills appear even when all dirs are empty.

### Builtin: `connector-integrations-setup` (s1251)

Residual-honest agent path for connector integrations via MCP (`list_connector_catalog` → `plan_connector_setup` → optional `get_webhook_signing_headers` → **browser portal HITL**). Explicit non-goals: no invent install green / Connected / INSTALL_STORE APPLY / GA · stub ≠ live · dual_write OFF · book-demo OFF · agent MCP cannot write installs.

Paired with the `<integrations>` system note injected on `AttachMCP` (`IntegrationsAgentGuidanceNote`).

### Builtin: `memory-advanced-agent` (s1288)

Residual-honest agent path for **advanced memory** surfaces already on main: `/memory related` + `prefer_shorter_hops` (s1281) · `/memory supersede --i-confirm` HITL (s1282) · `/memory facts-as-of` (s1276) · `/memory digest` (s1200) · peer s1287 `/memory patterns|anomalies`. MCP inventory: `memory_related`, `memory_supersede_entity`, `memory_facts_as_of`, `ops_digest_export`, `memory_patterns_list`, `memory_anomalies_list`. Honesty locks: multi-hop lite ≠ graph RAG · PreferShorterHops default true · A3 lite ≠ NLP · supersede HITL · K4 lite ≠ dual-clock · patterns/anomalies not medical · dual_write OFF · not Memory GA · no invent lean HTTP for supersede/facts-as-of/patterns. Skill-only (no extra system-note inject). See [memory-mcp.md](./memory-mcp.md).

## Format

```markdown
---
name: review
description: Structured code review checklist
---

# Review skill

1. Diff against main
2. …
```

YAML frontmatter needs only `name` and `description` (minimal parser; no full YAML stack).

## Agent tools

| Tool | Mutating | Purpose |
|------|----------|---------|
| `list_skills` | no | Catalog names + descriptions |
| `read_skill` | no | Full skill body |

A compact catalog is also injected into the system prompt as `<skills>…</skills>`.

## CLI

```bash
iomesh skills
iomesh skills -C /path/to/workspace
```
