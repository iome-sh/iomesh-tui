package iomesh

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// MemoryPullSourceHint is the classified source stamped on durable mesh pull
// ingest so cite-both digests can see mesh. MCP defaults an omitted hint to private.
// Companion: iomesh-memory-mcp#64 accepts optional source_hint on memory_ingest_turn;
// until that lands, MCP should ignore the unknown field.
const MemoryPullSourceHint = "mesh"

// MemoryPullOptions configures RunMemoryPull (mesh durable consumer → local ingest).
// Cost-max M1 (s652): pull egress fills customer-local Palace; dual_write remains optional audit.
type MemoryPullOptions struct {
	Stream   string
	Name     string // durable consumer name
	Filter   string // optional filter_subject
	Batch    int
	MaxWait  time.Duration
	MaxLoops int  // 0 = forever; 1 = single fetch cycle (dogfood / --once)
	Ack      bool // ack after successful ingest (or dry-run)
	DryRun   bool // map + log only; no LocalIngest
	// LocalIngest writes one mapped envelope to local palace (MCP memory_ingest_turn).
	// Required when DryRun is false.
	LocalIngest func(ctx context.Context, env MemoryEnvelope) error
	// OnMessage is optional observability after each message (dry-run or ingest).
	OnMessage func(msg StreamMessage, env MemoryEnvelope, skipped bool, err error)
}

// MemoryPullStats summarizes one RunMemoryPull invocation.
type MemoryPullStats struct {
	Loops     int
	Fetched   int
	Ingested  int
	Skipped   int
	Acked     int
	Errors    int
	LastError string
	CreateOK  bool
	Consumer  string
	Stream    string
	Filter    string // effective filter_subject passed to CreateConsumer
}

// MemoryPullStatsPrint is a CLI-side print DTO for `iomesh memory pull` text/JSON.
// Always emits identity + knobs + counters + process evidence without omitempty so
// CI scrapers can key stable fields (empty string / false / 0 honest when unset).
// Separate from MemoryPullStats so the runtime stats shape stays lean.
//
// s705: peer create FormatConsumerInfo s696 + status/wait pull identity continuum;
// peer mesh s704 sales claim suite/retention honesty.
// s717: process evidence always-emit (endpoint/org/workspace + result/exit_code +
// duration_ms/ack); peer mesh s716 residual.
// s747: process-evidence completeness pin — docs + unit tests lock the complete
// s705 identity/knobs/counters + s717 process evidence surface; does not invent
// new always-emit fields or re-claim s717 product body. Peer mesh s746 residual.
// Beta · offline unit ≠ live APPLY · dual_write default OFF (report-only) ·
// fail-open empty role/tenant · not full mesh RBAC GA · process evidence ≠ invent
// pull success from identity fields alone.
type MemoryPullStatsPrint struct {
	// Identity (always emit; empty string when unset).
	Stream          string `json:"stream"`
	Consumer        string `json:"consumer"`
	FilterSubject   string `json:"filter_subject"`
	PullRole        string `json:"pull_role"`
	PullAllowSuffix string `json:"pull_allow_suffix"`
	Tenant          string `json:"tenant"`
	// Process / mesh identity (s717 always emit; empty string when unset).
	Endpoint  string `json:"endpoint"`
	Org       string `json:"org"`
	Workspace string `json:"workspace"`
	// Knobs (always emit).
	DryRun    bool `json:"dry_run"`
	DualWrite bool `json:"dual_write"` // report-only from [memory].dual_write; default false
	Batch     int  `json:"batch"`
	MaxWaitMS int  `json:"max_wait_ms"`
	Once      bool `json:"once"`
	Ack       bool `json:"ack"` // effective ack knob (true when not --no-ack)
	// Counters (always emit).
	CreateOK  bool   `json:"create_ok"`
	Loops     int    `json:"loops"`
	Fetched   int    `json:"fetched"`
	Ingested  int    `json:"ingested"`
	Skipped   int    `json:"skipped"`
	Acked     int    `json:"acked"`
	Errors    int    `json:"errors"`
	LastError string `json:"last_error"` // empty when none
	// Process evidence (s717 always emit).
	// Result is "ok" | "err" matching process intent (not invent from identity alone).
	Result string `json:"result"`
	// ExitCode is intended process exit (0 success; 1 hard/soft fail). Always emitted.
	ExitCode int `json:"exit_code"`
	// DurationMS is wall-clock pull duration in ms (0 when not timed). Always emitted.
	DurationMS int `json:"duration_ms"`
}

