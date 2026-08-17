package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/iome-sh/iomesh-tui/internal/agentplugins"
)

// AionAgentOnboardingGuidanceNote residual-honest system note (s1363 + s1368 + s1372 + s1377 + s1382 + s1387 + s1402 + s1407 + s1413 + s1417 + s1432 + s1437 + s1442 + s1447 + s1542).
// Injected on AttachMCP after integrations + memory-advanced notes.
// Steers TUI agent ↔ aion CP/MCP onboarding without inventing install green,
// Memory GA, Agent Plugins GA, or dual_write ON.
// s1368: adds explicit portal Agent/MCP handoff lane (mint key → copy MCP →
// test invoke probe only) complementary to integrations portal HITL.
// s1372: cross-link post-onboard continuum → /onboard next operator lanes.
// s1377: /onboard next <lane> drill-down (plugins · gtm · memory).
// s1382: /onboard next status lane status board (pulse|board aliases).
// s1387: /onboard next export status export receipt (aliases receipt|stamp|evidence).
// s1402: /onboard next mesh streaming lane (org heartbeats on dept.* · mesh ≠ memory).
// s1407: /onboard next memory-pull Ops Pack pull path (mesh → local palace egress).
// s1413: /onboard next human-gates residual-honest still-required vs offline APPLY honesty.
// s1417: /onboard next agentic product plane 3 agentic integrations (MCP list/plan + portal HITL).
// s1432: /onboard next planes residual-honest three product planes board (mesh · memory-pull · agentic).
// s1437: /onboard next sales residual-honest sales/buyer claims board (may claim / must not claim).
// s1442: /onboard next demo residual-honest demo readiness board (Lighthouse · book-demo OFF · Landgrab NOT READY).
// s1447: /onboard next operator residual-honest operator readiness matrix (demo · sales · planes · human-gates).
// s1542: /onboard next setup residual-honest setup lifecycle P1–P7 closeout map (init→preflight→reload→portal→pull→analyze→drift→repair).
// s1546: still-human APPLY reaffirm after setup closeout — setup residual ≠ invent human-gate green / live APPLY.
// s1550: edge-first human-gates residual pin — knowledge multi-tenant punted · Slack HMAC punted · portal HITL when connect.
// s1558 Wave B: /onboard next journey residual-honest 7-stage edge-user-journey first-run map (setup = stage 4).
// s1562: /onboard next portal-hitl residual-honest journey stage-5 portal HITL board + soft offline dogfood.
// s1566: /onboard next e4 residual-honest journey stage-6 E4 client-attach board + soft offline dogfood.
// s1570 Wave C: /onboard next wizard residual-honest guided first-run wizard residual + soft offline dogfood.
// s1574: still-human APPLY soft dogfood residual after Wave C continuum — open boxes stay open · never invent human-gate green / live APPLY.
// s1578: /onboard next tool-call residual-honest deeper tool-call soft dogfood (E4 path depth beyond tools=6 attach: ingest→retrieve→list→as-of).
// s1582: OSS packaging residual — MIT harness boundary · Edge OSS path first · optional platform residual honesty (anti-claims).
// s1586: /onboard next e10 residual-honest E10 Open reaffirm residual-check after OSS packaging continuum.
// s1590: /onboard next marketing-demo plain-language marketing demo path (local agent + local memory for videos/sales).
// Unit-tested for honesty needles. Molds IntegrationsAgentGuidanceNote /
// GtmDraftOnlyAgentGuidanceNote / MemoryAdvancedAgentGuidanceNote.

// OSSPackagingHonestyOneLiner residual-honest MIT OSS packaging boundary (s1582).
// Used on bare /onboard residual packaging line and continuum help. Prefer user-facing
// "residual-check" alongside slash token dogfood. Never invents control plane / Memory GA /
// dual_write ON / book-demo ON / live dogfood green.
const OSSPackagingHonestyOneLiner = "MIT OSS harness · not control plane · dual_write OFF · not Memory GA · Edge Memory GA candidacy only · book-demo OFF · residual PASS ≠ invent control plane in MIT repo · soft residual-check (… dogfood slash) = offline residual honesty check · session soft ≠ live dogfood · ≠ invent platform green · free eng s1582 · free-floor peer s1584+ mention only · docs/architecture/oss-packaging-boundary.md"

// OnboardNextStepLines residual-honest post /onboard maps (s1825).
// Dual path after status/checklist/next lanes/portal handoff: in-session setup continuum
// vs cold start. Peer of IntegrationsNextStepLines (s1727) · setup next-step continuum (s1686–s1723).
// dual_write OFF · package wire ≠ Connected · catalog ≠ Connected · agent MCP cannot write installs ·
// not Memory GA · free eng s1825. Never invent Connected / Memory GA from onboard maps alone.
//
// AionAgentOnboardingNextStepLines is the same helper (family alias).
func OnboardNextStepLines() []string {
	return []string{
		"next: dual path residual-honest after onboard maps",
		"      if TUI/session running → /setup preflight · /setup reload · optional /integrations list · /onboard next portal-hitl|setup|memory",
		"      else cold start → restart iomesh · iomesh setup preflight",
		"note: dual_write OFF · package wire ≠ Connected · catalog ≠ Connected · agent MCP cannot write installs · not Memory GA · free eng s1825",
	}
}

// AionAgentOnboardingNextStepLines is the family-named alias for OnboardNextStepLines (s1825).
func AionAgentOnboardingNextStepLines() []string { return OnboardNextStepLines() }

// AionAgentOnboardingStartHere is the lean first-run path shown above residual
// boards (s1982 TUI UX · peer console s1981). Honesty needles stay in the
// residual body. Never invents Connected / Memory GA / install APPLY.
func AionAgentOnboardingStartHere() string {
	return strings.TrimSpace(`start here (TUI agent · MCP · integrations):
  1. Portal: https://console.iome.sh/settings/agent — mint iomesh_ag_* → export IOMESH_TOKEN → copy TUI fragment ([[mcp.servers]] + [iomesh]) → test invoke (stub|live)
  2. TUI: paste both blocks (apiv1 /v7/mcp catalog ≠ hooks.iome.sh streams) → restart / reattach
  3. Sources: /integrations list · /integrations plan <id> — finish in portal HITL
     https://console.iome.sh/integrations  (agent MCP cannot write installs)
  4. Local: /setup init · /setup preflight · /setup reload
  5. Map: /onboard next wizard · /onboard next journey
operator notes: /onboard next  (lanes · residual boards · never invent Connected)`)
}

func AionAgentOnboardingGuidanceNote() string {
	return AionAgentOnboardingStartHere() + "\n\n" + strings.TrimSpace(`aion agent onboarding (residual-honest TUI ↔ aion CP/MCP · s1363+s1368+s1372+s1377+s1382+s1387+s1402+s1407+s1413+s1417+s1432+s1437+s1442+s1447+s1542+s1546+s1550+s1558+s1562+s1566+s1570+s1574+s1578+s1582+s1586+s1590):
Point IOMESH/MCP at aion tools — fail-open offline (never invent tool green).

Connector path (integrations portal HITL · product plane 3 agentic integrations):
1. Discover: MCP list_connector_catalog — catalog status ≠ install Connected · catalog ≠ Connected
2. Plan: MCP plan_connector_setup — portal deep links + honesty notes (browser HITL only · template= ≠ install APPLY green)
3. Org installs residual: MCP list_org_connector_installs — fail-open (available=false · installs=null) ≠ empty-as-none · never invent Connected
4. Complete OAuth/install in portal HITL at https://console.iome.sh/integrations — agent MCP cannot write installs

Portal Agent/MCP lane (complementary · s1368 · credential → copy connection → test invoke):
- Portal: mint iomesh_ag_* → export IOMESH_TOKEN → Settings → Agent/MCP → copy TUI fragment ([[mcp.servers]] + [iomesh]) → test invoke (stub|live · 42ms/no preview = stub · ≠ live tools/call · ≠ consume · ≠ Memory GA)
- TUI: paste both blocks (streamable HTTP portal MCP + [iomesh] broker) — apiv1.iome.sh/v7/mcp catalog ≠ hooks.iome.sh streams. mesh streams without [iomesh] is mesh disabled, not an MCP failure
- Console Agent/MCP: https://console.iome.sh/settings/agent (connectors still /integrations)

Memory + operator:
5. Memory: dual_write OFF · local-primary · not Memory GA · optional plugins dogfood ≠ invent Agent Plugins GA (Base ~$132 · Memory Ops Pack hidden public · local memory free OSS)
6. Operator pulse: /integrations status · /onboard checklist · /onboard portal · portal HITL
7. Post-onboard continuum: /onboard next [plugins|gtm|memory|mesh|memory-pull|agentic|portal-hitl|e4|tool-call|e10|planes|sales|demo|marketing-demo|operator|setup|journey|wizard|status|export|human-gates] (plugins dogfood · /gtm checklist · iomesh-memory-mcp local · mesh streaming heartbeats · Ops Pack pull path · agentic integrations MCP list/plan · portal HITL stage-5 connectors · E4 client-attach stage-6 · deeper tool-call residual · E10 Open reaffirm residual-check · three product planes board · sales/buyer claims · demo readiness · marketing demo path (local agent + memory) · operator readiness matrix · setup lifecycle P1–P7 map · edge-user-journey first-run map · Wave C first-run wizard residual · lane status board · status export receipt · human-gates still-required vs offline)
8. Human gates (s1413+s1546+s1550+s1574 Wave C continuum): /onboard next human-gates — still-human APPLY open · edge-first · knowledge multi-tenant punted · Slack HMAC punted · portal HITL when connect · book-demo OFF · ON_SIGNAL unset · dual_write OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA · residual PASS ≠ invent Edge Memory GA declared · E10 Open · soft /onboard next human-gates dogfood · free eng s1574 · never invent Connected
9. Agentic integrations (s1417 · product plane 3): /onboard next agentic — MCP list/plan residual-honest · plan_connector_setup → portal deep links · browser HITL only · list_org fail-open ≠ empty-as-none · never invent Connected
10. Three product planes (s1432): /onboard next planes — mesh · memory-pull · agentic residual-honest consolidate · streams_not_probed · pull_not_probed · list_plan_not_connected · dual_auth_candidacy_open · never invent Connected
11. Sales/buyer claims (s1437): /onboard next sales — may claim / must not claim residual-honest · three-planes grounded · never invent Connected / Memory GA / dual-auth live
12. Demo readiness (s1442): /onboard next demo — Lighthouse beachhead packaging · book-demo OFF · Landgrab NOT READY · three planes · sales claims · human gates still open · never invent Connected
13. Operator readiness matrix (s1447): /onboard next operator — consolidate demo · sales · planes · human-gates · dual-auth candidacy · policy locks residual-honest · never invent Connected / GA
14. Setup lifecycle map (s1542+s1558 · stage 4 of edge-user-journey · P1–P7 closeout residual): /onboard next setup — init → preflight → reload → portal HITL → pull → analyze → drift → repair plan/apply --yes · setup_not_probed · offline static ≠ live dogfood · never invent Connected
15. Edge-user-journey first-run map (s1558 Wave B · 7 stages): /onboard next journey — Signup → Download TUI → TUI auth/keys → Setup wizard → Connectors → Local store → Analyze · dual_write OFF · Edge Memory GA candidacy only · free eng s1558
16. Portal HITL connectors (s1562 · journey stage 5): /onboard next portal-hitl — MCP list/plan → browser portal HITL → human OAuth/install · soft dogfood residual · free eng s1562
17. E4 client attach (s1566 · journey stage 6): /onboard next e4 — iomesh-memory-mcp local-primary · client attach · tools=6 · iomesh mcp --connect residual · soft dogfood residual · free eng s1566
18. First-run wizard residual (s1570 Wave C): /onboard next wizard — guided first-run residual map + soft dogfood · NOT invent full interactive auto wizard · free eng s1570
19. Still-human APPLY soft residual (s1574 Wave C continuum): /onboard next human-gates dogfood — open boxes stay open · PASS ≠ invent human-gate green · PASS ≠ live APPLY · free eng s1574
20. Deeper tool-call residual (s1578 · stage 6/7 depth after E4 attach): /onboard next tool-call — ingest→retrieve→list→as-of operator map soft residual · free eng s1578
21. E10 Open reaffirm residual-check (s1586 · Platform residual honesty after OSS packaging): /onboard next e10 — residual PASS ≠ invent E10 closed · residual PASS ≠ invent Edge Memory GA declared · founder sign-off only if declaring Edge Memory GA · candidacy allowed without E10 · soft residual-check /onboard next e10 dogfood · free eng s1586
22. Marketing demo path (s1590 · videos/sales · local agent + local memory): /onboard next marketing-demo — plain-language operator script · dual_write OFF · local memory · not Memory GA · mesh optional · never invent Connected · free eng s1590

Skill: read_skill aion-agent-onboarding when available

Locks (never violate):
- dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood
- never invent install green / Connected / INSTALL_STORE APPLY
- list_org_connector_installs available=false ≠ empty-as-none
- catalog status ≠ Connected · portal HITL for OAuth/install · agent MCP cannot write installs
- plugins dogfood ≠ invent Agent Plugins GA · Base ~$132 · Memory Ops Pack hidden public · local memory free OSS
- no invent GA for knowledge/analytical
- test invoke = stub|live probe ≠ Memory GA · mint iomesh_ag_* ≠ invent install Connected
- drafts only · no auto-send · package load ≠ Memory GA
- board/export evidence ≠ invent Connected
- mesh = streaming org heartbeats · mesh ≠ memory · never invent stream green / Connected · not OTel/APM
- pull = mesh → local palace egress · pull ≠ freemium hosted palace · Ops Pack ≠ GPU fleet · pull_not_probed honest
- PASS ≠ invent human-gate green · PASS ≠ live APPLY · open boxes stay open · leave ON_SIGNAL unset
- Knowledge Beta→GA cannot invent H1/H2 offline · local memory / dual_write OFF / agent MCP list/plan do not close human APPLY gates
- setup closeout residual ≠ invent APPLY (s1546) · E10 Open · residual PASS ≠ invent Edge Memory GA
- agentic: MCP list/plan residual-honest · plan deep links = browser HITL only · template= ≠ install APPLY · list_plan_not_connected · never invent Connected
- planes: mesh · memory-pull · agentic consolidate · never invent stream green / pull green / Connected · dual_auth_candidacy_open
- sales claims: may claim residual-honest only · must not invent Connected / Memory GA / dual-auth live / human-gate green
- demo readiness: Lighthouse packaging · book-demo OFF · Landgrab NOT READY · residual PASS ≠ logos met · founder-led walkthrough only when scheduled · never invent book-demo ON / Connected
- marketing demo path (s1590): dual_write OFF · local memory · not Memory GA · mesh optional · never invent Connected · book-demo OFF · free eng s1590
- operator matrix: residual_only · path_ready · still_human · policy_off · not_ready · portal_hitl_still · dual_auth_candidacy_open · never invent Connected / GA / dual-auth live
- setup lifecycle: dual_write OFF · not Memory GA · PASS ≠ invent Connected · package wire ≠ Connected · repair apply ≠ invent Connected · dual_write never auto ON · E10 Open · setup_not_probed · offline static lane ≠ live dogfood · setup closeout residual ≠ invent Edge Memory GA
- edge-user-journey (s1558): dual_write OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA · portal HITL · no invent TUI portal SSO · host not auto · book-demo OFF · free eng s1558 · free-floor peer s1560+ mention only
- portal HITL stage 5 (s1562): portal HITL when connect · agent MCP cannot write installs · catalog ≠ Connected · soft offline ≠ invent Connected · session soft ≠ live dogfood · residual PASS ≠ live dogfood · free eng s1562 · free-floor peer s1564+ mention only
- E4 client attach stage 6 (s1566): dual_write OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · E10 Open · tip ≠ invent forever-green product dogfood · residual PASS ≠ live dogfood · session soft ≠ live dogfood · soft offline ≠ invent Connected · free eng s1566 · free-floor peer s1568+ mention only
- first-run wizard residual (s1570 Wave C): dual_write OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · E10 Open · portal HITL when connect · agent MCP cannot write installs · catalog ≠ Connected · no invent TUI portal SSO · host not auto · residual PASS ≠ invent full interactive auto wizard · session soft ≠ live dogfood · free eng s1570 · free-floor peer s1572+ mention only
- deeper tool-call residual (s1578): dual_write OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · E10 Open · tip ≠ invent forever-green product dogfood · residual PASS ≠ live dogfood · session soft ≠ live dogfood · soft offline ≠ invent Connected · free eng s1578 · free-floor peer s1580+ mention only
- E10 Open reaffirm residual-check (s1586): dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · residual PASS ≠ invent E10 closed · E10 Open · founder sign-off only if declaring Edge Memory GA · candidacy allowed without E10 · PASS ≠ live APPLY · residual PASS ≠ live dogfood · session soft ≠ live dogfood · residual-check · free eng s1586 · free-floor peer s1588+ mention only`)
}

// AionAgentOnboardingChecklist residual-honest numbered onboarding checklist (s1363 + s1368 + s1372 + s1377 + s1382 + s1387 + s1402 + s1407 + s1413 + s1417 + s1432 + s1437 + s1442 + s1447 + s1542).
// Used by /onboard help and /onboard checklist — operator HITL only; never invents
// install green, Memory GA, Agent Plugins GA, dual_write ON, or agent APPLY.
// s1368: portal Agent/MCP handoff steps (mint/copy/probe) + TUI [[mcp.servers]].
// s1372: cross-link → /onboard next operator lanes (post-onboard continuum).
// s1377: /onboard next [plugins|gtm|memory] lane drills.
// s1382: /onboard next status lane status board.
// s1387: /onboard next export status export receipt.
// s1402: /onboard next mesh streaming lane (org heartbeats).
// s1407: /onboard next memory-pull Ops Pack pull path.
// s1413: /onboard next human-gates residual-honest still-required vs offline.
// s1417: /onboard next agentic product plane 3 agentic integrations (MCP list/plan + portal HITL).
// s1432: /onboard next planes residual-honest three product planes board (mesh · memory-pull · agentic).
// s1437: /onboard next sales residual-honest sales/buyer claims board (may / must-not).
// s1442: /onboard next demo residual-honest demo readiness board (Lighthouse · Landgrab NOT READY).
// s1447: /onboard next operator residual-honest operator readiness matrix (demo · sales · planes · human-gates).
// s1542: /onboard next setup residual-honest setup lifecycle P1–P7 closeout map.
// s1558 Wave B: /onboard next journey residual-honest 7-stage edge-user-journey first-run map.
// s1562: /onboard next portal-hitl residual-honest journey stage-5 portal HITL + soft offline dogfood.
// s1566: /onboard next e4 residual-honest journey stage-6 E4 client-attach + soft offline dogfood.
// s1570 Wave C: /onboard next wizard residual-honest guided first-run wizard residual + soft offline dogfood.
func AionAgentOnboardingChecklist() string {
	return AionAgentOnboardingStartHere() + "\n\n" + strings.TrimSpace(`aion agent onboarding checklist (residual-honest · s1363+s1368+s1372+s1377+s1382+s1387+s1402+s1407+s1413+s1417+s1432+s1437+s1442+s1447+s1542+s1558+s1562+s1566+s1570 · TUI ↔ aion):
  1. Point IOMESH/MCP at aion tools (fail-open offline)
  2. list_connector_catalog — catalog status ≠ Connected
  3. plan_connector_setup → portal deep links (browser HITL · template= ≠ install APPLY)
  4. list_org_connector_installs residual fail-open (available=false ≠ empty-as-none)
  5. Portal Agent/MCP: mint iomesh_ag_* → export IOMESH_TOKEN → copy TUI fragment ([[mcp.servers]] + [iomesh]) → test invoke (stub|live · ≠ consume) at https://console.iome.sh/settings/agent
  6. TUI: paste both blocks (streamable HTTP portal MCP + [iomesh] broker) → /onboard · /integrations status (agent MCP cannot write installs)
  7. Memory dual_write OFF · local-primary · not Memory GA · optional plugins dogfood ≠ Agent Plugins GA
  8. Operator: /integrations status · /onboard checklist · /onboard portal · portal https://console.iome.sh/integrations
  9. Post-onboard: /onboard next [plugins|gtm|memory|mesh|memory-pull|agentic|portal-hitl|e4|planes|sales|demo|operator|setup|journey|wizard|status|export|human-gates] (plugins · gtm · memory local · mesh streaming heartbeats · Ops Pack pull path · agentic integrations MCP list/plan · portal HITL stage-5 connectors · E4 client-attach stage-6 · three product planes board · sales/buyer claims · demo readiness · operator readiness matrix · setup lifecycle P1–P7 map · edge-user-journey first-run map · Wave C first-run wizard residual · lane status board · status export receipt · human-gates still-required vs offline)
  10. Human gates: /onboard next human-gates — still-human APPLY · Slack HMAC · Stripe Customers:Write · H1/H2 INSTALL_STORE · D1–D5 · book-demo OFF · ON_SIGNAL unset · PASS ≠ invent human-gate green · PASS ≠ live APPLY · open boxes stay open · soft /onboard next human-gates dogfood (s1574) · never invent APPLY
  11. Agentic integrations (product plane 3): /onboard next agentic — MCP list/plan residual-honest · plan_connector_setup → portal deep links · browser HITL only · list_org fail-open ≠ empty-as-none · catalog ≠ Connected · never invent Connected
  12. Three product planes (s1432): /onboard next planes — mesh · memory-pull · agentic residual-honest consolidate · streams_not_probed · pull_not_probed · list_plan_not_connected · dual_auth_candidacy_open · never invent Connected
  13. Sales/buyer claims (s1437): /onboard next sales — may claim / must not claim residual-honest · three-planes grounded · never invent Connected / Memory GA / dual-auth live
  14. Demo readiness (s1442): /onboard next demo — Lighthouse beachhead packaging · book-demo OFF · Landgrab NOT READY · three planes · sales claims · human gates still open · never invent Connected
  15. Operator readiness matrix (s1447): /onboard next operator — consolidate demo · sales · planes · human-gates · dual-auth candidacy · policy locks residual-honest · never invent Connected / GA
  16. Setup lifecycle map (s1542+s1558 · stage 4 of edge-user-journey): /onboard next setup (aliases setup-lifecycle|lifecycle|setup_lifecycle) — init → preflight → reload → portal HITL → pull → analyze → drift → repair plan/apply --yes · setup_not_probed · repair apply ≠ invent Connected · dual_write never auto ON · E10 Open
  17. Edge-user-journey first-run map (s1558 Wave B): /onboard next journey (aliases edge-journey|user-journey|first-run|edge_user_journey) — 7 stages residual-honest · dual_write OFF · Edge Memory GA candidacy only · free eng s1558
  18. Portal HITL connectors (s1562 · journey stage 5): /onboard next portal-hitl (aliases hitl|portal_hitl|portal-dogfood|stage5|connectors-hitl) — MCP list/plan → browser portal HITL · soft dogfood residual · free eng s1562
  19. E4 client attach (s1566 · journey stage 6): /onboard next e4 (aliases e4-dogfood|client-attach|edge-memory-e4|e4_attach) — iomesh-memory-mcp local-primary · tools=6 · iomesh mcp --connect residual · soft dogfood residual · free eng s1566
  20. First-run wizard residual (s1570 Wave C): /onboard next wizard (aliases first-run-wizard|guided|wave-c|wave_c|wizard-residual) — guided first-run residual map + soft dogfood · NOT invent full interactive auto wizard · free eng s1570
  Locks: never invent install green / Connected / INSTALL_STORE APPLY · book-demo OFF · Landgrab NOT READY · residual PASS ≠ live dogfood · PASS ≠ live APPLY · rates ~$88/$119 optional · no invent GA knowledge/analytical · catalog status ≠ Connected · portal HITL · drafts only · no auto-send · package load ≠ Memory GA · board/export evidence ≠ invent Connected · mesh = streaming org heartbeats · mesh ≠ memory · never invent stream green · pull ≠ freemium hosted palace · Ops Pack ≠ GPU fleet · pull_not_probed · open boxes stay open · Knowledge Beta→GA cannot invent H1/H2 offline · leave ON_SIGNAL unset · list_plan_not_connected · plan deep links = browser HITL only · template= ≠ install APPLY · dual_auth_candidacy_open · sales claims residual-honest only · residual PASS ≠ logos met · setup_not_probed · package wire ≠ Connected · repair apply ≠ invent Connected · dual_write never auto ON · E10 Open · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA · residual PASS ≠ invent Edge Memory GA declared · free eng s1558 · free eng s1562 · free eng s1566 · free eng s1570 · soft offline ≠ invent Connected · free-floor peer s1564+ mention only · free-floor peer s1568+ mention only · free-floor peer s1572+ mention only`) + "\n" + strings.Join(OnboardNextStepLines(), "\n")
}

