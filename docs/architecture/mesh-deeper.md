# Deeper I/O Mesh (lineage · policy · metering)

Extends the offline-first mesh client beyond health/context/emit dogfood.

## Lineage-aware context

When `include_lineage = true` (default) the client POSTs:

```http
POST /v1/context/query
{ "tenant", "workspace", "query", "limit", "include_lineage": true }
```

Response shapes accepted:

```json
{ "text": "…", "lineage": [{ "id", "product", "subject", "source", "freshness" }] }
```

or `{ "items": [ { "text", "lineage" }, … ] }`.

Injected system block:

```xml
<iomesh-context>
…text…
<iomesh-lineage>
- dp-1 · subject=dept.eng.events · source=kafka · freshness=2m
</iomesh-lineage>
</iomesh-context>
```

Fail-open: empty string on transport/HTTP/decode errors.

## Policy gates (Rego / broker evaluate)

Config / env:

| Value | Behaviour |
|-------|-----------|
| `off` (default) | no remote calls |
| `advisory` | `POST /v1/policy/evaluate`; log `EventMeshPolicy` + dept emit; **never block** |
| `enforce` | same, but **deny tool** when mesh returns `allow:false` |

Fail-open: transport errors, non-OK (except explicit 200 deny), and **404** (`source=unavailable`) never block tools.

Agent order: **mesh policy → interactive approval → execute**.

## Local metering “dashboard”

`Client` implements `router.MetricsSink` and keeps an **in-process** rollup (`UsageMeter`):

- `iomesh mesh usage` — print table for the **current process** (empty in a fresh CLI)
- `iomesh mesh usage --json` — same snapshot as indented JSON (stage scrapers / CI)
- After agent LLM calls, totals accumulate; emit still goes to `dept.agent.llm_call` when mesh is enabled
- **Not** a remote multi-tenant dashboard UI — that lives on the platform. This CLI surface is operator-local cost telemetry only.

## Remote metering path (platform dashboards)

When `[iomesh]` is enabled and `emit_dept_streams = true` (default):

1. Each LLM call → local `UsageMeter` **and** `POST /v1/streams/dept` with `type=dept.agent.llm_call`
2. Request headers: `X-IOMesh-Org` / `X-IOMesh-Workspace` when `[iomesh] org` / `workspace` are set (PlanGate / multi-tenant attribution)
3. Payload includes `tenant`, `org`, `workspace`, token counts, `est_usd`, model ids (errors redacted)

Stage smoke:

```bash
export IOMESH_ENDPOINT=…
export IOMESH_ORG=org_…
export IOMESH_WORKSPACE=ws_…
iomesh mesh dogfood --json
# steps emit + llm_meter PASS with org= / workspace= / session_id=
```

Dogfood step **`llm_meter`** publishes a zero-token probe event of the same shape so CI can prove the remote path without an LLM round-trip. Soft-SKIP on transport unless `--strict`.

## Config

```toml
[iomesh]
include_lineage = true
policy_mode = "off"   # off | advisory | enforce
emit_dept_streams = true
# org = "org_…"        # X-IOMesh-Org on dept emit + memory
# workspace = "ws_…"   # X-IOMesh-Workspace
```

Env: `IOMESH_INCLUDE_LINEAGE`, `IOMESH_POLICY_MODE`, `IOMESH_ORG`, `IOMESH_WORKSPACE`.

## Dogfood

`iomesh mesh dogfood` adds:

- **policy** when mode ≠ off (SKIP on 404/fail-open unless `--strict`)
- **llm_meter** when dept streams enabled (same soft/strict matrix as **emit**)

## Catalog composition + portal federation

When `catalog_plane = true` (default), discovery tries **broker then portal** (aion control plane):

| Order | Path | Source label |
|------|------|--------------|
| 1 | `GET /v1/catalog/data-products` | `mesh` |
| 2 | `GET /v1/catalog/products` | `mesh` |
| 3 | `GET /v17/portal/catalog/data-products` | `portal` |
| 4 | `GET /v16/portal/catalog/marketing/data-products` | `portal` |

Portal JSON fields (`mesh_layer`, `subject_pattern`, `sample_subjects`, `summary`) normalize into the shared product shape.

| Surface | Behaviour |
|---------|-----------|
| CLI `iomesh mesh catalog [--query q]` | Operator table (`source=mesh\|portal`) |
| TUI `/catalog [query]` | Same |
| Agent `list_mesh_catalog` / `get_mesh_catalog_product` / `mesh_status` | Read-only |
| `inject_catalog = true` | Per-turn `<iomesh-catalog>` system block (opt-in) |
| Dogfood **catalog** step | PASS for mesh **or** portal; soft-skip on 404 |
| Dogfood `--json` | Machine-readable report for stage CI |

## Packages

- `internal/iomesh/client.go` — QueryContext, lineage format, meter hook
- `internal/iomesh/policy.go` — EvaluatePolicy
- `internal/iomesh/meter.go` — UsageMeter / FormatUsage
- `internal/iomesh/catalog.go` — ListCatalog / FormatCatalog / CatalogSnippet
- `internal/agent` — policy before tool execute; mesh catalog tools; `EventMeshPolicy`
