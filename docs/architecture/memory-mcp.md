# Memory Palace + temporal MCP

First-class **Agentic Memory Palace** and **temporal recall** for `iomesh-tui`, without embedding Palace inside the TUI process.

Product edge ships **`iomesh-memory-mcp`** (stdio **and** streamable HTTP; public host for `github.com/iome-sh/memory`) with tools:

| Tool | Purpose |
|------|---------|
| `memory_ingest_turn` | Persist a conversation turn (tiered Palace) |
| `memory_retrieve` | Query memories (optional `session_id`, `since`/`until`, `session_seq`) |
| `memory_related` | Multi-hop lite related recall (`seed_entity` / `query` / `max_hops`; optional `prefer_shorter_hops` omit=true · s1135 + s1281 / aion s1277) |
| `ops_digest_export` | Ops heartbeat digest export (`window` / `horizon` / `limit`; s1200 opt-in; MCP + HTTP) |
| `memory_facts_as_of` | Bi-temporal lite validity listing (`as_of` required RFC3339; optional `entity` / `query` / `session_id` / `limit`; s1276 opt-in; **MCP-first** — no lean HTTP invent) |
| `memory_supersede_entity` | A3 lite entity supersession (`entity` required; optional `as_of`; **HITL** · s1282 / aion s640; **MCP-first** — no lean HTTP invent) |
| `memory_patterns_list` | Ops pulse Beta pattern list (shipped s1287 · MCP; no lean HTTP invent) |
| `memory_anomalies_list` | Ops pulse Beta anomaly list (shipped s1287 · MCP; no lean HTTP invent) |
| `memory_timeline` | Temporal timeline slice (s1296 slash: `/memory timeline`; MCP-first) |
| `memory_compact_status` | Palace tier counts + last compaction (s1296 slash: `/memory compact-status`; **read-only**) |
| `memory_search_semantic` | Tier-4 semantic facts (s1301 slash: `/memory semantic`; MCP-first) |
| `memory_ingest_event` | Ops/telemetry event ingest (s1301 slash: `/memory ingest-event`; s138 T1; **not** conversation turn) |
| `memory_trigger_compact` | Mutating RecMem compaction advisory — **HITL wired** (s1311 slash: `/memory trigger-compact --i-confirm`) |
| compact / other ops helpers | Residual ops helpers (not product Memory GA) |

Resources: `memory://{tenant}/…` (stats, timeline, session turns, facts).

## Phases

| Phase | Status | Work |
|-------|--------|------|
| **0** | **done** | Attach product Memory MCP (`iomesh-memory-mcp`) via existing MCP client; documented example |
| **1** | **done** | `[memory]` auto-recall inject, opt-in auto-ingest, `/memory` slash |
| **2** | **done (v0.3.0)** | HTTP MCP primary path + optional dual-write to mesh `MEMORY_INGEST` |
| **3 partial** | **done (dogfood)** | Async `MEMORY_RPC` recall probe (`PublishMemoryRecall`) |
| **3** | **done (v0.4.0 dogfood)** | Sync `RetrieveMemory` → `POST /v1/memory/retrieve` (+ `/v5` fallback) + dogfood `memory_retrieve` step |
| **3+** | **done (v0.4.0 agent)** | Agent auto-recall + `/memory recall` prefer sync HTTP when mesh **or** `[memory].endpoint` sidecar is set; MCP fallback |
| **3 temporal (s1068)** | **done** | Sync retrieve options: `since`/`until`/`session_seq` on lean HTTP + auto-recall config + `/memory recall` flags |
| **3 related multi-hop (s1135)** | **done (opt-in)** | Sync `POST /v1|/v5/memory/related` + MCP `memory_related` fallback; `/memory related`; default auto-recall stays single-hop |
| **3 related hop ranking (s1281)** | **done (opt-in)** | `prefer_shorter_hops` on HTTP + MCP `memory_related` (omit = kernel default **true**; `--legacy-sort` → false); aion s1277 parity; hop ranking path-aware lite · multi-hop lite ≠ graph RAG |
| **3 ops digest (s1200)** | **done (opt-in)** | Sync `POST /v1|/v5/memory/ops_digest` + MCP `ops_digest_export` fallback; `/memory digest`; ops GA-path · knowledge/analytical Beta |
| **3 facts-as-of (s1276)** | **done (opt-in · MCP-first)** | MCP `memory_facts_as_of` (aion Beta K4 lite); `/memory facts-as-of`; bi-temporal lite · not full dual-clock Graphiti · no lean HTTP route today |
| **3 supersede HITL (s1282)** | **done (opt-in · MCP-first · HITL)** | MCP `memory_supersede_entity` (aion A3 lite / s640); `/memory supersede --entity … --i-confirm`; mutating closes `valid_until`; not NLP contradiction · no lean HTTP invent |
| **3 patterns/anomalies (s1287)** | **done (opt-in · MCP ops pulse Beta)** | `/memory patterns|anomalies` + MCP `memory_patterns_list` / `memory_anomalies_list` when present; ops pulse Beta · not medical · no invent GA window engine · no lean HTTP invent |
| **3 advanced agent skill (s1288)** | **done (docs + builtin skill)** | Builtin skill `memory-advanced-agent` residual-honest playbook for advanced surfaces; docs inventory lock; skill-only (no product path invent) |
| **3 advanced agent system note (s1291)** | **done (AttachMCP inject)** | Residual-honest `<memory-advanced>` system note (`MemoryAdvancedAgentGuidanceNote`) injected on `AttachMCP` (mirror integrations s1251); steers opt-in advanced memory locks |
| **3 timeline + compact-status (s1296)** | **done (opt-in · MCP-first · read-only compact)** | `/memory timeline` → MCP `memory_timeline`; `/memory compact-status` → MCP `memory_compact_status`; temporal timeline · filters before limit · Palace tier counts residual · not Memory GA · dual_write OFF · mutating compact deferred to s1311 HITL · no lean HTTP invent |
| **3 semantic + ingest-event (s1301)** | **done (opt-in · MCP-first)** | `/memory semantic` → MCP `memory_search_semantic` (tier-4 semantic facts residual · empty ≠ invent); `/memory ingest-event` → MCP `memory_ingest_event` (s138 T1 temporal event telemetry · not conversation turn · never invent memory_id); not Memory GA · dual_write OFF · no lean HTTP invent |
| **3 local-edge Docker attach docs (s1308)** | **done (docs only)** | Operator/agent docs for attaching **local-edge Docker** Memory MCP streamable HTTP (prefer product `iomesh-memory-mcp` compose; peer aion **s1306** private compose residual + **s1307**); TUI `[[mcp.servers]]` URL attach · dual_write OFF · local-primary · hosted Palace sunset · **not** Memory GA · docker edge ≠ invent GA · no public registry tag claim |
| **3 edge OSS Option A install honesty (s1453)** | **done (docs + onboard memory lane)** | Residual-honest edge install story: product MCP host **`iomesh-memory-mcp`** · kernel `github.com/iome-sh/memory` · aion broker **private** · dual_write OFF · not Memory GA · Palace sunset · mesh optional for pull · **OSS path ≠ invent public flip complete** |
| **3 M2 lean host attach tip (s1458)** | **done (docs + onboard memory lane)** | Residual-honest **M2 lean edge host** attach when built from `github.com/iome-sh/iomesh-memory-mcp`: go run/build · streamable HTTP `http://127.0.0.1:8080/mcp` or stdio · dual_write OFF · not Memory GA · aion broker private · scaffold/M2 · tool parity may be lean vs platform residual · **PASS ≠ invent full platform sidecar parity** · keeps Option A honesty |
| **3 M3 edge dogfood tip (s1463)** | **done (docs + onboard memory lane)** | Residual-honest **M3 edge dogfood** TUI↔`iomesh-memory-mcp` path: build/run from product repo · `docker compose up --build` → image `iomesh-memory-mcp:local` · attach `http://127.0.0.1:8080/mcp` · healthz · stdio alternate · peer mcp `make edge-dogfood-gate` (mention only) · dual_write OFF · not Memory GA · **offline dogfood tip ≠ invent live dogfood as green** · **PASS ≠ invent full platform sidecar parity** · residual PASS ≠ invent public flip (edge packs public as of s1478) · M3 after M2 · M4 later deliberate |
| **3 M4 public flip readiness tip (s1469)** | **done (docs + onboard memory lane)** | Residual-honest **M4 public flip readiness** tip history: order kernel `github.com/iome-sh/memory` **first** · then `iomesh-memory-mcp` · readiness docs/gates in those repos (mention only) · **readiness tip ≠ invent public flip complete** · residual PASS ≠ invent public flip · dual_write OFF · not Memory GA · aion broker private · M5 signing later after flip · **s1478 supersedes operator tip** (edge packs public) · keeps Option A + M2 + M3 honesty |
| **3 public product attach (s1478)** | **done (docs + onboard memory lane + product sample plugin)** | Residual-honest **post-public** product edge attach: both `github.com/iome-sh/memory` + `github.com/iome-sh/iomesh-memory-mcp` **PUBLIC** · `go install …@main` / `go get …@main` · **no GOPRIVATE** · HTTP `http://127.0.0.1:8080/mcp` or stdio · docker compose still valid · dual_write OFF · not Memory GA · aion broker **still private** · flip complete residual ≠ invent Memory GA · public OSS ≠ invent platform GA · aion broker private · s1517 product-only in-tree sample |
| **3 E4 MCP client attach dogfood (s1508)** | **done (docs + evidence stamp + onboard memory lane tip)** | Residual-honest **E4 full MCP client attach** dogfood: lean host HTTP → TUI `iomesh mcp --connect` **connected=1 · tools=6** (observed stamp) · dual_write OFF · local-primary · **Edge Memory GA candidacy only** · residual PASS ≠ invent Edge Memory GA declared · not bare Memory GA · not hosted Memory GA · aion broker private · **E10 Open** · tip ≠ invent forever-green product dogfood · evidence [EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md](../EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md) |
| **3 memory edge usage demo (s1513)** | **done (docs only)** | Residual-honest **utilization/demo example**: signup (optional) → TUI MCP integrations list/plan + portal HITL → local memory install (kernel + `iomesh-memory-mcp`; **not fully automatic**) → attach → show `/memory` + `mcp --connect` usage · dual_write OFF · not Memory GA · Edge Memory GA candidacy only · E10 Open · catalog ≠ Connected · aion broker private · walkthrough [memory-edge-usage-demo.md](./memory-edge-usage-demo.md) |
| **3 trigger-compact HITL + advanced status (s1311)** | **done (opt-in · MCP-first · HITL)** | `/memory trigger-compact --i-confirm` → MCP `memory_trigger_compact` (RecMem advisory · mutating HITL · refuse without confirm); `/memory status` prints `MemoryStatusLine` + `MemoryAdvancedStatus` residual inventory (related · facts-as-of · supersede · timeline · compact-status · semantic · ingest-event · patterns · anomalies · digest · trigger-compact); dual_write OFF · not Memory GA · not invent compaction green · no lean HTTP invent |
| **4 pull (s652)** | **done (M1)** | `iomesh memory pull` — durable mesh consumer → local MCP `memory_ingest_turn` (cost-max local palace; dual_write remains optional audit) |

