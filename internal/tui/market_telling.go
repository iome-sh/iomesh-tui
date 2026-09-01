package tui

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/config"
)

// Market-telling / voc_brief palace artifact (#372 / FR-23 · FR-27 · FR-28 · FR-29 · FR-32 · FR-34).
// Named palace entry written by the brief skill (source=agent-brief) under tenant gtm/founder.
// SoR is the local palace file — not a git markdown. dual_write stays OFF.
// Hands (win-back, price change) stay off this plane. One RevOps support-theme recipe
// uses the same metadata contract as incidents. At most three first-party sources.

const (
	MarketTellingKind          = "market_telling"
	VocBriefKind               = "voc_brief"
	MarketTellingSource        = "agent-brief"
	MarketTellingTenant        = "gtm/founder"
	MarketTellingTenantDir     = "gtm-founder"
	MarketTellingFileName      = "market_telling.json"
	MarketTellingDailyFloor    = 8
	MarketTellingRecipeName    = "support_theme"
	marketTellingHonestyFooter = "dual_write OFF · not Memory GA · not git SoR · no Slack persist · catalog ≠ Connected · CRM ≠ Connected"

	LedgerShipped   = "shipped"
	LedgerMoved     = "moved"
	LedgerKilled    = "killed"
	LedgerFalsified = "falsified"

	CadenceDaily       = "daily"
	CadenceWeekly      = "weekly"
	CadenceOnThreshold = "on_threshold"
)

// First-party sources allowed on the one RevOps recipe (cap 3). Not a seven-source MCP.
var marketTellingFirstPartySources = []string{"mesh", "private", "github"}

var marketTellingForbiddenSources = []string{
	"slack", "crm", "salesforce", "hubspot", "stripe", "zendesk",
	"external", "sponsored", "demand_feed",
}

// Overridable in tests. Production resolves beside user config.
var marketTellingPathFn = defaultMarketTellingPath

// Overridable clock.
var marketTellingNowFn = func() time.Time { return time.Now() }

func defaultMarketTellingPath() (string, error) {
	cfgPath, err := config.UserConfigPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(cfgPath), "palace", MarketTellingTenantDir, MarketTellingFileName), nil
}

func marketTellingHelp() string {
	return strings.TrimSpace(`usage: /gtm brief [show|write|ledger|cadence|recipe|help]
  show     palace voc_brief / market_telling (tenant gtm/founder · source=agent-brief)
  write    --hypothesis TEXT --confidence 0-1 --falsify TEXT [--kind market_telling|voc_brief]
           (slash tokens are space-split; use --hypothesis=one-token)
  ledger   [shipped|moved|killed|falsified <id> [--hypothesis TEXT] [--vs-yesterday STATUS] [--contradiction TEXT]]
  cadence  [daily|weekly|on_threshold] [--volume N]   daily refused below floor=` + strconv.Itoa(MarketTellingDailyFloor) + `
  recipe   one RevOps support-theme (incident metadata) [--id --summary --pointer --event-time --subject --account-hash --sources mesh,private,github]
` + marketTellingHonestyFooter + `
hands (win-back, price change) stay off this plane`)
}

// PalaceMarketTelling is the named palace artifact. Not a git filename SoR.
type PalaceMarketTelling struct {
	Kind       string              `json:"kind"`
	Source     string              `json:"source"`
	Tenant     string              `json:"tenant"`
	DualWrite  bool                `json:"dual_write"`
	Hypothesis string              `json:"hypothesis"`
	Confidence float64             `json:"confidence"`
	Falsify    string              `json:"falsification_test"`
	WrittenAt  string              `json:"written_at"`
	Cadence    PalaceCadence       `json:"cadence"`
	Ledger     []PalaceLedgerEntry `json:"ledger"`
	Recipe     *PalaceRevOpsRecipe `json:"recipe,omitempty"`
	Note       string              `json:"note"`
}

// PalaceCadence is daily|weekly|on_threshold with a volume floor.
type PalaceCadence struct {
	Mode   string `json:"mode"`
	Volume int    `json:"volume"`
	Floor  int    `json:"floor"`
}

// PalaceLedgerEntry is shipped / moved / killed vs falsified, plus contradiction vs yesterday.
type PalaceLedgerEntry struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	Hypothesis    string `json:"hypothesis,omitempty"`
	Yesterday     string `json:"yesterday,omitempty"`
	Contradiction string `json:"contradiction,omitempty"`
	Dropped       bool   `json:"dropped,omitempty"`
	Falsified     bool   `json:"falsified,omitempty"`
	UpdatedAt     string `json:"updated_at"`
}