// MemoryPullPrintMeta holds pull identity + knobs that are not on MemoryPullStats
// (resolved CLI/config auth, dual_write mode, batch/wait/once, process evidence).
// Used by NewMemoryPullStatsPrint so scrapers always see the same surface as
// stderr start log + process result/exit.
type MemoryPullPrintMeta struct {
	Tenant          string
	PullRole        string
	PullAllowSuffix string
	// Endpoint / Org / Workspace from [iomesh] (empty string honest when unset).
	Endpoint  string
	Org       string
	Workspace string
	DryRun    bool
	DualWrite bool // report-only; does not gate pull
	Batch     int
	MaxWaitMS int
	Once      bool
	Ack       bool // effective ack after --no-ack resolution
	// Process evidence (caller sets after path resolution; empty/0 honest).
	Result     string // "ok" | "err"
	ExitCode   int    // 0 | 1 matching process intent
	DurationMS int    // wall-clock ms; 0 if not timed
}

// NewMemoryPullStatsPrint builds a print DTO from runtime stats + resolved meta.
// Empty identity strings and zero knobs are always present (not omitted on marshal).
// Does not invent success from identity: counters come from st only; result/exit_code
// come from meta (caller sets process intent). FilterSubject is st.Filter (effective
// filter_subject passed to CreateConsumer).
func NewMemoryPullStatsPrint(st MemoryPullStats, meta MemoryPullPrintMeta) MemoryPullStatsPrint {
	return MemoryPullStatsPrint{
		Stream:          st.Stream,
		Consumer:        st.Consumer,
		FilterSubject:   st.Filter,
		PullRole:        meta.PullRole,
		PullAllowSuffix: meta.PullAllowSuffix,
		Tenant:          meta.Tenant,
		Endpoint:        meta.Endpoint,
		Org:             meta.Org,
		Workspace:       meta.Workspace,
		DryRun:          meta.DryRun,
		DualWrite:       meta.DualWrite,
		Batch:           meta.Batch,
		MaxWaitMS:       meta.MaxWaitMS,
		Once:            meta.Once,
		Ack:             meta.Ack,
		CreateOK:        st.CreateOK,
		Loops:           st.Loops,
		Fetched:         st.Fetched,
		Ingested:        st.Ingested,
		Skipped:         st.Skipped,
		Acked:           st.Acked,
		Errors:          st.Errors,
		LastError:       st.LastError,
		Result:          meta.Result,
		ExitCode:        meta.ExitCode,
		DurationMS:      meta.DurationMS,
	}
}

// FormatMemoryPullStats renders memory pull outcome as multi-line operator text.
// Always emits identity (stream/consumer/filter_subject/pull_role/pull_allow_suffix/tenant
// + endpoint/org/workspace), knobs (dry_run/dual_write/batch/max_wait_ms/once/ack),
// counters including last_error, and process evidence (result/exit_code/duration_ms)
// so scrapers do not rely only on stderr start log. Pure helper; no I/O.
//
// ok=true → PASS header; ok=false → FAIL header (errMsg optional detail; empty uses last_error).
func FormatMemoryPullStats(p MemoryPullStatsPrint, ok bool, errMsg string) string {
	var b strings.Builder
	if ok {
		b.WriteString("PASS memory pull\n")
	} else {
		if strings.TrimSpace(errMsg) == "" {
			errMsg = p.LastError
		}
		if strings.TrimSpace(errMsg) == "" {
			errMsg = "unknown error"
		}
		fmt.Fprintf(&b, "FAIL memory pull: %s\n", errMsg)
	}
	fmt.Fprintf(&b, "stream:            %s\n", p.Stream)
	fmt.Fprintf(&b, "consumer:          %s\n", p.Consumer)
	fmt.Fprintf(&b, "filter_subject:    %s\n", p.FilterSubject)
	fmt.Fprintf(&b, "pull_role:         %s\n", p.PullRole)
	fmt.Fprintf(&b, "pull_allow_suffix: %s\n", p.PullAllowSuffix)
	fmt.Fprintf(&b, "tenant:            %s\n", p.Tenant)
	fmt.Fprintf(&b, "endpoint:          %s\n", p.Endpoint)
	fmt.Fprintf(&b, "org:               %s\n", p.Org)
	fmt.Fprintf(&b, "workspace:         %s\n", p.Workspace)
	fmt.Fprintf(&b, "dry_run:           %t\n", p.DryRun)
	fmt.Fprintf(&b, "dual_write:        %t\n", p.DualWrite)
	fmt.Fprintf(&b, "batch:             %d\n", p.Batch)
	fmt.Fprintf(&b, "max_wait_ms:       %d\n", p.MaxWaitMS)
	fmt.Fprintf(&b, "once:              %t\n", p.Once)
	fmt.Fprintf(&b, "ack:               %t\n", p.Ack)
	fmt.Fprintf(&b, "create_ok:         %t\n", p.CreateOK)
	fmt.Fprintf(&b, "loops:             %d\n", p.Loops)
	fmt.Fprintf(&b, "fetched:           %d\n", p.Fetched)
	fmt.Fprintf(&b, "ingested:          %d\n", p.Ingested)
	fmt.Fprintf(&b, "skipped:           %d\n", p.Skipped)
	fmt.Fprintf(&b, "acked:             %d\n", p.Acked)
	fmt.Fprintf(&b, "errors:            %d\n", p.Errors)
	fmt.Fprintf(&b, "last_error:        %s\n", p.LastError)
	fmt.Fprintf(&b, "result:            %s\n", p.Result)
	fmt.Fprintf(&b, "exit_code:         %d\n", p.ExitCode)
	fmt.Fprintf(&b, "duration_ms:       %d\n", p.DurationMS)
	return b.String()
}

