# Documentation

| Doc | Description |
|-----|-------------|
| [security.md](security.md) | Threat model, controls, residual risks |
| [architecture/overview.md](architecture/overview.md) | Package map, runtime flow, milestones |
| [architecture/llm-cascade.md](architecture/llm-cascade.md) | Multi-model catalog (DeepSeek, Grok, Gemini, Vertex) + cascade |
| [architecture/permissions.md](architecture/permissions.md) | Tool approval (y/n/a, yolo) |
| [architecture/subagents.md](architecture/subagents.md) | Spawn, parallel, worktrees |
| [architecture/sessions.md](architecture/sessions.md) | Session persistence |
| [architecture/tui.md](architecture/tui.md) | Full-screen TUI, themes, multi-line |
| [architecture/acp.md](architecture/acp.md) | ACP stdio + WebSocket |
| [architecture/skills.md](architecture/skills.md) | SKILL.md loader |
| [architecture/mcp.md](architecture/mcp.md) | MCP stdio/HTTP, resources, prompts, OAuth |
| [architecture/mesh-dogfood.md](architecture/mesh-dogfood.md) | Stage mesh smoke (`iomesh mesh smoke`; legacy dogfood) |
| [architecture/mesh-deeper.md](architecture/mesh-deeper.md) | Lineage context, policy gates, local metering, portal catalog |
| [architecture/memory-mcp.md](architecture/memory-mcp.md) | Memory Palace + temporal MCP integration plan |
| [architecture/memory-edge-usage-demo.md](architecture/memory-edge-usage-demo.md) | Residual-honest signup → integrations → local memory (kernel + MCP) → show usage demo |
| [architecture/memory-advanced-install.md](architecture/memory-advanced-install.md) | Advanced Memory install ladder: ONNX optional · Qdrant residual · maximize TUI benefit |
| [architecture/setup-lifecycle.md](architecture/setup-lifecycle.md) | Setup lifecycle: `iomesh setup init\|preflight` · `/setup` slash · skill · managed config · residual honesty |
| [architecture/agent-integrations-setup.md](architecture/agent-integrations-setup.md) | `/integrations` MCP catalog/plan + portal HITL |
| [EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md](EDGE_MEMORY_E4_CLIENT_ATTACH_EVIDENCE.md) | E4 MCP client attach dogfood evidence stamp |

Project process docs (repo root): [CONTRIBUTING](../CONTRIBUTING.md) · [SECURITY](../SECURITY.md) · [SUPPORT](../SUPPORT.md) · [RELEASING](../RELEASING.md) · [CHANGELOG](../CHANGELOG.md)
