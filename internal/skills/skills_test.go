package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParse_Frontmatter(t *testing.T) {
	raw := `---
name: check-work
description: >
  Verify changes with a subagent
metadata:
  short-description: "x"
---

# Body

Do the thing.
`
	sk, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if sk.Name != "check-work" {
		t.Fatalf("name=%q", sk.Name)
	}
	if !strings.Contains(sk.Description, "Verify") {
		t.Fatalf("desc=%q", sk.Description)
	}
	if !strings.Contains(sk.Body, "Do the thing") {
		t.Fatalf("body=%q", sk.Body)
	}
}

func TestLoadDirs_SkillMDLayout(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "help")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: help\ndescription: Help skill\n---\n\n# Help\n\nBe helpful.\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	// second skill
	dir2 := filepath.Join(root, "review")
	_ = os.MkdirAll(dir2, 0o755)
	_ = os.WriteFile(filepath.Join(dir2, "SKILL.md"), []byte("---\nname: review\ndescription: Review code\n---\n\nReview well.\n"), 0o644)

	cat, err := LoadDirs(root)
	if err != nil {
		t.Fatal(err)
	}
	if cat.Len() != 2 {
		t.Fatalf("len=%d names=%v", cat.Len(), cat.Names())
	}
	sk, ok := cat.Get("help")
	if !ok || !strings.Contains(sk.Body, "helpful") {
		t.Fatalf("%+v ok=%v", sk, ok)
	}
	block := cat.PromptBlock()
	if !strings.Contains(block, "help") || !strings.Contains(block, "review") {
		t.Fatal(block)
	}
}

func TestLoadDirs_MissingOK(t *testing.T) {
	cat, err := LoadDirs(filepath.Join(t.TempDir(), "nope"))
	if err != nil {
		t.Fatal(err)
	}
	if cat.Len() != 0 {
		t.Fatal(cat.Len())
	}
}

func TestSanitizeAndDefaultDirs(t *testing.T) {
	if sanitizeName("Hello World!") != "hello-world" {
		t.Fatal(sanitizeName("Hello World!"))
	}
	dirs := DefaultDirs(t.TempDir())
	if len(dirs) < 1 {
		t.Fatal(dirs)
	}
}

// s1251: builtin connector-integrations-setup always loads via go:embed.
func TestLoadBuiltin_ConnectorIntegrationsSetup(t *testing.T) {
	cat, err := LoadBuiltin()
	if err != nil {
		t.Fatal(err)
	}
	if cat.Len() == 0 {
		t.Fatal("builtin catalog empty")
	}
	sk, ok := cat.Get("connector-integrations-setup")
	if !ok {
		t.Fatalf("missing connector-integrations-setup; names=%v", cat.Names())
	}
	if sk.Name != "connector-integrations-setup" {
		t.Fatalf("name=%q", sk.Name)
	}
	if strings.TrimSpace(sk.Description) == "" {
		t.Fatal("description empty")
	}
	// Body honesty needles
	for _, want := range []string{
		"list_connector_catalog",
		"plan_connector_setup",
		"portal HITL",
		"never invent install green",
	} {
		if !strings.Contains(sk.Body, want) && !strings.Contains(sk.Description, want) {
			// Body preferred; description may not have all needles
			if !strings.Contains(sk.Body, want) {
				t.Fatalf("body missing %q:\n%s", want, sk.Body)
			}
		}
	}
	// Extra residual locks in body
	for _, want := range []string{
		"browser HITL",
		"dual_write OFF",
		"book-demo OFF",
		"get_webhook_signing_headers",
	} {
		if !strings.Contains(sk.Body, want) {
			t.Fatalf("body missing %q", want)
		}
	}
	// Source marks as builtin
	if !strings.Contains(sk.Path, "builtin") && sk.SourceDir != "builtin" {
		t.Fatalf("path/source not builtin: path=%q source=%q", sk.Path, sk.SourceDir)
	}
}

