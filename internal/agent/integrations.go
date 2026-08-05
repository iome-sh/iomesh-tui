package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// MCP tool names for residual-honest agent connector setup
// (s1238 TUI · peer aion s1237 v178 · s1242 wire parity · s1243 signing).
// Not install CRUD · not OAuth complete · not checklist/API-key mint · not secret mint/rotate.
const (
	mcpToolListConnectorCatalog     = "list_connector_catalog"
	mcpToolPlanConnectorSetup       = "plan_connector_setup"
	mcpToolGetWebhookSigningHeaders = "get_webhook_signing_headers"
)

// Integrations offline / residual honesty copy (fail-open when MCP or tools missing).
// Browser HITL for OAuth complete · stub ≠ live · dual_write OFF · no invent GA ·
// catalog Beta honesty · never invent install green.
const (
	integrationsPortalURL = "https://console.iome.sh/integrations"
	// IntegrationsHonestyOneLiner is the bare /integrations status line.
	IntegrationsHonestyOneLiner = "agent setup = catalog + plan + portal HITL · not full install CRUD · stub ≠ live · dual_write OFF · catalog Beta honesty · never invent install green"
)

// IntegrationsOfflineMessage is printed when MCP manager is missing or tools are not connected.
func IntegrationsOfflineMessage() string {
	return strings.TrimSpace(`integrations: MCP connector tools unavailable (fail-open).
  portal HITL: ` + integrationsPortalURL + `
  aion MCP tools list_connector_catalog / plan_connector_setup (v178/s1237) · get_webhook_signing_headers (v30) · TUI wire parity s1242/s1243.
  ` + IntegrationsHonestyOneLiner)
}

// IntegrationsToolMissingMessage is printed when MCP is up but the named tool is absent.
func IntegrationsToolMissingMessage(tool string) string {
	return fmt.Sprintf(strings.TrimSpace(`integrations: MCP tool %q not found on any connected server (fail-open).
  portal HITL: %s
  aion MCP tools list_connector_catalog / plan_connector_setup (v178) · get_webhook_signing_headers (v30).
  %s`), tool, integrationsPortalURL, IntegrationsHonestyOneLiner)
}

// IntegrationsCatalog lists connector catalog via MCP list_connector_catalog (s1238/s1242).
// meshLayer optional filter: operational|knowledge|analytical (empty = all).
// Fail-open: when MCP/tool unavailable returns residual-honest offline guidance (nil error).
// Never invents install green; catalog status is display-only honesty (Beta/planned/available).
//
// aion v178 wire: {"count":N,"entries":[{id,label,status,mesh_layer,ingress_type,
// webhook_path,summary,oauth_install_supported,portal_path}]}.
func (rt *Runtime) IntegrationsCatalog(ctx context.Context, meshLayer string) (string, error) {
	layer := strings.ToLower(strings.TrimSpace(meshLayer))
	switch layer {
	case "", "operational", "knowledge", "analytical", "all":
		// ok; "all" and empty omit filter
	default:
		return "", fmt.Errorf("invalid mesh_layer %q (operational|knowledge|analytical)", meshLayer)
	}
	if layer == "all" {
		layer = ""
	}

	args := map[string]any{}
	if layer != "" {
		args["mesh_layer"] = layer
	}

	raw, err := rt.callMCPToolByName(ctx, mcpToolListConnectorCatalog, args)
	if err != nil {
		if isMCPUnavailable(err) {
			return IntegrationsOfflineMessage(), nil
		}
		if isMCPToolMissing(err) {
			return IntegrationsToolMissingMessage(mcpToolListConnectorCatalog), nil
		}
		// Soft fail-open: show residual note + error detail (do not invent catalog rows).
		return fmt.Sprintf("%s\n  detail: %v", IntegrationsOfflineMessage(), err), nil
	}
	if formatted := formatConnectorCatalog(raw, layer); formatted != "" {
		return formatted, nil
	}
	// Pass-through when not JSON; append honesty footer.
	out := strings.TrimSpace(raw)
	if out == "" {
		return "integrations catalog: (empty)\n" + catalogHonestyFooter(), nil
	}
	return out + "\n" + catalogHonestyFooter(), nil
}

