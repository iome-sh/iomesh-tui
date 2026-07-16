package iomesh

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// PolicyInput is sent to POST /v1/policy/evaluate.
type PolicyInput struct {
	Action     string         `json:"action"` // e.g. tool.run_shell
	Resource   string         `json:"resource,omitempty"`
	Tool       string         `json:"tool,omitempty"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

// PolicyDecision is the evaluate response (broker Rego / OPA path).
type PolicyDecision struct {
	Allow   bool     `json:"allow"`
	Reasons []string `json:"reasons,omitempty"`
	// Source describes how the decision was reached (mesh | fail-open | off | local).
	Source string `json:"source,omitempty"`
	// Mode echoes client mode used for this check.
	Mode PolicyMode `json:"mode,omitempty"`
}

// EvaluatePolicy calls the mesh policy endpoint when PolicyMode is advisory or enforce.
// Transport and non-OK responses are fail-open (Allow=true, Source=fail-open) so agent
// DX is not blocked when the broker is down. Enforce mode only denies on an explicit
// mesh response with allow=false.
func (c *Client) EvaluatePolicy(ctx context.Context, in PolicyInput) PolicyDecision {
	mode := PolicyOff
	if c != nil {
		mode = c.cfg.PolicyMode
	}
	if c == nil || !c.Enabled() || mode == PolicyOff {
		return PolicyDecision{Allow: true, Source: "off", Mode: mode}
	}
	if strings.TrimSpace(in.Action) == "" && in.Tool != "" {
		in.Action = "tool." + in.Tool
	}
	url := strings.TrimRight(c.cfg.Endpoint, "/") + "/v1/policy/evaluate"
	payload := map[string]any{
		"tenant":     c.cfg.Tenant,
		"action":     in.Action,
		"resource":   in.Resource,
		"tool":       in.Tool,
		"attributes": in.Attributes,
		"mode":       string(mode),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return PolicyDecision{Allow: true, Source: "fail-open", Mode: mode, Reasons: []string{"marshal: " + err.Error()}}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return PolicyDecision{Allow: true, Source: "fail-open", Mode: mode, Reasons: []string{err.Error()}}
	}
	c.auth(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Debug("iomesh policy: request failed (fail-open)", "err", err)
		return PolicyDecision{Allow: true, Source: "fail-open", Mode: mode, Reasons: []string{err.Error()}}
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		// Endpoint not deployed yet — treat as soft off.
		return PolicyDecision{Allow: true, Source: "unavailable", Mode: mode, Reasons: []string{"policy endpoint 404"}}
	}
	if resp.StatusCode != http.StatusOK {
		c.logger.Debug("iomesh policy: non-OK (fail-open)", "status", resp.StatusCode)
		return PolicyDecision{
			Allow: true, Source: "fail-open", Mode: mode,
			Reasons: []string{fmt.Sprintf("http %d", resp.StatusCode)},
		}
	}
	var out struct {
		Allow   *bool    `json:"allow"`
		Allowed *bool    `json:"allowed"`
		Reasons []string `json:"reasons"`
		Reason  string   `json:"reason"`
		Deny    bool     `json:"deny"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return PolicyDecision{Allow: true, Source: "fail-open", Mode: mode, Reasons: []string{"decode: " + err.Error()}}
	}
	allow := true
	if out.Allow != nil {
		allow = *out.Allow
	} else if out.Allowed != nil {
		allow = *out.Allowed
	} else if out.Deny {
		allow = false
	}
	reasons := out.Reasons
	if out.Reason != "" {
		reasons = append(reasons, out.Reason)
	}
	dec := PolicyDecision{Allow: allow, Reasons: reasons, Source: "mesh", Mode: mode}

	// Best-effort audit emit (does not affect decision).
	if c.cfg.EmitDeptStreams {
		emitCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		c.Emit(emitCtx, DeptEvent{
			Type: "dept.agent.policy_eval",
			Payload: map[string]any{
				"action":  in.Action,
				"tool":    in.Tool,
				"allow":   dec.Allow,
				"mode":    string(mode),
				"source":  dec.Source,
				"reasons": dec.Reasons,
			},
		})
	}
	return dec
}

// ShouldBlockTool returns true only when mode is enforce and mesh explicitly denies.
func (d PolicyDecision) ShouldBlockTool() bool {
	return d.Mode == PolicyEnforce && !d.Allow && d.Source == "mesh"
}

// Summary is a short operator-facing string.
func (d PolicyDecision) Summary() string {
	if d.Allow {
		return fmt.Sprintf("allow source=%s mode=%s", d.Source, d.Mode)
	}
	r := strings.Join(d.Reasons, "; ")
	if r == "" {
		r = "denied"
	}
	return fmt.Sprintf("deny source=%s mode=%s reasons=%s", d.Source, d.Mode, r)
}
