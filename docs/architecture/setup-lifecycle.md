# Setup lifecycle (agent-native wizard foundation)

**Serial:** free eng **s1525** P1–P2 · **s1526** P3–P4 · **s1530** P5 · **s1534** P6 · **s1538** P7 · **s1542** closeout residual · **s1546** still-human APPLY reaffirm · **s1550** edge-first human-gates residual pin · **s1574** still-human APPLY soft dogfood residual · **s1670** easy setup: `/setup reload` re-scans skills · residual-honest · **s1686** CLI `setup init` next-step dual path (`/setup reload` vs cold restart) · **s1699** setup preflight next-step dual path (`/setup reload` vs cold restart) · **s1707** setup drift/repair dual-path next-step · **s1711** setup reload/pull/analyze next-step honesty  
**Status:** foundation + agent-native slash/skill + package wire + `ReplaceMCP` + **`ReplaceSkills` (s1670)** + in-session opt-in continuous pull + analyze ticks + report-only drift + **guided repair** (safe steps · explicit `--yes`) + **onboard next setup** lane + **CLI init dual-path next-step (s1686)** + **preflight dual-path next-step (s1699)** + **drift/repair dual-path next-step (s1707)** + **reload/pull/analyze next-step (s1711)**  
**Shipped P7:** `/setup repair` plan + apply `--yes` (safe steps only · notes stay human)  
**Shipped s1542:** residual-honest `/onboard next setup` consolidates P1–P7 map story  
**Related (s1546):** still-human APPLY reaffirm after closeout — setup residual complete ≠ invent human-gate green / live APPLY / E10 (`/onboard next human-gates`)  
**Related (s1550):** edge-first human-gates residual pin — knowledge multi-tenant punted · Slack HMAC punted · portal HITL when connect · dual_write OFF · Edge Memory GA candidacy only (`/onboard next human-gates`)  
**Related (s1574):** still-human APPLY soft dogfood residual after Wave C continuum — open boxes stay open · PASS ≠ invent human-gate green / live APPLY (`/onboard next human-gates dogfood`)  
**Related (s1554):** edge-user-journey SSOT — this doc is **stage 4 (Setup wizard)** in the 7-stage narrative · see [edge-user-journey.md](./edge-user-journey.md)  
**Related (s1558 Wave B first-run polish):** setup is stage 4 of edge-user-journey · full first-run map `/onboard next journey` · setup guidance maps stages 1–7 residual-honest · free eng **s1558**

## Goal

Enable the map story without inventing Connected / Memory GA:

```text
setup init → write managed config → start memory host → preflight probe
  → /setup reload (or restart TUI) → optional portal HITL integrations
  → /setup pull start (or CLI iomesh memory pull)
  → /setup analyze start (or /memory digest one-shot)
  → /setup drift (report-only)
  → /setup repair plan · /setup repair apply --yes (safe steps only)
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

# Continuous / once pull still valid as CLI
iomesh memory pull --stream EVENTS --name tui-local-palace --once --dry-run
```

### After `iomesh setup init` (s1686 dual path)

Post-write next steps are residual-honest (helper `setup.SetupInitNextStepLines`):

| Path | When | Next |
|------|------|------|
| **In-session** | TUI/session already running | `/setup preflight` · **`/setup reload`** (hot-swap MCP + re-scan skills · package wire ≠ Connected) |
| **Cold start** | No session / CLI-only | **restart `iomesh`** · `iomesh setup preflight` |

**Honesty:** CLI has **no** `iomesh setup reload` subcommand — in-session `/setup reload` only · dual_write **OFF** · not Memory GA · catalog ≠ Connected · package wire ≠ Connected · free eng **s1686**.

### After `iomesh setup preflight` / `/setup preflight` (s1699 dual path)

Post-probe next steps are residual-honest (helper `setup.SetupPreflightNextStepLines` · appended by `FormatPreflightText`):

