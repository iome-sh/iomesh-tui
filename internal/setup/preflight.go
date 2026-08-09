package setup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/config"
)

// PreflightReport is always-emit JSON for `iomesh setup preflight --json`.
// Empty strings / false honest; never invent Connected / Memory GA.
type PreflightReport struct {
	OK              bool   `json:"ok"`
	ConfigPath      string `json:"config_path"`
	ConfigPresent   bool   `json:"config_present"`
	DualWrite       bool   `json:"dual_write"`        // must be false for residual-honest local-primary
	DualWriteHonest bool   `json:"dual_write_honest"` // true when dual_write == false
	MCPEnabled      bool   `json:"mcp_enabled"`
	PluginsEnabled  bool   `json:"plugins_enabled"`
	MemoryEnabled   bool   `json:"memory_enabled"`
	// Local memory probe
	MemoryServer     string `json:"memory_server"`
	MemoryURL        string `json:"memory_url"`
	MemoryCommand    string `json:"memory_command"`
	MemoryHealthOK   bool   `json:"memory_health_ok"`
	MemoryHealthBody string `json:"memory_health_body"`
	MemoryHealthErr  string `json:"memory_health_err"`
	MemoryBinaryPATH bool   `json:"memory_binary_on_path"`
	// Mesh
	MeshEnabled  bool   `json:"mesh_enabled"`
	MeshEndpoint string `json:"mesh_endpoint"`
	// Residual copy
	Honesty string   `json:"honesty"`
	Notes   []string `json:"notes"`
	// Setup state hints (not product Connected)
	State string `json:"state"`
}

// Preflight loads config and probes local memory when configured.
// Fail-open: network errors become notes, not invented green.
func Preflight(ctx context.Context, cfgPath string) (*PreflightReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	rep := &PreflightReport{
		Honesty: "dual_write OFF · not Memory GA · catalog ≠ Connected · portal HITL · setup PASS ≠ invent install green",
		Notes:   []string{},
		State:   "not_started",
	}
	path := strings.TrimSpace(cfgPath)
	if path == "" {
		p, err := config.UserConfigPath()
		if err != nil {
			return rep, fmt.Errorf("setup preflight: config path: %w", err)
		}
		path = p
	}
	rep.ConfigPath = path
	if st, err := os.Stat(path); err == nil && !st.IsDir() {
		rep.ConfigPresent = true
	}

	cfg, err := config.Load(path)
	if err != nil {
		return rep, fmt.Errorf("setup preflight: load: %w", err)
	}
	rep.DualWrite = cfg.Memory.DualWrite
	rep.DualWriteHonest = !cfg.Memory.DualWrite
	rep.MCPEnabled = cfg.MCP.Enabled
	rep.PluginsEnabled = cfg.Plugins.Enabled
	rep.MemoryEnabled = cfg.Memory.Enabled
	rep.MeshEnabled = cfg.IOMesh.Enabled
	rep.MeshEndpoint = strings.TrimSpace(cfg.IOMesh.Endpoint)
	rep.MemoryServer = strings.TrimSpace(cfg.Memory.Server)
	if rep.MemoryServer == "" {
		rep.MemoryServer = "memory"
	}

	if !rep.DualWriteHonest {
		rep.Notes = append(rep.Notes, "dual_write is true — residual-honest local-primary prefers dual_write=false (audit only when intentionally enabled)")
	}

	// Find memory server entry
	for _, s := range cfg.MCP.Servers {
		if strings.TrimSpace(s.Name) == rep.MemoryServer ||
			(rep.MemoryServer == "memory" && strings.Contains(s.Name, "memory")) ||
			s.Name == "iomesh-memory-mcp" {
			rep.MemoryURL = strings.TrimSpace(s.URL)
			rep.MemoryCommand = strings.TrimSpace(s.Command)
			if rep.MemoryServer == "memory" || rep.MemoryServer == "" {
				rep.MemoryServer = s.Name
			}
			break
		}
	}
	// Prefer exact match second pass
	for _, s := range cfg.MCP.Servers {
		if s.Name == rep.MemoryServer {
			rep.MemoryURL = strings.TrimSpace(s.URL)
			rep.MemoryCommand = strings.TrimSpace(s.Command)
			break
		}
	}

	if rep.MemoryCommand != "" {
		if _, err := exec.LookPath(rep.MemoryCommand); err == nil {
			rep.MemoryBinaryPATH = true
		} else {
			rep.Notes = append(rep.Notes, fmt.Sprintf("memory command %q not on PATH (fail-open; install iomesh-memory-mcp)", rep.MemoryCommand))
		}
	}
	if strings.Contains(rep.MemoryCommand, "iomesh-memory-mcp") || rep.MemoryServer == "iomesh-memory-mcp" {
		if _, err := exec.LookPath("iomesh-memory-mcp"); err == nil {
			rep.MemoryBinaryPATH = true
		}
	}

	// HTTP healthz when URL present
	if u := rep.MemoryURL; u != "" {
		healthURL := healthzFromMCPURL(u)
		ok, body, herr := probeHealthz(ctx, healthURL)
		rep.MemoryHealthOK = ok
		rep.MemoryHealthBody = body
		if herr != "" {
			rep.MemoryHealthErr = herr
			rep.Notes = append(rep.Notes, "memory healthz: "+herr+" · start iomesh-memory-mcp or docker compose")
		}
	} else if cfg.Memory.Enabled && rep.MemoryCommand == "" {
		rep.Notes = append(rep.Notes, "memory enabled but no url/command for server — run: iomesh setup init local-memory")
	}

	// State synthesis (residual)
	switch {
	case !rep.ConfigPresent:
		rep.State = "not_started"
		rep.Notes = append(rep.Notes, "no config file — run: iomesh setup init local-memory")
	case cfg.Memory.Enabled && rep.MemoryHealthOK:
		rep.State = "local_memory_probe_ok"
	case cfg.Memory.Enabled && (rep.MemoryURL != "" || rep.MemoryCommand != ""):
		rep.State = "awaiting_memory_host"
	case cfg.MCP.Enabled:
		rep.State = "config_written"
	default:
		rep.State = "config_present"
	}

	// ok = dual_write honest + (if memory enabled, probe ok OR stdio binary on path)
	rep.OK = rep.DualWriteHonest
	if cfg.Memory.Enabled {
		if rep.MemoryURL != "" {
			rep.OK = rep.OK && rep.MemoryHealthOK
		} else if rep.MemoryCommand != "" {
			rep.OK = rep.OK && rep.MemoryBinaryPATH
		} else {
			rep.OK = false
		}
	}
	if !rep.ConfigPresent {
		rep.OK = false
	}

	rep.Notes = append(rep.Notes,
		"preflight PASS ≠ invent Connected / INSTALL_STORE green / Memory GA",
		"portal HITL still required for connector OAuth/install",
		"continuous pull: opt-in /setup pull start|once or pull_continuous=true · CLI iomesh memory pull still valid · after mesh + consumer configured · pull ≠ invent Connected",
		"continuous analyze: opt-in /setup analyze start or analyze_continuous=true (default false) · analyze ≠ invent Connected",
		"maintenance drift (report-only): /setup drift · no auto-repair · drift ≠ invent install green",
	)
	return rep, nil
}