// AionAgentOnboardingPortalHandoff residual-honest short block for /onboard portal (s1368).
// Portal Agent/MCP lane complementary to integrations portal HITL — mint key /
// copy MCP connection / test invoke (probe only) + TUI [[mcp.servers]] attach.
// Never invents install green, Memory GA, or agent write installs.
func AionAgentOnboardingPortalHandoff() string {
	return AionAgentOnboardingStartHere() + "\n\n" + strings.TrimSpace(`aion portal Agent/MCP handoff (residual-honest · s1368+s1542):
Portal half (browser HITL · https://console.iome.sh/settings/agent):
  1. Mint iomesh_ag_* principal (settings only · not install APPLY · export IOMESH_TOKEN)
  2. Settings → Agent/MCP → copy TUI fragment ([[mcp.servers]] portal MCP + [iomesh] hooks)
  3. Test invoke = stub|live probe · 42ms/no preview = stub · ≠ live tools/call · ≠ consume · ≠ Memory GA

TUI half (local config · streamable HTTP):
  4. Paste both blocks — /v7/mcp is catalog; hooks.iome.sh is streams (mesh disabled without [iomesh])
  5. Restart / reattach MCP → /onboard · /integrations status · /onboard status
  6. Connector OAuth/install still portal HITL at https://console.iome.sh/integrations — agent MCP cannot write installs
  7. Setup lifecycle companion (s1542+s1558 · stage 4): /onboard next setup · /setup portal · map init→preflight→reload→portal HITL→pull→analyze→drift→repair · dual_write OFF · package wire ≠ Connected · repair apply ≠ invent Connected · E10 Open
  8. Human-gates companion after setup (s1546): /onboard next human-gates — setup closeout residual ≠ invent APPLY · open boxes stay open
  9. First-run journey map (s1558 Wave B): /onboard next journey — 7-stage edge-user-journey residual-honest · free eng s1558
  10. Portal HITL connectors (s1562 · journey stage 5): /onboard next portal-hitl — browser portal HITL when connect · soft dogfood residual · free eng s1562
  11. E4 client attach (s1566 · journey stage 6): /onboard next e4 — iomesh-memory-mcp local-primary · client attach residual · soft dogfood residual · free eng s1566
  12. First-run wizard residual (s1570 Wave C): /onboard next wizard — guided first-run residual map + soft dogfood · free eng s1570

Locks: dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · never invent install green / Connected / INSTALL_STORE APPLY · list_org fail-open ≠ empty-as-none · catalog ≠ Connected · portal HITL · plugins dogfood ≠ invent Agent Plugins GA · setup closeout residual ≠ invent Edge Memory GA · still-human APPLY open · setup closeout residual ≠ invent APPLY · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · E10 Open · free eng s1562 · free eng s1566 · free eng s1570 · free-floor peer s1564+ mention only · free-floor peer s1568+ mention only · free-floor peer s1572+ mention only`) + "\n" + strings.Join(OnboardNextStepLines(), "\n")
}

// AionAgentOnboardingStatus residual-honest static offline status lines for /onboard status (s1368 + s1372 + s1377 + s1382 + s1387 + s1402 + s1407 + s1413 + s1417 + s1432 + s1437 + s1442 + s1447 + s1542).
// No MCP dial — operator pulse only. Never invents attach green, install Connected, or Memory GA.
// s1372: cross-link → /onboard next operator lanes.
// s1377: lane drills via /onboard next [plugins|gtm|memory].
// s1382: cross-link → /onboard next status lane status board.
// s1387: cross-link → /onboard next export status export receipt.
// s1402: cross-link → /onboard next mesh streaming lane.
// s1407: cross-link → /onboard next memory-pull Ops Pack pull path.
// s1413: cross-link → /onboard next human-gates residual-honest still-required vs offline.
// s1417: cross-link → /onboard next agentic product plane 3 agentic integrations.
// s1432: cross-link → /onboard next planes three product planes board.
// s1437: cross-link → /onboard next sales sales/buyer claims board.
// s1442: cross-link → /onboard next demo demo readiness board.
// s1447: cross-link → /onboard next operator operator readiness matrix.
// s1542: cross-link → /onboard next setup setup lifecycle P1–P7 closeout map.
// s1558 Wave B: cross-link → /onboard next journey edge-user-journey first-run map.
// s1570 Wave C: cross-link → /onboard next wizard guided first-run wizard residual.
func AionAgentOnboardingStatus() string {
	return AionAgentOnboardingStartHere() + "\n\n" + strings.TrimSpace(`aion onboard status (residual-honest · offline static · s1368+s1372+s1377+s1382+s1387+s1402+s1407+s1413+s1417+s1432+s1437+s1442+s1447+s1542+s1558+s1570):
  MCP attach: expected for full path · fail-open offline (never invent tool green / install green)
  dual_write OFF · local-primary · not Memory GA · book-demo OFF · leave ON_SIGNAL unset
  portal HITL: Agent/MCP mint/copy/probe @ https://console.iome.sh/settings/agent · connectors @ https://console.iome.sh/integrations
  never invent install green / Connected / INSTALL_STORE APPLY · PASS ≠ invent human-gate green · PASS ≠ live APPLY
  list_org fail-open (available=false) ≠ empty-as-none · catalog ≠ Connected · list_plan_not_connected
  agent MCP cannot write installs · plugins dogfood ≠ invent Agent Plugins GA
  residual PASS ≠ live dogfood · test invoke = probe only ≠ Memory GA
  mesh = streaming org heartbeats · mesh ≠ memory · never invent stream green
  pull = mesh → local palace egress · pull ≠ freemium hosted palace · Ops Pack ≠ GPU fleet · pull_not_probed
  agentic: product plane 3 · MCP list/plan residual-honest · plan deep links = browser HITL only · template= ≠ install APPLY · portal_hitl_still
  three planes: /onboard next planes — mesh · memory-pull · agentic residual-honest consolidate · streams_not_probed · pull_not_probed · list_plan_not_connected · dual_auth_candidacy_open
  sales claims: /onboard next sales — may claim / must not claim residual-honest · three-planes grounded · never invent Connected / Memory GA
  demo readiness: /onboard next demo — Lighthouse beachhead · book-demo OFF · Landgrab NOT READY · human gates still open · never invent Connected
  operator matrix: /onboard next operator — demo · sales · planes · human-gates · dual-auth candidacy · policy locks residual-honest · never invent Connected (s1447)
  setup lifecycle (s1542+s1558 · stage 4): /onboard next setup — P1–P7 map · setup_not_probed · package wire ≠ Connected · repair apply ≠ invent Connected · dual_write never auto ON · E10 Open · offline static ≠ live dogfood
  edge-user-journey first-run (s1558 Wave B): /onboard next journey — 7 stages residual-honest · dual_write OFF · Edge Memory GA candidacy only · free eng s1558
  first-run wizard residual (s1570 Wave C): /onboard next wizard — guided first-run residual map + soft dogfood · free eng s1570
  portal HITL stage 5 (s1562): /onboard next portal-hitl — MCP list/plan → browser portal HITL · soft dogfood residual · free eng s1562
  E4 client attach stage 6 (s1566): /onboard next e4 — iomesh-memory-mcp local-primary · client attach · tools=6 · iomesh mcp --connect residual · soft dogfood residual · free eng s1566
  human-gates: still-human APPLY · Slack HMAC · Stripe Customers:Write · H1/H2 INSTALL_STORE · D1–D5 · open boxes stay open · Knowledge Beta→GA cannot invent H1/H2 offline · soft /onboard next human-gates dogfood (s1574 Wave C continuum)
  slash: /onboard portal · /onboard checklist · /onboard next [plugins|gtm|memory|mesh|memory-pull|agentic|portal-hitl|e4|planes|sales|demo|operator|setup|journey|wizard|status|export|human-gates] · /onboard next status · /onboard next export · /onboard next mesh · /onboard next memory-pull · /onboard next agentic · /onboard next portal-hitl · /onboard next e4 · /onboard next planes · /onboard next sales · /onboard next demo · /onboard next operator · /onboard next setup · /onboard next journey · /onboard next wizard · /onboard next human-gates · /onboard next human-gates dogfood · /integrations status`) + "\n" + strings.Join(OnboardNextStepLines(), "\n")
}

// AionAgentOnboardingNextLanes residual-honest post-onboard continuum for /onboard next (s1372 + s1377 + s1382 + s1387 + s1402 + s1407 + s1413 + s1417 + s1432 + s1437 + s1442 + s1447 + s1542).
// Static offline block — no MCP dial. Lists residual-honest operator lanes after
// core onboarding (plugins dogfood · GTM drafts · local memory · mesh streaming · Ops Pack pull · agentic integrations · portal HITL still · human gates · sales claims · setup lifecycle).
// s1377: drill-down via /onboard next plugins|gtm|memory (see lane helpers below).
// s1382: lane status board via /onboard next status (aliases pulse|board).
// s1387: status export receipt via /onboard next export (aliases receipt|stamp|evidence).
// s1402: mesh streaming lane via /onboard next mesh (aliases stream|streams|heartbeat|heartbeats|pull).
// s1407: memory-pull Ops Pack pull path via /onboard next memory-pull (aliases ops-pack|pull-path|memorypull|ops_pack).
// s1413: human-gates honesty board via /onboard next human-gates (aliases human|gates|apply-gates).
// s1417: agentic integrations product plane 3 via /onboard next agentic (aliases agentic-integrations|integrations|list-plan).
// s1562: portal HITL journey stage 5 via /onboard next portal-hitl (aliases hitl|portal_hitl|portal-dogfood|stage5|connectors-hitl) · soft dogfood residual.
// s1566: E4 client attach journey stage 6 via /onboard next e4 (aliases e4-dogfood|client-attach|edge-memory-e4|e4_attach) · soft dogfood residual.
// s1432: three product planes board via /onboard next planes (aliases three-planes|product-planes|product|pillars|three_planes).
// s1437: sales/buyer claims board via /onboard next sales (aliases claims|buyer|claim-matrix|sales-claims|buyer-claims).
// s1442: demo readiness board via /onboard next demo (aliases demo-ready|readiness|demo-readiness|lighthouse|landgrab).
// s1447: operator readiness matrix via /onboard next operator (aliases operator-matrix|ops-matrix|operator-readiness|ops-readiness|matrix).
// s1542: setup lifecycle P1–P7 closeout map via /onboard next setup (aliases setup-lifecycle|lifecycle|setup_lifecycle).
// s1558 Wave B: edge-user-journey first-run map via /onboard next journey (aliases edge-journey|user-journey|first-run|edge_user_journey).
// s1570 Wave C: first-run wizard residual via /onboard next wizard (aliases first-run-wizard|guided|wave-c|wave_c|wizard-residual) · soft dogfood residual.
// s1582: OSS packaging residual — split continuum into Edge OSS path vs Platform residual honesty (optional anti-claims).
// s1586: E10 Open reaffirm residual-check in Platform residual honesty group (after OSS packaging continuum).
// s1590: marketing demo path via /onboard next marketing-demo (aliases marketing|sales-demo|demo-script|gtm-demo) — plain-language local agent + memory for videos/sales.
// Never invents Agent Plugins GA, Memory GA, auto-send, install Connected, stream green, pull green, or human-gate green.
func AionAgentOnboardingNextLanes() string {
	return AionAgentOnboardingStartHere() + "\n\n" + strings.TrimSpace(`aion onboard next lanes (residual-honest · post-onboard continuum · s1372+s1377+s1382+s1387+s1402+s1407+s1413+s1417+s1432+s1437+s1442+s1447+s1542+s1558+s1562+s1566+s1570+s1574+s1578+s1582+s1586+s1590 · no MCP dial · OSS packaging residual):
OSS packaging (s1582 · MIT OSS harness · not control plane · Edge path first · residual PASS ≠ invent control plane in MIT repo):
  packaging: `+OSSPackagingHonestyOneLiner+`
  soft … dogfood = offline residual honesty check (user-facing: residual-check) · session soft ≠ live dogfood · ≠ invent platform green · slash token dogfood kept for compatibility
  docs: docs/architecture/oss-packaging-boundary.md

=== Edge OSS path ===
  (setup · journey · wizard · memory · e4 attach · portal HITL for connectors when used · marketing demo path)
  1. iomesh plugins dogfood · /plugins dogfood — offline sample validate (examples/agent-plugins) · ≠ invent Agent Plugins GA
     drill: /onboard next plugins (aliases plugin|dogfood) · slash: /plugins dogfood
  2. /gtm checklist + skill gtm-draft-only-agent — drafts only · no auto-send · human publish · GTM checklist ≠ invent GTM agent GA
     drill: /onboard next gtm (alias drafts)
  3. local-primary Memory edge (TUI + Memory MCP + memory kernel + local palace) — dual_write OFF · package load ≠ Memory GA · ≠ freemium palace · product host iomesh-memory-mcp · aion broker private · public product attach (s1478): go install github.com/iome-sh/iomesh-memory-mcp/cmd/iomesh-memory-mcp@main · go get github.com/iome-sh/memory@main · no GOPRIVATE · HTTP http://127.0.0.1:8080/mcp or stdio · docker compose still valid · flip complete residual ≠ invent Memory GA
     drill: /onboard next memory (aliases mcp|palace) · companion E4: /onboard next e4 (s1566) · deeper tool-call residual-check: /onboard next tool-call (s1578)
  4. I/O Mesh streaming org heartbeats on dept.* — mesh ≠ memory · not OTel/APM · not hosted Memory Palace · empty streams honest
     drill: /onboard next mesh (aliases stream|streams|heartbeat|heartbeats|pull) · residual soft: /mesh · iomesh mesh status|streams|consumer
  5. Memory Ops Pack pull path — iomesh memory pull = mesh → local palace egress · dual_write OFF · Ops Pack ≠ GPU fleet · pull_not_probed
     drill: /onboard next memory-pull (aliases ops-pack|pull-path|memorypull|ops_pack) · NOT bare pull (pull stays mesh lane)
  6. agentic integrations (product plane 3) — MCP list/plan residual-honest · plan_connector_setup → portal deep links · browser HITL only · catalog ≠ Connected · agent MCP cannot write installs
     drill: /onboard next agentic (aliases agentic-integrations|integrations|list-plan) · NOT bare mcp (memory lane) · NOT bare portal (portal handoff) · NOT portal-hitl|hitl (those are s1562 portal HITL lane)
  7. portal HITL connectors (s1562 · journey stage 5) — MCP list/plan → browser portal HITL → human OAuth/install · agent MCP cannot write installs · catalog ≠ Connected · portal_hitl_still
     drill: /onboard next portal-hitl (aliases hitl|portal_hitl|portal-dogfood|stage5|connectors-hitl) · soft residual-check: /onboard next portal-hitl dogfood · free eng s1562
  8. E4 client attach (s1566 · journey stage 6 local store / MCP attach) — iomesh-memory-mcp · local-primary · client attach · tools=6 · iomesh mcp --connect residual · dual_write OFF · Edge Memory GA candidacy only
     drill: /onboard next e4 (aliases e4-dogfood|client-attach|edge-memory-e4|e4_attach) · soft residual-check: /onboard next e4 dogfood · free eng s1566 · deeper residual-check: /onboard next tool-call (s1578)
  setup lifecycle map (s1542+s1558 · stage 4 of edge-user-journey · P1–P7 closeout residual): /onboard next setup (aliases setup-lifecycle|lifecycle|setup_lifecycle) — init → preflight → reload → portal HITL → pull → analyze → drift → repair plan/apply --yes · setup_not_probed · dual_write OFF · package wire ≠ Connected · repair apply ≠ invent Connected · dual_write never auto ON · E10 Open · offline static ≠ live dogfood · setup closeout residual ≠ invent Edge Memory GA · companion /onboard next journey · /onboard next wizard · memory · memory-pull · human-gates · e10 · operator · docs/architecture/setup-lifecycle.md
  edge-user-journey first-run map (s1558 Wave B · 7 stages): /onboard next journey (aliases edge-journey|user-journey|first-run|edge_user_journey) — Signup → Download TUI → TUI auth/keys → Setup wizard → Connectors → Local store → Analyze · dual_write OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA · portal HITL · no invent TUI portal SSO · host not auto · free eng s1558 · free-floor peer s1560+ mention only · companion stage 5 /onboard next portal-hitl · stage 6 /onboard next e4 · deeper tool-call /onboard next tool-call · E10 Open reaffirm /onboard next e10 (s1586) · Wave C /onboard next wizard · docs/architecture/edge-user-journey.md · setup-lifecycle · memory-edge-usage-demo
  first-run wizard residual (s1570 Wave C · guided residual map + soft residual-check): /onboard next wizard (aliases first-run-wizard|guided|wave-c|wave_c|wizard-residual) — deeper guided residual after Wave B journey map · soft residual-check /onboard next wizard dogfood · NOT invent full interactive auto wizard · dual_write OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · E10 Open · portal HITL when connect · agent MCP cannot write installs · catalog ≠ Connected · no invent TUI portal SSO · host not auto · free eng s1570 · free-floor peer s1572+ mention only · companion /onboard next journey · setup · portal-hitl · e4 · tool-call · e10 · human-gates · human-gates dogfood (s1574)
  agentic lane: /onboard next agentic — product plane 3 MCP list/plan residual-honest · list_plan_not_connected · companion portal HITL /onboard next portal-hitl (s1562) · never invent Connected / install green
  portal-hitl lane: /onboard next portal-hitl — journey stage 5 connectors · portal HITL when connect · soft residual-check dogfood residual (s1562) · free eng s1562 · free-floor peer s1564+ mention only
  e4 lane: /onboard next e4 — journey stage 6 local store / MCP attach · E4 client attach soft residual-check dogfood residual (s1566) · free eng s1566 · free-floor peer s1568+ mention only · deeper: /onboard next tool-call (s1578) · E10 Open reaffirm: /onboard next e10 (s1586)
  wizard lane: /onboard next wizard — Wave C first-run wizard residual · soft residual-check dogfood residual (s1570) · free eng s1570 · free-floor peer s1572+ mention only
  marketing demo path (s1590 · demo-oriented · videos/sales · local agent + local memory): /onboard next marketing-demo (aliases marketing|sales-demo|demo-script|gtm-demo) — plain-language operator script: install/build → LLM key or Ollama → /setup init local-memory + preflight → start/attach iomesh-memory-mcp → /memory ingest + recall · mesh optional only if configured · dual_write OFF · local memory · not Memory GA · never invent Connected · book-demo OFF · free eng s1590 · free-floor peer s1592+ mention only · NOT bare demo (demo readiness s1442) · NOT bare sales (sales claims) · NOT bare gtm (GTM drafts) · docs/architecture/marketing-demo-path.md

=== Platform residual honesty (optional · anti-claims · offline residual checks) ===
  (human-gates · soft residual-check dogfood subcommands · still-human APPLY · tool-call residual · E10 Open reaffirm)
  soft residual-check honesty: soft … dogfood = offline residual honesty check · residual-check · session soft ≠ live dogfood · ≠ invent platform green · residual PASS ≠ invent control plane in MIT repo · never dial MCP / never start host from soft residual-check
  9. deeper tool-call residual (s1578 · stage 6/7 depth after E4 attach) — operator map ingest→retrieve→list→as-of · soft offline residual-check only · Partial→client-attach-evidence · dual_write OFF · Edge Memory GA candidacy only
     drill: /onboard next tool-call (aliases tool-calls|deeper-e4|e4-tools|ingest-retrieve|tool_call) · soft residual-check: /onboard next tool-call dogfood · free eng s1578
  10. human-gates still-required vs offline residual (s1413+s1546+s1550+s1574 Wave C continuum) — still-human APPLY · Slack HMAC · Stripe Customers:Write · H1/H2 INSTALL_STORE · D1–D5 · book-demo OFF · ON_SIGNAL unset
     drill: /onboard next human-gates (aliases human|gates|apply-gates|still-human|apply-residual) · soft residual-check: /onboard next human-gates dogfood · PASS ≠ invent human-gate green · PASS ≠ live APPLY · open boxes stay open · free eng s1574
  11. E10 Open reaffirm residual-check (s1586 · after OSS packaging continuum) — pin E10 Open · residual PASS ≠ invent E10 closed · residual PASS ≠ invent Edge Memory GA declared · Edge Memory GA candidacy only · founder sign-off only if declaring Edge Memory GA · candidacy allowed without E10 · PASS ≠ live APPLY · dual_write OFF · book-demo OFF
     drill: /onboard next e10 (aliases e10-open|edge-memory-e10|ga-signoff|e10_open) · soft residual-check: /onboard next e10 dogfood · free eng s1586 · free-floor peer s1588+ mention only
  still-human APPLY soft residual-check: /onboard next human-gates dogfood — Wave C continuum residual reaffirm open inventory (s1574) · free eng s1574 · free-floor peer s1576+ mention only
  tool-call lane: /onboard next tool-call — deeper tool-call soft residual-check dogfood residual after E4 attach (s1578) · free eng s1578 · free-floor peer s1580+ mention only
  e10 lane: /onboard next e10 — E10 Open reaffirm soft residual-check dogfood residual after OSS packaging (s1586) · free eng s1586 · free-floor peer s1588+ mention only · residual PASS ≠ invent E10 closed
  human-gates board (s1413+s1546+s1550+s1574 Wave C continuum still-human APPLY residual): /onboard next human-gates (aliases human|gates|apply-gates|still-human|apply-residual) · soft residual-check /onboard next human-gates dogfood — still-human APPLY open · PASS ≠ invent human-gate green · PASS ≠ live APPLY · open boxes stay open · free eng s1574 · free-floor peer s1576+ mention only (local memory / dual_write OFF / agent MCP list/plan do not close human APPLY gates) · companion E10 Open reaffirm /onboard next e10 (s1586)
  three product planes: /onboard next planes (aliases three-planes|product-planes|product|pillars|three_planes) — mesh · memory-pull · agentic residual-honest consolidate · streams_not_probed · pull_not_probed · list_plan_not_connected · dual_auth_candidacy_open · never invent Connected (s1432)
  sales/buyer claims: /onboard next sales (aliases claims|buyer|claim-matrix|sales-claims|buyer-claims) — may claim / must not claim residual-honest · three-planes grounded · never invent Connected / Memory GA / dual-auth live (s1437) · NOT product/planes (those stay three-planes) · NOT gtm (drafts) · NOT pulse/board (status)
  demo readiness: /onboard next demo (aliases demo-ready|readiness|demo-readiness|lighthouse|landgrab) — Lighthouse beachhead packaging · book-demo OFF · Landgrab NOT READY · three planes · sales claims · human gates still open · residual PASS ≠ logos met (s1442) · NOT sales/claims (sales claims) · NOT product/planes (three-planes) · NOT pulse/board (status) · NOT gtm/drafts · NOT marketing-demo (s1590 demo script)
  marketing demo path: /onboard next marketing-demo (aliases marketing|sales-demo|demo-script|gtm-demo) — plain-language local agent + local memory script for videos/sales (s1590) · dual_write OFF · local memory · not Memory GA · mesh optional · never invent Connected · book-demo OFF · free eng s1590 · free-floor peer s1592+ mention only · NOT bare demo (demo readiness) · NOT bare sales · NOT bare gtm
  operator readiness matrix: /onboard next operator (aliases operator-matrix|ops-matrix|operator-readiness|ops-readiness|matrix) — consolidate demo · sales · planes · human-gates · dual-auth candidacy · policy locks residual-honest · residual_only · path_ready · still_human · policy_off · not_ready · portal_hitl_still (s1447) · NOT demo/readiness/lighthouse/landgrab (demo board) · NOT sales/claims · NOT product/planes · NOT pulse/board · NOT export/receipt
  status board: /onboard next status (aliases pulse|board) — residual-honest lane states only (never invent connected/ga/apply as success · pulse stays board)
  export receipt: /onboard next export (aliases receipt|stamp|evidence) — offline markdown evidence of board (board/export evidence ≠ invent Connected)

Locks: dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · PASS ≠ live APPLY · PASS ≠ invent human-gate green · open boxes stay open · leave ON_SIGNAL unset · Knowledge Beta→GA cannot invent H1/H2 offline · never invent install green / Connected / INSTALL_STORE APPLY · list_org fail-open ≠ empty-as-none · plugins dogfood ≠ invent Agent Plugins GA · drafts only · no auto-send · rates ~$88/$119 optional · package load ≠ Memory GA · board/export evidence ≠ invent Connected · mesh = streaming org heartbeats · mesh ≠ memory · never invent stream green / Connected · not OTel/APM · streams_not_probed honest · pull ≠ freemium hosted palace · Ops Pack ≠ GPU fleet · pull_not_probed honest · never invent pull green · list_plan_not_connected · dual_auth_candidacy_open · plan deep links = browser HITL only · template= ≠ install APPLY · sales claims residual-honest only · demo readiness residual-honest only · Landgrab NOT READY · residual PASS ≠ logos met · marketing demo path dual_write OFF · local memory · mesh optional · never invent Connected · free eng s1590 · free-floor peer s1592+ mention only · operator matrix residual-honest only · setup_not_probed · package wire ≠ Connected · repair apply ≠ invent Connected · dual_write never auto ON · E10 Open · residual PASS ≠ invent E10 closed · setup closeout residual ≠ invent Edge Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA · residual PASS ≠ invent Edge Memory GA declared · free eng s1558 · free-floor peer s1560+ mention only · no invent TUI portal SSO · host not auto · free eng s1562 · free-floor peer s1564+ mention only · free eng s1566 · free-floor peer s1568+ mention only · free eng s1570 · free-floor peer s1572+ mention only · free eng s1574 · free-floor peer s1576+ mention only · free eng s1578 · free-floor peer s1580+ mention only · free eng s1582 · free-floor peer s1584+ mention only · free eng s1586 · free-floor peer s1588+ mention only · OSS harness · residual-check · not control plane · residual PASS ≠ invent control plane in MIT repo · soft offline ≠ invent Connected · session soft ≠ live dogfood · portal HITL when connect · tip ≠ invent forever-green product dogfood · residual PASS ≠ invent full interactive auto wizard · still-human APPLY open · Wave C continuum · deeper tool-call residual candidacy only · E10 Open reaffirm residual-check`) + "\n" + strings.Join(OnboardNextStepLines(), "\n")
}