// PalaceRevOpsRecipe is the one support-theme recipe. Same metadata contract as incidents.
type PalaceRevOpsRecipe struct {
	Name        string   `json:"name"`
	ID          string   `json:"id"`
	EventTime   string   `json:"event_time"`
	Summary     string   `json:"summary"`
	SourceHint  string   `json:"source_hint"`
	Pointer     string   `json:"pointer"`
	AccountHash string   `json:"account_hash"`
	Kind        string   `json:"kind"`
	Subject     string   `json:"subject"`
	Sources     []string `json:"sources"`
}

func emptyMarketTelling() PalaceMarketTelling {
	return PalaceMarketTelling{
		Kind:      MarketTellingKind,
		Source:    MarketTellingSource,
		Tenant:    MarketTellingTenant,
		DualWrite: false,
		Cadence: PalaceCadence{
			Floor: MarketTellingDailyFloor,
		},
		Note: "palace SoR · source=agent-brief · tenant gtm/founder · dual_write OFF · not Memory GA",
	}
}

func loadMarketTelling() (PalaceMarketTelling, bool, error) {
	path, err := marketTellingPathFn()
	if err != nil || strings.TrimSpace(path) == "" {
		return emptyMarketTelling(), false, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return emptyMarketTelling(), false, nil
		}
		return emptyMarketTelling(), false, err
	}
	var doc PalaceMarketTelling
	if json.Unmarshal(b, &doc) != nil {
		return emptyMarketTelling(), false, nil
	}
	doc.DualWrite = false
	if strings.TrimSpace(doc.Source) == "" {
		doc.Source = MarketTellingSource
	}
	if strings.TrimSpace(doc.Tenant) == "" {
		doc.Tenant = MarketTellingTenant
	}
	if strings.TrimSpace(doc.Kind) == "" {
		doc.Kind = MarketTellingKind
	}
	if doc.Cadence.Floor <= 0 {
		doc.Cadence.Floor = MarketTellingDailyFloor
	}
	return doc, true, nil
}

