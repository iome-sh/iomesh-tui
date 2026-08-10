# Edge user journey (7-stage SSOT)

**Serial:** free eng **s1554** (Wave A docs SSOT) · **s1558** (Wave B first-run polish residual) · **s1562** (stage-5 portal HITL soft dogfood residual) · **s1566** (stage-6 E4 client-attach soft dogfood residual) · **s1570** (Wave C first-run wizard residual) · **s1574** (still-human APPLY soft dogfood residual) · **s1578** (deeper tool-call soft dogfood residual) · residual-honest **edge-first product narrative**  
**Status:** Wave A docs SSOT **shipped** · Wave B first-run polish **partially in product** · Wave C guided wizard residual **shipped residual-honest** (`/onboard next journey` + setup stage-4 map · stage-5 `/onboard next portal-hitl` soft dogfood residual · stage-6 `/onboard next e4` soft dogfood residual · Wave C `/onboard next wizard` + soft dogfood residual) · still-human APPLY soft residual **s1574** (`/onboard next human-gates dogfood`) · deeper tool-call soft residual **s1578** (`/onboard next tool-call dogfood`)  
**Audience:** operators, demo hosts, residual eng, GTM-adjacent claims review  
**Planes:** local TUI + public edge Memory (`iomesh-memory-mcp` + kernel) · optional portal/mesh · **aion private**

```text
1 Signup  →  2 Download TUI  →  3 TUI auth/keys  →  4 Setup wizard
           →  5 Connectors / events on mesh  →  6 Local store  →  7 Analyze
```

This document is the **single source of truth** for the edge-user journey narrative.  
Demo runbooks, setup lifecycle, and integrations docs **map into** these stages — they do not invent a different product path.

**Operator surface (s1558 Wave B · s1570 Wave C · s1574 still-human APPLY · s1578 deeper tool-call):** `/onboard next journey` (aliases `edge-journey` · `user-journey` · `first-run` · `edge_user_journey`) — residual-honest first-run map of the 7 stages · Wave C guided residual `/onboard next wizard` (aliases `first-run-wizard` · `guided` · `wave-c` · `wave_c` · `wizard-residual` · soft `/onboard next wizard dogfood`) · companion stage-4 detail `/onboard next setup` · stage-5 portal HITL `/onboard next portal-hitl` (s1562 · soft dogfood residual `/onboard next portal-hitl dogfood`) · stage-6 E4 client-attach `/onboard next e4` (s1566 · soft dogfood residual `/onboard next e4 dogfood`) · stage 6/7 deeper tool-call `/onboard next tool-call` (s1578 · soft `/onboard next tool-call dogfood`) · still-human APPLY residual `/onboard next human-gates` (s1574 · soft `/onboard next human-gates dogfood`) · setup guidance notes stages 1–7 with in-session focus on 4–7.

**Honesty one-liner:** drafts only · dual_write **OFF** · not Memory GA · Edge Memory GA **candidacy only** · edge-first · knowledge multi-tenant **punted** · Slack HMAC **punted** · H1/H2 **not** launch gate · portal HITL when connect · book-demo **OFF** · agent MCP **cannot** write installs · catalog ≠ Connected · residual PASS ≠ invent Edge Memory GA · residual PASS ≠ invent Edge Memory GA declared · E10 Open · PASS ≠ invent human-gate green · PASS ≠ live APPLY · open boxes stay open · tip ≠ invent forever-green product dogfood · rates ~$88/$119 · aion private · Palace sunset · free eng **s1554** / Wave B **s1558** / stage-5 soft residual **s1562** / stage-6 E4 soft residual **s1566** / Wave C wizard residual **s1570** / still-human APPLY soft residual **s1574** / deeper tool-call soft residual **s1578** · free-floor peer **s1572+** / **s1576+** / **s1580+** mention only.

---

## Wave scope

