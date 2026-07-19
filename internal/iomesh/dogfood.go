package iomesh

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// StepStatus is one dogfood check outcome.
type StepStatus string

const (
	StepPass StepStatus = "PASS"
	StepFail StepStatus = "FAIL"
	StepSkip StepStatus = "SKIP"
)

// Step is one named check in a dogfood report.
type Step struct {
	Name    string        `json:"name"`
	Status  StepStatus    `json:"status"`
	Detail  string        `json:"detail,omitempty"`
	Latency time.Duration `json:"latency,omitempty"`
}

// DogfoodReport aggregates probe results for stage/CI evidence.
type DogfoodReport struct {
	Endpoint string `json:"endpoint"`
	Tenant   string `json:"tenant,omitempty"`
	// Org is optional PlanGate / entitlements org from Client cfg (X-IOMesh-Org).
	// Populated when [iomesh] org / IOMESH_ORG is set; omitted from JSON when empty.
	Org string `json:"org,omitempty"`
	// Workspace is optional workspace scope from Client cfg (X-IOMesh-Workspace).
	// Populated when [iomesh] workspace / IOMESH_WORKSPACE is set; omitted from JSON when empty.
	// Distinct from DogfoodOptions.Workspace (context-plane path).
	Workspace string `json:"workspace,omitempty"`
	// DualWrite is agent [memory].dual_write / IOMESH_MEMORY_DUAL_WRITE from Client cfg.
	// Always emitted in JSON (false when unset) so CI sees dual-write mode without
	// scraping step detail. Does not gate the memory_ingest probe.
	DualWrite bool `json:"dual_write"`
	// MemoryEndpoint is optional memory sidecar base used for sync memory_retrieve.
	// Omitted when empty (retrieve uses mesh Endpoint). Stage warm-plane evidence.
	MemoryEndpoint string `json:"memory_endpoint,omitempty"`
	// UserAgent is the package mesh HTTP User-Agent (iomesh-tui/<version>).
	// Always set from UserAgent() for CI evidence (s290); not scraped from server.
	UserAgent string    `json:"user_agent"`
	Strict    bool      `json:"strict"`
	Steps     []Step    `json:"steps"`
	OK        bool      `json:"ok"`
	Summary   string    `json:"summary"`
	Started   time.Time `json:"started"`
	Finished  time.Time `json:"finished"`
}

// DogfoodOptions tune the probe.
type DogfoodOptions struct {
	// Workspace path for context plane query (optional).
	Workspace string
	// Query string for context plane (default: dogfood probe).
	Query string
	// Strict: context plane and emit failures are FAIL (not SKIP/soft).
	// Health always fails the report when enabled.
	Strict bool
	// SkipContext skips context plane check.
	SkipContext bool
	// SkipEmit skips dept stream emit check.
	SkipEmit bool
	// SkipMemory skips MEMORY_INGEST dual-write publish probe.
	// When false (default), the step runs whenever mesh is enabled (fail-open unless Strict).
	SkipMemory bool
}