| Path | When | Next |
|------|------|------|
| **In-session** | Preflight ok · TUI/session already running | **`/setup reload`** (hot-swap MCP + skills · package wire ≠ Connected) |
| **Host/secrets missing** | Probe not ok · host or env still missing | start `iomesh-memory-mcp` · set secret env · re-run preflight |
| **Cold start** | Preflight ok · no session / CLI-only | **restart `iomesh`** (CLI has **no** `setup reload`) · then `/setup reload` in session if needed |

**Honesty:** CLI has **no** `iomesh setup reload` · dual_write **OFF** · not Memory GA · catalog ≠ Connected · PASS ≠ invent install green · package wire ≠ Connected · free eng **s1699** (peer of s1686 init next-step).

### Profiles

| Profile | Writes |
|---------|--------|
| `local-memory` | `[mcp]` + memory server URL/stdio + `[memory]` dual_write=false · pull_continuous=false · analyze_continuous=false |
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

## Slash `/setup` (s1526 P3 + s1530 P5 + s1534 P6 + s1538 P7)

Agent-native operator surface (alias `/setup-lifecycle`):

```text
/setup                         # residual-honest usage + honesty locks
/setup init [profiles] …       # write managed fragment (user config path)
/setup init local-memory --print-only
/setup init --stdio            # stdio iomesh-memory-mcp instead of HTTP URL
/setup preflight               # aliases status|check — FormatPreflightText
/setup portal                  # console.iome.sh/integrations + settings/agent
/setup reload                  # hot-swap MCP + re-scan skills (P4 + s1670 · package wire ≠ Connected)
/setup pull                    # continuous pull status (alias status)
/setup pull status
/setup pull start [--once] [--config path]
/setup pull once [--config path]
/setup pull stop
/setup analyze                 # analyze tick status (alias status)
/setup analyze status
/setup analyze start [--mode status|digest] [--interval N] [--window day|week] [--config path]
/setup analyze once …
/setup analyze stop
/setup drift [--config path]   # report-only FormatDriftText
/setup maintain                # alias of drift
/setup repair                  # guided plan from current drift (default = plan)
/setup repair plan [--config path]
/setup repair apply --yes [--config path]   # safe steps only · refuse without --yes
```

| Subcommand | Behavior |
|------------|----------|
| bare / `help` | usage + honesty one-liner (dual_write OFF · not Memory GA · pull/analyze opt-in · drift · guided repair) |
| `init` | `setup.BuildManagedFragment` + `config.WriteSetupManagedUser` (or `--print-only`) |
| `preflight` / `status` / `check` | `setup.Preflight` + `FormatPreflightText` (s1699 dual-path next-step appended) |
| `portal` | browser HITL URLs only |
| `reload` | `Wire` + `ReplaceSkills` + `ConnectMCP` + `ReplaceMCP` (s1670 skills re-scan · optional `--config path` · s1711 next-step appended via `SetupReloadNextStepLines`) |
| `pull` | in-session continuous pull status/start/once/stop (s1530 P5 · s1711 next-step via `SetupPullNextStepLines`) |
| `analyze` | in-session analyze tick status/start/once/stop (s1534 P6 · s1711 next-step via `SetupAnalyzeNextStepLines`) |
| `drift` / `maintain` | report-only `BuildDriftReport` + `FormatDriftText` (s1534 P6 · s1707 dual-path next-step appended) |
| `repair` | guided `PlanRepair` / `ApplyRepairPlan` (s1538 P7 · plan default · apply requires `--yes` · s1707 dual-path next-step on FormatRepair*) |

Simple flags on slash `init`: `--stdio` · `--print-only` · `--plugins-dir path` · `--memory-url URL`. Full flag set remains on CLI `iomesh setup init`.

After init: start memory host (if needed) · set secret env vars · `/setup reload` (or restart TUI) · optional `/setup pull start` when mesh + consumer configured · optional `/setup analyze start` · `/setup drift` for residual next steps · optional `/setup repair apply --yes` for safe guided steps only.

## Continuous pull (s1530 P5 + s1711 next-step)

In-session opt-in continuous mesh → local MCP palace pull:

