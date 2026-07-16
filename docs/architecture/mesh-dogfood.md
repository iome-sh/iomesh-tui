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

Context requests set `include_lineage` when configured (lineage count shown on PASS detail).

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
iomesh mesh dogfood --skip-context --skip-emit   # health-only
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

- `internal/iomesh/dogfood.go` — `Client.Dogfood`, `Ready`, `EmitErr`, `FormatReport`
- `cmd/iomesh` — `mesh dogfood|probe`
