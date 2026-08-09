---
name: setup-lifecycle-agent
description: Residual-honest agent-native setup lifecycle (init/preflight · dual_write OFF · not Memory GA · catalog ≠ Connected · portal HITL · not invent install green · continuous pull/analyze opt-in · drift report-only · guided repair apply --yes)
---

# Setup lifecycle agent (residual-honest · s1526 P3 + s1530 P5 + s1534 P6 + s1538 P7 + s1542 closeout + s1558 Wave B first-run)

Agent-native path to **bootstrap** local TUI config planes via managed fragment write + preflight probes + in-session opt-in continuous pull / analyze ticks + report-only drift + **guided repair** (safe steps only with explicit `--yes`) — **not** invent Connected / Memory GA / INSTALL_STORE green.

Prefer slash `/setup` (alias `/setup-lifecycle`) or CLI `iomesh setup` when the operator is at a terminal. Use this skill when planning setup steps in chat.

**Onboard companion (s1542):** residual-honest offline map via `/onboard next setup` (aliases `setup-lifecycle` / `wizard` / `lifecycle` / `setup_lifecycle`) → `AionAgentOnboardingNextSetupLane` — consolidates P1–P7 map story · **setup_not_probed** · dual_write OFF · package wire ≠ Connected · repair apply ≠ invent Connected · dual_write never auto ON · E10 Open · offline static ≠ live dogfood · setup closeout residual ≠ invent Edge Memory GA.

**Wave B first-run (s1558):** setup is **stage 4** of the 7-stage edge-user-journey. Full first-run map via companion `/onboard next journey` (aliases `edge-journey` / `user-journey` / `first-run` / `edge_user_journey`) → `AionAgentOnboardingNextJourneyLane` — Signup → Download TUI → TUI auth/keys → Setup wizard → Connectors → Local store → Analyze · dual_write OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA · portal HITL · host not auto · no invent TUI portal SSO · free eng **s1558** · free-floor peer **s1560+** mention only · docs `edge-user-journey.md` · `setup-lifecycle.md` · `memory-edge-usage-demo.md`.

## Workflow

1. **Init managed config** — write residual-honest fragment into user `config.toml`.
   - CLI: `iomesh setup init [profiles]`
   - Slash: `/setup init [profiles] [--stdio] [--print-only] [--plugins-dir path]`
   - Profiles: `local-memory` (default) · `plugins` · `mesh` · `platform-mcp` · `all`
   - Managed block markers: `# BEGIN iomesh-setup-managed` … `# END iomesh-setup-managed`
   - **dual_write = false always** — setup path refuses `dual_write = true`
   - **pull_continuous = false** default — continuous pull is opt-in only
   - **analyze_continuous = false** default — analyze ticks are opt-in only
   - Secrets as **env names only** (`api_key_env`, `oauth_token_env`) — never commit secret values
   - After write: start memory host if local-memory · set env vars · **`/setup reload`** (hot-swaps MCP · package wire ≠ Connected · skills may still need restart)

2. **Preflight probe** — residual-honest state, never invent install green.
   - CLI: `iomesh setup preflight [--json]`
   - Slash: `/setup preflight` (aliases `status` · `check`)
   - States: `not_started` · `config_present` / `config_written` · `awaiting_memory_host` · `local_memory_probe_ok`
   - **PASS ≠ invent Connected / INSTALL_STORE green / Memory GA**

3. **Portal HITL** — OAuth / connector install still browser session.
   - Slash: `/setup portal`
   - URLs: https://console.iome.sh/integrations · https://console.iome.sh/settings/agent
   - Agent MCP **cannot write installs**

4. **Continuous pull (s1530 P5 · residual-honest opt-in)** — in-session slash **or** CLI.
   - Slash: `/setup pull` · `/setup pull status` · `/setup pull start` · `/setup pull once` · `/setup pull stop`
   - Config opt-in: `[memory] pull_continuous = true` (default **false** in setup fragment)
   - CLI still valid: `iomesh memory pull` (requires mesh + consumer configured)
   - Loads `[memory] pull_stream` / `pull_consumer` / `pull_filter` / batch / max wait
   - **pull ≠ invent Connected** · dual_write OFF · not Memory GA · idle/status must not invent green

5. **Analyze ticks (s1534 P6 · residual-honest opt-in)** — in-session status/digest pulse.
   - Slash: `/setup analyze` · `/setup analyze status` · `/setup analyze start` · `/setup analyze once` · `/setup analyze stop`
   - Flags: `--mode status|digest` · `--interval N` · `--window day|week` (digest) · `--config path`
   - Config opt-in: `[memory] analyze_continuous = true` (default **false**) · `analyze_interval_sec` · `analyze_mode`
   - **`/memory digest` still valid** as one-shot residual ops pulse
   - **analyze tick ≠ invent Connected** · dual_write OFF · not Memory GA

