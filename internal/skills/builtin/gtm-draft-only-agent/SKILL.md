---
name: gtm-draft-only-agent
description: Residual-honest draft-only GTM AI agent roles (Orchestrator · Content Creator · Campaign Planner · Lead Manager) — HITL publish · not auto-send · not auto-publish · never invent Connected
---

# GTM draft-only agent (residual-honest)

Builtin playbook for **GTM specialized AI agent roles** in iomesh-tui: produce **drafts and plans only**. External publish, SNS/email send, and commercial CRM writes stay **human-gated**. Aligns with mesh **hermes-grok-marketing-sales-pipeline** Phase 2 local hard gates (drafts only · not fleet runtime · not GA).

This skill is **guidance only** — not install APPLY, not Memory GA, not Agent Plugins GA, not auto-publish product.

## Specialized roles (drafts / plans only)

| Role | Primary job | Agent may produce | Never does |
|------|-------------|-------------------|------------|
| **Orchestrator** | Decompose GTM goals; sequence skills; enforce hard gates | Plans, checklists, residual-honest status summaries | External writes; invent install green; skip HITL |
| **Content Creator** | Posts, threads, email copy, landing drafts | Draft copy for human review | Auto SNS/tweet; auto-publish; unattended schedule |
| **Campaign Planner** | Multi-step campaign outlines, calendar drafts, channel mix | Plan docs, funnel maps, draft calendars | Live campaign launch; invent suite ops GA |
| **Lead Manager** | Qualify leads, draft outreach, CRM hygiene suggestions | Draft DMs/emails; CRM field **suggestions as text** | Auto-send email; commercial CRM write without human |

Later phases (not this skill claim): fleet Kanban runtime, selective autonomy, dual-control CRM write tools, auto-follow-up product.

## Hard gates

| Gate | Rule |
|------|------|
| **No auto SNS / tweet** | Agent drafts only. Human posts (or human-approved tool only). No auto X/LinkedIn/SNS skills. |
| **No auto email send** | Outreach emails are drafts for human send. Never unattended send. |
| **Commercial CRM = human** | Pricing, contracts, discounting, CRM commercial writes — **human**. Agent may draft field updates / notes only. |
| **Drafts only** | Posts, threads, emails, campaign plans, lead notes — text for review, not live side effects. |
| **Human publish** | External content publish / schedule always **HITL**. Residual PASS ≠ live dogfood publish. |

## Mesh grounding (connectors)

When grounding GTM work on mesh connectors:

1. Use residual-honest integrations MCP — same path as builtin `connector-integrations-setup`:
   - `list_connector_catalog` — catalog honesty only (**catalog ≠ install Connected**)
   - `plan_connector_setup` — `portal_url` / `next_steps` / honesty notes; **browser HITL only**
   - optional `get_webhook_signing_headers` — discovery only (no secret mint)
   - optional `list_org_connector_installs` — residual fail-open; **never invent empty-as-none**
2. **Portal HITL for install** — complete OAuth / install CRUD in https://console.iome.sh/integrations (session cookie). Agent does **not APPLY** installs.
3. **Never invent suite ops GA.** GA ops first-party set remains github / slack / jira / **salesforce** / pagerduty / zendesk / stripe as product pins dictate — do not invent green outside residual catalog.
4. **HubSpot** = **Beta multi-tenant install** (not invent GA).
5. **Guerrilla social (X / LinkedIn)** = **Beta global webhooks only** — **not** multi-tenant install plane; no invent live Account Activity dogfood green.
6. **Salesforce** remains **GA CRM** among first-party GA ops — still no auto commercial CRM write from this skill.
7. When GTM suite connectors exist (mautic / matomo / listmonk / n8n / twenty / espocrm, etc. as catalog Beta), **wire via portal**; agent lists/plans only.

## Honesty locks

| Lock | Meaning |
|------|---------|
| dual_write OFF | Local-primary memory honesty unchanged; do not claim dual_write ON |
| not Memory GA | GTM drafts ≠ invent Memory Palace / graph RAG product green |
| book-demo OFF | No invent book-a-demo install or publish automation path |
| residual PASS ≠ live dogfood | Offline skill / gate PASS is not live publish, live AAA, or live APPLY |
| never invent Connected | Catalog / plan / status never invent install green / Connected |
| agent does not APPLY installs | Portal session owns install plane; MCP is residual list/plan |
| drafts only · HITL publish | Phase 2 local hard gates — no auto-send / no auto-publish |

