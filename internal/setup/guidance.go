package setup

import "strings"

// Portal URLs for residual-honest setup handoff (browser HITL).
const (
	PortalIntegrationsURL  = "https://console.iome.sh/integrations"
	PortalAgentSettingsURL = "https://console.iome.sh/settings/agent"
)

// SetupLifecycleHonestyOneLiner is the bare /setup status honesty line (s1526 P3 + s1530 P5 + s1534 P6 + s1538 P7 + s1558 Wave B).
const SetupLifecycleHonestyOneLiner = "dual_write OFF · not Memory GA · Edge Memory GA candidacy only · catalog ≠ Connected · portal HITL · setup PASS ≠ invent install green · continuous pull opt-in (/setup pull · pull_continuous) · analyze ticks opt-in (/setup analyze · analyze_continuous) · drift report-only (/setup drift) · guided repair (/setup repair · apply --yes only) · CLI iomesh memory pull still valid · /memory digest still valid · pull/analyze/repair ≠ invent Connected · drift ≠ invent install green · repair apply ≠ invent Connected · stage 4 of edge-user-journey · free eng s1558 · full first-run /onboard next journey"

// SetupLifecycleFirstRunJourneyOneLiner residual-honest companion for 7-stage first-run map (s1558 Wave B).
const SetupLifecycleFirstRunJourneyOneLiner = "edge-user-journey 7 stages · free eng s1558 · Signup → Download TUI → TUI auth/keys → Setup wizard (this lifecycle · stage 4) → Connectors portal HITL → Local store iomesh-memory-mcp → Analyze · dual_write OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA · portal HITL · host not auto · no invent TUI portal SSO · free-floor peer s1560+ mention only"

// SetupLifecycleAgentGuidanceNote is the residual-honest system note injected on
// AttachMCP (s1526 P3 + s1530 P5 + s1534 P6 + s1538 P7 + s1558 Wave B). Steers the LLM: setup init →
// preflight → portal HITL → in-session opt-in continuous pull / analyze ticks / drift
// report / guided repair — without inventing Connected / Memory GA / INSTALL_STORE green.
// s1558: maps onto stages 4–7 of edge-user-journey; full first-run map via /onboard next journey.
// Unit-tested for honesty needles.
func SetupLifecycleAgentGuidanceNote() string {
	return strings.TrimSpace(`setup lifecycle (residual-honest agent path · s1526 P3 / s1530 P5 / s1534 P6 / s1538 P7 / s1558 Wave B / skill setup-lifecycle-agent):
First-run journey map (s1558 · residual-honest · free eng s1558): 1 Signup (portal · optional pure local) · 2 Download TUI · 3 TUI auth/keys (LLM/Ollama · no invent portal SSO) · 4 Setup wizard (this lifecycle · /setup · /onboard next setup) · 5 Connectors MCP list/plan + portal HITL · 6 Local store iomesh-memory-mcp (host not auto) · 7 Analyze (/memory digest · /setup analyze). Full map: /onboard next journey · docs/architecture/edge-user-journey.md. In-session setup focuses stages 4–7 residual-honest.
1. Init managed config: slash /setup init [profiles] or CLI iomesh setup init — dual_write OFF · secrets env names only · pull_continuous=false · analyze_continuous=false default
2. Preflight probe: /setup preflight (aliases status|check) or iomesh setup preflight — state probe · PASS ≠ invent Connected
3. Portal HITL for OAuth/install: ` + PortalIntegrationsURL + ` · agent settings ` + PortalAgentSettingsURL + ` — agent MCP cannot write installs
4. Continuous pull in-session opt-in: /setup pull start|once|stop|status (loads [memory] pull_* · pull_continuous=true is config opt-in) · CLI iomesh memory pull still valid
5. Analyze ticks in-session opt-in (s1534 P6): /setup analyze start|once|stop|status (loads [memory] analyze_* · analyze_continuous=true is config opt-in · --mode status|digest · --interval N · --window day|week) · /memory digest still valid · analyze tick ≠ invent Connected
6. Drift / maintain report-only (s1534 P6): /setup drift · /setup maintain — FormatDriftText · residual next steps · drift report ≠ invent install green · package wire ≠ Connected
7. Guided repair (s1538 P7): /setup repair · /setup repair plan — PlanRepair from drift (dry plan) · /setup repair apply --yes — ApplyRepairPlan safe steps only (reload_mcp · start_pull · start_analyze) · refuse without --yes · notes for human host/mesh/dual_write · repair apply ≠ invent Connected · dual_write never auto-flipped ON · portal HITL still human
8. Skill: read_skill setup-lifecycle-agent when available · operator slash /setup (alias /setup-lifecycle) · companion /onboard next journey (s1558 first-run map) · /onboard next setup (stage 4 detail)

Locks (never violate):
- dual_write OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA · book-demo OFF · free eng s1558
- never invent install green / Connected / INSTALL_STORE APPLY / GA
- catalog status ≠ install Connected
- portal HITL for OAuth/install · secrets as env names only · no invent TUI portal SSO · host not auto on signup
- setup PASS / local_memory_probe_ok ≠ invent product green
- continuous pull is opt-in only · /setup pull · pull_continuous · CLI iomesh memory pull still valid · pull ≠ invent Connected
- analyze ticks are opt-in only · /setup analyze · analyze_continuous · /memory digest still valid · analyze tick ≠ invent Connected
- drift is report-only · /setup drift · /setup maintain · residual next steps · drift report ≠ invent install green · package wire ≠ Connected
- guided repair needs explicit /setup repair apply --yes · safe steps only · repair apply ≠ invent Connected · no auto-repair without --yes · dual_write never auto-flipped ON · portal HITL still human
- free-floor peer s1560+ mention only (do not rewrite free-floor)`)
}