// Dogfood runs a stage-oriented mesh smoke:
// health → ready → context → emit → llm_meter → policy → catalog → memory_*.
// Disabled client returns a single SKIP step and OK=true (offline-first).
func (c *Client) Dogfood(ctx context.Context, opts DogfoodOptions) DogfoodReport {
	rep := DogfoodReport{
		Started: time.Now().UTC(),
		Strict:  opts.Strict,
	}
	// Always emit package UA for CI/operator evidence (even when mesh disabled).
	rep.UserAgent = UserAgent()
	if c != nil {
		rep.Endpoint = c.cfg.Endpoint
		rep.Tenant = c.cfg.Tenant
		rep.Org = strings.TrimSpace(c.cfg.OrgID)
		rep.Workspace = strings.TrimSpace(c.cfg.WorkspaceID)
		rep.DualWrite = c.cfg.DualWrite
		rep.MemoryEndpoint = strings.TrimSpace(c.cfg.MemoryEndpoint)
	}
	if opts.Query == "" {
		opts.Query = "iomesh-tui stage mesh dogfood"
	}
	if opts.Workspace == "" {
		opts.Workspace = "."
	}

	if c == nil || !c.Enabled() {
		rep.Steps = append(rep.Steps, Step{
			Name: "enabled", Status: StepSkip,
			Detail: "mesh client disabled or endpoint empty (offline-first OK)",
		})
		rep.OK = true
		rep.Summary = "SKIP (mesh disabled)"
		rep.Finished = time.Now().UTC()
		return rep
	}

	// Stable session id for temporal correlation across emit / llm_meter / memory_* steps.
	tenant := strings.TrimSpace(c.cfg.Tenant)
	sessionID := dogfoodSessionID(tenant)

	rep.Steps = append(rep.Steps, Step{Name: "enabled", Status: StepPass, Detail: "endpoint=" + c.cfg.Endpoint})

	// 1) Health
	rep.Steps = append(rep.Steps, c.stepTimed("health", func() (StepStatus, string) {
		if err := c.Health(ctx); err != nil {
			return StepFail, err.Error()
		}
		return StepPass, "GET /health OK"
	}))

	// 2) Ready (optional path — fail-open unless strict)
	rep.Steps = append(rep.Steps, c.stepTimed("ready", func() (StepStatus, string) {
		err := c.Ready(ctx)
		if err == nil {
			return StepPass, "GET /ready OK"
		}
		// If /ready missing (404), soft-skip unless strict.
		if strings.Contains(err.Error(), "http 404") {
			if opts.Strict {
				return StepFail, err.Error()
			}
			return StepSkip, "GET /ready not found (optional): " + err.Error()
		}
		if opts.Strict {
			return StepFail, err.Error()
		}
		return StepSkip, "ready soft-fail: " + err.Error()
	}))

	// 3) Context plane (optionally lineage-aware)
	if opts.SkipContext || !c.cfg.ContextPlane {
		rep.Steps = append(rep.Steps, Step{Name: "context", Status: StepSkip, Detail: "context plane disabled or skipped"})
	} else {
		rep.Steps = append(rep.Steps, c.stepTimed("context", func() (StepStatus, string) {
			res := c.QueryContext(ctx, opts.Workspace, opts.Query)
			text := FormatContextSnippet(res)
			if text == "" {
				if opts.Strict {
					return StepFail, "empty context (strict)"
				}
				// Fail-open is intentional for agent; dogfood soft-skips unless strict.
				return StepSkip, "empty context (fail-open; use --strict to require)"
			}
			detail := text
			if len(detail) > 120 {
				detail = detail[:117] + "…"
			}
			extra := ""
			if n := len(res.Lineage); n > 0 {
				extra = fmt.Sprintf(" lineage=%d", n)
			}
			return StepPass, fmt.Sprintf("got context (%d chars%s): %s", len(text), extra, detail)
		}))
	}

	// 4) Dept emit (generic dogfood event)
	if opts.SkipEmit || !c.cfg.EmitDeptStreams {
		rep.Steps = append(rep.Steps, Step{Name: "emit", Status: StepSkip, Detail: "dept streams disabled or skipped"})
		rep.Steps = append(rep.Steps, Step{Name: "llm_meter", Status: StepSkip, Detail: "dept streams disabled or skipped"})
	} else {
		rep.Steps = append(rep.Steps, c.stepTimed("emit", func() (StepStatus, string) {
			err := c.EmitErr(ctx, DeptEvent{
				Type:      "dept.agent.dogfood",
				SessionID: sessionID,
				Payload: map[string]any{
					"source":  "iomesh-tui",
					"probe":   "stage-mesh-dogfood",
					"version": "dogfood",
				},
			})
			if err != nil {
				if opts.Strict {
					return StepFail, err.Error()
				}
				return StepSkip, "emit soft-fail: " + err.Error()
			}
			detail := "POST /v1/streams/dept/publish type=dept.agent.dogfood"
			if org := strings.TrimSpace(c.cfg.OrgID); org != "" {
				detail = fmt.Sprintf("%s org=%s", detail, org)
			}
			if ws := strings.TrimSpace(c.cfg.WorkspaceID); ws != "" {
				detail = fmt.Sprintf("%s workspace=%s", detail, ws)
			}
			if sessionID != "" {
				detail = fmt.Sprintf("%s session_id=%s", detail, sessionID)
			}
			return StepPass, detail
		}))

		// 4b) Remote metering path: dept.agent.llm_call (same shape as RecordLLMCall → platform dashboards)
		rep.Steps = append(rep.Steps, c.stepTimed("llm_meter", func() (StepStatus, string) {
			payload := map[string]any{
				"source":   "iomesh-tui",
				"probe":    "stage-mesh-dogfood-llm-meter",
				"model":    "iomesh-tui-dogfood",
				"model_id": "iomesh-tui-dogfood",
				"est_usd":  0.0,
				"tokens": map[string]int{
					"prompt": 0, "completion": 0, "total": 0,
				},
			}
			if tenant != "" {
				payload["tenant"] = tenant
			}
			if org := strings.TrimSpace(c.cfg.OrgID); org != "" {
				payload["org"] = org
			}
			if ws := strings.TrimSpace(c.cfg.WorkspaceID); ws != "" {
				payload["workspace"] = ws
			}
			err := c.EmitErr(ctx, DeptEvent{
				Type:      "dept.agent.llm_call",
				Tenant:    tenant,
				SessionID: sessionID,
				Payload:   payload,
			})
			if err != nil {
				if opts.Strict {
					return StepFail, err.Error()
				}
				return StepSkip, "llm_meter soft-fail: " + err.Error()
			}
			detail := "POST /v1/streams/dept/publish type=dept.agent.llm_call"
			if org := strings.TrimSpace(c.cfg.OrgID); org != "" {
				detail = fmt.Sprintf("%s org=%s", detail, org)
			}
			if ws := strings.TrimSpace(c.cfg.WorkspaceID); ws != "" {
				detail = fmt.Sprintf("%s workspace=%s", detail, ws)
			}
			if sessionID != "" {
				detail = fmt.Sprintf("%s session_id=%s", detail, sessionID)
			}
			return StepPass, detail
		}))
	}

	// 5) Policy evaluate (optional — skip when mode off)
	if c.cfg.PolicyMode == PolicyOff || c.cfg.PolicyMode == "" {
		rep.Steps = append(rep.Steps, Step{Name: "policy", Status: StepSkip, Detail: "policy mode off"})
	} else {
		rep.Steps = append(rep.Steps, c.stepTimed("policy", func() (StepStatus, string) {
			dec := c.EvaluatePolicy(ctx, PolicyInput{
				Action: "dogfood.probe",
				Tool:   "mesh_dogfood",
				Attributes: map[string]any{
					"source": "iomesh-tui",
				},
			})
			detail := dec.Summary()
			if dec.Source == "unavailable" {
				if opts.Strict {
					return StepFail, detail
				}
				return StepSkip, detail
			}
			if dec.Source == "fail-open" {
				if opts.Strict {
					return StepFail, detail
				}
				return StepSkip, detail
			}
			if dec.Source == "mesh" {
				return StepPass, detail
			}
			return StepSkip, detail
		}))
	}

	// 6) Catalog plane
	if !c.cfg.CatalogPlane {
		rep.Steps = append(rep.Steps, Step{Name: "catalog", Status: StepSkip, Detail: "catalog plane disabled"})
	} else {
		rep.Steps = append(rep.Steps, c.stepTimed("catalog", func() (StepStatus, string) {
			res := c.ListCatalog(ctx, "")
			detail := fmt.Sprintf("source=%s n=%d %s", res.Source, len(res.Products), res.Detail)
			switch res.Source {
			case "mesh", "portal":
				return StepPass, detail
			case "fail-open":
				if opts.Strict {
					return StepFail, detail
				}
				return StepSkip, detail
			default:
				return StepSkip, detail
			}
		}))
	}

	// 7) MEMORY_INGEST dual-write (Phase 2 path via PublishMemoryIngest)
	// 8) MEMORY_RPC async recall request (same session_id for temporal correlation — s247)
	// 9) Sync HTTP retrieve POST /v1/memory/retrieve (Phase 3 — request/response hits — s249)
	sessionSeq := 1
	if opts.SkipMemory {
		rep.Steps = append(rep.Steps, Step{Name: "memory_ingest", Status: StepSkip, Detail: "skipped (--skip-memory)"})
		rep.Steps = append(rep.Steps, Step{Name: "memory_recall", Status: StepSkip, Detail: "skipped (--skip-memory)"})
		rep.Steps = append(rep.Steps, Step{Name: "memory_retrieve", Status: StepSkip, Detail: "skipped (--skip-memory)"})
	} else {
		rep.Steps = append(rep.Steps, c.stepTimed("memory_ingest", func() (StepStatus, string) {
			ack, err := c.PublishMemoryIngest(ctx, tenant, MemoryEnvelope{
				Type:       memoryEnvelopeIngest,
				Role:       "tool",
				Content:    "iomesh-tui dual-write dogfood",
				EventTime:  time.Now().UTC().Format(time.RFC3339),
				SessionID:  sessionID,
				SessionSeq: sessionSeq,
			})
			if err != nil {
				if opts.Strict {
					return StepFail, err.Error()
				}
				return StepSkip, "memory_ingest soft-fail: " + err.Error()
			}
			stream := streamMemoryIngest
			subject := tenant + ".memory.ingest.turn"
			if ack != nil {
				if ack.Stream != "" {
					stream = ack.Stream
				}
				if ack.Subject != "" {
					subject = ack.Subject
				}
			}
			detail := fmt.Sprintf("POST /v1/streams/%s/publish subject=%s", stream, subject)
			if ack != nil && ack.Seq > 0 {
				detail = fmt.Sprintf("%s seq=%d", detail, ack.Seq)
			}
			// Operator-visible evidence that dual-write publish used org/workspace headers (s231).
			if org := strings.TrimSpace(c.cfg.OrgID); org != "" {
				detail = fmt.Sprintf("%s org=%s", detail, org)
			}
			if ws := strings.TrimSpace(c.cfg.WorkspaceID); ws != "" {
				detail = fmt.Sprintf("%s workspace=%s", detail, ws)
			}
			// Temporal correlation from the envelope sent (s243): always session_seq; session_id when set.
			detail = fmt.Sprintf("%s session_seq=%d", detail, sessionSeq)
			if sessionID != "" {
				detail = fmt.Sprintf("%s session_id=%s", detail, sessionID)
			}
			// Always emit dual_write mode on PASS detail (s241) so human logs show mode without JSON.
			detail = fmt.Sprintf("%s dual_write=%v", detail, c.cfg.DualWrite)
			return StepPass, detail
		}))

		rep.Steps = append(rep.Steps, c.stepTimed("memory_recall", func() (StepStatus, string) {
			// Fire-and-forget MEMORY_RPC request; same session_id as ingest for correlation (s247).
			ack, err := c.PublishMemoryRecall(ctx, tenant, "iomesh-tui dual-write dogfood", 8, sessionID)
			if err != nil {
				if opts.Strict {
					return StepFail, err.Error()
				}
				return StepSkip, "memory_recall soft-fail: " + err.Error()
			}
			stream := streamMemoryRPC
			subject := tenant + ".memory.retrieve.request"
			if ack != nil {
				if ack.Stream != "" {
					stream = ack.Stream
				}
				if ack.Subject != "" {
					subject = ack.Subject
				}
			}
			detail := fmt.Sprintf("POST /v1/streams/%s/publish subject=%s", stream, subject)
			if ack != nil && ack.Seq > 0 {
				detail = fmt.Sprintf("%s seq=%d", detail, ack.Seq)
			}
			if org := strings.TrimSpace(c.cfg.OrgID); org != "" {
				detail = fmt.Sprintf("%s org=%s", detail, org)
			}
			if ws := strings.TrimSpace(c.cfg.WorkspaceID); ws != "" {
				detail = fmt.Sprintf("%s workspace=%s", detail, ws)
			}
			if sessionID != "" {
				detail = fmt.Sprintf("%s session_id=%s", detail, sessionID)
			}
			detail = fmt.Sprintf("%s dual_write=%v", detail, c.cfg.DualWrite)
			return StepPass, detail
		}))

		rep.Steps = append(rep.Steps, c.stepTimed("memory_retrieve", func() (StepStatus, string) {
			// Sync request/response against memory sidecar HTTP (not MEMORY_RPC).
			// Uses MemoryEndpoint when set (stage warm plane); else mesh Endpoint (broker-only often 404).
			// Empty hits still PASS (HTTP 200 + memories=[]); transport/404 soft-SKIP unless strict.
			res, err := c.RetrieveMemory(ctx, tenant, "iomesh-tui dual-write dogfood", 8, sessionID)
			if err != nil {
				if opts.Strict {
					return StepFail, err.Error()
				}
				return StepSkip, "memory_retrieve soft-fail: " + err.Error()
			}
			path := "/v1/memory/retrieve"
			if res != nil && res.Path != "" {
				path = res.Path
			}
			hits := 0
			if res != nil {
				hits = len(res.Memories)
			}
			detail := fmt.Sprintf("POST %s hits=%d", path, hits)
			if org := strings.TrimSpace(c.cfg.OrgID); org != "" {
				detail = fmt.Sprintf("%s org=%s", detail, org)
			}
			if ws := strings.TrimSpace(c.cfg.WorkspaceID); ws != "" {
				detail = fmt.Sprintf("%s workspace=%s", detail, ws)
			}
			if sessionID != "" {
				detail = fmt.Sprintf("%s session_id=%s", detail, sessionID)
			}
			// memory_base=sidecar when dedicated MemoryEndpoint set; else mesh (same as Endpoint).
			if c.MemoryEndpointConfigured() {
				detail = fmt.Sprintf("%s memory_base=sidecar", detail)
			} else {
				detail = fmt.Sprintf("%s memory_base=mesh", detail)
			}
			detail = fmt.Sprintf("%s dual_write=%v", detail, c.cfg.DualWrite)
			return StepPass, detail
		}))
	}

	// Aggregate: any FAIL ⇒ not OK; SKIP/PASS ok.
	rep.OK = true
	var fails, passes, skips int
	for _, s := range rep.Steps {
		switch s.Status {
		case StepFail:
			rep.OK = false
			fails++
		case StepPass:
			passes++
		case StepSkip:
			skips++
		}
	}
	if rep.OK {
		rep.Summary = fmt.Sprintf("PASS (pass=%d skip=%d)", passes, skips)
	} else {
		rep.Summary = fmt.Sprintf("FAIL (pass=%d fail=%d skip=%d)", passes, fails, skips)
	}
	rep.Finished = time.Now().UTC()
	return rep
}