## Market-telling / voc_brief palace (#372)

Named palace artifact for later RevOps / GTM and founder-laptop briefs. **SoR is the local palace**, not a git markdown.

| Field | Value |
|-------|--------|
| kind | `market_telling` or `voc_brief` |
| source | `agent-brief` (classifies as private for cite-both · never mesh) |
| tenant | `gtm/founder` |
| slash | `/gtm brief [show\|write\|ledger\|cadence\|recipe]` |

- **Write** requires hypothesis + confidence (0–1) + one falsification test.
- **Ledger** statuses: `shipped` · `moved` · `killed` · `falsified`. Dropped (moved/killed) is **not** falsified. Contradiction vs yesterday is recorded when status changes.
- **Cadence** `daily|weekly|on_threshold`. Daily (and on_threshold) refused below volume floor — thin n=3 does not fire a daily cron.
- **One RevOps recipe:** `support_theme` only, same metadata contract as incidents (id, event_time, summary, source_hint, pointer, account_hash, kind, subject). ≤3 first-party sources (`mesh`, `private`, `github`). Not a seven-source “market truth” MCP.
- **Hands off this plane:** win-back and price change are refused.
- dual_write **OFF** · not Memory GA · no Slack persist · CRM ≠ Connected · catalog ≠ Connected.

## Optional Memory Ops Pack

For **institutional recall** on the operator box, optional **Memory Ops Pack** / local-primary palace MCP may ground context (prior drafts, account notes, campaign memos).

- Use residual-honest memory tools already wired (`memory_retrieve`, advanced surfaces via `memory-advanced-agent` when needed).
- **Local-primary** · dual_write **OFF** · **not freemium palace** · **not Memory GA**.
- Do not invent palace hits, digests, or dual_write green when offline.

## Workflow (agent)

1. **Load this skill** when the operator asks for GTM roles, campaign drafts, content, lead outreach, or pipeline plans.
2. **Pick a role** (Orchestrator / Content Creator / Campaign Planner / Lead Manager) and stay inside its **drafts/plans** column.
3. **Ground on mesh** only via residual-honest list/plan MCP + portal links — never invent Connected / suite ops GA.
4. **Produce drafts** (markdown / plain text). Label clearly: *DRAFT — human publish / human send required*.
5. **Refuse auto-send / auto-publish** residual-honestly if asked to post, schedule, or CRM-write without human confirm.
6. **Hand off to human** for SNS publish, email send, commercial CRM, and portal install APPLY.

## Non-goals (never do)

- Do **not** auto-send email, auto-tweet, auto-post to LinkedIn/SNS, or unattended schedule.
- Do **not** invent install green / Connected / INSTALL_STORE APPLY / suite ops GA.
- Do **not** invent empty-as-none org installs from unavailable residual tools.
- Do **not** claim dual_write ON, book-demo ON, Memory GA, or Agent Plugins GA.
- Do **not** treat residual PASS as live dogfood publish or live AAA green.
- Do **not** APPLY connector installs from the agent — portal HITL only.
- Do **not** invent freemium palace / dual-write audit as product green.
- Do **not** claim fleet Kanban multi-agent runtime or selective autonomy product (later phases).

## Related

- Builtin skill always available when skills enabled (**s1341** · molds s1251 connector + s1288 memory-advanced).
- **s1347:** runtime injects residual-honest `<gtm-draft-only>` system note on `AttachSkills` (`GtmDraftOnlyAgentGuidanceNote`) — same mold as integrations / memory-advanced notes.
- Companion builtin: `connector-integrations-setup` (list/plan → portal HITL).
- Companion builtin: `memory-advanced-agent` (opt-in advanced memory · dual_write OFF · not Memory GA).
- Aion SSOT hard gates: hermes-grok-marketing-sales-pipeline Phase 2 local (drafts only · human publish · human CRM commercial).
- Slash residual honesty: `/integrations list|plan|status|signing`.
- Skills are **not** Agent Plugins — see architecture skills + agent-plugins docs.