// s1251: LoadWithBuiltin always includes builtin even when dirs empty/missing.
func TestLoadWithBuiltin_EmptyDirsStillBuiltin(t *testing.T) {
	cat, err := LoadWithBuiltin(filepath.Join(t.TempDir(), "nope"))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cat.Get("connector-integrations-setup"); !ok {
		t.Fatalf("builtin missing when dirs empty; names=%v", cat.Names())
	}
}

// s1251: user skill overrides builtin on name collision; new names merge.
func TestLoadWithBuiltin_UserOverrides(t *testing.T) {
	root := t.TempDir()
	// Override builtin name with a stub user skill
	dir := filepath.Join(root, "connector-integrations-setup")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: connector-integrations-setup\ndescription: User override\n---\n\n# Override\n\nUser body.\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	// Extra user skill
	dir2 := filepath.Join(root, "my-skill")
	_ = os.MkdirAll(dir2, 0o755)
	_ = os.WriteFile(filepath.Join(dir2, "SKILL.md"), []byte("---\nname: my-skill\ndescription: Custom\n---\n\nCustom body.\n"), 0o644)

	cat, err := LoadWithBuiltin(root)
	if err != nil {
		t.Fatal(err)
	}
	sk, ok := cat.Get("connector-integrations-setup")
	if !ok {
		t.Fatal("missing skill")
	}
	if !strings.Contains(sk.Body, "User body") {
		t.Fatalf("want user override body: %s", sk.Body)
	}
	if sk.Description != "User override" {
		t.Fatalf("desc=%q", sk.Description)
	}
	if _, ok := cat.Get("my-skill"); !ok {
		t.Fatalf("my-skill missing: %v", cat.Names())
	}
}

func TestCatalogMerge_NilSafe(t *testing.T) {
	var c *Catalog
	out := c.Merge(nil)
	if out == nil || out.Len() != 0 {
		t.Fatalf("%+v", out)
	}
	builtin, err := LoadBuiltin()
	if err != nil {
		t.Fatal(err)
	}
	out2 := (*Catalog)(nil).Merge(builtin)
	if out2.Len() != builtin.Len() {
		t.Fatalf("merge from nil: %d vs %d", out2.Len(), builtin.Len())
	}
}

// --- s1257: builtin skill dogfood (residual-honest connector setup) ---

// TestLoadWithBuiltin_S1257ConnectorSkillDogfood proves LoadWithBuiltin always
// surfaces connector-integrations-setup with residual-honest body + description.
func TestLoadWithBuiltin_S1257ConnectorSkillDogfood(t *testing.T) {
	cat, err := LoadWithBuiltin() // no dirs — pure builtin
	if err != nil {
		t.Fatal(err)
	}
	sk, ok := cat.Get("connector-integrations-setup")
	if !ok {
		t.Fatalf("skill missing; names=%v", cat.Names())
	}
	if sk.Name != "connector-integrations-setup" {
		t.Fatalf("name=%q", sk.Name)
	}
	// Description residual-honest (frontmatter).
	desc := strings.ToLower(sk.Description)
	for _, want := range []string{
		"residual",
		"portal",
		"hitl",
	} {
		if !strings.Contains(desc, want) {
			t.Fatalf("description missing residual needle %q: %q", want, sk.Description)
		}
	}
	// Description must not invent install APPLY / Connected success claims.
	if strings.Contains(sk.Description, "Connected: yes") || strings.Contains(desc, "install apply green") {
		t.Fatalf("description invents install green: %q", sk.Description)
	}
	// Body must mention core MCP workflow + honesty locks (s1257 · s1273 org installs).
	body := sk.Body
	for _, want := range []string{
		"list_connector_catalog",
		"plan_connector_setup",
		"get_webhook_signing_headers",
		"list_org_connector_installs",
		"never invent empty-as-none",
		"portal HITL",
		"browser HITL",
		"never invent install green",
		"dual_write OFF",
		"book-demo OFF",
		"portal_add_url",
		"deep_links",
		"stub",
		"INSTALL_STORE",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("skill body missing %q:\n%s", want, body)
		}
	}
	// Must not invent install success language as a claim.
	if strings.Contains(body, "Connected: yes") {
		t.Fatalf("body invents install green claim")
	}
	// Builtin source marker.
	if sk.SourceDir != "builtin" && !strings.Contains(sk.Path, "builtin") {
		t.Fatalf("source not builtin: path=%q source=%q", sk.Path, sk.SourceDir)
	}
}

