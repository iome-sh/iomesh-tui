---
name: connector-integrations-setup
description: Residual-honest agent path to list/plan connector setup via MCP then portal browser HITL (not install APPLY)
---

# Connector integrations setup (residual-honest)

Agent path for **connector integrations setup** via aion MCP tools — **not** full install CRUD, **not** OAuth complete without browser, **not** INSTALL_STORE APPLY green.

## Workflow

1. **Discover** — call MCP `list_connector_catalog` (aion v178).
   - Returns `{count, entries[]}` with `id`, `status`, `mesh_layer`, `oauth_install_supported`, `portal_path`.
   - **Catalog status ≠ install Connected.** Status chips (`available` / `beta` / `planned`) are display honesty only.
   - Never invent install green or org Connected counts from the catalog.

2. **Plan** — call MCP `plan_connector_setup` with `connector_id`.
   - Surfaces `portal_url`, `portal_add_url`, `deep_links` (s1244 proven console routes), `oauth_mode_hint`, `signing_headers_tool`, `next_steps`, `honesty.notes`.
   - Deep links are **browser HITL only** — not install APPLY success.

3. **Optional signing discovery** — call MCP `get_webhook_signing_headers` (optional `mesh_layer`).
   - Header parity / scheme / primary header names only.
   - **Discovery only** — no secret mint, no rotate, no invent secrets.

4. **Org installs residual snapshot** — call MCP `list_org_connector_installs` with `org_id` (aion v179 / s1268 · TUI s1271/s1273).
   - Residual-honest **fail-open** by default: `available=false`, `status=unavailable`, `installs=null`.
   - **Never invent empty-as-none** — `installs=null` is residual honesty, **not** "none connected".
   - **dual_auth candidacy open** — not product dual-auth live green · **not Connected**.
   - Read-only residual tool only; portal session owns install index (cookie + org membership).

5. **Complete in browser portal HITL** — open https://console.iome.sh/integrations (session cookie).
   - OAuth authorize/callback and install CRUD live in the **console portal**, not agent MCP.
   - Agent MCP **cannot write installs**.

6. **Operator pulse** — slash `/integrations status|list|plan|signing` mirrors the same residual honesty for humans (status probes `list_org_connector_installs` when present).

## Non-goals (never do)

- Do **not** invent install green / Connected / INSTALL_STORE APPLY / GA.
- Do **not** invent empty-as-none installs from unavailable / `installs=null`.
- Do **not** complete OAuth without browser HITL (stub OAuth ≠ live).
- Do **not** claim dual_write ON or book-demo ON.
- Do **not** treat catalog Beta/available/planned as Connected/installed.
- Do **not** mint or rotate webhook secrets from discovery output.
- Do **not** invent focus= fantasy portal query params.

## Honesty locks

| Lock | Meaning |
|------|---------|
| never invent install green | Plan/list/status never claim Connected success |
| never invent empty-as-none | `list_org_connector_installs` unavailable/null ≠ none connected |
| stub ≠ live | OAuth mode stub is not live provider token exchange |
| browser HITL | Human finishes OAuth / install in console session |
| dual_write OFF | Local-primary memory honesty unchanged |
| book-demo OFF | No invent book-a-demo install path |
| catalog ≠ installs | Catalog inventory is honesty only, not install count |
| signing = discovery only | Header names only; no secret values |
| dual_auth candidacy open | Residual candidacy only — not dual-auth product green |

## Related

- Builtin skill always available when skills enabled (s1251 · s1273 residual installs).
- System note `<integrations>` injected on MCP attach.
- Slash: `/integrations list|plan|signing|status`.
