# Memory edge usage demo (example)

**Serial:** free eng **s1513** · residual-honest **utilization / demo walkthrough** for product tip  
**SSOT:** product narrative stages live in **[edge-user-journey.md](./edge-user-journey.md)** (free eng **s1554** · 7-stage edge-first journey) — this file is the **runbook**, not the ownership SSOT.  
**Audience:** operators, demo hosts, sales eng dogfood, new users  
**Planes used:** local Memory **kernel** (`github.com/iome-sh/memory`) + product **MCP host** (`iomesh-memory-mcp`) + **iomesh-tui** client

End-to-end **example** story:

```text
signup (portal)  →  install TUI  →  MCP integrations (list/plan + portal HITL)
                 →  local memory install (kernel + MCP host)  →  attach TUI
                 →  show usage (/memory · mcp --connect · agent turn)
                 →  optional /setup lifecycle map (residual-honest · dual_write OFF):
                      init → preflight → portal HITL → reload
                      → pull (opt-in) → analyze (opt-in) → drift (report)
                      → repair plan · repair apply --yes (safe steps only)
```

This is a **runbook-style example**, not a product claim that every step is automatic or GA-green.  
**Honesty on the setup map:** PASS / pull / analyze / drift OK / repair apply **≠ invent Connected** · not Memory GA · portal HITL still human · no auto-repair without explicit `apply --yes`. See [setup-lifecycle.md](./setup-lifecycle.md).

### Phase → 7-stage SSOT mapping (s1554)

| Demo phase (this file) | Edge journey stage | Notes |
|------------------------|--------------------|--------|
| Phase 0 — Signup | **1 Signup** | Optional for pure local memory |
| Phase 1 — Install TUI (+ LLM keys) | **2 Download TUI** · **3 TUI auth/keys** | Keys/Ollama in Phase 1 body |
| Optional `/setup` map (intro + lifecycle doc) | **4 Setup wizard** | Full detail: [setup-lifecycle.md](./setup-lifecycle.md) |
| Phase 2 — Integrations | **5 Connectors / events on mesh** | list/plan + portal HITL · catalog ≠ Connected |
| Phase 3 — Local memory install · Phase 4 — Attach | **6 Local store** | host not auto on signup · dual_write OFF |
| Phase 5 — Show usage (+ digest / analyze) | **7 Analyze** | `/memory` · digest · optional mesh pull Ops Pack ~$119 |

---

## Honesty hard locks (read first)

| Claim | Truth |
|-------|--------|
| **local-primary** | Customer-edge FS palace via MCP host + kernel · not freemium hosted palace |
| **dual_write** | **OFF** by default · optional mesh audit only · not primary palace |
| **not Memory GA** | Public OSS edge + TUI attach ≠ invent bare Memory GA |
| **Edge Memory GA candidacy only** | Residual candidacy · **PASS ≠ invent Edge Memory GA declared** |
| **E10 Open** | Founder/GTM sign-off remains open · tip ≠ invent E10 closed |
| **aion broker private** | Cloud control plane stays private · not OSS edge pack · product host is `iomesh-memory-mcp` only |
| **integrations ≠ install APPLY** | MCP `list` / `plan` + portal deep links · human finishes OAuth/install in browser |
| **memory install ≠ fully automatic** | TUI does **not** auto-download/start the MCP host on signup · operator installs host |
| **kernel “automatic?”** | Kernel is a **library dep** of `iomesh-memory-mcp` at build/install · no separate palace product install for binary path · still not “signup auto-provisions Memory” |
| **catalog ≠ Connected** | Catalog status chips / plan URLs ≠ org install Connected / INSTALL_STORE green |
| **attach dogfood ≠ forever-green** | E4 stamp (s1508) is one observed residual · not continuous product dogfood green |
| **book-demo OFF** | No invent book-a-demo install path |
| **mesh optional** | Mesh / Ops Pack (~$119 language if any) is **pull/retain/audit/support** only · not required for local-primary Memory |
| **rates honesty** | Mesh base footprint ~$88 · Memory Ops Pack ~$119 when platform SKUs are mentioned · not cloud GPU palace |

Deep architecture: [memory-mcp.md](./memory-mcp.md) · integrations: [agent-integrations-setup.md](./agent-integrations-setup.md) · MCP client: [mcp.md](./mcp.md) · E4 stamp: [EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md](../EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md).

---

## What you will have at the end

