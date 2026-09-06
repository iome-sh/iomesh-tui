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

// KVBucketInfo is broker KV bucket metadata from POST /v1/kv/{bucket} (create).
// Wire shape matches iomesh-client-sdk-go bucketResponse / CreateBucketConfig fields.
// Lean TUI surface — no SDK dependency.
type KVBucketInfo struct {
	Name       string `json:"name"`
	MaxBytes   *int64 `json:"max_bytes,omitempty"`
	History    int    `json:"history,omitempty"`
	TTLSeconds *int64 `json:"ttl_seconds,omitempty"`
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

// KVCreateBucket registers a KV bucket via POST /v1/kv/{bucket}.
// Optional empty body (no create config). 201 decodes KVBucketInfo; 409 Conflict is
// treated as success (idempotent) returning &KVBucketInfo{Name: name}.
// Empty name returns an error. Other non-2xx returns an error.
// When mesh is disabled / endpoint empty: returns (nil, error) with "mesh disabled".
// Mutating — CLI gates with --create-bucket --yes.
func (c *Client) KVCreateBucket(ctx context.Context, name string) (*KVBucketInfo, error) {
	if c == nil || !c.Enabled() {
		return nil, fmt.Errorf("mesh disabled")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("iomesh kv: bucket required")
	}
	u := strings.TrimRight(c.cfg.Endpoint, "/") + "/v1/kv/" + url.PathEscape(name)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, nil)
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
	// Idempotent: bucket already exists.
	if resp.StatusCode == http.StatusConflict {
		return &KVBucketInfo{Name: name}, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("iomesh kv: http %d", resp.StatusCode)
	}
	var info KVBucketInfo
	if len(bytes.TrimSpace(body)) > 0 {
		if err := json.Unmarshal(body, &info); err != nil {
			return nil, err
		}
	}
	if info.Name == "" {
		info.Name = name
	}
	return &info, nil
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

// KVBucketInfoPrint is a CLI-side print DTO for mesh kv create-bucket --json.
// Always emits name, history (0 when unset), max_bytes / ttl_seconds (0 when
// wire *int64 nil) without omitempty gaps for CI scrapers. Separate from wire
// KVBucketInfo so broker decode stays lean (omitempty intact on the wire type).
//
// s714 always-emit knobs. Peer FormatKVBucketInfo text (s560) + StreamInfoPrint
// s699/s702 mold. Peer mesh s713 lifecycle completeness. Beta · offline unit ≠
// live APPLY · empty/0 honest · dual_write default OFF · does not invent KV
// success from knobs alone.
type KVBucketInfoPrint struct {
	Name       string `json:"name"`
	History    int    `json:"history"`
	MaxBytes   int64  `json:"max_bytes"`
	TTLSeconds int64  `json:"ttl_seconds"`
}

// NewKVBucketInfoPrint builds a print DTO from wire KVBucketInfo. Nil *int64
// knobs become 0; history stays 0 when unset (never omitted on marshal).
func NewKVBucketInfoPrint(info KVBucketInfo) KVBucketInfoPrint {
	p := KVBucketInfoPrint{
		Name:    info.Name,
		History: info.History,
	}
	if info.MaxBytes != nil {
		p.MaxBytes = *info.MaxBytes
	}
	if info.TTLSeconds != nil {
		p.TTLSeconds = *info.TTLSeconds
	}
	return p
}

// FormatKVBucketInfoJSON returns indented JSON for stage CI / scrapers.
// Always emits all KVBucketInfoPrint fields without omitempty gaps.
// s741: Format*JSON helper completeness (DTO already always-emit s714).
// Peer mesh s740 residual. Mold FormatKVPutJSON / FormatPubJSON.
func FormatKVBucketInfoJSON(p KVBucketInfoPrint) string {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return `{"error":"kv bucket info json marshal failed"}` + "\n"
	}
	return string(b) + "\n"
}

// KVEntryPrint is a CLI-side print DTO for mesh kv get --json.
// Always emits bucket, key, value (base64; empty string when nil/empty),
// revision, created_at ("" when zero; RFC3339 UTC when set) without omitempty
// gaps. Separate from wire KVEntry so zero created_at is not omitempty-hidden.
//
// s714 always-emit. Peer FormatKVEntry text (s560) + StreamInfoPrint s699/s702
// mold. Peer mesh s713. Beta · offline unit ≠ live APPLY · empty/0 honest ·
// dual_write default OFF · does not invent KV success from knobs alone.
type KVEntryPrint struct {
	Bucket    string `json:"bucket"`
	Key       string `json:"key"`
	Value     []byte `json:"value"`
	Revision  uint64 `json:"revision"`
	CreatedAt string `json:"created_at"`
}

