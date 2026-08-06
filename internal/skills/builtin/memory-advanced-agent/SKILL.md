---
name: memory-advanced-agent
description: Residual-honest agent path for advanced memory surfaces (related hops · supersede HITL · facts-as-of · digest · patterns/anomalies · timeline · compact-status · trigger-compact HITL · semantic · ingest-event · status inventory) — not Memory GA · dual_write OFF
---

# Memory advanced agent (residual-honest)

Agent path for **advanced Memory Palace surfaces** already wired in iomesh-tui (slash + MCP). Default auto-recall stays **single-hop** `memory_retrieve` / sync retrieve. These surfaces are **opt-in** only — never invent Memory GA, full graph RAG, dual-clock Graphiti, or silent mutates.

**System note (s1291):** when MCP is attached (`AttachMCP`), runtime also injects a residual-honest `<memory-advanced>` system note (`MemoryAdvancedAgentGuidanceNote`) that steers the same locks below. Skill + note stay consistent; skill is the full playbook.

## When to use each surface

| Surface | Use when | Do not use for |
|---------|----------|----------------|
| **related** (s1135 + s1281) | Operator wants multi-hop lite neighbors of a seed entity / query | Default chat auto-recall; full KG / graph RAG claims |
| **supersede** (s1282) | Human-confirmed close of open `valid_until` for an entity tag (mutating) | Silent auto-supersede; NLP contradiction detection |
| **facts-as-of** (s1276) | List facts valid at a point in time (`as_of` RFC3339) | Auto-recall; inventing lean HTTP route; dual-clock KG |
| **digest** (s1200) | Ops heartbeat day/week pattern + receipts pack | Claiming knowledge/analytical digests as GA |
| **patterns / anomalies** (shipped s1287) | MCP ops pulse Beta list of patterns or anomalies | Medical diagnosis; inventing GA window engine |
| **timeline** (s1296) | Temporal event-ordered palace slice (`since`/`until`/`query`/`limit`) | Auto-recall; inventing lean HTTP timeline; claiming Memory GA |
| **compact-status** (s1296 · read-only) | Palace tier counts + last compaction residual | Auto-compact product claims; inventing compaction green |
| **trigger-compact** (s1311 · HITL) | Human-confirmed RecMem compaction advisory (`memory.compact.trigger`) | Silent auto-trigger; inventing compaction green; calling without HITL |
| **semantic** (s1301) | Tier-4 semantic facts search (`query` / `limit`) | Auto-recall; inventing lean HTTP semantic; inventing empty-as-success |
| **ingest-event** (s1301) | Ops/telemetry event ingest (`subject` + `content`; s138 T1) | Conversation turns (use `/memory ingest` / `memory_ingest_turn`); inventing memory_id offline |
| **status** (s1311 advanced inventory) | Residual-honest advanced MCP tool presence pulse | Inventing product green from tool presence |

## Slash ↔ MCP mapping

| Slash | MCP tool(s) | Transport honesty |
|-------|-------------|-------------------|
| `/memory related --seed … [--query …] [--max-hops N] [--prefer-shorter-hops\|--legacy-sort]` | `memory_related` | Lean HTTP `POST /v1\|/v5/memory/related` prefer → MCP fallback |
| `/memory supersede --entity <key> [--as-of RFC3339] --i-confirm` | `memory_supersede_entity` | **MCP-first only** — no lean HTTP supersede invent |
| `/memory facts-as-of\|facts\|as-of --as-of <RFC3339> […]` | `memory_facts_as_of` | **MCP-first only** — no lean HTTP facts_as_of invent |
| `/memory digest [--window day\|week] [--horizon …] [--limit N]` | `ops_digest_export` | Lean HTTP `POST /v1\|/v5/memory/ops_digest` prefer → MCP fallback |
| `/memory patterns` (shipped s1287) | `memory_patterns_list` | MCP ops pulse Beta — shipped; no invent lean HTTP |
| `/memory anomalies` (shipped s1287) | `memory_anomalies_list` | MCP ops pulse Beta — shipped; no invent lean HTTP |
| `/memory timeline [--since\|--until\|--session-id\|--query\|--limit]` (s1296) | `memory_timeline` | **MCP-first only** — no lean HTTP timeline invent |
| `/memory compact-status` (s1296 · read-only) | `memory_compact_status` | **MCP-first only** — read-only tier counts residual; no invent lean HTTP |
| `/memory trigger-compact --i-confirm` (s1311 · HITL) | `memory_trigger_compact` | **MCP-first only** — mutating RecMem advisory; HITL required; no invent lean HTTP |
| `/memory semantic\|search-semantic\|sem [query\|--query …] [--limit N]` (s1301) | `memory_search_semantic` | **MCP-first only** — tier-4 semantic facts residual; empty ≠ invent |
| `/memory ingest-event\|event --subject <id> --content <text> […]` (s1301) | `memory_ingest_event` | **MCP-first only** — s138 T1 temporal event telemetry; not conversation turn |
| `/memory status` (s1311 advanced inventory) | (presence probe only) | Residual inventory of advanced MCP tools · dual_write OFF · not Memory GA |