// SetupLifecyclePortalHandoff residual-honest portal URLs for /setup portal.
func SetupLifecyclePortalHandoff() string {
	return strings.TrimSpace(`setup portal handoff (residual-honest · browser HITL · s1526):
  integrations: ` + PortalIntegrationsURL + `
  agent / MCP settings: ` + PortalAgentSettingsURL + `
  agent MCP cannot write installs · OAuth complete in console session
  honesty: ` + SetupLifecycleHonestyOneLiner)
}

// SetupInitNextStepLines is the residual-honest post-write next-step block for
// CLI `iomesh setup init` (s1686). Dual path: in-session /setup reload when a
// TUI/session is already running · cold start → restart iomesh. CLI has no
// `iomesh setup reload` subcommand — do not invent it. package wire ≠ Connected ·
// dual_write OFF · not Memory GA · free eng s1686.
func SetupInitNextStepLines() []string {
	return []string{
		"next: ensure iomesh-memory-mcp is running (if local-memory) · set secret env vars",
		"then: if TUI/session already running → /setup preflight · /setup reload (hot-swap MCP + skills · package wire ≠ Connected)",
		"      else cold start → restart iomesh · iomesh setup preflight",
		"note: CLI has no `iomesh setup reload` · in-session /setup reload only · dual_write OFF · not Memory GA · catalog ≠ Connected · free eng s1686",
	}
}

// SetupInitMeshNextStepLines is appended after mesh / platform-mcp init (s2055).
// IOMESH_TOKEN → reload → create stream → --messages. Create ≠ PULSE.
// Infer ≠ Connected · catalog MCP ≠ hooks streams · mesh pub ephemeral ≠ consume.
func SetupInitMeshNextStepLines() []string {
	return []string{
		"next (mesh): export IOMESH_TOKEN (env ref · never inline secret) · /setup reload (hot-swap mesh + MCP · infer ≠ Connected)",
		"then: iomesh mesh streams --create --yes  # create ≠ PULSE",
		"      iomesh mesh streams --messages --name OPERATIONAL_EVENTS",
		"note: listed stream + 0 messages is still empty · catalog MCP ≠ hooks streams · mesh pub ephemeral ≠ /dashboard consume · dual_write OFF · not Memory GA",
	}
}

// SetupPreflightNextStepLines residual-honest post-preflight next-step (s1699).
// Dual path: in-session /setup reload when TUI running · cold CLI → restart.
// CLI has no iomesh setup reload. dual_write OFF · package wire ≠ Connected · free eng s1699.
func SetupPreflightNextStepLines() []string {
	return []string{
		"next: if preflight ok and TUI/session already running → /setup reload (hot-swap MCP + skills · package wire ≠ Connected)",
		"      else if host/secrets still missing → start iomesh-memory-mcp · set secret env · re-run preflight",
		"      else cold start → restart iomesh (CLI has no setup reload) · then /setup reload in session if needed",
		"note: dual_write OFF · not Memory GA · catalog ≠ Connected · PASS ≠ invent install green · free eng s1699",
	}
}

