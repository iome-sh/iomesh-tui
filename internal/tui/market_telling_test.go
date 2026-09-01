package tui

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setupMarketTelling(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("IOMESH_CONFIG", filepath.Join(dir, "config.toml"))
	path := filepath.Join(dir, "palace", MarketTellingTenantDir, MarketTellingFileName)

	prevPath := marketTellingPathFn
	prevNow := marketTellingNowFn
	marketTellingPathFn = func() (string, error) { return path, nil }
	marketTellingNowFn = func() time.Time {
		return time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		marketTellingPathFn = prevPath
		marketTellingNowFn = prevNow
	})
	return path
}

func assertMarketTellingHonesty(t *testing.T, out string) {
	t.Helper()
	for _, n := range []string{
		"dual_write OFF",
		"not Memory GA",
		"not git SoR",
		"no Slack persist",
		"catalog ≠ Connected",
	} {
		if !strings.Contains(out, n) {
			t.Fatalf("honesty missing %q:\n%s", n, out)
		}
	}
	for _, bad := range []string{
		"dual_write ON",
		"Memory GA shipped",
		"hosted Memory GA",
		"Slack persist ON",
		"CRM Connected",
		"leftover_is_host",
		"seven-source",
		"auto-apply green",
	} {
		if strings.Contains(out, bad) {
			t.Fatalf("must not invent %q:\n%s", bad, out)
		}
	}
}

func TestMarketTelling_MissingIsEmptyFailOpen(t *testing.T) {
	path := setupMarketTelling(t)
	doc, exists, err := loadMarketTelling()
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("missing palace must fail-open empty")
	}
	if doc.Source != MarketTellingSource || doc.Tenant != MarketTellingTenant {
		t.Fatalf("empty identity source=%s tenant=%s", doc.Source, doc.Tenant)
	}
	if doc.DualWrite {
		t.Fatal("dual_write must stay OFF")
	}
	out := formatMarketTellingShow(doc, false, path)
	if !strings.Contains(out, "(empty)") {
		t.Fatalf("empty show missing:\n%s", out)
	}
	if strings.Contains(out, "market truth MCP") && strings.Contains(out, "Connected") {
		t.Fatalf("empty must not invent connected market truth:\n%s", out)
	}
	assertMarketTellingHonesty(t, out)
}

func TestMarketTelling_WritePalaceNotGitSoR(t *testing.T) {
	path := setupMarketTelling(t)
	msg, err := writeMarketTelling(VocBriefKind, "support theme: billing cancel language up", "theme vanishes next window", 0.7)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(msg, "voc_brief written") {
		t.Fatalf("write confirm missing:\n%s", msg)
	}
	if !strings.Contains(msg, "source=agent-brief") {
		t.Fatalf("source=agent-brief missing:\n%s", msg)
	}
	if !strings.Contains(msg, "tenant=gtm/founder") {
		t.Fatalf("tenant gtm/founder missing:\n%s", msg)
	}
	if !strings.Contains(msg, "palace") || strings.Contains(msg, "git SoR") && !strings.Contains(msg, "not git SoR") {
		t.Fatalf("must pin palace SoR not git:\n%s", msg)
	}
	assertMarketTellingHonesty(t, msg)

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("palace file missing: %v", err)
	}
	// SoR is the palace path, not a tracked repo markdown.
	if strings.Contains(path, "/docs/") || strings.HasSuffix(path, ".md") {
		t.Fatalf("palace path looks like git SoR: %s", path)
	}
	if !strings.Contains(path, filepath.Join("palace", MarketTellingTenantDir, MarketTellingFileName)) {
		t.Fatalf("palace path want palace/gtm-founder/market_telling.json, got %s", path)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc PalaceMarketTelling
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Kind != VocBriefKind || doc.Source != MarketTellingSource || doc.Tenant != MarketTellingTenant {
		t.Fatalf("doc identity %+v", doc)
	}
	if doc.DualWrite {
		t.Fatal("JSON dual_write must be false")
	}
	if doc.Confidence != 0.7 || doc.Falsify != "theme vanishes next window" {
		t.Fatalf("confidence/falsify %+v", doc)
	}

}

