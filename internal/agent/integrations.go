package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// MCP tool names for residual-honest agent connector setup (s1238 TUI · peer aion s1237).
// Not install CRUD · not OAuth complete · not checklist/API-key mint.
const (
	mcpToolListConnectorCatalog = "list_connector_catalog"
	mcpToolPlanConnectorSetup   = "plan_connector_setup"
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
  aion MCP tools list_connector_catalog / plan_connector_setup ship in s1237 (concurrent with this TUI path).
  ` + IntegrationsHonestyOneLiner)
}

// IntegrationsToolMissingMessage is printed when MCP is up but the named tool is absent.
func IntegrationsToolMissingMessage(tool string) string {
	return fmt.Sprintf(strings.TrimSpace(`integrations: MCP tool %q not found on any connected server (fail-open).
  portal HITL: %s
  aion MCP tools list_connector_catalog / plan_connector_setup ship in s1237.
  %s`), tool, integrationsPortalURL, IntegrationsHonestyOneLiner)
}

// IntegrationsCatalog lists connector catalog via MCP list_connector_catalog (s1238).
// meshLayer optional filter: operational|knowledge|analytical (empty = all).
// Fail-open: when MCP/tool unavailable returns residual-honest offline guidance (nil error).
// Never invents install green; catalog status is display-only honesty (Beta/planned/available).
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

// IntegrationsPlan plans connector setup via MCP plan_connector_setup (s1238).
// Prints portal_url + next_steps + honesty from the tool; never invents install green.
// Fail-open when MCP/tool unavailable.
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

type connectorCatalogItem struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Status    string `json:"status"`
	MeshLayer string `json:"mesh_layer"`
	// OAuth may arrive as bool or string depending on server wire shape.
	OAuth any `json:"oauth"`
	// IngressType / ingress_type when oauth bool absent.
	IngressType string `json:"ingress_type"`
}

type connectorCatalogPayload struct {
	Connectors []connectorCatalogItem `json:"connectors"`
	// Some servers nest under "items" or "catalog".
	Items   []connectorCatalogItem `json:"items"`
	Catalog []connectorCatalogItem `json:"catalog"`
	Honesty any                    `json:"honesty"`
	Note    string                 `json:"note"`
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
		items = p.Connectors
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
		oauth := oauthYesNo(it.OAuth, it.IngressType)
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

type connectorPlanPayload struct {
	ConnectorID  string   `json:"connector_id"`
	ID           string   `json:"id"`
	PortalURL    string   `json:"portal_url"`
	URL          string   `json:"url"`
	NextSteps    []string `json:"next_steps"`
	Steps        []string `json:"steps"`
	Honesty      any      `json:"honesty"`
	HonestyNotes []string `json:"honesty_notes"`
	Note         string   `json:"note"`
	Status       string   `json:"status"`
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
	portal := firstNonEmpty(p.PortalURL, p.URL)
	if portal == "" {
		// Residual default: portal detail deep-link (HITL; not install green).
		portal = integrationsPortalURL + "/" + id
	}
	steps := p.NextSteps
	if len(steps) == 0 {
		steps = p.Steps
	}

	var b strings.Builder
	fmt.Fprintf(&b, "integrations plan connector=%s\n", id)
	if st := strings.TrimSpace(p.Status); st != "" {
		fmt.Fprintf(&b, "status:     %s  (catalog honesty — not install green)\n", st)
	}
	fmt.Fprintf(&b, "portal_url: %s\n", portal)
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
		// Flatten common honesty object fields into short notes.
		for _, k := range []string{"note", "summary", "browser_hitl", "oauth", "dual_write", "never_invent_ga", "stub"} {
			if val, ok := v[k]; ok {
				out = append(out, fmt.Sprintf("%s=%v", k, val))
			}
		}
	}
	return out
}

func catalogHonestyFooter() string {
	return "honesty: " + IntegrationsHonestyOneLiner + " · browser HITL for OAuth · dual_write OFF · no invent GA"
}

func planHonestyFooter() string {
	return "Browser HITL for OAuth complete · stub ≠ live · dual_write OFF · no invent GA · never invent install green · " + IntegrationsHonestyOneLiner
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
