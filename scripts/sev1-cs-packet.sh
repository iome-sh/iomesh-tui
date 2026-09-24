#!/usr/bin/env bash
# V2-D Sev-1 CS packet overlay (PD-HMAC-5xx). Copy/code overlay — CS sitting stays unchecked.
#
# This overlay is not E-G1. leftover_is_bind OPEN.
# dual_write OFF · Cloud Memory GA · catalog ≠ Connected · catalog ≠ heartbeat
# EMPTY until consume · CLIENT ≠ PULSE · overlay does not GET CRM
# --live is not overlay PULSE (not this overlay).
# Zendesk pulse does not exist · V2-F parked · D5b/D6 parked · V2 ≠ E-G1
# Joins ops incident PD-HMAC-5xx (event_time 2026-06-15T14:08:00Z).
# ACC-1001 / ZD-1001 are private pointers only.
#
# Prefer `iomesh` on PATH, else `go run ./cmd/iomesh` from repo root.
# Default is ingest-dir --dry-run of the CS kit (sev1 packet). --unit echoes
# the walk only. --yes is mutating overlay ingest, still not overlay PULSE.
#
# Usage:
#   ./scripts/sev1-cs-packet.sh
#   ./scripts/sev1-cs-packet.sh --unit
#   ./scripts/sev1-cs-packet.sh --dry-run
#   ./scripts/sev1-cs-packet.sh --yes
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

KIT="examples/dept-rca/customer_success"
UNIT=0
DRY=1
YES=0

usage() {
  cat <<'EOF'
sev1-cs-packet: V2-D PD-HMAC-5xx (not E-G1 · leftover_is_bind OPEN)

Usage:
  ./scripts/sev1-cs-packet.sh            dry-run ingest-dir of CS sev1 packet
  ./scripts/sev1-cs-packet.sh --unit     echo walk only (no MCP)
  ./scripts/sev1-cs-packet.sh --dry-run  same as default
  ./scripts/sev1-cs-packet.sh --yes      mutating ingest-dir --yes (not overlay PULSE)

--live is not overlay PULSE and is not this overlay.
Prefer iomesh on PATH, else go run ./cmd/iomesh.
Zendesk pulse does not exist · V2-F parked · CS sitting stays unchecked.
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
      echo "sev1-cs-packet: --live is not overlay PULSE · not this overlay · skip" >&2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "sev1-cs-packet: unknown arg $arg" >&2
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

echo "sev1-cs-packet: V2-D PD-HMAC-5xx · not E-G1 · leftover_is_bind OPEN · Cloud Memory GA"
echo "sev1-cs-packet: dual_write OFF · catalog ≠ Connected · catalog ≠ heartbeat · EMPTY until consume · CLIENT ≠ PULSE"
echo "sev1-cs-packet: overlay does not GET CRM · never mesh stamp · sev1-packet.md in ${KIT}"
echo "sev1-cs-packet: --live is not overlay PULSE · CS sitting stays unchecked"
echo "sev1-cs-packet: Zendesk pulse does not exist · V2-F parked"
echo "sev1-cs-packet: via ${IOMESH[*]}"

if [[ "$YES" -eq 1 ]]; then
  INGEST_FLAGS=(memory ingest-dir --yes --department customer_success --source-hint private "${KIT}")
else
  INGEST_FLAGS=(memory ingest-dir --dry-run --department customer_success --source-hint private "${KIT}")
fi

echo "sev1-cs-packet: ingest-dir ${INGEST_FLAGS[*]}"
if [[ "$UNIT" -eq 0 ]]; then
  "${IOMESH[@]}" "${INGEST_FLAGS[@]}"
else
  echo "sev1-cs-packet: --unit · skip ingest-dir call · echo walk only"
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
echo "  this overlay is not E-G1 · leftover_is_bind OPEN · dual_write OFF · Cloud Memory GA"

exit 0
