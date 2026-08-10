# Marketing demo path (local agent + local memory)

**Serial:** free eng **s1590**  
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

free eng **s1590** · free-floor peer **s1592+** mention only (do not rewrite free-floor ownership from this doc).
