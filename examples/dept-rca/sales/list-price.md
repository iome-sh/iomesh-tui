# List-seat price fact (sample · private overlay)

This is a residual-honest **list-seat price fact** for local palace ingest. It is **not** live APPLY, **not** customer ARR, **not** a Salesforce/CRM GET, and **not** mesh. Do **not** stamp mesh on this file. Overlay does **not** GET Salesforce/CRM.

```
price_id:         list-seat
kind:             published list-seat unit rate (sample)
effective_until:  2026-03-01
rate:             12 list-units / seat / month (sample · not customer ARR · not a quote)
change_on:        2026-03-01 list-seat price change
opp:              OPP-1001
```

No customer names in this sample. No invented ARR.

## Rule (as of this fact)

This sample **list-seat** unit rate is in force **until 2026-03-01**. On **2026-03-01** the list-seat price changes. Facts-as-of `2026-02-28T18:00:00Z` is **before** that change — cite this rate, not a post-change figure, and not customer ARR.

Human `decision_stub` only. **Never YAML APPLY.** Call notes and QBR may **point at** this fact; they must not auto-quote or APPLY a price book.

## Temporal note

Opportunity **OPP-1001** call notes were created `2026-02-20T16:00:00Z`. QBR notes are dated 2026-02-15. This list-seat fact (effective until 2026-03-01) was in force at as-of `2026-02-28T18:00:00Z`. Palace:

```
/memory facts-as-of --as-of 2026-02-28T18:00:00Z [--department sales]
```

`dept.sales.events.*` is routing, not Connected. Cite-both vs entitled sales heartbeats is after R4 pull, not this overlay. Mesh miss is success.

## Honesty

private overlay · dual_write **OFF** · catalog ≠ Connected · CLIENT ≠ PULSE · empty until consume · Cloud Memory GA · **not E-G1** · **D5c ≠ E-G1** · never YAML APPLY · mesh miss is success · overlay does not GET CRM/docs
