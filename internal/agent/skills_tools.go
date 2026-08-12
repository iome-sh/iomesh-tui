package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/iome-sh/iomesh-tui/internal/router"
	"github.com/iome-sh/iomesh-tui/internal/skills"
)

// skillToolNames are registry names for skill meta tools (list/read).
var skillToolNames = []string{
	"list_skills",
	"read_skill",
}

// SkillsNextStepLines residual-honest post list_skills / read_skill (s1837).
// Dual path after skills list/read (and skills re-scan via /setup reload): in-session
// setup continuum vs cold start. Peer of PluginsNextStepLines (s1829) · OnboardNextStepLines
// (s1825) · MemoryNextStepLines (s1831) · IntegrationsNextStepLines (s1727).
// skills re-scan ≠ invent Connected · package wire ≠ Connected · dual_write OFF ·
// not Agent Plugins GA · not Memory GA · free eng s1837. Never invent success/Connected
// from list/read alone (no dedicated /skills slash — catalog + tools + reload only).
func SkillsNextStepLines() []string {
	return []string{
		"next: dual path residual-honest after skills list/read or skills reload",
		"      if TUI/session running → /setup preflight · /setup reload (skills re-scan · package wire ≠ Connected) · optional list_skills tool · /onboard next setup",
		"      else cold start → restart iomesh · iomesh setup preflight",
		"note: skills re-scan ≠ invent Connected · package wire ≠ Connected · dual_write OFF · not Agent Plugins GA · not Memory GA · free eng s1837",
	}
}

// appendSkillsNextStep appends residual-honest next-step lines to a successful
// list_skills / read_skill body. Errors stay bare (never invent success).
func appendSkillsNextStep(body string) string {
	body = strings.TrimRight(body, "\n")
	return body + "\n" + strings.Join(SkillsNextStepLines(), "\n") + "\n"
}

// UnregisterSkillsTools removes list_skills and read_skill from the registry.
// Used by Runtime.ReplaceSkills before re-attach (s1670 /setup reload skills re-scan).
func (r *ToolRegistry) UnregisterSkillsTools() {
	if r == nil || r.funcs == nil {
		return
	}
	for _, name := range skillToolNames {
		delete(r.funcs, name)
		delete(r.meta, name)
	}
}

// RegisterSkillsTools adds list_skills / read_skill (read-only).
func (r *ToolRegistry) RegisterSkillsTools(cat *skills.Catalog) {
	if cat == nil || cat.Len() == 0 {
		return
	}
	r.register("list_skills", false, router.Tool{
		Type: "function",
		Function: router.ToolFunction{
			Name:        "list_skills",
			Description: "List available skills (name + description). Use read_skill for full instructions.",
			Parameters:  json.RawMessage(`{"type":"object","properties":{}}`),
		},
	}, func(ctx context.Context, args string) (string, error) {
		var b strings.Builder
		for _, sk := range cat.List() {
			fmt.Fprintf(&b, "%s\t%s\n", sk.Name, strings.ReplaceAll(sk.Description, "\n", " "))
		}
		return appendSkillsNextStep(b.String()), nil
	})

	r.register("read_skill", false, router.Tool{
		Type: "function",
		Function: router.ToolFunction{
			Name:        "read_skill",
			Description: "Read the full body of a skill by name (from list_skills).",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`),
		},
	}, func(ctx context.Context, args string) (string, error) {
		var p struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal([]byte(args), &p); err != nil {
			return "", err
		}
		sk, ok := cat.Get(p.Name)
		if !ok {
			return "", fmt.Errorf("unknown skill %q (try list_skills)", p.Name)
		}
		var b strings.Builder
		fmt.Fprintf(&b, "# skill: %s\n", sk.Name)
		if sk.Description != "" {
			fmt.Fprintf(&b, "description: %s\n\n", sk.Description)
		}
		b.WriteString(sk.Body)
		return appendSkillsNextStep(b.String()), nil
	})
}