**Related (not Memory Palace):** agent connector setup slash `/integrations` (s1238/s1242/s1243) uses MCP `list_connector_catalog` / `plan_connector_setup` (aion v178) + `get_webhook_signing_headers` (v30) with residual honesty — see [agent-integrations-setup.md](./agent-integrations-setup.md).

**Cost-max (s650+):** primary Memory UX is **local Palace** (this TUI + product `iomesh-memory-mcp`). Mesh is **pull egress** of ops events; hosted cloud Palace is **sunset until scale**. Dual-write = optional **audit** only (default OFF).

### Local edge stack (s761 product · s765 completeness pin)

End-to-end **cost-max** path (Beta · local only · not platform GPU · not invent GA):

```text
I/O Mesh cloud (pull egress)  ↔  iomesh-tui  ↔  local memory MCP (Palace)  ↔  local AI (Ollama pin)
```

| Layer | Role | Honesty |
|-------|------|---------|
| Mesh pull | Ops event egress into local palace (`iomesh memory pull`) | Cloud mesh ≠ local AI |
| Local MCP Palace | Primary memory UX (`iomesh-memory-mcp`) | Hosted Palace sunset until scale |
| Ollama pin | Local LLM (`-m ollama-llama3.2` / `IOMESH_DEFAULT_MODEL`) | OpenAI-compat `/v1` only; $0 catalog tier; not cascade default |
| Dual-write | Optional mesh audit | **Default OFF** |

See [llm-cascade.md](./llm-cascade.md) for Ollama install (`ollama serve` / `ollama pull llama3.2`) and env overrides (`OLLAMA_URL` / `OLLAMA_HOST`).

**s765 (Beta · completeness pin):** local edge stack **complete** after s761 first-class Ollama product Shipped — inventory (mesh pull egress ↔ TUI ↔ local memory MCP ↔ Ollama pin) locked by docs + unit tests (`TestDefaultModels_OllamaLocalEdgeCompletenessPin`: catalog name `ollama-llama3.2`, model id `llama3.2`, cost 0, caps `local`+`ollama`, priority after DeepSeek cascade / Premium). Completeness pin **does not** invent new catalog models and **does not** re-claim s761 product body. Peer aion **s764** heartbeat-local-edge-path residual. Local AI ≠ platform GPU · dual_write OFF · Palace sunset · offline unit ≠ live Ollama green · catalog pin ≠ cascade default · not full mesh RBAC GA · no invent GA.

#### Local-primary LT honesty (s768 pin)

Honesty pin so TUI surfaces agree with the **cost-max local-primary** charter after s761/s765 product bodies (this serial is docs/tests only — **does not** re-claim s761 Ollama product or s765 completeness inventory):

| Claim | Honesty |
|-------|---------|
| **Long-term / temporal memory** | Customer **local palace** — local MCP (`iomesh-memory-mcp`) + optional local AI (Ollama pin). Not hosted cloud GPU palace. |
| **Mesh heartbeats** | **Pull egress** into local palace (`iomesh memory pull`) — not a push-to-cloud-palace product path. |
| **dual_write** | Optional mesh audit only · **default OFF** (`[memory].dual_write` / `IOMESH_MEMORY_DUAL_WRITE`). |
| **Optional mesh pull** | Mesh credentials + platform endpoint can pull into local palace — **not** hosted cloud GPU palace. Do not invent a priced add-on SKU here. |
| **Local AI** | Customer-edge Ollama only · **≠ platform GPU**. |
| **Hosted Palace** | **Sunset** until scale. |
| **Does not invent** | GA · freemium unlimited palace · platform GPU · full mesh RBAC GA. |

Peer aion **s767** bi-heartbeats local-primary honesty continuum. Unit pin: `TestDefaultMemoryConfig` / config `Default().Memory.DualWrite == false` (s768). Offline unit ≠ live APPLY · Beta · no invent GA.

#### Naming honesty (s771 pin)

Docs **naming honesty** residual after aion **s770** usage-rates honesty (this serial is docs/tests only — **does not** re-claim s761 Ollama product, s765 local-edge completeness, or s768 local-primary LT pin bodies):

| Name / phrase | Honest meaning in this TUI |
|---------------|----------------------------|
| **“Memory Palace”** | **Local-primary customer-edge palace** — MCP (`iomesh-memory-mcp`) + mesh **pull** into local store. **Not** multi-tenant hosted cloud GPU / Qdrant Palace fleet. |
| **Optional mesh pull** | Pull / retain into local palace when mesh is configured. **Not** hosted GPU palace fleet. Do not invent a priced add-on SKU here. |
| **dual_write** | Optional mesh audit only · **default OFF**. |
| **Hosted Palace** | **Sunset** until scale. |
| **Local AI** | Customer-edge Ollama · **≠ platform GPU**. |
| **Does not invent** | GA · freemium unlimited palace · platform GPU · full mesh RBAC GA. |

Peer aion **s770** usage-rates residual · prior TUI **s768** local-primary LT honesty. dual_write OFF · Palace sunset · no invent GA · offline unit ≠ live APPLY · Beta.

#### Buyer claim pin (s774)

OSS / buyer-claim boundary pin peer aion **s773** cost-max claim suite expand (this serial is docs/tests only — **does not** re-claim s761 Ollama product, s765 local-edge completeness, s768 local-primary LT, or s771 naming honesty bodies):

| Surface | May claim | Must not claim |
|---------|-----------|----------------|
| **Public MIT `iomesh-tui`** | OSS agent harness + optional mesh **client** surface (SDK elsewhere) | Hosted multi-tenant mesh **control plane** shipped inside free TUI |
| **Memory** | Local palace via MCP (`iomesh-memory-mcp`); mesh **pull** into local store | Multi-tenant hosted / freemium cloud palace included with free TUI |
| **Optional mesh pull** | Mesh pull into local palace — **optional** · **not first-run required** (s1695) | Cloud GPU / hosted Palace fleet as a first-run product · freemium hosted palace · pay-to-get-memory |
| **Local AI** | Customer-edge Ollama pin | Platform GPU from Ollama pin |
| **dual_write** | Optional mesh audit · **default OFF** | dual_write as primary palace path |
| **Hosted Palace** | Sunset until scale | Invent GA · freemium unlimited palace · full mesh RBAC GA |
| **First-run (s1695)** | TUI OSS + `iomesh-memory-mcp` local-primary only · mesh optional | Mesh required for first-run · pay-to-get-memory invent |

Peer aion **s773** cost-max claim suite expand · prior TUI **s771** naming · **s768** local-primary · **s1695** mesh not first-run required. dual_write OFF · Palace sunset · local AI ≠ platform GPU · no invent GA · offline unit ≠ live APPLY · Beta. Unit pin peer: `TestDefault_DualWriteOff` / `TestDefaultMemoryConfig_DualWriteOff` (s768 body + s771/s774 comment peers).

**Non-goals:** private monorepo imports in public TUI; embedding Qdrant/Palace in-process; dependency on `iomesh-client-sdk-go` or `iomesh-client-sdk-python` (TUI does not package either SDK).

## Public client SDKs (Go + Python)

Operators who need the **full** mesh client surface (beyond this TUI’s lean HTTP/MCP path) should use the official public SDKs — **not** invent a third client inside the TUI:

| SDK | Repo | Notes |
|-----|------|--------|
| **Go** | **[iomesh-client-sdk-go](https://github.com/iome-sh/iomesh-client-sdk-go)** | Official MIT Go module · full client surface for services / stage-gate jobs |
| **Python** | **[iomesh-client-sdk-python](https://github.com/iome-sh/iomesh-client-sdk-python)** | Official MIT Python client (`iomeshclient`) · **Beta** · stdlib `urllib` · wire parity with Go for services/agents that need mesh I/O outside the lean TUI · tip **v0.10.x** · **v0.10 ≠ invent 1.0** · **GitHub release ≠ invent PyPI green** |

### Go capability map (vs lean TUI)

| Capability | In the Go SDK | In iomesh-tui |
|------------|---------------|---------------|
| **M2** sync retrieve | `RetrieveMemory` / memory helpers | Lean `POST /v1/memory/retrieve` (+ `/v5`) in `internal/iomesh` |
| **M3** temporal envelope | Full temporal fields on publish | Dual-write mirrors a subset (`event_time`, `session_seq`, …) |
| Multi-tenant workspace | `WithWorkspace` (and related options) | Optional org/workspace headers when configured |

**iomesh-tui stays lean:** no `github.com/iome-sh/iomesh-client-sdk-go` module dependency and **no** Python SDK packaging inside TUI. Memory dual-write and sync retrieve mirror SDK wire shapes over plain HTTP so the agent harness remains a thin, zero-SDK client. Prefer the public SDKs for custom services (Go or Python), stage gate jobs, or anything that should track the full client API; keep the TUI lean HTTP/MCP path for the agent harness.

## Phase 0–1 — MCP hooks (stdio or HTTP)

### Preferred: streamable HTTP (platform M1)

```bash
iomesh-memory-mcp -http-addr :8080 -palace-root /data/memory-palaces
# MCP endpoint: http://127.0.0.1:8080/mcp
# health:       http://127.0.0.1:8080/healthz
```

```toml
[mcp]
enabled = true

[[mcp.servers]]
name = "memory"
url = "http://127.0.0.1:8080/mcp"
allow_loopback = true
mutating = true

[memory]
enabled = true
server = "memory"          # must match [[mcp.servers]].name
tenant = "dept.research"   # or MEMORY_TENANT / IOMESH_MEMORY_TENANT
auto_recall = true         # inject <memory-context> each turn (fail-open)
auto_ingest = false        # opt-in: write user+assistant turns after success
# dual_write = false       # optional mesh audit only (needs [iomesh]); not primary palace
# pull_stream = "EVENTS"   # iomesh memory pull (s652)
# pull_consumer = "tui-local-palace"
# pull_filter = ""
# pull_batch = 8
# pull_max_wait_ms = 2000
# pull_role = ""           # optional X-IOMesh-Role (s675 Beta; fail-open empty)
# pull_allow_suffix = ""   # optional X-IOMesh-Pull-Allow-Suffix for role=custom
# limit = 8
# max_snippet_bytes = 6000
# Optional time-window for auto-recall + default /memory recall (s1068; RFC3339):
# recall_since = "2026-07-01T00:00:00Z"
# recall_until = "2026-07-31T23:59:59Z"
# recall_session_seq = 0   # omit when 0; maps to body session_seq
```

### Local-edge Docker Memory MCP (s1308 · product host preferred · s1517)

**Product local-primary** path when you prefer a containerized palace over a host binary: run **`iomesh-memory-mcp`** via Docker Compose (public product repo), then attach this TUI over streamable HTTP.

#### 1. Start Docker edge (product repo · preferred)

```bash
git clone https://github.com/iome-sh/iomesh-memory-mcp.git
cd iomesh-memory-mcp
docker compose up --build
# image: iomesh-memory-mcp:local · MCP: http://127.0.0.1:8080/mcp · healthz: /healthz
curl -fsS http://127.0.0.1:8080/healthz
```

| Artifact (product) | Notes |
|--------------------|--------|
| Compose | product repo `docker-compose.yml` |
| Image (local build) | `iomesh-memory-mcp:local` — local build only · not invent public registry GA |
| Host bind | `8080:8080` → MCP path `/mcp`, health `GET /healthz` |
| Palace volume | under compose bind / volume as documented in product repo |

**Private residual (optional):** aion monorepo may still ship private local-edge compose (historical peer **s1306** / residual **s1307**) — not product naming, not required for public OSS attach, and not an in-tree TUI sample (s1517). dual_write OFF · not Memory GA · aion broker private.

#### 2. Attach TUI (streamable HTTP MCP)

Same keys as preferred HTTP / `configs/config.example.toml` — URL transport wins over `command`:

```toml
[mcp]
enabled = true
# inject_iomesh_context = false   # s1267 opt-in multi-tenant headers; default false

[[mcp.servers]]
name = "memory"
url = "http://127.0.0.1:8080/mcp"   # Docker edge MCP (streamable HTTP)
allow_loopback = true
mutating = true

[memory]
enabled = true
server = "memory"          # must match [[mcp.servers]].name
tenant = "dept.research"   # or MEMORY_TENANT / IOMESH_MEMORY_TENANT
auto_recall = true
auto_ingest = false
# dual_write = false       # optional mesh audit only · default OFF · not primary palace
```

Env helpers (optional; same as binary path):

| Env | Effect |
|-----|--------|
| `IOMESH_MCP=1` | Enable `[mcp]` section |
| `IOMESH_MEMORY=1` | Enable `[memory]` hooks |
| `IOMESH_MEMORY_TENANT` / `MEMORY_TENANT` | Default tenant |
| `IOMESH_MEMORY_DUAL_WRITE=1` | Opt-in mesh audit only — **leave unset** for local-primary |

There is **no** separate `MEMORY_MCP_URL` config key in the TUI — point `[[mcp.servers]].url` at the edge. (aion Grok wire scripts may use `MEMORY_MCP_URL` / `AION_MEMORY_MCP_URL` for other hosts; TUI uses TOML `url`.)

#### 3. Verify

```bash
curl -fsS http://127.0.0.1:8080/healthz
iomesh mcp --connect
# expect memory_* tools under server "memory"

iomesh
# /memory                 → status (mcp=true when attached)
# /memory recall <query>  → MCP memory_retrieve when no sync HTTP sidecar
```

#### Honesty (s1308 hard locks)

| Claim | Truth |
|-------|--------|
| **dual_write** | **OFF** by default — optional mesh audit only; not primary palace |
| **local-primary** | Customer-edge FS palace via local Docker MCP — primary Memory UX |
| **hosted Palace** | **Sunset** until scale — Docker edge is **not** freemium hosted Qdrant fleet |
| **not Memory GA** | Local MCP attach ≠ product Memory GA · multi-tenant cloud palace · full graph RAG |
| **docker edge ≠ invent GA** | Containerizing product Memory MCP does **not** invent hosted Memory GA, platform GPU, or public registry GA |
| **image tag** | product local image `iomesh-memory-mcp:local` — do not claim public Hub/GHCR pull GA |
| **TUI tree** | Does **not** vendor aion compose/Dockerfiles — operators use aion monorepo for `docker compose` |
| **Peer residual** | aion **s1306** compose body · aion residual **s1307** · TUI docs **s1308** |

See also [Local edge stack (s761/s765)](#local-edge-stack-s761-product--s765-completeness-pin) · [Local-primary LT honesty (s768)](#local-primary-lt-honesty-s768-pin) · [Edge OSS Option A (s1453)](#edge-oss-option-a-s1453) · [mcp.md](./mcp.md) streamable HTTP.

#### Edge OSS Option A (s1453) · M2 lean attach (s1458) · M3 edge dogfood (s1463) · M4 public flip readiness (s1469) · s1478 public product attach · E4 client attach dogfood (s1508)

Residual-honest **edge Memory OSS install path** for operators (docs + `/onboard next memory` lane honesty — **does not** invent Memory GA, freemium palace, live dogfood green, Connected green, or **Edge Memory GA declared**). **s1478:** both product edge repos are **public**. **s1508:** E4 full MCP client attach dogfood evidence stamp (local-primary · dual_write OFF · **Edge Memory GA candidacy only** · **E10 Open**).

| Layer | Role | Honesty |
|-------|------|---------|
| **iomesh-tui** | Public MIT agent harness | Already public · local-primary client |
| **Memory MCP host** | Product name **`iomesh-memory-mcp`** only (public · go install / compose · s1517) | aion broker/CP private · no in-tree residual aion Memory sample |
| **Kernel** | `github.com/iome-sh/memory` | **Public** (s1478) · `go get …@main` · **no GOPRIVATE** |
| **Local palace** | FS palace via MCP (`-palace-root` / docker edge) | Hosted Palace **sunset** · dual_write **OFF** |
| **aion broker / CP** | Private cloud control plane | **Stays private** · not OSS edge pack |

**Local-primary stack (what end users install for Memory):**

```text
iomesh-tui (public MIT)  +  Memory MCP (iomesh-memory-mcp product name)  +  github.com/iome-sh/memory kernel  +  local palace
```

Optional mesh feed: credentials + platform for durable `iomesh memory pull` only. Mesh is **optional for pull only**, not required for local-primary Memory. **First-run (s1695):** OSS local-primary complete **without** mesh. Do not invent a priced add-on SKU or a mesh base rate here.

**M2 lean host attach (s1458 · residual-honest · not invent flip complete / GA):**

When built from **`github.com/iome-sh/iomesh-memory-mcp`** (peer free eng M2 scaffold · mention only):

| Surface | Residual honesty |
|---------|------------------|
| **Product host** | **`iomesh-memory-mcp`** · go run / go build from that repo |
| **Attach** | Streamable HTTP `http://127.0.0.1:8080/mcp` **or** stdio · TUI `[[mcp.servers]]` URL or command |
| **dual_write** | **OFF** · not primary palace path |
| **not Memory GA** | M2 lean edge ≠ product Memory GA / multi-tenant hosted palace |
| **aion broker private** | Cloud broker/CP stays private · not OSS edge pack |
| **scaffold/M2 residual** | Tool parity may be **lean** vs platform residual sidecar · **PASS ≠ invent full platform sidecar parity** |
| **public flip** | **OSS path ≠ invent public flip complete** · M2 scaffold ≠ M4 public flip |

**M3 edge dogfood (s1463 · residual-honest · after M2 scaffold · not invent live dogfood green / flip):**

TUI ↔ product **`iomesh-memory-mcp`** compose / HTTP / stdio dogfood tip (docs + `/onboard next memory` — **offline dogfood tip ≠ invent live dogfood as green**):

| Surface | Residual honesty |
|---------|------------------|
| **Product repo** | Build/run from **`github.com/iome-sh/iomesh-memory-mcp`** (peer free eng **s1462** edge dogfood product — mention only) |
| **Compose** | In product repo: `docker compose up --build` → image **`iomesh-memory-mcp:local`** · port `8080:8080` |
| **Attach (HTTP preferred)** | Streamable HTTP `http://127.0.0.1:8080/mcp` · TUI `[[mcp.servers]]` URL |
| **healthz** | `curl -fsS http://127.0.0.1:8080/healthz` (probe only · ≠ invent tool green / Memory GA) |
| **stdio alternate** | `go run` / `go build` binary · `command` + `args` in `[[mcp.servers]]` (no HTTP) |
| **Offline gate** | Peer mcp `make edge-dogfood-gate` (mention only) · TUI docs this section |
| **dual_write** | **OFF** · not primary palace path |
| **not Memory GA** | M3 edge dogfood ≠ product Memory GA / multi-tenant hosted palace |
| **post-s1478 public edge** | Host/kernel **public** (s1478) · aion broker/CP **still private** · aion broker private · s1517 product-only in-tree sample |
| **compose PASS ≠ public registry** | Local image tag `iomesh-memory-mcp:local` ≠ invent GHCR public pull green |
| **offline ≠ live dogfood** | **offline dogfood tip ≠ invent live dogfood as green** · residual PASS ≠ live APPLY |
| **parity / flip** | **PASS ≠ invent full platform sidecar parity** · residual PASS ≠ invent public flip (edge packs public as of s1478) · M3 after M2 · M4 readiness tip history below |

**M4 public flip readiness (s1469 · residual-honest history · after M3 dogfood):**

TUI residual-honest **readiness tip history** (pre-flip). **s1478 supersedes operator tip** — edge packs are now public; readiness tip ≠ invent Memory GA.

| Surface | Residual honesty |
|---------|------------------|
| **Order** | Kernel **`github.com/iome-sh/memory` first** · then product host **`iomesh-memory-mcp`** |
| **Post-s1478** | Both edge repos **public** · install without GOPRIVATE (see s1478 below) |
| **dual_write** | **OFF** · not primary palace path |
| **not Memory GA** | M4 readiness / public flip ≠ product Memory GA / multi-tenant hosted palace |
| **aion broker private** | Cloud broker/CP stays private · not OSS edge pack · not flipped with edge pack |
| **M5 later** | **M5 signing later after flip** · signing residual is post-flip deliberate wave |

**s1478 public product attach (residual-honest · after public flip of both edge repos):**

| Surface | Residual honesty |
|---------|------------------|
| **Kernel public** | `go get github.com/iome-sh/memory@main` · **no GOPRIVATE** |
| **Host public** | `go install github.com/iome-sh/iomesh-memory-mcp/cmd/iomesh-memory-mcp@main` (or clone) · **no GOPRIVATE** / PAT |
| **Attach** | Streamable HTTP `http://127.0.0.1:8080/mcp` **or** stdio `iomesh-memory-mcp` |
| **docker compose** | Still valid in product repo → image `iomesh-memory-mcp:local` · healthz |
| **dual_write** | **OFF** · not primary palace path |
| **not Memory GA** | Public OSS edge ≠ invent Memory GA / freemium palace |
| **aion broker private** | Cloud broker/CP **still private** · aion broker private · s1517 product-only sample |
| **flip complete residual** | Flip complete residual ≠ invent Memory GA · public OSS ≠ invent platform GA · package load ≠ Memory GA |
| **Product sample plugin** | `examples/agent-plugins/iomesh-memory-mcp` (dogfood primary with hello-iome) |

```bash
# Public install — no GOPRIVATE / PAT
go install github.com/iome-sh/iomesh-memory-mcp/cmd/iomesh-memory-mcp@main
go get github.com/iome-sh/memory@main
# attach TUI: url = "http://127.0.0.1:8080/mcp"  or  command = "iomesh-memory-mcp"
```

**Attach TOML snippet (HTTP preferred · residual-honest):**

```toml
[mcp]
enabled = true
[[mcp.servers]]
name = "iomesh-memory-mcp"
url = "http://127.0.0.1:8080/mcp"
allow_loopback = true
mutating = true

# Alternate stdio (no HTTP):
# command = "iomesh-memory-mcp"
# args = ["-palace-root", "./data/memory-palaces", "-tenant", "default"]

[memory]
enabled = true
server = "iomesh-memory-mcp"
tenant = "default"
dual_write = false   # OFF · not primary palace · package load ≠ Memory GA
```

**healthz curl (probe only):**

```bash
# from product repo github.com/iome-sh/iomesh-memory-mcp (public · no GOPRIVATE)
docker compose up --build
curl -fsS http://127.0.0.1:8080/healthz
# attach TUI [[mcp.servers]] url = "http://127.0.0.1:8080/mcp"
```

**Also residual attach:**

- Streamable HTTP MCP (local edge docker when available — peer aion s1306 compose; see [Local-edge Docker](#local-edge-docker-memory-mcp-s1308--peer-aion-s1306) above)
- s1517: in-tree product sample is `examples/agent-plugins/iomesh-memory-mcp` only (residual aion Memory sample removed)
- Product sample map: `examples/agent-plugins/iomesh-memory-mcp`
- Package load ≠ Memory GA · docker edge ≠ invent GA · flip complete residual ≠ invent Memory GA · public OSS ≠ invent platform GA · lean surface may lag platform residual · offline dogfood tip ≠ invent live dogfood as green
- **E4 full MCP client attach dogfood (s1508)** — observed stamp below · **not** forever-green product dogfood

##### E4 MCP client attach dogfood (s1508)

Residual-honest **full MCP client attach** dogfood for product tip TUI ↔ lean product host `iomesh-memory-mcp` over streamable HTTP. This is an **operator runbook + pinned evidence stamp** — **not** Edge Memory GA declared, not bare Memory GA, not hosted Memory GA, not forever-green product dogfood.

**Edge Memory GA candidacy (honesty frame):**

| Claim | Truth |
|-------|--------|
| **local-primary** | Customer-edge FS palace via lean host + TUI MCP client |
| **dual_write** | **OFF** · not primary palace path |
| **Edge Memory GA candidacy only** | Residual candidacy language · **residual PASS ≠ invent Edge Memory GA declared** |
| **not bare Memory GA** | Attach dogfood ≠ invent bare product Memory GA |
| **not hosted Memory GA** | Local lean host ≠ multi-tenant hosted / freemium cloud palace |
| **aion broker private** | Cloud broker/CP stays private · not OSS edge pack |
| **E10 Open** | Founder/GTM sign-off remains **Open** · tip ≠ invent E10 closed |
| **tip ≠ invent forever-green** | One observed stamp · not continuous CI green product dogfood |

**Runbook (observed steps that worked):**

1. Start lean host (example ports for isolated dogfood; default product path often uses `:8080`):

```bash
iomesh-memory-mcp \
  -palace-root /tmp/e4-palace \
  -tenant e4attach \
  -http-addr 127.0.0.1:18081 \
  -http-path /mcp
```

2. healthz OK — expect residual honesty fields such as `dual_write=off` and `not_memory_ga=true` (probe only · ≠ invent tool green / Memory GA):

```bash
curl -fsS http://127.0.0.1:18081/healthz
```

3. Temp TUI `config.toml`:

```toml
[mcp]
enabled = true
[[mcp.servers]]
name = "iomesh-memory-mcp"
url = "http://127.0.0.1:18081/mcp"
```

4. Connect from TUI tip binary:

```bash
./bin/iomesh mcp --config <cfg> --connect
# observed: connected=1 · tools=6
# tools listed: memory_ingest_turn, memory_retrieve, memory_search_semantic,
#   memory_list, memory_compact_status, memory_facts_as_of
```

**Pinned evidence (do not invent more):**

| Field | Value |
|-------|--------|
| **UTC stamp** | `2026-08-09T06:23:34Z` |
| **TUI tip** | `6b3958a90a01d2c8f50ee161c8dc1009637b64f1` |
| **MCP tip** | `f46afe2462ebaa94890b30296b1a19d03d6853da` (host binary version stamp `f46afe2`) |
| **Result** | `connected=1` · `tools=6` · tools listed above |
| **Evidence note** | [docs/EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md](../EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md) |

**Honesty hard locks (s1453+s1458+s1463+s1469+s1478+s1508):**

| Claim | Truth |
|-------|--------|
| **dual_write** | **OFF** — optional mesh audit only · not primary palace |
| **not Memory GA** | Edge install residual ≠ product Memory GA / multi-tenant hosted palace |
| **Edge Memory GA candidacy only** | Local-primary candidacy residual · **PASS ≠ invent Edge Memory GA declared** |
| **E10 Open** | Founder/GTM sign-off open · tip ≠ invent E10 closed |
| **Palace sunset** | Hosted cloud Palace sunset until scale |
| **mesh optional** | Mesh optional for pull only · memory lane is local-edge palace |
| **iomesh-memory-mcp** | Product MCP host only (s1517) |
| **public (s1478)** | Kernel + host **public** · `go install` / `go get` · **no GOPRIVATE** |
| **M2 lean attach** | Host available from `github.com/iome-sh/iomesh-memory-mcp` · HTTP `:8080/mcp` or stdio |
| **M3 edge dogfood** | Product compose/HTTP/stdio tip · image `iomesh-memory-mcp:local` · healthz · offline tip only |
| **E4 client attach (s1508)** | Full MCP client attach stamp · `connected=1` · `tools=6` · tip ≠ invent forever-green dogfood |
| **offline dogfood tip ≠ invent live dogfood as green** | Docs/onboard tip residual · does not claim forever-green live dogfood / Connected |
| **compose PASS ≠ public registry** | Local image tag ≠ invent public GHCR pull green |
| **aion broker private** | aion cloud broker/CP stays private · not open-sourced with edge pack |
| **aion still private** | Cloud broker/CP private · not OSS edge pack · s1517 no in-tree residual aion Memory sample |
| **flip complete residual ≠ invent Memory GA** | Public edge packs ≠ invent Memory GA / freemium palace |
| **public OSS ≠ invent platform GA** | Public MIT edge modules ≠ invent multi-tenant platform Memory GA |
| **PASS ≠ invent full platform sidecar parity** | Lean tool surface may lag platform residual · attach residual ≠ invent full parity |
| **never invent Connected** | Attach residual fail-open · residual PASS ≠ invent forever-green dogfood · PASS ≠ live APPLY |

Operator surfaces: `/onboard next memory` · `/onboard next memory-pull` · `/onboard next operator` · product sample `examples/agent-plugins/iomesh-memory-mcp` · E4 evidence stamp · **end-to-end usage/demo walkthrough (s1513):** [memory-edge-usage-demo.md](./memory-edge-usage-demo.md) (signup → integrations HITL → local kernel+MCP install honesty → attach → show `/memory` usage).

### Temporal retrieve options (s1068)

Platform sidecar accepts `since` / `until` / `session_seq` on `POST /v1/memory/retrieve` (and `/v5`) and maps them to kernel `SearchMemoryWithOptions`. TUI lean client and agent auto-recall wire those fields so time-windowed recall drops irrelevant hits:

| Surface | How |
|---------|-----|
| Lean HTTP | `iomesh.MemoryRetrieveOptions` → `RetrieveMemoryWithOptions` (body includes non-empty `since`/`until`/`session_seq`); `RetrieveMemory` remains a thin wrapper |
| Config | `[memory] recall_since` / `recall_until` / `recall_session_seq` (env `IOMESH_MEMORY_RECALL_SINCE` / `_UNTIL` / `_SESSION_SEQ`) applied on auto-recall and default `/memory recall` |
| Slash | `/memory recall --since RFC3339 --until RFC3339 --session-seq N [query]` (overrides config for that call) |
| MCP fallback | Same keys forwarded on `memory_retrieve` tool args when set |

Fail-open unchanged; dual_write remains optional audit (**default OFF**); does not invent temporal pipeline GA.

### Mesh pull → local palace (s652 M1)

```bash
# Terminal A: local palace MCP
iomesh-memory-mcp -http-addr :8080 -palace-root ~/.iomesh/palace

# Terminal B: pull mesh events into local palace (requires [iomesh] + [mcp] memory server)
iomesh memory pull --stream EVENTS --name tui-local-palace --once --yes
# dry-run (map only, no MCP):
iomesh memory pull --stream EVENTS --name tui-local-palace --once --dry-run
# JSON always-emit identity + knobs + counters + process evidence for CI scrapers
# (s705/s717 product; s747 completeness pin — surface complete):
iomesh memory pull --stream EVENTS --name tui-local-palace --once --dry-run --json
```

Loop: `CreateConsumer` (idempotent) → `ConsumerFetch` → map envelope → MCP `memory_ingest_turn` → `ConsumerAck`.  
Primary: connector/`dept.*` or `EVENTS`. Optional: pull `MEMORY_INGEST` when using mesh as audit mirror.  
When `--filter` / `[memory].pull_filter` is empty and `[memory].tenant` or `[iomesh].tenant` is hierarchical (`dept.*` or contains `.`), default `filter_subject` is `tenant.>` (s660); create/fetch/ack send `X-IOMesh-Tenant` via client auth.

**s705 (Beta):** PASS/summary text and `--json` always emit identity (`stream`, `consumer`, `filter_subject`, `pull_role`, `pull_allow_suffix`, `tenant` — empty string when unset), knobs (`dry_run`, `dual_write` report-only default false, `batch`, `max_wait_ms`, `once`), and counters including `last_error` (empty when none). Print DTO is `MemoryPullStatsPrint` (no omitempty on scraper keys). Does not invent pull success from identity fields alone. Peer create FormatConsumerInfo s696 + status/wait pull identity continuum; peer aion s704. Fail-open empty role/tenant; dual_write default OFF; not full mesh RBAC GA.

**s717 (Beta):** process evidence always-emit on the same print DTO: `endpoint` / `org` / `workspace` (from `[iomesh]`; empty string honest when unset), `result` (`ok`|`err`), `exit_code` (0 success; 1 hard-fail non-cancel or soft-fail `errors>0 && ingested==0 && !dryRun`), `duration_ms` (wall-clock; 0 if not timed), `ack` knob. Early fail paths (mesh disabled / MCP missing) emit the DTO when possible. Process evidence ≠ invent pull success from identity alone. Peer aion s716 residual after s705. Dual_write default OFF; not full mesh RBAC GA.

**s747 (Beta · completeness pin):** memory pull process evidence **complete** — identity (s705) + knobs/counters + process evidence (s717) locked by docs + unit tests (`MemoryPullStatsPrint` JSON always-emits `result` / `exit_code` / `endpoint` / `org` / `workspace` on empty and populated paths). Completeness pin **does not** invent new always-emit fields and **does not** re-claim s717 product body. Peer aion s746 residual honesty. Process evidence ≠ invent pull success · dual_write OFF · not full mesh RBAC GA · offline unit ≠ live APPLY · empty honest.

**s675 (Beta):** optional `--role` / `[memory].pull_role` → `X-IOMesh-Role` and `--pull-allow-suffix` / `[memory].pull_allow_suffix` → `X-IOMesh-Pull-Allow-Suffix` on authenticated mesh requests (create/fetch/ack). Fail-open when empty (headers omitted). Not full mesh IdP RBAC GA; dual_write remains optional audit (default OFF). Hosted Palace sunset — TUI path is local palace.

**s678 (Beta):** when `--filter` / `pull_filter` is empty, default `filter_subject` is role-aware (`DefaultMemoryPullFilterForRole`): empty role keeps s660; `agent`/`viewer` → `tenant.events.>`; `auditor` → `tenant.audit.>`; `operator`/`admin` → `tenant.>`; `custom` with exactly one allow-suffix token → `tenant.<suffix>.>`; custom multi/no suffix → empty (fail closed). Explicit filter always wins. Peer aion s678; not full mesh RBAC GA.

**s681 (Beta):** `iomesh mesh consumer create` also accepts `--role` / `[memory].pull_role` and `--pull-allow-suffix` / `[memory].pull_allow_suffix` (sets client `Config.Role` / `PullAllowSuffix` so create auth sends the same headers as memory pull). Empty `--filter` uses the same `DefaultMemoryPullFilterForRole` defaults (IOMesh tenant). Fail-open without role; dual_write default OFF; peer aion s680 continuum — not full mesh RBAC GA.

**s684 (Beta):** `iomesh mesh consumer fetch` (and ack/nack/delete) accept the same `--role` / `--pull-allow-suffix` flags and config fallbacks so broker fetch validates federated ACL headers (aion s683 continuum). Fail-open without role; dual_write default OFF — not full mesh RBAC GA.

**s687 (Beta):** role=`memory` default filter → `tenant.memory.>` when tenant set (peer aion s686 federated pull memory role; local-palace memory subjects only). Dogfood report always-emits `pull_role` / `pull_allow_suffix` from Client Config (empty when unset) for CI scrapers; dogfood CLI wires `[memory].pull_role` / `pull_allow_suffix` onto Client so the soft consumer probe sends headers + applies role-aware empty-filter defaults. Fail-open without role; dual_write default OFF; not full mesh RBAC GA.

**s690 (Beta):** `iomesh mesh status` always-emits `pull_role` / `pull_allow_suffix` (empty when unset) in text and JSON from the same Client Config path (`[memory].pull_role` / `pull_allow_suffix` wired onto Client like dogfood s687). CI scrapers can key stable identity without omitempty gaps. Fail-open without role; dual_write default OFF; not full mesh RBAC GA; peer aion s689 residual gate continuum.

**s693 (Beta):** `iomesh mesh wait` always-emits `pull_role` / `pull_allow_suffix` (empty when unset) in text and JSON from the same Client Config path (`[memory].pull_role` / `pull_allow_suffix` wired onto Client like status s690). CI scrapers can key stable identity without omitempty gaps. Fail-open without role; dual_write default OFF; not full mesh RBAC GA; peer aion s692 Ops Pack floors residual gate continuum.

**s696 (Beta):** `iomesh mesh consumer create` text (`FormatConsumerInfoWithAuth`) and JSON (`ConsumerInfoPrint`) always-emit `pull_role` / `pull_allow_suffix` (empty when unset) next to `filter_subject` from resolved create auth (s681). Wire `ConsumerInfo` decode stays free of auth fields. CI scrapers can key stable identity without omitempty gaps. Fail-open without role; dual_write default OFF; not full mesh RBAC GA; peer aion s695 sales claim continuum.

**s705 (Beta):** `iomesh memory pull` always-emits full pull identity + knobs + counters in PASS/summary text and `--json` (`MemoryPullStatsPrint`) so scrapers are not limited to stderr start log. Empty role/tenant/filter honest; dual_write report-only default OFF; peer aion s704 continuum — not full mesh RBAC GA.

**s717 (Beta):** same surface always-emits process evidence (`endpoint`/`org`/`workspace` empty honest; `result` ok|err; `exit_code` 0|1; `duration_ms`; `ack`) so CI scrapers record intended exit without shell `$?`. Soft-fail and hard-fail set `result=err`/`exit_code=1`; success sets `ok`/`0`. Process evidence ≠ invent pull success. Peer aion s716 residual — dual_write default OFF; not full mesh RBAC GA.

**s747 (Beta · completeness pin):** process evidence surface complete (s705 + s717) — unit tests pin always-emit keys; no new DTO fields. Peer aion s746 · process evidence ≠ invent pull success · dual_write OFF · not full mesh RBAC GA · offline unit ≠ live APPLY.

Env: `MEMORY_MCP_HTTP_ADDR` / `AION_MEMORY_MCP_HTTP_ADDR`, path `MEMORY_MCP_HTTP_PATH` (default `/mcp`).

### Alternate: stdio

The platform Memory MCP server binary is supplied by the I/O Mesh platform / operator install (not built from this repo). Place it on `PATH` or pass an absolute `command` path:

```toml
[mcp]
enabled = true

[[mcp.servers]]
name = "memory"
command = "iomesh-memory-mcp"   # product Memory MCP server binary name
args = ["-palace-root", "/data/memory-palaces"]
# env = { "MEMORY_TENANT" = "dept.research", "QDRANT_URL" = "…" }
mutating = true   # ingest tools need approval unless --yolo
# tool_timeout_sec = 60

[memory]
enabled = true
server = "memory"
tenant = "dept.research"
auto_recall = true
auto_ingest = false
```

### Verify

```bash
iomesh mcp --connect
# expect memory_* tools under server "memory"

iomesh   # interactive
# /memory                 → status (sync_http= / mcp=)
# /memory recall <query>  → sync POST /v1/memory/retrieve when mesh enabled, else MCP memory_retrieve
```

Agent tools also appear as `mcp__memory__memory_retrieve` (etc.) when MCP is attached.

### Env overrides

| Env | Effect |
|-----|--------|
| `IOMESH_MEMORY=1` | Enable `[memory]` hooks |
| `IOMESH_MEMORY_TENANT` / `MEMORY_TENANT` | Default tenant for hooks + slash |
| `IOMESH_MEMORY_AUTO_RECALL=0` | Disable per-turn retrieve inject |
| `IOMESH_MEMORY_AUTO_INGEST=1` | Enable post-turn ingest (MCP and/or dual-write) |
| `IOMESH_MEMORY_DUAL_WRITE=1` | Also publish async `MEMORY_INGEST` when mesh enabled |
| `IOMESH_MEMORY_ENDPOINT` / `MEMORY_SIDECAR_URL` | Sync retrieve base (memory sidecar); overrides mesh endpoint for `RetrieveMemory` only |
| `IOMESH_MEMORY_RECALL_SINCE` / `IOMESH_MEMORY_RECALL_UNTIL` | Optional RFC3339 auto-recall window (s1068) |
| `IOMESH_MEMORY_RECALL_SESSION_SEQ` | Optional session_seq filter for temporal recall (s1068; 0 omits) |
| `IOMESH_MCP=1` | Enable MCP section |

## Phase 1 — runtime loop

```
user turn
  → [optional auto_recall]
        prefer: RetrieveMemory → POST /v1/memory/retrieve (+ /v5)   # when [iomesh] enabled
        else:   MCP memory_retrieve                                 # when server connected
        → <memory-context> system msg (fail-open)
  → LLM + tools
  → [optional auto_ingest]
        memory_ingest_turn(user) + memory_ingest_turn(assistant)   # MCP when connected
        + PublishMemoryIngest → MEMORY_INGEST                      # when dual_write
```

- **Fail-open**: MCP down, empty hits, dual-write errors, or tool errors never fail the turn.
- **Sync prefer**: when mesh is enabled **or** `[memory].endpoint` / `IOMESH_MEMORY_ENDPOINT` is set, auto-recall and `/memory recall` try lean HTTP first; on transport/404 (broker-only URL without sidecar) fall back to MCP.
- **No Palace import**: only MCP `tools/call` and/or lean HTTP (no SDK module dependency).
- **Mutating**: auto-ingest bypasses the interactive approval UI (operator opt-in via `auto_ingest`); interactive `mcp__memory__*` still requires approval when `mutating=true`.

### Phase 3+ — sync auto-recall without MCP

Operators with a memory sidecar (or gateway that routes `/v1/memory/retrieve`) can enable auto-recall with mesh only:

```toml
[iomesh]
enabled = true
endpoint = "https://mesh.stage.example"   # broker / control plane
tenant = "dept.research"

[memory]
enabled = true
auto_recall = true
tenant = "dept.research"
# Dedicated sidecar when mesh endpoint is broker-only (stage warm plane):
endpoint = "http://127.0.0.1:8765"
# Env: IOMESH_MEMORY_ENDPOINT / MEMORY_SIDECAR_URL
# MCP server optional when sync HTTP works
```

`/memory` status shows `sync_http=true|false` and `mcp=true|false`. Empty hit lists still succeed (no injection).

## Phase 2 — dual-write MEMORY_INGEST (v0.3.0)

When `dual_write = true` (or `IOMESH_MEMORY_DUAL_WRITE=1`) **and** `[iomesh]` mesh client is enabled:

1. After MCP ingest (or **instead of** MCP when no server is connected), publish an async envelope to  
   `POST /v1/streams/MEMORY_INGEST/publish`.
2. Subject: `{tenant}.memory.ingest.turn`
3. Payload (base64 JSON, same wire as public SDK):

```json
{
  "type": "memory_ingest",
  "session_id": "…",
  "role": "user|assistant",
  "content": "…",
  "event_time": "2026-07-16T12:00:00Z",
  "session_seq": 1
}
```

- `session_seq` is **monotonic per Runtime** (process lifetime of the agent session).
- Tenant from `[memory].tenant`, else mesh tenant.
- Dual-write is **independent** of MCP success (fail-open both ways).
- Useful for durable stream consumers / temporal pipelines without embedding Palace in the TUI.

Requires mesh:

```toml
[iomesh]
enabled = true
endpoint = "https://mesh.example"
tenant = "dept.research"
# api_key_env = "IOMESH_API_KEY"

[memory]
enabled = true
auto_ingest = true
dual_write = true
tenant = "dept.research"
```

### Dogfood dual-write probe

`iomesh mesh dogfood` includes a **memory_ingest** step by default when mesh is enabled. It exercises the same `PublishMemoryIngest` path (not MCP Palace write):

```bash
iomesh mesh dogfood --tenant dept.research
# soft: FAIL on publish → SKIP unless --strict
iomesh mesh dogfood --strict
iomesh mesh dogfood --skip-memory   # omit the step
```

See [mesh-dogfood.md](mesh-dogfood.md) for soft vs strict matrix. Unit coverage: `go test ./internal/iomesh` (httptest mock returns 200 on `/v1/streams/MEMORY_INGEST/publish`).

## Slash commands

| Command | Behavior |
|---------|----------|
| `/memory` | Status: enabled, `mcp=`, `sync_http=`, flags (incl. `dual_write`), tenant · **s1831** residual next-step dual path |
| `/memory status\|st` | Base status line + s1311 advanced MCP inventory pulse (`MemoryAdvancedStatus`) · **s1831** next-step footer |
| `/memory recall [query]` | Sync HTTP retrieve when mesh enabled, else MCP (default query = last user text or `"*"`) |
| `/memory recall --since|--until|--session-seq … [query]` | Same + per-call temporal filters (s1068; override config) |
| `/memory related --seed <entity> [--query …] [--max-hops N] [--prefer-shorter-hops\|--legacy-sort]` | Opt-in multi-hop lite related recall (s1135 + s1281; HTTP + MCP `memory_related`; PreferShorterHops omit=true; not auto-recall) |
| `/memory digest [--window day\|week] [--horizon ops\|knowledge\|analytical\|all] [--limit N] [--require-sources mesh,private]` | Opt-in ops heartbeat digest export (s1200; HTTP + MCP `ops_digest_export`) · **#373** cite-both opt-in (`mesh,private` or explicit miss; catalog/grant/external never count) · **#369** honesty · **#370** delta briefs · **s1831** next-step footer |
| `/memory facts-as-of\|facts\|as-of --as-of <RFC3339> [--entity …] [--query …] [--limit N]` | Opt-in bi-temporal lite validity listing (s1276; MCP `memory_facts_as_of`; MCP-first) |
| `/memory supersede\|super --entity <key> [--as-of RFC3339] --i-confirm` | Opt-in HITL A3 lite supersede (s1282; MCP `memory_supersede_entity`; MCP-first; mutating) |
| `/memory timeline\|tl […]` | Opt-in temporal timeline (s1296; MCP `memory_timeline`; MCP-first) |
| `/memory compact-status\|compact` | Opt-in Palace tier counts residual (s1296; MCP `memory_compact_status`; read-only) |
| `/memory trigger-compact\|tcompact --i-confirm` | Opt-in HITL RecMem compact advisory (s1311; MCP `memory_trigger_compact`; mutating) |
| `/memory semantic\|sem [query]` | Opt-in tier-4 semantic facts (s1301; MCP `memory_search_semantic`) |
| `/memory ingest-event\|event --subject … --content …` | Opt-in s138 T1 event telemetry (s1301; MCP `memory_ingest_event`; not conversation turn) |
| `/memory patterns` / `/memory anomalies` | Opt-in MCP ops pulse Beta lists (shipped s1287; `memory_patterns_list` / `memory_anomalies_list`; when present) |
| `/memory ingest <text>` | Ingest a user turn (MCP and/or dual-write). `session_id` is minted as `local-overlay` when the operator has none (`iomesh-memory-mcp` v0.1.0 requires it). Retrieve without a session stays unfiltered and finds the overlay. |
| `/memory ingest-dir <path> [--dry-run] [--limit N]` | Folder ingest into the private overlay (`#384`). Workspace path jail. Same minted `session_id`. CLI: `iomesh memory ingest-dir`. Catalog list ≠ consume. |

## Ops heartbeat digest (s1200 · opt-in)

Operators can export a **day/week pattern + receipts pack** without changing default auto-recall:

| Surface | Path |
|---------|------|
| Lean HTTP | `iomesh.ExportOpsDigest` → `POST /v1/memory/ops_digest` (+ `/v5` fallback); body: `window` / `horizon` / `limit` / `as_of` |
| MCP fallback | `ops_digest_export` tool args (`window`, `horizon`, `limit`, `as_of`, `tenant`) when sync fails |
| Slash | `/memory digest --window week --horizon ops --limit 10` · optional `--require-sources mesh,private` |
| Output | Human-readable patterns + receipts + honesty line · with `--require-sources`, prefixed `require-sources: ok` or `require-sources: miss` |

**Cite-both (#373, opt-in):** `--require-sources mesh,private` requires receipt `source_hint`s for **both** mesh and private, or prints an explicit miss. Catalog-only / grant-only never satisfy cite-both (catalog list is not consume). Default `/memory digest` without the flag stays the existing ops pack. dual_write OFF · not Memory GA · local palace on disk.

**Honesty:** ops pulse **GA-path** · knowledge/analytical digests **Beta** · **never invent GA** · dual_write default OFF · **not** product Memory GA · **not** full graph RAG · human owns irreversible decisions. **#369:** may emit **insufficient-signal / nothing reliable today** · rate claims need **n of N + window** (else rejected) · receipts are **pointers + hashes** (not raw customer text) · catalog list ≠ consume. **#373:** `--require-sources mesh,private` cites both or explicit miss (catalog/grant ≠ cite-both). **#370:** briefs are a **change vs the prior window** (no-delta / “what is true” recap → insufficient-signal) · `source=external` is a **third labeled pane**, never the heartbeat, and never satisfies cite-both · first-party consume remains the only mesh citation path.

**Market-telling / voc_brief (#372):** `/gtm brief` writes a named palace entry (`kind=market_telling|voc_brief`, `source=agent-brief`, tenant `gtm/founder`) to the **local palace** (`palace/gtm-founder/market_telling.json` beside user config). That file is the system of record — not a git markdown. `source=agent-brief` classifies as **private** for cite-both (never mesh; catalog/grant/external still do not satisfy). Hypothesis ledger: shipped / moved / killed (dropped ≠ falsified) vs falsified + contradiction vs yesterday. Cadence `daily|weekly|on_threshold`; daily refused below event floor. One RevOps **support-theme** recipe uses the same receipt metadata as incidents (id · event_time · summary · source_hint · pointer · account_hash · kind · subject) with ≤3 first-party sources (`mesh`, `private`, `github`). No Slack persist. CRM ≠ Connected. Hands (win-back, price change) stay off this plane. dual_write OFF · not Memory GA.

## Multi-hop related recall (s1135 · opt-in · s1281 hop ranking)

Operators can run **multi-hop lite** related recall without changing default auto-recall (still single-hop `RetrieveMemory` / `memory_retrieve`):

| Surface | Path |
|---------|------|
| Lean HTTP | `iomesh.RetrieveMemoryRelated` → `POST /v1/memory/related` (+ `/v5` fallback); body: `seed_entity` / `query` / `max_hops` / `limit` / `session_id` / optional `prefer_shorter_hops` |
| MCP fallback | `memory_related` tool args (`seed_entity`, `query`, `max_hops`, `limit`, `session_id`, `tenant`, optional `prefer_shorter_hops`) when sync fails |
| Slash | `/memory related --seed person:alice --query "…" --max-hops 2 [--prefer-shorter-hops\|--legacy-sort]` |
| Hits | Optional `hop_distance` on `MemoryHit` (hop ranking lite · s1067 kernel); formatted as `[hop=N]` |

### prefer_shorter_hops (s1281 · aion s1277)

| Rule | Behavior |
|------|----------|
| Omit / nil | Kernel default **true** (path-aware hop ranking: shorter BFS hops then event time) |
| `true` | Explicit shorter-hops ranking |
| `false` | Legacy seed-first sort (`--legacy-sort` / `--no-prefer-shorter-hops`) |
| HTTP body | Field **omitted** when nil (do not send false invent); sent only when set |
| MCP args | Same omit-when-nil pattern |

**Honesty:** multi-hop lite opt-in · hop ranking path-aware lite · PreferShorterHops default true · **not** full graph RAG · **not** product Memory GA · dual_write default OFF · fail-open.

## Bi-temporal lite facts-as-of (s1276 · opt-in · MCP-first)

Operators can list palace entries **valid at a point in time** without changing default auto-recall:

| Surface | Path |
|---------|------|
| Lean HTTP | **None today** — platform does not expose `POST /v1\|/v5/memory/facts_as_of`; do not invent |
| MCP (primary) | `memory_facts_as_of` tool args (`tenant`, `as_of` required RFC3339; optional `entity`, `query`, `session_id`, `limit`) |
| Slash | `/memory facts-as-of --as-of 2026-08-04T12:00:00Z [--entity person:alice] [--query …] [--limit N]` |
| Aliases | `facts`, `as-of` |
| Output | Human-readable facts + honesty footer; empty → `facts: (none)` (never invent memories) |
| Offline | Residual-honest fail-open message (status: unavailable · MCP-first · empty ≠ invent) |

**Honesty hard locks:** opt-in only (not auto-recall) · K4 bi-temporal **lite** · **not** full dual-clock Graphiti KG · **not** Memory GA · dual_write OFF · book-demo OFF · empty facts ≠ invent memories · fail-open.

## A3 lite supersede HITL (s1282 · opt-in · MCP-first · mutating)

Operators can **close open validity windows** for entity-tagged facts only with explicit human confirmation:

| Surface | Path |
|---------|------|
| Lean HTTP | **None today** — do not invent `POST /v1\|/v5/memory/supersede` |
| MCP (primary) | `memory_supersede_entity` args (`entity` required; optional `as_of` RFC3339, `tenant`) |
| Slash | `/memory supersede --entity person:alice [--as-of 2026-08-04T12:00:00Z] --i-confirm` |
| HITL | `--i-confirm` / `--confirm` / `--yes` required; without it residual refuse (no MCP call) |
| Output | `superseded_count` from wire only · honesty footer; offline / empty → never invent count |

**Honesty hard locks:** opt-in only · **HITL required** · A3 lite · closes `valid_until` · **not** NLP contradiction · **not** full dual-clock Graphiti · **not** Memory GA · dual_write OFF · book-demo OFF · MCP-first (no lean HTTP invent) · fail-open · empty/zero count honest when wire says so.

## Patterns / anomalies ops pulse (shipped s1287 · MCP Beta)

Shipped **ops pulse Beta** lists (MCP-first; no lean HTTP invent):

| Surface | Path |
|---------|------|
| Lean HTTP | **None invent** — do not invent `/memory/patterns` or `/memory/anomalies` lean routes |
| MCP | `memory_patterns_list` / `memory_anomalies_list` when platform exposes them |
| Slash | `/memory patterns` · `/memory anomalies` (when wired) |
| Honesty | ops pulse **Beta** · **not** medical · **not** invent GA window engine · dual_write OFF · **not** Memory GA · fail-open |

## Advanced agent skill (s1288) + system note (s1291)

Builtin skill **`memory-advanced-agent`** (`internal/skills/builtin/memory-advanced-agent/SKILL.md`) is residual-honest agent guidance for the advanced surfaces above. Loaded via `skills.LoadBuiltin` / `LoadWithBuiltin` whenever skills are enabled (same mold as s1251 `connector-integrations-setup`). Skill-only — does not change product slash/agent paths.

**s1291:** on `AttachMCP`, runtime injects residual-honest `<memory-advanced>` system note from `MemoryAdvancedAgentGuidanceNote()` (mirror integrations s1251). Opt-in advanced locks only; does not invent Memory GA / silent supersede / auto multi-hop.

| Skill maps | Slash | MCP |
|------------|-------|-----|
| related + prefer_shorter_hops | `/memory related …` | `memory_related` |
| supersede HITL | `/memory supersede … --i-confirm` | `memory_supersede_entity` |
| facts-as-of | `/memory facts-as-of …` | `memory_facts_as_of` |
| digest | `/memory digest …` | `ops_digest_export` |
| patterns / anomalies (shipped s1287) | `/memory patterns\|anomalies` | `memory_patterns_list` / `memory_anomalies_list` |
| timeline (s1296) | `/memory timeline …` | `memory_timeline` (MCP-first) |
| compact-status (s1296 · read-only) | `/memory compact-status` | `memory_compact_status` (MCP-first; read-only) |
| trigger-compact HITL (s1311) | `/memory trigger-compact --i-confirm` | `memory_trigger_compact` (MCP-first; mutating RecMem advisory; HITL) |
| advanced status inventory (s1311) | `/memory status` | presence probe (`MemoryAdvancedStatus`) — no invent green |

## Timeline + compact-status (s1296 · MCP-first)

Opt-in residual-honest advanced surfaces (not auto-recall):

| Surface | Path |
|---------|------|
| Lean HTTP | **None invent** — do not invent `/memory/timeline` or `/memory/compact_status` lean routes |
| MCP | `memory_timeline` · `memory_compact_status` when platform exposes them |
| Slash | `/memory timeline [--since\|--until\|--session-id\|--query\|--limit]` · `/memory compact-status` |
| Output | timeline entries (id/summary/event_time) + honesty footer; compact tiers + `last_compaction` from wire only |
| Mutating compact | s1311 HITL: `/memory trigger-compact --i-confirm` → `memory_trigger_compact` (not auto from compact-status) |

**Honesty hard locks:** opt-in only · temporal timeline ≠ Memory GA · filters before limit · compact status ≠ invent compaction green · not auto-compact product · dual_write OFF · MCP-first (no lean HTTP invent) · fail-open · empty entries honest.

## Trigger-compact HITL + advanced status (s1311 · MCP-first)

Opt-in residual-honest mutating compact advisory + operator inventory pulse:

| Surface | Path |
|---------|------|
| Lean HTTP | **None invent** — do not invent `/memory/trigger_compact` lean route |
| MCP | `memory_trigger_compact` when platform exposes it (`{triggered, cluster_size}`) |
| Slash | `/memory trigger-compact\|compact-trigger\|tcompact --i-confirm` (aliases `--confirm` / `--yes`) |
| HITL | Without `--i-confirm` residual refuse (no MCP call) — mirror supersede |
| Output | `triggered` + `cluster_size` from wire + honesty footer (RecMem advisory · not invent green) |
| Status | `/memory status` → `MemoryStatusLine` + `MemoryAdvancedStatus` (present/missing/offline inventory) |

**Honesty hard locks:** HITL required · RecMem advisory ≠ invent compaction green · dual_write OFF · not Memory GA · presence ≠ product green · MCP-first · fail-open · never invent triggered/cluster_size offline.

## Platform gaps

| ID | Gap |
|----|-----|
| M1 | Streamable HTTP for platform Memory MCP — **shipped** (TUI HTTP path ready) |
| M2 | Sync `POST /v5/memory/retrieve` (SDK) — optional non-MCP clients |
| M3 | SDK temporal envelope fields — **shipped** (TUI dual-write mirrors subset) |
| M4 | Optional stage warm memory path (prod lean may be absent) |
| M5 | Entitlements fail-closed on MCP |

## Package map

| Path | Role |
|------|------|
| `internal/config` | `[memory]` section + env (`dual_write`) |
| `internal/iomesh/memory.go` | `PublishMemoryIngest`, `PublishMemoryRecall`, `RetrieveMemory` / `RetrieveMemoryWithOptions`, `RetrieveMemoryRelated` (+ PreferShorterHops s1281), `ExportOpsDigest` lean HTTP (no SDK dep; s1068 temporal + s1135 related + s1200 digest; **no** facts_as_of / supersede / patterns HTTP invent) |
| `internal/agent/memory.go` | Recall (sync prefer → MCP; config + opts temporal filters) / related multi-hop + prefer_shorter_hops (s1135/s1281) / ops digest (s1200) / facts-as-of MCP-first (s1276) / supersede HITL MCP-first (s1282) / patterns+anomalies (s1287) / timeline+compact-status MCP-first (s1296) / trigger-compact HITL + advanced status inventory (s1311) / semantic+ingest-event (s1301) / ingest / dual-write helpers |
| `internal/agent/memory_guidance.go` | s1291 `MemoryAdvancedAgentGuidanceNote` residual-honest system note (AttachMCP inject; s1296 timeline+compact-status · s1311 trigger-compact HITL) · **s1831** `MemoryNextStepLines` residual dual-path next-step after `/memory` status/help/digest |
| `internal/agent/agent.go` | `RunTurn` hooks · `AttachMCP` injects integrations (s1251) + memory-advanced (s1291) notes |
| `internal/tui/tui.go` | `/memory` slash (related · digest · facts-as-of · timeline · compact-status · trigger-compact · status advanced · supersede · patterns · anomalies · …) |
| `internal/skills/builtin/memory-advanced-agent/` | s1288 residual-honest advanced memory agent skill |
| `configs/config.example.toml` | Copy-paste wire-up |

## After /memory (s1831)

Residual-honest **next-step footers** after primary `/memory` surfaces — peer of onboard next-step (s1825) · integrations next-step (s1727) · setup continuum (s1686–s1723).

Helper `MemoryNextStepLines()` is appended by:

| Surface | Path |
|---------|------|
| bare `/memory` (help) | TUI slash after status line + usage |
| `/memory status` | end of `MemoryAdvancedStatus` |
| `/memory digest` | TUI slash after digest body (incl. empty residual) |

Post-surface dual path:

1. **If TUI/session running** → `/setup preflight` · `/setup reload` · optional `/memory digest` · `/onboard next memory|memory-pull`
2. **Else cold start** → restart `iomesh` · `iomesh setup preflight` · optional `iomesh memory pull`

**Honesty:** dual_write **OFF** · not Memory GA · local-primary · package wire ≠ Connected · soft ≠ invent live dogfood · free eng **s1831**. Advanced slash surfaces (related · facts-as-of · timeline · …) keep their existing honesty footers and are **not** all re-wired in s1831 (fragmented residual).

## Honesty

- Local Palace via stdio/HTTP MCP or lean sidecar HTTP ≠ multi-tenant Cloud Run Memory Palace with full entitlements.
- Dual-write is **best-effort** stream publish; it does not guarantee Palace persistence by itself.
- “Native Vertex” / G4S claims are separate (see marketing claim matrix); memory is **Palace via MCP and/or lean HTTP sidecar**, not Vertex.
- Do not claim temporal pipeline is live unless stage/prod embedding + temporal flags are on.
- **s1308:** local-edge Docker MCP attach (peer aion s1306) is customer-edge FS palace only — dual_write OFF · hosted Palace sunset · **not** Memory GA · docker edge ≠ invent GA · local image tag ≠ public registry claim.
- **s1453:** edge OSS Option A residual-honest install story — product host **`iomesh-memory-mcp`** · kernel `github.com/iome-sh/memory` · aion broker **private** · dual_write OFF · not Memory GA · Palace sunset · mesh optional for pull · **OSS path ≠ invent public flip complete**.
- **s1458:** M2 lean edge host attach tip — when built from `github.com/iome-sh/iomesh-memory-mcp`: go run/build · streamable HTTP `http://127.0.0.1:8080/mcp` or stdio · dual_write OFF · not Memory GA · aion broker private · scaffold/M2 · tool parity may be lean vs platform residual · **PASS ≠ invent full platform sidecar parity** · keeps Option A honesty.
- **s1463:** M3 edge dogfood tip — TUI↔`iomesh-memory-mcp` compose/HTTP/stdio path: `docker compose up --build` → image `iomesh-memory-mcp:local` · attach `http://127.0.0.1:8080/mcp` · healthz · stdio alternate · peer mcp `make edge-dogfood-gate` (mention only) · dual_write OFF · not Memory GA · host/kernel **public as of s1478** · aion broker private · **offline dogfood tip ≠ invent live dogfood as green** · **PASS ≠ invent full platform sidecar parity** · residual PASS ≠ invent public flip · M3 after M2 · M4 later deliberate · keeps Option A + M2 honesty.
- **s1469:** M4 public flip readiness tip — order kernel `github.com/iome-sh/memory` **first** · then `iomesh-memory-mcp` · readiness tip history · dual_write OFF · not Memory GA · aion broker private · **M5 signing later after flip** · **s1478 supersedes operator tip** (edge packs public) · keeps Option A + M2 + M3 honesty.
- **s1478:** public product attach continuum — both edge repos **public** · `go install github.com/iome-sh/iomesh-memory-mcp/cmd/iomesh-memory-mcp@main` · `go get github.com/iome-sh/memory@main` · **no GOPRIVATE** · HTTP `:8080/mcp` or stdio · docker compose still valid · dual_write OFF · not Memory GA · aion broker **still private** · flip complete residual ≠ invent Memory GA · public OSS ≠ invent platform GA · aion broker private · s1517 product-only sample · product sample plugin `iomesh-memory-mcp`.
- **s1508:** E4 full MCP client attach dogfood — lean host HTTP → TUI `iomesh mcp --connect` observed **connected=1 · tools=6** (UTC `2026-08-09T06:23:34Z` · TUI tip `6b3958a…` · MCP tip `f46afe2…`) · dual_write OFF · local-primary · **Edge Memory GA candidacy only** · residual PASS ≠ invent Edge Memory GA declared · not bare Memory GA · not hosted Memory GA · aion broker private · **E10 Open** · tip ≠ invent forever-green product dogfood · evidence [EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md](../EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md).
- **s1513:** memory edge usage/demo example — residual-honest walkthrough signup (optional) → integrations list/plan + portal HITL → local kernel+MCP install (**not fully automatic**) → attach → show `/memory` usage · dual_write OFF · not Memory GA · Edge Memory GA candidacy only · E10 Open · catalog ≠ Connected · aion broker private · [memory-edge-usage-demo.md](./memory-edge-usage-demo.md).


## s1069 recall efficiency

Client-side short-TTL cache for sync `RetrieveMemory` (`[memory] recall_cache_ttl_ms`, default 3000; `0` disables). Keyed by tenant+session+query+limit+since/until. Fail-open process-local only — not product Memory GA. Snippet early-stop at `max_snippet_bytes`. Auto-recall events always emit retrieve latency (`Nms` / `Nms cache`).
- **s1517:** drop residual aion Memory sample from TUI tree — product host/docs/onboard/`examples/agent-plugins` use **`iomesh-memory-mcp` only** · aion broker/CP private · dual_write OFF · not Memory GA · no invent platform GA from rename.
- **s1525:** advanced Memory install ladder for TUI — optional ONNX on host maximizes semantic; Qdrant not required / not wired lean host search · dual_write OFF · not Memory GA · [memory-advanced-install.md](./memory-advanced-install.md).