// SetupDriftNextStepLines residual-honest post-drift next-step (s1707).
// Dual path: in-session /setup repair · /setup reload vs cold restart.
// CLI has no setup drift/repair/reload as full product surface. dual_write OFF ·
// package wire ≠ Connected · not Memory GA · free eng s1707.
func SetupDriftNextStepLines() []string {
	return []string{
		"next: if TUI/session running → /setup repair plan · /setup repair apply --yes (safe only) · /setup reload when MCP drift · optional /setup pull|analyze start",
		"      else cold start → fix host/config · iomesh setup preflight · restart iomesh (CLI has no setup drift/repair/reload)",
		"note: drift report-only · dual_write OFF · package wire ≠ Connected · not Memory GA · free eng s1707",
	}
}

// SetupRepairNextStepLines residual-honest post-repair plan/result next-step (s1707).
// Dual path: in-session re-run /setup drift · /setup reload vs cold restart.
// CLI has no setup repair/reload. repair apply ≠ invent Connected · dual_write never
// auto ON · package wire ≠ Connected · free eng s1707.
func SetupRepairNextStepLines() []string {
	return []string{
		"next: if TUI/session running → re-run /setup drift · /setup reload after safe apply · optional pull/analyze",
		"      else cold start → restart iomesh · iomesh setup preflight (CLI has no setup repair/reload)",
		"note: repair apply ≠ invent Connected · dual_write OFF · dual_write never auto ON · package wire ≠ Connected · not Memory GA · free eng s1707",
	}
}

// SetupReloadNextStepLines residual-honest post-reload next-step (s1711).
// Reload is in-session only. After hot-swap: optional pull/analyze · drift residual.
// package wire ≠ Connected · dual_write OFF · not Memory GA · free eng s1711.
func SetupReloadNextStepLines() []string {
	return []string{
		"next: optional /setup pull start (mesh+consumer) · /setup analyze start · /setup drift for residual",
		"      re-run /setup preflight if host/secrets still missing · portal HITL for installs",
		"note: reload in-session only · CLI has no `iomesh setup reload` · package wire ≠ Connected · dual_write OFF · not Memory GA · free eng s1711",
	}
}

// SetupPullNextStepLines residual-honest post-pull status/start next-step (s1711).
// Dual path: in-session /setup pull vs cold CLI iomesh memory pull.
// pull ≠ invent Connected · dual_write OFF · not Memory GA · free eng s1711.
func SetupPullNextStepLines() []string {
	return []string{
		"next: if TUI/session running → /setup pull start|once after mesh+pull_consumer · /setup pull status · optional /setup analyze|drift",
		"      else cold CLI → iomesh memory pull (still valid) · dual_write OFF · not Memory GA",
		"note: pull ≠ invent Connected · pull_continuous opt-in · CLI iomesh memory pull still valid · free eng s1711",
	}
}

// SetupAnalyzeNextStepLines residual-honest post-analyze status/start next-step (s1711).
// Dual path: in-session /setup analyze vs one-shot /memory digest.
// analyze tick ≠ invent Connected · dual_write OFF · not Memory GA · free eng s1711.
func SetupAnalyzeNextStepLines() []string {
	return []string{
		"next: if TUI/session running → /setup analyze start|once · /setup analyze status · optional /setup drift",
		"      else one-shot → /memory digest still valid · dual_write OFF · not Memory GA",
		"note: analyze tick ≠ invent Connected · analyze_continuous opt-in · /memory digest still valid · free eng s1711",
	}
}

// SetupPortalNextStepLines residual-honest post-/setup portal handoff next-step (s1723).
// Dual path: complete OAuth/install in browser HITL · then TUI/session → /setup preflight ·
// /setup reload (package wire ≠ Connected) · else cold → restart iomesh · iomesh setup preflight.
// CLI has no setup portal/reload as product invent. agent MCP cannot write installs ·
// catalog ≠ Connected · dual_write OFF · not Memory GA · free eng s1723.
func SetupPortalNextStepLines() []string {
	return []string{
		"next: complete OAuth/install in browser HITL · then if TUI/session running → /setup preflight · /setup reload (package wire ≠ Connected)",
		"      else cold start → restart iomesh · iomesh setup preflight (CLI has no setup portal/reload)",
		"note: agent MCP cannot write installs · catalog ≠ Connected · dual_write OFF · not Memory GA · free eng s1723",
	}
}
