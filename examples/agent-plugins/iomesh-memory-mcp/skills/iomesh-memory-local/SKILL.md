---
name: iomesh-memory-local
description: Residual-honest product edge memory playbook (sample plugin skill; public iomesh-memory-mcp · not Memory GA · dual_write OFF · package load ≠ Connected)
---

# iomesh-memory-mcp local edge (sample plugin skill)

Short playbook for operators dogfooding the **iomesh-memory-mcp** product Agent Plugins sample package — stdio **map only** against the public product edge Memory MCP binary.

This skill is a **playbook only** — guidance text the agent may load when plugins are enabled. It does not install software, mint secrets, auto dual-write, or invent Memory product green.

## Orientation

1. **What this package is** — portable product sample under `examples/agent-plugins/iomesh-memory-mcp` with closed `plugin.json`, `mcp.json` (stdio `memory` → `iomesh-memory-mcp`), and this skill. Map-only; binary not shipped.
2. **How it loads** — only when the operator **opts in** via TOML `[plugins] enabled = true` and a `dirs` entry pointing at this package root (or a parent of package roots). Default is **disabled**.
3. **Discover / map vs Connected** — `agentplugins.Discover` and `MCPServersFromPlugins` success means the package validated and mapped. It is **not** process Connected, install APPLY green, marketplace install, Agent Plugins product GA, or Memory GA.
4. **Connect requirement** — the operator must install **`iomesh-memory-mcp`** (public) and put it on **PATH**. Attach is **fail-open** if the binary is missing (no invent green).

## Public install (s1478 · residual-honest)

Both product edge repos are **public** — **no GOPRIVATE** / PAT required:

```bash
# Product MCP host (preferred)
go install github.com/iome-sh/iomesh-memory-mcp/cmd/iomesh-memory-mcp@main
# or: clone github.com/iome-sh/iomesh-memory-mcp && go build ./cmd/iomesh-memory-mcp

# Kernel module (public tip)
go get github.com/iome-sh/memory@main
```

Attach after install:

- **HTTP preferred:** run host → `http://127.0.0.1:8080/mcp` (TUI `[[mcp.servers]]` URL)
- **stdio:** command `iomesh-memory-mcp` on PATH (this package maps that)
- **docker compose still valid:** in product repo `docker compose up --build` → image `iomesh-memory-mcp:local`

## Operator checklist (dogfood)

1. `go install github.com/iome-sh/iomesh-memory-mcp/cmd/iomesh-memory-mcp@main` — **no GOPRIVATE** · confirm `which iomesh-memory-mcp`.
2. Point `[plugins].dirs` at this package (see package `README.md`).
3. Set `enabled = true` (opt-in only).
4. Restart / reload the TUI session so skills catalog merge and plugin MCP map can see this package.
5. Prefer dual_write **OFF** (default). Local-primary palace; not freemium hosted palace.
6. Mutating tools remain **HITL / approval-gated** — refuse without operator confirm.

## Explicit non-claims

| Do **not** claim | Truth |
|------------------|--------|
| Agent Plugins GA | Sample package + client candidacy only |
| Memory GA | Local-primary edge residual · not product Memory green |
| dual_write ON | dual_write remains **OFF** (unchanged default) |
| freemium hosted palace | Hosted Palace sunset until scale · local-primary only |
| install APPLY / Connected green | Discover/load/map ≠ install or process Connected |
| public OSS = platform GA | Public edge packages ≠ invent Memory GA / freemium palace |
| Secrets in plugin.json / mcp.json | Portable package fields must not carry secrets |
| Map success = tools always available | Connect needs binary on PATH; MCP tools stay **approval-gated** |
| residual aion Memory sample | **Removed** from TUI tree (s1517) · product host is `iomesh-memory-mcp` only |

## Related

- Package README: `examples/agent-plugins/iomesh-memory-mcp/README.md`
- s1517 product-only: this package is the in-tree Memory sample (hello-iome is skills-only)
- Architecture: `docs/architecture/agent-plugins.md` · `docs/architecture/memory-mcp.md` (s1478 public product attach)
- Spec peer: [Agent Plugins 1.0.0](https://agent-plugins.org/specification)

## Advanced install tip (s1525)

- Default: hash embeddings · no Qdrant · dual_write OFF · not Memory GA.
- Maximize semantic: set `MEMORY_ONNX_MODEL_PATH` on the **MCP host** (not only TUI env).
- Qdrant docker/podman is **optional residual** (kernel VectorStore) — lean host does not wire it into search.
- Operator doc: `docs/architecture/memory-advanced-install.md`.