func (c *Client) stepTimed(name string, fn func() (StepStatus, string)) Step {
	start := time.Now()
	st, detail := fn()
	return Step{Name: name, Status: st, Detail: detail, Latency: time.Since(start)}
}

// Ready checks GET /ready (or /readyz). Error if non-2xx.
func (c *Client) Ready(ctx context.Context) error {
	if !c.Enabled() {
		return nil
	}
	for _, path := range []string{"/ready", "/readyz"} {
		url := strings.TrimRight(c.cfg.Endpoint, "/") + path
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		c.auth(req)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return nil
		}
		if resp.StatusCode == http.StatusNotFound {
			continue
		}
		return fmt.Errorf("iomesh ready: http %d", resp.StatusCode)
	}
	return fmt.Errorf("iomesh ready: http 404")
}

// streamDept is the broker stream name for operational dept.* events
// (POST /v1/streams/dept/publish — same wire as iomesh-client-sdk-go Publish).
const streamDept = "dept"

// EmitErr is like Emit but returns transport/HTTP errors (for dogfood).
// Sets X-IOMesh-Org / X-IOMesh-Workspace when configured (remote multi-tenant metering).
// Wire: POST /v1/streams/dept/publish with base64 JSON DeptEvent payload (aion stream API).
func (c *Client) EmitErr(ctx context.Context, ev DeptEvent) error {
	if !c.Enabled() || !c.cfg.EmitDeptStreams {
		return nil
	}
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now().UTC()
	}
	if ev.Tenant == "" {
		ev.Tenant = c.cfg.Tenant
	}
	if strings.TrimSpace(ev.Type) == "" {
		return fmt.Errorf("iomesh emit: type required")
	}
	raw, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	subject := strings.TrimSpace(ev.Type)
	pubBody := map[string]any{
		"subject": subject,
		"payload": base64.StdEncoding.EncodeToString(raw),
	}
	body, err := json.Marshal(pubBody)
	if err != nil {
		return err
	}
	url := strings.TrimRight(c.cfg.Endpoint, "/") + "/v1/streams/" + streamDept + "/publish"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	c.auth(req)
	req.Header.Set("Content-Type", "application/json")
	c.applyEntitlementHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("iomesh emit: http %d", resp.StatusCode)
	}
	return nil
}

