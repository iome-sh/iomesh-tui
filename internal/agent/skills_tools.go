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
		return b.String(), nil
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
		return b.String(), nil
	})
}
