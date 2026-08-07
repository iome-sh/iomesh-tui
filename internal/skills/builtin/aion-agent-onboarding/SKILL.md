---
name: aion-agent-onboarding
description: Residual-honest TUI agent ↔ aion CP/MCP onboarding (portal Agent/MCP mint/copy/probe · list/plan connectors → portal HITL · memory dual_write OFF · not Memory GA · never invent Connected)
---

# Aion agent onboarding (residual-honest)

Builtin playbook for **onboarding a TUI agent session against aion CP/MCP** — residual-honest path only. Molds `connector-integrations-setup` + operator onboarding checklist. **Not** install APPLY, **Not** Memory GA, **Not** Agent Plugins GA, **Not** dual_write ON.

**System note (s1363+s1368):** when MCP is attached (`AttachMCP`), runtime injects residual-honest `<aion-onboarding>` (`AionAgentOnboardingGuidanceNote`) with the same locks. Skill + note stay consistent; skill is the full playbook. Operator slash: `/onboard` (aliases `/aion-onboard` `/agent-onboard`) · `/onboard portal` · `/onboard status` · `/onboard checklist`.

## Workflow — two complementary halves

### A. Portal Agent/MCP half (credential → copy connection → test invoke · s1368)

1. **Mint credential in portal HITL** — open https://console.iome.sh/settings/agent (session cookie).
   - Mint API key / agent principal as needed (settings only).
   - **Not** install APPLY · **not** invent Connected from mint success.

2. **Settings → Agent/MCP → copy MCP connection**
   - Copy streamable HTTP MCP URL + auth env hint from the portal Agent/MCP panel.
   - Connection copy is **handoff only** — not Memory GA, not install green.

3. **Test invoke = probe only ≠ Memory GA**
   - Portal test invoke is a residual-honest **probe** (latency / tool snippet).
   - **Never invent** tool green, Memory Palace GA, or Connected from a probe.

### B. TUI half (`[[mcp.servers]]` streamable HTTP · operator pulse)

4. **Configure TUI MCP attach** — add `[[mcp.servers]]` with streamable HTTP `url` (+ `oauth_token_env` if needed).
   - Example shape (placeholders only — never invent live install green):
     ```toml
     [[mcp.servers]]
     name = "aion"
     url = "https://…/mcp"          # streamable HTTP from portal copy
     oauth_token_env = "AION_TOKEN" # env only — never commit secrets
     ```
   - Restart / reattach MCP. Offline / missing tools → residual-honest **fail-open**; **never invent** tool green or Connected.

5. **Operator pulse** — `/onboard` · `/onboard portal` · `/onboard status` · `/integrations status`.

### C. Connector integrations path (portal HITL · peer to Agent/MCP lane)

6. **Discover** — call MCP `list_connector_catalog` (aion v178).
   - Returns `{count, entries[]}` with `id`, `status`, `mesh_layer`, `oauth_install_supported`, `portal_path`.
   - **Catalog status ≠ install Connected.** Status chips (`available` / `beta` / `planned`) are display honesty only.
   - Never invent install green or org Connected counts from the catalog.
   - **No invent GA for knowledge/analytical** layers.

7. **Plan** — call MCP `plan_connector_setup` with `connector_id`.
   - Surfaces `portal_url`, `portal_add_url`, `deep_links`, `oauth_mode_hint`, `signing_headers_tool`, `next_steps`, `honesty.notes`.
   - Deep links are **browser HITL only** — not install APPLY success.

8. **Org installs residual snapshot** — call MCP `list_org_connector_installs` with `org_id` when present (aion v179 residual).
   - Residual-honest **fail-open** by default: `available=false`, `status=unavailable`, `installs=null`.
   - **Never invent empty-as-none** — `available=false` / `installs=null` is residual honesty, **not** "none connected".
   - Read-only residual tool only; portal session owns install index.

9. **Complete connector OAuth/install in browser portal HITL** — open https://console.iome.sh/integrations (session cookie).
   - OAuth authorize/callback and install CRUD live in the **console portal**, not agent MCP.
   - Agent MCP **cannot write installs** · never invent INSTALL_STORE APPLY.

10. **Memory residual** — dual_write **OFF** · **local-primary** · **not Memory GA**.
    - Optional advanced memory via `memory-advanced-agent` skill when needed (opt-in only).
    - Optional **plugins dogfood** (in-repo samples / offline validate) ≠ invent **Agent Plugins GA**.
    - Rates **~$88/$119 optional** — commercial framing only; not product GA claim.

11. **Operator pulse** — slash `/integrations status` · `/onboard checklist` · `/onboard portal` · portal HITL.

## Honesty locks

| Lock | Meaning |
|------|---------|
| dual_write OFF | Local-primary memory honesty; do not claim dual_write ON |
| book-demo OFF | No invent book-a-demo install path |
| not Memory GA | Onboarding ≠ invent Memory Palace / graph RAG product green; test invoke = probe only |
| residual PASS ≠ live dogfood | Offline skill / gate PASS is not live install, live AAA, or live APPLY |
| never invent install green | Plan/list/status never claim Connected / INSTALL_STORE APPLY success |
| available=false ≠ empty-as-none | `list_org_connector_installs` fail-open residual ≠ "no installs" |
| catalog ≠ Connected | Catalog status is display honesty only, not install Connected |
| portal HITL | Human finishes OAuth / install / key mint in console session |
| agent MCP cannot write installs | Portal session owns install plane; MCP is residual list/plan |
| plugins dogfood ≠ Agent Plugins GA | Sample/offline dogfood is not product GA; rates ~$88/$119 optional |
| no invent GA knowledge/analytical | Do not invent GA for knowledge or analytical mesh layers |

## Non-goals (never do)

- Do **not** invent install green / Connected / INSTALL_STORE APPLY / GA.
- Do **not** invent empty-as-none installs from unavailable / `installs=null` / `available=false`.
- Do **not** complete OAuth without browser HITL (stub OAuth ≠ live).
- Do **not** claim dual_write ON, book-demo ON, Memory GA, or Agent Plugins GA.
- Do **not** treat catalog Beta/available/planned as Connected/installed.
- Do **not** treat residual PASS as live dogfood publish or live APPLY green.
- Do **not** invent GA for knowledge/analytical connectors or digests.
- Do **not** invent freemium palace / dual-write audit as product green.
- Do **not** treat portal test invoke / mint key as Memory GA or install Connected.

## Related

- Builtin skill always available when skills enabled (**s1363+s1368** · molds s1251 connector + s1288 memory-advanced + s1341 gtm-draft-only).
- System note inject on `AttachMCP`: `<aion-onboarding>` via `AionAgentOnboardingGuidanceNote` (s1363+s1368).
- Portal handoff block: `AionAgentOnboardingPortalHandoff` · slash `/onboard portal` (aliases `agent-mcp` / `mcp`).
- Offline status: `AionAgentOnboardingStatus` · slash `/onboard status` (no MCP dial).
- Companion builtin: `connector-integrations-setup` (list/plan → portal HITL).
- Companion builtin: `memory-advanced-agent` (opt-in advanced memory · dual_write OFF · not Memory GA).
- Companion builtin: `gtm-draft-only-agent` (drafts only · human publish · no auto-send).
- Slash residual honesty: `/onboard [help|checklist|portal|status]` · `/integrations list|plan|status|signing` · `/memory status`.
- Skills are **not** Agent Plugins — plugins dogfood ≠ invent Agent Plugins GA.