// FormatReportJSON returns the dogfood report as indented JSON (stage CI evidence).
func FormatReportJSON(r DogfoodReport) string {
	type stepJSON struct {
		Name    string `json:"name"`
		Status  string `json:"status"`
		Detail  string `json:"detail,omitempty"`
		Latency string `json:"latency,omitempty"`
	}
	type out struct {
		Endpoint       string     `json:"endpoint"`
		Tenant         string     `json:"tenant,omitempty"`
		Org            string     `json:"org,omitempty"`
		Workspace      string     `json:"workspace,omitempty"`
		DualWrite      bool       `json:"dual_write"` // always emit (CI dual-write mode)
		MemoryEndpoint string     `json:"memory_endpoint,omitempty"`
		UserAgent      string     `json:"user_agent,omitempty"`
		Strict         bool       `json:"strict"`
		OK             bool       `json:"ok"`
		Summary        string     `json:"summary"`
		Started        time.Time  `json:"started"`
		Finished       time.Time  `json:"finished"`
		Steps          []stepJSON `json:"steps"`
		Result         string     `json:"result"` // PASS|FAIL|SKIP mirror of Summary prefix
	}
	o := out{
		Endpoint:       r.Endpoint,
		Tenant:         r.Tenant,
		Org:            r.Org,
		Workspace:      r.Workspace,
		DualWrite:      r.DualWrite,
		MemoryEndpoint: r.MemoryEndpoint,
		UserAgent:      r.UserAgent,
		Strict:         r.Strict,
		OK:             r.OK,
		Summary:        r.Summary,
		Started:        r.Started,
		Finished:       r.Finished,
	}
	if strings.HasPrefix(r.Summary, "PASS") {
		o.Result = "PASS"
	} else if strings.HasPrefix(r.Summary, "FAIL") {
		o.Result = "FAIL"
	} else {
		o.Result = "SKIP"
	}
	for _, s := range r.Steps {
		sj := stepJSON{Name: s.Name, Status: string(s.Status), Detail: s.Detail}
		if s.Latency > 0 {
			sj.Latency = s.Latency.String()
		}
		o.Steps = append(o.Steps, sj)
	}
	b, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		return `{"ok":false,"summary":"json marshal error"}` + "\n"
	}
	return string(b) + "\n"
}