// IntegrationsPlan plans connector setup via MCP plan_connector_setup (s1238/s1242).
// Surfaces portal_url, oauth_mode_hint, signing_headers_tool, next_steps, honesty.notes.
// Never invents install green. Fail-open when MCP/tool unavailable.
//
// aion v178 wire: {connector_id, org_id, connector, portal_url, oauth_install_supported,
// oauth_mode_hint, signing_headers_tool, next_steps, honesty:{…, notes:[]}}.
func (rt *Runtime) IntegrationsPlan(ctx context.Context, connectorID string) (string, error) {
	id := strings.TrimSpace(connectorID)
	if id == "" {
		return "", fmt.Errorf("connector_id required")
	}
	args := map[string]any{
		"connector_id": id,
	}
	raw, err := rt.callMCPToolByName(ctx, mcpToolPlanConnectorSetup, args)
	if err != nil {
		if isMCPUnavailable(err) {
			return IntegrationsOfflineMessage(), nil
		}
		if isMCPToolMissing(err) {
			return IntegrationsToolMissingMessage(mcpToolPlanConnectorSetup), nil
		}
		return fmt.Sprintf("%s\n  detail: %v", IntegrationsOfflineMessage(), err), nil
	}
	if formatted := formatConnectorPlan(raw, id); formatted != "" {
		return formatted, nil
	}
	out := strings.TrimSpace(raw)
	if out == "" {
		return fmt.Sprintf("integrations plan %s: (empty)\n%s", id, planHonestyFooter()), nil
	}
	return out + "\n" + planHonestyFooter(), nil
}

// IntegrationsSigning discovers webhook signing header parity via MCP
// get_webhook_signing_headers (s1243 · aion v30).
//
// meshLayerOrConnector: optional mesh_layer (operational|knowledge|analytical) or a
// connector id for client-side filter. Empty = full catalog.
// Discovery only — does not mint or rotate secrets. Fail-open offline.
func (rt *Runtime) IntegrationsSigning(ctx context.Context, meshLayerOrConnector string) (string, error) {
	hint := strings.ToLower(strings.TrimSpace(meshLayerOrConnector))
	args := map[string]any{}
	clientFilterID := ""
	switch hint {
	case "", "all":
		// full catalog
	case "operational", "knowledge", "analytical":
		args["mesh_layer"] = hint
	default:
		// Treat as connector_id hint — aion input is mesh_layer only; filter client-side.
		clientFilterID = hint
	}

	raw, err := rt.callMCPToolByName(ctx, mcpToolGetWebhookSigningHeaders, args)
	if err != nil {
		if isMCPUnavailable(err) {
			return IntegrationsOfflineMessage(), nil
		}
		if isMCPToolMissing(err) {
			return IntegrationsToolMissingMessage(mcpToolGetWebhookSigningHeaders), nil
		}
		return fmt.Sprintf("%s\n  detail: %v", IntegrationsOfflineMessage(), err), nil
	}
	if formatted := formatWebhookSigning(raw, hint, clientFilterID); formatted != "" {
		return formatted, nil
	}
	out := strings.TrimSpace(raw)
	if out == "" {
		return "integrations signing: (empty)\n" + signingHonestyFooter(), nil
	}
	return out + "\n" + signingHonestyFooter(), nil
}