| Wave | In scope | Out of scope (do not invent shipped) |
|------|----------|--------------------------------------|
| **A (s1554)** | Docs SSOT for 7 stages · cross-links + phase mapping · residual-honest maturity stamps · ownership map · honesty locks + residual gaps | Auto memory host · TUI portal SSO · Connected invent · dual_write ON · book-demo ON · Memory GA |
| **B (s1558 · residual shipped)** | `/onboard next journey` first-run map · setup lane stage-4 mapping · setup guidance first-run polish · skill/docs stamps | Deeper interactive wizard UX beyond residual map · invent auto host / SSO / APPLY / dual_write ON |
| **s1562 free eng** | Stage-5 portal HITL board `/onboard next portal-hitl` · soft offline dogfood residual · proven path honesty needles · independent session soft labels | Live OAuth/Connected invent · live dogfood · agent install APPLY · free-floor rewrite |
| **s1566 free eng** | Stage-6 E4 client-attach board `/onboard next e4` · soft offline dogfood residual · tools=6 / `iomesh mcp --connect` honesty needles · independent session soft labels | Edge Memory GA declared invent · forever-green product dogfood · E10 closed · dual_write ON · live host start · free-floor rewrite |
| **C (s1570 · residual shipped)** | `/onboard next wizard` guided first-run residual map · soft offline dogfood residual · per-stage primary next actions · independent session soft labels | Full interactive auto wizard invent · TUI portal SSO · auto host · dual_write ON · Edge Memory GA declared · live dogfood · free-floor rewrite |
| **s1574 free eng** | Still-human APPLY soft offline dogfood residual `/onboard next human-gates dogfood` · open inventory reaffirm after Wave A–C continuum · independent session soft labels | Invent human-gate green · live APPLY · Edge Memory GA declared · E10 closed · free-floor rewrite |
| **s1578 free eng** | Deeper tool-call soft offline dogfood residual `/onboard next tool-call` · soft `/onboard next tool-call dogfood` · stage 6/7 depth after E4 attach · ingest→retrieve→list→as-of honesty needles · independent session soft labels | Forever-green product dogfood invent · live tool-call dogfood · Edge Memory GA declared · E10 closed · dual_write ON · free-floor rewrite |

**Still out of scope (all waves until residual closes):** TUI portal SSO / full console login in TUI · Memory host auto-provision on signup · Agent MCP INSTALL_STORE APPLY / Connected invent · dual_write ON · book-demo ON · Memory GA declared · Edge Memory GA declared · forever-green full product dogfood · E10 closed · invent human-gate green / live APPLY offline.

**Free-floor peer:** free-floor ownership remains **s1556+** / Wave B mention **s1560+** / stage-6 E4 mention **s1568+** / Wave C mention **s1572+** / still-human APPLY mention **s1576+** / deeper tool-call mention **s1580+** (mention only). Residual product narrative **does not** rewrite free-floor.

---

## Stage table