| Path | How |
|------|-----|
| Slash | `/setup pull start` (MaxLoops=0 continuous) · `/setup pull once` · `/setup pull stop` · `/setup pull status` |
| Config | `[memory] pull_continuous = true` (setup fragment default **false**) |
| CLI | `iomesh memory pull` still valid |

Config knobs reused: `pull_stream` (default `EVENTS`) · `pull_consumer` · `pull_filter` · `pull_batch` · `pull_max_wait_ms` · `server` · `tenant`.

### After `/setup pull` (s1711 dual path)

Post-pull status/start/once/stop next steps are residual-honest (helper `setup.SetupPullNextStepLines`):

| Path | When | Next |
|------|------|------|
| **In-session** | TUI/session running | `/setup pull status` · optional `/setup analyze start` · `/setup drift` · `/memory digest` |
| **CLI** | No session / operator prefers CLI | `iomesh memory pull` (once or continuous · mesh + consumer required) |

**Honesty:** dual_write **OFF** · not Memory GA · **pull ≠ invent Connected** · idle/status must not invent green · CLI still valid · free eng **s1711** (peer of s1686 init · s1699 preflight · s1707 drift/repair).

## Analyze ticks (s1534 P6 + s1711 next-step)

In-session opt-in status/digest pulse on agent Runtime:

| Path | How |
|------|-----|
| Slash | `/setup analyze start` · `/setup analyze once` · `/setup analyze stop` · `/setup analyze status` |
| Config | `[memory] analyze_continuous = true` (setup fragment default **false**) · `analyze_interval_sec` · `analyze_mode` |
| One-shot | `/memory digest` still valid |

Flags: `--mode status|digest` · `--interval N` (seconds; Runtime default 300, floor 30) · `--window day|week` (digest) · `--config path`.

### After `/setup analyze` (s1711 dual path)

Post-analyze status/start/once/stop next steps are residual-honest (helper `setup.SetupAnalyzeNextStepLines`):

| Path | When | Next |
|------|------|------|
| **In-session** | TUI/session running | `/setup analyze status` · optional `/setup pull start` · `/setup drift` · re-run analyze |
| **One-shot digest** | Prefer residual ops pulse without tick | **`/memory digest`** (still valid · not invent Connected) |

**Honesty:** dual_write **OFF** · not Memory GA · **analyze tick ≠ invent Connected** · `/memory digest` still valid · idle/status must not invent green · free eng **s1711** (peer of s1686 init · s1699 preflight · s1707 drift/repair).

## Drift / maintain (s1534 P6 + s1707 dual-path next-step)

Report-only config intent vs runtime snapshot:

| Path | How |
|------|-----|
| Slash | `/setup drift` · `/setup maintain` (alias) |
| API | `setup.BuildDriftReport(cfg, snap)` · `setup.FormatDriftText(rep)` · `Runtime.DriftSnapshot()` |

### After `/setup drift` / `/setup maintain` (s1707 dual path)

Post-report next steps are residual-honest (helper `setup.SetupDriftNextStepLines` · appended by `FormatDriftText`):

| Path | When | Next |
|------|------|------|
| **In-session** | TUI/session running | `/setup repair plan` · `/setup repair apply --yes` (safe only) · `/setup reload` when MCP drift · optional `/setup pull\|analyze start` |
| **Cold start** | No session / CLI-only | fix host/config · `iomesh setup preflight` · **restart `iomesh`** (CLI has **no** setup drift/repair/reload) |

**Honesty:** drift report-only · dual_write **OFF** · package wire ≠ Connected · not Memory GA · CLI has **no** setup drift/repair/reload as full product · free eng **s1707** (peer of s1686 init · s1699 preflight). Residual notes still point at guided `/setup repair plan` · `/setup repair apply --yes` (safe steps only · dual_write never auto ON).

## Guided repair (s1538 P7 + s1707 dual-path next-step)

Plan + optional apply of **safe** residual steps from a drift report:

