# Call notes (sample · private overlay)

This is a residual-honest **Salesforce-shaped** call-notes export for local palace ingest. It is **not** a live Salesforce install, **not** a connector, and **not** mesh. Do **not** stamp mesh on this file. Overlay does **not** GET Salesforce/CRM. `dept.sales.events.*` is routing, not Connected.

```
id:          OPP-1001
object:      Opportunity (sample shape)
type:        call_notes
created_at:  2026-02-20T16:00:00Z
updated_at:  2026-02-20T16:00:00Z
stage:       discovery (sample)
owner:       sales-ae
account:     account-owner
next_step:   cite list-seat price as-of 2026-02-28T18:00:00Z
```

No customer names in this sample. No invented ARR.

## Description

Follow-up call on opportunity **OPP-1001** at `2026-02-20T16:00:00Z`, after QBR notes dated 2026-02-15. Account owner asked what list-seat rate applies before the **2026-03-01** price change.

The operator question is temporal: **what was the list-seat price as-of `2026-02-28T18:00:00Z`?**

Palace (local overlay, not Memory GA):

```
/memory facts-as-of --as-of 2026-02-28T18:00:00Z [--department sales]
```

Do not invent a mesh heartbeat from these notes. `/memory digest --require-sources mesh,private` without pull → **miss is success**.

## Honesty

private overlay · dual_write **OFF** · catalog ≠ Connected · CLIENT ≠ PULSE · empty until consume · not Memory GA · **not E-G1** · **D5c ≠ E-G1** · never YAML APPLY · mesh miss is success · overlay does not GET CRM/docs