| Layer | Artifact | Role |
|-------|----------|------|
| **TUI** | `iomesh` binary | Agent harness · MCP client · `/memory` + `/integrations` |
| **MCP host** | `iomesh-memory-mcp` | Exposes kernel over stdio or streamable HTTP |
| **Kernel** | `github.com/iome-sh/memory` | Local PalaceStore (FS under palace root) |
| **Palace data** | e.g. `./data/memory-palaces/<tenant>/` | Durable local memory |
| **Optional** | portal session + mesh credentials | Integrations HITL · optional `iomesh memory pull` |

```text
user / agent
    │
    ▼
iomesh-tui  ──MCP──►  iomesh-memory-mcp  ──►  memory kernel (PalaceStore)
    │                      │
    │                      └── local FS palace
    │
    ├── /integrations  → platform MCP tools (catalog/plan) + portal HITL
    └── /memory        → local MCP tools (ingest/retrieve/…)
```

---

## Phase 0 — Signup (portal · optional mesh)

**Goal:** org/workspace identity for integrations and optional mesh pull. **Not required** for pure local Memory dogfood.

1. Open **https://console.iome.sh** (or your org’s portal URL).
2. Sign up / sign in · create or join an **org** (and workspace if prompted).
3. Optionally create an API key / mesh credentials for later `iomesh memory pull` or platform MCP that lists connectors.
4. Note **org id** / tenant style strings you will use in config (`[iomesh].org`, `[memory].tenant`).

**Honesty**

| Step | Not invent |
|------|------------|
| Signup complete | ≠ Memory GA · ≠ freemium hosted palace |
| Org exists | ≠ connectors Connected · ≠ INSTALL_STORE green |
| API key minted | ≠ dual_write ON · ≠ Edge Memory GA declared |

If you only want **local Memory** (no mesh / no connectors), you may skip portal signup and jump to Phase 1 + Phase 3.

---

## Phase 1 — Install and run TUI

```bash
# From source
git clone https://github.com/iome-sh/iomesh-tui.git
cd iomesh-tui
make build          # → ./bin/iomesh

# Or latest tagged release (@latest = latest semver tag, not untagged main)
go install github.com/iome-sh/iomesh-tui/cmd/iomesh@latest
# Tip of default branch: go install github.com/iome-sh/iomesh-tui/cmd/iomesh@main
```

Set at least one LLM key (default cascade uses DeepSeek):

```bash
export DEEPSEEK_API_KEY=…
# optional: XAI_API_KEY, GEMINI_API_KEY, or pin local Ollama:
# ollama serve && ollama pull llama3.2
# ./bin/iomesh -m ollama-llama3.2
```

Smoke:

```bash
./bin/iomesh version
./bin/iomesh models
./bin/iomesh -p "Reply with ok"
```

Copy config scaffold if useful:

```bash
mkdir -p ~/.iomesh
cp configs/config.example.toml ~/.iomesh/config.toml   # edit as you go
```

---

## Phase 2 — Integrations via TUI MCP (catalog / plan · portal HITL)

**Goal:** discover connectors and plan setup from the agent/TUI path. **Install/OAuth complete only in the browser.**

### 2a. What the TUI can do (MCP tools)

When a platform MCP server exposing connector tools is attached, slash `/integrations` can:

| Subcommand | MCP tool | Meaning |
|------------|----------|---------|
| `list` | `list_connector_catalog` | Catalog inventory (status chips ≠ Connected) |
| `plan <id>` | `plan_connector_setup` | Portal URLs + next steps + honesty notes |
| `signing …` | `get_webhook_signing_headers` | Header **names** discovery only (no secret mint) |
| `status` | probe + optional list | Residual-honest operator pulse |

See [agent-integrations-setup.md](./agent-integrations-setup.md).

### 2b. Typical demo flow

```text
iomesh
  /integrations status          # MCP path · tools present · catalog honesty
  /integrations list            # catalog table
  /integrations plan github     # portal_url / portal_add_url · browser HITL only
```

Then **human** opens the portal deep link (e.g. `https://console.iome.sh/integrations/...`) and finishes OAuth / install in the browser.

Agent guidance on attach also injects residual-honest `<integrations>` notes and builtin skill `connector-integrations-setup`.

### 2c. Honesty (integrations)

| May claim | Must not claim |
|-----------|----------------|
| Catalog + plan + signing discovery | Agent MCP wrote installs |
| Browser HITL required for OAuth complete | Catalog status = Connected |
| Fail-open offline → portal URL | `list_org_connector_installs` empty = none connected |
| dual_auth candidacy open | dual-auth product live green |

**Integrations setup = list + plan + portal HITL · not full install CRUD.**

