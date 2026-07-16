# Security Policy

## Supported versions

| Version | Supported |
|---------|-----------|
| `v0.1.x` (latest minor on `main`) | ✅ security fixes |
| `main` (unreleased) | ✅ development tip |
| older 0.x tags | best-effort until EOL notice |

## Reporting a vulnerability

**Please do not open a public GitHub issue for security vulnerabilities.**

Preferred channels (in order):

1. **GitHub Security Advisory** (private) — Security → Advisories → Report a vulnerability on this repository  
2. Email **security@iome.sh**

Include:

- Description of the issue and impact  
- Reproduction steps or proof-of-concept  
- Affected commit / tag if known  

We aim to acknowledge reports within **72 hours** and provide a remediation timeline after triage.

## Threat model (coding agent)

`iomesh` is a **local developer tool** that can read/write files and run shell commands **inside a workspace** when the user approves (or passes `--yolo`).

| Trust boundary | Posture |
|----------------|---------|
| Workspace filesystem | Path jail + symlink escape checks; read size caps |
| Shell tool | Requires approval/`--yolo`; secret env scrubbed; hard-denied catastrophic patterns |
| LLM / mesh HTTP | `http`/`https` only; error bodies redacted; response size limits |
| Configuration | Prefer `env_key` over inline `api_key`; never commit secrets |
| Logs | Credential-like strings redacted before warn/error logs |

### What this is *not*

- Not a multi-tenant remote agent sandbox
- Not a substitute for OS-level sandboxing (Seatbelt, bubblewrap, containers)
- `--yolo` grants the model authority to mutate the workspace and run shell commands — treat it as **full trust of the model + tools**

## Hardening checklist for operators

1. Prefer **interactive approval** (no `--yolo`) for untrusted prompts/repos  
2. Scope `-C` / workspace to the smallest project directory  
3. Store API keys in environment variables (`DEEPSEEK_API_KEY`, `XAI_API_KEY`, `IOMESH_API_KEY`, MCP token envs)  
4. Do not put secrets in `config.toml` committed to git  
5. Review shell tool output before pasting into tickets/logs  
6. Disable subagents if not needed: `IOMESH_SUBAGENTS=0`  
7. Point model `base_url` only at trusted OpenAI-compatible endpoints  
8. ACP serve: keep loopback default; use `--token` if binding beyond localhost  
9. MCP HTTP: use `oauth_token_env` / `client_secret_env`, never inline secrets  

See also [docs/security.md](docs/security.md) (architecture + residual risks).

## Dependency security

```bash
make vuln   # govulncheck ./...
make test-race
```

CI runs tests, `go vet`, and `govulncheck` on every PR.

## Disclosure preference

Coordinated disclosure: we prefer to ship a fix (or mitigating docs) before public write-ups when the issue is exploitable in default configurations.