// FormatReport returns a human-readable multi-line dogfood report.
func FormatReport(r DogfoodReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "iomesh mesh dogfood\n")
	fmt.Fprintf(&b, "  endpoint: %s\n", r.Endpoint)
	if r.Tenant != "" {
		fmt.Fprintf(&b, "  tenant:   %s\n", r.Tenant)
	}
	if r.Org != "" {
		fmt.Fprintf(&b, "  org:      %s\n", r.Org)
	}
	if r.Workspace != "" {
		fmt.Fprintf(&b, "  workspace: %s\n", r.Workspace)
	}
	if r.MemoryEndpoint != "" {
		fmt.Fprintf(&b, "  memory_endpoint: %s\n", r.MemoryEndpoint)
	}
	fmt.Fprintf(&b, "  dual_write: %v\n", r.DualWrite)
	if r.UserAgent != "" {
		fmt.Fprintf(&b, "  user_agent: %s\n", r.UserAgent)
	}
	fmt.Fprintf(&b, "  strict:   %v\n", r.Strict)
	for _, s := range r.Steps {
		lat := ""
		if s.Latency > 0 {
			lat = fmt.Sprintf(" (%s)", s.Latency.Round(time.Millisecond))
		}
		fmt.Fprintf(&b, "  %-10s %-4s%s  %s\n", s.Name, s.Status, lat, s.Detail)
	}
	fmt.Fprintf(&b, "RESULT=%s\n", r.Summary)
	return b.String()
}