If no platform connector MCP is available in the demo environment, say so honestly and continue with **local Memory** (Phase 3+). Offline residual copy points at portal HITL — never invent catalog rows.

---

## Phase 3 — Local memory install (kernel + MCP)

For ONNX embeddings, durable palace knobs, and residual-honest Qdrant notes, see **[memory-advanced-install.md](./memory-advanced-install.md)** (s1525 · maximize benefit ladder).

### Is install automatic?

| Piece | Automatic? | What actually happens |
|-------|------------|------------------------|
| **TUI signup → memory** | **No** | Signup does not download or start a palace |
| **`iomesh-memory-mcp` host** | **No (operator step)** | `go install` / clone+build / `docker compose` |
| **Kernel `github.com/iome-sh/memory`** | **Mostly as dependency** | Pulled when building/installing the MCP host; library tip also `go get …@main` |
| **Palace directory** | **On first write** | Created under `-palace-root` / `PALACE_ROOT` when tools ingest |
| **TUI attach config** | **No** | Operator sets `[[mcp.servers]]` + `[memory]` (or opt-in Agent Plugin map) |
| **Agent Plugins sample map** | **Opt-in only** | `[plugins] enabled = true` + `dirs` — default **disabled** · map ≠ Connected |

**Product path (public · no `GOPRIVATE` / PAT):**

```bash
# 1) Product MCP host (preferred install surface)
go install github.com/iome-sh/iomesh-memory-mcp/cmd/iomesh-memory-mcp@main
# confirm:
which iomesh-memory-mcp

# 2) Kernel tip (optional explicit; already dep of host at build time)
go get github.com/iome-sh/memory@main
```

**Run host — HTTP preferred (demo / multi-client):**

```bash
mkdir -p ./data/memory-palaces
iomesh-memory-mcp \
  -palace-root ./data/memory-palaces \
  -tenant demo \
  -http-addr :8080 \
  -http-path /mcp

# other terminal:
curl -fsS http://127.0.0.1:8080/healthz
# expect residual honesty fields such as dual_write=off · not_memory_ga=true (probe only)
```

**Docker alternate (product repo):**

```bash
git clone https://github.com/iome-sh/iomesh-memory-mcp.git
cd iomesh-memory-mcp
docker compose up --build
# image: iomesh-memory-mcp:local · http://127.0.0.1:8080/mcp
curl -fsS http://127.0.0.1:8080/healthz
```

**Stdio alternate** (TUI spawns process — no separate HTTP):

```bash
# binary on PATH; TUI command transport (see Phase 4)
iomesh-memory-mcp -palace-root ./data/memory-palaces -tenant demo
```

s1517: TUI product path is **`iomesh-memory-mcp` only** (in-tree residual aion Memory sample removed) · aion broker/CP stays private.

---

## Phase 4 — Attach TUI to local Memory MCP

### 4a. Primary path: TOML `[[mcp.servers]]` (HTTP)

Add to `~/.iomesh/config.toml` (or `--config` path):

```toml
[mcp]
enabled = true

[[mcp.servers]]
name = "iomesh-memory-mcp"
url = "http://127.0.0.1:8080/mcp"
allow_loopback = true
mutating = true

[memory]
enabled = true
server = "iomesh-memory-mcp"   # must match [[mcp.servers]].name
tenant = "demo"
auto_recall = true             # inject <memory-context> fail-open
auto_ingest = false            # opt-in after successful turns
dual_write = false             # OFF · not primary palace
```

Stdio variant:

```toml
[[mcp.servers]]
name = "iomesh-memory-mcp"
command = "iomesh-memory-mcp"
args = ["-palace-root", "./data/memory-palaces", "-tenant", "demo"]
mutating = true
```

Env helpers (optional):

| Env | Effect |
|-----|--------|
| `IOMESH_MCP=1` | Enable `[mcp]` |
| `IOMESH_MEMORY=1` | Enable `[memory]` |
| `IOMESH_MEMORY_TENANT` / `MEMORY_TENANT` | Default tenant |
| `IOMESH_MEMORY_DUAL_WRITE` | Leave **unset** for local-primary |

### 4b. Opt-in sample Agent Plugin map

Product sample: [`examples/agent-plugins/iomesh-memory-mcp`](../../examples/agent-plugins/iomesh-memory-mcp/).

```toml
[plugins]
enabled = true
dirs = ["/absolute/path/to/iomesh-tui/examples/agent-plugins/iomesh-memory-mcp"]
```

