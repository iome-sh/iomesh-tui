# Memory Palace + temporal MCP

First-class **Agentic Memory Palace** and **temporal recall** for `iomesh-tui`, without embedding Palace inside the TUI process.

Platform ships `aion-memory-mcp` (stdio **and** streamable HTTP) with tools:

| Tool | Purpose |
|------|---------|
| `memory_ingest_turn` | Persist a conversation turn (tiered Palace) |
| `memory_retrieve` | Query memories (optional `session_id`, `since`/`until`, `session_seq`) |
| `memory_timeline` | Temporal timeline slice |
| `memory_search_semantic` | Semantic facts |
| patterns / anomalies / compact | Ops helpers |

Resources: `memory://{tenant}/…` (stats, timeline, session turns, facts).

## Phases

| Phase | Status | Work |
|-------|--------|------|
| **0** | **done** | Attach `aion-memory-mcp` via existing MCP client; documented example |
| **1** | **done** | `[memory]` auto-recall inject, opt-in auto-ingest, `/memory` slash |
| **2** | **done (v0.3.0)** | HTTP MCP primary path + optional dual-write to mesh `MEMORY_INGEST` |
| **3 partial** | **done (dogfood)** | Async `MEMORY_RPC` recall probe (`PublishMemoryRecall`) |
| **3** | **done (v0.4.0 dogfood)** | Sync `RetrieveMemory` → `POST /v1/memory/retrieve` (+ `/v5` fallback) + dogfood `memory_retrieve` step |
| **3+** | **done (v0.4.0 agent)** | Agent auto-recall + `/memory recall` prefer sync HTTP when mesh **or** `[memory].endpoint` sidecar is set; MCP fallback |
| **3 temporal (s1068)** | **done** | Sync retrieve options: `since`/`until`/`session_seq` on lean HTTP + auto-recall config + `/memory recall` flags |
| **4 pull (s652)** | **done (M1)** | `iomesh memory pull` — durable mesh consumer → local MCP `memory_ingest_turn` (cost-max local palace; dual_write remains optional audit) |

**Cost-max (s650+):** primary Memory UX is **local Palace** (this TUI + `aion-memory-mcp`). Mesh is **pull egress** of ops events; hosted cloud Palace is **sunset until scale**. Dual-write = optional **audit** only (default OFF).

### Local edge stack (s761 product · s765 completeness pin)

End-to-end **cost-max** path (Beta · local only · not platform GPU · not invent GA):

```text
I/O Mesh cloud (pull egress)  ↔  iomesh-tui  ↔  local memory MCP (Palace)  ↔  local AI (Ollama pin)
```

| Layer | Role | Honesty |
|-------|------|---------|
| Mesh pull | Ops event egress into local palace (`iomesh memory pull`) | Cloud mesh ≠ local AI |
| Local MCP Palace | Primary memory UX (`aion-memory-mcp`) | Hosted Palace sunset until scale |
| Ollama pin | Local LLM (`-m ollama-llama3.2` / `IOMESH_DEFAULT_MODEL`) | OpenAI-compat `/v1` only; $0 catalog tier; not cascade default |
| Dual-write | Optional mesh audit | **Default OFF** |

See [llm-cascade.md](./llm-cascade.md) for Ollama install (`ollama serve` / `ollama pull llama3.2`) and env overrides (`OLLAMA_URL` / `OLLAMA_HOST`).

**s765 (Beta · completeness pin):** local edge stack **complete** after s761 first-class Ollama product Shipped — inventory (mesh pull egress ↔ TUI ↔ local memory MCP ↔ Ollama pin) locked by docs + unit tests (`TestDefaultModels_OllamaLocalEdgeCompletenessPin`: catalog name `ollama-llama3.2`, model id `llama3.2`, cost 0, caps `local`+`ollama`, priority after DeepSeek cascade / Premium). Completeness pin **does not** invent new catalog models and **does not** re-claim s761 product body. Peer aion **s764** heartbeat-local-edge-path residual. Local AI ≠ platform GPU · dual_write OFF · Palace sunset · offline unit ≠ live Ollama green · catalog pin ≠ cascade default · not full mesh RBAC GA · no invent GA.

#### Local-primary LT honesty (s768 pin)

Honesty pin so TUI surfaces agree with the **cost-max local-primary** charter after s761/s765 product bodies (this serial is docs/tests only — **does not** re-claim s761 Ollama product or s765 completeness inventory):

| Claim | Honesty |
|-------|---------|
| **Long-term / temporal memory** | Customer **local palace** — local MCP (`aion-memory-mcp`) + optional local AI (Ollama pin). Not hosted cloud GPU palace. |
| **Mesh heartbeats** | **Pull egress** into local palace (`iomesh memory pull`) — not a push-to-cloud-palace product path. |
| **dual_write** | Optional mesh audit only · **default OFF** (`[memory].dual_write` / `IOMESH_MEMORY_DUAL_WRITE`). |
| **$119 language** (if any) | **Memory Ops Pack** (pull / retain / audit / support) — **not** hosted cloud GPU palace. |
| **Local AI** | Customer-edge Ollama only · **≠ platform GPU**. |
| **Hosted Palace** | **Sunset** until scale. |
| **Does not invent** | GA · freemium unlimited palace · platform GPU · full mesh RBAC GA. |

Peer aion **s767** bi-heartbeats local-primary honesty continuum. Unit pin: `TestDefaultMemoryConfig` / config `Default().Memory.DualWrite == false` (s768). Offline unit ≠ live APPLY · Beta · no invent GA.

#### Naming honesty (s771 pin)

Docs **naming honesty** residual after aion **s770** usage-rates Ops Pack honesty (this serial is docs/tests only — **does not** re-claim s761 Ollama product, s765 local-edge completeness, or s768 local-primary LT pin bodies):

