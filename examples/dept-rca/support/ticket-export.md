# Ticket export (sample · private overlay)

This is a residual-honest **Zendesk-shaped** ticket export for local palace ingest. It is **not** a live ticket, **not** a connector install, and **not** mesh. Do **not** stamp mesh on this file. `dept.support.events.*` is routing, not Connected.

```
id:          ZD-1001
subject:     unused-seat refund
created_at:  2026-06-15T14:22:00Z
updated_at:  2026-06-15T14:22:00Z
status:      open
channel:     email
priority:    normal
requester:   account-owner
assignee:    support-macro
tags:        unused-seat, refund, billing
```

No customer names in this sample.

## Description

Account owner requested a refund for an **unused seat**. Purchase of the seat was 2026-06-08. Ticket opened 2026-06-15T14:22:00Z.

The operator question is temporal: **what was the unused-seat refund rule as-of ticket created_at `2026-06-15T14:22:00Z`?**

Palace (local overlay, Cloud Memory GA):

```
/memory facts-as-of --as-of 2026-06-15T14:22:00Z
```

Do not invent a mesh heartbeat from this export. `/memory digest --require-sources mesh,private` without pull → **miss is success**.

## Honesty

private overlay · dual_write **OFF** · catalog ≠ Connected · CLIENT ≠ PULSE · empty until consume · Cloud Memory GA · **not E-G1** · never YAML APPLY · mesh miss is success
