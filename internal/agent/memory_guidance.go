package agent

import "strings"

// MemoryNextStepLines residual-honest post /memory surfaces (s1831).
// Dual path after status/help/digest (and peer honesty footers): in-session setup continuum
// vs cold start. Peer of OnboardNextStepLines (s1825) · IntegrationsNextStepLines (s1727) ·
// setup next-step continuum (s1686–s1723).
// dual_write OFF · not Memory GA · local-primary · package wire ≠ Connected ·
// soft ≠ invent live dogfood · free eng s1831. Never invent Connected / Memory GA from memory slash alone.
func MemoryNextStepLines() []string {
	return []string{
		"next: dual path residual-honest after memory surfaces",
		"      if TUI/session running → /setup preflight · /setup reload · optional /memory digest · /onboard next memory|memory-pull",
		"      else cold start → restart iomesh · iomesh setup preflight · optional iomesh memory pull",
		"note: dual_write OFF · not Memory GA · local-primary · package wire ≠ Connected · soft ≠ invent live dogfood · free eng s1831",
	}
}

// MemoryAdvancedAgentGuidanceNote residual-honest system note (s1291 + s1296 + s1301 + s1311).
// Injected on AttachMCP. Does not invent Memory GA / silent supersede / auto multi-hop.
func MemoryAdvancedAgentGuidanceNote() string {
	return strings.TrimSpace(`memory advanced (residual-honest agent path · s1291):
Opt-in advanced memory only — default auto-recall stays single-hop memory_retrieve.
1. related: multi-hop lite · prefer_shorter_hops omit=true · seed_query lean + query legacy · not full graph RAG
2. facts-as-of: MCP-first K4 lite · not dual-clock Graphiti
3. supersede: MCP-first A3 lite · requires HITL / --i-confirm · entity_key lean + entity legacy · not NLP contradiction
4. digest: ops GA-path framing · knowledge/analytical Beta
5. patterns/anomalies: ops pulse Beta · not medical · not OTel · not invent GA window engine
6. timeline: MCP-first temporal timeline · filters before limit · read-only (s1296)
7. compact-status: MCP-first Palace tier counts residual · read-only · not auto-compact product (s1296)
8. semantic: MCP-first tier-4 semantic facts residual · empty ≠ invent (s1301)
9. ingest-event: MCP-first s138 T1 temporal event telemetry · not conversation turn (s1301)
10. trigger-compact: MCP-first RecMem advisory · requires HITL / --i-confirm · not invent compaction green (s1311)
11. write: MCP-first durable fact · memory_write · summary/full/tags/tier/entity_key · not conversation turn (s2006)
12. status: /memory status prints advanced MCP inventory residual (s1311)

Slash mirrors: /memory write · /memory related|facts-as-of|supersede|digest|patterns|anomalies|timeline|compact-status|semantic|ingest-event|trigger-compact|status
Skill: read_skill memory-advanced-agent when available

Locks (never violate):
- dual_write OFF · not Memory GA · no invent GA
- multi-hop lite ≠ graph RAG · PreferShorterHops omit=true
- supersede requires HITL / --i-confirm · never silent mutate
- write ≠ turn · never invent memory_id · entity_key host-defaults WriteAndSupersede
- patterns/anomalies not medical · not OTel · no invent GA window engine
- timeline/compact-status read-only · memory_trigger_compact requires HITL (s1311 shipped)
- semantic empty ≠ invent · ingest-event never invent memory_id · not conversation turn
- trigger-compact requires HITL / --i-confirm · RecMem advisory · not invent compaction green
- opt-in only · never auto multi-hop on default recall · fail-open
- lean recopy is not a new live tools=N stamp · s1508/s1509 tools=6 stays past`)
}
