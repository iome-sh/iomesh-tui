# Customer success department RCA kit (private overlay)

Sample UTF-8 corpus for **department temporal RCA** (V1.6 D5d). Import is **private overlay** only. Do **not** stamp mesh on these files. Mesh miss is success. Ingest after `examples/dept-rca/support`, `examples/dept-rca/ops`, and `examples/dept-rca/sales` into **one** palace = one-tenant composition (≠ two-org · not a Salesforce Connected install · **not** an org-wide RCA engine · **D5d ≠ E-G1**). Overlay does **not** GET Salesforce/CRM.

```bash
iomesh memory ingest-dir --yes examples/dept-rca/customer_success
# inventory only:
iomesh memory ingest-dir --dry-run examples/dept-rca/customer_success
# slash twin:
# /memory ingest-dir examples/dept-rca/customer_success --dry-run
```

`session_id` mints as `local-overlay` when the walk has none (`--department customer_success` mints `local-overlay:customer_success`). `--source-hint` is **private** only (`source_hint=private`; `mesh` is an error). dual_write stays **OFF**. Catalog list ≠ consume.

## Files

| File | Role |
|------|------|
| `health-note.md` | Account-health snapshot **ACC-1001**. Dated **2026-08-15**. Not a live Salesforce/CS install. |
| `renewal.md` | Renewal window **2026-09-01**. Entitled **seats as a count** (not ARR). Human `decision_stub` only — never YAML APPLY. |
| `playbook.md` | CS playbook that **points at** `health-note.md`. Never auto-APPLY a save/renew. |

`dept.customer_success.events.*` is **routing**, not Connected. An account-health snapshot is not a Salesforce/CS install.

## After ingest (R2 companion)

```bash
/memory digest --require-sources mesh,private
# miss is success without pull — do not invent mesh cite-both from this kit
/memory facts-as-of --as-of 2026-08-31T18:00:00Z [--department customer_success]
```

Temporal ask: what was the entitled seat count **as-of** `2026-08-31T18:00:00Z` (before the **2026-09-01** renewal window)? Cite-both vs entitled `dept.customer_success` heartbeats is **after R4 pull**, not this overlay.

**not E-G1.** **not Memory GA.** 1.6 E-G1 **Parked**. **D5d ≠ E-G1**. empty until consume · CLIENT ≠ PULSE · catalog ≠ Connected · dual_write **OFF** · never YAML APPLY.