| # | Stage | Owner surface | What the user does | Honesty non-claims | Maturity |
|---|-------|---------------|--------------------|--------------------|----------|
| **1** | **Signup** | Portal **console.iome.sh** | Create/join org · optional API key / mesh credentials | Signup ≠ Memory GA · ≠ connectors Connected · ≠ INSTALL_STORE green · optional for pure local memory | **Shipped** (portal product) · TUI does not own signup |
| **2** | **Download TUI** | This repo / releases · `go install` / `make build` | Install `iomesh` binary | Binary install ≠ platform control plane · public OSS ≠ invent multi-tenant mesh | **Shipped** (MIT harness) |
| **3** | **TUI auth / keys** | TUI local env/config | Set **LLM API keys** (default cascade) · optional **Ollama** local · optional portal/mesh credentials | LLM keys ≠ platform SSO · Ollama = local only · **not** platform-bundled weights · optional mesh keys ≠ dual_write ON | **Shipped** (keys/env) · **no** invent TUI portal SSO |
| **4** | **Setup wizard** | TUI `/setup` · `/onboard next setup` · CLI `iomesh setup` | `init` → managed config · start host · `preflight` · optional reload · opt-in pull/analyze · drift · repair | setup PASS ≠ invent Connected · repair apply ≠ invent install green · dual_write never auto ON · portal HITL still human | **Shipped** P1–P7 + closeout residual (s1525–s1542) · human-gates reaffirm s1546/s1550 · still-human APPLY soft residual **s1574** |
| **5** | **Connectors / events on mesh** | TUI agent MCP + **portal HITL** | `/integrations` list/plan · `/onboard next portal-hitl` · open portal deep links · human finishes OAuth/install · events land on mesh when installed · soft residual `/onboard next portal-hitl dogfood` (s1562 · soft offline ≠ invent Connected) | agent MCP **cannot** write installs · catalog ≠ Connected · knowledge multi-tenant **punted** · Slack HMAC **punted** · H1/H2 not launch gate · book-demo OFF · residual PASS ≠ live dogfood · session soft ≠ live dogfood | **Shipped** list/plan/status residual · **still human** for OAuth/install APPLY · soft dogfood residual **s1562** |
| **6** | **Local store** | `iomesh-memory-mcp` + memory kernel · TUI attach | Install/run host · attach MCP · local FS palace · dual_write **OFF** · soft residual `/onboard next e4` · `/onboard next e4 dogfood` (s1566 · soft offline ≠ invent Edge Memory GA declared · tip ≠ invent forever-green product dogfood) · deeper soft residual `/onboard next tool-call` · `/onboard next tool-call dogfood` (s1578 · ingest→retrieve→list→as-of) | not Memory GA · Edge Memory GA **candidacy only** · residual PASS ≠ invent Edge Memory GA · residual PASS ≠ invent Edge Memory GA declared · E10 Open · host **not** auto on signup · aion broker private · Palace sunset (hosted) · residual PASS ≠ live dogfood · session soft ≠ live dogfood | **Shipped** public edge attach path · **candidacy only** for Edge Memory GA · soft dogfood residual **s1566** · deeper tool-call soft residual **s1578** |
| **7** | **Analyze** | TUI `/memory digest` · `/setup analyze` · optional mesh pull · companion deeper tool residual | Digest / analyze ticks · optional Ops Pack pull into local palace · companion `/onboard next tool-call` path residual | analyze ≠ invent Connected · pull ≠ invent Memory GA · residual PASS ≠ invent forever-green tool-call dogfood · mesh base ~**$88** · Memory Ops Pack ~**$119** · local-primary still holds | **Shipped** digest + opt-in analyze/pull surfaces · rates language residual-honest · deeper tool residual **s1578** candidacy |

---

## Ownership map

| Surface | Owns | Does **not** own |
|---------|------|------------------|
| **iomesh-tui** (this repo) | Agent harness · LLM router · MCP client · `/setup` · `/integrations` list/plan · `/memory` · optional mesh **client** pull | Hosted control plane · install-store fleet APPLY · portal OAuth complete · freemium hosted palace |
| **Portal console** (console.iome.sh) | Signup · org/workspace · OAuth/install HITL · API keys · billing surfaces | Shipping the MIT TUI binary · local palace FS |
| **aion** (private) | Broker / mesh control plane · connector install plane · platform MCP tools server-side | Public OSS edge claim as control plane · invent platform GA from residual docs |
| **iomesh-memory-mcp** + **memory** kernel (public) | Local Memory host + PalaceStore · MCP tools for ingest/retrieve/… | dual_write primary · multi-tenant hosted palace · Memory GA invent |
| **Website / marketing** | Public copy · rates language · docs links | Overclaim Connected / Memory GA / dual_write ON / book-demo |

---

## Honesty locks (must hold)

| Lock | Residual truth |
|------|----------------|
| **drafts only** | Sales/demo boards and residual stamps are drafts · not live install green |
| **dual_write OFF** | Default OFF · optional mesh audit only · never primary palace |
| **not Memory GA** | Public edge + TUI attach ≠ bare Memory GA |
| **Edge Memory GA candidacy only** | Residual candidacy · **PASS ≠ invent Edge Memory GA declared** |
| **edge-first** | Local TUI + local memory primary · mesh optional |
| **knowledge multi-tenant punted** | INSTALL_STORE multi-tenant knowledge path **punted** for edge-first launch |
| **Slack HMAC punted** | Live signed Slack not a launch human gate for current path |
| **H1/H2 not launch gate** | Knowledge multi-tenant human gates demoted off launch path |
| **portal HITL when connect** | OAuth / install complete only in browser portal |
| **book-demo OFF** | No invent book-a-demo install path |
| **agent MCP cannot write installs** | list/plan/discovery only · no INSTALL_STORE APPLY from agent MCP |
| **catalog ≠ Connected** | Catalog chips / plan URLs ≠ org install Connected |
| **residual PASS ≠ invent Edge Memory GA** | Setup/onboard residual complete ≠ GA declaration |
| **rates ~$88 / ~$119** | Mesh base footprint ~$88 · Memory Ops Pack ~$119 when SKUs mentioned · not cloud GPU palace |
| **aion private** | Control plane / broker stay private |
| **Palace sunset** | Hosted multi-tenant palace path sunset until scale · local-primary |
| **free eng s1554** | This SSOT serial |
| **free-floor peer s1556+** | Free-floor continuum owned elsewhere · residual product does **not** rewrite free-floor |