// FormatMemoryPullStatsJSON returns indented JSON for stage CI / scrapers.
// Always emits all MemoryPullStatsPrint fields without omitempty gaps.
func FormatMemoryPullStatsJSON(p MemoryPullStatsPrint) string {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return `{"error":"memory pull stats json marshal failed"}` + "\n"
	}
	return string(b) + "\n"
}

// DefaultDeptEventsPullFilter is the wire-aligned default for org-shaped tenants.
// Durable connector / org-pulse subjects are dept.<dept>.events.* — not org_<id>.events.>.
// NATS `>` matches remaining tokens (`dept.*.events.>` entitles `dept.*.events.*`).
const DefaultDeptEventsPullFilter = "dept.*.events.>"

// DefaultMemoryPullFilter returns an effective consumer filter_subject with empty
// role (s660). Prefer DefaultMemoryPullFilterForRole when pull role is known.
// Pure: no I/O.
func DefaultMemoryPullFilter(explicit, tenant string) string {
	return DefaultMemoryPullFilterForRole(explicit, tenant, "", "")
}

// DefaultMemoryPullFilterForRole returns an effective consumer filter_subject (s678/s687).
// When explicit is non-empty after trim, it always wins.
// When filter is empty:
//   - role empty: s660 — tenant.> only for hierarchical / dept.* tenants
//   - agent|viewer: tenant.events.> when tenant set
//   - memory (s687 / peer mesh s686): tenant.memory.> when tenant set
//   - auditor: tenant.audit.> when tenant set
//   - operator|admin: tenant.> when tenant set
//   - custom + exactly one allow-suffix token: tenant.<suffix>.>
//   - custom + multiple/no suffixes: empty (fail closed; operator must pass --filter)
//   - unknown role: empty (no invent)
//
// Org-shaped tenants (`org_*`) are identity ids, not subject tokens. They are
// remapped to dept.* (or dept.<department> when a safe department token is set)
// so the default entitles durable dept.*.events.* wire subjects. Explicit
// --filter always wins (including a mismatched org_*.events.> — see
// MemoryPullFilterMismatchHint).
//
// Beta federated ACL defaults — not full mesh RBAC GA. Fail-open headers remain
// separate (empty role/suffix still omit X-IOMesh-* auth headers).
// Pure: no I/O.
func DefaultMemoryPullFilterForRole(explicit, tenant, role, allowSuffix string) string {
	return DefaultMemoryPullFilterForRoleWithDept(explicit, tenant, role, allowSuffix, "")
}

