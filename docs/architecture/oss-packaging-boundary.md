# OSS packaging boundary (MIT harness)

**Serial:** free eng **s1582** (OSS packaging residual) · residual-honest **public MIT packaging boundary**  
**Status:** residual-honest SSOT for what public `iomesh-tui` is vs is not  
**Audience:** OSS readers, operators, residual eng, GTM-adjacent claims review  
**Planes:** local TUI + public edge Memory (`iomesh-memory-mcp` + kernel) · optional portal/mesh client · **aion private**

This document is the **packaging boundary** for the public MIT repo.  
Edge harness and local-primary path come first. Platform residual honesty rails stay labeled **optional anti-claim residual** — not product claims that this MIT surface is the multi-tenant control plane.

**Honesty one-liner:** MIT OSS · dual_write **OFF** · not Memory GA · Edge Memory GA **candidacy only** · book-demo **OFF** · residual PASS ≠ invent control plane in MIT repo · soft residual-check (`… dogfood` slash token) = offline residual honesty check · session soft ≠ live dogfood · free eng **s1582** · free-floor peer **s1584+** mention only.

---

## MIT OSS harness vs private control plane

| This public MIT repo **is** | This public MIT repo is **not** |
|-----------------------------|-------------------------------|
| Local agent loop (TUI / headless / ACP), tools, subagents, skills, permissions | Hosted multi-tenant mesh control plane or cloud admin UI |
| Multi-provider LLM router (API keys + optional **Ollama** local models) | A cloud GPU / managed model product |
| **Local memory** setup, attach, and use via public [`iomesh-memory-mcp`](https://github.com/iome-sh/iomesh-memory-mcp) + [`memory`](https://github.com/iome-sh/memory) | Hosted multi-tenant memory-as-a-service |
| Optional **mesh client** hooks against a broker you run or subscribe to | Free access to IOMesh Cloud / private platform backends |
| Optional mesh **pull** into local store; dual_write default **OFF** | A product that requires dual_write ON or cloud palace as the primary store |
| Residual-honest operator boards + offline residual-check harnesses (maintainers) | A claim that offline residual PASS is live platform green |

**Public OSS ≠ invent platform GA.** Residual monorepo / private `aion` paths stay private.

---

## Edge OSS path (read this first)

Primary public path for operators and OSS readers:

1. **Install TUI** — `go install` / `make build` / GitHub Releases (this repo).
2. **LLM keys** — default cascade env (`DEEPSEEK_API_KEY`, optional Grok/Gemini/Vertex) · optional **Ollama** local pin (not platform GPU).
3. **Setup** — `/setup` · `/onboard next setup` · `iomesh setup init|preflight` (dual_write never auto ON).
4. **Optional connectors** — MCP list/plan + **portal HITL** when OAuth/install is needed (`/onboard next portal-hitl` · agent MCP **cannot** write installs · catalog ≠ Connected).
5. **Local memory** — public `iomesh-memory-mcp` + kernel · attach HTTP/stdio · dual_write **OFF** · `/onboard next memory` · `/onboard next e4` attach residual.
6. **Analyze** — `/memory digest` · `/setup analyze` · optional mesh pull into local palace.

Operator maps: [edge-user-journey.md](./edge-user-journey.md) · [setup-lifecycle.md](./setup-lifecycle.md) · [memory-mcp.md](./memory-mcp.md) · [memory-edge-usage-demo.md](./memory-edge-usage-demo.md).

Slash continuum primary edge path: `/onboard next setup` · `journey` · `wizard` · `memory` · `e4` · `portal-hitl` (connectors when used).

---

## Platform residual honesty (optional · anti-claims)

These rails exist so residual eng and operators **do not invent** platform green from offline boards. They are **optional residual honesty**, not the Edge OSS product path and **not** the control plane:

| Rail | Purpose (anti-claims) | Surface (examples) |
|------|------------------------|--------------------|
| **Human-gates board** | Keep still-human APPLY inventory open; PASS ≠ invent human-gate green / live APPLY | `/onboard next human-gates` |
| **Soft residual-check** (`dogfood` slash token) | Offline string/board honesty check; never dial MCP / never start host | `/onboard next <lane> dogfood` (aliases `soft`/`samples`/`offline`/…) |
| **Still-human APPLY soft residual** | Reaffirm open boxes after Wave A–C continuum | `/onboard next human-gates dogfood` |
| **Tool-call residual** | Deeper E4 path map residual (ingest→retrieve→list→as-of) without inventing GA | `/onboard next tool-call` · soft residual-check |
| **E10 Open reaffirm (s1586)** | Pin E10 Open after packaging continuum; residual PASS ≠ invent E10 closed / Edge Memory GA declared | `/onboard next e10` · soft residual-check `/onboard next e10 dogfood` |
| **Book-demo OFF locks** | Never invent book-demo ON / public booking live | boards: demo · sales · human-gates · Locks lines |
| **Serial stamps (`sNNNN`)** | Internal residual/roadmap labels in CHANGELOG/docs | free eng serials · free-floor peer mention only |

**Locks shared with Edge path:** dual_write **OFF** · not Memory GA · Edge Memory GA **candidacy only** · book-demo **OFF** · residual PASS ≠ invent Edge Memory GA declared · residual PASS ≠ invent E10 closed · residual PASS ≠ invent control plane in MIT repo · E10 Open · catalog ≠ Connected · portal HITL when connect.

Keep soft residual-check harnesses — they are **anti-claim rails**, not live customer dogfood.

**Optional env label (s1723 · not a hiding gate):** operators or residual eng may set `IOMESH_PLATFORM_RESIDUAL=1` (also `true` / `yes`, case-insensitive) to label platform residual honesty surfaces. Helper `setup.PlatformResidualEnvOn` / `PlatformResidualLabelNote` shipped for **labeling only** — does **not** hide Edge OSS lanes · residual PASS ≠ invent control plane · free eng **s1723**. Continuum labeling already splits **Edge OSS path** vs **Platform residual honesty**; soft residual-checks remain available.

---

## Glossary

| Term | Meaning in this MIT repo |
|------|---------------------------|
| **soft dogfood** / **soft residual-check** | Offline residual honesty string/board check in-process · **≠** live customer dogfood · **≠** invent platform green · slash token remains `dogfood` for compatibility; user-facing phrase prefers **residual-check** |
| **session soft** | In-session marker after a soft residual-check run (`*_soft_not_run` · `soft_offline_*_session_pass|fail`) · **session soft ≠ live dogfood** |
| **residual PASS** | Offline board/soft harness PASS · **≠** invent Connected · Memory GA · dual_write ON · book-demo ON · control plane in MIT repo · forever-green product dogfood |
| **Edge Memory GA candidacy only** | Public edge attach path residual may candidacy; **not** bare Memory GA · **not** Edge Memory GA declared · **E10 Open** |
| **free eng `sNNNN`** | Internal free-engineering residual serial for continuum work · not a public product version claim |
| **free-floor peer** | Separate free-floor ownership serial continuum · **mention only** · packaging residual does not rewrite free-floor |
| **control plane** | Private multi-tenant platform (`aion` / portal / install-store fleet) · **not** shipped as open control-plane code in this MIT harness |

---

## How to read CHANGELOG serials (`sNNNN`)

CHANGELOG and deep architecture docs use serials like **s1578**, **s1582**:

- They are **internal residual / roadmap labels** for free eng continuum work.
- They help residual eng track honesty rails, evidence stamps, and anti-claim boards.
- They are **not** claims that the MIT repo is the multi-tenant control plane.
- They are **not** GA / Connected / dual_write ON product milestones.
- **free eng** owns the residual serial; **free-floor peer `sNNNN+`** is mention-only ownership elsewhere.

Example: free eng **s1582** = this OSS packaging residual · free-floor peer **s1584+** mention only.

---

## Operator surface packaging (s1582)

`/onboard next` continuum help is labeled in two groups:

1. **Edge OSS path** — setup · journey · wizard · memory · e4 attach · portal HITL for connectors when used  
2. **Platform residual honesty (optional · anti-claims · offline residual checks)** — human-gates · soft residual-check (`dogfood` slash) · still-human APPLY · tool-call residual · E10 Open reaffirm (`/onboard next e10` · s1586)  

Bare `/onboard` residual packaging line: `OSSPackagingHonestyOneLiner` in `internal/agent`  
API continuum overview: `AionAgentOnboardingNextLanes`

---

## Honesty locks (never invent)

- MIT OSS harness · **not** control plane in this repo  
- dual_write **OFF** · book-demo **OFF** · not Memory GA · Edge Memory GA **candidacy only**  
- residual PASS ≠ invent control plane in MIT repo  
- residual PASS ≠ invent Edge Memory GA declared · residual PASS ≠ live dogfood  
- soft residual-check (`… dogfood`) = offline residual honesty check · session soft ≠ live dogfood · ≠ invent platform green  
- public OSS ≠ invent platform GA · aion broker **private**  
- free eng **s1582** · free-floor peer **s1584+** mention only (do not rewrite free-floor)

## Non-goals

- Do **not** invent Edge Memory GA / Connected / dual_write ON / book-demo ON  
- Do **not** delete soft residual-check (soft dogfood) harnesses — keep residual anti-claim rails  
- Do **not** rewrite free-floor  
- Do **not** treat residual PASS / session soft as live customer dogfood or platform green  

## Related

- [edge-user-journey.md](./edge-user-journey.md) — 7-stage edge-first SSOT  
- [setup-lifecycle.md](./setup-lifecycle.md) — `/setup` residual map  
- [memory-mcp.md](./memory-mcp.md) — Edge OSS attach path  
- [memory-edge-usage-demo.md](./memory-edge-usage-demo.md) — usage/demo walkthrough  
- [../OPEN_SOURCE_AUDIT.md](../OPEN_SOURCE_AUDIT.md) — public launch audit checklist  
- Root [README.md](../../README.md) — OSS / local-primary table + packaging pointer  
- Skill `aion-agent-onboarding` — operator continuum  
- free eng **s1582** · free-floor peer **s1584+** mention only  