// IntegrationsStatus is the residual-honest operator pulse for /integrations status (s1247).
//
// Reports MCP path availability, presence of list/plan/signing tools (lightweight probe —
// same discovery as callMCPToolByName, no invent), optional catalog count + per-mesh_layer
// counts when list_connector_catalog works, and always an honesty footer.
//
// Hard residual rules:
//   - NEVER invent org install Connected / INSTALL_STORE green / GA
//   - Catalog count ≠ install count (label as catalog honesty only)
//   - Offline fail-open preserved
//   - dual_write OFF · book-demo OFF · stub ≠ live · browser HITL · signing discovery only
func (rt *Runtime) IntegrationsStatus(ctx context.Context) (string, error) {
	var b strings.Builder
	b.WriteString("integrations status (s1247 residual-honest operator pulse)\n")

	// 1) MCP path
	pathState, nServers := rt.mcpPathState()
	switch pathState {
	case "available":
		fmt.Fprintf(&b, "MCP path:     available (%d server(s))\n", nServers)
	case "empty":
		b.WriteString("MCP path:     connected-empty (manager present, 0 servers) · fail-open\n")
	default:
		b.WriteString("MCP path:     offline (no MCP manager/clients) · fail-open\n")
	}

	// 2) Tools present (probe carefully — list bindings/tools only; do not invent)
	tools := []string{
		mcpToolListConnectorCatalog,
		mcpToolPlanConnectorSetup,
		mcpToolGetWebhookSigningHeaders,
	}
	b.WriteString("tools:\n")
	listState := ""
	for _, tool := range tools {
		st := rt.mcpToolPresence(tool)
		if tool == mcpToolListConnectorCatalog {
			listState = st
		}
		fmt.Fprintf(&b, "  %-30s %s\n", tool+":", st)
	}

	// 3) Catalog pulse only when list tool is present (call + parse; honesty labeled)
	b.WriteString("catalog pulse:\n")
	if listState != "present" {
		fmt.Fprintf(&b, "  (skipped — list_connector_catalog is %s; no invent counts)\n", listState)
		b.WriteString("  note: catalog status Beta/available/planned is NOT install Connected\n")
	} else {
		raw, err := rt.callMCPToolByName(ctx, mcpToolListConnectorCatalog, map[string]any{})
		if err != nil {
			// Soft fail-open: do not invent counts on call error.
			fmt.Fprintf(&b, "  (list call failed — no invent counts) detail: %v\n", err)
			b.WriteString("  note: catalog status Beta/available/planned is NOT install Connected\n")
		} else if pulse := formatCatalogPulse(raw); pulse != "" {
			b.WriteString(pulse)
		} else {
			b.WriteString("  (list returned non-JSON/empty — no invent counts)\n")
			b.WriteString("  note: catalog status Beta/available/planned is NOT install Connected\n")
		}
	}

	// 4) Honesty footer always
	b.WriteString(statusHonestyFooter())
	return strings.TrimSpace(b.String()), nil
}

// mcpPathState reports whether the MCP call path is usable.
// available | empty | offline
func (rt *Runtime) mcpPathState() (state string, nServers int) {
	if rt == nil || rt.mcp == nil {
		return "offline", 0
	}
	n := rt.mcp.Len()
	if n == 0 {
		return "empty", 0
	}
	return "available", n
}

// mcpToolPresence returns present | missing | offline for a bare MCP tool name.
// Uses the same discovery as callMCPToolByName (bindings, then each client's Tools list).
// Does not invoke the tool.
func (rt *Runtime) mcpToolPresence(tool string) string {
	if rt == nil || rt.mcp == nil || rt.mcp.Len() == 0 {
		return "offline"
	}
	tool = strings.TrimSpace(tool)
	if tool == "" {
		return "missing"
	}
	if rt.hasMCPTool(tool) {
		return "present"
	}
	return "missing"
}

// hasMCPTool reports whether a bare tool name is bound or listed on any connected client.
func (rt *Runtime) hasMCPTool(tool string) bool {
	if rt == nil || rt.mcp == nil {
		return false
	}
	tool = strings.TrimSpace(tool)
	if tool == "" {
		return false
	}
	for _, b := range rt.mcp.Bindings() {
		if b.Tool == tool && b.Client != nil {
			return true
		}
	}
	for _, c := range rt.mcp.Clients() {
		if c == nil {
			continue
		}
		for _, t := range c.Tools() {
			if t.Name == tool {
				return true
			}
		}
	}
	return false
}

