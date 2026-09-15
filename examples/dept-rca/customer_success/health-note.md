# Account-health snapshot (sample · private overlay)

This is a residual-honest **account-health snapshot** for local palace ingest. It is **not** a live Salesforce/CS install, **not** a connector, and **not** mesh. Do **not** stamp mesh on this file. Overlay does **not** GET Salesforce/CRM. `dept.customer_success.events.*` is routing, not Connected.

```
id:          ACC-1001
object:      Account (sample shape)
type:        health_note
date:        2026-08-15
owner:       cs-csm
account:     account-owner
health:      watch (sample qualitative · not a churn % · not MTTR)
next_step:   cite entitled seat count as-of 2026-08-31T18:00:00Z
```

No customer names in this sample. No invented ARR. No churn %. No MTTR %.

## Description

Account-health snapshot for **ACC-1001** dated **2026-08-15**. Account: `account-owner`. Qualitative health is **watch** — not a churn percentage, not MTTR. Entitled seats (a **count**, not ARR) live in `renewal.md` for the **2026-09-01** renewal window.

The operator question is temporal: **what was the entitled seat count as-of `2026-08-31T18:00:00Z`?** (before the **2026-09-01** renewal window)

Palace (local overlay, not Memory GA):

```
/memory facts-as-of --as-of 2026-08-31T18:00:00Z [--department customer_success]
```

Do not invent a mesh heartbeat from this snapshot. `/memory digest --require-sources mesh,private` without pull → **miss is success**.

## Honesty

private overlay · dual_write **OFF** · catalog ≠ Connected · CLIENT ≠ PULSE · empty until consume · not Memory GA · **not E-G1** · **D5d ≠ E-G1** · never YAML APPLY · mesh miss is success · overlay does not GET CRM/docs
