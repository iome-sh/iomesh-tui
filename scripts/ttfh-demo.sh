#!/usr/bin/env bash
# Residual-honest TTFH demo (V1.5 tracker 5.4).
#
# not E-G1 · not Memory GA · dual_write OFF · never invent PULSE/Connected
# empty until consume · CLIENT ≠ PULSE
#
# Always runs `iomesh ttfh --unit` (offline). If IOMESH_ENDPOINT is set, also
# runs `iomesh ttfh --live` (fail-open). Exit 0 if unit succeeded even when
# live is EMPTY/unreachable.
#
# Prefer `iomesh` on PATH, else `go run ./cmd/iomesh` from repo root.
# Never curl APPLY. Never set dual_write ON.
#
# Usage:
#   ./scripts/ttfh-demo.sh
#   IOMESH_ENDPOINT=https://hooks.example ./scripts/ttfh-demo.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if command -v iomesh >/dev/null 2>&1; then
  IOMESH=(iomesh)
else
  IOMESH=(go run ./cmd/iomesh)
fi

echo "ttfh-demo: residual-honest · not E-G1 · not Memory GA · dual_write OFF"
echo "ttfh-demo: never invent PULSE/Connected · empty until consume · CLIENT ≠ PULSE"
echo "ttfh-demo: unit (offline) via ${IOMESH[*]}"

set +e
unit_out="$("${IOMESH[@]}" ttfh --unit 2>&1)"
unit_code=$?
set -e
printf '%s\n' "$unit_out"

if [[ "$unit_code" -ne 0 ]]; then
  echo "ttfh-demo: unit FAIL (exit $unit_code)" >&2
  exit "$unit_code"
fi

if [[ -n "${IOMESH_ENDPOINT:-}" ]]; then
  echo "ttfh-demo: live (fail-open · never invent PULSE) · IOMESH_ENDPOINT set"
  set +e
  live_out="$("${IOMESH[@]}" ttfh --live 2>&1)"
  live_code=$?
  set -e
  printf '%s\n' "$live_out"
  if [[ "$live_code" -ne 0 ]]; then
    echo "ttfh-demo: live EMPTY/unreachable (fail-open · unit already succeeded · not PULSE)" >&2
  fi
else
  echo "ttfh-demo: IOMESH_ENDPOINT unset · skip --live · EMPTY until consume · not PULSE"
fi

echo
echo "Walk (one list · mesh not required for R0–R2 · do not invent PULSE / Connected / Memory GA / dual_write ON / APPLY):"
echo "  R0  iomesh ttfh --unit                 offline · no mesh · not E-G1"
echo "  R1  optional iomesh ttfh --live        fail-open probe · EMPTY unless decoded · not overlay PULSE"
echo "      Optional mesh: IOMESH_ENDPOINT for R1 only. Overlay /dashboard consume is R3, not this."
echo "  R2  ingest: iomesh memory ingest   # three RCA-shaped turns, source_hint=private"
echo "      digest cite-both-or-miss: /memory digest --require-sources mesh,private"
echo "      patterns (Beta): /memory patterns [--limit N]   # empty ≠ invent · never APPLY"
echo "      facts-as-of: /memory facts-as-of --as-of <RFC3339>   # palace SoT · not Memory GA"
echo "  R3  /dashboard consume                 entitled overlay PULSE (parked · required for E-G1)"
echo "  R4  after PULSE (optional): iomesh memory pull   # mesh → local palace · dual_write OFF · pull ≠ invent Connected"

exit 0