**Honesty:** Discover/map success ≠ process Connected · binary still required on PATH for stdio · package load ≠ Memory GA. Prefer TOML HTTP for demos.

### 4c. Connect verify

```bash
./bin/iomesh mcp --connect
# healthy local host: connected ≥ 1 · memory_* tools listed
# lean product host often shows tools=6 (s1508 stamp family), e.g.:
#   memory_ingest_turn, memory_retrieve, memory_search_semantic,
#   memory_list, memory_compact_status, memory_facts_as_of
```

Pinned residual stamp (do not invent more): [EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md](../EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md) · `connected=1` · `tools=6` · UTC `2026-08-09T06:23:34Z`.

---

## Phase 5 — Show usage (demo script)

Run interactive TUI with the config above:

```bash
./bin/iomesh --config ~/.iomesh/config.toml
# or: ./bin/iomesh
```

### 5a. Memory status and recall

```text
/memory
/memory status
/memory recall project alpha
/memory compact-status
```

Expect residual-honest status lines (`mcp=true` when attached · `dual_write=false` · not invent Memory GA). Empty recall → honest empty · **never invent memories**.

### 5b. Ingest then recall (show the loop)

```text
/memory ingest Demo note: Project alpha ships Friday; owner is Alice.
/memory recall alpha
# folder overlay (no session_id required):
/memory ingest-dir ./notes --dry-run
/memory ingest-dir ./notes
# CLI equivalent:
# iomesh memory ingest --yes "Demo note: Project alpha ships Friday; owner is Alice."
# iomesh memory ingest-dir --dry-run ./notes
# iomesh memory ingest-dir --yes ./notes
```

`session_id` is minted as `local-overlay` when the walk has no TUI/config session (`iomesh-memory-mcp` v0.1.0 requires it on `memory_ingest_turn`). `/memory recall` without a session_id stays unfiltered and finds those private overlay entries. Catalog list ≠ consume. dual_write OFF.

With `auto_ingest = true` (opt-in), successful agent turns also write user/assistant turns via MCP `memory_ingest_turn` (mutating; interactive MCP tools still approval-gated unless `--yolo`).

### 5c. Optional advanced surfaces (when tools present)

Lean product host may expose a **subset** of platform residual tools. Only call what `iomesh mcp --connect` listed:

```text
/memory semantic project alpha
/memory facts-as-of --as-of 2026-08-01T00:00:00Z
/memory timeline --limit 5
```

Platform residual extras (related multi-hop, supersede HITL, patterns, trigger-compact, …) require matching MCP tools — **PASS ≠ invent full platform sidecar parity**. See [memory-mcp.md](./memory-mcp.md) slash table.

### 5d. Agent turn with auto-recall

```text
# with auto_recall=true and MCP up:
What do we remember about project alpha?
```

Runtime injects `<memory-context>` when hits exist (fail-open if MCP down). Advanced guidance notes (`<memory-advanced>`, integrations) may appear on attach — playbook only.

### 5e. Integrations pulse (if platform MCP attached)

```text
/integrations status
/integrations list
/integrations plan github
```

Then open portal URL in browser for HITL install. **Do not** claim Connected from catalog alone.

### 5f. Optional mesh pull into local palace

Requires `[iomesh]` credentials + Memory Ops Pack entitlement language when commercial SKUs apply:

```bash
# Terminal A: local MCP host running
# Terminal B (CLI still valid):
iomesh memory pull --stream EVENTS --name tui-local-palace --once --dry-run
# map-only dry-run; drop --dry-run to ingest via MCP memory_ingest_turn
```

In-session opt-in (s1530 P5 residual-honest · after mesh + `pull_consumer` configured):

```text
/setup pull status
/setup pull once
/setup pull start
/setup pull stop
```

Or set `[memory] pull_continuous = true` (setup fragment default **false**). Mesh is **pull egress** into local palace · dual_write stays OFF · pull ≠ invent Connected · hosted Palace sunset · CLI `iomesh memory pull` still valid.

In-session analyze ticks + drift + guided repair (s1534 P6 + s1538 P7 residual-honest · opt-in · explicit apply):

```text
/setup analyze status
/setup analyze once --mode status
/setup analyze start --mode digest --window day --interval 300
/setup analyze stop
/setup drift
/setup maintain
/setup repair
/setup repair plan
/setup repair apply --yes
```