// DefaultMemoryPullFilterForRoleWithDept is DefaultMemoryPullFilterForRole with
// an optional department token. When tenant is org-shaped and department is a
// single safe token (no dots/wildcards), the wire prefix is dept.<department>
// instead of dept.*. Explicit filter still wins. Pure: no I/O.
func DefaultMemoryPullFilterForRoleWithDept(explicit, tenant, role, allowSuffix, department string) string {
	explicit = strings.TrimSpace(explicit)
	if explicit != "" {
		return explicit
	}
	tenant = memoryPullWireSubjectPrefix(strings.TrimSpace(tenant), department)
	if tenant == "" {
		return ""
	}
	role = strings.ToLower(strings.TrimSpace(role))
	switch role {
	case "":
		// s660: hierarchical / dept.* only.
		if strings.Contains(tenant, ".") || strings.HasPrefix(tenant, "dept") {
			return tenant + ".>"
		}
		return ""
	case "agent", "viewer":
		return tenant + ".events.>"
	case "memory":
		// s687: local-palace memory subjects under tenant (peer mesh s686).
		return tenant + ".memory.>"
	case "auditor":
		return tenant + ".audit.>"
	case "operator", "admin":
		return tenant + ".>"
	case "custom":
		tokens := splitPullAllowSuffixTokens(allowSuffix)
		if len(tokens) == 1 {
			return tenant + "." + tokens[0] + ".>"
		}
		// Multiple or zero suffixes: leave empty so mesh fails closed without --filter.
		return ""
	default:
		// Unknown role: do not invent a default filter.
		return ""
	}
}

// memoryPullWireSubjectPrefix maps a configured tenant onto a broker subject prefix.
// Org ids (org_*) are not subject tokens — durable events live on dept.<dept>.events.*.
// Hierarchical dept.* / dotted tenants stay as-is. Plain tenants stay as-is.
// Pure: no I/O.
func memoryPullWireSubjectPrefix(tenant, department string) string {
	tenant = strings.TrimSpace(tenant)
	if tenant == "" {
		return ""
	}
	if !isOrgShapedTenant(tenant) {
		return tenant
	}
	if prefix := deptSubjectPrefixFromDepartment(department); prefix != "" {
		return prefix
	}
	return "dept.*"
}

func isOrgShapedTenant(tenant string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(tenant)), "org_")
}

// isDeptFamilyFilter reports whether filter_subject is in the durable dept.* family.
func isDeptFamilyFilter(filter string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(filter)), "dept.")
}

// filterSubjectUnderTenant reports whether filter_subject is namespaced under tenant
// (exact match or tenant. prefix). Broker Role+Tenant ACL requires this. Pure: no I/O.
func filterSubjectUnderTenant(filter, tenant string) bool {
	filter = strings.ToLower(strings.TrimSpace(filter))
	tenant = strings.ToLower(strings.TrimSpace(tenant))
	if filter == "" || tenant == "" {
		return false
	}
	return filter == tenant || strings.HasPrefix(filter, tenant+".")
}

// OmitPullRoleForDeptFilter reports whether X-IOMesh-Role and the Role-gated
// X-IOMesh-Tenant bind must be omitted so a dept.* consumer can be created.
// Broker: Role without Tenant → 400; Role + Tenant requires filter_subject under
// that tenant. Org-shaped tenants remapped to dept.* fail that check. Proven
// create is no-Role 201. Does not change broker ACL. Pure: no I/O.
func OmitPullRoleForDeptFilter(filter, tenant, role string) bool {
	if strings.TrimSpace(role) == "" {
		return false
	}
	if !isDeptFamilyFilter(filter) {
		return false
	}
	return !filterSubjectUnderTenant(filter, tenant)
}

// ApplyDeptPullWireAuth returns the Role and Tenant that should be sent on the
// wire for a pull/create. When OmitPullRoleForDeptFilter is true, both are
// empty (headers omitted). Configured identity for print/scrapers is unchanged.
// Pure: no I/O.
func ApplyDeptPullWireAuth(filter, tenant, role string) (wireRole, wireTenant string) {
	role = strings.TrimSpace(role)
	tenant = strings.TrimSpace(tenant)
	if OmitPullRoleForDeptFilter(filter, tenant, role) {
		return "", ""
	}
	return role, tenant
}

// MemoryPullRoleTenantACLHint names the Role/Tenant/filter ACL when a dept.*
// filter cannot be created under a Role+Tenant bind. Empty when the combo is
// compatible. Does not invent pull success. Pure: no I/O.
func MemoryPullRoleTenantACLHint(filter, tenant, role string) string {
	role = strings.TrimSpace(role)
	tenant = strings.TrimSpace(tenant)
	filter = strings.TrimSpace(filter)
	if !OmitPullRoleForDeptFilter(filter, tenant, role) {
		return ""
	}
	if tenant == "" {
		return "Role requires Tenant; dept.* consumer create omits X-IOMesh-Role (broker no-Role create)"
	}
	return fmt.Sprintf("Role + Tenant %q requires filter_subject under that tenant; dept.* filter %q omits Role and Role-gated Tenant (broker no-Role create)", tenant, filter)
}

