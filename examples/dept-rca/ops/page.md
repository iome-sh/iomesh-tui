# Page (sample · private overlay)

This is a residual-honest **PagerDuty-shaped** page for local palace ingest. It is **not** a live page, **not** a PagerDuty install, and **not** mesh. Do **not** stamp mesh on this file. `dept.ops.events.*` is routing, not Connected.

```
id:          PD-HMAC-5xx
title:       HMAC 5xx (consumer 500)
created_at:  2026-06-15T14:08:00Z
updated_at:  2026-06-15T14:08:00Z
status:      triggered
urgency:     high
service:     mesh-consumer
incident_key: hmac-5xx
```

No customer names in this sample.

## Description

HMAC-signed consumer returned **5xx** at `2026-06-15T14:08:00Z`. Same incident window as the TTFH RCA (HMAC 200 is not a consume receipt) — extend that fact; do not invent a new incident.

The operator question is temporal: **what was the HMAC/consume rule as-of page created_at `2026-06-15T14:08:00Z`?**

Palace (local overlay, not Memory GA):

```
/memory facts-as-of --as-of 2026-06-15T14:08:00Z [--department ops]
```

Do not invent a mesh heartbeat from this page. `/memory digest --require-sources mesh,private` without pull → **miss is success**. HMAC 5xx is not overlay PULSE. CLIENT ≠ PULSE. empty until consume.

## Honesty

private overlay · dual_write **OFF** · catalog ≠ Connected · CLIENT ≠ PULSE · empty until consume · not Memory GA · **not E-G1** · never YAML APPLY · mesh miss is success
