# Living memo (sample · private overlay · support.theme)

This is a residual-honest **RevOps living memo** for local palace ingest (V2-C). Same incident metadata as `/gtm brief recipe`. It is **not** a live Zendesk/Salesforce consume, **not** a health score, **not** a connector install, and **not** mesh. Do **not** stamp mesh on this file. Overlay does **not** GET Salesforce/CRM. `summarize_account_health` is not a health score.

One stream noun: **`support.theme`** (`dept.gtm.support.theme`). Hands (win-back, price change) stay off this plane.

```
id: THM-1001
event_time: 2026-08-20T15:00:00Z
summary: unused-seat refunds clustering (qualitative support.theme)
source_hint: private
pointer: living-memo.md
kind: support_theme
subject: dept.gtm.support.theme
account: ACC-1001
```

`source_hint=private`. No customer names. No invented ARR. No churn %. No MTTR %. No health score.

## Theme (qualitative)

Operators seeing unused-seat refund tickets (ZD-1001 shape) cluster as a **support.theme** — billing-cancel language in refund asks, not a churn percentage, not `summarize_account_health` as a score, not linked_pr_miss as MTTR.

Account **ACC-1001** is the overlay pointer only. Entitled seats stay a **count** in `renewal.md` (two dated figures with sales list-seat; do not collapse to one percentage). This memo does not replace the health snapshot and does not GET Salesforce/CRM.

Palace (local overlay, not Memory GA):

```
iomesh memory ingest-dir --yes --department customer_success examples/dept-rca/customer_success
/memory digest --require-sources mesh,private
```

ingest-dir is enough. Do not invent a mesh heartbeat from this memo. `/memory digest --require-sources mesh,private` without pull → **miss is success** (cite-both or named miss).

## Honesty

private overlay · `source_hint=private` · dual_write **OFF** · catalog ≠ Connected · catalog ≠ heartbeat · CLIENT ≠ PULSE · empty until consume · not Memory GA · **not E-G1** · **V2 ≠ E-G1** · D5b/D6 parked · never YAML APPLY · mesh miss is success · overlay does not GET CRM/docs · never mesh stamp · not a health score · not a churn % · linked_pr_miss ≠ MTTR
