# Support department RCA kit (private overlay)

Sample UTF-8 corpus for **department temporal RCA** (V1.6 D1). Import is **private overlay** only. Do **not** stamp mesh on these files. Mesh miss is success.

```bash
iomesh memory ingest-dir --yes examples/dept-rca/support
# inventory only:
iomesh memory ingest-dir --dry-run examples/dept-rca/support
# slash twin:
# /memory ingest-dir examples/dept-rca/support --dry-run
```

`session_id` mints as `local-overlay` when the walk has none (`--department support` mints `local-overlay:support`). `--source-hint` is **private** only (`mesh` is an error). dual_write stays **OFF**. Catalog list ≠ consume.

## Files

| File | Role |
|------|------|
| `ticket-export.md` | Zendesk-shaped sample ticket **ZD-1001** (unused-seat refund). Created `2026-06-15T14:22:00Z`. |
| `policy.md` | 14-day unused-seat refund policy, effective **2026-01-01**. |
| `macro.md` | Macro that points at `policy.md`. Human `decision_stub` only — never YAML APPLY. |

`dept.support.events.*` is **routing**, not Connected. A Zendesk-shaped filename is not a Zendesk install.

## After ingest (R2 companion)

```bash
/memory digest --require-sources mesh,private
# miss is success without pull — do not invent mesh cite-both from this kit
/memory facts-as-of --as-of 2026-06-15T14:22:00Z
```

Temporal ask: what was the unused-seat refund rule **as-of ticket created_at** `2026-06-15T14:22:00Z`? Policy effective 2026-01-01 was in force. Cite-both vs entitled `dept.support` heartbeats is **after R4 pull**, not this overlay.

**not E-G1.** **Cloud Memory GA.** 1.6 E-G1 **Parked**. empty until consume · CLIENT ≠ PULSE · catalog ≠ Connected · dual_write **OFF** · never YAML APPLY.