// callMCPToolByName finds a bare MCP tool name across connected servers and invokes it.
// Prefer Manager bindings; fall back to scanning each client's Tools list.
func (rt *Runtime) callMCPToolByName(ctx context.Context, tool string, args map[string]any) (string, error) {
	if rt == nil || rt.mcp == nil || rt.mcp.Len() == 0 {
		return "", errMCPUnavailable
	}
	tool = strings.TrimSpace(tool)
	if tool == "" {
		return "", errMCPToolMissing
	}
	for _, b := range rt.mcp.Bindings() {
		if b.Tool == tool && b.Client != nil {
			return b.Client.CallTool(ctx, tool, args)
		}
	}
	for _, c := range rt.mcp.Clients() {
		if c == nil {
			continue
		}
		for _, t := range c.Tools() {
			if t.Name == tool {
				return c.CallTool(ctx, tool, args)
			}
		}
	}
	return "", errMCPToolMissing
}

var (
	errMCPUnavailable = fmt.Errorf("mcp unavailable")
	errMCPToolMissing = fmt.Errorf("mcp tool missing")
)

func isMCPUnavailable(err error) bool {
	return err != nil && (err == errMCPUnavailable || strings.Contains(err.Error(), "mcp unavailable"))
}

func isMCPToolMissing(err error) bool {
	return err != nil && (err == errMCPToolMissing || strings.Contains(err.Error(), "mcp tool missing"))
}

// --- formatting (JSON fail-open → compact table / plan block) ---

// connectorCatalogItem matches aion v178 ConnectorCatalogEntry (+ legacy aliases).
type connectorCatalogItem struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Status    string `json:"status"`
	MeshLayer string `json:"mesh_layer"`
	// OAuthInstallSupported is aion v178 wire (bool). Pointer distinguishes absent vs false.
	OAuthInstallSupported *bool `json:"oauth_install_supported"`
	// OAuth is a legacy any-shaped field (bool or string) for pre-v178 servers.
	OAuth any `json:"oauth"`
	// IngressType / ingress_type when oauth bool absent.
	IngressType string `json:"ingress_type"`
	WebhookPath string `json:"webhook_path"`
	Summary     string `json:"summary"`
	PortalPath  string `json:"portal_path"`
}

// connectorCatalogPayload accepts aion v178 {count,entries} and legacy keys.
type connectorCatalogPayload struct {
	Count      int                    `json:"count"`
	Entries    []connectorCatalogItem `json:"entries"` // aion v178
	Connectors []connectorCatalogItem `json:"connectors"`
	Items      []connectorCatalogItem `json:"items"`
	Catalog    []connectorCatalogItem `json:"catalog"`
	Honesty    any                    `json:"honesty"`
	Note       string                 `json:"note"`
}