// TestS1257SkillDescriptionResidualHonest pins description frontmatter honesty.
func TestS1257SkillDescriptionResidualHonest(t *testing.T) {
	cat, err := LoadBuiltin()
	if err != nil {
		t.Fatal(err)
	}
	sk, ok := cat.Get("connector-integrations-setup")
	if !ok {
		t.Fatal("missing skill")
	}
	// Description says residual-honest + not install APPLY.
	if !strings.Contains(sk.Description, "Residual-honest") && !strings.Contains(sk.Description, "residual-honest") {
		t.Fatalf("description not residual-honest: %q", sk.Description)
	}
	if !strings.Contains(sk.Description, "portal") && !strings.Contains(sk.Description, "HITL") {
		t.Fatalf("description missing portal HITL: %q", sk.Description)
	}
	// Explicit non-install-APPLY framing.
	if !strings.Contains(sk.Description, "not install") && !strings.Contains(sk.Description, "not install APPLY") {
		// Accept either phrasing from SKILL.md frontmatter.
		if !strings.Contains(strings.ToLower(sk.Description), "not install") {
			t.Fatalf("description should say not install APPLY: %q", sk.Description)
		}
	}
}

// --- s1288: builtin memory-advanced-agent skill (residual-honest advanced memory) ---

// TestLoadBuiltin_MemoryAdvancedAgent proves go:embed loads memory-advanced-agent.
func TestLoadBuiltin_MemoryAdvancedAgent(t *testing.T) {
	cat, err := LoadBuiltin()
	if err != nil {
		t.Fatal(err)
	}
	sk, ok := cat.Get("memory-advanced-agent")
	if !ok {
		t.Fatalf("missing memory-advanced-agent; names=%v", cat.Names())
	}
	if sk.Name != "memory-advanced-agent" {
		t.Fatalf("name=%q", sk.Name)
	}
	if strings.TrimSpace(sk.Description) == "" {
		t.Fatal("description empty")
	}
	// Builtin source marker.
	if !strings.Contains(sk.Path, "builtin") && sk.SourceDir != "builtin" {
		t.Fatalf("path/source not builtin: path=%q source=%q", sk.Path, sk.SourceDir)
	}
}

// TestLoadWithBuiltin_MemoryAdvancedAgentAlwaysPresent: skill present even with empty dirs.
func TestLoadWithBuiltin_MemoryAdvancedAgentAlwaysPresent(t *testing.T) {
	cat, err := LoadWithBuiltin(filepath.Join(t.TempDir(), "nope"))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cat.Get("memory-advanced-agent"); !ok {
		t.Fatalf("memory-advanced-agent missing when dirs empty; names=%v", cat.Names())
	}
	// Still keep connector skill.
	if _, ok := cat.Get("connector-integrations-setup"); !ok {
		t.Fatalf("connector-integrations-setup missing; names=%v", cat.Names())
	}
}

