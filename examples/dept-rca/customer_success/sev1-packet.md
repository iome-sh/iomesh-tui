# Sev-1 CS packet (sample · private overlay · PD-HMAC-5xx)

This is a residual-honest **Sev-1 CS packet overlay** for local palace ingest (V2-D). It joins the ops incident **PD-HMAC-5xx** (same `event_time` as `examples/dept-rca/ops/page.md`). It is **not** a live Zendesk/Salesforce consume, **not** a connector install, and **not** mesh. Do **not** stamp mesh on this file. Overlay does **not** GET Salesforce/CRM. Zendesk is optional; pulse does not exist (V2-F parked). CS sitting stays unchecked.

```
id: PD-HMAC-5xx
event_time: 2026-06-15T14:08:00Z
summary: HMAC 5xx Sev-1 customer packet (qualitative CS overlay)
source_hint: private
pointer: sev1-packet.md
kind: sev1_cs_packet
account: ACC-1001
ticket: ZD-1001
```

`source_hint=private`. No customer names. No invented ARR. ACC-1001 / ZD-1001 are private pointers only.

## Packet (qualitative)

HMAC-signed consumer returned **5xx** at `2026-06-15T14:08:00Z`. CS overlay is a **Sev-1 customer packet** on that same incident — extend **PD-HMAC-5xx**; do not invent a new page. Account **ACC-1001** and ticket **ZD-1001** are overlay pointers only (not a live Zendesk consume).

Palace (local overlay, not Memory GA):

```
iomesh memory ingest-dir --yes --department customer_success examples/dept-rca/customer_success
/memory digest --require-sources mesh,private
```

ingest-dir is enough. Do not invent a mesh heartbeat from this packet. `/memory digest --require-sources mesh,private` without pull → **miss is success** (cite-both or named miss).

## Honesty

private overlay · `source_hint=private` · dual_write **OFF** · catalog ≠ Connected · catalog ≠ heartbeat · CLIENT ≠ PULSE · empty until consume · not Memory GA · **not E-G1** · **V2 ≠ E-G1** · D5b/D6 parked · V2-F parked · never YAML APPLY · mesh miss is success · overlay does not GET Salesforce/CRM · never mesh stamp