func saveMarketTelling(doc PalaceMarketTelling) (string, error) {
	path, err := marketTellingPathFn()
	if err != nil {
		return "", fmt.Errorf("palace path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("palace dir: %w", err)
	}
	doc.DualWrite = false
	doc.Source = MarketTellingSource
	doc.Tenant = MarketTellingTenant
	if strings.TrimSpace(doc.Kind) == "" {
		doc.Kind = MarketTellingKind
	}
	if doc.Cadence.Floor <= 0 {
		doc.Cadence.Floor = MarketTellingDailyFloor
	}
	if strings.TrimSpace(doc.Note) == "" {
		doc.Note = "palace SoR · source=agent-brief · tenant gtm/founder · dual_write OFF · not Memory GA"
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o600); err != nil {
		return "", fmt.Errorf("palace write: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("palace rename: %w", err)
	}
	return path, nil
}

func normalizeMarketTellingKind(kind string) (string, string) {
	k := strings.ToLower(strings.TrimSpace(kind))
	k = strings.ReplaceAll(k, "-", "_")
	switch k {
	case "", MarketTellingKind:
		return MarketTellingKind, ""
	case VocBriefKind:
		return VocBriefKind, ""
	default:
		return "", "kind must be market_telling or voc_brief"
	}
}

func isForbiddenHand(s string) bool {
	h := strings.ToLower(strings.TrimSpace(s))
	h = strings.ReplaceAll(h, "-", "_")
	switch h {
	case "win_back", "winback", "price_change":
		return true
	}
	return false
}

func writeMarketTelling(kind, hypothesis, falsify string, confidence float64) (string, error) {
	k, kerr := normalizeMarketTellingKind(kind)
	if kerr != "" {
		return "", fmt.Errorf("%s", kerr)
	}
	hypothesis = strings.TrimSpace(hypothesis)
	falsify = strings.TrimSpace(falsify)
	if hypothesis == "" {
		return "", fmt.Errorf("hypothesis required")
	}
	if falsify == "" {
		return "", fmt.Errorf("falsification test required")
	}
	if confidence < 0 || confidence > 1 {
		return "", fmt.Errorf("confidence must be 0-1")
	}
	if isForbiddenHand(k) {
		return "", fmt.Errorf("hands (win-back, price change) stay off this plane")
	}
	doc, _, err := loadMarketTelling()
	if err != nil {
		return "", err
	}
	now := marketTellingNowFn().UTC().Format(time.RFC3339)
	doc.Kind = k
	doc.Source = MarketTellingSource
	doc.Tenant = MarketTellingTenant
	doc.DualWrite = false
	doc.Hypothesis = hypothesis
	doc.Confidence = confidence
	doc.Falsify = falsify
	doc.WrittenAt = now
	path, err := saveMarketTelling(doc)
	if err != nil {
		return "", err
	}
	return formatMarketTellingWrite(doc, path), nil
}

func formatMarketTellingWrite(doc PalaceMarketTelling, path string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s written · palace %s · not git SoR\n", doc.Kind, path)
	fmt.Fprintf(&b, "  source=%s tenant=%s dual_write=OFF\n", doc.Source, doc.Tenant)
	fmt.Fprintf(&b, "  confidence=%.2f falsify=%s\n", doc.Confidence, doc.Falsify)
	fmt.Fprintf(&b, "  hypothesis=%s\n", doc.Hypothesis)
	b.WriteString(marketTellingHonestyFooter)
	return b.String()
}

func formatMarketTellingShow(doc PalaceMarketTelling, exists bool, path string) string {
	var b strings.Builder
	if !exists {
		fmt.Fprintf(&b, "palace %s / %s: (empty) · tenant %s · source=%s · not git SoR\n",
			MarketTellingKind, VocBriefKind, MarketTellingTenant, MarketTellingSource)
		fmt.Fprintf(&b, "  write with /gtm brief write — fail-open empty ≠ invent market truth\n")
		b.WriteString(marketTellingHonestyFooter)
		return b.String()
	}
	fmt.Fprintf(&b, "%s · palace %s · not git SoR\n", doc.Kind, path)
	fmt.Fprintf(&b, "  source=%s tenant=%s dual_write=OFF written_at=%s\n", doc.Source, doc.Tenant, doc.WrittenAt)
	fmt.Fprintf(&b, "  confidence=%.2f falsify=%s\n", doc.Confidence, doc.Falsify)
	if h := strings.TrimSpace(doc.Hypothesis); h != "" {
		fmt.Fprintf(&b, "  hypothesis=%s\n", h)
	}
	b.WriteString(formatCadenceLines(doc.Cadence))
	b.WriteString(formatLedgerLines(doc.Ledger))
	if doc.Recipe != nil {
		b.WriteString(formatRecipeLines(*doc.Recipe))
	}
	b.WriteString(marketTellingHonestyFooter)
	return b.String()
}

func formatCadenceLines(c PalaceCadence) string {
	mode := strings.TrimSpace(c.Mode)
	if mode == "" {
		mode = "(unset)"
	}
	return fmt.Sprintf("  cadence=%s volume=%d floor=%d\n", mode, c.Volume, c.Floor)
}

func formatLedgerLines(entries []PalaceLedgerEntry) string {
	if len(entries) == 0 {
		return "  ledger: (empty) · shipped/moved/killed vs falsified\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "  ledger (%d):\n", len(entries))
	for i, e := range entries {
		fmt.Fprintf(&b, "    %d. %s %s", i+1, e.ID, e.Status)
		if e.Dropped && !e.Falsified {
			b.WriteString(" · dropped ≠ falsified")
		}
		if e.Falsified {
			b.WriteString(" · falsified")
		}
		if y := strings.TrimSpace(e.Yesterday); y != "" {
			fmt.Fprintf(&b, " · vs yesterday=%s", y)
		}
		if c := strings.TrimSpace(e.Contradiction); c != "" {
			fmt.Fprintf(&b, " · contradiction=%s", c)
		}
		if h := strings.TrimSpace(e.Hypothesis); h != "" {
			fmt.Fprintf(&b, " · %s", h)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func formatRecipeLines(r PalaceRevOpsRecipe) string {
	var b strings.Builder
	fmt.Fprintf(&b, "  recipe=%s · same metadata as incidents\n", r.Name)
	fmt.Fprintf(&b, "    id=%s event_time=%s kind=%s subject=%s\n",
		emptyDashShow(r.ID), emptyDashShow(r.EventTime), emptyDashShow(r.Kind), emptyDashShow(r.Subject))
	fmt.Fprintf(&b, "    summary=%s pointer=%s source_hint=%s account_hash=%s\n",
		emptyDashShow(r.Summary), emptyDashShow(r.Pointer), emptyDashShow(r.SourceHint), emptyDashShow(r.AccountHash))
	fmt.Fprintf(&b, "    sources=%s (%d/3 first-party)\n", strings.Join(r.Sources, ","), len(r.Sources))
	return b.String()
}

func emptyDashShow(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return strings.TrimSpace(s)
}

func normalizeLedgerStatus(status string) (string, string) {
	s := strings.ToLower(strings.TrimSpace(status))
	s = strings.ReplaceAll(s, "-", "_")
	switch s {
	case LedgerShipped, LedgerMoved, LedgerKilled, LedgerFalsified:
		return s, ""
	default:
		return "", "ledger status must be shipped|moved|killed|falsified"
	}
}

func recordLedger(id, status, hypothesis, yesterday, contradiction string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("ledger id required")
	}
	st, errMsg := normalizeLedgerStatus(status)
	if errMsg != "" {
		return "", fmt.Errorf("%s", errMsg)
	}
	if isForbiddenHand(st) {
		return "", fmt.Errorf("hands (win-back, price change) stay off this plane")
	}
	doc, _, err := loadMarketTelling()
	if err != nil {
		return "", err
	}
	yesterday = strings.ToLower(strings.TrimSpace(yesterday))
	yesterday = strings.ReplaceAll(yesterday, "-", "_")
	if yesterday != "" {
		if _, yerr := normalizeLedgerStatus(yesterday); yerr != "" {
			return "", fmt.Errorf("yesterday: %s", yerr)
		}
	}
	contradiction = strings.TrimSpace(contradiction)
	if contradiction == "" && yesterday != "" && yesterday != st {
		contradiction = "status vs yesterday: " + yesterday + " → " + st
	}
	if st == LedgerFalsified && contradiction == "" {
		return "", fmt.Errorf("falsified requires contradiction vs yesterday (dropped ≠ falsified)")
	}
	dropped := st == LedgerMoved || st == LedgerKilled
	falsified := st == LedgerFalsified
	now := marketTellingNowFn().UTC().Format(time.RFC3339)
	entry := PalaceLedgerEntry{
		ID:            id,
		Status:        st,
		Hypothesis:    strings.TrimSpace(hypothesis),
		Yesterday:     yesterday,
		Contradiction: contradiction,
		Dropped:       dropped,
		Falsified:     falsified,
		UpdatedAt:     now,
	}
	replaced := false
	for i, existing := range doc.Ledger {
		if existing.ID == id {
			doc.Ledger[i] = entry
			replaced = true
			break
		}
	}
	if !replaced {
		doc.Ledger = append(doc.Ledger, entry)
	}
	path, err := saveMarketTelling(doc)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "ledger %s %s · palace %s · not git SoR\n", id, st, path)
	if dropped && !falsified {
		b.WriteString("  dropped ≠ falsified\n")
	}
	if falsified {
		b.WriteString("  falsified (not merely dropped)\n")
	}
	if st == LedgerShipped {
		b.WriteString("  shipped (not dropped, not falsified)\n")
	}
	if yesterday != "" {
		fmt.Fprintf(&b, "  vs yesterday=%s\n", yesterday)
	}
	if contradiction != "" {
		fmt.Fprintf(&b, "  contradiction=%s\n", contradiction)
	}
	b.WriteString(marketTellingHonestyFooter)
	return b.String(), nil
}

func evaluateCadence(mode string, volume, floor int) (ok bool, line string) {
	m := strings.ToLower(strings.TrimSpace(mode))
	m = strings.ReplaceAll(m, "-", "_")
	if floor <= 0 {
		floor = MarketTellingDailyFloor
	}
	switch m {
	case CadenceDaily:
		if volume < floor {
			return false, fmt.Sprintf("cadence: refuse daily · insufficient volume n=%d floor=%d · use weekly|on_threshold", volume, floor)
		}
		return true, fmt.Sprintf("cadence: daily · volume=%d floor=%d", volume, floor)
	case CadenceWeekly:
		return true, fmt.Sprintf("cadence: weekly · volume=%d floor=%d · weekly allowed below daily floor", volume, floor)
	case CadenceOnThreshold:
		if volume < floor {
			return false, fmt.Sprintf("cadence: refuse on_threshold · insufficient volume n=%d floor=%d", volume, floor)
		}
		return true, fmt.Sprintf("cadence: on_threshold · volume=%d floor=%d", volume, floor)
	default:
		return false, "cadence must be daily|weekly|on_threshold"
	}
}

func setCadence(mode string, volume int) (string, error) {
	ok, line := evaluateCadence(mode, volume, MarketTellingDailyFloor)
	if !ok && !strings.HasPrefix(line, "cadence: refuse") {
		return "", fmt.Errorf("%s", line)
	}
	doc, _, err := loadMarketTelling()
	if err != nil {
		return "", err
	}
	m := strings.ToLower(strings.TrimSpace(mode))
	m = strings.ReplaceAll(m, "-", "_")
	doc.Cadence = PalaceCadence{Mode: m, Volume: volume, Floor: MarketTellingDailyFloor}
	path, err := saveMarketTelling(doc)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString(line)
	fmt.Fprintf(&b, " · palace %s · not git SoR\n", path)
	if !ok {
		b.WriteString("  daily cron on thin volume refused · insufficient-signal\n")
	}
	b.WriteString(marketTellingHonestyFooter)
	return b.String(), nil
}

func parseFirstPartySources(val string) ([]string, error) {
	if strings.TrimSpace(val) == "" {
		return append([]string{}, marketTellingFirstPartySources...), nil
	}
	seen := map[string]bool{}
	var out []string
	for _, part := range strings.Split(val, ",") {
		s := strings.ToLower(strings.TrimSpace(part))
		s = strings.ReplaceAll(s, "-", "_")
		if s == "" {
			continue
		}
		if s == "palace" || s == "local" || s == "local_palace" || s == "agent_brief" {
			s = "private"
		}
		if s == "mesh_consume" || s == "mesh_stream" {
			s = "mesh"
		}
		if s == "gh" || s == "ops_github" {
			s = "github"
		}
		for _, bad := range marketTellingForbiddenSources {
			if s == bad || strings.HasPrefix(s, bad+"_") {
				return nil, fmt.Errorf("source %s refused · no Slack persist · CRM ≠ Connected · first-party only", s)
			}
		}
		allowed := false
		for _, a := range marketTellingFirstPartySources {
			if s == a {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, fmt.Errorf("source %s refused · <=3 first-party (mesh,private,github)", s)
		}
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("need at least one first-party source (mesh,private,github)")
	}
	if len(out) > 3 {
		return nil, fmt.Errorf("<=3 first-party sources (not a seven-source market truth MCP)")
	}
	return out, nil
}

func defaultSupportThemeRecipe() PalaceRevOpsRecipe {
	return PalaceRevOpsRecipe{
		Name:        MarketTellingRecipeName,
		Kind:        MarketTellingRecipeName,
		SourceHint:  MarketTellingSource,
		Subject:     "dept.gtm.support.theme",
		Sources:     append([]string{}, marketTellingFirstPartySources...),
		AccountHash: shortPalaceHash(MarketTellingTenant + "|" + MarketTellingRecipeName),
	}
}

func shortPalaceHash(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:12]
}

func writeRecipe(r PalaceRevOpsRecipe) (string, error) {
	name := strings.ToLower(strings.TrimSpace(r.Name))
	name = strings.ReplaceAll(name, "-", "_")
	if name == "" {
		name = MarketTellingRecipeName
	}
	if name != MarketTellingRecipeName {
		return "", fmt.Errorf("one RevOps recipe: support_theme (not %s) · hands (win-back, price change) stay off this plane", name)
	}
	if isForbiddenHand(name) || isForbiddenHand(r.Kind) {
		return "", fmt.Errorf("hands (win-back, price change) stay off this plane")
	}
	srcs := r.Sources
	if len(srcs) == 0 {
		srcs = append([]string{}, marketTellingFirstPartySources...)
	}
	if len(srcs) > 3 {
		return "", fmt.Errorf("<=3 first-party sources")
	}
	doc, _, err := loadMarketTelling()
	if err != nil {
		return "", err
	}
	r.Name = MarketTellingRecipeName
	r.Kind = MarketTellingRecipeName
	r.Sources = srcs
	if strings.TrimSpace(r.SourceHint) == "" {
		r.SourceHint = MarketTellingSource
	}
	if strings.TrimSpace(r.Subject) == "" {
		r.Subject = "dept.gtm.support.theme"
	}
	if strings.TrimSpace(r.AccountHash) == "" {
		r.AccountHash = shortPalaceHash(r.ID + "|" + r.Pointer + "|" + r.Summary)
	}
	doc.Recipe = &r
	path, err := saveMarketTelling(doc)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "recipe support_theme · palace %s · same metadata as incidents · not git SoR\n", path)
	b.WriteString(formatRecipeLines(r))
	b.WriteString("  hands off (win-back, price change) · no Slack persist · CRM ≠ Connected\n")
	b.WriteString(marketTellingHonestyFooter)
	return b.String(), nil
}

func handleGtmBriefSlash(out io.Writer, args []string) {
	if len(args) == 0 {
		path, _ := marketTellingPathFn()
		doc, exists, err := loadMarketTelling()
		if err != nil {
			fmt.Fprintf(out, "gtm brief: %v\n", err)
			return
		}
		fmt.Fprintln(out, formatMarketTellingShow(doc, exists, path))
		return
	}
	sub := strings.ToLower(strings.TrimSpace(args[0]))
	switch sub {
	case "help", "?":
		fmt.Fprintln(out, marketTellingHelp())
	case "show":
		path, _ := marketTellingPathFn()
		doc, exists, err := loadMarketTelling()
		if err != nil {
			fmt.Fprintf(out, "gtm brief show: %v\n", err)
			return
		}
		fmt.Fprintln(out, formatMarketTellingShow(doc, exists, path))
	case "write":
		opts, errMsg := parseMarketTellingWriteArgs(args[1:])
		if errMsg != "" {
			fmt.Fprintf(out, "gtm brief write: %s\n%s\n", errMsg, marketTellingHelp())
			return
		}
		msg, err := writeMarketTelling(opts.Kind, opts.Hypothesis, opts.Falsify, opts.Confidence)
		if err != nil {
			fmt.Fprintf(out, "gtm brief write: %v\n", err)
			return
		}
		fmt.Fprintln(out, msg)
	case "ledger":
		if len(args) == 1 {
			doc, exists, err := loadMarketTelling()
			if err != nil {
				fmt.Fprintf(out, "gtm brief ledger: %v\n", err)
				return
			}
			if !exists || len(doc.Ledger) == 0 {
				fmt.Fprintln(out, "ledger: (empty) · shipped/moved/killed vs falsified · dropped ≠ falsified")
				fmt.Fprintln(out, marketTellingHonestyFooter)
				return
			}
			fmt.Fprint(out, formatLedgerLines(doc.Ledger))
			fmt.Fprintln(out, marketTellingHonestyFooter)
			return
		}
		opts, errMsg := parseMarketTellingLedgerArgs(args[1:])
		if errMsg != "" {
			fmt.Fprintf(out, "gtm brief ledger: %s\n%s\n", errMsg, marketTellingHelp())
			return
		}
		msg, err := recordLedger(opts.ID, opts.Status, opts.Hypothesis, opts.Yesterday, opts.Contradiction)
		if err != nil {
			fmt.Fprintf(out, "gtm brief ledger: %v\n", err)
			return
		}
		fmt.Fprintln(out, msg)
	case "cadence":
		if len(args) == 1 {
			doc, exists, err := loadMarketTelling()
			if err != nil {
				fmt.Fprintf(out, "gtm brief cadence: %v\n", err)
				return
			}
			if !exists || strings.TrimSpace(doc.Cadence.Mode) == "" {
				fmt.Fprintf(out, "cadence: (unset) · daily|weekly|on_threshold · daily floor=%d\n", MarketTellingDailyFloor)
				fmt.Fprintln(out, marketTellingHonestyFooter)
				return
			}
			ok, line := evaluateCadence(doc.Cadence.Mode, doc.Cadence.Volume, doc.Cadence.Floor)
			fmt.Fprintln(out, line)
			if !ok {
				fmt.Fprintln(out, "  daily cron on thin volume refused · insufficient-signal")
			}
			fmt.Fprintln(out, marketTellingHonestyFooter)
			return
		}
		opts, errMsg := parseMarketTellingCadenceArgs(args[1:])
		if errMsg != "" {
			fmt.Fprintf(out, "gtm brief cadence: %s\n%s\n", errMsg, marketTellingHelp())
			return
		}
		msg, err := setCadence(opts.Mode, opts.Volume)
		if err != nil {
			fmt.Fprintf(out, "gtm brief cadence: %v\n", err)
			return
		}
		fmt.Fprintln(out, msg)
	case "recipe":
		if len(args) == 1 {
			doc, exists, err := loadMarketTelling()
			if err != nil {
				fmt.Fprintf(out, "gtm brief recipe: %v\n", err)
				return
			}
			r := defaultSupportThemeRecipe()
			if exists && doc.Recipe != nil {
				r = *doc.Recipe
			}
			var b strings.Builder
			b.WriteString("recipe support_theme · same metadata as incidents · <=3 first-party sources\n")
			b.WriteString(formatRecipeLines(r))
			b.WriteString("  hands off (win-back, price change) · no Slack persist · CRM ≠ Connected\n")
			b.WriteString(marketTellingHonestyFooter)
			fmt.Fprintln(out, b.String())
			return
		}
		opts, errMsg := parseMarketTellingRecipeArgs(args[1:])
		if errMsg != "" {
			fmt.Fprintf(out, "gtm brief recipe: %s\n%s\n", errMsg, marketTellingHelp())
			return
		}
		msg, err := writeRecipe(opts)
		if err != nil {
			fmt.Fprintf(out, "gtm brief recipe: %v\n", err)
			return
		}
		fmt.Fprintln(out, msg)
	default:
		fmt.Fprintf(out, "gtm brief: unknown subcommand %q\n%s\n", args[0], marketTellingHelp())
	}
}

type marketTellingWriteOpts struct {
	Kind       string
	Hypothesis string
	Falsify    string
	Confidence float64
	confSet    bool
}

func takeFlagVal(args []string, i int, hasEq bool, val string) (string, int, string) {
	if hasEq {
		return val, i, ""
	}
	if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
		return args[i+1], i + 1, ""
	}
	return "", i, "missing value"
}

func parseMarketTellingWriteArgs(args []string) (opts marketTellingWriteOpts, errMsg string) {
	opts.Kind = MarketTellingKind
	for i := 0; i < len(args); i++ {
		a := args[i]
		key, val, hasEq := splitFlagKV(a)
		switch key {
		case "--kind", "-k":
			val, i, errMsg = takeFlagVal(args, i, hasEq, val)
			if errMsg != "" {
				return opts, "missing --kind"
			}
			k, kerr := normalizeMarketTellingKind(val)
			if kerr != "" {
				return opts, kerr
			}
			opts.Kind = k
		case "--hypothesis", "--hyp", "-H":
			val, i, errMsg = takeFlagVal(args, i, hasEq, val)
			if errMsg != "" {
				return opts, "missing --hypothesis"
			}
			opts.Hypothesis = val
		case "--falsify", "--falsification", "--test":
			val, i, errMsg = takeFlagVal(args, i, hasEq, val)
			if errMsg != "" {
				return opts, "missing --falsify"
			}
			opts.Falsify = val
		case "--confidence", "--conf", "-c":
			val, i, errMsg = takeFlagVal(args, i, hasEq, val)
			if errMsg != "" {
				return opts, "missing --confidence"
			}
			n, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
			if err != nil {
				return opts, "invalid --confidence"
			}
			opts.Confidence = n
			opts.confSet = true
		default:
			if strings.HasPrefix(a, "-") {
				return opts, "unknown flag " + a
			}
			return opts, "unexpected argument " + a
		}
	}
	if strings.TrimSpace(opts.Hypothesis) == "" {
		return opts, "missing --hypothesis"
	}
	if strings.TrimSpace(opts.Falsify) == "" {
		return opts, "missing --falsify"
	}
	if !opts.confSet {
		return opts, "missing --confidence"
	}
	return opts, ""
}

type marketTellingLedgerOpts struct {
	Status        string
	ID            string
	Hypothesis    string
	Yesterday     string
	Contradiction string
}

func parseMarketTellingLedgerArgs(args []string) (opts marketTellingLedgerOpts, errMsg string) {
	var positionals []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		key, val, hasEq := splitFlagKV(a)
		switch key {
		case "--hypothesis", "--hyp", "-H":
			val, i, errMsg = takeFlagVal(args, i, hasEq, val)
			if errMsg != "" {
				return opts, "missing --hypothesis"
			}
			opts.Hypothesis = val
		case "--vs-yesterday", "--yesterday", "--vs":
			val, i, errMsg = takeFlagVal(args, i, hasEq, val)
			if errMsg != "" {
				return opts, "missing --vs-yesterday"
			}
			opts.Yesterday = val
		case "--contradiction", "--contra":
			val, i, errMsg = takeFlagVal(args, i, hasEq, val)
			if errMsg != "" {
				return opts, "missing --contradiction"
			}
			opts.Contradiction = val
		default:
			if strings.HasPrefix(a, "-") {
				return opts, "unknown flag " + a
			}
			positionals = append(positionals, a)
		}
	}
	if len(positionals) < 2 {
		return opts, "want shipped|moved|killed|falsified <id>"
	}
	opts.Status = positionals[0]
	opts.ID = positionals[1]
	if len(positionals) > 2 {
		return opts, "unexpected argument " + positionals[2]
	}
	return opts, ""
}

type marketTellingCadenceOpts struct {
	Mode   string
	Volume int
}

func parseMarketTellingCadenceArgs(args []string) (opts marketTellingCadenceOpts, errMsg string) {
	var positionals []string
	volSet := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		key, val, hasEq := splitFlagKV(a)
		switch key {
		case "--volume", "--vol", "-n":
			val, i, errMsg = takeFlagVal(args, i, hasEq, val)
			if errMsg != "" {
				return opts, "missing --volume"
			}
			n, err := strconv.Atoi(strings.TrimSpace(val))
			if err != nil || n < 0 {
				return opts, "invalid --volume"
			}
			opts.Volume = n
			volSet = true
		default:
			if strings.HasPrefix(a, "-") {
				return opts, "unknown flag " + a
			}
			positionals = append(positionals, a)
		}
	}
	if len(positionals) != 1 {
		return opts, "want daily|weekly|on_threshold"
	}
	opts.Mode = positionals[0]
	if !volSet {
		opts.Volume = 0
	}
	return opts, ""
}

