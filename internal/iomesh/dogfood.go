package iomesh

import (
	"bytes"
	"context"
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
	Endpoint string    `json:"endpoint"`
	Tenant   string    `json:"tenant,omitempty"`
	Strict   bool      `json:"strict"`
	Steps    []Step    `json:"steps"`
	OK       bool      `json:"ok"`
	Summary  string    `json:"summary"`
	Started  time.Time `json:"started"`
	Finished time.Time `json:"finished"`
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
}

// Dogfood runs a stage-oriented mesh smoke: health → ready → context → emit.
// Disabled client returns a single SKIP step and OK=true (offline-first).
func (c *Client) Dogfood(ctx context.Context, opts DogfoodOptions) DogfoodReport {
	rep := DogfoodReport{
		Started: time.Now().UTC(),
		Strict:  opts.Strict,
	}
	if c != nil {
		rep.Endpoint = c.cfg.Endpoint
		rep.Tenant = c.cfg.Tenant
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

	// 4) Dept emit
	if opts.SkipEmit || !c.cfg.EmitDeptStreams {
		rep.Steps = append(rep.Steps, Step{Name: "emit", Status: StepSkip, Detail: "dept streams disabled or skipped"})
	} else {
		rep.Steps = append(rep.Steps, c.stepTimed("emit", func() (StepStatus, string) {
			err := c.EmitErr(ctx, DeptEvent{
				Type: "dept.agent.dogfood",
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
			return StepPass, "POST /v1/streams/dept OK"
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

// EmitErr is like Emit but returns transport/HTTP errors (for dogfood).
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
	url := strings.TrimRight(c.cfg.Endpoint, "/") + "/v1/streams/dept"
	body, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	c.auth(req)
	req.Header.Set("Content-Type", "application/json")
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
		Endpoint string     `json:"endpoint"`
		Tenant   string     `json:"tenant,omitempty"`
		Strict   bool       `json:"strict"`
		OK       bool       `json:"ok"`
		Summary  string     `json:"summary"`
		Started  time.Time  `json:"started"`
		Finished time.Time  `json:"finished"`
		Steps    []stepJSON `json:"steps"`
		Result   string     `json:"result"` // PASS|FAIL|SKIP mirror of Summary prefix
	}
	o := out{
		Endpoint: r.Endpoint,
		Tenant:   r.Tenant,
		Strict:   r.Strict,
		OK:       r.OK,
		Summary:  r.Summary,
		Started:  r.Started,
		Finished: r.Finished,
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
