# Sales department RCA kit (private overlay)

Sample UTF-8 corpus for **department temporal RCA** (V1.6 D5c). Import is **private overlay** only. Do **not** stamp mesh on these files. Mesh miss is success. Ingest after `examples/dept-rca/support` and `examples/dept-rca/ops` into **one** palace = one-tenant composition (≠ two-org · not a Salesforce Connected install · **not** an org-wide RCA engine · **D5c ≠ E-G1**). Overlay does **not** GET Salesforce/CRM.

```bash
iomesh memory ingest-dir --yes examples/dept-rca/sales
# inventory only:
iomesh memory ingest-dir --dry-run examples/dept-rca/sales
# slash twin:
# /memory ingest-dir examples/dept-rca/sales --dry-run
```

`session_id` mints as `local-overlay` when the walk has none (`--department sales` mints `local-overlay:sales`). `--source-hint` is **private** only (`source_hint=private`; `mesh` is an error). dual_write stays **OFF**. Catalog list ≠ consume.

## Files

| File | Role |
|------|------|
| `call-notes.md` | Salesforce-shaped sample call notes **OPP-1001**. Created `2026-02-20T16:00:00Z`. |
| `qbr.md` | QBR UTF-8 notes for the same opp, dated **2026-02-15**. Not a deck binary. Not CRM GET. |
| `list-price.md` | List-seat **price fact** effective until **2026-03-01** (price change). Human `decision_stub` only — never YAML APPLY. |

`dept.sales.events.*` is **routing**, not Connected. A Salesforce-shaped filename is not a Salesforce install.

## After ingest (R2 companion)

```bash
/memory digest --require-sources mesh,private
# miss is success without pull — do not invent mesh cite-both from this kit
/memory facts-as-of --as-of 2026-02-28T18:00:00Z [--department sales]
```

Temporal ask: what was the list-seat price **as-of** `2026-02-28T18:00:00Z` (before the **2026-03-01** price change)? Cite-both vs entitled `dept.sales` heartbeats is **after R4 pull**, not this overlay.

**not E-G1.** **not Memory GA.** 1.6 E-G1 **Parked**. **D5c ≠ E-G1**. empty until consume · CLIENT ≠ PULSE · catalog ≠ Connected · dual_write **OFF** · never YAML APPLY.
