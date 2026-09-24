# Playbook: account-health save/renew (sample · private overlay)

This is a residual-honest **CS playbook** that **points at** `health-note.md`. It is **not** live APPLY, **not** a Salesforce/CS Connected install, and **not** mesh. Do **not** stamp mesh on this file. Overlay does **not** GET Salesforce/CRM.

```
playbook_id: account-health-save-renew
account:     ACC-1001
points_at:   health-note.md
health_date: 2026-08-15
renewal:     2026-09-01 (entitled seats as a count · see renewal.md)
```

## When to use

Account **ACC-1001** is in a **watch** health snapshot dated **2026-08-15**. Cite `health-note.md`. Entitled seats (a count, not ARR) for the **2026-09-01** renewal window are in `renewal.md`. Compare that count at facts-as-of `2026-08-31T18:00:00Z` (before the renewal window).

## Must not

- Auto-APPLY a save or renew (never YAML APPLY)
- Invent Connected / mesh cite-both from this overlay
- Stamp mesh on local files
- Treat `dept.customer_success.events.*` routing as Connected
- Invent ARR, dollar amounts, churn %, or customer names

Human `decision_stub` only.

## Honesty

private overlay · dual_write **OFF** · catalog ≠ Connected · CLIENT ≠ PULSE · empty until consume · Cloud Memory GA · **not E-G1** · **D5d ≠ E-G1** · never YAML APPLY · mesh miss is success · overlay does not GET CRM/docs
