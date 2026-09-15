# Ops department RCA kit (private overlay)

Sample UTF-8 corpus for **department temporal RCA** (V1.6 D5). Import is **private overlay** only. Do **not** stamp mesh on these files. Mesh miss is success. Ingest after `examples/dept-rca/support` into **one** palace = one-tenant composition (≠ two-org · not federated org search · **not** an org-wide RCA engine · **D5 ≠ E-G1**).

```bash
iomesh memory ingest-dir --yes examples/dept-rca/ops
# inventory only:
iomesh memory ingest-dir --dry-run examples/dept-rca/ops
# slash twin:
# /memory ingest-dir examples/dept-rca/ops --dry-run
```

`session_id` mints as `local-overlay` when the walk has none (`--department ops` mints `local-overlay:ops`). `--source-hint` is **private** only (`source_hint=private`; `mesh` is an error). dual_write stays **OFF**. Catalog list ≠ consume.

## Files

| File | Role |
|------|------|
| `page.md` | PagerDuty-shaped sample page **PD-HMAC-5xx** (HMAC 5xx). Created `2026-06-15T14:08:00Z`. |
| `runbook.md` | Residual-honest TTFH RCA fact: **HMAC 200 is not a consume receipt**. Extend this incident; do not invent a new one. |
| `deploy-note.md` | Consumer 500 deploy note in the same incident window. Human `decision_stub` only — never YAML APPLY. |

`dept.ops.events.*` is **routing**, not Connected. A PagerDuty-shaped filename is not a PagerDuty install.

## After ingest (R2 companion)

```bash
/memory digest --require-sources mesh,private
# miss is success without pull — do not invent mesh cite-both from this kit
/memory facts-as-of --as-of 2026-06-15T14:08:00Z [--department ops]
```

Temporal ask: what was the HMAC/consume rule **as-of page created_at** `2026-06-15T14:08:00Z`? HMAC 200 is not a consume receipt. Cite-both vs entitled `dept.ops` heartbeats is **after R4 pull**, not this overlay.

**not E-G1.** **not Memory GA.** 1.6 E-G1 **Parked**. **D5 ≠ E-G1**. empty until consume · CLIENT ≠ PULSE · catalog ≠ Connected · dual_write **OFF** · never YAML APPLY.