func formatConnectorCatalog(raw, layerFilter string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || (raw[0] != '{' && raw[0] != '[') {
		return ""
	}
	var items []connectorCatalogItem
	if raw[0] == '[' {
		if err := json.Unmarshal([]byte(raw), &items); err != nil {
			return ""
		}
	} else {
		var p connectorCatalogPayload
		if err := json.Unmarshal([]byte(raw), &p); err != nil {
			return ""
		}
		// Prefer aion v178 "entries", then legacy keys.
		items = p.Entries
		if len(items) == 0 {
			items = p.Connectors
		}
		if len(items) == 0 {
			items = p.Items
		}
		if len(items) == 0 {
			items = p.Catalog
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "integrations catalog")
	if layerFilter != "" {
		fmt.Fprintf(&b, " layer=%s", layerFilter)
	}
	b.WriteByte('\n')
	if len(items) == 0 {
		b.WriteString("(no connectors)\n")
		b.WriteString(catalogHonestyFooter())
		return b.String()
	}
	fmt.Fprintf(&b, "%-20s %-12s %-14s %s\n", "ID", "STATUS", "MESH_LAYER", "OAUTH")
	shown := 0
	for _, it := range items {
		id := strings.TrimSpace(it.ID)
		if id == "" {
			id = strings.TrimSpace(it.Label)
		}
		if id == "" {
			continue
		}
		layer := strings.ToLower(strings.TrimSpace(it.MeshLayer))
		if layerFilter != "" && layer != "" && layer != layerFilter {
			continue
		}
		status := strings.TrimSpace(it.Status)
		if status == "" {
			status = "-"
		}
		if layer == "" {
			layer = "-"
		}
		oauth := oauthYesNoFromItem(it)
		fmt.Fprintf(&b, "%-20s %-12s %-14s %s\n",
			truncateRunes(id, 20), truncateRunes(status, 12), truncateRunes(layer, 14), oauth)
		shown++
		if shown >= 80 {
			fmt.Fprintf(&b, "… (%d more not shown)\n", len(items)-shown)
			break
		}
	}
	if shown == 0 {
		b.WriteString("(no connectors match filter)\n")
	}
	b.WriteString(catalogHonestyFooter())
	return b.String()
}

func oauthYesNoFromItem(it connectorCatalogItem) string {
	if it.OAuthInstallSupported != nil {
		if *it.OAuthInstallSupported {
			return "yes"
		}
		return "no"
	}
	return oauthYesNo(it.OAuth, it.IngressType)
}

func oauthYesNo(oauth any, ingressType string) string {
	switch v := oauth.(type) {
	case bool:
		if v {
			return "yes"
		}
		return "no"
	case string:
		s := strings.ToLower(strings.TrimSpace(v))
		if s == "true" || s == "yes" || s == "oauth" {
			return "yes"
		}
		if s == "false" || s == "no" {
			return "no"
		}
	}
	it := strings.ToLower(strings.TrimSpace(ingressType))
	if it == "oauth" {
		return "yes"
	}
	if it != "" {
		return "no"
	}
	return "-"
}

// connectorPlanPayload matches aion v178 plan_connector_setup (+ legacy aliases).
type connectorPlanPayload struct {
	ConnectorID           string                `json:"connector_id"`
	ID                    string                `json:"id"`
	OrgID                 string                `json:"org_id"`
	Connector             *connectorCatalogItem `json:"connector"`
	PortalURL             string                `json:"portal_url"`
	URL                   string                `json:"url"`
	OAuthInstallSupported *bool                 `json:"oauth_install_supported"`
	OAuthModeHint         string                `json:"oauth_mode_hint"`
	SigningHeadersTool    string                `json:"signing_headers_tool"`
	NextSteps             []string              `json:"next_steps"`
	Steps                 []string              `json:"steps"`
	Honesty               any                   `json:"honesty"`
	HonestyNotes          []string              `json:"honesty_notes"`
	Note                  string                `json:"note"`
	Status                string                `json:"status"`
}

func formatConnectorPlan(raw, requestedID string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw[0] != '{' {
		return ""
	}
	var p connectorPlanPayload
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return ""
	}
	id := firstNonEmpty(p.ConnectorID, p.ID, requestedID)
	if p.Connector != nil && strings.TrimSpace(p.Connector.ID) != "" {
		if id == "" || id == requestedID {
			id = firstNonEmpty(p.Connector.ID, id)
		}
	}
	portal := firstNonEmpty(p.PortalURL, p.URL)
	if portal == "" {
		// Residual default: portal detail deep-link (HITL; not install green).
		portal = integrationsPortalURL + "/" + id
	}
	steps := p.NextSteps
	if len(steps) == 0 {
		steps = p.Steps
	}

	// Status from nested connector when top-level absent (display-only; not install green).
	status := strings.TrimSpace(p.Status)
	if status == "" && p.Connector != nil {
		status = strings.TrimSpace(p.Connector.Status)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "integrations plan connector=%s\n", id)
	if status != "" {
		fmt.Fprintf(&b, "status:     %s  (catalog honesty — not install green)\n", status)
	}
	fmt.Fprintf(&b, "portal_url: %s\n", portal)
	if p.OAuthInstallSupported != nil {
		fmt.Fprintf(&b, "oauth_install_supported: %v\n", *p.OAuthInstallSupported)
	}
	if hint := strings.TrimSpace(p.OAuthModeHint); hint != "" {
		fmt.Fprintf(&b, "oauth_mode_hint: %s  (stub ≠ live · do not invent install green)\n", hint)
	}
	if tool := strings.TrimSpace(p.SigningHeadersTool); tool != "" {
		fmt.Fprintf(&b, "signing_headers_tool: %s  (discovery only · /integrations signing)\n", tool)
	}
	b.WriteString("next_steps:\n")
	if len(steps) == 0 {
		b.WriteString("  - Open portal_url and complete setup in browser (OAuth / webhook HITL)\n")
		b.WriteString("  - Do not treat this plan as Connected/install success\n")
	} else {
		for i, s := range steps {
			if i >= 20 {
				fmt.Fprintf(&b, "  … +%d more\n", len(steps)-20)
				break
			}
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			fmt.Fprintf(&b, "  %d. %s\n", i+1, s)
		}
	}
	// Honesty notes from tool (never invent).
	notes := honestyNotes(p.Honesty, p.HonestyNotes, p.Note)
	b.WriteString("honesty:\n")
	if len(notes) == 0 {
		b.WriteString("  - " + planHonestyFooter() + "\n")
	} else {
		for _, n := range notes {
			fmt.Fprintf(&b, "  - %s\n", n)
		}
		// Always pin residual footer even when server sent notes.
		fmt.Fprintf(&b, "  - %s\n", IntegrationsHonestyOneLiner)
	}
	return b.String()
}

