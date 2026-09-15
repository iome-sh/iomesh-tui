# Macro: unused-seat refund (sample · private overlay)

This is a residual-honest **support macro** that **points at** `policy.md`. It is **not** live APPLY, **not** a Zendesk Connected install, and **not** mesh. Do **not** stamp mesh on this file.

```
macro_id:    unused-seat-refund
ticket:      ZD-1001
points_at:   policy.md
policy:      14-day unused-seat refund, effective 2026-01-01
```

## When to use

Requester asks for an **unused-seat refund**. Cite the 14-day policy in `policy.md` (effective 2026-01-01). Compare seat purchase date to ticket created_at `2026-06-15T14:22:00Z`.

## Must not

- Auto-issue a refund (never YAML APPLY)
- Invent Connected / mesh cite-both from this overlay
- Stamp mesh on local files
- Treat `dept.support.events.*` routing as Connected

Human `decision_stub` only.

## Honesty

private overlay · dual_write **OFF** · catalog ≠ Connected · CLIENT ≠ PULSE · empty until consume · not Memory GA · **not E-G1** · never YAML APPLY · mesh miss is success
