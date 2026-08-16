# Advanced Memory install (TUI · maximize benefit)

**Serial:** free eng **s1525** · residual-honest  
**Audience:** operators who want the most from local Memory with `iomesh-tui`  
**Product host:** [`iomesh-memory-mcp`](https://github.com/iome-sh/iomesh-memory-mcp)  
**Kernel:** [`github.com/iome-sh/memory`](https://github.com/iome-sh/memory)

This guide layers **requirements that improve Memory quality** for the TUI path. It does **not** invent Memory GA, freemium hosted palace, or dual_write ON.

---

## Honesty locks

| Claim | Truth |
|-------|--------|
| **default path** | Hash embeddings · FS palace · **no** Qdrant · **no** ONNX required |
| **ONNX** | Optional · set `MEMORY_ONNX_MODEL_PATH` on the **MCP host** process · improves hybrid / semantic ranking |
| **Qdrant** | **Not required** for TUI `/memory` · lean host search reports `qdrant=off` · kernel has optional VectorStore for custom Go |
| **Docker / Podman** | Host: product compose or binary · Qdrant container only if you experiment with kernel VectorStore residual |
| **dual_write** | **OFF** · local-primary |
| **not Memory GA** | Advanced install ≠ invent bare / Edge / hosted Memory GA |
| **package / map** | Sample plugin map ≠ Connected / Agent Plugins GA |

---

## Benefit ladder (what actually helps TUI)

| Level | What you run | TUI benefit | Extra deps |
|-------|----------------|-------------|------------|
| **L0 Baseline** | `iomesh-memory-mcp` + TUI attach | Ingest / retrieve / list / compact-status / facts-as-of (tools present) · auto-recall fail-open | Go + binary or Docker |
| **L1 Operator UX** | + `[memory] auto_recall` · optional `auto_ingest` · durable `palace-root` · tenant | Continuous context inject · durable palace across sessions | Config only |
| **L2 Advanced slash** | + builtin `memory-advanced-agent` skill · `/memory status` inventory | related / digest / timeline / semantic / HITL supersede·compact when **tools exist** | Skills on (default builtin) |
| **L3 Semantic quality** | + `MEMORY_ONNX_MODEL_PATH` on **host** | Stronger hybrid retrieve + `/memory semantic` ranking | ONNX model download · more CPU/RAM |
| **L4 Kernel residual** | Optional Qdrant container + custom Go `VectorStore` | **Not** wired into lean host search today · do not expect TUI to use it | Docker/Podman Qdrant |

**Maximize practical benefit for most users: L0 + L1 + L2.** Add **L3** when semantic quality matters. Treat **L4** as residual, not a TUI install requirement.

---

## L0 — Baseline product host (required)

```bash
# Public install — no GOPRIVATE
go install github.com/iome-sh/iomesh-memory-mcp/cmd/iomesh-memory-mcp@main

mkdir -p ~/.iomesh/palace
iomesh-memory-mcp \
  -palace-root ~/.iomesh/palace \
  -tenant default \
  -http-addr :8080 \
  -http-path /mcp

curl -fsS http://127.0.0.1:8080/healthz
# dual_write=off · not_memory_ga=true · embeddings=hash · qdrant=off
```

**Docker / Podman (product compose):**

```bash
git clone https://github.com/iome-sh/iomesh-memory-mcp.git
cd iomesh-memory-mcp
docker compose up --build
# or: podman compose up --build
curl -fsS http://127.0.0.1:8080/healthz
```

Attach TUI (primary TOML):

```toml
[mcp]
enabled = true
[[mcp.servers]]
name = "iomesh-memory-mcp"
url = "http://127.0.0.1:8080/mcp"
allow_loopback = true
mutating = true

[memory]
enabled = true
server = "iomesh-memory-mcp"
tenant = "default"
auto_recall = true
auto_ingest = false   # opt-in for durable turn write-back
dual_write = false
```

Verify:

```bash
iomesh mcp --connect
# expect memory_* tools under iomesh-memory-mcp
iomesh
# /memory status · /memory ingest … · /memory recall …
```

Sample plugin map (optional): [`examples/agent-plugins/iomesh-memory-mcp`](../../examples/agent-plugins/iomesh-memory-mcp/).

---

## L1 — Operator knobs (max benefit, zero infra)

| Knob | Why |
|------|-----|
| Durable `-palace-root` / volume | Survives restarts · multi-session palace |
| Stable `tenant` | Isolation + consistent recall |
| `auto_recall = true` | Inject `<memory-context>` each turn (fail-open) |
| `auto_ingest = true` (opt-in) | Persist user+assistant turns after success |
| Local LLM pin (optional) | `iomesh -m ollama-llama3.2` · local-edge cost-max · ≠ platform GPU |

---

## L2 — Advanced Memory surfaces (TUI)

Once MCP is attached, use residual-honest advanced inventory:

```text
/memory status          # base + advanced tool presence pulse
/memory write …         # durable fact (MCP memory_write · not a turn)
/memory semantic …      # tier-4 semantic (MCP memory_search_semantic)
/memory related --seed …
/memory facts-as-of --as-of <RFC3339>
/memory timeline
/memory compact-status
/memory digest …
/memory supersede … --i-confirm    # HITL
/memory trigger-compact --i-confirm  # HITL
```

Builtin skill **`memory-advanced-agent`** steers agents; skill-only · dual_write OFF · not Memory GA.  
Tools available depend on lean host surface. Historical s1508/s1509 attach stamp is **tools=6** (past evidence — do **not** invent a new live tools=N). Lean host recopies `memory_write` / `memory_related` / `memory_supersede_entity` (s2006) when the attached host lists them — **PASS ≠ invent full platform sidecar parity**.

---

## L3 — Optional ONNX embeddings (maximize semantic quality)

Host process must see the model path (**not** the TUI process alone):

```bash
# Download helper lives in the memory kernel repo (public):
git clone https://github.com/iome-sh/memory.git
cd memory
go run ./scripts/download_onnx_model.go

export MEMORY_ONNX_MODEL_PATH=/absolute/path/to/model   # hugot dir or .onnx
# optional:
# export MEMORY_EMBEDDING_STRICT=true
# export MEMORY_HUGOT_BACKEND=go   # or ort / auto

iomesh-memory-mcp -palace-root ~/.iomesh/palace -tenant default -http-addr :8080
curl -fsS http://127.0.0.1:8080/healthz
# embeddings should be "onnx" when load succeeds
```

Compose:

```bash
export MEMORY_ONNX_MODEL_PATH=/absolute/path/to/model
# optional bind-mount in product compose if running containerized
docker compose up --build
```

**Requirements:** disk for model · CPU (or ORT/CUDA residual per kernel docs) · more RAM than hash path.  
**Honesty:** ONNX ≠ Memory GA · dual_write OFF · Qdrant still off for lean host.

---

## L4 — Optional Qdrant (Docker / Podman) — residual honesty

Kernel docs show a Qdrant container for the optional **VectorStore** API:

```bash
# Podman (from memory README)
podman run -d --name qdrant \
  -p 6333:6333 -p 6334:6334 \
  -v qdrant_storage:/qdrant/storage:z \
  qdrant/qdrant

# Docker equivalent
docker run -d --name qdrant \
  -p 6333:6333 -p 6334:6334 \
  -v qdrant_storage:/qdrant/storage \
  qdrant/qdrant
```

| Expectation | Truth |
|-------------|--------|
| Improves TUI `/memory recall` automatically | **No** — lean `iomesh-memory-mcp` does **not** wire Qdrant into hybrid search (`healthz.qdrant=off`) |
| Useful today | Custom Go using `memory.NewVectorStore` / compaction callbacks · future host work |
| Required for advanced TUI | **No** |

Do **not** claim hosted multi-tenant Qdrant palace or freemium cloud Memory from running a local container.

---

## Recommended “maximize benefit” stack (checklist)

1. [ ] Install/run **`iomesh-memory-mcp`** (binary or compose) with durable palace volume  
2. [ ] TUI `[mcp]` + `[memory]` with **auto_recall** · dual_write **false**  
3. [ ] Confirm `iomesh mcp --connect` lists memory tools  
4. [ ] Optional: **auto_ingest** for durable turns  
5. [ ] Optional: **ONNX** via `MEMORY_ONNX_MODEL_PATH` on the host for better semantic  
6. [ ] Optional: Ollama local pin for offline LLM  
7. [ ] Use `/memory status` + advanced slash as tools allow  
8. [ ] Skip Qdrant unless you need kernel VectorStore residual (not TUI-required)

---

## Related

| Doc | Role |
|-----|------|
| [memory-edge-usage-demo.md](./memory-edge-usage-demo.md) | Signup → integrations → local memory → usage |
| [memory-mcp.md](./memory-mcp.md) | Full Memory MCP architecture + slash table |
| [examples/agent-plugins/iomesh-memory-mcp](../../examples/agent-plugins/iomesh-memory-mcp/) | Sample plugin map |
| Peer host README | Product flags · compose · ONNX env |
| Peer kernel README | ONNX download · optional Qdrant VectorStore |

## Non-goals

- Auto-provision Memory on signup  
- Require Qdrant for TUI Memory  
- dual_write ON as primary path  
- Invent Memory GA / freemium hosted palace  
- Claim lean host uses Qdrant when `qdrant=off`
