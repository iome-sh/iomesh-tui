#!/usr/bin/env bash
# Stage I/O Mesh dogfood for iomesh-tui.
#
# Probes: health → ready → context plane → dept emit.
# Exit 0 when RESULT=PASS (or mesh disabled SKIP offline-first).
# Exit 1 on FAIL.
#
# Env (stage):
#   IOMESH_ENDPOINT          required for live probe (e.g. https://mesh.stage.example)
#   IOMESH_MEMORY_ENDPOINT   optional memory sidecar for sync memory_retrieve (warm plane)
#   MEMORY_SIDECAR_URL       alias for IOMESH_MEMORY_ENDPOINT when that is unset
#   IOMESH_API_KEY           optional Bearer
#   IOMESH_TENANT            optional tenant header
#   IOMESH_CONFIG            optional config.toml
#
# Usage:
#   ./scripts/mesh_dogfood.sh              # live (uses env/config)
#   ./scripts/mesh_dogfood.sh --strict     # require context+emit+ready
#   ./scripts/mesh_dogfood.sh --unit       # offline unit tests only (CI)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

STRICT=0
UNIT=0
EXTRA=()
for arg in "$@"; do
  case "$arg" in
    --strict) STRICT=1 ;;
    --unit|--mock) UNIT=1 ;;
    -h|--help)
      sed -n '1,25p' "$0"
      exit 0
      ;;
    *) EXTRA+=("$arg") ;;
  esac
done

if [[ "$UNIT" -eq 1 ]]; then
  echo "mesh dogfood: unit mode (go test internal/iomesh)"
  go test ./internal/iomesh/ -count=1 -timeout=60s
  echo "RESULT=PASS (unit)"
  exit 0
fi

if [[ -z "${IOMESH_ENDPOINT:-}" ]] && [[ -z "${IOMESH_CONFIG:-}" ]]; then
  echo "warning: IOMESH_ENDPOINT unset — dogfood will SKIP if config has no mesh endpoint" >&2
fi

ARGS=(mesh dogfood)
if [[ "$STRICT" -eq 1 ]]; then
  ARGS+=(--strict)
fi
ARGS+=("${EXTRA[@]+"${EXTRA[@]}"}")

if [[ -x ./bin/iomesh ]]; then
  BIN=./bin/iomesh
else
  BIN="go run ./cmd/iomesh"
fi

set +e
# shellcheck disable=SC2086
out="$($BIN "${ARGS[@]}" 2>&1)"
code=$?
set -e
printf '%s\n' "$out"

if [[ $code -ne 0 ]]; then
  echo "RESULT=FAIL (exit $code)" >&2
  exit 1
fi
if echo "$out" | grep -q '^RESULT=FAIL'; then
  exit 1
fi
exit 0