// AionAgentOnboardingNextPluginsLane residual-honest plugins dogfood drill for /onboard next plugins (s1377+s1392).
// Static offline — iomesh plugins dogfood + /plugins dogfood path. Never invents Agent Plugins GA,
// install Connected, dual_write ON, or live dogfood green.
func AionAgentOnboardingNextPluginsLane() string {
	return strings.TrimSpace(`aion onboard next plugins lane (residual-honest · s1377+s1392+s1521 · no MCP dial):
  Path: iomesh plugins smoke · /plugins smoke — offline sample validate only (legacy: iomesh plugins dogfood)
  Samples: examples/agent-plugins/{hello-iome,iomesh-memory-mcp} (product primary)
  Steps:
    1. iomesh plugins list · /plugins list — closed-manifest discovery map (≠ invent install green / Connected)
    2. iomesh plugins validate <path> · /plugins validate — offline package shape residual
    3. iomesh plugins smoke · /plugins smoke — both in-repo samples offline (legacy: plugins dogfood · residual PASS ≠ live dogfood)
  Honesty:
    · plugins smoke ≠ invent Agent Plugins GA · plugins dogfood ≠ invent Agent Plugins GA (legacy name)
    · soft offline dogfood ≠ invent Agent Plugins GA · residual PASS ≠ live dogfood
    · never invent install green / Connected / INSTALL_STORE APPLY
    · catalog ≠ Connected · agent MCP cannot write installs · portal HITL still for OAuth/install
    · package load ≠ Memory GA · rates ~$88/$119 optional
  Slash: /plugins smoke (aliases dogfood|soft|samples|offline) · /plugins list · /plugins validate · /plugins status
  Back: /onboard next · companion samples offline only

Locks: dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · never invent install green / Connected / INSTALL_STORE APPLY · plugins dogfood ≠ invent Agent Plugins GA · plugins smoke ≠ invent Agent Plugins GA · catalog ≠ Connected · portal HITL · agent MCP cannot write installs · package load ≠ Memory GA · rates ~$88/$119 optional`)
}

// AionAgentOnboardingNextGtmLane residual-honest GTM draft-only drill for /onboard next gtm (s1377).
// Static offline — /gtm checklist + gtm-draft-only-agent skill. Never invents auto-send,
// GTM agent GA, suite ops GA, or install Connected.
func AionAgentOnboardingNextGtmLane() string {
	return strings.TrimSpace(`aion onboard next gtm lane (residual-honest · s1377 · no MCP dial):
  Path: /gtm checklist + skill gtm-draft-only-agent — drafts only · no auto-send · human publish
  Steps:
    1. /gtm checklist (or /gtm help) — residual-honest draft-only checklist
    2. read_skill gtm-draft-only-agent — roles draft/plan only (Orchestrator · Content · Campaign · Lead)
    3. Draft content/outreach only — never auto-send email/SNS · human publish · human CRM commercial
  Honesty:
    · drafts only · no auto-send · human publish
    · GTM checklist ≠ invent GTM agent GA · residual PASS ≠ live dogfood publish
    · Salesforce = GA CRM; HubSpot + GTM suite Beta multi-tenant; guerrilla global-only
    · portal HITL for installs · agent MCP cannot write installs · never invent Connected / INSTALL_STORE APPLY
  Back: /onboard next · /gtm [help|checklist]

Locks: dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · drafts only · no auto-send · human publish · GTM checklist ≠ invent GTM agent GA · never invent install green / Connected / INSTALL_STORE APPLY · catalog ≠ Connected · portal HITL · agent MCP cannot write installs · rates ~$88/$119 optional`)
}

// AionAgentOnboardingNextMemoryLane residual-honest memory local drill for /onboard next memory
// (s1377+s1453+s1458+s1463+s1469+s1478+s1508+s1517+s1695).
// Static offline — local-primary Memory edge (TUI OSS + iomesh-memory-mcp + github.com/iome-sh/memory kernel + local palace).
// Naming honesty (s1453 Option A · s1517 cleanup): product MCP host = iomesh-memory-mcp only (public edge);
// aion = private cloud broker/CP (not OSS edge pack; s1517 dropped in-tree residual memory sample). s1458–s1469: M2 lean /
// M3 dogfood / M4 readiness history. s1478: both product edge repos are PUBLIC — go install / go get without
// GOPRIVATE · attach HTTP :8080/mcp or stdio · docker compose still valid · flip complete residual ≠ invent
// Memory GA · dual_write OFF · aion broker still private · not freemium hosted palace.
// s1508: E4 full MCP client attach dogfood tip residual (connected=1 · tools=6 stamp) · Edge Memory GA candidacy
// only · E10 Open · tip ≠ invent Edge Memory GA declared · tip ≠ invent forever-green product dogfood.
// s1695: first-run = OSS local-primary only · OSS first-run complete without mesh · Memory Ops Pack optional overlay
// (not first-run required) · mesh optional · Ops Pack ≠ GPU · not freemium hosted palace · companion memory-pull
// only when mesh configured.
// Never invents Memory GA, freemium palace, dual_write ON, install Connected, or live dogfood green.
func AionAgentOnboardingNextMemoryLane() string {
	return strings.TrimSpace(`aion onboard next memory lane (residual-honest · s1377+s1453+s1458+s1463+s1469+s1478+s1508+s1517+s1695 · no MCP dial):
  Path: local-primary Memory edge — TUI OSS + iomesh-memory-mcp + github.com/iome-sh/memory kernel + local palace — dual_write OFF
  First-run honesty (s1695): OSS first-run complete without mesh · Ops Pack not first-run required · Memory Ops Pack optional · mesh optional · dual_write OFF · not Memory GA · not freemium hosted palace
  Edge OSS (Option A · s1453+s1458+s1463+s1469+s1478+s1508+s1517 · public product attach continuum):
    · product MCP host = iomesh-memory-mcp only (public · go install / compose)
    · s1478 PUBLIC product path (both edge repos public · no GOPRIVATE / PAT required):
        go install github.com/iome-sh/iomesh-memory-mcp/cmd/iomesh-memory-mcp@main
        (or clone github.com/iome-sh/iomesh-memory-mcp and go build)
        kernel: go get github.com/iome-sh/memory@main — no GOPRIVATE
        attach: streamable HTTP http://127.0.0.1:8080/mcp or stdio command iomesh-memory-mcp
        docker compose still valid: docker compose up --build → image iomesh-memory-mcp:local → http://127.0.0.1:8080/mcp · healthz
    · history (s1458–s1469 residual): M2 lean host · M3 edge dogfood tip · M4 public flip readiness (kernel first · then iomesh-memory-mcp) — flip is now complete for edge packs; readiness tip ≠ invent Memory GA
    · s1508 E4 MCP client attach dogfood tip: lean host HTTP → iomesh mcp --connect · connected=1 · tools=6 stamp residual · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · E10 Open · tip ≠ invent forever-green product dogfood · dual_write OFF · not bare Memory GA · not hosted Memory GA · docs/EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md
    · offline gate mention: peer mcp make edge-dogfood-gate (mention only) · TUI docs docs/architecture/memory-mcp.md
    · dual_write OFF · not Memory GA · aion broker private (cloud CP stays private · not OSS edge pack) · aion still private
    · flip complete residual: public OSS edge ≠ invent Memory GA · ≠ freemium palace · dual_write OFF · package load ≠ Memory GA
    · residual: tool parity may be lean vs platform residual · PASS ≠ invent full platform sidecar parity
    · offline dogfood tip ≠ invent live dogfood as green · public product attach ≠ invent platform GA
    · kernel = github.com/iome-sh/memory (public) · product host github.com/iome-sh/iomesh-memory-mcp (public)
    · product attach path is iomesh-memory-mcp only (s1517) · package load ≠ Memory GA
  Steps:
    1. Public install product host: go install github.com/iome-sh/iomesh-memory-mcp/cmd/iomesh-memory-mcp@main — no GOPRIVATE · package load ≠ Memory GA · ≠ freemium palace
    2. Kernel (public): go get github.com/iome-sh/memory@main — no GOPRIVATE
    3. Attach local-primary: HTTP http://127.0.0.1:8080/mcp or stdio iomesh-memory-mcp · dual_write OFF · not Memory GA
    4. docker compose still valid: docker compose up --build in github.com/iome-sh/iomesh-memory-mcp → http://127.0.0.1:8080/mcp · curl http://127.0.0.1:8080/healthz · stdio alternate — offline dogfood tip ≠ invent live dogfood as green
    5. dual_write OFF · local-primary only · not Memory GA · Palace sunset · aion broker private · aion still private · OSS first-run complete without mesh
    6. Optional: read_skill memory-advanced-agent (opt-in advanced · still dual_write OFF · not Memory GA)
    7. Optional mesh pull only (later path · not first-run required): /onboard next memory-pull · Memory Ops Pack optional (~$119 pull/retain/support · local-primary overlay · Ops Pack ≠ GPU fleet) · mesh optional · dual_write OFF · only when mesh configured
    8. Operator pulse: /memory status · /onboard status · /onboard next operator (fail-open offline · never invent tool green)
    9. Optional E4 client attach dogfood (s1508): iomesh mcp --connect after lean host HTTP — stamp residual · Edge Memory GA candidacy only · E10 Open · tip ≠ invent forever-green product dogfood
  Honesty:
    · package load ≠ Memory GA · ≠ freemium palace · not freemium hosted palace · dual_write OFF · Palace sunset
    · OSS first-run complete without mesh · Ops Pack not first-run required · Memory Ops Pack optional · mesh optional
    · residual PASS ≠ live dogfood · offline dogfood tip ≠ invent live dogfood as green · test invoke = probe only ≠ Memory GA · PASS ≠ live APPLY
    · PASS ≠ invent full platform sidecar parity · tool parity may be lean vs platform residual
    · public product attach (s1478) · no GOPRIVATE · go install · flip complete residual ≠ invent Memory GA · public OSS ≠ invent platform GA
    · E4 client attach (s1508) · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · E10 Open · tip ≠ invent forever-green product dogfood · not bare Memory GA · not hosted Memory GA
    · kernel first history · then iomesh-memory-mcp · aion broker private · aion still private
    · never invent install green / Connected / INSTALL_STORE APPLY
    · catalog ≠ Connected · portal HITL · agent MCP cannot write installs
    · mesh ≠ memory · mesh optional · mesh optional for pull only · memory lane is local-edge palace, not streaming org heartbeats
    · iomesh-memory-mcp product host only · TUI OSS · aion broker private · s1517 product-only memory sample
    · rates ~$88 mesh / ~$119 Memory Ops Pack optional · package load ≠ Memory GA · Ops Pack ≠ GPU fleet
  Companion: /onboard next e4 (s1566 · journey stage 6 E4 client-attach soft dogfood residual) · /onboard next e4 dogfood · /onboard next tool-call (s1578 · deeper tool-call residual after attach) · /onboard next tool-call dogfood · /onboard next memory-pull (optional Ops Pack pull path only when mesh configured · s1695 · not first-run required) · /onboard next operator · docs/architecture/memory-mcp.md Edge OSS Option A · public product attach (s1478) · E4 client attach (s1508) · docs/EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md
  Back: /onboard next · /memory status · portal Agent/MCP https://console.iome.sh/settings/agent

Locks: dual_write OFF · book-demo OFF · not Memory GA · Palace sunset · residual PASS ≠ live dogfood · offline dogfood tip ≠ invent live dogfood as green · PASS ≠ live APPLY · PASS ≠ invent full platform sidecar parity · flip complete residual ≠ invent Memory GA · public OSS ≠ invent platform GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · E10 Open · tip ≠ invent forever-green product dogfood · no GOPRIVATE · go install · package load ≠ Memory GA · ≠ freemium palace · not freemium hosted palace · never invent install green / Connected / INSTALL_STORE APPLY · catalog ≠ Connected · portal HITL · agent MCP cannot write installs · rates ~$88/$119 optional · Memory Ops Pack optional · Ops Pack not first-run required · OSS first-run complete without mesh · mesh ≠ memory · mesh optional · mesh optional for pull · Ops Pack ≠ GPU fleet · TUI OSS · iomesh-memory-mcp · aion broker private · aion still private · s1517 product-only memory sample (iomesh-memory-mcp) · companion E4 soft residual s1566 · deeper tool-call soft residual s1578 · free eng s1695`)
}

// AionAgentOnboardingNextMeshLane residual-honest mesh streaming lane for /onboard next mesh (s1402).
// Static offline — I/O Mesh = streaming org heartbeats / governed dept.* (product plane 1).
// NOT hosted Memory Palace · not OTel/APM · not medical · mesh ≠ memory lane.
// Operator residual soft only (status/streams/consumers) · never invent stream green / Connected.
// Pull honesty: iomesh memory pull = mesh egress into local palace · dual_write OFF · not freemium hosted palace.
// Rates: mesh base ~$88 · Memory Ops Pack ~$119 pull/retain/support.
func AionAgentOnboardingNextMeshLane() string {
	return strings.TrimSpace(`aion onboard next mesh lane (residual-honest · s1402 · no MCP dial · product plane 1):
  Path: I/O Mesh = streaming org heartbeats on governed dept.* — NOT hosted Memory Palace · not OTel/APM
  Separation: mesh ≠ memory · mesh lane ≠ plugins/gtm lanes · pull ≠ freemium hosted palace · Palace sunset
  Steps:
    1. Residual soft operator: /mesh · iomesh mesh status (fail-open offline · never invent stream green / Connected)
    2. Streams residual: iomesh mesh streams — empty streams honest · streams_not_probed until operator probes
    3. Consumers residual: iomesh mesh consumer (durable pull consumers · residual soft · requires --yes when mutating)
    4. Pull honesty: iomesh memory pull = mesh → local palace egress · dual_write OFF · not freemium hosted palace · not Memory GA
  Honesty:
    · mesh = streaming org heartbeats · not OTel/APM · not medical · not hosted Memory Palace
    · mesh ≠ memory · memory lane is local-edge palace; mesh lane is streaming heartbeats
    · never invent stream green / Connected · empty streams honest · streams_not_probed residual
    · residual PASS ≠ live dogfood · PASS ≠ live APPLY · dual_write OFF · book-demo OFF
    · pull ≠ freemium hosted palace · package load ≠ Memory GA · rates ~$88 mesh / ~$119 Memory Ops Pack optional
    · catalog ≠ Connected · portal HITL · agent MCP cannot write installs
  Slash: /onboard next mesh (aliases stream|streams|heartbeat|heartbeats|pull) · NOT pulse (pulse stays /onboard next status board)
  Companion: /mesh · iomesh mesh status|streams|consumer · iomesh memory pull (egress only · dual_write OFF)
  Back: /onboard next · /onboard next status · /onboard next memory (separate local-edge lane)

Locks: dual_write OFF · book-demo OFF · not Memory GA · Palace sunset · residual PASS ≠ live dogfood · PASS ≠ live APPLY · mesh = streaming org heartbeats · mesh ≠ memory · not OTel/APM · not medical · never invent stream green / Connected / INSTALL_STORE APPLY · empty streams honest · streams_not_probed · pull ≠ freemium hosted palace · catalog ≠ Connected · portal HITL · agent MCP cannot write installs · rates ~$88/$119 optional · board/export evidence ≠ invent Connected`)
}

// AionAgentOnboardingNextMemoryPullLane residual-honest Memory Ops Pack pull path for
// /onboard next memory-pull (s1407+s1695). Static offline — pull = mesh → local palace egress
// (iomesh memory pull · CreateConsumer → fetch → map envelope → local MCP memory_ingest_turn → ack).
// NOT freemium hosted palace · dual_write OFF · not Memory GA · Palace sunset · Ops Pack ≠ GPU fleet.
// Ops Pack ~$119 = pull / retain / support · local-primary · TUI OSS + mesh pull entitlement —
// not hosted GPU palace · not first-run required (first-run is local OSS only).
// Mesh base ~$88 is separate · mesh ≠ memory · bare pull alias stays mesh lane (s1402).
// Honest residual: path_ready · residual_only · pull_not_probed (never invent pull green).
// s1695: Ops Pack not first-run required · first-run is local OSS only · package load ≠ Ops Pack entitlement.
func AionAgentOnboardingNextMemoryPullLane() string {
	return strings.TrimSpace(`aion onboard next memory-pull lane (residual-honest · s1407+s1695 · no MCP dial · Ops Pack pull path):
  Path: iomesh memory pull = mesh → local palace egress — CreateConsumer → fetch → map envelope → local MCP memory_ingest_turn → ack
  Product: Memory Ops Pack ~$119 = pull / retain / support · local-primary · TUI OSS + mesh pull entitlement — Ops Pack ≠ GPU fleet · not freemium hosted palace · Palace sunset
  First-run honesty (s1695): Ops Pack not first-run required · first-run is local OSS only · Memory Ops Pack optional commercial overlay · dual_write OFF
  Separation: mesh ≠ memory · mesh base ~$88 separate · pull ≠ freemium hosted palace · dual_write OFF · package load ≠ Ops Pack entitlement
  Steps:
    1. Residual soft: iomesh memory pull --dry-run / config [memory] pull_stream · pull_consumer · pull_filter (fail-open offline · never invent pull green)
    2. Durable consumer residual: CreateConsumer on mesh stream (requires --yes when mutating · residual soft only)
    3. Fetch → map envelope → local MCP memory_ingest_turn → ack (dual_write OFF · local-primary only)
    4. Operator pulse: /onboard next status · /onboard next export — board shows pull_not_probed until operator probes
  Honesty:
    · pull = mesh → local palace egress · dual_write OFF · not freemium hosted palace · not Memory GA · Palace sunset
    · Ops Pack ≠ GPU fleet · package load ≠ Ops Pack entitlement · package load ≠ Memory GA
    · not first-run required · first-run is local OSS only · Memory Ops Pack optional · local-primary · TUI OSS
    · residual PASS ≠ live dogfood · PASS ≠ live APPLY · never invent pull green / Connected
    · pull_not_probed residual honest · board/export evidence ≠ invent Connected
    · mesh ≠ memory · rates ~$88 mesh / ~$119 Memory Ops Pack optional · book-demo OFF
    · catalog ≠ Connected · portal HITL · agent MCP cannot write installs
  Slash: /onboard next memory-pull (aliases ops-pack|pull-path|memorypull|ops_pack) · bare pull stays mesh lane (s1402)
  Companion: iomesh memory pull · /onboard next mesh (streaming heartbeats · product plane 1) · /onboard next memory (local-edge attach · OSS first-run · not first-run required for Ops Pack)
  Back: /onboard next · /onboard next status · /onboard next export

Locks: dual_write OFF · book-demo OFF · not Memory GA · Palace sunset · residual PASS ≠ live dogfood · PASS ≠ live APPLY · pull ≠ freemium hosted palace · not freemium hosted palace · Ops Pack ≠ GPU fleet · pull_not_probed · never invent pull green / Connected / INSTALL_STORE APPLY · package load ≠ Ops Pack entitlement · package load ≠ Memory GA · mesh ≠ memory · not first-run required · first-run is local OSS only · Memory Ops Pack optional · local-primary · TUI OSS · rates ~$88/$119 optional · board/export evidence ≠ invent Connected · catalog ≠ Connected · portal HITL · agent MCP cannot write installs · free eng s1695`)
}