func honestyNotes(honesty any, notes []string, note string) []string {
	var out []string
	for _, n := range notes {
		if s := strings.TrimSpace(n); s != "" {
			out = append(out, s)
		}
	}
	if s := strings.TrimSpace(note); s != "" {
		out = append(out, s)
	}
	switch v := honesty.(type) {
	case string:
		if s := strings.TrimSpace(v); s != "" {
			out = append(out, s)
		}
	case []any:
		for _, x := range v {
			if s, ok := x.(string); ok {
				if s = strings.TrimSpace(s); s != "" {
					out = append(out, s)
				}
			}
		}
	case []string:
		for _, s := range v {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
	case map[string]any:
		// aion v178: honesty.notes []string (+ residual bool flags).
		if notesRaw, ok := v["notes"]; ok {
			switch ns := notesRaw.(type) {
			case []any:
				for _, x := range ns {
					if s, ok := x.(string); ok {
						if s = strings.TrimSpace(s); s != "" {
							out = append(out, s)
						}
					}
				}
			case []string:
				for _, s := range ns {
					if s = strings.TrimSpace(s); s != "" {
						out = append(out, s)
					}
				}
			}
		}
		// Compact residual flags when notes empty or as short summary of key truths.
		// Prefer named notes; only flatten flags if no notes were extracted from honesty.
		notesFromHonesty := len(out) > 0
		if !notesFromHonesty {
			for _, k := range []string{
				"browser_hitl_required_for_oauth_complete",
				"stub_oauth_not_live",
				"pass_not_invent_install_green",
				"dual_write_off",
				"book_demo_off",
				"no_invent_ga",
				"agent_mcp_cannot_write_installs",
				"session_portal_owns_install_crud",
				"note", "summary", "browser_hitl", "oauth", "dual_write", "never_invent_ga", "stub",
			} {
				if val, ok := v[k]; ok {
					out = append(out, fmt.Sprintf("%s=%v", k, val))
				}
			}
		}
	}
	return out
}

// webhookSigningPayload matches aion v30 get_webhook_signing_headers output.
type webhookSigningPayload struct {
	FleetEnabled bool                  `json:"fleet_enabled"`
	FleetEnvVar  string                `json:"fleet_env_var"`
	Count        int                   `json:"count"`
	Entries      []webhookSigningEntry `json:"entries"`
	// legacy aliases
	Items []webhookSigningEntry `json:"items"`
}

type webhookSigningEntry struct {
	ConnectorID      string   `json:"connector_id"`
	ID               string   `json:"id"`
	Label            string   `json:"label"`
	MeshLayer        string   `json:"mesh_layer"`
	Scheme           string   `json:"scheme"`
	PrimaryHeader    string   `json:"primary_header"`
	AuxiliaryHeaders []string `json:"auxiliary_headers"`
	// SecretEnvVar is server-side env name for operator docs — discovery only, not a secret value.
	SecretEnvVar       string `json:"secret_env_var"`
	SignaturePrefix    string `json:"signature_prefix"`
	DocsURL            string `json:"docs_url"`
	VendorNativeVerify string `json:"vendor_native_verify"`
}

func formatWebhookSigning(raw, layerFilter, clientFilterID string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || (raw[0] != '{' && raw[0] != '[') {
		return ""
	}
	var entries []webhookSigningEntry
	var fleetEnabled bool
	var fleetEnv string
	if raw[0] == '[' {
		if err := json.Unmarshal([]byte(raw), &entries); err != nil {
			return ""
		}
	} else {
		var p webhookSigningPayload
		if err := json.Unmarshal([]byte(raw), &p); err != nil {
			return ""
		}
		entries = p.Entries
		if len(entries) == 0 {
			entries = p.Items
		}
		fleetEnabled = p.FleetEnabled
		fleetEnv = strings.TrimSpace(p.FleetEnvVar)
	}

	// Client-side connector_id filter when hint was not a mesh layer.
	if clientFilterID != "" {
		var filtered []webhookSigningEntry
		for _, e := range entries {
			id := strings.ToLower(firstNonEmpty(e.ConnectorID, e.ID))
			if id == clientFilterID {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	var b strings.Builder
	fmt.Fprintf(&b, "integrations signing")
	if layerFilter != "" {
		fmt.Fprintf(&b, " filter=%s", layerFilter)
	}
	b.WriteByte('\n')
	if fleetEnv != "" {
		fmt.Fprintf(&b, "fleet_env_var: %s  fleet_enabled: %v  (discovery only — not secret mint)\n", fleetEnv, fleetEnabled)
	}
	if len(entries) == 0 {
		b.WriteString("(no signing header entries)\n")
		b.WriteString(signingHonestyFooter())
		return b.String()
	}
	fmt.Fprintf(&b, "%-16s %-12s %-18s %-28s %s\n", "CONNECTOR", "LAYER", "SCHEME", "PRIMARY_HEADER", "PREFIX")
	shown := 0
	for _, e := range entries {
		id := firstNonEmpty(e.ConnectorID, e.ID, e.Label)
		if id == "" {
			continue
		}
		layer := strings.TrimSpace(e.MeshLayer)
		if layer == "" {
			layer = "-"
		}
		scheme := strings.TrimSpace(e.Scheme)
		if scheme == "" {
			scheme = "-"
		}
		header := strings.TrimSpace(e.PrimaryHeader)
		if header == "" {
			header = "-"
		}
		prefix := strings.TrimSpace(e.SignaturePrefix)
		if prefix == "" {
			prefix = "-"
		}
		fmt.Fprintf(&b, "%-16s %-12s %-18s %-28s %s\n",
			truncateRunes(id, 16), truncateRunes(layer, 12), truncateRunes(scheme, 18),
			truncateRunes(header, 28), truncateRunes(prefix, 12))
		shown++
		if shown >= 80 {
			fmt.Fprintf(&b, "… (%d more not shown)\n", len(entries)-shown)
			break
		}
	}
	if shown == 0 {
		b.WriteString("(no signing header entries match filter)\n")
	}
	b.WriteString(signingHonestyFooter())
	return b.String()
}

func catalogHonestyFooter() string {
	return "honesty: " + IntegrationsHonestyOneLiner + " · browser HITL for OAuth · dual_write OFF · no invent GA"
}

func planHonestyFooter() string {
	return "Browser HITL for OAuth complete · stub ≠ live · dual_write OFF · no invent GA · never invent install green · " + IntegrationsHonestyOneLiner
}

func signingHonestyFooter() string {
	return "honesty: discovery only · do not invent secrets mint/rotate · dual_write OFF · book-demo OFF · never invent install green · " + IntegrationsHonestyOneLiner
}

// statusHonestyFooter is always appended to /integrations status (s1247).
func statusHonestyFooter() string {
	return strings.TrimSpace(`honesty:
  never invent install green · browser HITL for OAuth complete · stub ≠ live
  dual_write OFF · book-demo OFF · signing discovery only · no invent GA
  catalog count ≠ install Connected · portal HITL ` + integrationsPortalURL + `
  ` + IntegrationsHonestyOneLiner)
}

// formatCatalogPulse summarizes list_connector_catalog for the status pulse (s1247).
// Labels clearly as catalog honesty — status chips are NOT install Connected.
// Returns empty string when raw is not parseable JSON catalog shape.
func formatCatalogPulse(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || (raw[0] != '{' && raw[0] != '[') {
		return ""
	}
	var items []connectorCatalogItem
	wireCount := -1
	if raw[0] == '[' {
		if err := json.Unmarshal([]byte(raw), &items); err != nil {
			return ""
		}
	} else {
		var p connectorCatalogPayload
		if err := json.Unmarshal([]byte(raw), &p); err != nil {
			return ""
		}
		// Prefer aion v178 "entries", then legacy keys (same order as formatConnectorCatalog).
		items = p.Entries
		if len(items) == 0 {
			items = p.Connectors
		}
		if len(items) == 0 {
			items = p.Items
		}
		if len(items) == 0 {
			items = p.Catalog
		}
		if p.Count > 0 {
			wireCount = p.Count
		}
	}

	// Count entries that have an id (or label fallback); tally mesh_layer.
	layerCounts := map[string]int{}
	n := 0
	for _, it := range items {
		id := strings.TrimSpace(it.ID)
		if id == "" {
			id = strings.TrimSpace(it.Label)
		}
		if id == "" {
			continue
		}
		n++
		layer := strings.ToLower(strings.TrimSpace(it.MeshLayer))
		if layer == "" {
			layer = "(unset)"
		}
		layerCounts[layer]++
	}
	// Prefer parsed entry count; fall back to wire count only when entries empty but count set.
	count := n
	if count == 0 && wireCount > 0 {
		count = wireCount
	}

	var b strings.Builder
	// Hard residual: catalog count is honesty inventory, never install Connected.
	fmt.Fprintf(&b, "  count: %d  (catalog honesty — NOT install Connected / INSTALL_STORE green)\n", count)
	if n > 0 {
		// Stable layer order for operator readability + deterministic tests.
		order := []string{"operational", "knowledge", "analytical"}
		seen := map[string]bool{}
		var parts []string
		for _, l := range order {
			if c, ok := layerCounts[l]; ok {
				parts = append(parts, fmt.Sprintf("%s=%d", l, c))
				seen[l] = true
			}
		}
		var extras []string
		for l := range layerCounts {
			if !seen[l] {
				extras = append(extras, l)
			}
		}
		sort.Strings(extras)
		for _, l := range extras {
			parts = append(parts, fmt.Sprintf("%s=%d", l, layerCounts[l]))
		}
		if len(parts) > 0 {
			fmt.Fprintf(&b, "  by mesh_layer: %s\n", strings.Join(parts, " "))
		}
	}
	b.WriteString("  note: catalog status Beta/available/planned is display honesty only — not Connected\n")
	return b.String()
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

func truncateRunes(s string, max int) string {
	if max <= 0 || s == "" {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 1 {
		return string(r[:max])
	}
	return string(r[:max-1]) + "…"
}