// withoutRoleGatedTenantBind returns a shallow client copy with Role and Tenant
// cleared so create/fetch/ack omit those headers. Org/Department stay.
func (c *Client) withoutRoleGatedTenantBind() *Client {
	if c == nil {
		return nil
	}
	clone := *c
	clone.cfg.Role = ""
	clone.cfg.Tenant = ""
	return &clone
}

// deptSubjectPrefixFromDepartment returns dept.<token> when department is a single
// safe subject token. Empty / dotted / wildcard department → empty (caller uses dept.*).
func deptSubjectPrefixFromDepartment(department string) string {
	d := strings.TrimSpace(department)
	if d == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(d), "dept.") {
		d = strings.TrimSpace(d[len("dept."):])
	}
	if d == "" || strings.ContainsAny(d, ".*> \t") {
		return ""
	}
	return "dept." + d
}

// MemoryPullFilterMismatchHint names an org-id filter that cannot match durable
// dept.*.events.* subjects. Empty when the filter is not that mismatch.
// Does not invent pull success. Pure: no I/O.
func MemoryPullFilterMismatchHint(filter string) string {
	filter = strings.TrimSpace(filter)
	if filter == "" || !isOrgShapedTenant(filter) {
		return ""
	}
	return fmt.Sprintf("filter_subject %q does not match durable dept.*.events.* subjects; pass --filter %s (or dept.<dept>.events.>)",
		filter, DefaultDeptEventsPullFilter)
}

// natsSubjectMatches reports whether subject matches a NATS-style filter using
// * (exactly one token) and > (one or more remaining tokens). Empty filter
// matches any subject (broker "no filter" meaning). Pure: no I/O.
func natsSubjectMatches(subject, filter string) bool {
	subject = strings.TrimSpace(subject)
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return subject != ""
	}
	if subject == "" {
		return false
	}
	sParts := strings.Split(subject, ".")
	fParts := strings.Split(filter, ".")
	si := 0
	for fi := 0; fi < len(fParts); fi++ {
		tok := fParts[fi]
		if tok == ">" {
			return si < len(sParts)
		}
		if si >= len(sParts) {
			return false
		}
		if tok != "*" && tok != sParts[si] {
			return false
		}
		si++
	}
	return si == len(sParts)
}