// AionAgentOnboardingNextSetupLane residual-honest setup lifecycle P1–P7 closeout map for
// /onboard next setup (s1542 + s1558 Wave B). Static offline — consolidates setup lifecycle map story:
// init → memory host · secrets → preflight → reload → portal HITL → optional pull/analyze →
// drift report-only → guided repair plan/apply --yes · /memory digest still valid.
// s1558 Wave B: setup is **stage 4** of the 7-stage edge-user-journey; full first-run map is
// /onboard next journey (does not invent auto host / TUI portal SSO / Connected / dual_write ON).
// Honest residual: path_ready · residual_only · setup_not_probed (never invent Connected).
// dual_write OFF · not Memory GA · catalog ≠ Connected · portal HITL · repair apply ≠ invent Connected ·
// dual_write never auto ON · still-human APPLY open · E10 Open residual · offline static ≠ live dogfood ·
// setup closeout residual ≠ invent Edge Memory GA.
// Aliases: setup-lifecycle|lifecycle|setup_lifecycle (wizard alias is s1570 Wave C first-run wizard residual — not setup).
// Companion: /onboard next journey · /onboard next wizard · memory · memory-pull · human-gates · operator · docs setup-lifecycle + edge-user-journey + memory-edge-usage-demo.
func AionAgentOnboardingNextSetupLane() string {
	return strings.TrimSpace(`aion onboard next setup lane (residual-honest · s1542+s1558 Wave B · setup lifecycle P1–P7 closeout residual · stage 4 of edge-user-journey · no MCP dial):
  Path: setup lifecycle map story — stage 4 of 7-stage edge-user-journey — managed config · preflight · reload · portal HITL · opt-in pull/analyze · drift report · guided repair — dual_write OFF
  Product: setup lifecycle P1–P7 residual closeout (init · preflight · reload · continuous pull · analyze ticks · drift · guided repair) — offline static lane ≠ live dogfood · setup closeout residual ≠ invent Edge Memory GA · full first-run map: /onboard next journey (s1558 Wave B) · guided residual: /onboard next wizard (s1570 Wave C)
  Steps:
    1. /setup init · iomesh setup init — dual_write OFF · managed fragment (local-memory default · secrets as env names only)
    2. start memory host · set secret env names (api_key_env · oauth_token_env) — never commit secret values
    3. /setup preflight — PASS ≠ invent Connected · states residual-honest only (not_started · config_present · awaiting_memory_host · local_memory_probe_ok)
    4. /setup reload — package wire ≠ Connected · hot MCP re-attach residual · skills re-scanned on /setup reload (s1670 · LoadWithBuiltin + ReplaceSkills) · skills re-scan ≠ invent Connected · package wire ≠ Connected
    5. portal HITL /setup portal — OAuth/install still browser · agent MCP cannot write installs · catalog ≠ Connected
    6. optional /setup pull start when mesh+consumer — pull ≠ invent Connected · pull_continuous opt-in · CLI iomesh memory pull still valid
    7. optional /setup analyze start — tick ≠ invent green · analyze_continuous opt-in · dual_write OFF
    8. /setup drift — report-only · drift PASS ≠ invent install green · package wire ≠ Connected
    9. /setup repair plan · /setup repair apply --yes — safe steps only · dual_write never auto ON · repair apply ≠ invent Connected · refuse without --yes
    10. /memory digest still valid as manual deep ops pulse · analyze tick ≠ invent Connected
  Honesty:
    · dual_write OFF · not Memory GA · catalog ≠ Connected · portal HITL
    · package wire ≠ Connected · PASS ≠ invent Connected · preflight PASS ≠ invent Connected
    · pull ≠ invent Connected · analyze tick ≠ invent green · drift PASS ≠ invent install green
    · repair apply ≠ invent Connected · dual_write never auto ON · still-human APPLY open · E10 Open residual
    · residual PASS ≠ live dogfood · offline static lane ≠ live dogfood · setup closeout residual ≠ invent Edge Memory GA
    · board/export evidence ≠ invent Connected · setup_not_probed residual honest
    · agent MCP cannot write installs · never invent install green / Connected / INSTALL_STORE APPLY
    · stage 4 of edge-user-journey · free eng s1558 · Edge Memory GA candidacy only
  Slash: /onboard next setup (aliases setup-lifecycle|lifecycle|setup_lifecycle) · companion slash /setup [init|preflight|portal|reload|pull|analyze|drift|repair]
  Companion: /onboard next journey · /onboard next wizard · /onboard next memory · /onboard next memory-pull · /onboard next human-gates · /onboard next operator · skill setup-lifecycle-agent · docs/architecture/setup-lifecycle.md · docs/architecture/edge-user-journey.md · docs/architecture/memory-edge-usage-demo.md
  Back: /onboard next · /onboard next journey · /onboard next wizard · /onboard next status · /onboard next export

Locks: dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · PASS ≠ live APPLY · package wire ≠ Connected · catalog ≠ Connected · portal HITL · agent MCP cannot write installs · pull ≠ invent Connected · analyze tick ≠ invent green · drift PASS ≠ invent install green · repair apply ≠ invent Connected · dual_write never auto ON · still-human APPLY open · E10 Open · setup_not_probed · offline static lane ≠ live dogfood · setup closeout residual ≠ invent Edge Memory GA · Edge Memory GA candidacy only · free eng s1558 · never invent install green / Connected / INSTALL_STORE APPLY · board/export evidence ≠ invent Connected`)
}

// AionAgentOnboardingNextJourneyLane residual-honest 7-stage edge-user-journey first-run map for
// /onboard next journey (s1558 Wave B). Static offline — first-run operator map after Wave A docs
// SSOT (s1554). Stages: 1 Signup · 2 Download TUI · 3 TUI auth/keys · 4 Setup wizard · 5 Connectors
// MCP list/plan + portal HITL · 6 Local store iomesh-memory-mcp · 7 Analyze.
// Never invents: auto memory host · TUI portal SSO · Connected · dual_write ON · Memory GA ·
// agent install APPLY · book-demo ON · Edge Memory GA declared.
// Aliases: edge-journey|user-journey|first-run|edge_user_journey.
// Companion: /onboard next setup (stage 4 detail) · /onboard next wizard (s1570 Wave C guided residual) ·
// agentic · memory · human-gates · operator · docs edge-user-journey + setup-lifecycle + memory-edge-usage-demo.
// free-floor peer s1560+ mention only (do not rewrite free-floor).
func AionAgentOnboardingNextJourneyLane() string {
	return strings.TrimSpace(`aion onboard next journey lane (residual-honest · s1558 Wave B · edge-user-journey first-run map · no MCP dial):
  Path: 7-stage edge-first first-run map — Signup → Download TUI → TUI auth/keys → Setup wizard → Connectors → Local store → Analyze
  Product: Wave B first-run polish after Wave A docs SSOT (s1554) — residual-honest operator map · free eng s1558 · free-floor peer s1560+ mention only · companion Wave C guided residual: /onboard next wizard (s1570)
  Stages (order residual-honest):
    1. Signup — portal console.iome.sh create/join org · optional pure local (skip for pure local memory) · signup ≠ Memory GA · ≠ Connected
    2. Download TUI — go install / releases / make build · MIT harness · binary ≠ platform control plane
    3. TUI auth/keys — LLM API keys (default cascade) · optional Ollama local · optional mesh/MCP bearer · not platform-bundled weights · no invent TUI portal SSO
       primary: configure env/config · optional Ollama · not platform SSO
    4. Setup wizard (stage 4 detail) — /setup · /onboard next setup · CLI iomesh setup · P1–P7 residual · dual_write OFF
       primary: /setup [init|preflight|portal|reload|pull|analyze|drift|repair] · /onboard next setup · iomesh setup
    5. Connectors MCP list/plan + portal HITL (journey stage 5) — /integrations list|plan|status · /onboard next portal-hitl · /onboard next agentic · browser OAuth/install only
       primary: /integrations list|plan|status · /onboard next portal-hitl · /onboard next agentic · portal HITL @ https://console.iome.sh/integrations
       soft residual: /onboard next portal-hitl dogfood (s1562 · soft offline ≠ invent Connected · session soft ≠ live dogfood)
    6. Local store iomesh-memory-mcp (journey stage 6) — install/run host + kernel · attach HTTP/stdio · dual_write OFF · host not auto on signup
       primary: /onboard next memory · /onboard next e4 · /onboard next tool-call · /memory status · go install github.com/iome-sh/iomesh-memory-mcp/cmd/iomesh-memory-mcp@main
       soft residual: /onboard next e4 dogfood (s1566 · soft offline ≠ invent Connected · residual PASS ≠ invent Edge Memory GA declared · session soft ≠ live dogfood)
       deeper soft residual: /onboard next tool-call dogfood (s1578 · ingest→retrieve→list→as-of · soft offline ≠ invent Connected · residual PASS ≠ invent Edge Memory GA declared)
    7. Analyze — /memory digest · /setup analyze · optional mesh Ops Pack pull (~$119) · analyze ≠ invent Connected
       primary: /memory digest · /setup analyze · /onboard next memory-pull (optional) · companion deeper tool residual /onboard next tool-call
  Honesty one-liner locks:
    · dual_write OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA
    · portal HITL · portal HITL when connect · agent MCP cannot write installs · catalog ≠ Connected · book-demo OFF
    · no invent TUI portal SSO · memory host not auto on signup · free eng s1558
  Residual gaps (do not invent closed):
    · no SSO invent · host not auto · portal HITL still human · dual_write OFF · Edge Memory GA candidacy only
    · free-floor peer s1560+ mention only (do not rewrite free-floor)
  Docs: docs/architecture/edge-user-journey.md · docs/architecture/setup-lifecycle.md · docs/architecture/memory-edge-usage-demo.md · docs/EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md
  Companion: /onboard next wizard (s1570 Wave C guided residual) · /onboard next wizard dogfood · /onboard next setup (stage 4 P1–P7) · /onboard next portal-hitl (stage 5) · /onboard next portal-hitl dogfood · /onboard next e4 (stage 6) · /onboard next e4 dogfood · /onboard next tool-call (s1578 deeper tool residual) · /onboard next tool-call dogfood · /onboard next agentic · /onboard next memory · /onboard next human-gates · /onboard next operator
  Slash: /onboard next journey (aliases edge-journey|user-journey|first-run|edge_user_journey)
  Back: /onboard next · /onboard next wizard · /onboard next status · /onboard next export

Locks: dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA · residual PASS ≠ invent Edge Memory GA declared · portal HITL · portal HITL when connect · agent MCP cannot write installs · catalog ≠ Connected · no invent TUI portal SSO · host not auto · never invent install green / Connected / INSTALL_STORE APPLY · free eng s1558 · free-floor peer s1560+ mention only · stage 5 soft residual s1562 · stage 6 E4 soft residual s1566 · deeper tool-call soft residual s1578 · Wave C wizard residual s1570`)
}

// AionAgentOnboardingNextWizardLane residual-honest guided first-run wizard residual for
// /onboard next wizard (s1570 Wave C). Static offline — deeper guided residual after Wave B
// journey map (s1558). For stages 1–7: short step + primary residual-honest next action + honesty non-claim.
// Soft offline dogfood: /onboard next wizard dogfood (session soft ≠ live dogfood · soft offline ≠ invent Connected).
// Explicit Wave C residual scope: deeper guided residual map + soft dogfood · NOT invent full interactive auto wizard.
// Never invents: auto memory host · TUI portal SSO · Connected · dual_write ON · Memory GA · agent install APPLY ·
// book-demo ON · Edge Memory GA declared · forever-green interactive wizard UX.
// Aliases: first-run-wizard|guided|wave-c|wave_c|wizard-residual (wizard primary; not setup-lifecycle alias).
// Companion: /onboard next journey (Wave B map) · setup · portal-hitl · e4 · human-gates.
// free eng s1570 · free-floor peer s1572+ mention only (do not rewrite free-floor).
func AionAgentOnboardingNextWizardLane() string {
	softLabel := WizardSoftSessionLabel()
	return AionAgentOnboardingStartHere() + "\n\n" + strings.TrimSpace(fmt.Sprintf(`aion onboard next wizard lane (residual-honest · s1570 Wave C · first-run wizard residual · guided residual map · no MCP dial):
  Path: guided first-run wizard residual — deeper residual-honest step map after Wave B journey · NOT invent full interactive auto wizard
  Product: Wave C first-run wizard residual (s1570) after Wave B journey map (s1558) — residual-honest guided residual · free eng s1570 · free-floor peer s1572+ mention only
  Wave C residual scope: deeper guided residual map + soft dogfood · residual PASS ≠ invent full interactive auto wizard · residual PASS ≠ invent Edge Memory GA declared
  Stages (guided residual · step + primary next action + honesty non-claim):
    1. Signup — portal console.iome.sh create/join org · optional pure local (skip for pure local memory)
       primary: open https://console.iome.sh (browser) · optional pure local skip
       honesty: signup ≠ Memory GA · ≠ Connected · book-demo OFF
    2. Download TUI — install binary (go install / releases / make build)
       primary: go install / make build / releases download
       honesty: binary ≠ platform control plane · public OSS ≠ invent multi-tenant mesh
    3. TUI auth/keys — LLM API keys (default cascade) · optional Ollama local · not platform-bundled weights
       primary: configure env/config · optional Ollama · not platform SSO
       honesty: no invent TUI portal SSO · LLM keys ≠ platform SSO · Ollama = local only
    4. Setup — stage 4 detail residual · /onboard next setup · /setup init · preflight
       primary: /onboard next setup · /setup init · /setup preflight · iomesh setup
       honesty: setup PASS ≠ invent Connected · dual_write OFF · dual_write never auto ON · E10 Open
    5. Connectors — stage 5 residual · /onboard next portal-hitl · list/plan + portal HITL when connect
       primary: /onboard next portal-hitl · /integrations list|plan|status · /onboard next agentic
       honesty: portal HITL when connect · agent MCP cannot write installs · catalog ≠ Connected
    6. Local store — stage 6 residual · /onboard next e4 · iomesh-memory-mcp attach
       primary: /onboard next e4 · /onboard next memory · /memory status · iomesh-memory-mcp attach
       honesty: dual_write OFF · not Memory GA · Edge Memory GA candidacy only · host not auto · residual PASS ≠ invent Edge Memory GA declared · E10 Open
    7. Analyze — /setup analyze · /memory digest · optional mesh Ops Pack pull
       primary: /setup analyze · /memory digest · /onboard next memory-pull (optional)
       honesty: analyze ≠ invent Connected · pull ≠ invent Memory GA
  Soft offline first-run wizard dogfood (s1570 Wave C · session soft ≠ live dogfood):
    · session soft state: %s (default wizard_soft_not_run · after run soft_offline_wizard_session_pass|fail)
    · slash: /onboard next wizard dogfood (aliases soft|samples|offline|wizard-soft) · bare /onboard next wizard stays this board
    · soft offline ≠ invent Connected · residual PASS ≠ live dogfood · session soft ≠ live dogfood · residual PASS ≠ invent Edge Memory GA declared
  Honesty:
    · Wave C · first-run wizard residual · free eng s1570 · free-floor peer s1572+ mention only
    · dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · E10 Open
    · portal HITL when connect · agent MCP cannot write installs · catalog ≠ Connected
    · residual PASS ≠ invent Edge Memory GA declared · residual PASS ≠ invent Edge Memory GA
    · no invent TUI portal SSO · host not auto · residual PASS ≠ invent full interactive auto wizard
    · residual PASS ≠ live dogfood · session soft ≠ live dogfood · soft offline ≠ invent Connected · PASS ≠ live APPLY
    · open boxes stay open · never invent install green / Connected / INSTALL_STORE APPLY
  Companion: /onboard next journey (Wave B map) · /onboard next setup · /onboard next portal-hitl · /onboard next e4 · /onboard next human-gates · /onboard next human-gates dogfood (s1574 still-human APPLY soft) · /onboard next wizard dogfood
  Slash: /onboard next wizard (aliases first-run-wizard|guided|wave-c|wave_c|wizard-residual)
    · dogfood: /onboard next wizard dogfood (aliases soft|samples|offline|wizard-soft)
    · NOT setup-lifecycle|lifecycle|setup_lifecycle (those stay setup lane) · NOT invent full interactive auto wizard
  Back: /onboard next · /onboard next journey · /onboard next status · /onboard next export
  Docs: docs/architecture/edge-user-journey.md Wave C · docs/architecture/setup-lifecycle.md · docs/architecture/memory-edge-usage-demo.md

Locks: dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · residual PASS ≠ invent Edge Memory GA · E10 Open · portal HITL when connect · agent MCP cannot write installs · catalog ≠ Connected · no invent TUI portal SSO · host not auto · residual PASS ≠ invent full interactive auto wizard · residual PASS ≠ live dogfood · PASS ≠ live APPLY · open boxes stay open · never invent install green / Connected / INSTALL_STORE APPLY · dual_write stays OFF (never invent primary ON) · E10 stays Open (never invent closed) · soft offline ≠ invent Connected · session soft ≠ live dogfood · free eng s1570 · free-floor peer s1572+ mention only · still-human APPLY open · companion /onboard next human-gates dogfood (s1574) · %s`, softLabel, softLabel))
}

// AionAgentOnboardingNextPortalHITLLane residual-honest journey stage-5 portal HITL connectors
// board for /onboard next portal-hitl (s1562). Static offline — no MCP dial.
// Path: MCP list/plan → browser portal HITL → human finishes OAuth/install.
// Stage 5 of edge-user-journey (connectors / portal HITL when connect).
// Soft offline dogfood: /onboard next portal-hitl dogfood (session soft ≠ live dogfood · soft offline ≠ invent Connected).
// Proven deep-link paths: /integrations/{id} · /integrations/add?template={id} · /integrations.
// template= ≠ install APPLY · agent MCP cannot write installs · catalog ≠ Connected.
// Aliases: hitl|portal_hitl|portal-dogfood|stage5|connectors-hitl.
// Companion: /onboard next agentic · /onboard next journey · /onboard next agentic dogfood · soft dogfood residual.
// free eng s1562 · free-floor peer s1564+ mention only (do not rewrite free-floor).
func AionAgentOnboardingNextPortalHITLLane() string {
	softLabel := PortalHITLSoftSessionLabel()
	return strings.TrimSpace(fmt.Sprintf(`aion onboard next portal-hitl lane (residual-honest · s1562 · journey stage 5 · portal HITL when connect · no MCP dial):
  Path: MCP list/plan → browser portal HITL → human finishes OAuth/install — agent MCP cannot write installs
  Product: edge-user-journey stage 5 connectors residual — portal HITL when connect · free eng s1562 · free-floor peer s1564+ mention only
  Steps:
    1. MCP list/plan residual-honest — /integrations list|plan|status · catalog ≠ Connected · never invent Connected
    2. Plan deep links (browser HITL only) — proven paths: /integrations/{id} · /integrations/add?template={id} · /integrations
    3. Browser portal HITL @ https://console.iome.sh/integrations — human finishes OAuth/install · portal HITL when connect
    4. Agent/MCP mint/copy/probe companion @ https://console.iome.sh/settings/agent · /onboard portal (complementary · probe only ≠ Memory GA)
    5. Soft offline portal HITL dogfood residual — /onboard next portal-hitl dogfood · soft offline ≠ invent Connected · session soft ≠ live dogfood
  Proven portal paths (static offline · browser HITL only):
    · /integrations/{id}
    · /integrations/add?template={id}
    · /integrations
    · template= ≠ install APPLY · portal HITL still · deep links never invent install green
  Soft offline portal HITL dogfood (s1562 · session soft ≠ live dogfood):
    · session soft state: %s (default portal_hitl_soft_not_run · after run soft_offline_portal_hitl_session_pass|fail)
    · slash: /onboard next portal-hitl dogfood (aliases soft|samples|offline|portal-hitl-soft) · bare /onboard next portal-hitl stays this board
    · soft offline ≠ invent Connected · residual PASS ≠ live dogfood · session soft ≠ live dogfood · portal HITL still
  Honesty:
    · journey stage 5 · portal HITL when connect · portal HITL still · portal_hitl_still
    · agent MCP cannot write installs · catalog ≠ Connected · never invent Connected
    · template= ≠ install APPLY · browser portal HITL · human finishes OAuth/install
    · dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only
    · residual PASS ≠ invent Edge Memory GA · residual PASS ≠ live dogfood · PASS ≠ live APPLY
    · soft offline ≠ invent Connected · session soft ≠ live dogfood · open boxes stay open
    · free eng s1562 · free-floor peer s1564+ mention only (do not rewrite free-floor)
  Companion: /onboard next agentic · /onboard next journey · /onboard next agentic dogfood · /onboard next portal-hitl dogfood · /onboard portal · /integrations list|plan|status · /onboard next human-gates
  Slash: /onboard next portal-hitl (aliases hitl|portal_hitl|portal-dogfood|stage5|connectors-hitl)
    · dogfood: /onboard next portal-hitl dogfood (aliases soft|samples|offline|portal-hitl-soft)
    · NOT bare portal|agent-mcp under /onboard (portal handoff) · NOT bare agentic list/plan soft under agentic (s1422 independent)
  Back: /onboard next · /onboard next journey · /onboard next agentic · /onboard next status · /onboard next export
  Docs: docs/architecture/edge-user-journey.md stage 5 · docs/architecture/agent-integrations-setup.md

Locks: dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA · residual PASS ≠ live dogfood · PASS ≠ live APPLY · open boxes stay open · never invent install green / Connected / INSTALL_STORE APPLY · catalog ≠ Connected · template= ≠ install APPLY · agent MCP cannot write installs · portal HITL when connect · portal HITL still · portal_hitl_still · soft offline ≠ invent Connected · session soft ≠ live dogfood · free eng s1562 · free-floor peer s1564+ mention only · %s`, softLabel, softLabel))
}

// AionAgentOnboardingNextE4Lane residual-honest journey stage-6 E4 client-attach board
// for /onboard next e4 (s1566). Static offline — no MCP dial · never start host.
// Path: lean iomesh-memory-mcp HTTP host → iomesh mcp --connect · tools=6 stamp residual (s1508 evidence).
// Stage 6 of edge-user-journey (local store / MCP attach).
// Soft offline dogfood: /onboard next e4 dogfood (session soft ≠ live dogfood · soft offline ≠ invent Connected).
// Honesty: dual_write OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared ·
// E10 Open · tip ≠ invent forever-green product dogfood · residual PASS ≠ live dogfood.
// Aliases: e4-dogfood|client-attach|edge-memory-e4|e4_attach.
// Companion: /onboard next memory · /onboard next journey · /onboard next memory-pull · soft dogfood residual.
// free eng s1566 · free-floor peer s1568+ mention only (do not rewrite free-floor).
func AionAgentOnboardingNextE4Lane() string {
	softLabel := E4SoftSessionLabel()
	return strings.TrimSpace(fmt.Sprintf(`aion onboard next e4 lane (residual-honest · s1566 · journey stage 6 · E4 client attach · no MCP dial):
  Path: local-primary iomesh-memory-mcp lean host HTTP → iomesh mcp --connect · tools=6 stamp residual — never start host from this board
  Product: edge-user-journey stage 6 local store / MCP attach residual — E4 client attach · free eng s1566 · free-floor peer s1568+ mention only
  Steps:
    1. Product host local-primary — iomesh-memory-mcp only (public · go install / compose) · dual_write OFF · not Memory GA
    2. E4 client attach residual — lean host HTTP → iomesh mcp --connect · tools=6 stamp residual · Edge Memory GA candidacy only
    3. Evidence stamp (static offline) — docs/EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md · residual PASS ≠ invent Edge Memory GA declared · E10 Open
    4. Soft offline E4 dogfood residual — /onboard next e4 dogfood · soft offline ≠ invent Connected · session soft ≠ live dogfood · residual PASS ≠ live dogfood
    5. Companion memory lane — /onboard next memory · /memory status · tip ≠ invent forever-green product dogfood
  Soft offline E4 dogfood (s1566 · session soft ≠ live dogfood):
    · session soft state: %s (default e4_soft_not_run · after run soft_offline_e4_session_pass|fail)
    · slash: /onboard next e4 dogfood (aliases soft|samples|offline|e4-soft) · bare /onboard next e4 stays this board
    · soft offline ≠ invent Connected · residual PASS ≠ live dogfood · session soft ≠ live dogfood · residual PASS ≠ invent Edge Memory GA declared
  Deeper tool-call residual (s1578 · after tools=6 attach · soft offline ≠ invent Connected):
    · companion: /onboard next tool-call · soft /onboard next tool-call dogfood — ingest→retrieve→list→as-of operator map residual
    · Partial→client-attach-evidence · deeper tool-call residual candidacy only · free eng s1578 · free-floor peer s1580+ mention only
  Honesty:
    · journey stage 6 · E4 client attach · client attach · tools=6 · iomesh mcp --connect
    · iomesh-memory-mcp · local-primary · dual_write OFF · book-demo OFF · not Memory GA
    · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · E10 Open
    · tip ≠ invent forever-green product dogfood · residual PASS ≠ live dogfood · PASS ≠ live APPLY
    · soft offline ≠ invent Connected · session soft ≠ live dogfood · open boxes stay open
    · free eng s1566 · free-floor peer s1568+ mention only (do not rewrite free-floor)
  Companion: /onboard next memory · /onboard next journey · /onboard next memory-pull · /onboard next e4 dogfood · /onboard next tool-call · /onboard next tool-call dogfood · /onboard next e10 · /onboard next e10 dogfood · /memory status · /onboard next human-gates
  Slash: /onboard next e4 (aliases e4-dogfood|client-attach|edge-memory-e4|e4_attach)
    · dogfood: /onboard next e4 dogfood (aliases soft|samples|offline|e4-soft)
    · deeper: /onboard next tool-call (s1578) · soft /onboard next tool-call dogfood
    · E10 Open reaffirm: /onboard next e10 (s1586) · soft residual-check /onboard next e10 dogfood
    · NOT bare mcp|palace under /onboard next (memory lane) · NOT live host start · dual_write stays OFF (never invent primary ON)
  Back: /onboard next · /onboard next journey · /onboard next memory · /onboard next tool-call · /onboard next e10 · /onboard next status · /onboard next export
  Docs: docs/EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md · docs/architecture/edge-user-journey.md stage 6 · docs/architecture/memory-mcp.md · docs/architecture/oss-packaging-boundary.md

Locks: dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · E10 Open · tip ≠ invent forever-green product dogfood · residual PASS ≠ live dogfood · PASS ≠ live APPLY · open boxes stay open · never invent install green / Connected / INSTALL_STORE APPLY · dual_write stays OFF (never invent primary ON) · E10 stays Open (never invent closed) · soft offline ≠ invent Connected · session soft ≠ live dogfood · free eng s1566 · free-floor peer s1568+ mention only · companion deeper tool-call s1578 · companion E10 Open reaffirm s1586 · %s`, softLabel, softLabel))
}