Or set `[memory] analyze_continuous = true` (setup fragment default **false**). **`/memory digest` still valid** as one-shot ops pulse · analyze tick ≠ invent Connected · drift report ≠ invent install green · package wire ≠ Connected · dual_write OFF · not Memory GA · **guided repair** plans from drift · `apply --yes` safe steps only (`reload_mcp` · `start_pull` · `start_analyze`) · refuse without `--yes` · repair apply ≠ invent Connected · portal HITL still human · dual_write never auto-flipped ON · notes for human host/mesh remain manual.

### 5g. Onboard residual lanes (no live dial)

Offline residual boards (docs-shaped, no invent green):

```text
/onboard next memory
/onboard next agentic
/onboard next memory-pull
/onboard next setup
/onboard next demo
```

Useful for demo hosts who need honesty boards without claiming live APPLY. Companion setup map: `/onboard next setup` (s1542 · P1–P7 closeout residual · dual_write OFF · never invent Connected).

---

## Minimal copy-paste demo (local only · ~5 minutes)

```bash
# A) TUI
git clone https://github.com/iome-sh/iomesh-tui.git && cd iomesh-tui && make build
export DEEPSEEK_API_KEY=…   # or use -m ollama-llama3.2

# B) Memory host (other shell)
go install github.com/iome-sh/iomesh-memory-mcp/cmd/iomesh-memory-mcp@main
mkdir -p /tmp/demo-palace
iomesh-memory-mcp -palace-root /tmp/demo-palace -tenant demo -http-addr :8080 -http-path /mcp

# C) Config
cat > /tmp/demo-iomesh.toml <<'EOF'
[mcp]
enabled = true
[[mcp.servers]]
name = "iomesh-memory-mcp"
url = "http://127.0.0.1:8080/mcp"
allow_loopback = true
mutating = true

[memory]
enabled = true
server = "iomesh-memory-mcp"
tenant = "demo"
auto_recall = true
auto_ingest = false
dual_write = false
EOF

# D) Show usage
curl -fsS http://127.0.0.1:8080/healthz
./bin/iomesh mcp --config /tmp/demo-iomesh.toml --connect
./bin/iomesh --config /tmp/demo-iomesh.toml
# then: /memory · /memory ingest … · /memory recall …
```

---

## Failure modes (fail-open · residual)

| Symptom | Likely cause | Honest next step |
|---------|--------------|------------------|
| `mcp --connect` connected=0 | Host not running / wrong URL | Start host · check `curl healthz` · fix TOML `url` |
| Tools missing | Lean host surface or wrong server | List tools from connect · do not invent platform-only tools |
| `/memory` mcp=false | `[memory].server` name mismatch | Match `[[mcp.servers]].name` |
| Empty recall | No ingest yet / wrong tenant | Ingest under same `tenant` · empty ≠ invent |
| Integrations offline | No platform connector MCP | Portal HITL URL · no invent catalog rows |
| Docker image pull fails | Local tag only | Build with `docker compose up --build` · no invent public registry GA |
| dual_write “needed” for demo | Misconception | Local palace is primary · dual_write OFF |

---

## Related surfaces

| Doc / path | Why |
|------------|-----|
| [edge-user-journey.md](./edge-user-journey.md) | **SSOT** 7-stage edge-first narrative (s1554) |
| [setup-lifecycle.md](./setup-lifecycle.md) | Stage 4 setup wizard P1–P7 residual map |
| [memory-mcp.md](./memory-mcp.md) | Full Memory phases, slash table, edge OSS honesty |
| [agent-integrations-setup.md](./agent-integrations-setup.md) | `/integrations` MCP tools + portal HITL |
| [mcp.md](./mcp.md) | MCP client transports + `iomesh mcp --connect` |
| [EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md](../EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md) | Pinned E4 attach stamp |
| [examples/agent-plugins/iomesh-memory-mcp](../../examples/agent-plugins/iomesh-memory-mcp/) | Product sample plugin map |
| Peer [iomesh-memory-mcp](https://github.com/iome-sh/iomesh-memory-mcp) | Product host README · compose · tools |
| Peer [memory](https://github.com/iome-sh/memory) | Kernel library API |

---

## Non-goals

- Auto-provision Memory on portal signup  
- Invent install Connected / INSTALL_STORE green from catalog  
- Invent Edge Memory GA declared / E10 closed / bare Memory GA  
- dual_write ON as primary path  
- Freemium multi-tenant hosted palace  
- Bundle aion private broker into OSS edge  
- Claim forever-green live dogfood from one attach stamp  

**Demo success** = operator can **signup (optional)** · **list/plan integrations with HITL honesty** · **install local kernel+MCP host (not fully automatic)** · **attach TUI** · **show `/memory` usage** — residual-honest throughout.
