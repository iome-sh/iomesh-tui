package iomesh

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// KVEntry is a versioned key-value record from the broker (GET /v1/kv/{bucket}/{key}).
// Wire shape matches iomesh-client-sdk-go KVEntry. Value is JSON base64-decoded into []byte.
// Lean TUI surface — no SDK dependency.
type KVEntry struct {
	Bucket    string    `json:"bucket"`
	Key       string    `json:"key"`
	Value     []byte    `json:"value"`
	Revision  uint64    `json:"revision"`
	CreatedAt time.Time `json:"created_at"`
}

// KVGet returns the current value for key in bucket via GET /v1/kv/{bucket}/{key}.
// Empty bucket/key returns an error. Non-2xx (including 404) returns an error.
// When mesh is disabled / endpoint empty: returns (nil, error) with "mesh disabled".
func (c *Client) KVGet(ctx context.Context, bucket, key string) (*KVEntry, error) {
	if c == nil || !c.Enabled() {
		return nil, fmt.Errorf("mesh disabled")
	}
	bucket = strings.TrimSpace(bucket)
	key = strings.TrimSpace(key)
	if bucket == "" {
		return nil, fmt.Errorf("iomesh kv: bucket required")
	}
	if key == "" {
		return nil, fmt.Errorf("iomesh kv: key required")
	}
	u := strings.TrimRight(c.cfg.Endpoint, "/") + "/v1/kv/" + url.PathEscape(bucket) + "/" + url.PathEscape(key)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	c.auth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("iomesh kv: http %d", resp.StatusCode)
	}
	var entry KVEntry
	if err := json.Unmarshal(body, &entry); err != nil {
		return nil, err
	}
	if entry.Bucket == "" {
		entry.Bucket = bucket
	}
	if entry.Key == "" {
		entry.Key = key
	}
	return &entry, nil
}

// KVListKeys returns keys in bucket via GET /v1/kv/{bucket}?prefix=.
// Empty bucket returns an error. Non-2xx returns an error.
// Accepts {"keys":[...]} envelope or a bare JSON string array.
// When mesh is disabled / endpoint empty: returns (nil, error) with "mesh disabled".
func (c *Client) KVListKeys(ctx context.Context, bucket, prefix string) ([]string, error) {
	if c == nil || !c.Enabled() {
		return nil, fmt.Errorf("mesh disabled")
	}
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return nil, fmt.Errorf("iomesh kv: bucket required")
	}
	u := strings.TrimRight(c.cfg.Endpoint, "/") + "/v1/kv/" + url.PathEscape(bucket)
	if p := strings.TrimSpace(prefix); p != "" {
		u += "?prefix=" + url.QueryEscape(p)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	c.auth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("iomesh kv: http %d", resp.StatusCode)
	}
	return decodeKVKeys(body)
}

// KVPut writes value for key in bucket via PUT /v1/kv/{bucket}/{key}.
// Body is {"value": base64} (SDK wire parity). Returns the broker revision on success.
// Empty bucket/key returns an error. Non-2xx returns an error.
// When mesh is disabled / endpoint empty: returns (0, error) with "mesh disabled".
func (c *Client) KVPut(ctx context.Context, bucket, key string, value []byte) (uint64, error) {
	if c == nil || !c.Enabled() {
		return 0, fmt.Errorf("mesh disabled")
	}
	bucket = strings.TrimSpace(bucket)
	key = strings.TrimSpace(key)
	if bucket == "" {
		return 0, fmt.Errorf("iomesh kv: bucket required")
	}
	if key == "" {
		return 0, fmt.Errorf("iomesh kv: key required")
	}
	if value == nil {
		value = []byte{}
	}
	body, err := json.Marshal(map[string]string{
		"value": base64.StdEncoding.EncodeToString(value),
	})
	if err != nil {
		return 0, err
	}
	u := strings.TrimRight(c.cfg.Endpoint, "/") + "/v1/kv/" + url.PathEscape(bucket) + "/" + url.PathEscape(key)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.auth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return 0, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("iomesh kv: http %d", resp.StatusCode)
	}
	var out struct {
		Bucket   string `json:"bucket"`
		Key      string `json:"key"`
		Revision uint64 `json:"revision"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return 0, err
	}
	return out.Revision, nil
}

// KVDelete removes key from bucket via DELETE /v1/kv/{bucket}/{key}.
// Empty bucket/key returns an error. 2xx (including 204 No Content) is success; non-2xx returns error.
// When mesh is disabled / endpoint empty: returns error with "mesh disabled".
// Destructive — CLI gates with --delete KEY --yes.
func (c *Client) KVDelete(ctx context.Context, bucket, key string) error {
	if c == nil || !c.Enabled() {
		return fmt.Errorf("mesh disabled")
	}
	bucket = strings.TrimSpace(bucket)
	key = strings.TrimSpace(key)
	if bucket == "" {
		return fmt.Errorf("iomesh kv: bucket required")
	}
	if key == "" {
		return fmt.Errorf("iomesh kv: key required")
	}
	u := strings.TrimRight(c.cfg.Endpoint, "/") + "/v1/kv/" + url.PathEscape(bucket) + "/" + url.PathEscape(key)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	c.auth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("iomesh kv: http %d", resp.StatusCode)
	}
	return nil
}

func decodeKVKeys(raw []byte) ([]string, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return []string{}, nil
	}
	// Bare JSON array of keys.
	var bare []string
	if err := json.Unmarshal(raw, &bare); err == nil {
		if bare == nil {
			bare = []string{}
		}
		return bare, nil
	}
	// Envelope {"keys":[...]} (SDK / broker default).
	var env struct {
		Keys []string `json:"keys"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Keys == nil {
		env.Keys = []string{}
	}
	return env.Keys, nil
}

// FormatKVEntry renders one KV entry for CLI operator display.
func FormatKVEntry(e KVEntry) string {
	var b strings.Builder
	b.WriteString("iomesh kv entry\n")
	fmt.Fprintf(&b, "bucket:    %s\n", e.Bucket)
	fmt.Fprintf(&b, "key:       %s\n", e.Key)
	fmt.Fprintf(&b, "revision:  %d\n", e.Revision)
	if !e.CreatedAt.IsZero() {
		fmt.Fprintf(&b, "created_at: %s\n", e.CreatedAt.UTC().Format(time.RFC3339))
	}
	val := string(e.Value)
	// Prefer printable preview; fall back to base64-looking hex length note for binary.
	if isMostlyPrintable(e.Value) {
		fmt.Fprintf(&b, "value:     %s\n", truncateRunes(val, 256))
	} else {
		fmt.Fprintf(&b, "value:     <%d bytes binary>\n", len(e.Value))
	}
	return b.String()
}

// FormatKVKeys renders a key list for CLI operator display.
func FormatKVKeys(bucket string, keys []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "iomesh kv keys bucket=%s count=%d\n", bucket, len(keys))
	if len(keys) == 0 {
		b.WriteString("(no keys)\n")
		return b.String()
	}
	for i, k := range keys {
		if i >= 200 {
			fmt.Fprintf(&b, "… (%d more)\n", len(keys)-200)
			break
		}
		fmt.Fprintf(&b, "  %s\n", k)
	}
	return b.String()
}

func isMostlyPrintable(v []byte) bool {
	if len(v) == 0 {
		return true
	}
	nonPrint := 0
	for _, c := range v {
		if c == '\n' || c == '\r' || c == '\t' {
			continue
		}
		if c < 32 || c > 126 {
			nonPrint++
		}
	}
	return nonPrint*4 <= len(v) // allow up to ~25% non-printable
}