// AionAgentOnboardingNextToolCallLane residual-honest deeper tool-call dogfood board for
// /onboard next tool-call (s1578 free eng residual). Static offline — journey stage 6/7 depth
// after E4 attach (tools=6 / iomesh mcp --connect). Operator map for ingest → retrieve → list →
// as-of/status soft residual only unless operator runs live. Never dials MCP · never starts host.
// Soft offline dogfood: /onboard next tool-call dogfood (session soft ≠ live dogfood).
// Aliases: tool-calls|deeper-e4|e4-tools|ingest-retrieve|tool_call.
// free eng s1578 · free-floor peer s1580+ mention only (do not rewrite free-floor).
func AionAgentOnboardingNextToolCallLane() string {
	softLabel := ToolCallSoftSessionLabel()
	return strings.TrimSpace(fmt.Sprintf(`aion onboard next tool-call lane (residual-honest · s1578 · deeper tool-call residual · journey stage 6/7 · no MCP dial):
  Path: after lean host + E4 attach · residual-honest operator map for ingest → retrieve → list → as-of/status — soft offline residual only unless operator runs live
  Product: edge-user-journey stage 6/7 depth after E4 client attach (tools=6 stamp residual) — free eng s1578 · free-floor peer s1580+ mention only
  Steps:
    1. Companion E4 attach residual — /onboard next e4 · tools=6 · iomesh mcp --connect · s1508/s1566 attach stamp residual · dual_write OFF · not Memory GA
    2. Deeper tool path names (operator map · soft offline residual): memory_ingest_turn → memory_retrieve → memory_search_semantic → memory_list → memory_compact_status → memory_facts_as_of
    3. Evidence / stamp residual — Partial→client-attach-evidence · docs/EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md · residual PASS ≠ invent Edge Memory GA declared · E10 Open
    4. Soft offline deeper tool-call dogfood residual — /onboard next tool-call dogfood · soft offline ≠ invent Connected · session soft ≠ live dogfood · residual PASS ≠ live dogfood
    5. Companion memory / journey — /onboard next memory · /onboard next journey · /memory status · tip ≠ invent forever-green product dogfood
  Soft offline deeper tool-call dogfood (s1578 · session soft ≠ live dogfood):
    · session soft state: %s (default tool_call_soft_not_run · after run soft_offline_tool_call_session_pass|fail)
    · slash: /onboard next tool-call dogfood (aliases soft|samples|offline|tool-call-soft) · bare /onboard next tool-call stays this board
    · soft offline ≠ invent Connected · residual PASS ≠ live dogfood · session soft ≠ live dogfood · residual PASS ≠ invent Edge Memory GA declared
  Tool path honesty residual (static offline · never dial MCP · never start host):
    · memory_ingest_turn · memory_retrieve · memory_search_semantic
    · memory_list · memory_compact_status · memory_facts_as_of
    · path shape: ingest → retrieve → list → as-of/status (soft offline residual candidacy · not forever-green full product dogfood)
  Honesty:
    · deeper tool-call residual · journey stage 6/7 · Partial→client-attach-evidence
    · companion /onboard next e4 · E4 attach tools=6 stamp residual · s1508 · s1566
    · iomesh-memory-mcp · dual_write OFF · book-demo OFF · not Memory GA
    · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · E10 Open
    · tip ≠ invent forever-green product dogfood · residual PASS ≠ live dogfood · PASS ≠ live APPLY
    · soft offline ≠ invent Connected · session soft ≠ live dogfood · open boxes stay open
    · free eng s1578 · free-floor peer s1580+ mention only (do not rewrite free-floor)
  Companion: /onboard next e4 · /onboard next e4 dogfood · /onboard next memory · /onboard next journey · /onboard next tool-call dogfood · /onboard next e10 · /onboard next e10 dogfood · /memory status · /onboard next human-gates
  Slash: /onboard next tool-call (aliases tool-calls|deeper-e4|e4-tools|ingest-retrieve|tool_call)
    · dogfood: /onboard next tool-call dogfood (aliases soft|samples|offline|tool-call-soft)
    · E10 Open reaffirm: /onboard next e10 (s1586) · soft residual-check /onboard next e10 dogfood
    · NOT bare mcp|palace under /onboard next (memory lane) · NOT bare e4 (E4 attach lane) · NOT live host start · dual_write stays OFF (never invent primary ON)
  Back: /onboard next · /onboard next e4 · /onboard next e10 · /onboard next journey · /onboard next memory · /onboard next status · /onboard next export
  Docs: docs/EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md · docs/architecture/edge-user-journey.md stage 6/7 · docs/architecture/memory-mcp.md · docs/architecture/oss-packaging-boundary.md

Locks: dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · E10 Open · tip ≠ invent forever-green product dogfood · residual PASS ≠ live dogfood · PASS ≠ live APPLY · open boxes stay open · never invent install green / Connected / INSTALL_STORE APPLY · dual_write stays OFF (never invent primary ON) · E10 stays Open (never invent closed) · soft offline ≠ invent Connected · session soft ≠ live dogfood · free eng s1578 · free-floor peer s1580+ mention only · Partial→client-attach-evidence · deeper tool-call residual candidacy (not forever-green full product dogfood) · companion E10 Open reaffirm /onboard next e10 (s1586) · %s`, softLabel, softLabel))
}

// AionAgentOnboardingNextE10Lane residual-honest E10 Open reaffirm residual-check board for
// /onboard next e10 (s1586 free eng residual). Static offline — pin E10 Open after OSS packaging continuum.
// Never dials MCP · never starts host. Soft residual-check: /onboard next e10 dogfood.
// Honesty: residual PASS ≠ invent E10 closed · residual PASS ≠ invent Edge Memory GA declared ·
// Edge Memory GA candidacy only · founder sign-off only if declaring Edge Memory GA · candidacy allowed without E10 ·
// Live APPLY still human · PASS ≠ live APPLY · dual_write OFF · book-demo OFF · not Memory GA.
// Aliases: e10-open|edge-memory-e10|ga-signoff|e10_open.
// Soft residual-check aliases: soft|samples|offline|e10-soft|residual-check.
// free eng s1586 · free-floor peer s1588+ mention only (do not rewrite free-floor).
// Platform residual honesty group companion (s1582 packaging continuum).
func AionAgentOnboardingNextE10Lane() string {
	softLabel := E10SoftSessionLabel()
	return strings.TrimSpace(fmt.Sprintf(`aion onboard next e10 lane (residual-honest · s1586 · E10 Open reaffirm · residual-check · no MCP dial):
  Path: residual-honest E10 Open reaffirm after OSS packaging continuum — soft offline residual-check only · never invent E10 closed / Edge Memory GA declared / live APPLY as green
  Product: Platform residual honesty (optional anti-claims) · pin E10 Open · free eng s1586 · free-floor peer s1588+ mention only
  Steps:
    1. Pin E10 Open — founder/GTM Edge Memory GA sign-off remains Open · residual PASS ≠ invent E10 closed · E10 Open
    2. Candidacy honesty — Edge Memory GA candidacy only · not Memory GA · residual PASS ≠ invent Edge Memory GA declared · founder sign-off only if declaring Edge Memory GA · candidacy allowed without E10
    3. Live APPLY still human — PASS ≠ live APPLY · open boxes stay open · companion /onboard next human-gates · still-human APPLY
    4. Soft offline E10 residual-check — /onboard next e10 dogfood · residual-check · soft offline ≠ invent Connected · session soft ≠ live dogfood · residual PASS ≠ live dogfood
    5. Companion continuum — /onboard next e4 · /onboard next tool-call · OSS packaging · MIT harness · not control plane · docs/architecture/oss-packaging-boundary.md
  Soft offline E10 residual-check (s1586 · session soft ≠ live dogfood):
    · session soft state: %s (default e10_soft_not_run · after run soft_offline_e10_session_pass|fail)
    · slash: /onboard next e10 dogfood (aliases soft|samples|offline|e10-soft|residual-check) · bare /onboard next e10 stays this board
    · soft offline ≠ invent Connected · residual PASS ≠ live dogfood · session soft ≠ live dogfood · residual PASS ≠ invent Edge Memory GA declared · residual PASS ≠ invent E10 closed
  Honesty:
    · E10 Open · E10 Open reaffirm · residual PASS ≠ invent E10 closed · residual PASS ≠ invent Edge Memory GA declared
    · Edge Memory GA candidacy only · not Memory GA · dual_write OFF · book-demo OFF
    · founder sign-off only if declaring Edge Memory GA · candidacy allowed without E10 · PASS ≠ live APPLY
    · residual-check · residual PASS ≠ live dogfood · session soft ≠ live dogfood · soft offline ≠ invent Connected
    · OSS packaging · MIT harness · not control plane · residual PASS ≠ invent control plane in MIT repo
    · free eng s1586 · free-floor peer s1588+ mention only (do not rewrite free-floor)
  Companion: /onboard next e4 · /onboard next e4 dogfood · /onboard next tool-call · /onboard next human-gates · /onboard next human-gates dogfood · /onboard next e10 dogfood · /onboard next · docs/architecture/oss-packaging-boundary.md · docs/architecture/edge-user-journey.md · docs/EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md
  Slash: /onboard next e10 (aliases e10-open|edge-memory-e10|ga-signoff|e10_open)
    · dogfood / residual-check: /onboard next e10 dogfood (aliases soft|samples|offline|e10-soft|residual-check)
    · NOT bare e4 (E4 attach lane) · NOT bare human-gates · NOT invent E10 closed · dual_write stays OFF (never invent primary ON)
  Back: /onboard next · /onboard next e4 · /onboard next human-gates · /onboard next tool-call · /onboard next status · /onboard next export
  Docs: docs/architecture/oss-packaging-boundary.md · docs/architecture/edge-user-journey.md · docs/EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md · docs/architecture/memory-mcp.md

Locks: dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · residual PASS ≠ invent E10 closed · E10 Open · founder sign-off only if declaring Edge Memory GA · candidacy allowed without E10 · residual PASS ≠ live dogfood · PASS ≠ live APPLY · open boxes stay open · never invent install green / Connected / INSTALL_STORE APPLY · dual_write stays OFF (never invent primary ON) · E10 stays Open (never invent closed) · soft offline ≠ invent Connected · session soft ≠ live dogfood · residual-check · MIT harness · not control plane · free eng s1586 · free-floor peer s1588+ mention only · %s`, softLabel, softLabel))
}

// AionAgentOnboardingNextMarketingDemoLane plain-language marketing demo path for
// /onboard next marketing-demo (s1590 + s1594 sales talk track + s1598 GTM claim-support +
// s1602 operator GTM boundary + s1606 GTM wave 6 network/CRM operator paths +
// s1610 GTM wave 7 Hermes mock dogfood / HubSpot dual-control / sales-loop outbox boundary +
// s1614 GTM wave 8 real Hermes daemon path / live dual-control dogfood / operator GTM status +
// s1618 GTM wave 9 scheduled dogfood / mesh outbox ingest / smoke-status operator boundary +
// s1666 free eng Python client SDK peer).
// Short operator script for videos/sales demos of local agent + local memory. Prefer clear
// steps over residual continuum jargon walls. Static offline board — no MCP dial · never starts host.
//
// Script:
//  1. Install/build iomesh
//  2. Set LLM key or Ollama
//  3. /setup init local-memory + preflight
//  4. Start/attach iomesh-memory-mcp
//  5. Show /memory ingest + recall
//  6. Optional mesh only if configured — clearly optional
//
// Sales talk track (s1594 + s1598 + s1602 + s1606 + s1610 + s1614 + s1618 + s1666, optional spoken bullets): local agent + memory beat ·
// private tool-marketing claims catalog for demoable vs do-not-claim · win-back /
// closed-lost are human/HITL sales process (not auto-CRM from TUI) · SEO exports scored
// offline (no auto rank claims) · publish draft→HITL→Hermes is operator tooling (TUI
// does not auto-post) · CRM closed-loop metrics after human actions (TUI is not the CRM) ·
// operator GTM boundary: GSC credentials on operator machine · Hermes exec outside public TUI
// (no social tokens) · CRM vendor adapters human-gated (commercial writes stay human) ·
// s1606: Hermes network dispatch = operator webhook / private runner (not public TUI) ·
// HubSpot / Twenty = operator-box OAuth + human approve · no social or CRM tokens in public harness ·
// s1610: Hermes dogfood = operator mock/daemon (not public TUI) · HubSpot dual control =
// human approve + write allow flags (TUI is not CRM) · sales-loop mesh outbox = operator/local
// envelope wiring (do not invent mesh GTM fleet GA) ·
// s1614: real Hermes daemon is operator-run (TUI does not host it) · live HubSpot/Twenty
// writes need dual control + tokens on operator box (demo stays local agent+memory) ·
// operator GTM status is private tooling (not a product dashboard claim) ·
// s1618: scheduled GTM dogfood = operator cron/offline tooling (not public TUI) · mesh outbox
// ingest is private aion-when-wired dry-run default (do not invent mesh GTM fleet GA) ·
// smoke/status tools (gtm_smoke) are private operator GTM (not a customer dashboard) ·
// s1666: optional mesh clients outside TUI — full mesh client surface for custom services
// lives in public SDKs (Go iomesh-client-sdk-go · Python iomesh-client-sdk-python Beta ·
// not invent 1.0 · GitHub release ≠ invent PyPI green) · TUI stays lean (no SDK dep) ·
// marketing-demo does not require either SDK · local agent + local memory remains the demo path.
//
// Constraints (do not overclaim): dual_write OFF · local memory · not Memory GA ·
// never invent Connected · book-demo OFF · mesh optional · not mesh GTM fleet GA ·
// not invent Python SDK 1.0 / PyPI green · TUI no SDK dep.
// Aliases: marketing|sales-demo|demo-script|gtm-demo
// Do NOT steal: bare demo (demo readiness s1442) · bare sales (sales claims) · bare gtm (GTM drafts).
// Companion: /onboard next memory · e4 · setup · demo (readiness packaging) · sales (claims).
// free eng s1590 · s1594 sales talk track · s1598 GTM claim-support · s1602 operator GTM boundary ·
// s1606 GTM wave 6 · s1610 GTM wave 7 · s1614 GTM wave 8 · s1618 GTM wave 9 ·
// s1666 free eng Python client SDK peer · free-floor peer s1592+ mention only.
func AionAgentOnboardingNextMarketingDemoLane() string {
	return strings.TrimSpace(`aion onboard next marketing-demo path (s1590 · plain-language operator script · local agent + local memory · videos/sales):
  Purpose: short demo path operators can run for videos and sales walkthroughs — show local agent + local memory working end-to-end
  Audience: demo hosts · sales eng · GTM video capture · free eng s1590 · s1594 sales talk track · s1598 GTM claim-support · s1602 operator GTM boundary · s1606 GTM wave 6 · s1610 GTM wave 7 · s1614 GTM wave 8 · s1618 GTM wave 9 · s1666 free eng Python client SDK peer · free-floor peer s1592+ mention only

  Demo script (follow in order):
    1. Install / build iomesh
       · go install · make build · or download a release binary
       · binary in PATH: iomesh --help
    2. Set LLM key or Ollama
       · cloud LLM: export API key for your provider (OpenAI / Anthropic / Gemini / etc.)
       · or local: run Ollama and point the TUI at it
       · LLM keys are for the agent — not platform SSO
    3. /setup init local-memory + preflight
       · /setup init (local-memory default) · iomesh setup init
       · /setup preflight — confirms config shape before attach
       · dual_write OFF (default stays off)
    4. Start / attach iomesh-memory-mcp
       · go install github.com/iome-sh/iomesh-memory-mcp/cmd/iomesh-memory-mcp@main
       · run host (HTTP http://127.0.0.1:8080/mcp or stdio) · attach in TUI [[mcp.servers]] or iomesh mcp --connect
       · product host = iomesh-memory-mcp · local-primary only
    5. Show /memory ingest + recall
       · /memory status · agent turn that stores a fact · /memory digest or retrieve/recall
       · demonstrate: agent remembers what you just told it (local palace)
    6. Optional: mesh only if configured
       · mesh is optional — skip entirely for pure local demos
       · if mesh credentials exist: /mesh status · optional /onboard next memory-pull
       · never require mesh for the core local agent + memory story

  Sales talk track (optional spoken bullets · s1594 · s1598 · s1602 · s1606 · s1610 · s1614 · s1618 · s1666):
    · Beat: local agent + local memory — ingest a fact, recall it on the laptop (dual_write OFF)
    · Claims: check private tool-marketing claims catalog for demoable vs do-not-claim (github.com/iome-sh/tool-marketing · operator-only)
    · Win-back / closed-lost: sales process (humans / HITL) — TUI does not auto-CRM follow-ups
    · SEO / publish: Search Console exports scored offline — no auto rank claims · draft → human approve → Hermes handoff is operator tooling — TUI does not auto-post
    · CRM: closed-loop metrics after human CRM actions — TUI is not the CRM
    · Operator GTM boundary (s1602): Search Console credentials stay on the operator machine — demos use offline exports / local tooling · Hermes exec for publish is outside the public TUI (no social tokens · never auto-post)
    · CRM vendor adapters / metrics are operator GTM (human-gated) — commercial CRM writes stay human
    · Hermes network dispatch (s1606): operator webhook / private runner — not the public TUI (secrets outside git)
    · HubSpot / Twenty CRM (s1606): operator-box OAuth + human approve — TUI is not the CRM
    · No social or CRM tokens in the public harness
    · Hermes dogfood (s1610): operator mock / daemon on the operator box — not the public TUI (demo is not Hermes control plane)
    · HubSpot dual control (s1610): commercial writes need human approve + write allow flags — TUI is not CRM
    · Sales-loop mesh outbox (s1610): operator / local envelope wiring only — do not invent mesh GTM fleet GA
    · Real Hermes daemon (s1614): operator-run on the operator box — TUI does not host it
    · Live HubSpot / Twenty writes (s1614): dual control + tokens on the operator box (default dry) — demo stays local agent + local memory
    · Operator GTM status (s1614): private tooling — not a product dashboard claim
    · Scheduled GTM dogfood (s1618): operator cron / offline tooling — not the public TUI (demo is not a GTM scheduler)
    · Mesh outbox ingest (s1618): private aion when wired (dry-run default) — do not invent mesh GTM fleet GA
    · Smoke / status tools (s1618 · gtm_smoke): private operator GTM — not a customer dashboard
    · Optional mesh clients outside TUI (s1666): full mesh client surface for custom services lives in public SDKs — Go iomesh-client-sdk-go · Python iomesh-client-sdk-python (Beta · v0.10.x · not invent 1.0 · GitHub release ≠ invent PyPI green) · TUI stays lean (no SDK dep) · marketing-demo does not require either SDK · local agent + local memory remains the demo path

  What this supports (honest claims):
    · Local agent harness with multi-provider LLM / Ollama
    · Local memory via iomesh-memory-mcp + local palace · dual_write OFF
    · Operator can install, attach, ingest, and recall on a laptop
    · Optional: public Go / Python mesh SDKs for custom services outside the lean TUI (s1666) — not required for this demo
  What not to claim:
    · not Memory GA · do not invent bare Memory GA product green
    · never invent Connected / install green / INSTALL_STORE APPLY
    · book-demo OFF · do not invent public book-a-demo as live
    · dual_write stays OFF · never invent dual_write as primary ON
    · mesh optional · not required for local demo · not mesh GTM fleet GA
    · not invent Python SDK 1.0 · not invent PyPI green · demo does not require SDKs

  Companion (deeper when needed):
    · /onboard next memory — edge OSS install detail
    · /onboard next e4 — client attach residual (tools=6 stamp)
    · /onboard next setup — full setup lifecycle map
    · /onboard next demo — packaging readiness board (Lighthouse · Landgrab NOT READY)
    · /onboard next sales — may claim / must not claim board
  Docs: docs/architecture/marketing-demo-path.md · docs/architecture/memory-edge-usage-demo.md · docs/architecture/edge-user-journey.md

  Slash: /onboard next marketing-demo (aliases marketing|sales-demo|demo-script|gtm-demo)
    · NOT bare demo|demo-ready|readiness|lighthouse|landgrab (those stay demo readiness s1442)
    · NOT bare sales|claims (sales claims board)
    · NOT bare gtm|drafts (GTM draft lane)
  Back: /onboard next · /onboard next memory · /onboard next demo · /onboard next sales

Locks: dual_write OFF · book-demo OFF · not Memory GA · local memory · local-primary · iomesh-memory-mcp · mesh optional · never invent Connected · never invent install green / INSTALL_STORE APPLY · dual_write stays OFF (never invent primary ON) · TUI stays lean (no SDK dep) · not invent 1.0 · not invent PyPI green · free eng s1590 · s1594 sales talk track · s1598 GTM claim-support · s1602 operator GTM boundary · s1606 GTM wave 6 · s1610 GTM wave 7 · s1614 GTM wave 8 · s1618 GTM wave 9 · s1666 free eng Python client SDK peer · free-floor peer s1592+ mention only`)
}

