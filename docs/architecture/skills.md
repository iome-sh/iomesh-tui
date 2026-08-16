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

**Agent Plugins package skills (s1326 + s1331 + s1670):** opt-in `[plugins] enabled = true` + `dirs` → `runtimewire.Wire` merges `agentplugins.SkillDirs` into the `skills.LoadWithBuiltin` path (fail-open DiscoverAll). Package wire ≠ Agent Plugins GA · Discover ≠ Connected / install APPLY green. **s1670:** `/setup reload` re-scans skills (including plugin skill dirs when plugins enabled) via `LoadWithBuiltin` + `Runtime.ReplaceSkills` — process restart is no longer required for skill-only path changes after reload. dual_write OFF · not Memory GA. See [agent-plugins.md](./agent-plugins.md) · [setup-lifecycle.md](./setup-lifecycle.md).

### Builtin: `connector-integrations-setup` (s1251)

Residual-honest agent path for connector integrations via MCP (`list_connector_catalog` → `plan_connector_setup` → optional `get_webhook_signing_headers` → **browser portal HITL**). Explicit non-goals: no invent install green / Connected / INSTALL_STORE APPLY / GA · stub ≠ live · dual_write OFF · book-demo OFF · agent MCP cannot write installs.

Paired with the `<integrations>` system note injected on `AttachMCP` (`IntegrationsAgentGuidanceNote`).

### Builtin: `memory-advanced-agent` (s1288)

Residual-honest agent path for **advanced memory** surfaces already on main: `/memory write` durable fact (s2006 · `memory_write`) · `/memory related` + `prefer_shorter_hops` (s1281) · `/memory supersede --i-confirm` HITL (s1282) · `/memory facts-as-of` (s1276) · `/memory digest` (s1200) · peer s1287 `/memory patterns|anomalies` · s1296 timeline+compact-status · s1301 semantic+ingest-event · **s1311** `/memory trigger-compact --i-confirm` HITL + `/memory status` advanced inventory. MCP inventory: `memory_write`, `memory_related`, `memory_supersede_entity`, `memory_facts_as_of`, `ops_digest_export`, `memory_patterns_list`, `memory_anomalies_list`, `memory_timeline`, `memory_compact_status`, `memory_search_semantic`, `memory_ingest_event`, `memory_trigger_compact`. Lean host recopy (s2006) is **not a new live tools=N** stamp — historical s1508/s1509 tools=6 stays past. Honesty locks: multi-hop lite ≠ graph RAG · PreferShorterHops default true · A3 lite ≠ NLP · supersede HITL · trigger-compact HITL · write ≠ turn · K4 lite ≠ dual-clock · patterns/anomalies not medical · dual_write OFF · not Memory GA · no invent lean HTTP for supersede/facts-as-of/patterns/trigger-compact. Paired with `<memory-advanced>` system note on `AttachMCP`. See [memory-mcp.md](./memory-mcp.md).

**Local-edge Docker attach (s1308 · peer aion s1306):** when palace MCP is the Docker edge (`http://127.0.0.1:8080/mcp`), advanced slash/MCP tools still apply once `[[mcp.servers]]` is attached — docker edge ≠ invent Memory GA. Operator steps: [Local-edge Docker Memory MCP](./memory-mcp.md#local-edge-docker-memory-mcp-s1308--peer-aion-s1306).

### Builtin: `gtm-draft-only-agent` (s1341)

Residual-honest **draft-only GTM AI agent roles** (Orchestrator · Content Creator · Campaign Planner · Lead Manager) — drafts/plans only · HITL publish · no auto-send / no auto-publish · mesh grounding via residual integrations list/plan MCP + portal HITL (never invent Connected / suite ops GA) · dual_write OFF · not Memory GA · book-demo OFF · residual PASS ≠ live dogfood. Aligns with aion hermes-grok-marketing-sales-pipeline Phase 2 local hard gates.

Paired with the `<gtm-draft-only>` system note injected on `AttachSkills` (`GtmDraftOnlyAgentGuidanceNote`, s1347).

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

**Residual next-step (s1837):** successful `list_skills` / `read_skill` results append `SkillsNextStepLines` — dual path residual-honest after skills list/read or skills reload: if TUI/session running → `/setup preflight` · `/setup reload` (skills re-scan · package wire ≠ Connected) · optional `list_skills` · `/onboard next setup`; cold start → restart `iomesh` · `iomesh setup preflight`. Honesty: skills re-scan ≠ invent Connected · package wire ≠ Connected · dual_write OFF · not Agent Plugins GA · not Memory GA. No dedicated `/skills` slash (catalog + tools + reload only). Errors stay bare (never invent success).

## CLI

```bash
iomesh skills
iomesh skills -C /path/to/workspace
```