// TestLoadBuiltin_S1288MemoryAdvancedSkillDogfood pins residual-honest body needles.
func TestLoadBuiltin_S1288MemoryAdvancedSkillDogfood(t *testing.T) {
	cat, err := LoadBuiltin()
	if err != nil {
		t.Fatal(err)
	}
	sk, ok := cat.Get("memory-advanced-agent")
	if !ok {
		t.Fatalf("skill missing; names=%v", cat.Names())
	}
	desc := strings.ToLower(sk.Description)
	for _, want := range []string{
		"residual",
		"memory",
	} {
		if !strings.Contains(desc, want) {
			t.Fatalf("description missing residual needle %q: %q", want, sk.Description)
		}
	}
	// Description must not invent Memory GA product green.
	if strings.Contains(desc, "memory ga green") || strings.Contains(sk.Description, "Memory GA product") {
		t.Fatalf("description invents Memory GA: %q", sk.Description)
	}
	body := sk.Body
	for _, want := range []string{
		"memory_related",
		"prefer_shorter_hops",
		"memory_supersede_entity",
		"--i-confirm",
		"memory_facts_as_of",
		"ops_digest_export",
		"memory_patterns_list",
		"memory_anomalies_list",
		"memory_timeline",         // s1296
		"memory_compact_status",   // s1296
		"/memory timeline",        // s1296 slash
		"/memory compact-status",  // s1296 slash
		"memory_search_semantic",  // s1301
		"memory_ingest_event",     // s1301
		"/memory semantic",        // s1301 slash
		"/memory ingest-event",    // s1301 slash
		"memory_trigger_compact",  // s1311 HITL
		"/memory trigger-compact", // s1311 slash
		"--i-confirm",
		"multi-hop lite",
		"graph RAG",
		"A3 lite",
		"HITL",
		"K4",
		"dual_write OFF",
		"not Memory GA",
		"no invent",
		"/memory related",
		"/memory supersede",
		"/memory facts-as-of",
		"/memory digest",
		"shipped s1287",                   // s1291 polish: not peer concurrent
		"MemoryAdvancedAgentGuidanceNote", // s1291 system note
		"s1291",
		"s1296",
		"s1301",
		"s1311",
		"not conversation turn",
		"tier-4 semantic",
		"not invent compaction green",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("skill body missing %q:\n%s", want, body)
		}
	}
	// Must not still claim s1287 as peer concurrent.
	if strings.Contains(body, "peer concurrent") {
		t.Fatalf("skill still says peer concurrent (s1291 marks s1287 shipped):\n%s", body)
	}
}

// TestS1288SkillDescriptionResidualHonest pins frontmatter honesty.
func TestS1288SkillDescriptionResidualHonest(t *testing.T) {
	cat, err := LoadBuiltin()
	if err != nil {
		t.Fatal(err)
	}
	sk, ok := cat.Get("memory-advanced-agent")
	if !ok {
		t.Fatal("missing skill")
	}
	if !strings.Contains(sk.Description, "Residual-honest") && !strings.Contains(sk.Description, "residual-honest") {
		t.Fatalf("description not residual-honest: %q", sk.Description)
	}
	if !strings.Contains(sk.Description, "not Memory GA") && !strings.Contains(strings.ToLower(sk.Description), "not memory ga") {
		t.Fatalf("description should say not Memory GA: %q", sk.Description)
	}
	if !strings.Contains(sk.Description, "dual_write OFF") && !strings.Contains(strings.ToLower(sk.Description), "dual_write") {
		t.Fatalf("description should mention dual_write OFF: %q", sk.Description)
	}
}

// --- s1341: builtin gtm-draft-only-agent skill (residual-honest GTM draft-only roles) ---

// TestLoadBuiltin_GtmDraftOnlyAgent proves go:embed loads gtm-draft-only-agent.
func TestLoadBuiltin_GtmDraftOnlyAgent(t *testing.T) {
	cat, err := LoadBuiltin()
	if err != nil {
		t.Fatal(err)
	}
	sk, ok := cat.Get("gtm-draft-only-agent")
	if !ok {
		t.Fatalf("missing gtm-draft-only-agent; names=%v", cat.Names())
	}
	if sk.Name != "gtm-draft-only-agent" {
		t.Fatalf("name=%q", sk.Name)
	}
	if strings.TrimSpace(sk.Description) == "" {
		t.Fatal("description empty")
	}
	// Builtin source marker.
	if !strings.Contains(sk.Path, "builtin") && sk.SourceDir != "builtin" {
		t.Fatalf("path/source not builtin: path=%q source=%q", sk.Path, sk.SourceDir)
	}
}