// AionAgentOnboardingNextAgenticLane residual-honest product plane 3 agentic integrations
// for /onboard next agentic (s1417 + s1422 soft dogfood + portal HITL polish + s1427 dual-auth candidacy tip).
// Static offline — no MCP dial on this board.
// MCP list (catalog / list connectors residual-honest · catalog ≠ Connected) ·
// MCP plan (plan_connector_setup → proven portal deep links only · browser HITL) ·
// portal HITL for OAuth/install (agent MCP cannot write installs).
// Proven deep-link paths: /integrations/{id} · /integrations/add?template={id} · /integrations.
// template= ≠ install APPLY green · deep_links = browser HITL only.
// list_org fail-open ≠ empty-as-none · available=false default residual · never invent Connected.
// Complements (does not replace) /onboard portal mint/copy/probe and human-gates lane.
// s1422: soft offline list/plan dogfood via /onboard next agentic dogfood (session soft ≠ live dogfood).
// s1422: Portal HITL polish block (proven paths · mint/copy/probe complementary · OAuth/install still portal).
// s1427: dual-auth candidacy depth via /onboard next agentic dual-auth (tool ship ≠ dual-auth live).
// Honest residual: path_ready · residual_only · portal_hitl_still · list_plan_not_connected · list_plan_soft_not_run default.
// Aliases: agentic-integrations|integrations|list-plan.
// DO NOT steal: bare mcp (memory lane) · bare portal/agent-mcp (portal handoff) · bare pull (mesh).
// portal-hitl|hitl are s1562 portal HITL lane (journey stage 5) — not agentic aliases.
func AionAgentOnboardingNextAgenticLane() string {
	softLabel := AgenticListPlanSoftSessionLabel()
	return strings.TrimSpace(fmt.Sprintf(`aion onboard next agentic lane (residual-honest · s1417+s1422+s1427 · no MCP dial · product plane 3 · agentic integrations):
  Path: MCP list/plan residual-honest + portal HITL for OAuth/install — agent MCP cannot write installs
  Product plane 3: agentic integrations continuum (post-onboard next lane · complements /onboard portal mint/copy/probe · does not replace human-gates)
  Steps:
    1. MCP list — list_connector_catalog / list connectors residual-honest · catalog ≠ Connected · catalog status ≠ install Connected
    2. MCP plan — plan_connector_setup → proven portal deep links only · browser HITL · template= ≠ install APPLY green
    3. Proven deep_links (browser HITL only): /integrations/{id} · /integrations/add?template={id} · /integrations
    4. Org residual: list_org_connector_installs fail-open · available=false default residual · installs=null · list_org fail-open ≠ empty-as-none · never invent Connected
    5. Portal HITL OAuth/install @ https://console.iome.sh/integrations — agent MCP cannot write installs · deep_links = browser HITL only
    6. Agent/MCP mint/copy/probe companion @ https://console.iome.sh/settings/agent (complementary · not this lane's primary · /onboard portal)
    7. Dual-auth candidacy depth (s1427): /onboard next agentic dual-auth — list_org fail-open · dual_auth_candidacy_open · tool ship ≠ dual-auth live
  Portal HITL polish (s1422 · proven paths only · browser HITL):
    · Proven deep-link paths: /integrations/{id} · /integrations/add?template={id} · /integrations
    · template= ≠ install APPLY · deep_links = browser HITL only (plan deep links never invent install green)
    · Complementary: /onboard portal mint/copy/probe @ https://console.iome.sh/settings/agent (probe only ≠ Memory GA)
    · OAuth/install still portal HITL @ https://console.iome.sh/integrations · agent MCP cannot write installs
    · dual_write OFF · catalog ≠ Connected · list_org fail-open ≠ empty-as-none · never invent Connected
    · Companion portal HITL stage-5 residual (s1562): /onboard next portal-hitl · soft /onboard next portal-hitl dogfood · soft offline ≠ invent Connected · never invent Connected
  Soft offline list/plan dogfood (s1422 · session soft ≠ live dogfood):
    · session soft state: %s (default list_plan_soft_not_run · after run soft_offline_list_plan_session_pass|fail)
    · slash: /onboard next agentic dogfood (aliases soft|samples|offline|list-plan-soft) · bare /onboard next agentic stays this board
    · soft offline list/plan ≠ live dogfood · ≠ invent Connected · portal HITL still · list_org fail-open ≠ empty-as-none
    · tip: after dogfood re-run /onboard next status then /onboard next export so agentic lane reflects session soft
    · companion portal HITL soft dogfood residual (s1562 · independent session marker): /onboard next portal-hitl dogfood · soft offline ≠ invent Connected
  Dual-auth candidacy depth (s1427 · residual-only · tool ship ≠ dual-auth live):
    · dual_auth_candidacy_open · list_org_unavailable · never invent dual-auth live
    · slash: /onboard next agentic dual-auth (aliases candidacy|list-org|org-installs|dual_auth|dual-auth-candidacy)
    · NOT dogfood|soft|samples|offline|list-plan-soft (those stay soft dogfood s1422)
  Honesty:
    · product plane 3 · agentic integrations · MCP list/plan residual-honest
    · plan_connector_setup · portal deep links · browser HITL only
    · template= ≠ install APPLY green · deep_links = browser HITL only
    · list_org fail-open ≠ empty-as-none · available=false default residual
    · agent MCP cannot write installs · catalog ≠ Connected · never invent Connected / install green
    · portal HITL @ https://console.iome.sh/integrations · Agent/MCP mint/copy/probe @ https://console.iome.sh/settings/agent
    · dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · PASS ≠ live APPLY
    · open boxes stay open · rates ~$88/$119 optional · board/export evidence ≠ invent Connected
    · does not claim dual-auth live for list_org · residual soft path only · tool ship ≠ dual-auth live
    · path_ready · residual_only · portal_hitl_still · list_plan_not_connected · dual_auth_candidacy_open · %s (honest vocab)
    · soft offline ≠ live dogfood · session soft ≠ live dogfood
  Slash: /onboard next agentic (aliases agentic-integrations|integrations|list-plan)
    · dogfood: /onboard next agentic dogfood (aliases soft|samples|offline|list-plan-soft)
    · dual-auth: /onboard next agentic dual-auth (aliases candidacy|list-org|org-installs|dual_auth|dual-auth-candidacy)
    · NOT bare mcp (mcp stays memory lane under /onboard next) · NOT bare portal|agent-mcp (portal handoff) · NOT bare pull (mesh)
    · NOT portal-hitl|hitl (those are s1562 portal HITL lane · journey stage 5)
  Companion: /onboard next portal-hitl (s1562 stage 5) · /onboard next portal-hitl dogfood · /onboard portal (mint/copy/probe) · /integrations list|plan|status · /onboard next human-gates · /onboard next status · /onboard next export
  Back: /onboard next · /onboard next status · /onboard status

Locks: dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · PASS ≠ live APPLY · open boxes stay open · never invent install green / Connected / INSTALL_STORE APPLY · catalog ≠ Connected · list_org fail-open ≠ empty-as-none · plan deep links = browser HITL only · template= ≠ install APPLY · agent MCP cannot write installs · portal HITL · list_plan_not_connected · portal_hitl_still · path_ready · residual_only · soft offline ≠ live dogfood · session soft ≠ live dogfood · rates ~$88/$119 optional · board/export evidence ≠ invent Connected · does not claim dual-auth live for list_org · tool ship ≠ dual-auth live · dual_auth_candidacy_open · companion portal-hitl soft residual s1562 · free-floor peer s1564+ mention only`, softLabel, softLabel))
}

// AionAgentOnboardingNextAgenticDualAuthCandidacy residual-honest dual-auth candidacy depth
// for /onboard next agentic dual-auth (s1427 · product plane 3). Static offline — no MCP dial.
// Surfaces list_org_connector_installs residual fail-open shape + dual_auth_candidacy_open honesty.
// tool ship ≠ dual-auth live · PASS ≠ invent dual-auth shipped · never invent empty-as-none
// (installs=null not []) · portal session owns install index · agent MCP cannot write installs.
// Honest vocab: path_ready · residual_only · dual_auth_candidacy_open · list_org_unavailable.
// Aliases (4th token): dual-auth|candidacy|list-org|org-installs|dual_auth|dual-auth-candidacy.
// DO NOT steal: dogfood|soft|samples|offline|list-plan-soft stay soft dogfood (s1422).
// Bare /onboard next agentic stays main agentic board.
func AionAgentOnboardingNextAgenticDualAuthCandidacy() string {
	return strings.TrimSpace(`aion onboard next agentic dual-auth candidacy (residual-honest · s1427 · no MCP dial · product plane 3 · dual_auth_candidacy_open):
  Path: list_org_connector_installs residual fail-open + dual-auth candidacy honesty — tool ship ≠ dual-auth live
  Product plane 3 depth: agentic org installs snapshot residual (post agentic lane s1417 · after soft dogfood s1422)
  Snapshot residual (fail-open shape · never invent live dual-auth):
    · MCP tool: list_org_connector_installs
    · available=false · status=unavailable · installs=null
    · never invent empty-as-none — installs=null not [] · available=false ≠ "none connected"
    · dual_auth_candidacy_open · list_org_unavailable · residual soft path only
  Ownership:
    · portal session owns install index · session-cookie + org membership only
    · agent MCP cannot write installs · catalog ≠ Connected · never invent Connected
    · portal HITL @ https://console.iome.sh/integrations for OAuth/install CRUD
  Honesty:
    · dual_auth_candidacy_open · never invent dual-auth live
    · tool ship ≠ dual-auth live · PASS ≠ invent dual-auth shipped
    · list_org_connector_installs · available=false · status=unavailable · installs=null
    · never invent empty-as-none (installs=null not [])
    · agent MCP cannot write installs · catalog ≠ Connected · never invent Connected
    · portal session owns install index · session-cookie + org membership only
    · portal HITL @ https://console.iome.sh/integrations
    · dual_write OFF · book-demo OFF · not Memory GA
    · residual PASS ≠ live dogfood · PASS ≠ live APPLY · open boxes stay open
    · rates ~$88/$119 optional · board/export evidence ≠ invent Connected
    · path_ready · residual_only · dual_auth_candidacy_open · list_org_unavailable (honest vocab)
  Slash: /onboard next agentic dual-auth (aliases candidacy|list-org|org-installs|dual_auth|dual-auth-candidacy)
    · NOT dogfood|soft|samples|offline|list-plan-soft (those stay soft dogfood s1422)
    · bare /onboard next agentic stays main agentic board
  Companion: /onboard next agentic · /onboard next agentic dogfood · /onboard portal · /onboard next status
  Back: /onboard next agentic · /onboard next status · /onboard status

Locks: dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · PASS ≠ live APPLY · open boxes stay open · never invent dual-auth live · tool ship ≠ dual-auth live · PASS ≠ invent dual-auth shipped · never invent empty-as-none · installs=null not [] · available=false · status=unavailable · dual_auth_candidacy_open · list_org_unavailable · agent MCP cannot write installs · catalog ≠ Connected · never invent Connected · portal session owns install index · session-cookie + org membership only · portal HITL · rates ~$88/$119 optional · board/export evidence ≠ invent Connected · path_ready · residual_only`)
}

// AionAgentOnboardingNextThreePlanes residual-honest consolidated three product planes board
// for /onboard next planes (s1432). Static offline — no MCP dial · no invent stream green /
// pull green / Connected / GA / APPLY. Surfaces product narrative planes:
//
//	1 Mesh (streaming org heartbeats on dept.*)
//	2 Memory-pull / Ops Pack (mesh → local palace egress)
//	3 Agentic integrations (MCP list/plan · portal HITL · dual-auth candidacy open)
//
// Honest vocab: path_ready · residual_only · streams_not_probed · pull_not_probed ·
// portal_hitl_still · list_plan_not_connected · dual_auth_candidacy_open.
// Aliases: three-planes|product-planes|product|pillars|three_planes.
// Do NOT steal: pulse|board (status) · pull (mesh) · mcp (memory).
// Cross-links: /onboard next mesh · memory-pull · agentic · agentic dual-auth · agentic dogfood ·
// status · export · human-gates.
func AionAgentOnboardingNextThreePlanes() string {
	return strings.TrimSpace(`aion onboard next three product planes (residual-honest · s1432 · no MCP dial · offline static · not live dogfood · not live APPLY):
  board (honest vocabulary only — never invent stream green / pull green / Connected / GA / APPLY as success):

  plane 1 · Mesh (I/O Mesh streaming · product plane 1):
    · path_ready · residual_only · streams_not_probed
    · streaming org heartbeats on dept.* · mesh ≠ memory · not OTel/APM · not medical · Palace sunset
    · empty streams honest · never invent stream green / Connected
    · rates ~$88 mesh optional (commercial framing only · not product GA claim)
    · drill: /onboard next mesh (aliases stream|streams|heartbeat|heartbeats|pull) · NOT pulse (pulse stays status board)

  plane 2 · Memory-pull / Ops Pack (customer-edge memory egress · product plane 2):
    · path_ready · residual_only · pull_not_probed
    · mesh → local palace egress · iomesh memory pull · dual_write OFF · not Memory GA
    · Ops Pack ≠ GPU fleet · pull ≠ freemium hosted palace · package load ≠ Ops Pack entitlement · package load ≠ Memory GA
    · never invent pull green / Connected · rates ~$119 Memory Ops Pack optional (commercial framing only)
    · drill: /onboard next memory-pull (aliases ops-pack|pull-path|memorypull|ops_pack) · bare pull stays mesh

  plane 3 · Agentic integrations (MCP list/plan + portal HITL · product plane 3):
    · path_ready · residual_only · portal_hitl_still · list_plan_not_connected · dual_auth_candidacy_open
    · MCP list/plan residual-honest · plan_connector_setup → portal deep links · browser HITL only
    · template= ≠ install APPLY · catalog ≠ Connected · list_org fail-open ≠ empty-as-none
    · agent MCP cannot write installs · never invent Connected · tool ship ≠ dual-auth live
    · dual_auth_candidacy_open · list_org_unavailable · never invent dual-auth live
    · drill: /onboard next agentic (aliases agentic-integrations|integrations|list-plan)
    · soft dogfood: /onboard next agentic dogfood (aliases soft|samples|offline|list-plan-soft) · soft offline ≠ live dogfood
    · dual-auth depth: /onboard next agentic dual-auth (aliases candidacy|list-org|org-installs|dual_auth|dual-auth-candidacy)
    · portal HITL stage 5 (s1562): /onboard next portal-hitl (aliases hitl|portal_hitl|portal-dogfood|stage5|connectors-hitl) · soft /onboard next portal-hitl dogfood · soft offline ≠ invent Connected
    · NOT bare mcp (memory) · NOT bare portal (portal handoff) · NOT bare pull (mesh)

  cross-plane honesty (all planes):
    · dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · PASS ≠ live APPLY
    · open boxes stay open · human-gates still open (tip only) · never invent GA
    · rates ~$88 mesh / ~$119 Memory Ops Pack optional · board/export evidence ≠ invent Connected
    · mesh ≠ memory · never invent stream green / pull green / Connected
    · human-gates tip: /onboard next human-gates (aliases human|gates|apply-gates) · PASS ≠ invent human-gate green

  slash: /onboard next planes (aliases three-planes|product-planes|product|pillars|three_planes)
    · NOT pulse|board (those stay status board) · NOT bare pull (mesh) · NOT bare mcp (memory)
  companion drills: /onboard next mesh · /onboard next memory-pull · /onboard next agentic · /onboard next agentic dual-auth · /onboard next agentic dogfood
  companion boards: /onboard next status (aliases pulse|board) · /onboard next export · /onboard next human-gates · /onboard next sales · /onboard next demo · /onboard next operator · /onboard next · /onboard status
  sales claims tip: /onboard next sales (aliases claims|buyer|claim-matrix|sales-claims|buyer-claims) — may claim / must not claim residual-honest (s1437)
  demo readiness tip: /onboard next demo (aliases demo-ready|readiness|demo-readiness|lighthouse|landgrab) — Lighthouse · book-demo OFF · Landgrab NOT READY (s1442)
  operator matrix tip: /onboard next operator (aliases operator-matrix|ops-matrix|operator-readiness|ops-readiness|matrix) — demo · sales · planes · human-gates consolidate residual-honest (s1447)

Locks: dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · PASS ≠ live APPLY · open boxes stay open · mesh ≠ memory · never invent stream green / pull green / Connected / INSTALL_STORE APPLY · streams_not_probed · pull_not_probed · Ops Pack ≠ GPU fleet · pull ≠ freemium hosted palace · agent MCP cannot write installs · list_plan_not_connected · portal_hitl_still · dual_auth_candidacy_open · tool ship ≠ dual-auth live · never invent dual-auth live · catalog ≠ Connected · plan deep links = browser HITL only · template= ≠ install APPLY · rates ~$88/$119 optional · no invent GA · board/export evidence ≠ invent Connected · path_ready · residual_only · leave ON_SIGNAL unset · Knowledge Beta→GA cannot invent H1/H2 offline`)
}

// AionAgentOnboardingNextSalesClaims residual-honest sales / buyer claims board for
// /onboard next sales (s1437). Static offline — no MCP dial · no invent Connected /
// Memory GA / dual-auth live / human-gate green / freemium palace / Ops Pack GPU.
// Grounded in three product planes (mesh · memory-pull · agentic) via companion
// /onboard next planes. Surfaces may-claim vs must-not-claim for founders/sales ops.
//
// May claim (residual-honest): mesh heartbeats · rates ~$88/~$119 local-primary ·
// dual_write OFF · Palace sunset · Salesforce GA CRM · HubSpot+GTM Beta multi-tenant ·
// knowledge/analytical Beta · agentic list/plan residual · portal HITL · catalog ≠ Connected.
// Must not claim: invent Connected / INSTALL_STORE green · Memory GA · dual_write ON ·
// freemium palace · Ops Pack = GPU · book-demo ON · dual-auth live · empty-as-none ·
// agent MCP write installs · Knowledge/Analytics GA · human-gate green.
//
// Aliases: claims|buyer|claim-matrix|sales-claims|buyer-claims.
// Do NOT steal: product|planes (three-planes) · gtm|drafts (GTM lane) · pulse|board (status).
// Cross-links: /onboard next planes · mesh · memory-pull · agentic · human-gates · status · export.
func AionAgentOnboardingNextSalesClaims() string {
	return strings.TrimSpace(`aion onboard next sales / buyer claims (residual-honest · s1437 · no MCP dial · offline static · not live dogfood · not live APPLY · three-planes grounded):
  board for founders/sales operators — may claim vs must not claim only (never invent Connected / GA / APPLY as success):

  MAY CLAIM (residual-honest · commercial framing only):
    · I/O Mesh = streaming org heartbeats / pulse on dept.* — not OTel/APM replacement · not medical · mesh ≠ memory
    · Mesh base ~$88 · Memory Ops Pack ~$119 (local-primary pull/retain/support · not freemium hosted GPU palace)
    · dual_write OFF · local-primary · Palace sunset · package load ≠ Memory GA
    · Salesforce = GA CRM · HubSpot + GTM suite Beta multi-tenant · guerrilla global-only
    · knowledge / analytical = Beta (not invent GA) · no invent GA knowledge/analytical
    · agentic: MCP list/plan residual-honest · portal HITL · catalog ≠ Connected framing · list_plan_not_connected
    · three product planes via /onboard next planes (mesh · memory-pull · agentic) — streams_not_probed · pull_not_probed · dual_auth_candidacy_open
    · drafts only for GTM · human publish · rates ~$88/$119 optional

  MUST NOT CLAIM (honesty locks — never invent):
    · invent Connected / install APPLY green / INSTALL_STORE green · never invent Connected / INSTALL_STORE APPLY
    · invent Memory GA · invent dual_write as ON · freemium hosted palace · Ops Pack as GPU fleet · Ops Pack ≠ GPU fleet
    · invent book-demo as ON (book-demo OFF) · leave ON_SIGNAL unset
    · invent dual-auth as live · empty-as-none invent · agent MCP write installs · tool ship ≠ dual-auth live · dual_auth_candidacy_open
    · invent Knowledge/Analytics as GA · Knowledge Beta→GA cannot invent H1/H2 offline
    · invent human-gate green (Slack HMAC · Stripe Customers:Write · H1/H2 INSTALL_STORE still human) · PASS ≠ invent human-gate green · open boxes stay open
    · residual PASS ≠ live dogfood · PASS ≠ live APPLY · board/export evidence ≠ invent Connected

  three-planes companion (product narrative ground):
    · plane 1 mesh: streaming org heartbeats · mesh ≠ memory · streams_not_probed · /onboard next mesh
    · plane 2 memory-pull: mesh → local palace egress · dual_write OFF · pull_not_probed · /onboard next memory-pull
    · plane 3 agentic: list/plan residual · portal HITL · list_plan_not_connected · dual_auth_candidacy_open · /onboard next agentic
    · consolidate: /onboard next planes (aliases three-planes|product-planes|product|pillars|three_planes)

  slash: /onboard next sales (aliases claims|buyer|claim-matrix|sales-claims|buyer-claims)
    · NOT product|planes (those stay three-planes board) · NOT gtm|drafts (GTM draft lane) · NOT pulse|board (status board)
  companion: /onboard next planes · /onboard next mesh · /onboard next memory-pull · /onboard next agentic · /onboard next agentic dual-auth
  companion boards: /onboard next status · /onboard next export · /onboard next human-gates · /onboard next gtm · /onboard next demo · /onboard next operator · /onboard next · /onboard status
  operator matrix tip: /onboard next operator (aliases operator-matrix|ops-matrix|operator-readiness|ops-readiness|matrix) — demo · sales · planes · human-gates consolidate residual-honest (s1447)

Locks: dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · PASS ≠ live APPLY · open boxes stay open · never invent Connected / INSTALL_STORE APPLY · mesh ≠ memory · dual_auth_candidacy_open · tool ship ≠ dual-auth live · never invent dual-auth live · drafts only · no auto-send · rates ~$88/$119 optional · Ops Pack ≠ GPU fleet · pull ≠ freemium hosted palace · agent MCP cannot write installs · catalog ≠ Connected · list_plan_not_connected · no invent GA · board/export evidence ≠ invent Connected · leave ON_SIGNAL unset · Knowledge Beta→GA cannot invent H1/H2 offline · PASS ≠ invent human-gate green`)
}

// AionAgentOnboardingNextDemoReadiness residual-honest demo readiness board for
// /onboard next demo (s1442). Static offline — no MCP dial · no invent book-demo ON /
// Landgrab READY / Connected / Memory GA / dual-auth live / human-gate green.
// Surfaces packaging + readiness honesty for founders/operators before public demo:
//
// Packaging: Lighthouse beachhead · B2B SaaS · book-demo OFF · secondary CTA See pricing ·
// leave ON_SIGNAL unset.
// Landgrab: NOT READY / empty-honest · residual PASS ≠ logos met · do not invent book-demo ON.
// Three planes companion: /onboard next planes (mesh · memory-pull · agentic residual-honest).
// Sales claims companion: /onboard next sales (may / must-not residual-honest).
// Human gates still open: Slack HMAC · Stripe Customers:Write · H1/H2 INSTALL_STORE · K-D* ·
// tip /onboard next human-gates.
// Demo path residual: founder-led walkthrough only when scheduled · operator runbook ≠
// public /demo booking live.
//
// Aliases: demo-ready|readiness|demo-readiness|lighthouse|landgrab
// (landgrab stays honesty NOT READY — not invent ready).
// Do NOT steal: sales|claims (sales claims) · planes|product (three-planes) ·
// pulse|board (status) · gtm|drafts (GTM lane).
// Cross-links: /onboard next planes · sales · human-gates · status · export · mesh ·
// memory-pull · agentic.
func AionAgentOnboardingNextDemoReadiness() string {
	return strings.TrimSpace(`aion onboard next demo readiness (residual-honest · s1442 · no MCP dial · offline static · not live dogfood · not live APPLY · Lighthouse packaging · Landgrab NOT READY):
  board for founders/operators — demo packaging + readiness honesty only (never invent book-demo as ON / Landgrab as READY / Connected / GA / APPLY as success):

  packaging (Lighthouse beachhead · B2B SaaS):
    · Lighthouse beachhead packaging residual-honest · B2B SaaS framing
    · book-demo OFF · secondary CTA See pricing · leave ON_SIGNAL unset
    · rates ~$88 mesh / ~$119 Memory Ops Pack optional (commercial framing only · not product GA claim)
    · dual_write OFF · not Memory GA · never invent Connected · dual_auth_candidacy_open
    · GTM drafts only · no auto-send · human publish

  Landgrab (NOT READY · empty-honest):
    · Landgrab NOT READY · empty-honest · do not invent book-demo as ON
    · residual PASS ≠ logos met · residual PASS ≠ live dogfood · PASS ≠ live APPLY
    · open boxes stay open · never invent Landgrab as READY / logos met / public demo booking live
    · landgrab alias stays honesty NOT READY (not invent ready)

  three planes companion (product narrative ground · residual-honest):
    · mesh · memory-pull · agentic residual-honest via /onboard next planes
    · streams_not_probed · pull_not_probed · list_plan_not_connected · dual_auth_candidacy_open
    · mesh ≠ memory · never invent stream green / pull green / Connected
    · consolidate: /onboard next planes (aliases three-planes|product-planes|product|pillars|three_planes)

  sales claims companion (may / must-not residual-honest):
    · may claim / must not claim residual-honest via /onboard next sales
    · never invent Connected / Memory GA / dual-auth live / human-gate green
    · companion: /onboard next sales (aliases claims|buyer|claim-matrix|sales-claims|buyer-claims)

  human gates still open (tip only — do not invent green):
    · still human Slack HMAC · Stripe Customers:Write · H1/H2 INSTALL_STORE · K-D* / D1–D5
    · book-demo OFF · leave ON_SIGNAL unset · PASS ≠ invent human-gate green · PASS ≠ live APPLY
    · open boxes stay open · Knowledge Beta→GA cannot invent H1/H2 offline
    · tip: /onboard next human-gates (aliases human|gates|apply-gates)

  demo path residual (founder-led only):
    · founder-led walkthrough only when scheduled · operator runbook ≠ public /demo booking live
    · never invent public /demo booking live · never invent book-demo as ON
    · board/export evidence ≠ invent Connected

  slash: /onboard next demo (aliases demo-ready|readiness|demo-readiness|lighthouse|landgrab)
    · NOT sales|claims (those stay sales claims board) · NOT product|planes (three-planes) · NOT pulse|board (status) · NOT gtm|drafts (GTM draft lane)
  companion: /onboard next planes · /onboard next sales · /onboard next human-gates · /onboard next mesh · /onboard next memory-pull · /onboard next agentic
  companion boards: /onboard next status · /onboard next export · /onboard next operator · /onboard next · /onboard status
  operator matrix tip: /onboard next operator (aliases operator-matrix|ops-matrix|operator-readiness|ops-readiness|matrix) — demo · sales · planes · human-gates consolidate residual-honest (s1447)

Locks: dual_write OFF · book-demo OFF · Landgrab NOT READY · not Memory GA · residual PASS ≠ live dogfood · PASS ≠ live APPLY · residual PASS ≠ logos met · open boxes stay open · never invent Connected / INSTALL_STORE APPLY · dual_auth_candidacy_open · tool ship ≠ dual-auth live · never invent dual-auth live · rates ~$88/$119 optional · no invent GA · leave ON_SIGNAL unset · Knowledge Beta→GA cannot invent H1/H2 offline · PASS ≠ invent human-gate green · board/export evidence ≠ invent Connected · founder-led walkthrough only when scheduled · operator runbook ≠ public /demo booking live · never invent book-demo as ON · never invent Landgrab as READY`)
}

