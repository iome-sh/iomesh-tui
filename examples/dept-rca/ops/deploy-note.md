# Deploy note: consumer 500 (sample · private overlay)

This is a residual-honest **deploy note** in the **PD-HMAC-5xx** incident window. It is **not** live APPLY, **not** a Connected install, and **not** mesh. Do **not** stamp mesh on this file.

```
note_id:     consumer-500
page:        PD-HMAC-5xx
window:      2026-06-15T14:08:00Z
symptom:     consumer 500
```

## Note

In the same incident window as page **PD-HMAC-5xx** (`2026-06-15T14:08:00Z`), a consumer deploy returned **500**. HMAC 5xx followed. This note records the window; it does not ship, roll, or APPLY.

HMAC 200 is not a consume receipt — a later 200 after this 500 is still not overlay PULSE. empty until consume · CLIENT ≠ PULSE.

Human `decision_stub` only. **Never YAML APPLY.**

## Temporal note

Palace (local overlay, not Memory GA):

```
/memory facts-as-of --as-of 2026-06-15T14:08:00Z [--department ops]
```

`dept.ops.events.*` is routing, not Connected. Do not invent a mesh heartbeat from this note. Mesh miss is success.

## Honesty

private overlay · dual_write **OFF** · catalog ≠ Connected · CLIENT ≠ PULSE · empty until consume · not Memory GA · **not E-G1** · never YAML APPLY · mesh miss is success