| Name / phrase | Honest meaning in this TUI |
|---------------|----------------------------|
| **“Memory Palace”** | **Local-primary customer-edge palace** — MCP (`aion-memory-mcp`) + mesh **pull** into local store. **Not** multi-tenant hosted cloud GPU / Qdrant Palace fleet. |
| **$119 / platform SKU** (if mentioned) | **Memory Ops Pack** — pull / Extended retain / audit / support entitlement. **Not** hosted GPU palace fleet. |
| **dual_write** | Optional mesh audit only · **default OFF**. |
| **Hosted Palace** | **Sunset** until scale. |
| **Local AI** | Customer-edge Ollama · **≠ platform GPU**. |
| **Does not invent** | GA · freemium unlimited palace · platform GPU · full mesh RBAC GA. |

Peer aion **s770** usage-rates Ops Pack residual · prior TUI **s768** local-primary LT honesty. dual_write OFF · Palace sunset · no invent GA · offline unit ≠ live APPLY · Beta.

#### Buyer claim pin (s774)

OSS / buyer-claim boundary pin peer aion **s773** cost-max claim suite expand (this serial is docs/tests only — **does not** re-claim s761 Ollama product, s765 local-edge completeness, s768 local-primary LT, or s771 naming honesty bodies):

| Surface | May claim | Must not claim |
|---------|-----------|----------------|
| **Public MIT `iomesh-tui`** | OSS agent harness + optional mesh **client** surface (SDK elsewhere) | Hosted multi-tenant mesh **control plane** shipped inside free TUI |
| **Memory** | Local palace via MCP (`aion-memory-mcp`); mesh **pull** into local store | Multi-tenant hosted / freemium cloud palace included with free TUI |
| **$119 / platform SKU** (if mentioned) | **Memory Ops Pack** (pull / retain / audit / support) | Cloud GPU / hosted Palace fleet as the $119 product |
| **Local AI** | Customer-edge Ollama pin | Platform GPU from Ollama pin |
| **dual_write** | Optional mesh audit · **default OFF** | dual_write as primary palace path |
| **Hosted Palace** | Sunset until scale | Invent GA · freemium unlimited palace · full mesh RBAC GA |

Peer aion **s773** cost-max claim suite expand · prior TUI **s771** naming · **s768** local-primary. dual_write OFF · Palace sunset · local AI ≠ platform GPU · no invent GA · offline unit ≠ live APPLY · Beta. Unit pin peer: `TestDefault_DualWriteOff` / `TestDefaultMemoryConfig_DualWriteOff` (s768 body + s771/s774 comment peers).

**Non-goals:** private monorepo imports in public TUI; embedding Qdrant/Palace in-process; dependency on `iomesh-client-sdk-go`.

## Public Go SDK

Operators who need the **full** client surface (beyond this TUI’s lean HTTP/MCP path) should use the public module **[iomesh-client-sdk-go](https://github.com/iome-sh/iomesh-client-sdk-go)**:

| Capability | In the SDK | In iomesh-tui |
|------------|------------|---------------|
| **M2** sync retrieve | `RetrieveMemory` / memory helpers | Lean `POST /v1/memory/retrieve` (+ `/v5`) in `internal/iomesh` |
| **M3** temporal envelope | Full temporal fields on publish | Dual-write mirrors a subset (`event_time`, `session_seq`, …) |
| Multi-tenant workspace | `WithWorkspace` (and related options) | Optional org/workspace headers when configured |

**iomesh-tui stays lean:** no `github.com/iome-sh/iomesh-client-sdk-go` module dependency. Memory dual-write and sync retrieve mirror SDK wire shapes over plain HTTP so the agent harness remains a thin, zero-SDK client. Prefer the public SDK for custom Go services, stage gate jobs, or anything that should track the full client API.

## Phase 0–1 — MCP hooks (stdio or HTTP)

### Preferred: streamable HTTP (platform M1)

```bash
aion-memory-mcp -http-addr :8080 -palace-root /data/memory-palaces
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
aion-memory-mcp -http-addr :8080 -palace-root ~/.iomesh/palace

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
command = "aion-memory-mcp"   # platform Memory MCP server binary name
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
| `/memory` | Status: enabled, `mcp=`, `sync_http=`, flags (incl. `dual_write`), tenant |
| `/memory recall [query]` | Sync HTTP retrieve when mesh enabled, else MCP (default query = last user text or `"*"`) |
| `/memory recall --since|--until|--session-seq … [query]` | Same + per-call temporal filters (s1068; override config) |
| `/memory ingest <text>` | Ingest a user turn (MCP and/or dual-write) |

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
| `internal/iomesh/memory.go` | `PublishMemoryIngest`, `PublishMemoryRecall`, `RetrieveMemory` / `RetrieveMemoryWithOptions` lean HTTP (no SDK dep; s1068 temporal fields) |
| `internal/agent/memory.go` | Recall (sync prefer → MCP; config + opts temporal filters) / ingest / dual-write helpers |
| `internal/agent/agent.go` | `RunTurn` hooks |
| `internal/tui/tui.go` | `/memory` slash |
| `configs/config.example.toml` | Copy-paste wire-up |

## Honesty

- Local Palace via stdio/HTTP MCP or lean sidecar HTTP ≠ multi-tenant Cloud Run Memory Palace with full entitlements.
- Dual-write is **best-effort** stream publish; it does not guarantee Palace persistence by itself.
- “Native Vertex” / G4S claims are separate (see marketing claim matrix); memory is **Palace via MCP and/or lean HTTP sidecar**, not Vertex.
- Do not claim temporal pipeline is live unless stage/prod embedding + temporal flags are on.
