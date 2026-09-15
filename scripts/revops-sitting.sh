#!/usr/bin/env bash
# V2-C RevOps sitting recipe (support.theme). Copy/code sitting — laptop sitting stays unchecked.
#
# This sitting is not E-G1. leftover_is_bind OPEN.
# dual_write OFF · not Memory GA · catalog ≠ Connected · catalog ≠ heartbeat
# EMPTY until consume · CLIENT ≠ PULSE · overlay does not GET CRM
# --live is not overlay PULSE (not this sitting).
# summarize_account_health is not a health score · linked_pr_miss ≠ MTTR
# One stream noun: support.theme (dept.gtm.support.theme).
# Hands (win-back, price change) stay off this plane.
#
# Prefer `iomesh` on PATH, else `go run ./cmd/iomesh` from repo root.
# Default is ingest-dir --dry-run of the CS kit (living memo). --unit echoes
# the walk only. --yes is mutating overlay ingest, still not overlay PULSE.
#
# Usage:
#   ./scripts/revops-sitting.sh
#   ./scripts/revops-sitting.sh --unit
#   ./scripts/revops-sitting.sh --dry-run
#   ./scripts/revops-sitting.sh --yes
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

KIT="examples/dept-rca/customer_success"
UNIT=0
DRY=1
YES=0

usage() {
  cat <<'EOF'
revops-sitting: V2-C support.theme (not E-G1 · leftover_is_bind OPEN)

Usage:
  ./scripts/revops-sitting.sh            dry-run ingest-dir of CS living memo
  ./scripts/revops-sitting.sh --unit     echo walk only (no MCP)
  ./scripts/revops-sitting.sh --dry-run  same as default
  ./scripts/revops-sitting.sh --yes      mutating ingest-dir --yes (not overlay PULSE)

--live is not overlay PULSE and is not this sitting.
Prefer iomesh on PATH, else go run ./cmd/iomesh.
EOF
}

for arg in "$@"; do
  case "$arg" in
    --unit)
      UNIT=1
      DRY=0
      YES=0
      ;;
    --dry-run|--dryrun)
      DRY=1
      YES=0
      UNIT=0
      ;;
    --yes)
      YES=1
      DRY=0
      UNIT=0
      ;;
    --live)
      echo "revops-sitting: --live is not overlay PULSE · not this sitting · skip" >&2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "revops-sitting: unknown arg $arg" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if command -v iomesh >/dev/null 2>&1; then
  IOMESH=(iomesh)
else
  IOMESH=(go run ./cmd/iomesh)
fi

echo "revops-sitting: V2-C support.theme · not E-G1 · leftover_is_bind OPEN · not Memory GA"
echo "revops-sitting: dual_write OFF · catalog ≠ Connected · catalog ≠ heartbeat · EMPTY until consume · CLIENT ≠ PULSE"
echo "revops-sitting: overlay does not GET CRM · never mesh stamp · living memo in ${KIT}"
echo "revops-sitting: --live is not overlay PULSE · laptop sitting stays unchecked"
echo "revops-sitting: via ${IOMESH[*]}"

if [[ "$YES" -eq 1 ]]; then
  INGEST_FLAGS=(memory ingest-dir --yes --department customer_success --source-hint private "${KIT}")
else
  INGEST_FLAGS=(memory ingest-dir --dry-run --department customer_success --source-hint private "${KIT}")
fi

echo "revops-sitting: ingest-dir ${INGEST_FLAGS[*]}"
if [[ "$UNIT" -eq 0 ]]; then
  "${IOMESH[@]}" "${INGEST_FLAGS[@]}"
else
  echo "revops-sitting: --unit · skip ingest-dir call · echo walk only"
fi

echo
echo "digest (cite-both or named miss · miss is success · ingest-dir is enough):"
echo "  /memory digest --require-sources mesh,private"
echo "ACK (existing · no send/pay/ship):"
echo "  /dashboard ack"
echo "  digest miss ≠ known · ACK via /dashboard ack (local ritual · no send/pay/ship) · local RCA stays on disk"
echo
echo "heartbeat vs catalog:"
echo "  catalog ≠ Connected · catalog ≠ heartbeat · EMPTY until consume · CLIENT ≠ PULSE"
echo "  this sitting is not E-G1 · leftover_is_bind OPEN · dual_write OFF · not Memory GA"

exit 0