Also inventory: `memory_retrieve` (default recall), `memory_ingest_turn` (conversation turns via `/memory ingest`) — see architecture docs.

**s1311 HITL shipped:** `memory_trigger_compact` is wired as `/memory trigger-compact --i-confirm` (aliases `--confirm` / `--yes`). Without HITL → residual-honest refuse (no MCP call).

## Workflow (agent)

1. **Discover memory MCP** — ensure `aion-memory-mcp` (or configured `[memory].server`) is attached. Offline → residual-honest fail-open; **never invent** hits / superseded_count / digests.

2. **Default recall** — use `memory_retrieve` / `/memory recall` (single-hop). Do **not** auto-run multi-hop related.

3. **Multi-hop related (opt-in)** — call `memory_related` with `seed_entity` and/or `query`, optional `max_hops`, `limit`, `tenant`, `session_id`.
   - `prefer_shorter_hops`: **omit = kernel default true** (path-aware hop ranking lite · s1067/s1277/s1281).
   - Pass `prefer_shorter_hops: false` only for legacy seed-first sort (`--legacy-sort` / `--no-prefer-shorter-hops`).
   - multi-hop lite ≠ full graph RAG · hop ranking path-aware lite · not Memory GA.

4. **Facts-as-of (opt-in · MCP-first)** — call `memory_facts_as_of` with required `as_of` (RFC3339). Optional `entity`, `query`, `limit`, `session_id`, `tenant`.
   - K4 bi-temporal **lite** · not full dual-clock Graphiti · empty facts ≠ invent memories.

5. **Ops digest (opt-in)** — call `ops_digest_export` with `window` (`day`|`week`), `horizon` (`ops`|`knowledge`|`analytical`|`all`), `limit`.
   - ops pulse **GA-path** · knowledge/analytical **Beta** · never invent GA.

6. **Supersede (opt-in · HITL mutating)** — call `memory_supersede_entity` only after **explicit human confirm**.
   - Slash requires `--i-confirm` (aliases `--confirm` / `--yes`). Agent must refuse residual-honestly without HITL.
   - Mutating A3 lite: closes open `valid_until` windows for entity tags · **not** NLP contradiction · **not** silent.
   - Never invent `superseded_count` offline.

7. **Patterns / anomalies (shipped s1287 · MCP ops pulse Beta)** — when tools present, list via `memory_patterns_list` / `memory_anomalies_list`.
   - Ops pulse Beta · not medical · not invent GA window engine · dual_write OFF.

8. **Timeline (s1296 · MCP-first · opt-in)** — call `memory_timeline` with optional `since`, `until`, `query`, `limit`, `session_id`, `tenant`.
   - Temporal timeline · filters before limit · empty entries ≠ invent memories · not Memory GA.

9. **Compact-status (s1296 · MCP-first · read-only)** — call `memory_compact_status` with optional `tenant`.
   - Palace tier counts residual · last_compaction from wire only · not auto-compact product · never invent compaction green.

10. **Trigger-compact (s1311 · MCP-first · HITL mutating)** — call `memory_trigger_compact` only after **explicit human confirm**.
    - Slash requires `--i-confirm` (aliases `--confirm` / `--yes`). Agent must refuse residual-honestly without HITL.
    - Mutating RecMem advisory: publishes `memory.compact.trigger` · **not** invent compaction green · **not** auto-compact product · dual_write OFF · not Memory GA.
    - Never invent `triggered` / `cluster_size` offline.

11. **Semantic (s1301 · MCP-first · opt-in)** — call `memory_search_semantic` with required `query`, optional `limit`, `tenant`.
    - Tier-4 semantic facts residual · empty facts ≠ invent memories · not Memory GA · dual_write OFF.