// TestLoadWithBuiltin_GtmDraftOnlyAgentAlwaysPresent: skill present even with empty dirs.
func TestLoadWithBuiltin_GtmDraftOnlyAgentAlwaysPresent(t *testing.T) {
	cat, err := LoadWithBuiltin(filepath.Join(t.TempDir(), "nope"))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cat.Get("gtm-draft-only-agent"); !ok {
		t.Fatalf("gtm-draft-only-agent missing when dirs empty; names=%v", cat.Names())
	}
	// Prior builtins still present.
	if _, ok := cat.Get("connector-integrations-setup"); !ok {
		t.Fatalf("connector-integrations-setup missing; names=%v", cat.Names())
	}
	if _, ok := cat.Get("memory-advanced-agent"); !ok {
		t.Fatalf("memory-advanced-agent missing; names=%v", cat.Names())
	}
}

// TestLoadBuiltin_S1341GtmDraftOnlySkillDogfood pins residual-honest body needles.
func TestLoadBuiltin_S1341GtmDraftOnlySkillDogfood(t *testing.T) {
	cat, err := LoadBuiltin()
	if err != nil {
		t.Fatal(err)
	}
	sk, ok := cat.Get("gtm-draft-only-agent")
	if !ok {
		t.Fatalf("skill missing; names=%v", cat.Names())
	}
	desc := strings.ToLower(sk.Description)
	for _, want := range []string{
		"residual",
		"draft",
	} {
		if !strings.Contains(desc, want) {
			t.Fatalf("description missing residual needle %q: %q", want, sk.Description)
		}
	}
	// Description must not invent auto-send / Connected success.
	if strings.Contains(desc, "auto-send green") || strings.Contains(sk.Description, "Connected: yes") {
		t.Fatalf("description invents auto-send/Connected: %q", sk.Description)
	}
	body := sk.Body
	for _, want := range []string{
		"Orchestrator",
		"Content Creator",
		"Campaign Planner",
		"Lead Manager",
		"No auto SNS",
		"No auto email send",
		"Commercial CRM",
		"Drafts only",
		"Human publish",
		"list_connector_catalog",
		"plan_connector_setup",
		"portal HITL",
		"Never invent suite ops GA",
		"HubSpot",
		"Beta multi-tenant",
		"Guerrilla",
		"Salesforce",
		"GA CRM",
		"dual_write OFF",
		"not Memory GA",
		"book-demo OFF",
		"residual PASS ≠ live dogfood",
		"Memory Ops Pack",
		"not freemium palace",
		"does not APPLY",
		"hermes-grok-marketing-sales-pipeline",
		"Phase 2",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("skill body missing %q:\n%s", want, body)
		}
	}
}

// TestS1341SkillDescriptionResidualHonest pins frontmatter honesty.
func TestS1341SkillDescriptionResidualHonest(t *testing.T) {
	cat, err := LoadBuiltin()
	if err != nil {
		t.Fatal(err)
	}
	sk, ok := cat.Get("gtm-draft-only-agent")
	if !ok {
		t.Fatal("missing skill")
	}
	if !strings.Contains(sk.Description, "Residual-honest") && !strings.Contains(sk.Description, "residual-honest") {
		t.Fatalf("description not residual-honest: %q", sk.Description)
	}
	desc := strings.ToLower(sk.Description)
	if !strings.Contains(desc, "draft") {
		t.Fatalf("description should say draft-only: %q", sk.Description)
	}
	if !strings.Contains(desc, "auto-send") && !strings.Contains(desc, "not auto-send") {
		t.Fatalf("description should say not auto-send: %q", sk.Description)
	}
	if !strings.Contains(desc, "hitl") && !strings.Contains(sk.Description, "HITL") {
		t.Fatalf("description should mention HITL publish: %q", sk.Description)
	}
}

// --- s1363: builtin aion-agent-onboarding skill (residual-honest TUI ↔ aion) ---