| Path | How |
|------|-----|
| Slash plan | `/setup repair` · `/setup repair plan` → `PlanRepair` + `FormatRepairPlan` (no side effects) |
| Slash apply | `/setup repair apply --yes` → `ApplyRepairPlan(…, dryRun=false)` · refuse without `--yes` |
| API | `setup.PlanRepair(rep)` · `FormatRepairPlan` · `FormatRepairResult` · `ApplyRepairPlan(ctx, plan, executor, dryRun)` |
| Executor | TUI `setupRepairExecutor`: `ReloadMCP` · `StartPull` · `StartAnalyze` |

**Safe auto-apply kinds (with `--yes` only):** `reload_mcp` · `start_pull` (when pull_continuous + mesh) · `start_analyze` (when analyze_continuous).

**Note/manual kinds (never auto-applied):** dual_write flip · memory host start · mesh `[iomesh]` config · noop.

### After `/setup repair` plan/result (s1707 dual path)

Post-plan / post-apply next steps are residual-honest (helper `setup.SetupRepairNextStepLines` · appended by `FormatRepairPlan` + `FormatRepairResult`):

| Path | When | Next |
|------|------|------|
| **In-session** | TUI/session running | re-run `/setup drift` · `/setup reload` after safe apply · optional pull/analyze |
| **Cold start** | No session / CLI-only | **restart `iomesh`** · `iomesh setup preflight` (CLI has **no** setup repair/reload) |

**Honesty:** dual_write OFF · not Memory GA · **repair apply ≠ invent Connected** · package wire ≠ Connected · portal HITL still human · dual_write never auto-flipped ON · no auto-repair without explicit `apply --yes` · free eng **s1707**.

## Agent surfaces (s1526 P3 + s1530 P5 + s1534 P6 + s1538 P7)

| Surface | Detail |
|---------|--------|
| Builtin skill | `setup-lifecycle-agent` via `go:embed` under `internal/skills/builtin/` — always merged when skills enabled |
| System note | `<setup-lifecycle>` via `setup.SetupLifecycleAgentGuidanceNote()` on `AttachMCP` |
| Slash | `/setup` / `/setup-lifecycle` including `pull` · `analyze` · `drift` · `repair` |

Skill + note + slash share honesty locks; skill is the full playbook.

## Honesty locks

| Lock | Rule |
|------|------|
| dual_write OFF | Managed fragment refuses `dual_write = true` |
| not Memory GA | Preflight never stamps Memory GA |
| catalog ≠ Connected | Setup PASS ≠ invent install green |
| secrets | Env **names** only (`api_key_env`, `oauth_token_env`) |
| portal HITL | OAuth/install still browser |
| continuous pull opt-in | `/setup pull` · `pull_continuous` · CLI `iomesh memory pull` still valid · pull ≠ invent Connected |
| analyze ticks opt-in | `/setup analyze` · `analyze_continuous` · `/memory digest` still valid · analyze tick ≠ invent Connected |
| drift report-only | `/setup drift` · `/setup maintain` · residual next steps · drift ≠ invent install green · package wire ≠ Connected |
| guided repair explicit | `/setup repair` · `apply --yes` only · safe steps · repair apply ≠ invent Connected · no auto-repair without `--yes` |
| onboard next setup (s1542) | `/onboard next setup` offline map · `setup_not_probed` · offline static ≠ live dogfood · setup closeout residual ≠ invent Edge Memory GA · E10 Open |

## Onboard next setup lane (s1542)

Residual-honest offline static continuum for the full setup lifecycle map (no MCP dial):

| Surface | Detail |
|---------|--------|
| Slash | `/onboard next setup` (aliases `setup-lifecycle` · `lifecycle` · `setup_lifecycle`) |
| API | `agent.AionAgentOnboardingNextSetupLane()` |
| Board vocab | `path_ready` · `residual_only` · **`setup_not_probed`** |
| Map steps | init → memory host/secrets → preflight → reload → portal HITL → optional pull/analyze → drift → repair plan/apply `--yes` → `/memory digest` still valid |
| Companion | `/onboard next journey` (s1558 first-run map) · `/onboard next wizard` (s1570 Wave C guided residual) · `memory` · `memory-pull` · `human-gates` · `operator` · skill `setup-lifecycle-agent` · [edge-user-journey.md](./edge-user-journey.md) · [memory-edge-usage-demo.md](./memory-edge-usage-demo.md) |