6. **Drift / maintain (s1534 P6 · report-only)** — config intent vs runtime snapshot.
   - Slash: `/setup drift` · `/setup maintain` (alias)
   - Report-only residual next steps · **drift report ≠ invent install green** · package wire ≠ Connected
   - Notes point at guided repair: `/setup repair plan` · `/setup repair apply --yes` (safe steps only · dual_write never auto ON)
   - After drift, optional guided repair (P7) — not automatic without explicit `--yes`

7. **Guided repair (s1538 P7 · explicit --yes only)** — plan from drift · apply safe steps only.
   - Slash: `/setup repair` · `/setup repair plan` — `PlanRepair` + `FormatRepairPlan` (dry plan · no side effects)
   - Slash: `/setup repair apply --yes` — `ApplyRepairPlan` safe steps only (`reload_mcp` · `start_pull` · `start_analyze`)
   - **`/setup repair apply` without `--yes` refuses** (residual-honest · no auto-repair)
   - Notes (never auto-applied): dual_write manual flip · memory host start · mesh `[iomesh]` config
   - **repair apply ≠ invent Connected** · dual_write never auto-flipped ON · portal HITL still human
   - dual_write OFF · not Memory GA · package wire ≠ Connected

## Honesty locks (never violate)

| Lock | Rule |
|------|------|
| dual_write OFF | Managed fragment + setup path never force dual_write ON |
| not Memory GA | Preflight / init never stamp Memory GA |
| catalog ≠ Connected | Setup PASS / probe_ok ≠ invent install Connected |
| portal HITL | OAuth / INSTALL_STORE APPLY stay browser |
| secrets env names only | `api_key_env` / `oauth_token_env` — no secret values in config |
| continuous pull opt-in | `/setup pull` · `pull_continuous` · CLI `iomesh memory pull` still valid · pull ≠ invent Connected |
| analyze ticks opt-in | `/setup analyze` · `analyze_continuous` · `/memory digest` still valid · analyze tick ≠ invent Connected |
| drift report-only | `/setup drift` · `/setup maintain` · residual next steps · drift ≠ invent install green · package wire ≠ Connected |
| guided repair explicit | `/setup repair` · `/setup repair apply --yes` only · safe steps · repair apply ≠ invent Connected · no auto-repair without `--yes` |
| never invent green | No Connected / INSTALL_STORE green / Memory GA from setup alone |

## Non-goals (never do)

- Do **not** invent Connected / INSTALL_STORE APPLY green from setup PASS, pull status, analyze ticks, drift OK, or repair apply.
- Do **not** claim Memory GA or dual_write ON.
- Do **not** mint OAuth tokens or write connector installs from agent MCP.
- Do **not** auto-start continuous pull without opt-in (`/setup pull start` · `pull_continuous=true` · or CLI).
- Do **not** auto-start analyze ticks without opt-in (`/setup analyze start` · `analyze_continuous=true`).
- Do **not** auto-repair without explicit `/setup repair apply --yes` (plan is dry; notes stay human).
- Do **not** auto-flip dual_write ON or invent host/mesh green from repair.
- Do **not** treat catalog Beta/available as org Connected counts.

## Operator surfaces

| Surface | Action |
|---------|--------|
| Slash `/setup` | help · init · preflight · portal · reload · **pull** · **analyze** · **drift** · **repair** |
| Alias `/setup-lifecycle` | same |
| `/setup pull` | status · start · once · stop (s1530 P5 residual-honest) |
| `/setup analyze` | status · start · once · stop (s1534 P6 residual-honest) |
| `/setup drift` / `/setup maintain` | report-only FormatDriftText (s1534 P6) |
| `/setup repair` | plan (default) · apply --yes (s1538 P7 guided safe steps) |
| CLI `iomesh setup` | init · preflight (s1525 P1–P2) |
| CLI `iomesh memory pull` | still valid continuous / once pull path |
| `/memory digest` | still valid one-shot ops pulse |
| System note | `<setup-lifecycle>` on AttachMCP |
| Skill | `read_skill setup-lifecycle-agent` |
| Onboard map (s1542) | `/onboard next setup` (aliases setup-lifecycle\|wizard\|lifecycle\|setup_lifecycle) |
| First-run journey (s1558) | `/onboard next journey` (aliases edge-journey\|user-journey\|first-run\|edge_user_journey) |

## Related

- Docs: `docs/architecture/setup-lifecycle.md` · `docs/architecture/edge-user-journey.md` · demo polish `docs/architecture/memory-edge-usage-demo.md`
- Builtin always merged when skills enabled (`go:embed`)
- Integrations residual path: skill `connector-integrations-setup` · `/integrations`
- Memory advanced residual: skill `memory-advanced-agent` · `/memory`
- Onboarding continuum: skill `aion-agent-onboarding` · `/onboard` · **`/onboard next setup`** (s1542 P1–P7 closeout residual · stage 4) · **`/onboard next journey`** (s1558 Wave B first-run map)