// TestLoadBuiltin_AionAgentOnboarding proves go:embed loads aion-agent-onboarding.
func TestLoadBuiltin_AionAgentOnboarding(t *testing.T) {
	cat, err := LoadBuiltin()
	if err != nil {
		t.Fatal(err)
	}
	sk, ok := cat.Get("aion-agent-onboarding")
	if !ok {
		t.Fatalf("missing aion-agent-onboarding; names=%v", cat.Names())
	}
	if sk.Name != "aion-agent-onboarding" {
		t.Fatalf("name=%q", sk.Name)
	}
	if strings.TrimSpace(sk.Description) == "" {
		t.Fatal("description empty")
	}
	if !strings.Contains(sk.Path, "builtin") && sk.SourceDir != "builtin" {
		t.Fatalf("path/source not builtin: path=%q source=%q", sk.Path, sk.SourceDir)
	}
}

// TestLoadWithBuiltin_AionAgentOnboardingAlwaysPresent: skill present even with empty dirs.
func TestLoadWithBuiltin_AionAgentOnboardingAlwaysPresent(t *testing.T) {
	cat, err := LoadWithBuiltin(filepath.Join(t.TempDir(), "nope"))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cat.Get("aion-agent-onboarding"); !ok {
		t.Fatalf("aion-agent-onboarding missing when dirs empty; names=%v", cat.Names())
	}
	// Prior builtins still present.
	if _, ok := cat.Get("connector-integrations-setup"); !ok {
		t.Fatalf("connector-integrations-setup missing; names=%v", cat.Names())
	}
	if _, ok := cat.Get("memory-advanced-agent"); !ok {
		t.Fatalf("memory-advanced-agent missing; names=%v", cat.Names())
	}
	if _, ok := cat.Get("gtm-draft-only-agent"); !ok {
		t.Fatalf("gtm-draft-only-agent missing; names=%v", cat.Names())
	}
}

