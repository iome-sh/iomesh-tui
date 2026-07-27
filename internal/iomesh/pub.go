package iomesh

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Pub publishes an ephemeral fire-and-forget message via POST /v1/pub.
// Wire matches iomesh-client-sdk-go Pub: body {"subject","payload" as raw string,"headers"?}
// (payload is NOT base64 — unlike stream Publish). Empty subject returns an error.
// Non-2xx returns an error. When mesh is disabled / endpoint empty: "mesh disabled".
// Mutating — CLI gates with --yes.
func (c *Client) Pub(ctx context.Context, subject string, payload []byte, headers map[string]string) error {
	if c == nil || !c.Enabled() {
		return fmt.Errorf("mesh disabled")
	}
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return fmt.Errorf("iomesh pub: subject required")
	}
	if payload == nil {
		payload = []byte{}
	}
	reqBody := struct {
		Subject string            `json:"subject"`
		Payload string            `json:"payload"`
		Headers map[string]string `json:"headers,omitempty"`
	}{
		Subject: subject,
		Payload: string(payload),
		Headers: headers,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	u := strings.TrimRight(c.cfg.Endpoint, "/") + "/v1/pub"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.auth(req)
	c.applyEntitlementHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("iomesh pub: http %d", resp.StatusCode)
	}
	return nil
}

// PubPrint is a CLI-side print DTO for mesh pub success.
// Always emits ok / subject / bytes (0 honest when unset; empty subject honest)
// so scrapers see a stable envelope without omitempty gaps. No pull_role invent
// and no payload echo (pub success ≠ get/fetch). Wire Pub stays lean (error
// return only).
//
// s732: mold StreamDeletePrint s726 + KVPutPrint s729; peer aion s731 residual.
// Ephemeral POST /v1/pub ≠ durable stream publish. Beta · offline unit ≠ live
// APPLY · empty/0 honest · dual_write OFF · not full mesh RBAC GA · does not
// invent pub success when HTTP failed (call only after Pub returns nil).
type PubPrint struct {
	OK      bool   `json:"ok"`
	Subject string `json:"subject"`
	Bytes   int    `json:"bytes"`
}

// NewPubPrint builds a pub success print DTO. OK is always true (call only
// after mesh.Pub returns a nil error). Bytes 0 is honest for empty payload;
// empty subject is honest when unset (CLI still requires --subject before call).
func NewPubPrint(subject string, bytes int) PubPrint {
	return PubPrint{
		OK:      true,
		Subject: subject,
		Bytes:   bytes,
	}
}

// FormatPub is a multi-line operator view for mesh pub success (s732).
// Always emits subject and bytes (0 when unset). Pure helper; no I/O.
func FormatPub(p PubPrint) string {
	var b strings.Builder
	b.WriteString("PASS mesh pub\n")
	fmt.Fprintf(&b, "subject: %s\n", p.Subject)
	fmt.Fprintf(&b, "bytes:   %d\n", p.Bytes)
	return b.String()
}

// FormatPubJSON returns indented JSON for stage CI / scrapers.
// Always emits all PubPrint fields without omitempty gaps.
func FormatPubJSON(p PubPrint) string {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return `{"error":"pub json marshal failed"}` + "\n"
	}
	return string(b) + "\n"
}
