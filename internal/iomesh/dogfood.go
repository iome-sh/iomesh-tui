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
	// CatalogSource is last catalog probe source (mesh|portal|fail-open|off|"").
	// Set when catalog step runs; empty when mesh disabled before catalog (s292).
	CatalogSource string `json:"catalog_source,omitempty"`
	// CatalogCount is number of products from last ListCatalog (0 when none/off).
	// Always emitted in JSON so CI sees catalog evidence without scraping step detail.
	CatalogCount int `json:"catalog_count"`
	// ContextChars is len of FormatContextSnippet from last context probe (0 if skip/empty).
	// Always emitted in JSON so CI sees context plane evidence without scraping step detail (s296).
	ContextChars int `json:"context_chars"`
	// ContextLineageCount is len(res.Lineage) from last QueryContext (0 if skip/empty).
	// Always emitted in JSON (s296).
	ContextLineageCount int `json:"context_lineage_count"`
	// StreamsCount is len of ListStreams from last streams probe (0 on skip/error/disabled).
	// Always emitted in JSON (s300).
	StreamsCount int `json:"streams_count"`
	// StreamsNames is a short sample of stream names from last ListStreams (max 8).
	// Always emit JSON array (empty when skip/error).
	StreamsNames []string `json:"streams_names"`
	// KVBucket is the bucket used for the soft kv list-keys probe (DogfoodOptions.KVBucket).
	// Omitted from JSON when empty (probe unset / step skipped without a bucket).
	KVBucket string `json:"kv_bucket,omitempty"`
	// KVKeyCount is len of KVListKeys from last kv probe (0 on skip/error/unset).
	// Always emitted in JSON so CI sees kv evidence without scraping step detail.
	KVKeyCount int `json:"kv_key_count"`
	// KVEnsured is true only when KVEnsure was requested, create was attempted, and succeeded
	// (including idempotent 409). Always emitted in JSON (false when unset/skip/soft-fail).
	KVEnsured bool `json:"kv_ensured"`
	// PubProbed is true when DogfoodOptions.PubSubject was set and a Pub attempt ran.
	// Always emitted in JSON (false when subject unset / mesh disabled before step).
	PubProbed bool `json:"pub_probed"`
	// PubOK is true when the soft pub probe succeeded. Always emitted (false when unset/skip/fail).
	PubOK bool `json:"pub_ok"`
	// ConsumerStream is the configured probe stream when both stream+name were provided
	// (set even if create fails). Omitted from JSON when empty (probe unset / partial).
	ConsumerStream string `json:"consumer_stream,omitempty"`
	// ConsumerName is the configured probe consumer name when both stream+name were provided.
	ConsumerName string `json:"consumer_name,omitempty"`
	// ConsumerFilter is the optional filter_subject when set with stream+name.
	ConsumerFilter string `json:"consumer_filter,omitempty"`
	// ConsumerProbed is true when both ConsumerStream and ConsumerName were set and a create attempt ran.
	// Always emitted in JSON (false when unset / partial / mesh disabled before step).
	ConsumerProbed bool `json:"consumer_probed"`
	// ConsumerOK is true when CreateConsumer succeeded (201 or idempotent 409). Always emitted.
	ConsumerOK bool `json:"consumer_ok"`
	// ConsumerFetchOK is true when optional ConsumerFetch ran without error (empty msgs still ok).
	// Always emitted (false when fetch not requested / create failed / fetch error / unset).
	ConsumerFetchOK bool `json:"consumer_fetch_ok"`
	// WaitReadyMS is the configured WaitReady budget in milliseconds (0 = off / no preflight).
	// Always emitted in JSON so CI sees soft preflight budget without scraping step detail.
	// Outcome lives on the wait_ready step (PASS/SKIP/FAIL); not a second boolean.
	WaitReadyMS int `json:"wait_ready_ms"`
	// PolicyMode is the configured policy mode (off|advisory|enforce). Always emitted (default "off").
	PolicyMode string `json:"policy_mode"`
	// PolicySource is last policy probe source (mesh|fail-open|unavailable|off|"").
	// Set when policy step runs; "off" when mode off; empty when mesh disabled before step.
	PolicySource string `json:"policy_source,omitempty"`
	// PolicyAllow is the evaluate decision when policy ran (mesh/fail-open/unavailable).
	// Omitted when mode off / step skipped without evaluation.
	PolicyAllow *bool `json:"policy_allow,omitempty"`
	// MemoryEndpoint is optional memory sidecar base used for sync memory_retrieve.
	// Omitted when empty (retrieve uses mesh Endpoint). Stage warm-plane evidence.
	MemoryEndpoint string `json:"memory_endpoint,omitempty"`
	// UserAgent is the package mesh HTTP User-Agent (iomesh-tui/<version>).
	// Always set from UserAgent() for CI evidence; not scraped from server.
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
	// SkipStreams skips the streams list probe (default false — run when mesh enabled).
	SkipStreams bool
	// KVBucket, when non-empty, runs a soft non-destructive KVListKeys probe on that bucket.
	// Empty (default) → kv step SKIP "kv probe unset" (no network). Put/Delete are CLI-only.
	KVBucket string
	// KVEnsure, when true with KVBucket set, best-effort KVCreateBucket before list-keys
	// (idempotent 409 = success). Soft fail-open: create errors never FAIL the step alone;
	// list-keys still runs. Only meaningful with KVBucket.
	KVEnsure bool
	// PubSubject, when non-empty, runs a soft non-destructive ephemeral Pub after emit/llm_meter
	// with a small fixed JSON payload. Empty (default) → pub step SKIP "pub probe unset".
	PubSubject string
	// ConsumerStream + ConsumerName, when both non-empty, run a soft CreateConsumer probe
	// (201 or idempotent 409 = success). Either empty alone → skip; both empty → "consumer probe unset".
	// Only one set → "consumer probe needs stream and name". Non-destructive relative to ack (no ack).
	ConsumerStream string
	ConsumerName   string
	// ConsumerFilter is optional filter_subject for CreateConsumer.
	ConsumerFilter string
	// ConsumerFetch, when true with stream+name set, after create success runs soft ConsumerFetch
	// with batch=1 and maxWait=500ms (empty message list is PASS). No ack.
	ConsumerFetch bool
	// WaitReady, when >0, polls WaitReady with that max budget (effective deadline is
	// min of this budget and any parent ctx deadline) before the single-shot ready step.
	// Soft: timeout → SKIP wait_ready unless Strict (then FAIL). Zero (default) = no wait preflight.
	WaitReady time.Duration
	// WaitReadyInterval poll interval (default 500ms if WaitReady>0 and this is 0).
	WaitReadyInterval time.Duration
	// WaitRequireHealth if true, WaitReady RequireHealth (Health OK each attempt before Ready).
	WaitRequireHealth bool
}

