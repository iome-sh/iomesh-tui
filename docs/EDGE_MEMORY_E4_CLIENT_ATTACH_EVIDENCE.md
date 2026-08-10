# Edge Memory E4 MCP client attach evidence (s1508)

Residual-honest **observed stamp only** for free eng **s1508** (TUI tip ↔ lean product host full MCP client attach).

**Soft offline residual (s1566):** optional E4 client-attach soft dogfood for journey **stage 6** local store / MCP attach — `/onboard next e4` · soft `/onboard next e4 dogfood` · session labels `e4_soft_not_run` · `soft_offline_e4_session_pass|fail` · **never dial MCP / never start host** from soft residual · residual PASS ≠ invent Edge Memory GA declared · tip ≠ invent forever-green product dogfood · residual PASS ≠ live dogfood · session soft ≠ live dogfood · dual_write OFF · E10 Open · free eng **s1566** · free-floor peer **s1568+** mention only.

**Deeper tool-call soft residual (s1578):** optional deeper tool path soft dogfood after E4 attach (tools=6 stamp residual) — journey **stage 6/7** depth · operator map ingest → retrieve → list → as-of/status · `/onboard next tool-call` · soft `/onboard next tool-call dogfood` · session labels `tool_call_soft_not_run` · `soft_offline_tool_call_session_pass|fail` · tool names `memory_ingest_turn` · `memory_retrieve` · `memory_search_semantic` · `memory_list` · `memory_compact_status` · `memory_facts_as_of` · **never dial MCP / never start host** from soft residual · Partial→client-attach-evidence · deeper tool-call residual candidacy only · residual PASS ≠ invent Edge Memory GA declared · tip ≠ invent forever-green product dogfood · residual PASS ≠ live dogfood · session soft ≠ live dogfood · dual_write OFF · E10 Open · free eng **s1578** · free-floor peer **s1580+** mention only.

**Not claimed:** Edge Memory GA declared · bare Memory GA · hosted Memory GA · E10 closed · forever-green product dogfood · dual_write ON · freemium palace · full platform sidecar parity · live tool-call dogfood green.

## Pin (do not invent more)

| Field | Value |
|-------|--------|
| **UTC** | `2026-08-09T06:23:34Z` |
| **TUI tip** | `6b3958a90a01d2c8f50ee161c8dc1009637b64f1` |
| **MCP tip** | `f46afe2462ebaa94890b30296b1a19d03d6853da` (host binary version stamp `f46afe2`) |
| **Attach result** | `connected=1` · `tools=6` |
| **Tools listed** | `memory_ingest_turn`, `memory_retrieve`, `memory_search_semantic`, `memory_list`, `memory_compact_status`, `memory_facts_as_of` |

## Steps that worked

1. Start lean host:

```bash
iomesh-memory-mcp \
  -palace-root … \
  -tenant e4attach \
  -http-addr 127.0.0.1:18081 \
  -http-path /mcp
```

2. healthz OK — residual fields observed include `dual_write=off` · `not_memory_ga=true` (probe only).

3. Temp `config.toml`:

```toml
[mcp]
enabled = true
[[mcp.servers]]
name = "iomesh-memory-mcp"
url = "http://127.0.0.1:18081/mcp"
```

4. `./bin/iomesh mcp --config <cfg> --connect` → **connected=1** · **tools=6** · tools listed above.

## Honesty

| Claim | Truth |
|-------|--------|
| **local-primary** | Customer-edge FS palace via lean host + TUI MCP client |
| **dual_write** | **OFF** |
| **Edge Memory GA candidacy only** | Residual candidacy · **PASS ≠ invent Edge Memory GA declared** |
| **not bare Memory GA** | Attach stamp ≠ invent bare product Memory GA |
| **not hosted Memory GA** | Local lean host ≠ multi-tenant hosted palace |
| **aion broker private** | Cloud broker/CP stays private |
| **E10 Open** | Founder/GTM sign-off remains open |
| **tip ≠ invent forever-green** | One observed stamp · not continuous product dogfood green |

See runbook: [architecture/memory-mcp.md](architecture/memory-mcp.md#e4-mcp-client-attach-dogfood-s1508).
