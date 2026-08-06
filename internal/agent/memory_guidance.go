package agent

import "strings"

// MemoryAdvancedAgentGuidanceNote residual-honest system note (s1291).
// Injected on AttachMCP. Does not invent Memory GA / silent supersede / auto multi-hop.
func MemoryAdvancedAgentGuidanceNote() string {
	return strings.TrimSpace(`memory advanced (residual-honest agent path · s1291):
Opt-in advanced memory only — default auto-recall stays single-hop memory_retrieve.
1. related: multi-hop lite · prefer_shorter_hops omit=true · not full graph RAG
2. facts-as-of: MCP-first K4 lite · not dual-clock Graphiti
3. supersede: MCP-first A3 lite · requires HITL / --i-confirm · not NLP contradiction
4. digest: ops GA-path framing · knowledge/analytical Beta
5. patterns/anomalies: ops pulse Beta · not medical · not OTel · not invent GA window engine

Slash mirrors: /memory related|facts-as-of|supersede|digest|patterns|anomalies
Skill: read_skill memory-advanced-agent when available

Locks (never violate):
- dual_write OFF · not Memory GA · no invent GA
- multi-hop lite ≠ graph RAG · PreferShorterHops omit=true
- supersede requires HITL / --i-confirm · never silent mutate
- patterns/anomalies not medical · not OTel · no invent GA window engine
- opt-in only · never auto multi-hop on default recall · fail-open`)
}
