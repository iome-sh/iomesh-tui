# Skills loader

Skills are markdown playbooks (`SKILL.md`) the agent can discover and read via tools.

## Discovery

When `[skills] enabled = true` (default):

| Path | Role |
|------|------|
| `<workspace>/.iomesh/skills/<name>/SKILL.md` | Project skills |
| `~/.iomesh/skills/<name>/SKILL.md` | User skills |
| `[skills].dirs` | Extra roots |

Load is fail-open: missing directories are skipped.

## Format

```markdown
---
name: review
description: Structured code review checklist
---

# Review skill

1. Diff against main
2. …
```

YAML frontmatter needs only `name` and `description` (minimal parser; no full YAML stack).

## Agent tools

| Tool | Mutating | Purpose |
|------|----------|---------|
| `list_skills` | no | Catalog names + descriptions |
| `read_skill` | no | Full skill body |

A compact catalog is also injected into the system prompt as `<skills>…</skills>`.

## CLI

```bash
iomesh skills
iomesh skills -C /path/to/workspace
```
