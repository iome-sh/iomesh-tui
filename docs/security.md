# Security architecture

This document describes the security posture of **iomesh-tui** for operators and contributors preparing for public release.

## Trust model

```
┌─────────────────────────────────────────────────────────┐
│  User (trusted)                                         │
│   CLI flags · config.toml · env secrets · --yolo        │
└───────────────────────────┬─────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────┐
│  Agent runtime                                          │
│   approval gate · tool registry · subagent manager      │
└───────┬─────────────────────────────┬───────────────────┘
        │                             │
        ▼                             ▼
┌───────────────────┐       ┌─────────────────────────────┐
│ Workspace jail    │       │ LLM / Mesh HTTP             │
│ path + symlink    │       │ https/http only             │
│ size caps         │       │ redacted errors             │
└───────────────────┘       │ response size limits        │
        │                   └─────────────────────────────┘
        ▼
┌───────────────────┐
│ Shell tool        │
│ scrubbed env      │
│ policy denylist   │
│ timeout + truncate│
└───────────────────┘
```

## Controls by surface

### Filesystem (`internal/workspace`)

- All paths resolved under workspace root (`PathUnderRoot`)
- Symlink targets re-checked after `EvalSymlinks`
- Absolute paths outside root rejected
- NUL / drive-letter paths rejected
- Read size capped (`DefaultMaxReadBytes` = 2 MiB)
- Grep skips `.git`, `vendor`, `node_modules`, binaries, oversized files

### Shell (`run_shell`)

- Marked **mutating** → denied unless `--yolo` / future interactive approve
- `security.ValidateShellCommand` blocks empty, oversized, non-UTF8, NUL, and known catastrophic patterns (e.g. `rm -rf /`, curl\|bash)
- Child `Env` = `security.ScrubEnv` (drops `*_API_KEY`, tokens, etc.)
- Output truncated + redacted before return to the model

### LLM router (`internal/router`)

- `base_url` must be `http` or `https` (`file://` rejected)
- Userinfo stripped from URLs
- Error bodies redacted (`Bearer`, `sk-…`)
- Response body read limits
- API keys via `env_key` preferred; never printed by `iomesh models`

### I/O Mesh (`internal/iomesh`)

- Optional; disabled by default
- Invalid endpoint disables client
- Fail-open on network/context errors (agent continues offline)
- Redirect limit + URL re-validation
- Metrics payloads use redacted error strings

### Subagents

- Depth and concurrency caps
- Built-in explore/plan cannot write files
- Nested spawn disabled on builtins

## Operator recommendations

| Scenario | Recommendation |
|----------|----------------|
| Untrusted repo / prompt | No `--yolo`; review every mutator |
| CI automation | Pin model + workspace; consider read-only tools only |
| Self-hosted LLM | Trusted `base_url`; network isolation |
| Shared machine | User-private `~/.iomesh/`; no world-readable keys |

## Residual risks

1. **Approved shell is powerful** — policy denylist is incomplete by design; OS sandbox is out of scope for v0
2. **Model prompt injection** — a malicious repo can try to coerce tool use; approval gates are the primary control
3. **Local HTTP LLM endpoints** — loopback is allowed for DX; do not expose unauthenticated proxies on LAN
4. **Supply chain** — keep `go.sum` committed; run `make vuln` in CI

## Related

- [SECURITY.md](../SECURITY.md) — reporting process
- [CONTRIBUTING.md](../CONTRIBUTING.md) — test expectations