// Dogfood runs a stage-oriented mesh smoke:
// health → [wait_ready] → ready → context → emit → llm_meter → [pub] → policy → catalog → streams → [consumer] → [kv] → memory_*.
// Optional wait_ready soft-preflight when DogfoodOptions.WaitReady > 0.
// Soft ephemeral Pub probe when DogfoodOptions.PubSubject is set (after emit path).
// Soft streams list probe after catalog when mesh enabled.
// Soft consumer create (+ optional fetch) when ConsumerStream+ConsumerName set (no ack).
// Soft kv list-keys probe when DogfoodOptions.KVBucket is set (non-destructive).
// Optional KVEnsure best-effort creates the bucket before list (soft fail-open).
// Disabled client returns a single SKIP step and OK=true (offline-first).
func (c *Client) Dogfood(ctx context.Context, opts DogfoodOptions) DogfoodReport {
	rep := DogfoodReport{
		Started:    time.Now().UTC(),
		Strict:     opts.Strict,
		PolicyMode: "off",
	}
	if opts.WaitReady > 0 {
		rep.WaitReadyMS = int(opts.WaitReady / time.Millisecond)
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
		mode := strings.ToLower(strings.TrimSpace(string(c.cfg.PolicyMode)))
		if mode == "" {
			mode = "off"
		}
		rep.PolicyMode = mode
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

	// 1b) Optional WaitReady soft preflight (s297) — budget WaitReady, then single-shot ready still runs.
	if opts.WaitReady > 0 {
		interval := opts.WaitReadyInterval
		if interval <= 0 {
			interval = 500 * time.Millisecond
		}
		rep.Steps = append(rep.Steps, c.stepTimed("wait_ready", func() (StepStatus, string) {
			// Child budget: WithTimeout respects parent cancel/deadline (effective min).
			wctx, cancel := context.WithTimeout(ctx, opts.WaitReady)
			defer cancel()
			err := c.WaitReady(wctx, WaitReadyOptions{
				Interval:      interval,
				RequireHealth: opts.WaitRequireHealth,
			})
			if err == nil {
				detail := fmt.Sprintf("WaitReady OK budget=%s interval=%s", opts.WaitReady, interval)
				if opts.WaitRequireHealth {
					detail += " require_health=true"
				}
				return StepPass, detail
			}
			if opts.Strict {
				return StepFail, "wait_ready: " + err.Error()
			}
			return StepSkip, "wait_ready soft-fail: " + err.Error()
		}))
	}

	// 2) Ready (optional path — fail-open unless strict).
	// Always one-shot even after wait_ready PASS for latency evidence consistency.
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
		// ContextChars / ContextLineageCount stay 0 (skip/off evidence, s296).
		rep.Steps = append(rep.Steps, Step{Name: "context", Status: StepSkip, Detail: "context plane disabled or skipped"})
	} else {
		rep.Steps = append(rep.Steps, c.stepTimed("context", func() (StepStatus, string) {
			res := c.QueryContext(ctx, opts.Workspace, opts.Query)
			text := FormatContextSnippet(res)
			rep.ContextChars = len(text)
			rep.ContextLineageCount = len(res.Lineage)
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

	// 4c) Soft ephemeral Pub probe (non-destructive fire-and-forget when PubSubject set)
	pubSubject := strings.TrimSpace(opts.PubSubject)
	if pubSubject == "" {
		rep.PubProbed = false
		rep.PubOK = false
		rep.Steps = append(rep.Steps, Step{Name: "pub", Status: StepSkip, Detail: "pub probe unset"})
	} else {
		rep.Steps = append(rep.Steps, c.stepTimed("pub", func() (StepStatus, string) {
			rep.PubProbed = true
			payload := []byte(`{"source":"iomesh-tui-dogfood"}`)
			err := c.Pub(ctx, pubSubject, payload, nil)
			if err != nil {
				rep.PubOK = false
				if opts.Strict {
					return StepFail, err.Error()
				}
				return StepSkip, "pub soft-fail: " + err.Error()
			}
			rep.PubOK = true
			return StepPass, fmt.Sprintf("POST /v1/pub subject=%s bytes=%d", pubSubject, len(payload))
		}))
	}

	// 5) Policy evaluate (optional — skip when mode off)
	if c.cfg.PolicyMode == PolicyOff || c.cfg.PolicyMode == "" {
		rep.PolicyMode = "off"
		rep.PolicySource = "off"
		// PolicyAllow omitted when mode off (no evaluate).
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
			rep.PolicySource = dec.Source
			if dec.Mode != "" {
				rep.PolicyMode = string(dec.Mode)
			}
			allow := dec.Allow
			rep.PolicyAllow = &allow
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
		rep.CatalogSource = "off"
		rep.CatalogCount = 0
		rep.Steps = append(rep.Steps, Step{Name: "catalog", Status: StepSkip, Detail: "catalog plane disabled"})
	} else {
		rep.Steps = append(rep.Steps, c.stepTimed("catalog", func() (StepStatus, string) {
			res := c.ListCatalog(ctx, "")
			rep.CatalogSource = res.Source
			rep.CatalogCount = len(res.Products)
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

	// 6b) Streams list probe (non-destructive ListStreams — s300; streams_names sample s302)
	if opts.SkipStreams {
		rep.StreamsCount = 0
		rep.StreamsNames = []string{}
		rep.Steps = append(rep.Steps, Step{Name: "streams", Status: StepSkip, Detail: "skipped (--skip-streams)"})
	} else {
		rep.Steps = append(rep.Steps, c.stepTimed("streams", func() (StepStatus, string) {
			streams, err := c.ListStreams(ctx)
			if err != nil {
				rep.StreamsCount = 0
				rep.StreamsNames = []string{}
				if opts.Strict {
					return StepFail, err.Error()
				}
				return StepSkip, "streams soft-fail: " + err.Error()
			}
			rep.StreamsCount = len(streams)
			// Top-level sample for CI (names only, max 8 — no "…(+N)" token; full count in streams_count).
			sample := make([]string, 0, 8)
			for i, s := range streams {
				if i >= 8 {
					break
				}
				sample = append(sample, s.Name)
			}
			rep.StreamsNames = sample
			// Compact detail: n= + truncated names for operator logs.
			names := make([]string, 0, len(streams))
			for i, s := range streams {
				if i >= 8 {
					names = append(names, fmt.Sprintf("…(+%d)", len(streams)-8))
					break
				}
				names = append(names, truncateRunes(s.Name, 32))
			}
			detail := fmt.Sprintf("n=%d", len(streams))
			if len(names) > 0 {
				detail = fmt.Sprintf("%s names=%s", detail, strings.Join(names, ","))
			}
			return StepPass, detail
		}))
	}

	// 6c) Soft consumer create (+ optional fetch) probe — after streams; no ack.
	// Both ConsumerStream and ConsumerName required; optional filter + fetch.
	consumerStream := strings.TrimSpace(opts.ConsumerStream)
	consumerName := strings.TrimSpace(opts.ConsumerName)
	consumerFilter := strings.TrimSpace(opts.ConsumerFilter)
	if consumerStream == "" && consumerName == "" {
		rep.ConsumerProbed = false
		rep.ConsumerOK = false
		rep.ConsumerFetchOK = false
		rep.Steps = append(rep.Steps, Step{Name: "consumer", Status: StepSkip, Detail: "consumer probe unset"})
	} else if consumerStream == "" || consumerName == "" {
		rep.ConsumerProbed = false
		rep.ConsumerOK = false
		rep.ConsumerFetchOK = false
		rep.Steps = append(rep.Steps, Step{Name: "consumer", Status: StepSkip, Detail: "consumer probe needs stream and name"})
	} else {
		// Identity evidence when both stream+name provided (even if create later fails).
		rep.ConsumerStream = consumerStream
		rep.ConsumerName = consumerName
		rep.ConsumerFilter = consumerFilter
		rep.Steps = append(rep.Steps, c.stepTimed("consumer", func() (StepStatus, string) {
			rep.ConsumerProbed = true
			_, err := c.CreateConsumer(ctx, consumerStream, consumerName, consumerFilter)
			if err != nil {
				rep.ConsumerOK = false
				rep.ConsumerFetchOK = false
				if opts.Strict {
					return StepFail, err.Error()
				}
				return StepSkip, "consumer soft-fail: " + err.Error()
			}
			rep.ConsumerOK = true
			detail := fmt.Sprintf("stream=%s name=%s create=ok", consumerStream, consumerName)
			if consumerFilter != "" {
				detail = fmt.Sprintf("%s filter=%s", detail, consumerFilter)
			}
			if !opts.ConsumerFetch {
				rep.ConsumerFetchOK = false
				return StepPass, detail
			}
			msgs, ferr := c.ConsumerFetch(ctx, consumerStream, consumerName, 1, 500*time.Millisecond)
			if ferr != nil {
				rep.ConsumerFetchOK = false
				if opts.Strict {
					return StepFail, fmt.Sprintf("%s fetch=%s", detail, ferr.Error())
				}
				return StepSkip, fmt.Sprintf("%s fetch soft-fail: %s", detail, ferr.Error())
			}
			rep.ConsumerFetchOK = true
			return StepPass, fmt.Sprintf("%s fetch=n=%d", detail, len(msgs))
		}))
	}

	// 6d) Soft KV list-keys probe (non-destructive — only when KVBucket set)
	// Optional KVEnsure: best-effort KVCreateBucket before list (soft fail-open).
	kvBucket := strings.TrimSpace(opts.KVBucket)
	if kvBucket == "" {
		rep.KVKeyCount = 0
		rep.KVEnsured = false
		rep.Steps = append(rep.Steps, Step{Name: "kv", Status: StepSkip, Detail: "kv probe unset"})
	} else {
		rep.KVBucket = kvBucket
		rep.Steps = append(rep.Steps, c.stepTimed("kv", func() (StepStatus, string) {
			ensureNote := "ensure=skip"
			if opts.KVEnsure {
				if _, err := c.KVCreateBucket(ctx, kvBucket); err != nil {
					// Soft fail-open: create error never fails the step alone; list still runs.
					ensureNote = "ensure=soft-fail"
					rep.KVEnsured = false
				} else {
					ensureNote = "ensure=ok"
					rep.KVEnsured = true
				}
			}
			keys, err := c.KVListKeys(ctx, kvBucket, "")
			if err != nil {
				rep.KVKeyCount = 0
				if opts.Strict {
					return StepFail, fmt.Sprintf("%s %s", ensureNote, err.Error())
				}
				return StepSkip, fmt.Sprintf("%s kv soft-fail: %s", ensureNote, err.Error())
			}
			rep.KVKeyCount = len(keys)
			return StepPass, fmt.Sprintf("bucket=%s n=%d %s", kvBucket, len(keys), ensureNote)
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
// Wire: POST /v1/streams/dept/publish with base64 JSON DeptEvent payload (broker stream API).
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
		Endpoint            string     `json:"endpoint"`
		Tenant              string     `json:"tenant,omitempty"`
		Org                 string     `json:"org,omitempty"`
		Workspace           string     `json:"workspace,omitempty"`
		DualWrite           bool       `json:"dual_write"` // always emit (CI dual-write mode)
		CatalogSource       string     `json:"catalog_source,omitempty"`
		CatalogCount        int        `json:"catalog_count"`         // always emit (CI catalog evidence)
		ContextChars        int        `json:"context_chars"`         // always emit (CI context evidence)
		ContextLineageCount int        `json:"context_lineage_count"` // always emit
		StreamsCount        int        `json:"streams_count"`         // always emit (CI streams list evidence)
		StreamsNames        []string   `json:"streams_names"`         // always emit array (CI name sample)
		KVBucket            string     `json:"kv_bucket,omitempty"`   // set when soft kv probe configured
		KVKeyCount          int        `json:"kv_key_count"`          // always emit (CI kv list evidence)
		KVEnsured           bool       `json:"kv_ensured"`            // always emit (true only if ensure create succeeded)
		PubProbed           bool       `json:"pub_probed"`            // always emit (true if pub subject set + attempt)
		PubOK               bool       `json:"pub_ok"`                // always emit (true only if soft pub succeeded)
		ConsumerStream      string     `json:"consumer_stream,omitempty"` // set when both stream+name provided
		ConsumerName        string     `json:"consumer_name,omitempty"`
		ConsumerFilter      string     `json:"consumer_filter,omitempty"`
		ConsumerProbed      bool       `json:"consumer_probed"`   // always emit (true if stream+name set + create attempt)
		ConsumerOK          bool       `json:"consumer_ok"`       // always emit (true if create 201/409)
		ConsumerFetchOK     bool       `json:"consumer_fetch_ok"` // always emit (true if optional fetch ok)
		WaitReadyMS         int        `json:"wait_ready_ms"`         // always emit (CI wait preflight budget)
		PolicyMode          string     `json:"policy_mode"`           // always emit (off|advisory|enforce)
		PolicySource        string     `json:"policy_source,omitempty"`
		PolicyAllow         *bool      `json:"policy_allow,omitempty"` // set when policy evaluated
		MemoryEndpoint      string     `json:"memory_endpoint,omitempty"`
		UserAgent           string     `json:"user_agent,omitempty"`
		Strict              bool       `json:"strict"`
		OK                  bool       `json:"ok"`
		Summary             string     `json:"summary"`
		Started             time.Time  `json:"started"`
		Finished            time.Time  `json:"finished"`
		Steps               []stepJSON `json:"steps"`
		Result              string     `json:"result"` // PASS|FAIL|SKIP mirror of Summary prefix
	}
	names := r.StreamsNames
	if names == nil {
		names = []string{} // always emit JSON array, never null
	}
	policyMode := r.PolicyMode
	if policyMode == "" {
		policyMode = "off"
	}
	o := out{
		Endpoint:            r.Endpoint,
		Tenant:              r.Tenant,
		Org:                 r.Org,
		Workspace:           r.Workspace,
		DualWrite:           r.DualWrite,
		CatalogSource:       r.CatalogSource,
		CatalogCount:        r.CatalogCount,
		ContextChars:        r.ContextChars,
		ContextLineageCount: r.ContextLineageCount,
		StreamsCount:        r.StreamsCount,
		StreamsNames:        names,
		KVBucket:            r.KVBucket,
		KVKeyCount:          r.KVKeyCount,
		KVEnsured:           r.KVEnsured,
		PubProbed:           r.PubProbed,
		PubOK:               r.PubOK,
		ConsumerStream:      r.ConsumerStream,
		ConsumerName:        r.ConsumerName,
		ConsumerFilter:      r.ConsumerFilter,
		ConsumerProbed:      r.ConsumerProbed,
		ConsumerOK:          r.ConsumerOK,
		ConsumerFetchOK:     r.ConsumerFetchOK,
		WaitReadyMS:         r.WaitReadyMS,
		PolicyMode:          policyMode,
		PolicySource:        r.PolicySource,
		PolicyAllow:         r.PolicyAllow,
		MemoryEndpoint:      r.MemoryEndpoint,
		UserAgent:           r.UserAgent,
		Strict:              r.Strict,
		OK:                  r.OK,
		Summary:             r.Summary,
		Started:             r.Started,
		Finished:            r.Finished,
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
	if r.CatalogSource != "" {
		fmt.Fprintf(&b, "  catalog_source: %s\n", r.CatalogSource)
	}
	fmt.Fprintf(&b, "  catalog_count: %d\n", r.CatalogCount)
	fmt.Fprintf(&b, "  context_chars: %d\n", r.ContextChars)
	fmt.Fprintf(&b, "  context_lineage_count: %d\n", r.ContextLineageCount)
	fmt.Fprintf(&b, "  streams_count: %d\n", r.StreamsCount)
	if len(r.StreamsNames) > 0 {
		fmt.Fprintf(&b, "  streams_names: %s\n", strings.Join(r.StreamsNames, ","))
	} else {
		fmt.Fprintf(&b, "  streams_names: (none)\n")
	}
	if r.KVBucket != "" {
		fmt.Fprintf(&b, "  kv_bucket: %s\n", r.KVBucket)
	}
	fmt.Fprintf(&b, "  kv_key_count: %d\n", r.KVKeyCount)
	fmt.Fprintf(&b, "  kv_ensured: %v\n", r.KVEnsured)
	fmt.Fprintf(&b, "  pub_probed: %v\n", r.PubProbed)
	fmt.Fprintf(&b, "  pub_ok: %v\n", r.PubOK)
	if r.ConsumerStream != "" {
		fmt.Fprintf(&b, "  consumer_stream: %s\n", r.ConsumerStream)
	}
	if r.ConsumerName != "" {
		fmt.Fprintf(&b, "  consumer_name: %s\n", r.ConsumerName)
	}
	if r.ConsumerFilter != "" {
		fmt.Fprintf(&b, "  consumer_filter: %s\n", r.ConsumerFilter)
	}
	fmt.Fprintf(&b, "  consumer_probed: %v\n", r.ConsumerProbed)
	fmt.Fprintf(&b, "  consumer_ok: %v\n", r.ConsumerOK)
	fmt.Fprintf(&b, "  consumer_fetch_ok: %v\n", r.ConsumerFetchOK)
	fmt.Fprintf(&b, "  wait_ready_ms: %d\n", r.WaitReadyMS)
	policyMode := r.PolicyMode
	if policyMode == "" {
		policyMode = "off"
	}
	fmt.Fprintf(&b, "  policy_mode: %s\n", policyMode)
	if r.PolicySource != "" {
		fmt.Fprintf(&b, "  policy_source: %s\n", r.PolicySource)
	}
	if r.PolicyAllow != nil {
		fmt.Fprintf(&b, "  policy_allow: %v\n", *r.PolicyAllow)
	}
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