func parseMarketTellingRecipeArgs(args []string) (opts PalaceRevOpsRecipe, errMsg string) {
	opts = defaultSupportThemeRecipe()
	for i := 0; i < len(args); i++ {
		a := args[i]
		key, val, hasEq := splitFlagKV(a)
		switch key {
		case "--id":
			val, i, errMsg = takeFlagVal(args, i, hasEq, val)
			if errMsg != "" {
				return opts, "missing --id"
			}
			opts.ID = val
		case "--summary":
			val, i, errMsg = takeFlagVal(args, i, hasEq, val)
			if errMsg != "" {
				return opts, "missing --summary"
			}
			opts.Summary = val
		case "--pointer":
			val, i, errMsg = takeFlagVal(args, i, hasEq, val)
			if errMsg != "" {
				return opts, "missing --pointer"
			}
			opts.Pointer = val
		case "--event-time", "--event_time":
			val, i, errMsg = takeFlagVal(args, i, hasEq, val)
			if errMsg != "" {
				return opts, "missing --event-time"
			}
			opts.EventTime = val
		case "--subject":
			val, i, errMsg = takeFlagVal(args, i, hasEq, val)
			if errMsg != "" {
				return opts, "missing --subject"
			}
			opts.Subject = val
		case "--account-hash", "--account_hash":
			val, i, errMsg = takeFlagVal(args, i, hasEq, val)
			if errMsg != "" {
				return opts, "missing --account-hash"
			}
			opts.AccountHash = val
		case "--source-hint", "--source_hint":
			val, i, errMsg = takeFlagVal(args, i, hasEq, val)
			if errMsg != "" {
				return opts, "missing --source-hint"
			}
			opts.SourceHint = val
		case "--sources":
			val, i, errMsg = takeFlagVal(args, i, hasEq, val)
			if errMsg != "" {
				return opts, "missing --sources"
			}
			srcs, err := parseFirstPartySources(val)
			if err != nil {
				return opts, err.Error()
			}
			opts.Sources = srcs
		case "--name", "--kind":
			val, i, errMsg = takeFlagVal(args, i, hasEq, val)
			if errMsg != "" {
				return opts, "missing --name"
			}
			opts.Name = val
			opts.Kind = val
		default:
			if strings.HasPrefix(a, "-") {
				return opts, "unknown flag " + a
			}
			return opts, "unexpected argument " + a
		}
	}
	return opts, ""
}