func TestMarketTelling_KindAliases(t *testing.T) {
	setupMarketTelling(t)
	msg, err := writeMarketTelling(MarketTellingKind, "new stall in gtm.pipeline", "stall clears next week", 0.4)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(msg, "market_telling written") {
		t.Fatalf("kind market_telling missing:\n%s", msg)
	}
	if _, err := writeMarketTelling("nope", "h", "f", 0.1); err == nil {
		t.Fatal("want reject unknown kind")
	}
}

func TestMarketTelling_LedgerDroppedVsFalsifiedVsShipped(t *testing.T) {
	setupMarketTelling(t)
	if _, err := writeMarketTelling(MarketTellingKind, "H1 support theme", "theme gone next window", 0.6); err != nil {
		t.Fatal(err)
	}

	shipped, err := recordLedger("H1", LedgerShipped, "landed", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(shipped, "shipped (not dropped, not falsified)") {
		t.Fatalf("shipped distinction missing:\n%s", shipped)
	}
	assertMarketTellingHonesty(t, shipped)

	killed, err := recordLedger("H2", LedgerKilled, "dropped idea", LedgerShipped, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(killed, "dropped ≠ falsified") {
		t.Fatalf("killed must not look falsified:\n%s", killed)
	}
	if !strings.Contains(killed, "vs yesterday=shipped") {
		t.Fatalf("contradiction vs yesterday missing:\n%s", killed)
	}
	if !strings.Contains(killed, "contradiction=status vs yesterday: shipped → killed") {
		t.Fatalf("auto contradiction vs yesterday missing:\n%s", killed)
	}
	if strings.Contains(killed, "falsified (not merely dropped)") {
		t.Fatalf("killed must not stamp falsified:\n%s", killed)
	}

	moved, err := recordLedger("H3", LedgerMoved, "moved to weekly", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(moved, "dropped ≠ falsified") {
		t.Fatalf("moved is dropped, not falsified:\n%s", moved)
	}

	if _, err := recordLedger("H4", LedgerFalsified, "busted", "", ""); err == nil {
		t.Fatal("falsified without contradiction must refuse")
	}
	falsified, err := recordLedger("H4", LedgerFalsified, "busted", LedgerShipped, "n of N receipts vs prior window")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(falsified, "falsified (not merely dropped)") {
		t.Fatalf("falsified distinction missing:\n%s", falsified)
	}
	if !strings.Contains(falsified, "contradiction=n of N receipts vs prior window") {
		t.Fatalf("explicit contradiction missing:\n%s", falsified)
	}

	doc, exists, err := loadMarketTelling()
	if err != nil || !exists {
		t.Fatalf("exists=%v err=%v", exists, err)
	}
	byID := map[string]PalaceLedgerEntry{}
	for _, e := range doc.Ledger {
		byID[e.ID] = e
	}
	if !byID["H2"].Dropped || byID["H2"].Falsified {
		t.Fatalf("H2 killed must be dropped not falsified: %+v", byID["H2"])
	}
	if !byID["H4"].Falsified || byID["H4"].Dropped {
		t.Fatalf("H4 falsified must not also be dropped: %+v", byID["H4"])
	}
	if byID["H1"].Dropped || byID["H1"].Falsified {
		t.Fatalf("H1 shipped is neither: %+v", byID["H1"])
	}
}

func TestMarketTelling_CadenceRefuseDailyBelowVolume(t *testing.T) {
	setupMarketTelling(t)
	ok, line := evaluateCadence(CadenceDaily, 3, MarketTellingDailyFloor)
	if ok {
		t.Fatalf("daily n=3 must refuse: %s", line)
	}
	if !strings.Contains(line, "refuse daily") || !strings.Contains(line, "insufficient volume n=3") {
		t.Fatalf("refuse wording:\n%s", line)
	}

	okW, lineW := evaluateCadence(CadenceWeekly, 3, MarketTellingDailyFloor)
	if !okW {
		t.Fatalf("weekly allowed below floor: %s", lineW)
	}
	if !strings.Contains(lineW, "weekly allowed below daily floor") {
		t.Fatalf("weekly wording:\n%s", lineW)
	}

	okT, lineT := evaluateCadence(CadenceOnThreshold, 3, MarketTellingDailyFloor)
	if okT {
		t.Fatalf("on_threshold n=3 must refuse: %s", lineT)
	}
	okT2, _ := evaluateCadence(CadenceOnThreshold, MarketTellingDailyFloor, MarketTellingDailyFloor)
	if !okT2 {
		t.Fatal("on_threshold at floor must pass")
	}
	okD, _ := evaluateCadence(CadenceDaily, MarketTellingDailyFloor, MarketTellingDailyFloor)
	if !okD {
		t.Fatal("daily at floor must pass")
	}

	msg, err := setCadence(CadenceDaily, 3)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(msg, "refuse daily") || !strings.Contains(msg, "insufficient-signal") {
		t.Fatalf("setCadence refuse missing:\n%s", msg)
	}
	assertMarketTellingHonesty(t, msg)

	msgW, err := setCadence(CadenceWeekly, 3)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(msgW, "cadence: weekly") {
		t.Fatalf("weekly set missing:\n%s", msgW)
	}
	assertMarketTellingHonesty(t, msgW)
}

func TestMarketTelling_OneSupportThemeRecipeIncidentMetadata(t *testing.T) {
	setupMarketTelling(t)
	r := defaultSupportThemeRecipe()
	if r.Name != MarketTellingRecipeName || r.Kind != MarketTellingRecipeName {
		t.Fatalf("recipe identity %+v", r)
	}
	if len(r.Sources) != 3 {
		t.Fatalf("want 3 first-party sources, got %v", r.Sources)
	}
	for _, s := range r.Sources {
		if s == "slack" || s == "crm" || s == "salesforce" {
			t.Fatalf("recipe must not include %s: %v", s, r.Sources)
		}
	}

	r.ID = "THM-1"
	r.Summary = "billing cancel language"
	r.Pointer = "ticket/THM-1"
	r.EventTime = "2026-09-01T12:00:00Z"
	r.Subject = "dept.gtm.support.theme"
	msg, err := writeRecipe(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{
		"recipe support_theme",
		"same metadata as incidents",
		"id=THM-1",
		"event_time=2026-09-01T12:00:00Z",
		"summary=billing cancel language",
		"pointer=ticket/THM-1",
		"source_hint=agent-brief",
		"account_hash=",
		"kind=support_theme",
		"subject=dept.gtm.support.theme",
		"sources=mesh,private,github (3/3 first-party)",
		"hands off (win-back, price change)",
		"no Slack persist",
		"CRM ≠ Connected",
	} {
		if !strings.Contains(msg, n) {
			t.Fatalf("recipe missing %q:\n%s", n, msg)
		}
	}
	assertMarketTellingHonesty(t, msg)

	if _, err := writeRecipe(PalaceRevOpsRecipe{Name: "win_back"}); err == nil {
		t.Fatal("win_back hand must refuse")
	}
	if _, err := writeRecipe(PalaceRevOpsRecipe{Name: "price_change"}); err == nil {
		t.Fatal("price_change hand must refuse")
	}
	if _, err := writeRecipe(PalaceRevOpsRecipe{Name: "billing_cancel"}); err == nil {
		t.Fatal("only support_theme recipe")
	}
	if _, err := writeRecipe(PalaceRevOpsRecipe{Name: "lost_deal"}); err == nil {
		t.Fatal("only support_theme recipe")
	}
}

func TestMarketTelling_SourcesCapAndForbidden(t *testing.T) {
	_, err := parseFirstPartySources("mesh,private,github,slack")
	if err == nil {
		t.Fatal("4th source must refuse")
	}
	if !strings.Contains(err.Error(), "slack") && !strings.Contains(err.Error(), "Slack") {
		t.Fatalf("slack refuse: %v", err)
	}
	_, err = parseFirstPartySources("mesh,crm")
	if err == nil {
		t.Fatal("crm must refuse")
	}
	_, err = parseFirstPartySources("salesforce,hubspot,stripe,zendesk,slack,jira,pagerduty")
	if err == nil {
		t.Fatal("seven-source market truth must refuse")
	}
	got, err := parseFirstPartySources("mesh,private,github")
	if err != nil || len(got) != 3 {
		t.Fatalf("got=%v err=%v", got, err)
	}
	got2, err := parseFirstPartySources("mesh,palace")
	if err != nil || len(got2) != 2 || got2[1] != "private" {
		t.Fatalf("palace alias → private: %v %v", got2, err)
	}
}

func TestHandleSlash_GtmBrief(t *testing.T) {
	setupMarketTelling(t)
	rt := testRuntime(t)
	adapter := runtimeAdapter{rt: rt}

	var out bytes.Buffer
	if quit, err := handleSlash(&out, adapter, "/gtm brief help"); quit || err != nil {
		t.Fatalf("quit=%v err=%v", quit, err)
	}
	help := out.String()
	if !strings.Contains(help, "usage: /gtm brief") || !strings.Contains(help, "source=agent-brief") {
		t.Fatalf("help:\n%s", help)
	}
	assertMarketTellingHonesty(t, help)
	if !strings.Contains(help, "hands (win-back, price change) stay off this plane") {
		t.Fatalf("hands pin missing:\n%s", help)
	}

	out.Reset()
	_, _ = handleSlash(&out, adapter, "/gtm brief")
	if !strings.Contains(out.String(), "(empty)") {
		t.Fatalf("empty show:\n%s", out.String())
	}
	assertMarketTellingHonesty(t, out.String())

	out.Reset()
	line := `/gtm brief write --kind voc_brief --hypothesis support-theme-billing-cancel --confidence 0.65 --falsify theme-vanishes-next-window`
	if quit, err := handleSlash(&out, adapter, line); quit || err != nil {
		t.Fatalf("write quit=%v err=%v", quit, err)
	}
	got := out.String()
	if !strings.Contains(got, "voc_brief written") || !strings.Contains(got, "source=agent-brief") {
		t.Fatalf("write:\n%s", got)
	}
	assertMarketTellingHonesty(t, got)

	out.Reset()
	_, _ = handleSlash(&out, adapter, "/gtm brief ledger killed H2 --vs-yesterday shipped")
	if !strings.Contains(out.String(), "dropped ≠ falsified") {
		t.Fatalf("ledger:\n%s", out.String())
	}

	out.Reset()
	_, _ = handleSlash(&out, adapter, "/gtm brief cadence daily --volume 3")
	if !strings.Contains(out.String(), "refuse daily") {
		t.Fatalf("cadence:\n%s", out.String())
	}

	out.Reset()
	_, _ = handleSlash(&out, adapter, "/gtm brief recipe --id THM-9 --summary theme --pointer ticket/THM-9 --sources mesh,private,github")
	if !strings.Contains(out.String(), "same metadata as incidents") {
		t.Fatalf("recipe:\n%s", out.String())
	}
	if strings.Contains(out.String(), "Slack persist ON") || strings.Contains(out.String(), "CRM Connected") {
		t.Fatalf("recipe invented connectors:\n%s", out.String())
	}

	out.Reset()
	_, _ = handleSlash(&out, adapter, "/gtm brief recipe --name win_back")
	if !strings.Contains(out.String(), "hands") {
		t.Fatalf("win_back refuse:\n%s", out.String())
	}

	// aliases
	out.Reset()
	_, _ = handleSlash(&out, adapter, "/gtm voc-brief help")
	if !strings.Contains(out.String(), "usage: /gtm brief") {
		t.Fatalf("voc-brief alias:\n%s", out.String())
	}
}

func TestMarketTellingHelp_MentionsContract(t *testing.T) {
	h := marketTellingHelp()
	for _, n := range []string{
		"market_telling",
		"voc_brief",
		"agent-brief",
		"gtm/founder",
		"shipped|moved|killed|falsified",
		"daily|weekly|on_threshold",
		"support-theme",
		"not git SoR",
		"dual_write OFF",
		"not Memory GA",
		"no Slack persist",
		"CRM ≠ Connected",
		"win-back",
	} {
		if !strings.Contains(h, n) {
			t.Fatalf("help missing %q:\n%s", n, h)
		}
	}
}

func TestEvaluateCadence_UnknownMode(t *testing.T) {
	ok, line := evaluateCadence("hourly", 99, 8)
	if ok {
		t.Fatal("hourly must refuse")
	}
	if !strings.Contains(line, "daily|weekly|on_threshold") {
		t.Fatalf("unknown mode: %s", line)
	}
}
