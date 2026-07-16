# Stage mesh dogfood

Operator smoke for **I/O Mesh** integration from the public `iomesh-tui` harness.

## Checks

| Step | Request | Soft (default) | Strict (`--strict`) |
|------|---------|----------------|---------------------|
| enabled | config | SKIP if disabled | same |
| health | `GET /health` | **FAIL** if down | **FAIL** |
| ready | `GET /ready` or `/readyz` | SKIP if 404 | FAIL if missing/error |
| context | `POST /v1/context/query` | SKIP if empty (fail-open) | FAIL if empty |
| emit | `POST /v1/streams/dept` | SKIP on error | FAIL on error |
| policy | `POST /v1/policy/evaluate` | SKIP if mode off / 404 / fail-open | FAIL if mode on and evaluate soft-fails |
| catalog | broker + portal list | SKIP if plane off / fail-open | FAIL if fail-open |
| memory_ingest | `POST /v1/streams/MEMORY_INGEST/publish` | SKIP on error (`--skip-memory` forces SKIP) | FAIL on error |

Context requests set `include_lineage` when configured (lineage count shown on PASS detail).

### memory_ingest (dual-write probe)

Included **by default** when mesh is enabled (not gated on agent `[memory].dual_write`). Calls the same lean path as Phase 2 dual-write (`PublishMemoryIngest`):

- Subject: `{tenant}.memory.ingest.turn`
- Envelope: `type=memory_ingest`, `role=tool`, `content=iomesh-tui dual-write dogfood`, `event_time=now`, `session_seq=1`
- Soft mode: publish/transport errors → **SKIP** (fail-open); `--strict` → **FAIL**
- Offline / mesh disabled: whole report is SKIP (no memory step)
- **PASS detail** includes stream, subject, and seq when available. When Client `[iomesh] org` / `workspace` (`OrgID` / `WorkspaceID`) are set, detail also appends `org=…` and/or `workspace=…` as operator-visible evidence that dual-write publish used those headers (`X-IOMesh-Org` / `X-IOMesh-Workspace`). Empty values are omitted (no `org=` token).

Final line: `RESULT=PASS …` or `RESULT=FAIL …`.

## CLI

```bash
export IOMESH_ENDPOINT=https://mesh.stage.example
# Or control-plane / portal edge for catalog federation:
# export IOMESH_ENDPOINT=https://cp.stage.example
export IOMESH_API_KEY=…          # optional
export IOMESH_TENANT=acme        # optional

iomesh mesh dogfood
iomesh mesh dogfood --strict
iomesh mesh dogfood --json       # stage CI evidence
iomesh mesh dogfood --endpoint "$IOMESH_ENDPOINT" --tenant acme
iomesh mesh dogfood --skip-context --skip-emit --skip-memory   # health-only-ish
iomesh mesh catalog              # broker then portal paths
```

## Script / Make

```bash
./scripts/mesh_dogfood.sh              # live
./scripts/mesh_dogfood.sh --strict
./scripts/mesh_dogfood.sh --unit       # go test ./internal/iomesh (CI)

make dogfood
make dogfood-unit
```

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | `RESULT=PASS` or mesh disabled SKIP (offline-first) |
| 1 | any hard FAIL |

## Package

- `internal/iomesh/dogfood.go` — `Client.Dogfood`, `Ready`, `EmitErr`, `PublishMemoryIngest` step, `FormatReport`
- `internal/iomesh/memory.go` — dual-write publish wire format
- `cmd/iomesh` — `mesh dogfood|probe`
