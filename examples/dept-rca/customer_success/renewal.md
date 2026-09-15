# Renewal window (sample · private overlay)

This is a residual-honest **renewal window** for local palace ingest. It is **not** live APPLY, **not** customer ARR, **not** a Salesforce/CRM GET, and **not** mesh. Do **not** stamp mesh on this file. Overlay does **not** GET Salesforce/CRM.

```
renewal_id:       acc-1001-renewal
kind:             entitled seats as a count (sample)
account:          ACC-1001
window:           2026-09-01
entitled_seats:   25 (count · not ARR · not a quote)
owner:            cs-csm
```

No customer names in this sample. No invented ARR. No dollar amounts. No churn %.

## Rule (as of this fact)

**ACC-1001** is entitled to **25 seats** (a count, not ARR) for the renewal window **2026-09-01**. Facts-as-of `2026-08-31T18:00:00Z` is **before** that window — cite this seat count, not a post-renewal figure, and not customer ARR.

Human `decision_stub` only. **Never YAML APPLY.** The playbook may **point at** the health snapshot; it must not auto-APPLY a save or renew.

## Temporal note

Account-health snapshot **ACC-1001** is dated **2026-08-15**. This renewal window is **2026-09-01**. The entitled seat count was in force at as-of `2026-08-31T18:00:00Z`. Palace:

```
/memory facts-as-of --as-of 2026-08-31T18:00:00Z [--department customer_success]
```

`dept.customer_success.events.*` is routing, not Connected. Cite-both vs entitled customer_success heartbeats is after R4 pull, not this overlay. Mesh miss is success.

## Honesty

private overlay · dual_write **OFF** · catalog ≠ Connected · CLIENT ≠ PULSE · empty until consume · not Memory GA · **not E-G1** · **D5d ≠ E-G1** · never YAML APPLY · mesh miss is success · overlay does not GET CRM/docs
