package agent

import (
	"context"
)

// Approval is the operator decision for a mutating tool call.
type Approval int

const (
	// ApprovalDeny rejects the tool call.
	ApprovalDeny Approval = iota
	// ApprovalOnce allows this single invocation.
	ApprovalOnce
	// ApprovalAlways allows this tool name for the rest of the session.
	ApprovalAlways
)

// Approver is invoked for mutating tools when Yolo is false.
// Returning an error is treated as deny.
type Approver func(ctx context.Context, toolName, arguments string) (Approval, error)

// SetApprover installs an interactive (or custom) approval callback.
func (rt *Runtime) SetApprover(a Approver) {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.approver = a
}

// AllowToolSession grants ApprovalAlways for name without prompting.
func (rt *Runtime) AllowToolSession(name string) {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.sessionAllow == nil {
		rt.sessionAllow = map[string]bool{}
	}
	rt.sessionAllow[name] = true
}

// ToolAllowedSession reports whether name was approved always this session.
func (rt *Runtime) ToolAllowedSession(name string) bool {
	if rt == nil {
		return false
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.sessionAllow[name]
}

func (rt *Runtime) decideApproval(ctx context.Context, tool, args string) Approval {
	if rt.cfg.Yolo {
		return ApprovalOnce
	}
	rt.mu.Lock()
	if rt.sessionAllow != nil && rt.sessionAllow[tool] {
		rt.mu.Unlock()
		return ApprovalOnce
	}
	approver := rt.approver
	rt.mu.Unlock()
	if approver == nil {
		return ApprovalDeny
	}
	dec, err := approver(ctx, tool, args)
	if err != nil {
		return ApprovalDeny
	}
	if dec == ApprovalAlways {
		rt.AllowToolSession(tool)
		return ApprovalOnce
	}
	return dec
}