// splitPullAllowSuffixTokens splits comma-separated X-IOMesh-Pull-Allow-Suffix
// tokens, trimming whitespace and dropping empty entries.
func splitPullAllowSuffixTokens(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// MemoryPullIngestArgs builds MCP memory_ingest_turn arguments for one pulled
// envelope. Always includes source_hint (mesh, or env.SourceHint when already set)
// so palace provenance is classified mesh. Local /memory ingest must not use this
// helper — those paths stay private. dual_write stays off.
func MemoryPullIngestArgs(env MemoryEnvelope, tenant string) map[string]any {
	hint := strings.TrimSpace(env.SourceHint)
	if hint == "" {
		hint = MemoryPullSourceHint
	}
	args := map[string]any{
		"role":        env.Role,
		"content":     env.Content,
		"source_hint": hint,
	}
	if env.EventTime != "" {
		args["event_time"] = env.EventTime
	}
	if env.SessionID != "" {
		args["session_id"] = env.SessionID
	}
	if strings.TrimSpace(tenant) != "" {
		args["tenant"] = strings.TrimSpace(tenant)
	}
	return args
}

func stampMemoryPullSourceHint(env MemoryEnvelope) MemoryEnvelope {
	env.SourceHint = MemoryPullSourceHint
	return env
}

// MapStreamMessageToEnvelope converts a durable-fetch message into a MemoryEnvelope for local ingest.
// Supports:
//   - MEMORY_INGEST style JSON (type memory_ingest / fields content, role, session_id, session_seq, event_time)
//   - generic JSON with content/text/body/message
//   - raw text payload (connector events)
//
// Returns ok=false when there is no ingestible content.
// Successful maps stamp SourceHint=mesh (pull path only; local ingest does not use this mapper).
func MapStreamMessageToEnvelope(msg StreamMessage) (MemoryEnvelope, string /*dedupeKey*/, bool) {
	payload := bytesTrimSpace(msg.Payload)
	if len(payload) == 0 {
		return MemoryEnvelope{}, "", false
	}
	dedupe := fmt.Sprintf("%s:%d", msg.Stream, msg.Seq)

	var env MemoryEnvelope
	if json.Unmarshal(payload, &env) == nil && strings.TrimSpace(env.Content) != "" {
		if strings.TrimSpace(env.Type) == "" {
			env.Type = memoryEnvelopeIngest
		}
		if env.SessionSeq > 0 && strings.TrimSpace(env.SessionID) != "" {
			dedupe = env.SessionID + ":" + fmt.Sprintf("%d", env.SessionSeq)
		}
		return stampMemoryPullSourceHint(env), dedupe, true
	}

	// Generic event JSON — prefer common text fields.
	var generic map[string]any
	if json.Unmarshal(payload, &generic) == nil {
		content := firstString(generic, "content", "text", "body", "message", "summary")
		if content == "" {
			// Compact JSON as content when structured but no text field.
			if b, err := json.Marshal(generic); err == nil && len(b) > 0 && len(b) < 32<<10 {
				content = string(b)
			}
		}
		if content != "" {
			role := firstString(generic, "role")
			if role == "" {
				role = "system"
			}
			sid := firstString(generic, "session_id", "session")
			if sid == "" {
				sid = msg.Subject
			}
			seq := intFromAny(generic["session_seq"])
			if seq == 0 {
				seq = int(msg.Seq)
			}
			if sid != "" && seq > 0 {
				dedupe = sid + ":" + fmt.Sprintf("%d", seq)
			}
			et := firstString(generic, "event_time", "timestamp", "time")
			if et == "" && !msg.Timestamp.IsZero() {
				et = msg.Timestamp.UTC().Format(time.RFC3339)
			}
			return stampMemoryPullSourceHint(MemoryEnvelope{
				Type:       memoryEnvelopeIngest,
				SessionID:  sid,
				Role:       role,
				Content:    content,
				EventTime:  et,
				SessionSeq: seq,
			}), dedupe, true
		}
	}

	// Raw text / non-JSON
	text := string(payload)
	if strings.TrimSpace(text) == "" {
		return MemoryEnvelope{}, "", false
	}
	et := ""
	if !msg.Timestamp.IsZero() {
		et = msg.Timestamp.UTC().Format(time.RFC3339)
	}
	return stampMemoryPullSourceHint(MemoryEnvelope{
		Type:       memoryEnvelopeIngest,
		SessionID:  msg.Subject,
		Role:       "system",
		Content:    text,
		EventTime:  et,
		SessionSeq: int(msg.Seq),
	}), dedupe, true
}

// RunMemoryPull creates (idempotent) a durable consumer and loops fetch → map → local ingest → ack.
// Mesh client must be enabled. Fail-open per message: ingest errors increment Errors and may skip ack.
func (c *Client) RunMemoryPull(ctx context.Context, opt MemoryPullOptions) (MemoryPullStats, error) {
	var st MemoryPullStats
	if c == nil || !c.Enabled() {
		return st, fmt.Errorf("mesh disabled")
	}
	opt.Stream = strings.TrimSpace(opt.Stream)
	opt.Name = strings.TrimSpace(opt.Name)
	if opt.Stream == "" || opt.Name == "" {
		return st, fmt.Errorf("memory pull: stream and consumer name required")
	}
	if opt.Batch <= 0 {
		opt.Batch = 8
	}
	if opt.MaxWait <= 0 {
		opt.MaxWait = 2 * time.Second
	}
	if !opt.DryRun && opt.LocalIngest == nil {
		return st, fmt.Errorf("memory pull: LocalIngest required unless DryRun")
	}
	// s660/s678: default filter_subject from tenant + role when unset (CLI also pre-resolves).
	// Org-shaped tenants remap to dept.* (or dept.<department>) so filter matches wire subjects.
	configuredRole := c.cfg.Role
	configuredTenant := c.Tenant()
	opt.Filter = DefaultMemoryPullFilterForRoleWithDept(opt.Filter, configuredTenant, configuredRole, c.cfg.PullAllowSuffix, c.cfg.Department)
	st.Stream = opt.Stream
	st.Consumer = opt.Name
	st.Filter = opt.Filter

	// dept.* + Role + org (or other off-tenant) Tenant cannot create: omit Role and
	// the Role-gated Tenant bind. Proven path is no-Role 201. dual_write stays off.
	pull := c
	if OmitPullRoleForDeptFilter(opt.Filter, configuredTenant, configuredRole) {
		pull = c.withoutRoleGatedTenantBind()
	}

	if _, err := pull.CreateConsumer(ctx, opt.Stream, opt.Name, opt.Filter); err != nil {
		st.LastError = err.Error()
		if hint := MemoryPullRoleTenantACLHint(opt.Filter, configuredTenant, configuredRole); hint != "" {
			st.LastError = st.LastError + "; " + hint
		}
		return st, fmt.Errorf("memory pull create consumer: %w", err)
	}
	st.CreateOK = true

	seen := map[string]struct{}{}
	for {
		if ctx.Err() != nil {
			applyMemoryPullFilterMissHint(&st)
			return st, ctx.Err()
		}
		if opt.MaxLoops > 0 && st.Loops >= opt.MaxLoops {
			applyMemoryPullFilterMissHint(&st)
			return st, nil
		}
		st.Loops++

		msgs, err := pull.ConsumerFetch(ctx, opt.Stream, opt.Name, opt.Batch, opt.MaxWait)
		if err != nil {
			st.Errors++
			st.LastError = err.Error()
			if opt.MaxLoops == 1 {
				return st, err
			}
			// Back off slightly on fetch errors in long-run mode.
			select {
			case <-ctx.Done():
				return st, ctx.Err()
			case <-time.After(500 * time.Millisecond):
			}
			continue
		}
		if len(msgs) == 0 {
			if opt.MaxLoops > 0 && st.Loops >= opt.MaxLoops {
				applyMemoryPullFilterMissHint(&st)
				return st, nil
			}
			continue
		}

		var ackSeqs []uint64
		for _, msg := range msgs {
			st.Fetched++
			env, key, ok := MapStreamMessageToEnvelope(msg)
			if !ok {
				st.Skipped++
				if opt.OnMessage != nil {
					opt.OnMessage(msg, env, true, nil)
				}
				// Ack empty/unusable to avoid poison redelivery.
				if opt.Ack {
					ackSeqs = append(ackSeqs, msg.Seq)
				}
				continue
			}
			if key != "" {
				if _, dup := seen[key]; dup {
					st.Skipped++
					if opt.OnMessage != nil {
						opt.OnMessage(msg, env, true, nil)
					}
					if opt.Ack {
						ackSeqs = append(ackSeqs, msg.Seq)
					}
					continue
				}
				seen[key] = struct{}{}
			}

			var ingErr error
			if opt.DryRun {
				// no-op
			} else {
				ingErr = opt.LocalIngest(ctx, env)
			}
			if opt.OnMessage != nil {
				opt.OnMessage(msg, env, false, ingErr)
			}
			if ingErr != nil {
				st.Errors++
				st.LastError = ingErr.Error()
				// Do not ack on ingest failure so message can redeliver.
				continue
			}
			st.Ingested++
			if opt.Ack {
				ackSeqs = append(ackSeqs, msg.Seq)
			}
		}
		if opt.Ack && len(ackSeqs) > 0 {
			if _, err := pull.ConsumerAck(ctx, opt.Stream, opt.Name, ackSeqs...); err != nil {
				st.Errors++
				st.LastError = err.Error()
			} else {
				st.Acked += len(ackSeqs)
			}
		}
	}
}

// applyMemoryPullFilterMissHint records an honest last_error when create
// succeeded but fetched=0 and filter_subject is an org-id events filter that
// cannot match dept.*.events.*. Does not invent pull success. create_ok alone
// is not proof messages were pulled.
func applyMemoryPullFilterMissHint(st *MemoryPullStats) {
	if st == nil || st.Fetched > 0 || strings.TrimSpace(st.LastError) != "" {
		return
	}
	if hint := MemoryPullFilterMismatchHint(st.Filter); hint != "" {
		st.LastError = hint
	}
}

func bytesTrimSpace(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch t := v.(type) {
			case string:
				if s := strings.TrimSpace(t); s != "" {
					return s
				}
			case fmt.Stringer:
				if s := strings.TrimSpace(t.String()); s != "" {
					return s
				}
			}
		}
	}
	return ""
}

func intFromAny(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	case json.Number:
		i, _ := t.Int64()
		return int(i)
	default:
		return 0
	}
}
