# Marketing demo path (local agent + local memory)

**Serial:** free eng **s1590** · sales talk track **s1594** · GTM claim-support **s1598** · operator GTM boundary **s1602** · GTM wave 6 **s1606** · GTM wave 7 **s1610** · GTM wave 8 **s1614** · GTM wave 9 **s1618**  
**Audience:** demo hosts, sales eng, GTM video capture  
**Operator surface:** `/onboard next marketing-demo`  
**Aliases:** `marketing` · `sales-demo` · `demo-script` · `gtm-demo`

Plain-language script for **videos and sales demos** of the local agent harness with local memory. Prefer this over residual continuum boards when the goal is a clear operator walkthrough.

> **Not the same as** `/onboard next demo` (s1442 **demo readiness** packaging board — Lighthouse · Landgrab NOT READY).  
> **Not the same as** `/onboard next sales` (claims matrix) or `/onboard next gtm` (draft-only GTM).

---

## Demo script (order)

| Step | Action | Notes |
|------|--------|--------|
| **1** | Install / build `iomesh` | `go install` · `make build` · or release binary |
| **2** | Set LLM key or Ollama | Cloud provider API key, or local Ollama |
| **3** | `/setup init` local-memory + preflight | Default dual_write **OFF** · `/setup preflight` |
| **4** | Start / attach `iomesh-memory-mcp` | HTTP `http://127.0.0.1:8080/mcp` or stdio · TUI attach |
| **5** | Show `/memory` ingest + recall | Agent stores a fact · retrieve / digest / recall |
| **6** | Optional mesh | **Only if configured** · skip for pure local demos |

Slash: `/onboard next marketing-demo` (or any alias above).

---

## What you can show

- Local **agent** with multi-provider LLM or Ollama  
- Local **memory** via `iomesh-memory-mcp` + local palace  
- End-to-end: install → setup → attach → ingest → recall on a laptop  

## What not to claim

| Claim | Truth |
|-------|--------|
| Memory GA | **Not** Memory GA · local edge attach only |
| Connected | Do **not** invent org install Connected |
| dual_write ON | Default stays **OFF** |
| book-demo ON | book-demo **OFF** |
| Mesh required | Mesh is **optional** |

---

## Sales talk track

Short spoken path for a **local agent + local memory** demo (follow the script above live).

1. **Setup story** — “On your laptop: install `iomesh`, set an LLM key or Ollama, `/setup init` with local-memory (dual_write stays **OFF**), attach `iomesh-memory-mcp`.”
2. **The beat** — “Tell the agent something once; show `/memory` ingest and recall so it comes back from local storage — not a cloud CRM.”
3. **Claims guardrail** — Before customer-facing wording, check the private GTM **claims catalog** for what is demoable vs do-not-claim: [github.com/iome-sh/tool-marketing](https://github.com/iome-sh/tool-marketing) (private · operator-only · not customer docs).
4. **Win-back / closed-lost** — Follow-ups are **sales process** (humans / HITL loops). The TUI does **not** auto-push CRM win-back or closed-lost sequences.
5. **SEO** — We import Search Console exports and score opportunities offline — **no auto rank claims**.
6. **Publish path** — draft → human approve → Hermes handoff/bind is **operator** tooling; the TUI does **not** auto-post.
7. **CRM** — Closed-loop metrics are recorded after human CRM actions — the TUI is **not** the CRM.

### Operator GTM boundary (s1602 · s1606 · s1610 · s1614 · s1618)

Where secrets and writes live — keep demos honest:

- **Search Console credentials** stay on the **operator machine** (no secrets in git). Demos use offline exports / local tooling — not live API keys in the public TUI.
- **Hermes exec** for publish runs **outside** the public TUI with operator-held secrets; the TUI does **not** hold social tokens and never auto-posts.
- **CRM vendor adapters / metrics** are operator GTM (dry-run / human-gated stub). Commercial CRM writes stay **human** — the TUI is not the CRM of record.
- **Hermes network dispatch** (s1606) is an **operator webhook / private runner** — not the public TUI. Secrets stay outside git and off the demo harness.
- **HubSpot / Twenty CRM** paths (s1606) are **operator-box OAuth + human approve** — the TUI is not the CRM and does not mint or store CRM OAuth tokens.
- **No social or CRM tokens** live in the public harness (TUI · OSS packaging · customer-facing demo path).
- **Hermes dogfood** (s1610) uses an **operator mock / daemon** on the operator box — not the public TUI. The demo path is not a Hermes control plane.
- **HubSpot dual control** (s1610): commercial HubSpot writes need **human approve + write allow flags** on the operator path — the TUI is not CRM and does not dual-control those writes.
- **Sales-loop mesh outbox** (s1610) is **operator / local envelope wiring** — local outbox emit only. Do **not** invent mesh GTM fleet GA in the demo.
- **Real Hermes daemon** (s1614) is **operator-run on the operator box** — the public TUI does **not** host or start it.
- **Live HubSpot / Twenty writes** (s1614) need **dual control + tokens on the operator box** (default dry). The marketing-demo path stays **local agent + local memory** only.
- **Operator GTM status** (s1614) is **private tooling** for operators — **not** a product dashboard claim.
- **Scheduled GTM dogfood** (s1618) is **operator cron / offline tooling** — not the public TUI. The demo path is not a GTM scheduler.
- **Mesh outbox ingest** (s1618) is **private aion when wired** (dry-run default) — the demo does **not** invent mesh GTM fleet GA.
- **Smoke / status tools** (s1618 · e.g. `gtm_smoke`) are **private operator GTM** — **not** a customer dashboard.

Keep the demo local-first. Skip mesh unless the room already has it configured. Do not invent Memory GA, Connected, dual_write ON, book-demo ON, or mesh GTM fleet GA.

---

## Companion surfaces

| Surface | Role |
|---------|------|
| `/onboard next memory` | Edge OSS install detail |
| `/onboard next e4` | Client attach residual (tools=6 stamp) |
| `/onboard next setup` | Full setup lifecycle map |
| `/onboard next demo` | Packaging readiness (Lighthouse · Landgrab) |
| `/onboard next sales` | May claim / must not claim matrix |
| [memory-edge-usage-demo.md](./memory-edge-usage-demo.md) | Longer runbook walkthrough |
| [edge-user-journey.md](./edge-user-journey.md) | 7-stage product narrative SSOT |

---

## Free-floor

free eng **s1590** · sales talk track **s1594** · GTM claim-support **s1598** · operator GTM boundary **s1602** · GTM wave 6 **s1606** · GTM wave 7 **s1610** · GTM wave 8 **s1614** · GTM wave 9 **s1618** · free-floor peer **s1592+** mention only (do not rewrite free-floor ownership from this doc).