// AionAgentOnboardingNextOperatorMatrix residual-honest operator readiness matrix board for
// /onboard next operator (s1447). Static offline — no MCP dial · no invent green / Connected /
// Memory GA / dual-auth live / human-gate green / book-demo ON / Landgrab READY.
// Consolidates demo · sales · planes · human-gates · dual-auth candidacy · policy locks for
// founders/operators post-onboard (residual-honest statuses only).
//
// Rows (residual-honest only):
//
//	1 Demo readiness — companion /onboard next demo · Lighthouse · book-demo OFF · Landgrab NOT READY
//	2 Sales claims — companion /onboard next sales · may/must-not residual-honest
//	3 Three planes — companion /onboard next planes · mesh · memory-pull · agentic residual-honest
//	4 Human gates — companion /onboard next human-gates · edge-first (s1550) · knowledge multi-tenant punted · Slack HMAC punted · portal HITL when connect
//	5 Agentic dual-auth — dual_auth_candidacy_open · tool ship ≠ dual-auth live · list_org unavailable
//	6 Policy locks — dual_write OFF · not Memory GA · leave ON_SIGNAL unset · rates ~$88/$119 optional
//	7 Export tip — /onboard next export for offline evidence · board/export evidence ≠ invent Connected
//
// Honest vocab: residual_only · path_ready · still_human · policy_off · not_ready · portal_hitl_still
// Aliases: operator-matrix|ops-matrix|operator-readiness|ops-readiness|matrix
// Do NOT steal: demo|readiness|lighthouse|landgrab (demo) · sales|claims · planes|product ·
// pulse|board (status) · export|receipt.
// Cross-links: /onboard next demo · sales · planes · human-gates · agentic dual-auth · status · export.
func AionAgentOnboardingNextOperatorMatrix() string {
	return strings.TrimSpace(`aion onboard next operator readiness matrix (residual-honest · s1447 · no MCP dial · offline static · not live dogfood · not live APPLY · residual_only · never invent Connected):
  board for founders/operators — consolidates demo · sales · planes · human-gates readiness honesty only (never invent green / Connected / GA / APPLY as success):

  row 1 · Demo readiness (not_ready · residual_only):
    · path_ready packaging residual · Lighthouse beachhead · B2B SaaS framing
    · book-demo OFF · Landgrab NOT READY · residual PASS ≠ logos met
    · founder-led walkthrough only when scheduled · operator runbook ≠ public /demo booking live
    · never invent book-demo as ON · never invent Landgrab as READY / Connected
    · companion: /onboard next demo (aliases demo-ready|readiness|demo-readiness|lighthouse|landgrab)

  row 2 · Sales claims (residual_only · three-planes grounded):
    · may claim / must not claim residual-honest · never invent Connected / Memory GA / dual-auth live
    · sales claims residual-honest only · never invent human-gate green
    · companion: /onboard next sales (aliases claims|buyer|claim-matrix|sales-claims|buyer-claims)

  row 3 · Three product planes (path_ready · residual_only):
    · mesh · memory-pull · agentic residual-honest consolidate
    · streams_not_probed · pull_not_probed · list_plan_not_connected · dual_auth_candidacy_open
    · mesh ≠ memory · never invent stream green / pull green / Connected
    · companion: /onboard next planes (aliases three-planes|product-planes|product|pillars|three_planes)

  row 4 · Human gates (still_human · edge-first s1550 · open policy boxes stay honest):
    · edge-first · knowledge multi-tenant punted · Slack HMAC punted · portal HITL when connect
    · book-demo OFF · leave ON_SIGNAL unset · dual_write OFF · not Memory GA · Edge Memory GA candidacy only
    · residual PASS ≠ invent Edge Memory GA · PASS ≠ invent Connected · H1/H2 not launch gate
    · setup closeout residual ≠ invent APPLY (s1546) · E10 only if claiming Edge Memory GA
    · companion: /onboard next human-gates (aliases human|gates|apply-gates)

  row 5 · Agentic dual-auth candidacy (portal_hitl_still · dual_auth_candidacy_open):
    · dual_auth_candidacy_open · list_org_unavailable · tool ship ≠ dual-auth live
    · list_plan_not_connected · portal_hitl_still · never invent dual-auth live · never invent Connected
    · agent MCP cannot write installs · catalog ≠ Connected · list_org fail-open ≠ empty-as-none
    · companion: /onboard next agentic dual-auth (aliases candidacy|list-org|org-installs|dual_auth|dual-auth-candidacy)
    · soft dogfood tip: /onboard next agentic dogfood · soft offline list/plan ≠ invent Connected

  row 6 · Policy locks (policy_off · residual_only):
    · dual_write OFF · not Memory GA · leave ON_SIGNAL unset · book-demo OFF
    · rates ~$88 mesh / ~$119 Memory Ops Pack optional (commercial framing only · not product GA claim)
    · no invent GA · drafts only · no auto-send · residual PASS ≠ live dogfood · PASS ≠ live APPLY
    · residual PASS ≠ logos met · board/export evidence ≠ invent Connected

  row 7 · Export tip (offline evidence only):
    · /onboard next export (aliases receipt|stamp|evidence) — offline markdown evidence
    · board/export evidence ≠ invent Connected · residual PASS ≠ live dogfood · PASS ≠ live APPLY
    · optional: /onboard next export json · /onboard next status (aliases pulse|board)

  honest vocab (this matrix only): residual_only · path_ready · still_human · policy_off · not_ready · portal_hitl_still · dual_auth_candidacy_open · list_org_unavailable · list_plan_not_connected · streams_not_probed · pull_not_probed

  slash: /onboard next operator (aliases operator-matrix|ops-matrix|operator-readiness|ops-readiness|matrix)
    · NOT demo|readiness|lighthouse|landgrab (those stay demo board) · NOT sales|claims (sales claims) · NOT product|planes (three-planes) · NOT pulse|board (status) · NOT export|receipt (export receipt)
  companion: /onboard next demo · /onboard next sales · /onboard next planes · /onboard next human-gates · /onboard next agentic · /onboard next agentic dual-auth
  companion boards: /onboard next status · /onboard next export · /onboard next · /onboard status

Locks: dual_write OFF · book-demo OFF · Landgrab NOT READY · not Memory GA · residual PASS ≠ live dogfood · PASS ≠ live APPLY · residual PASS ≠ logos met · open boxes stay open · never invent Connected / INSTALL_STORE APPLY · dual_auth_candidacy_open · tool ship ≠ dual-auth live · never invent dual-auth live · rates ~$88/$119 optional · no invent GA · leave ON_SIGNAL unset · Knowledge Beta→GA cannot invent H1/H2 offline · PASS ≠ invent human-gate green · board/export evidence ≠ invent Connected · residual_only · path_ready · still_human · policy_off · not_ready · portal_hitl_still`)
}

// AionAgentOnboardingNextLaneStatus residual-honest post-onboard lane status board for
// /onboard next status (aliases pulse|board) — free eng s1382 (+ s1387 export · s1397 session soft dogfood · s1402 mesh · s1407 memory-pull · s1417 agentic).
// Default path: no MCP dial · no invent install green / Connected / GA / APPLY / stream green / pull green.
// Honest state vocabulary only: path_ready · samples_ok · samples_missing ·
// dogfood_not_run · soft_offline_dogfood_session_pass|fail · skill_ready · residual_only ·
// streams_not_probed · pull_not_probed · portal_hitl_still · list_plan_not_connected.
// Optional soft check: sample package dirs via agentplugins.
// Plugins dogfood state: default dogfood_not_run; after /plugins dogfood session marker
// soft_offline_dogfood_session_pass|fail (session soft ≠ live dogfood · ≠ invent Agent Plugins GA).
// Mesh: path_ready · residual_only · streams_not_probed (never invent live green / Connected).
// Memory-pull: path_ready · residual_only · pull_not_probed (never invent pull green).
// Agentic (s1417+s1422+s1427): path_ready · residual_only · portal_hitl_still · list_plan_not_connected · <soft label>
// · dual_auth_candidacy_open · list_org_unavailable (never invent dual-auth live · tool ship ≠ dual-auth live).
// (never invent Connected · session soft ≠ live dogfood · soft offline ≠ invent Connected).
// Never claims dogfood PASS live, Agent Plugins GA, Memory GA, or install Connected.
func AionAgentOnboardingNextLaneStatus() string {
	samplesState := nextLanePluginsSamplesSoftState()
	dogfoodState := nextLanePluginsDogfoodSessionState()
	agenticSoft := AgenticListPlanSoftSessionLabel()
	// samples_ok|samples_missing is path soft-check only ≠ residual PASS / live dogfood.
	// dogfoodState is session soft marker only (default dogfood_not_run) — ≠ live dogfood.
	// agenticSoft is independent session soft for list/plan (default list_plan_soft_not_run).
	return strings.TrimSpace(fmt.Sprintf(`aion onboard next lane status (residual-honest · s1382+s1397+s1402+s1407+s1417+s1422+s1427 · no MCP dial · not live dogfood):
  board (honest vocabulary only — never invent connected / ga / apply / stream green / pull green as success):

  plugins: %s · %s · path_ready
    · offline samples soft-check only (examples/agent-plugins) · ≠ invent Agent Plugins GA
    · session soft marker ≠ live dogfood · soft offline dogfood ≠ invent Agent Plugins GA
    · residual PASS ≠ live dogfood · never invent install green / Connected / INSTALL_STORE APPLY
    · board/export evidence ≠ invent Connected · drill: /onboard next plugins (aliases plugin|dogfood)

  gtm: skill_ready · path_ready · residual_only
    · /gtm checklist + skill gtm-draft-only-agent path ready
    · drafts only · no auto-send · human publish · GTM checklist ≠ invent GTM agent GA
    · drill: /onboard next gtm (alias drafts)

  memory: path_ready · residual_only
    · dual_write OFF · local-primary · package load ≠ Memory GA · ≠ freemium palace · Palace sunset
    · not Memory GA · book-demo OFF · rates ~$88/$119 optional · mesh ≠ memory · mesh optional for pull
    · edge OSS tip (s1453+s1458+s1463+s1469+s1478+s1508): iomesh-memory-mcp product host · public product attach · go install · no GOPRIVATE · HTTP 8080/mcp or stdio · docker compose still valid · dual_write OFF · not Memory GA · aion broker private · flip complete residual ≠ invent Memory GA · public OSS ≠ invent platform GA · PASS ≠ invent full platform sidecar parity · offline dogfood tip ≠ invent live dogfood as green · E4 client attach (s1508) · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · E10 Open · tip ≠ invent forever-green product dogfood
    · drill: /onboard next memory (aliases mcp|palace)

  mesh: path_ready · residual_only · streams_not_probed
    · I/O Mesh = streaming org heartbeats on dept.* · not OTel/APM · not hosted Memory Palace
    · mesh ≠ memory · empty streams honest · never invent stream green / Connected
    · pull = mesh → local palace egress · dual_write OFF · rates ~$88 mesh / ~$119 Memory Ops Pack optional
    · drill: /onboard next mesh (aliases stream|streams|heartbeat|heartbeats|pull) · NOT pulse (pulse stays this board)

  memory-pull: path_ready · residual_only · pull_not_probed
    · Ops Pack pull path · iomesh memory pull = mesh → local palace egress · dual_write OFF
    · pull ≠ freemium hosted palace · Ops Pack ≠ GPU fleet · never invent pull green
    · package load ≠ Ops Pack entitlement · package load ≠ Memory GA · Palace sunset
    · rates ~$88 mesh / ~$119 Memory Ops Pack optional · residual PASS ≠ live dogfood · PASS ≠ live APPLY
    · drill: /onboard next memory-pull (aliases ops-pack|pull-path|memorypull|ops_pack) · bare pull stays mesh

  setup: path_ready · residual_only · setup_not_probed
    · setup lifecycle P1–P7 closeout residual (s1542+s1558 · stage 4 of edge-user-journey) · offline static map · dual_write OFF · not Memory GA
    · init → preflight → reload → portal HITL → pull → analyze → drift → repair plan/apply --yes
    · package wire ≠ Connected · repair apply ≠ invent Connected · dual_write never auto ON · E10 Open
    · setup closeout residual ≠ invent Edge Memory GA · offline static lane ≠ live dogfood
    · drill: /onboard next setup (aliases setup-lifecycle|lifecycle|setup_lifecycle) · full first-run: /onboard next journey · guided residual: /onboard next wizard

  journey: path_ready · residual_only
    · edge-user-journey first-run map (s1558 Wave B · 7 stages) · dual_write OFF · not Memory GA · Edge Memory GA candidacy only
    · Signup → Download TUI → TUI auth/keys → Setup wizard → Connectors → Local store → Analyze
    · residual PASS ≠ invent Edge Memory GA · portal HITL · no invent TUI portal SSO · host not auto · free eng s1558
    · drill: /onboard next journey (aliases edge-journey|user-journey|first-run|edge_user_journey) · Wave C guided residual: /onboard next wizard

  wizard: path_ready · residual_only
    · first-run wizard residual (s1570 Wave C · guided residual map + soft dogfood) · dual_write OFF · not Memory GA · Edge Memory GA candidacy only
    · residual PASS ≠ invent Edge Memory GA declared · E10 Open · portal HITL when connect · agent MCP cannot write installs · catalog ≠ Connected
    · no invent TUI portal SSO · host not auto · residual PASS ≠ invent full interactive auto wizard · free eng s1570 · free-floor peer s1572+ mention only
    · drill: /onboard next wizard (aliases first-run-wizard|guided|wave-c|wave_c|wizard-residual) · soft: /onboard next wizard dogfood

  tool-call: path_ready · residual_only
    · deeper tool-call residual (s1578 · stage 6/7 depth after E4 attach · soft offline residual) · dual_write OFF · not Memory GA · Edge Memory GA candidacy only
    · memory_ingest_turn · memory_retrieve · memory_list · memory_facts_as_of path residual · Partial→client-attach-evidence
    · residual PASS ≠ invent Edge Memory GA declared · E10 Open · tip ≠ invent forever-green product dogfood · free eng s1578 · free-floor peer s1580+ mention only
    · drill: /onboard next tool-call (aliases tool-calls|deeper-e4|e4-tools|ingest-retrieve|tool_call) · soft: /onboard next tool-call dogfood

  agentic: path_ready · residual_only · portal_hitl_still · list_plan_not_connected · dual_auth_candidacy_open · list_org_unavailable · %s
    · product plane 3 agentic integrations · MCP list/plan residual-honest · never invent Connected
    · plan_connector_setup → portal deep links · browser HITL only · template= ≠ install APPLY green
    · list_org fail-open ≠ empty-as-none · agent MCP cannot write installs · catalog ≠ Connected
    · portal HITL @ https://console.iome.sh/integrations · Agent/MCP @ https://console.iome.sh/settings/agent
    · session soft list/plan ≠ live dogfood · soft offline ≠ invent Connected · portal HITL still
    · dual_auth_candidacy_open · list_org_unavailable · tool ship ≠ dual-auth live · never invent dual-auth live
    · drill: /onboard next agentic (aliases agentic-integrations|integrations|list-plan)
    · soft dogfood: /onboard next agentic dogfood (aliases soft|samples|offline|list-plan-soft)
    · dual-auth depth: /onboard next agentic dual-auth (aliases candidacy|list-org|org-installs|dual_auth|dual-auth-candidacy)
    · NOT bare mcp (memory) · NOT bare portal (portal handoff)

  portal: portal_hitl_still
    · agent MCP cannot write installs · catalog ≠ Connected · portal HITL still for OAuth/install
    · list_org fail-open ≠ empty-as-none · never invent Connected / INSTALL_STORE APPLY
    · Agent/MCP mint/copy/probe @ https://console.iome.sh/settings/agent · connectors @ https://console.iome.sh/integrations

  slash: /onboard next status (aliases pulse|board) · /onboard next export (aliases receipt|stamp|evidence) · /onboard next mesh · /onboard next memory-pull · /onboard next setup (aliases setup-lifecycle|lifecycle|setup_lifecycle) · /onboard next journey (aliases edge-journey|user-journey|first-run|edge_user_journey) · /onboard next wizard (aliases first-run-wizard|guided|wave-c|wave_c|wizard-residual) · /onboard next agentic (aliases agentic-integrations|integrations|list-plan) · /onboard next agentic dogfood · /onboard next agentic dual-auth · /onboard next planes (aliases three-planes|product-planes|product|pillars|three_planes) · /onboard next sales (aliases claims|buyer|claim-matrix|sales-claims|buyer-claims) · /onboard next demo (aliases demo-ready|readiness|demo-readiness|lighthouse|landgrab) · /onboard next operator (aliases operator-matrix|ops-matrix|operator-readiness|ops-readiness|matrix) · /onboard next human-gates (aliases human|gates|apply-gates) · /onboard next · /onboard status · /integrations status · /plugins dogfood · /plugins status · /mesh
  export receipt: /onboard next export — offline markdown evidence of this board (board/export evidence ≠ invent Connected)
  three product planes: /onboard next planes — mesh · memory-pull · agentic residual-honest consolidate · streams_not_probed · pull_not_probed · list_plan_not_connected · dual_auth_candidacy_open · never invent Connected (s1432)
  sales/buyer claims: /onboard next sales (aliases claims|buyer|claim-matrix|sales-claims|buyer-claims) — may claim / must not claim residual-honest · three-planes grounded · never invent Connected / Memory GA (s1437)
  demo readiness: /onboard next demo (aliases demo-ready|readiness|demo-readiness|lighthouse|landgrab) — Lighthouse beachhead · book-demo OFF · Landgrab NOT READY · human gates still open (s1442)
  operator readiness matrix: /onboard next operator (aliases operator-matrix|ops-matrix|operator-readiness|ops-readiness|matrix) — demo · sales · planes · human-gates · dual-auth candidacy · policy locks residual-honest (s1447)
  setup lifecycle map: /onboard next setup — P1–P7 closeout residual · stage 4 of edge-user-journey · setup_not_probed · package wire ≠ Connected · repair apply ≠ invent Connected · E10 Open (s1542+s1558)
  edge-user-journey first-run: /onboard next journey — 7 stages residual-honest · dual_write OFF · Edge Memory GA candidacy only · free eng s1558 (s1558 Wave B)
  first-run wizard residual: /onboard next wizard — Wave C guided residual map + soft dogfood · free eng s1570 · residual PASS ≠ invent full interactive auto wizard
  deeper tool-call residual: /onboard next tool-call — stage 6/7 depth after E4 attach · soft offline residual · free eng s1578 · free-floor peer s1580+ mention only · soft: /onboard next tool-call dogfood
  human-gates: /onboard next human-gates — still-human APPLY residual · PASS ≠ invent human-gate green · PASS ≠ live APPLY · open boxes stay open (s1413+s1546+s1550+s1574 Wave C continuum) · soft: /onboard next human-gates dogfood · free eng s1574
  plugins soft offline: /plugins dogfood (aliases soft|samples|offline) — soft offline ≠ live dogfood · ≠ invent Agent Plugins GA · session soft refreshes this board
  agentic soft offline: /onboard next agentic dogfood (aliases soft|samples|offline|list-plan-soft) — soft offline list/plan ≠ live dogfood · ≠ invent Connected · session soft refreshes agentic lane
  agentic dual-auth candidacy: /onboard next agentic dual-auth (aliases candidacy|list-org|org-installs|dual_auth|dual-auth-candidacy) — dual_auth_candidacy_open · list_org_unavailable · tool ship ≠ dual-auth live (s1427)
  tool-call soft offline: /onboard next tool-call dogfood (aliases soft|samples|offline|tool-call-soft) — soft offline ≠ invent Edge Memory GA declared · ≠ live dogfood · free eng s1578 · free-floor peer s1580+ mention only
  still-human APPLY soft offline: /onboard next human-gates dogfood (aliases soft|samples|offline|still-human-soft|apply-soft) — soft offline ≠ invent human-gate green · ≠ live APPLY · open boxes stay open · free eng s1574 · free-floor peer s1576+ mention only

Locks: dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · PASS ≠ live APPLY · PASS ≠ invent human-gate green · open boxes stay open · never invent install green / Connected / INSTALL_STORE APPLY · catalog ≠ Connected · portal HITL · agent MCP cannot write installs · plugins dogfood ≠ invent Agent Plugins GA · session soft ≠ live dogfood · soft offline ≠ live dogfood · drafts only · no auto-send · package load ≠ Memory GA · rates ~$88/$119 optional · board/export evidence ≠ invent Connected · mesh = streaming org heartbeats · mesh ≠ memory · never invent stream green · streams_not_probed · not OTel/APM · pull_not_probed · pull ≠ freemium hosted palace · Ops Pack ≠ GPU fleet · never invent pull green · list_plan_not_connected · plan deep links = browser HITL only · template= ≠ install APPLY · dual_auth_candidacy_open · list_org_unavailable · tool ship ≠ dual-auth live · never invent dual-auth live · setup_not_probed · package wire ≠ Connected · repair apply ≠ invent Connected · dual_write never auto ON · E10 Open · setup closeout residual ≠ invent Edge Memory GA · Knowledge Beta→GA cannot invent H1/H2 offline · leave ON_SIGNAL unset`, samplesState, dogfoodState, agenticSoft))
}