func healthzFromMCPURL(mcpURL string) string {
	u := strings.TrimSpace(mcpURL)
	// http://127.0.0.1:8080/mcp → http://127.0.0.1:8080/healthz
	if i := strings.LastIndex(u, "/"); i > 0 {
		base := u[:i]
		return base + "/healthz"
	}
	return strings.TrimSuffix(u, "/mcp") + "/healthz"
}

func probeHealthz(ctx context.Context, url string) (ok bool, body string, errStr string) {
	if url == "" {
		return false, "", "empty healthz url"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, "", err.Error()
	}
	client := &http.Client{Timeout: 3 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return false, "", err.Error()
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
	body = strings.TrimSpace(string(b))
	if res.StatusCode != http.StatusOK {
		return false, body, fmt.Sprintf("status %d", res.StatusCode)
	}
	return true, body, ""
}

// FormatPreflightText human residual-honest report.
func FormatPreflightText(r *PreflightReport) string {
	if r == nil {
		return "setup preflight: nil report\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "setup preflight (residual-honest · not Memory GA · catalog ≠ Connected)\n")
	fmt.Fprintf(&b, "  state: %s\n", r.State)
	fmt.Fprintf(&b, "  ok: %v  (probe path · never invent install green)\n", r.OK)
	fmt.Fprintf(&b, "  config: present=%v path=%s\n", r.ConfigPresent, r.ConfigPath)
	fmt.Fprintf(&b, "  dual_write: %v  honest_off=%v\n", r.DualWrite, r.DualWriteHonest)
	fmt.Fprintf(&b, "  mcp.enabled=%v  plugins.enabled=%v  memory.enabled=%v\n", r.MCPEnabled, r.PluginsEnabled, r.MemoryEnabled)
	fmt.Fprintf(&b, "  memory.server=%q url=%q command=%q\n", r.MemoryServer, r.MemoryURL, r.MemoryCommand)
	fmt.Fprintf(&b, "  memory.health_ok=%v binary_on_path=%v\n", r.MemoryHealthOK, r.MemoryBinaryPATH)
	if r.MemoryHealthErr != "" {
		fmt.Fprintf(&b, "  memory.health_err: %s\n", r.MemoryHealthErr)
	}
	if r.MemoryHealthBody != "" && len(r.MemoryHealthBody) < 500 {
		fmt.Fprintf(&b, "  memory.health_body: %s\n", r.MemoryHealthBody)
	}
	fmt.Fprintf(&b, "  mesh.enabled=%v endpoint=%q\n", r.MeshEnabled, r.MeshEndpoint)
	fmt.Fprintf(&b, "  honesty: %s\n", r.Honesty)
	for _, n := range r.Notes {
		fmt.Fprintf(&b, "  note: %s\n", n)
	}
	return b.String()
}

// FormatPreflightJSON always-emit JSON.
func FormatPreflightJSON(r *PreflightReport) string {
	if r == nil {
		r = &PreflightReport{Notes: []string{}, Honesty: "nil"}
	}
	if r.Notes == nil {
		r.Notes = []string{}
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return `{"ok":false,"error":"marshal"}` + "\n"
	}
	return string(b) + "\n"
}