// NewKVEntryPrint builds a print DTO from wire KVEntry. Nil value becomes empty
// []byte (JSON ""); zero CreatedAt becomes "" (never omitted on marshal).
func NewKVEntryPrint(e KVEntry) KVEntryPrint {
	p := KVEntryPrint{
		Bucket:   e.Bucket,
		Key:      e.Key,
		Value:    e.Value,
		Revision: e.Revision,
	}
	if p.Value == nil {
		p.Value = []byte{}
	}
	if !e.CreatedAt.IsZero() {
		p.CreatedAt = e.CreatedAt.UTC().Format(time.RFC3339)
	}
	return p
}

// FormatKVEntryJSON returns indented JSON for stage CI / scrapers.
// Always emits all KVEntryPrint fields without omitempty gaps.
// s741: Format*JSON helper completeness (DTO already always-emit s714).
// Peer mesh s740 residual. Mold FormatKVPutJSON / FormatPubJSON.
func FormatKVEntryJSON(p KVEntryPrint) string {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return `{"error":"kv entry json marshal failed"}` + "\n"
	}
	return string(b) + "\n"
}

// KVKeysPrint is a CLI-side print DTO for mesh kv list --json.
// Always emits bucket, prefix (empty when unset), count, and keys ([] when
// empty) so CI scrapers get a stable envelope rather than a bare string array.
//
// s714 list envelope. Peer StreamInfoPrint list mold. Peer mesh s713. Beta ·
// offline unit ≠ live APPLY · empty/0 honest · dual_write default OFF.
type KVKeysPrint struct {
	Bucket string   `json:"bucket"`
	Prefix string   `json:"prefix"`
	Count  int      `json:"count"`
	Keys   []string `json:"keys"`
}

// NewKVKeysPrint builds a list print envelope. Nil keys become []string{}.
func NewKVKeysPrint(bucket, prefix string, keys []string) KVKeysPrint {
	if keys == nil {
		keys = []string{}
	}
	return KVKeysPrint{
		Bucket: bucket,
		Prefix: prefix,
		Count:  len(keys),
		Keys:   keys,
	}
}

// FormatKVKeysJSON returns indented JSON for stage CI / scrapers.
// Always emits all KVKeysPrint fields without omitempty gaps.
// s741: Format*JSON helper completeness (DTO already always-emit s714).
// Peer mesh s740 residual. Mold FormatKVPutJSON / FormatPubJSON.
func FormatKVKeysJSON(p KVKeysPrint) string {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return `{"error":"kv keys json marshal failed"}` + "\n"
	}
	return string(b) + "\n"
}

// KVPutPrint is a CLI-side print DTO for mesh kv --put success.
// Always emits ok / bucket / key / revision (0 honest when unset) so scrapers
// see a stable envelope without omitempty gaps. No pull_role invent and no
// value echo on put JSON (mutate success ≠ get). Wire KVPut stays lean
// (revision, error return only).
//
// s729: mold StreamDeletePrint s726 + s714 read DTOs; peer mesh s728 residual.
// s756: completeness pin — docs + unit tests lock KVPutPrint/KVDeletePrint
// (s729) with UsagePrint (s738) + PubPrint (s732) always-emit keys; does not
// invent new DTO fields or re-claim s729/s732/s738 product bodies. Peer mesh
// s755 residual. DTO ≠ invent mutate success · s714 ≠ mutate residual ·
// dual_write OFF · offline unit ≠ live APPLY · not full mesh RBAC GA.
// Closes s714 mutate half-gap. Beta · offline unit ≠ live APPLY · empty/0
// honest · dual_write OFF · not full mesh RBAC GA · does not invent put
// success when HTTP failed (call only after KVPut returns nil error).
type KVPutPrint struct {
	OK       bool   `json:"ok"`
	Bucket   string `json:"bucket"`
	Key      string `json:"key"`
	Revision uint64 `json:"revision"`
}

// NewKVPutPrint builds a put success print DTO. OK is always true (call only
// after KVPut returns a nil error). Revision 0 is honest when the broker
// returns 0.
func NewKVPutPrint(bucket, key string, revision uint64) KVPutPrint {
	return KVPutPrint{
		OK:       true,
		Bucket:   bucket,
		Key:      key,
		Revision: revision,
	}
}