**Honesty:** dual_write OFF · not Memory GA · package wire ≠ Connected · PASS ≠ invent Connected · repair apply ≠ invent Connected · dual_write never auto ON · portal HITL still human · still-human APPLY open · **E10 Open** · offline static lane ≠ live dogfood · setup closeout residual ≠ invent Edge Memory GA · stage 4 of edge-user-journey · free eng **s1558**.

## Onboard next journey lane (s1558 Wave B)

Residual-honest offline static first-run map of the 7-stage edge-user-journey (no MCP dial):

| Surface | Detail |
|---------|--------|
| Slash | `/onboard next journey` (aliases `edge-journey` · `user-journey` · `first-run` · `edge_user_journey`) |
| API | `agent.AionAgentOnboardingNextJourneyLane()` |
| Stages | 1 Signup · 2 Download TUI · 3 TUI auth/keys · 4 Setup wizard · 5 Connectors · 6 Local store · 7 Analyze |
| Stage 4 detail | `/onboard next setup` · `/setup` · this document |
| Docs | [edge-user-journey.md](./edge-user-journey.md) · this file · [memory-edge-usage-demo.md](./memory-edge-usage-demo.md) |

**Honesty:** dual_write OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA · portal HITL · agent MCP cannot write installs · catalog ≠ Connected · book-demo OFF · no invent TUI portal SSO · host not auto · free eng **s1558** · free-floor peer **s1560+** mention only.

## Onboard next wizard lane (s1570 Wave C)

Residual-honest offline guided first-run wizard residual after Wave B journey map (no MCP dial · never invent full interactive auto wizard):

| Surface | Detail |
|---------|--------|
| Slash | `/onboard next wizard` (aliases `first-run-wizard` · `guided` · `wave-c` · `wave_c` · `wizard-residual`) |
| Soft dogfood | `/onboard next wizard dogfood` (aliases `soft` · `samples` · `offline` · `wizard-soft`) → `RunFirstRunWizardSoftDogfood` |
| API | `agent.AionAgentOnboardingNextWizardLane()` |
| Scope | Deeper guided residual map + soft dogfood · **not** full interactive auto wizard UX |
| Companion | `/onboard next journey` (Wave B) · setup · portal-hitl · e4 · human-gates |
| Docs | [edge-user-journey.md](./edge-user-journey.md) Wave C row · this file |

**Honesty:** dual_write OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · E10 Open · portal HITL when connect · agent MCP cannot write installs · catalog ≠ Connected · no invent TUI portal SSO · host not auto · residual PASS ≠ invent full interactive auto wizard · free eng **s1570** · free-floor peer **s1572+** mention only.

## Preflight states

| State | Meaning |
|-------|---------|
| `not_started` | No config file |
| `config_present` / `config_written` | File exists |
| `awaiting_memory_host` | Memory configured but healthz/PATH fail |
| `local_memory_probe_ok` | Memory host healthz OK (or stdio binary on PATH) |

## Hot reload (s1526 P4 · s1670 skills re-scan · s1711 next-step)

Shared package wire (`internal/runtimewire`):

| API | Role |
|-----|------|
| `Wire(cfg, workspace, logger)` | skill dirs + `[]mcp.ServerConfig` (TOML then plugins) |
| `ConnectMCP(ctx, cfg, workspace, logger)` | `*mcp.Manager` when MCP feature on |
| `Runtime.ReplaceMCP(mgr)` | close previous · unregister `mcp__*` · re-attach |
| `Runtime.ReplaceSkills(cat)` | unregister `list_skills`/`read_skill` · re-attach (s1670) |