// AionAgentOnboardingNextLaneStatusExport residual-honest markdown status export receipt
// for /onboard next export (aliases receipt|stamp|evidence) — free eng s1387 (+ s1397 session soft · s1402 mesh · s1407 memory-pull · s1417 agentic · s1422 agentic list/plan soft).
// Offline evidence of the s1382 lane status board vocabulary; plugins dogfood lane reflects
// session soft marker from /plugins dogfood when present (default dogfood_not_run).
// Mesh lane: path_ready · residual_only · streams_not_probed (never invent stream green).
// Memory-pull lane: path_ready · residual_only · pull_not_probed (never invent pull green).
// Agentic lane (s1417+s1422): path_ready · residual_only · portal_hitl_still · list_plan_not_connected · <soft label>.
// Does NOT run plugins/agentic dogfood itself, dial MCP, or invent install green / Connected / GA / APPLY.
// Header pins evidence_kind=onboard_next_lane_status_export · offline_static ·
// not_live_dogfood · serial s1387.
// board/export evidence ≠ invent Connected · session soft ≠ live dogfood.
func AionAgentOnboardingNextLaneStatusExport() string {
	samplesState := nextLanePluginsSamplesSoftState()
	dogfoodState := nextLanePluginsDogfoodSessionState()
	agenticSoft := AgenticListPlanSoftSessionLabel()
	return strings.TrimSpace(fmt.Sprintf(`# aion onboard next lane status export receipt

evidence_kind=onboard_next_lane_status_export
offline_static=true
not_live_dogfood=true
serial=s1387
format=markdown

## board (honest vocabulary only — s1382+s1397+s1402+s1407+s1417+s1422 reuse · never invent connected / ga / apply / stream green / pull green as success)

plugins: %s · %s · path_ready
  · offline samples soft-check only (examples/agent-plugins) · ≠ invent Agent Plugins GA
  · session soft marker ≠ live dogfood · soft offline dogfood ≠ invent Agent Plugins GA
  · residual PASS ≠ live dogfood · never invent install green / Connected / INSTALL_STORE APPLY
  · board/export evidence ≠ invent Connected · drill: /onboard next plugins (aliases plugin|dogfood)

gtm: skill_ready · path_ready · residual_only
  · /gtm checklist + skill gtm-draft-only-agent path ready
  · drafts only · no auto-send · human publish · GTM checklist ≠ invent GTM agent GA
  · drill: /onboard next gtm (alias drafts)

memory: path_ready · residual_only
  · dual_write OFF · local-primary · package load ≠ Memory GA · ≠ freemium palace · Palace sunset
  · not Memory GA · book-demo OFF · rates ~$88/$119 optional · mesh ≠ memory · mesh optional for pull
  · edge OSS tip (s1453+s1458+s1463+s1469+s1478+s1508): iomesh-memory-mcp product host · public product attach · go install · no GOPRIVATE · HTTP 8080/mcp or stdio · docker compose still valid · dual_write OFF · not Memory GA · aion broker private · flip complete residual ≠ invent Memory GA · public OSS ≠ invent platform GA · PASS ≠ invent full platform sidecar parity · offline dogfood tip ≠ invent live dogfood as green · E4 client attach (s1508) · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · E10 Open · tip ≠ invent forever-green product dogfood
  · drill: /onboard next memory (aliases mcp|palace)

mesh: path_ready · residual_only · streams_not_probed
  · I/O Mesh = streaming org heartbeats on dept.* · not OTel/APM · not hosted Memory Palace
  · mesh ≠ memory · empty streams honest · never invent stream green / Connected
  · pull = mesh → local palace egress · dual_write OFF · rates ~$88 mesh / ~$119 Memory Ops Pack optional
  · drill: /onboard next mesh (aliases stream|streams|heartbeat|heartbeats|pull)

memory-pull: path_ready · residual_only · pull_not_probed
  · Ops Pack pull path · iomesh memory pull = mesh → local palace egress · dual_write OFF
  · pull ≠ freemium hosted palace · Ops Pack ≠ GPU fleet · never invent pull green
  · package load ≠ Ops Pack entitlement · package load ≠ Memory GA · Palace sunset
  · rates ~$88 mesh / ~$119 Memory Ops Pack optional · residual PASS ≠ live dogfood · PASS ≠ live APPLY
  · drill: /onboard next memory-pull (aliases ops-pack|pull-path|memorypull|ops_pack) · bare pull stays mesh

setup: path_ready · residual_only · setup_not_probed
  · setup lifecycle P1–P7 closeout residual (s1542) · offline static map · dual_write OFF · not Memory GA
  · init → preflight → reload → portal HITL → pull → analyze → drift → repair plan/apply --yes
  · package wire ≠ Connected · repair apply ≠ invent Connected · dual_write never auto ON · E10 Open
  · setup closeout residual ≠ invent Edge Memory GA · offline static lane ≠ live dogfood
  · drill: /onboard next setup (aliases setup-lifecycle|lifecycle|setup_lifecycle) · guided residual: /onboard next wizard

agentic: path_ready · residual_only · portal_hitl_still · list_plan_not_connected · dual_auth_candidacy_open · list_org_unavailable · %s
  · product plane 3 agentic integrations · MCP list/plan residual-honest · never invent Connected
  · plan_connector_setup → portal deep links · browser HITL only · template= ≠ install APPLY green
  · list_org fail-open ≠ empty-as-none · agent MCP cannot write installs · catalog ≠ Connected
  · portal HITL @ https://console.iome.sh/integrations · Agent/MCP @ https://console.iome.sh/settings/agent
  · session soft list/plan ≠ live dogfood · soft offline ≠ invent Connected · portal HITL still
  · dual_auth_candidacy_open · list_org_unavailable · tool ship ≠ dual-auth live · never invent dual-auth live
  · drill: /onboard next agentic (aliases agentic-integrations|integrations|list-plan)
  · soft dogfood: /onboard next agentic dogfood (aliases soft|samples|offline|list-plan-soft)
  · dual-auth depth: /onboard next agentic dual-auth (aliases candidacy|list-org|org-installs|dual_auth|dual-auth-candidacy)

portal: portal_hitl_still
  · agent MCP cannot write installs · catalog ≠ Connected · portal HITL still for OAuth/install
  · list_org fail-open ≠ empty-as-none · never invent Connected / INSTALL_STORE APPLY
  · Agent/MCP mint/copy/probe @ https://console.iome.sh/settings/agent · connectors @ https://console.iome.sh/integrations

## honesty

- dual_write OFF · book-demo OFF · not Memory GA
- residual PASS ≠ live dogfood · never invent install green / Connected / INSTALL_STORE APPLY
- catalog ≠ Connected · portal HITL · agent MCP cannot write installs
- plugins dogfood ≠ invent Agent Plugins GA · session soft ≠ live dogfood · drafts only · no auto-send
- GTM checklist ≠ invent GTM agent GA · package load ≠ Memory GA
- board/export evidence ≠ invent Connected · rates ~$88/$119 optional
- mesh = streaming org heartbeats · mesh ≠ memory · never invent stream green · streams_not_probed · not OTel/APM
- pull_not_probed · pull ≠ freemium hosted palace · Ops Pack ≠ GPU fleet · never invent pull green
- list_plan_not_connected · plan deep links = browser HITL only · template= ≠ install APPLY
- soft offline list/plan ≠ live dogfood · ≠ invent Connected · portal HITL still
- dual_auth_candidacy_open · list_org_unavailable · tool ship ≠ dual-auth live · never invent dual-auth live
- setup_not_probed · package wire ≠ Connected · repair apply ≠ invent Connected · dual_write never auto ON · E10 Open
- setup closeout residual ≠ invent Edge Memory GA · offline static lane ≠ live dogfood
- this receipt does NOT run plugins dogfood · does NOT run agentic list/plan dogfood · does NOT dial MCP · does NOT invent green
- session soft marker (if present) ≠ live dogfood · ≠ invent Agent Plugins GA · board ≠ invent Connected

## slash

/onboard next export (aliases receipt|stamp|evidence) · optional /onboard next export json
/onboard next status (aliases pulse|board) · /onboard next mesh · /onboard next memory-pull · /onboard next setup · /onboard next agentic · /onboard next agentic dogfood · /onboard next agentic dual-auth · /onboard next human-gates · /onboard next · /onboard status
/onboard next setup (aliases setup-lifecycle|lifecycle|setup_lifecycle) — P1–P7 closeout residual · setup_not_probed · package wire ≠ Connected · repair apply ≠ invent Connected · E10 Open (s1542)
/onboard next wizard (aliases first-run-wizard|guided|wave-c|wave_c|wizard-residual) — Wave C first-run wizard residual · free eng s1570 (s1570)
/onboard next agentic (aliases agentic-integrations|integrations|list-plan) — product plane 3 MCP list/plan + portal HITL · list_plan_not_connected · never invent Connected (s1417)
/onboard next agentic dogfood (aliases soft|samples|offline|list-plan-soft) — soft offline list/plan residual · session soft ≠ live dogfood · ≠ invent Connected (s1422)
/onboard next agentic dual-auth (aliases candidacy|list-org|org-installs|dual_auth|dual-auth-candidacy) — dual_auth_candidacy_open · list_org_unavailable · tool ship ≠ dual-auth live (s1427)
/onboard next human-gates (aliases human|gates|apply-gates) — still human vs offline residual · PASS ≠ invent human-gate green · PASS ≠ live APPLY (s1413)
/plugins dogfood (aliases soft|samples|offline) · /plugins status — soft offline ≠ live dogfood · ≠ invent Agent Plugins GA

Locks: dual_write OFF · book-demo OFF · not Memory GA · residual PASS ≠ live dogfood · PASS ≠ live APPLY · PASS ≠ invent human-gate green · open boxes stay open · never invent install green / Connected / INSTALL_STORE APPLY · catalog ≠ Connected · portal HITL · agent MCP cannot write installs · plugins dogfood ≠ invent Agent Plugins GA · session soft ≠ live dogfood · soft offline ≠ live dogfood · drafts only · no auto-send · package load ≠ Memory GA · board/export evidence ≠ invent Connected · rates ~$88/$119 optional · mesh = streaming org heartbeats · mesh ≠ memory · never invent stream green · streams_not_probed · pull_not_probed · pull ≠ freemium hosted palace · Ops Pack ≠ GPU fleet · never invent pull green · list_plan_not_connected · plan deep links = browser HITL only · template= ≠ install APPLY · dual_auth_candidacy_open · list_org_unavailable · tool ship ≠ dual-auth live · never invent dual-auth live · setup_not_probed · package wire ≠ Connected · repair apply ≠ invent Connected · dual_write never auto ON · E10 Open · setup closeout residual ≠ invent Edge Memory GA · Knowledge Beta→GA cannot invent H1/H2 offline · leave ON_SIGNAL unset`, samplesState, dogfoodState, agenticSoft))
}

// nextLaneStatusExportDTO is the offline JSON shape for AionAgentOnboardingNextLaneStatusExportJSON (s1387+s1397+s1422).
// Honest vocabulary only — never invents Connected/GA/APPLY success.
// Plugins dogfood session state (s1397): dogfood_not_run default; soft_offline_dogfood_session_pass|fail after /plugins dogfood.
// Agentic list/plan soft state (s1422): list_plan_soft_not_run default; soft_offline_list_plan_session_pass|fail after /onboard next agentic dogfood.
type nextLaneStatusExportDTO struct {
	EvidenceKind             string            `json:"evidence_kind"`
	OfflineStatic            bool              `json:"offline_static"`
	NotLiveDogfood           bool              `json:"not_live_dogfood"`
	Serial                   string            `json:"serial"`
	Format                   string            `json:"format"`
	Lanes                    map[string]string `json:"lanes"`
	SamplesState             string            `json:"samples_state"`
	PluginsDogfoodState      string            `json:"plugins_dogfood_state"`
	DogfoodNotRun            bool              `json:"dogfood_not_run"`
	AgenticListPlanSoftState string            `json:"agentic_list_plan_soft_state"`
	HonestyLocks             []string          `json:"honesty_locks"`
	Slash                    []string          `json:"slash"`
	Note                     string            `json:"note"`
}

// AionAgentOnboardingNextLaneStatusExportJSON residual-honest JSON status export receipt
// for /onboard next export json (s1387+s1397+s1402+s1407+s1413+s1417+s1422). Same honesty as markdown; offline only.
// Reflects session soft dogfood marker when set by /plugins dogfood (≠ live dogfood).
// Reflects agentic list/plan soft marker when set by /onboard next agentic dogfood (≠ invent Connected).
// Mesh lane: path_ready · residual_only · streams_not_probed (never invent stream green).
// Memory-pull / ops_pack lane: path_ready · residual_only · pull_not_probed (never invent pull green).
// Agentic lane (s1417+s1422): path_ready · residual_only · portal_hitl_still · list_plan_not_connected · <soft label>.
// s1413: slash tip + honesty locks for human-gates board (still ≠ invent human-gate green / live APPLY).
// Does NOT run plugins/agentic dogfood itself, dial MCP, or invent install green / Connected / GA / APPLY.
func AionAgentOnboardingNextLaneStatusExportJSON() string {
	samplesState := nextLanePluginsSamplesSoftState()
	dogfoodState := nextLanePluginsDogfoodSessionState()
	agenticSoft := AgenticListPlanSoftSessionLabel()
	ran, _ := agentplugins.GetSoftDogfoodSessionState()
	dto := nextLaneStatusExportDTO{
		EvidenceKind:   "onboard_next_lane_status_export",
		OfflineStatic:  true,
		NotLiveDogfood: true,
		Serial:         "s1387",
		Format:         "json",
		Lanes: map[string]string{
			"plugins":     fmt.Sprintf("%s · %s · path_ready", samplesState, dogfoodState),
			"gtm":         "skill_ready · path_ready · residual_only",
			"memory":      "path_ready · residual_only",
			"mesh":        "path_ready · residual_only · streams_not_probed",
			"memory-pull": "path_ready · residual_only · pull_not_probed",
			"ops_pack":    "path_ready · residual_only · pull_not_probed",
			"setup":       "path_ready · residual_only · setup_not_probed",
			"agentic":     fmt.Sprintf("path_ready · residual_only · portal_hitl_still · list_plan_not_connected · %s", agenticSoft),
			"portal":      "portal_hitl_still",
		},
		SamplesState:        samplesState,
		PluginsDogfoodState: dogfoodState,
		// dogfood_not_run true only when session soft dogfood has not run (s1397).
		// When ran, still not_live_dogfood=true — session soft ≠ live dogfood.
		DogfoodNotRun:            !ran,
		AgenticListPlanSoftState: agenticSoft,
		HonestyLocks: []string{
			"dual_write OFF",
			"book-demo OFF",
			"not Memory GA",
			"residual PASS ≠ live dogfood",
			"session soft ≠ live dogfood",
			"soft offline list/plan ≠ live dogfood",
			"soft offline ≠ invent Connected",
			"never invent install green / Connected / INSTALL_STORE APPLY",
			"catalog ≠ Connected",
			"portal HITL",
			"agent MCP cannot write installs",
			"plugins dogfood ≠ invent Agent Plugins GA",
			"soft offline dogfood ≠ invent Agent Plugins GA",
			"drafts only",
			"no auto-send",
			"GTM checklist ≠ invent GTM agent GA",
			"package load ≠ Memory GA",
			"board/export evidence ≠ invent Connected",
			"rates ~$88/$119 optional",
			"mesh = streaming org heartbeats",
			"mesh ≠ memory",
			"never invent stream green / Connected",
			"streams_not_probed",
			"not OTel/APM",
			"pull_not_probed",
			"pull ≠ freemium hosted palace",
			"Ops Pack ≠ GPU fleet",
			"never invent pull green",
			"package load ≠ Ops Pack entitlement",
			"setup_not_probed",
			"package wire ≠ Connected",
			"repair apply ≠ invent Connected",
			"dual_write never auto ON",
			"E10 Open",
			"setup closeout residual ≠ invent Edge Memory GA",
			"list_plan_not_connected",
			"plan deep links = browser HITL only",
			"template= ≠ install APPLY",
			"PASS ≠ invent human-gate green",
			"PASS ≠ live APPLY",
			"open boxes stay open",
			"Knowledge Beta→GA cannot invent H1/H2 offline",
			"leave ON_SIGNAL unset",
		},
		Slash: []string{
			"/onboard next export",
			"/onboard next export json",
			"/onboard next status",
			"/onboard next mesh",
			"/onboard next memory-pull",
			"/onboard next setup",
			"/onboard next agentic",
			"/onboard next agentic dogfood",
			"/onboard next human-gates",
			"/plugins dogfood",
			"/plugins status",
		},
		Note: "offline residual-honest lane board evidence; plugins_dogfood_state is session soft marker only (default dogfood_not_run); agentic_list_plan_soft_state is independent session soft for list/plan (default list_plan_soft_not_run · s1422); mesh lane is path_ready · residual_only · streams_not_probed (never invent stream green); memory-pull/ops_pack lane is path_ready · residual_only · pull_not_probed (never invent pull green); setup lane is path_ready · residual_only · setup_not_probed (s1542 · never invent Connected · package wire ≠ Connected · repair apply ≠ invent Connected · E10 Open); agentic lane is path_ready · residual_only · portal_hitl_still · list_plan_not_connected · soft label (never invent Connected · product plane 3 MCP list/plan + portal HITL · s1417+s1422); session soft ≠ live dogfood; soft offline list/plan ≠ invent Connected; board/export evidence ≠ invent Connected; does NOT run plugins dogfood or agentic list/plan dogfood or dial MCP; soft offline dogfood ≠ invent Agent Plugins GA; mesh ≠ memory; pull ≠ freemium hosted palace; Ops Pack ≠ GPU fleet; human-gates: PASS ≠ invent human-gate green · PASS ≠ live APPLY · open boxes stay open (s1413 tip only)",
	}
	b, err := json.MarshalIndent(dto, "", "  ")
	if err != nil {
		// Fail-open residual: never invent green; return a static honesty stub.
		return `{"evidence_kind":"onboard_next_lane_status_export","offline_static":true,"not_live_dogfood":true,"serial":"s1387","error":"marshal_failed","note":"board/export evidence ≠ invent Connected"}`
	}
	return string(b)
}

// nextLanePluginsSamplesSoftState soft-checks in-repo sample package dirs (s1382/s1387/s1392).
// samples_ok when both hello-iome + iomesh-memory-mcp dirs exist under module root;
// samples_missing otherwise (including when module root is not found).
// Does not run dogfood, Dial MCP, or invent Agent Plugins GA / Connected.
func nextLanePluginsSamplesSoftState() string {
	return agentplugins.SamplesSoftState("")
}

// nextLanePluginsDogfoodSessionState returns session soft dogfood vocabulary (s1397).
// Default dogfood_not_run; after /plugins dogfood: soft_offline_dogfood_session_pass|fail.
// Session soft ≠ live dogfood · ≠ invent Agent Plugins GA · board ≠ invent Connected.
func nextLanePluginsDogfoodSessionState() string {
	return agentplugins.SoftDogfoodSessionLabel()
}

// AionAgentHumanGatesHonestyBoard residual-honest human-gates status section for operators
// (/onboard next human-gates · aliases human|gates|apply-gates) — free eng s1413 + s1546 + s1550 edge-first.
// Static offline — no MCP dial · no gcloud · never invents human-gate green or live APPLY.
// s1550 edge-first pin (founders): local TUI + iomesh-memory-mcp + optional mesh pull · dual_write OFF ·
// knowledge multi-tenant INSTALL_STORE punted (H1/H2 not launch) · Slack HMAC punted for now ·
// integrations path = TUI agent MCP list/plan + portal HITL when OAuth needed · Stripe residual closed
// unless ACL regresses · E10 only if claiming Edge Memory GA · book-demo OFF.
// s1546: setup residual complete ≠ invent human-gate green / live APPLY / OAuth Connected / E10.
// Honesty locks: dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only ·
// residual PASS ≠ invent Edge Memory GA · PASS ≠ invent Connected · agent MCP cannot write installs ·
// portal HITL when connect · knowledge multi-tenant punted · Slack HMAC punted · H1/H2 not launch gate ·
// open policy boxes stay honest.
func AionAgentHumanGatesHonestyBoard() string {
	softLabel := StillHumanSoftSessionLabel()
	return strings.TrimSpace(fmt.Sprintf(`aion human-gates honesty board (residual-honest · s1550 edge-first · s1574 Wave C continuum still-human APPLY residual · no MCP dial):

  architecture (locked):
    · edge-first local memory · dual_write OFF · hosted palace sunset
    · knowledge multi-tenant INSTALL_STORE punted · H1/H2 not launch gate
    · Slack HMAC punted for now · no live Slack signed-webhook requirement for this residual
    · integrations path: TUI agent MCP list/plan + portal HITL (agent cannot write installs)

  still_human_or_policy (edge-first launch residual · still-human APPLY open inventory):
    · still-human APPLY boxes stay open after Wave A–C continuum — PASS ≠ invent human-gate green · PASS ≠ live APPLY · open boxes stay open
    · portal HITL OAuth/connect when customer must Connect — agent MCP cannot write installs · catalog ≠ Connected · portal HITL when connect
    · book-demo OFF · leave ON_SIGNAL unset
    · dual_write OFF · not Memory GA · Edge Memory GA candidacy only · E10 Open
    · residual PASS ≠ invent Edge Memory GA declared · residual PASS ≠ invent Edge Memory GA
    · E10 founder sign-off only if claiming Edge Memory GA (candidacy allowed without E10)
    · optional E4 edge dogfood if tightening Edge Memory claim beyond candidacy

  open inventory residual-honest (s1574 Wave C continuum reaffirm — never invent green):
    · Slack HMAC — punted for launch / still open as policy · residual PASS ≠ invent Slack Connected green
    · Stripe Customers:Write residual — key material largely closed · ACL residual only if Dashboard regresses · still residual-honest open policy
    · H1/H2 knowledge INSTALL_STORE — punted / not launch gate with knowledge multi-tenant
    · OAuth Connected still portal HITL when connect · agent MCP cannot write installs · catalog ≠ Connected
    · book-demo OFF · dual_write OFF · E10 Open

  punted_or_demoted (not launch peer for edge-first):
    · Slack HMAC rotate — punted for now · residual PASS ≠ invent Slack Connected green
    · H1/H2 knowledge INSTALL_STORE — punted with knowledge multi-tenant · gcloud ops residual only if un-punt later
    · knowledge multi-tenant punted · knowledge D1–D5 / exit-Beta — deferred with multi-tenant knowledge punt
    · Stripe key material + Customers/Checkout Write — validated residual (SM update) · ACL residual only if Dashboard regresses

  offline_residual_only / shipped_or_policy:
    · setup lifecycle residual complete ≠ invent Connected / Memory GA
    · Wave A–C continuum residual (journey · wizard · portal-hitl · e4 soft) ≠ invent human-gate green / live APPLY / Edge Memory GA declared
    · agent MCP list/plan residual-honest · dual_write OFF · not Memory GA · Edge Memory GA candidacy only
    · local memory / dual_write OFF do NOT invent Edge Memory GA declared
    · residual PASS ≠ invent Edge Memory GA · residual PASS ≠ invent Edge Memory GA declared · PASS ≠ invent Connected · open policy boxes stay honest
    · PASS ≠ invent human-gate green · PASS ≠ live APPLY · open boxes stay open

  Soft offline still-human APPLY dogfood (s1574 · session soft ≠ live dogfood):
    · session soft state: %s (default still_human_soft_not_run · after run soft_offline_still_human_session_pass|fail)
    · slash: /onboard next human-gates dogfood (aliases soft|samples|offline|still-human-soft|apply-soft) · bare /onboard next human-gates stays this board
    · soft offline ≠ invent Connected · residual PASS ≠ live dogfood · session soft ≠ live dogfood · residual PASS ≠ invent Edge Memory GA declared
    · PASS ≠ invent human-gate green · PASS ≠ live APPLY · open boxes stay open · free eng s1574 · free-floor peer s1576+ mention only

  operator:
    · slash: /onboard next human-gates · /onboard next human-gates dogfood · /onboard next e10 · /onboard next e10 dogfood · /onboard next setup · /integrations list|plan|status
    · companion Wave C: /onboard next wizard · /onboard next journey · /onboard next portal-hitl · /onboard next e4
    · companion E10 Open reaffirm (s1586): /onboard next e10 · residual PASS ≠ invent E10 closed · residual PASS ≠ invent Edge Memory GA declared · residual-check
    · companion aion ops residual: edge-first human gates residual (s1550) · still-human APPLY soft residual (s1574 Wave C continuum)
    · never invent Connected / INSTALL_STORE green / Edge Memory GA / book-demo as ON
    · aliases human|gates|apply-gates|still-human|apply-residual · companion /onboard next · /onboard status · /setup portal

Locks: dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA · residual PASS ≠ invent Edge Memory GA declared · E10 Open · residual PASS ≠ invent E10 closed · PASS ≠ invent Connected · PASS ≠ invent human-gate green · PASS ≠ live APPLY · open boxes stay open · agent MCP cannot write installs · portal HITL when connect · knowledge multi-tenant punted · Slack HMAC punted · H1/H2 not launch gate · open policy boxes stay honest · soft offline ≠ invent Connected · session soft ≠ live dogfood · free eng s1574 · free-floor peer s1576+ mention only · Wave C continuum · companion E10 Open reaffirm s1586 · %s`, softLabel, softLabel))
}
