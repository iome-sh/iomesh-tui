package agent

import "strings"

// MemoryAdvancedAgentGuidanceNote residual-honest system note (s1291 + s1296 + s1301 + s1311).
// Injected on AttachMCP. Does not invent Memory GA / silent supersede / auto multi-hop.
func MemoryAdvancedAgentGuidanceNote() string {
	return strings.TrimSpace(`memory advanced (residual-honest agent path · s1291):
Opt-in advanced memory only — default auto-recall stays single-hop memory_retrieve.
1. related: multi-hop lite · prefer_shorter_hops omit=true · not full graph RAG
2. facts-as-of: MCP-first K4 lite · not dual-clock Graphiti
3. supersede: MCP-first A3 lite · requires HITL / --i-confirm · not NLP contradiction
4. digest: ops GA-path framing · knowledge/analytical Beta
5. patterns/anomalies: ops pulse Beta · not medical · not OTel · not invent GA window engine
6. timeline: MCP-first temporal timeline · filters before limit · read-only (s1296)
7. compact-status: MCP-first Palace tier counts residual · read-only · not auto-compact product (s1296)
8. semantic: MCP-first tier-4 semantic facts residual · empty ≠ invent (s1301)
9. ingest-event: MCP-first s138 T1 temporal event telemetry · not conversation turn (s1301)
10. trigger-compact: MCP-first RecMem advisory · requires HITL / --i-confirm · not invent compaction green (s1311)
11. status: /memory status prints advanced MCP inventory residual (s1311)

Slash mirrors: /memory related|facts-as-of|supersede|digest|patterns|anomalies|timeline|compact-status|semantic|ingest-event|trigger-compact|status
Skill: read_skill memory-advanced-agent when available

Locks (never violate):
- dual_write OFF · not Memory GA · no invent GA
- multi-hop lite ≠ graph RAG · PreferShorterHops omit=true
- supersede requires HITL / --i-confirm · never silent mutate
- patterns/anomalies not medical · not OTel · no invent GA window engine
- timeline/compact-status read-only · memory_trigger_compact requires HITL (s1311 shipped)
- semantic empty ≠ invent · ingest-event never invent memory_id · not conversation turn
- trigger-compact requires HITL / --i-confirm · RecMem advisory · not invent compaction green
- opt-in only · never auto multi-hop on default recall · fail-open`)
}