---

## Explicit residual gaps (do not invent closed)

1. **No TUI portal SSO invent** — stage 3 is LLM keys + optional mesh credentials · not full console SSO inside the TUI for v1.
2. **Memory host not auto on signup** — stage 6 requires operator install/run of `iomesh-memory-mcp` (kernel as dep) · signup does not provision a palace · operator map `/onboard next e4` · soft offline dogfood residual **s1566** (soft offline ≠ invent Edge Memory GA declared · tip ≠ invent forever-green product dogfood · residual PASS ≠ live dogfood · E10 Open).
3. **Events need portal install HITL** — stage 5 list/plan is not install complete · connectors become Connected only after human portal OAuth/install · operator map `/onboard next portal-hitl` · soft offline dogfood residual **s1562** (soft offline ≠ invent Connected · residual PASS ≠ live dogfood).
4. **Still-human APPLY / E10** — setup closeout and residual PASS do not close human-gate APPLY or founder E10 Edge Memory GA sign-off.
5. **Continuous pull / analyze are opt-in** — not silent default green paths · dual_write stays OFF.

---

## Cross-links (existing residual-honest docs)

| Doc | Role relative to this SSOT |
|-----|----------------------------|
| [memory-edge-usage-demo.md](./memory-edge-usage-demo.md) | Runbook walkthrough · phase numbering maps to stages (see mapping there) |
| [setup-lifecycle.md](./setup-lifecycle.md) | Stage **4** detail — P1–P7 · `/setup` · repair honesty |
| [agent-integrations-setup.md](./agent-integrations-setup.md) | Stage **5** detail — `/integrations` MCP list/plan + portal HITL |
| [memory-mcp.md](./memory-mcp.md) | Stage **6–7** depth — edge attach · pull · slash surfaces · buyer claim pins |
| [memory-advanced-install.md](./memory-advanced-install.md) | Optional advanced ladder (ONNX · Qdrant honesty) under stage 6 |
| [mcp.md](./mcp.md) | MCP client transports · `iomesh mcp --connect` |
| [EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md](../EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md) | One residual attach stamp · not forever-green dogfood |

---

## Narrative (short form)

1. **Signup** at the portal when you need org identity, integrations, or optional mesh — skip for pure local memory.  
2. **Download** the MIT `iomesh` TUI.  
3. **Auth** with your own LLM keys (or Ollama local) — not platform-bundled weights; optional portal/mesh credentials.  
4. **Setup** with `/setup` / `iomesh setup` — residual-honest preflight, reload, opt-in pull/analyze, guided repair.  
5. **Connect** via agent MCP catalog/plan and **finish in portal HITL** — never claim agent wrote installs.  
6. **Store** locally with `iomesh-memory-mcp` + kernel — dual_write OFF · Edge Memory GA candidacy only.  
7. **Analyze** with `/memory digest` / setup analyze ticks — optional mesh Ops Pack pull (~$119) · rates residual-honest.

---

## Non-goals

- Invent TUI as multi-tenant control plane  
- Invent Connected / INSTALL_STORE green from catalog or residual PASS  
- Invent Memory GA / Edge Memory GA declared / E10 closed  
- dual_write ON as primary path  
- Auto Memory provision on signup  
- Rewrite free-floor (peer **s1556+** / **s1560+** mention) from residual product docs  
- Claim full interactive first-run wizard beyond residual map lanes (Wave B ships residual-honest map + setup stage-4 polish; Wave C ships deeper guided residual map + soft dogfood only — **not** full interactive auto wizard UX)  
- Invent auto Memory host · TUI portal SSO · Connected · dual_write ON · Memory GA declared  
