package agent

import "time"

// EventType classifies runtime events for TUI / ACP adapters.
type EventType string

const (
	EventTextDelta      EventType = "text_delta"
	EventThinkingDelta  EventType = "thinking_delta"
	EventModelSelected  EventType = "model_selected"
	EventLLMDone        EventType = "llm_done"
	EventToolStart      EventType = "tool_start"
	EventToolEnd        EventType = "tool_end"
	EventToolError      EventType = "tool_error"
	EventToolDenied     EventType = "tool_denied"
	EventToolPermission EventType = "tool_permission" // optional UI hint before approver runs
	EventMeshContext    EventType = "mesh_context"
	EventMeshPolicy     EventType = "mesh_policy"
	EventSubagentStart  EventType = "subagent_start"
	EventSubagentEnd    EventType = "subagent_end"
)

// Event is a UI-facing runtime notification.
type Event struct {
	Type     EventType
	Text     string
	Model    string
	Tool     string
	Tokens   int
	CostUSD  float64
	Duration time.Duration
}