// TestLoadBuiltin_S1363AionAgentOnboardingSkillDogfood pins residual-honest body needles.
func TestLoadBuiltin_S1363AionAgentOnboardingSkillDogfood(t *testing.T) {
	cat, err := LoadBuiltin()
	if err != nil {
		t.Fatal(err)
	}
	sk, ok := cat.Get("aion-agent-onboarding")
	if !ok {
		t.Fatalf("skill missing; names=%v", cat.Names())
	}
	desc := strings.ToLower(sk.Description)
	for _, want := range []string{
		"residual",
		"onboard",
	} {
		if !strings.Contains(desc, want) {
			t.Fatalf("description missing residual needle %q: %q", want, sk.Description)
		}
	}
	if strings.Contains(sk.Description, "Connected: yes") || strings.Contains(desc, "memory ga shipped") {
		t.Fatalf("description invents Connected/Memory GA: %q", sk.Description)
	}
	body := sk.Body
	for _, want := range []string{
		"list_connector_catalog",
		"plan_connector_setup",
		"list_org_connector_installs",
		"Catalog status ≠ install Connected",
		"available=false",
		"empty-as-none",
		"portal HITL",
		"console.iome.sh/integrations",
		"cannot write installs",
		"dual_write OFF",
		"local-primary",
		"not Memory GA",
		"book-demo OFF",
		"residual PASS ≠ live dogfood",
		"never invent install green",
		"INSTALL_STORE APPLY",
		"plugins dogfood",
		"Agent Plugins GA",
		"~$88/$119",
		"knowledge/analytical",
		"/onboard",
		"/integrations status",
		"aion-onboarding",
		"AttachMCP",
		// s1368 portal Agent/MCP half + TUI half
		"console.iome.sh/settings/agent",
		"Agent/MCP",
		"copy MCP connection",
		"test invoke",
		"probe only",
		"[[mcp.servers]]",
		"streamable HTTP",
		"/onboard portal",
		"/onboard status",
		// s1372 post-onboard next lanes + s1377 lane drills + s1382 status board
		"/onboard next",
		"iomesh plugins dogfood",
		"Agent Plugins GA",
		"/gtm checklist",
		"gtm-draft-only-agent",
		"drafts only",
		"no auto-send",
		"human publish",
		"aion-memory-mcp",
		"package load ≠ Memory GA",
		"freemium palace",
		"AionAgentOnboardingNextLanes",
		// s1377 per-lane drills
		"/onboard next plugins",
		"/onboard next gtm",
		"/onboard next memory",
		"AionAgentOnboardingNextPluginsLane",
		"AionAgentOnboardingNextGtmLane",
		"AionAgentOnboardingNextMemoryLane",
		"plugins|gtm|memory",
		// s1402 mesh streaming lane
		"/onboard next mesh",
		"AionAgentOnboardingNextMeshLane",
		"streaming org heartbeats",
		"mesh ≠ memory",
		"streams_not_probed",
		"not OTel/APM",
		"never invent stream green",
		"plugins|gtm|memory|mesh",
		// s1407 Ops Pack / memory-pull lane
		"/onboard next memory-pull",
		"AionAgentOnboardingNextMemoryPullLane",
		"Ops Pack pull path",
		"pull_not_probed",
		"Ops Pack ≠ GPU fleet",
		"never invent pull green",
		"ops-pack",
		"pull-path",
		"plugins|gtm|memory|mesh|memory-pull",
		// s1417 agentic integrations product plane 3
		"/onboard next agentic",
		"AionAgentOnboardingNextAgenticLane",
		"product plane 3",
		"agentic integrations",
		"MCP list/plan residual-honest",
		"list_plan_not_connected",
		"plan_connector_setup",
		"portal-hitl",
		"list-plan",
		"agentic-integrations",
		"template= ≠ install APPLY",
		"plugins|gtm|memory|mesh|memory-pull|agentic",
		// s1413 human-gates honesty board
		"/onboard next human-gates",
		"AionAgentHumanGatesHonestyBoard",
		"PASS ≠ invent human-gate green",
		"PASS ≠ live APPLY",
		"open boxes stay open",
		"Slack HMAC",
		"Stripe Customers:Write",
		"H1/H2 INSTALL_STORE",
		"Knowledge Beta→GA cannot invent H1/H2 offline",
		"leave ON_SIGNAL unset",
		"apply-gates",
		"make human-gates-status",
		// s1382 lane status board
		"/onboard next status",
		"AionAgentOnboardingNextLaneStatus",
		"dogfood_not_run",
		"portal_hitl_still",
		"samples_ok",
		"path_ready",
		"skill_ready",
		"residual_only",
		// s1387 status export receipt
		"/onboard next export",
		"AionAgentOnboardingNextLaneStatusExport",
		"evidence_kind=onboard_next_lane_status_export",
		"board/export evidence ≠ invent Connected",
		"plugins|gtm|memory|mesh|memory-pull|agentic|status|export|human-gates",
		// s1397 session soft dogfood on status/export
		"soft_offline_dogfood_session_pass",
		"soft_offline_dogfood_session_fail",
		"session soft ≠ live dogfood",
		"/plugins dogfood",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("skill body missing %q:\n%s", want, body)
		}
	}
}

// TestS1363SkillDescriptionResidualHonest pins frontmatter honesty.
func TestS1363SkillDescriptionResidualHonest(t *testing.T) {
	cat, err := LoadBuiltin()
	if err != nil {
		t.Fatal(err)
	}
	sk, ok := cat.Get("aion-agent-onboarding")
	if !ok {
		t.Fatal("missing skill")
	}
	if !strings.Contains(sk.Description, "Residual-honest") && !strings.Contains(sk.Description, "residual-honest") {
		t.Fatalf("description not residual-honest: %q", sk.Description)
	}
	desc := strings.ToLower(sk.Description)
	if !strings.Contains(desc, "portal") && !strings.Contains(sk.Description, "HITL") {
		t.Fatalf("description should mention portal HITL: %q", sk.Description)
	}
	if !strings.Contains(desc, "connected") && !strings.Contains(sk.Description, "Connected") {
		t.Fatalf("description should say never invent Connected: %q", sk.Description)
	}
	if !strings.Contains(desc, "memory") {
		t.Fatalf("description should mention memory honesty: %q", sk.Description)
	}
}