12. **Ingest-event (s1301 · MCP-first · opt-in)** — call `memory_ingest_event` with required `subject` + `content`, optional `event_time` / `session_id` / `session_seq` / `severity` / `source_stream` / `tenant`.
    - s138 T1 temporal event telemetry · **not** conversation turn (use `memory_ingest_turn` / `/memory ingest`) · never invent memory_id offline · dual_write OFF.

13. **Status inventory (s1311)** — `/memory status` prints `MemoryStatusLine` + `MemoryAdvancedStatus` residual tool presence (present/missing/offline). Presence ≠ product green.

## Residual honesty table

| Lock | Meaning |
|------|---------|
| multi-hop lite ≠ graph RAG | Related BFS hops + hop ranking lite — not full knowledge graph / Graph RAG product |
| PreferShorterHops default true | Omit `prefer_shorter_hops` → kernel true; false = legacy seed-first only |
| A3 lite ≠ NLP | Supersede closes validity windows; not contradiction NLP / auto-repair |
| supersede requires HITL | No silent supersede; `--i-confirm` / explicit Confirm required |
| K4 lite ≠ dual-clock | facts-as-of is bi-temporal lite validity listing — not full dual-clock Graphiti KG |
| patterns/anomalies not medical | Ops pulse Beta lists only — not clinical/diagnostic claims |
| dual_write OFF | Default dual-write audit OFF; local-primary palace honesty |
| not Memory GA | Advanced surfaces are residual/Beta/lite — do not invent product Memory GA |
| no invent GA | No invent GA window engine, lean HTTP for supersede/facts-as-of/patterns/timeline/compact, or empty-as-success |
| opt-in only | Never auto multi-hop on default recall; never auto-mutate supersede / trigger-compact |
| fail-open | Offline / missing tool → residual status, not invented payloads |
| empty ≠ invent | Empty facts / zero superseded_count / empty digest / empty timeline / empty semantic = honest empty |
| compact-status read-only | Tier counts residual only — not auto-compact product · not invent compaction green |
| trigger_compact requires HITL (s1311) | `memory_trigger_compact` is mutating RecMem advisory — **HITL shipped** via `/memory trigger-compact --i-confirm` · not invent compaction green |
| semantic tier-4 residual | `memory_search_semantic` · not Memory GA · empty ≠ invent (s1301) |
| ingest-event ≠ turn | s138 T1 telemetry event · not conversation turn · never invent memory_id (s1301) |

## Non-goals (never do)

- Do **not** auto-run multi-hop related on default auto-recall.
- Do **not** silent supersede (always HITL / `--i-confirm`).
- Do **not** silent trigger-compact (always HITL / `--i-confirm`; s1311).
- Do **not** invent lean HTTP for supersede, facts-as-of, patterns/anomalies, timeline, compact-status, trigger-compact, semantic, or ingest-event.
- Do **not** invent Memory GA, full graph RAG, dual-clock Graphiti, or medical diagnosis.
- Do **not** claim dual_write ON or book-demo ON by default.
- Do **not** invent GA window engine from digest / patterns / anomalies Beta surfaces.
- Do **not** treat knowledge/analytical digest horizons as ops GA-path.
- Do **not** invent memories, superseded_count, digests, timeline entries, semantic facts, memory_id, triggered/cluster_size, or compaction green when offline / empty.
- Do **not** treat `memory_ingest_event` / `/memory ingest-event` as conversation turn ingest (use `memory_ingest_turn` / `/memory ingest`).
- Do **not** treat advanced tool presence as product Memory GA green.

## Related

- Builtin skill always available when skills enabled (s1288 · mold s1251 connector skill).
- System note inject on `AttachMCP`: `<memory-advanced>` via `MemoryAdvancedAgentGuidanceNote` (s1291).
- Architecture SSOT: `docs/architecture/memory-mcp.md`.
- Shipped s1287: `/memory patterns|anomalies` MCP ops pulse Beta.
- Shipped s1296: `/memory timeline|compact-status` MCP-first (read-only compact-status).
- Shipped s1301: `/memory semantic|ingest-event` MCP-first (tier-4 semantic · s138 T1 event telemetry).
- Shipped s1311: `/memory trigger-compact --i-confirm` HITL + `/memory status` advanced inventory pulse.
- Serials: s1135 related · s1281 prefer_shorter_hops · s1282 supersede · s1276 facts-as-of · s1200 digest · s1287 patterns/anomalies · s1288 skill · s1291 system note · s1296 timeline+compact-status · s1301 semantic+ingest-event · s1311 trigger-compact HITL + advanced status · aion s1277 / s640 A3 lite / K4 lite / s138 T1.
