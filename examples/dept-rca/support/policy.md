# Unused-seat refund policy (sample · private overlay)

This is a residual-honest **policy fact** for local palace ingest. It is **not** live APPLY, **not** a CRM/docs SoR GET, and **not** mesh. Do **not** stamp mesh on this file.

```
policy_id:   unused-seat-refund
effective:   2026-01-01
window:      14-day unused-seat refund from purchase
supersedes:  none in this sample
```

## Rule (as of 2026-01-01)

This is a **14-day** unused-seat refund policy from 2026-01-01. Unused seats may be refunded when the refund request is received **within 14 days of the seat purchase**. After 14 days, unused-seat refunds are declined except where required by law.

Human `decision_stub` only. **Never YAML APPLY.** Macros may **point at** this policy; they must not auto-issue a refund.

## Temporal note

Ticket **ZD-1001** was created `2026-06-15T14:22:00Z`. This policy (effective 2026-01-01) was in force at that as-of. Palace:

```
/memory facts-as-of --as-of 2026-06-15T14:22:00Z
```

`dept.support.events.*` is routing, not Connected. Cite-both vs entitled support heartbeats is after R4 pull, not this overlay. Mesh miss is success.

## Honesty

private overlay · dual_write **OFF** · catalog ≠ Connected · CLIENT ≠ PULSE · empty until consume · not Memory GA · **not E-G1** · never YAML APPLY · mesh miss is success
