# Runbook: HMAC 200 is not a consume receipt (sample · private overlay)

This is a residual-honest **ops runbook** for the existing TTFH RCA. It is **not** live APPLY, **not** a PagerDuty Connected install, and **not** mesh. Do **not** stamp mesh on this file. Extend **PD-HMAC-5xx**; do not invent a new incident.

```
runbook_id:  hmac-200-not-consume-receipt
page:        PD-HMAC-5xx
created_at:  2026-06-15T14:08:00Z
fact:        HMAC 200 is not a consume receipt
```

## Rule (as of this page)

**HMAC 200 is not a consume receipt.** An HMAC-signed webhook HTTP 200 is ACK of the request, not overlay consume, not `/dashboard` PULSE, and not Connected. PULSE only after ≥1 decoded broker message. CLIENT ≠ PULSE. empty until consume.

HMAC 5xx (this page) is also not a consume-clock. A later HMAC 200 does not close the RCA.

Human `decision_stub` only. **Never YAML APPLY.**

## Temporal note

Page **PD-HMAC-5xx** was created `2026-06-15T14:08:00Z`. This runbook fact was in force at that as-of. Palace:

```
/memory facts-as-of --as-of 2026-06-15T14:08:00Z [--department ops]
```

`dept.ops.events.*` is routing, not Connected. Cite-both vs entitled ops heartbeats is after R4 pull, not this overlay. Mesh miss is success. Overlay does not query CRM/docs.

## Honesty

private overlay · dual_write **OFF** · catalog ≠ Connected · CLIENT ≠ PULSE · empty until consume · Cloud Memory GA · **not E-G1** · never YAML APPLY · mesh miss is success