// FormatKVPut is a multi-line operator view for mesh kv put success (s729).
// Always emits bucket, key, revision (0 when unset). Pure helper; no I/O.
func FormatKVPut(p KVPutPrint) string {
	var b strings.Builder
	b.WriteString("PASS mesh kv put\n")
	fmt.Fprintf(&b, "bucket:   %s\n", p.Bucket)
	fmt.Fprintf(&b, "key:      %s\n", p.Key)
	fmt.Fprintf(&b, "revision: %d\n", p.Revision)
	return b.String()
}

// FormatKVPutJSON returns indented JSON for stage CI / scrapers.
// Always emits all KVPutPrint fields without omitempty gaps.
func FormatKVPutJSON(p KVPutPrint) string {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return `{"error":"kv put json marshal failed"}` + "\n"
	}
	return string(b) + "\n"
}

// KVDeletePrint is a CLI-side print DTO for mesh kv --delete success.
// Always emits ok / bucket / key (empty string honest when unset) so scrapers
// see a stable envelope without omitempty gaps. No pull_role invent. Wire
// KVDelete stays lean (error return only).
//
// s729: mold StreamDeletePrint s726 + s714 read DTOs; peer mesh s728 residual.
// Closes s714 mutate half-gap. Beta · offline unit ≠ live APPLY · empty
// honest · dual_write OFF · not full mesh RBAC GA · does not invent delete
// success when HTTP failed (call only after KVDelete returns nil).
type KVDeletePrint struct {
	OK     bool   `json:"ok"`
	Bucket string `json:"bucket"`
	Key    string `json:"key"`
}

// NewKVDeletePrint builds a delete success print DTO. OK is always true
// (call only after KVDelete returns nil).
func NewKVDeletePrint(bucket, key string) KVDeletePrint {
	return KVDeletePrint{
		OK:     true,
		Bucket: bucket,
		Key:    key,
	}
}

// FormatKVDelete is a multi-line operator view for mesh kv delete success
// (s729). Always emits bucket and key (empty when unset). Pure helper; no I/O.
func FormatKVDelete(p KVDeletePrint) string {
	var b strings.Builder
	b.WriteString("PASS mesh kv delete\n")
	fmt.Fprintf(&b, "bucket: %s\n", p.Bucket)
	fmt.Fprintf(&b, "key:    %s\n", p.Key)
	return b.String()
}

// FormatKVDeleteJSON returns indented JSON for stage CI / scrapers.
// Always emits all KVDeletePrint fields without omitempty gaps.
func FormatKVDeleteJSON(p KVDeletePrint) string {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return `{"error":"kv delete json marshal failed"}` + "\n"
	}
	return string(b) + "\n"
}

// FormatKVBucketInfo renders bucket metadata for CLI operator display.
// Always emits history, max_bytes, and ttl_seconds (history as 0 when unset;
// *int64 nil prints blank after the colon rather than omitting the line) so
// operator/CI scrapers can key on stable fields without omitempty gaps.
func FormatKVBucketInfo(info KVBucketInfo) string {
	var b strings.Builder
	b.WriteString("iomesh kv bucket\n")
	fmt.Fprintf(&b, "name:       %s\n", info.Name)
	fmt.Fprintf(&b, "history:    %d\n", info.History)
	// Always emit max_bytes, ttl_seconds (blank when *int64 nil; do not invent 0).
	if info.MaxBytes != nil {
		fmt.Fprintf(&b, "max_bytes:  %d\n", *info.MaxBytes)
	} else {
		fmt.Fprintf(&b, "max_bytes:  \n")
	}
	if info.TTLSeconds != nil {
		fmt.Fprintf(&b, "ttl_seconds: %d\n", *info.TTLSeconds)
	} else {
		fmt.Fprintf(&b, "ttl_seconds: \n")
	}
	return b.String()
}

// FormatKVEntry renders one KV entry for CLI operator display.
// Always emits created_at (RFC3339 UTC when set; blank when zero/unset) so
// operator/CI scrapers can key a stable field without omitempty gaps.
func FormatKVEntry(e KVEntry) string {
	var b strings.Builder
	b.WriteString("iomesh kv entry\n")
	fmt.Fprintf(&b, "bucket:    %s\n", e.Bucket)
	fmt.Fprintf(&b, "key:       %s\n", e.Key)
	fmt.Fprintf(&b, "revision:  %d\n", e.Revision)
	created := ""
	if !e.CreatedAt.IsZero() {
		created = e.CreatedAt.UTC().Format(time.RFC3339)
	}
	fmt.Fprintf(&b, "created_at: %s\n", created)
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