**Honesty:** package wire ≠ Connected · dual_write OFF · Discover/map ≠ install APPLY green · skills re-scan ≠ invent Connected · not Agent Plugins GA.  
Skills catalog **is** re-scanned on `/setup reload` via `Wire` SkillDirs + `LoadWithBuiltin` + `ReplaceSkills` (s1670 · including plugin skill dirs when `[plugins]` enabled · no process restart for skill-only path changes after reload). Guided repair `reload_mcp` uses the same path.

### After `/setup reload` (s1711 next-step · in-session only)

Post-reload next steps are residual-honest (helper `setup.SetupReloadNextStepLines`):

| Path | When | Next |
|------|------|------|
| **In-session** | After successful hot-swap | optional `/setup pull start` · `/setup analyze start` · `/setup drift` · `/memory digest` |
| **CLI** | — | **none** — CLI has **no** `iomesh setup reload` (in-session only · peers s1686/s1699) |

**Honesty:** package wire ≠ Connected · dual_write **OFF** · not Memory GA · skills re-scan ≠ invent Connected · CLI has **no** setup reload · free eng **s1711** (peer of s1686 init · s1699 preflight · s1707 drift/repair).

## Phases (plan)

- ~~`/setup` slash + `setup-lifecycle-agent` skill~~ **shipped s1526 P3**  
- ~~`ReplaceMCP` / package wire / `/setup reload`~~ **shipped s1526 P4**  
- ~~In-session continuous pull (`/setup pull` · `pull_continuous`)~~ **shipped s1530 P5**  
- ~~Analyze auto-ticks (`/setup analyze` · `analyze_continuous`) + drift report~~ **shipped s1534 P6**  
- ~~Guided repair (`/setup repair` plan · `apply --yes` safe steps)~~ **shipped s1538 P7**  
- ~~Onboard next setup lane (P1–P7 closeout residual map)~~ **shipped s1542**  
- ~~Still-human APPLY reaffirm after setup closeout~~ **shipped s1546** (`/onboard next human-gates` · setup residual ≠ invent APPLY)
- ~~Edge-first human-gates residual pin~~ **shipped s1550** (`/onboard next human-gates` · knowledge multi-tenant punted · Slack HMAC punted · portal HITL when connect)
- ~~Wave B first-run journey polish~~ **shipped s1558 residual** (`/onboard next journey` · setup stage-4 map · guidance first-run · free eng s1558)
- ~~Still-human APPLY soft dogfood residual after Wave C continuum~~ **shipped s1574** (`/onboard next human-gates dogfood` · open boxes stay open · free eng s1574)
- ~~Easy setup skills re-scan on `/setup reload`~~ **shipped s1670** (`ReplaceSkills` · Wire SkillDirs · restart no longer required for skill-only path changes)
- ~~CLI `setup init` next-step dual path~~ **shipped s1686** (in-session `/setup reload` vs cold restart · no invent CLI `setup reload`)
- ~~setup preflight next-step dual path~~ **shipped s1699** (`FormatPreflightText` appends `SetupPreflightNextStepLines` · in-session `/setup reload` vs cold restart · no invent CLI `setup reload`)
- ~~setup drift/repair next-step dual path~~ **shipped s1707** (`FormatDriftText` / `FormatRepairPlan` / `FormatRepairResult` append dual-path next-step · in-session slash vs cold restart · no invent CLI setup drift/repair/reload)
- ~~setup reload/pull/analyze next-step honesty~~ **shipped s1711** (`SetupReloadNextStepLines` · `SetupPullNextStepLines` · `SetupAnalyzeNextStepLines` · reload in-session only · pull dual path slash vs CLI `iomesh memory pull` · analyze dual path slash vs `/memory digest` · package wire ≠ Connected · pull/analyze tick ≠ invent Connected)

See product plan: agent-native MCP/plugin setup wizard + continuous pull/analyze + guided repair.

## Related

- [edge-user-journey.md](./edge-user-journey.md) — 7-stage edge-first SSOT (s1554); this file = stage 4  
- [memory-advanced-install.md](./memory-advanced-install.md)  
- [memory-edge-usage-demo.md](./memory-edge-usage-demo.md)  
- [agent-plugins.md](./agent-plugins.md)  
- [skills.md](./skills.md)  
