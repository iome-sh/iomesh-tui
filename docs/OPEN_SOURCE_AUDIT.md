# Open-source launch audit

Checklist completed for making **iomesh-tui** a public repository. Re-run before each major release.

## Security

| Check | Status |
|-------|--------|
| No committed API keys / private keys / `.env` secrets | Pass (tests use fake `api_key = "k"` / redaction fixtures only) |
| No SR&ED / private control-plane ledger strings in tree | **Partial** — docs continuum serials stripped; private monorepo paths removed. Residual: git history of past PR titles/commits may still contain ledger serials (do not rewrite tags). Forward policy in CONTRIBUTING. |
| Path jail + shell policy + secret scrub covered by tests | Pass |
| Mutating tools fail-closed without approval/`--yolo` | Pass |
| ACP default bind loopback; token warned off-loopback | Pass |
| MCP HTTP URL scheme validation; OAuth secrets via env only | Pass |
| `.gitignore` covers `.env`, keys, `.iomesh/`, binaries | Pass |
| Residual risks documented in SECURITY.md / docs/security.md | Pass |
| Vulnerability reporting path (advisory + security@iome.sh) | Pass |

## Open-source process

| Artifact | Status |
|----------|--------|
| LICENSE (MIT) | Present |
| NOTICE (third-party acknowledgements) | Present |
| CODE_OF_CONDUCT | Present |
| CONTRIBUTING | Present |
| SECURITY | Present |
| SUPPORT | Present |
| CHANGELOG | Present |
| RELEASING | Present |
| PR template | Present |
| Issue templates (bug/feature) + security contact link | Present |
| CI (lint, test+race, build, govulncheck, ci-success) | Present |
| Dependabot (gomod + actions) | Present |
| README quick start + security callouts | Present |

## Maintainer actions after going public

1. GitHub → **Settings → General → Danger Zone → Change visibility → Public**  
2. Enable **Private vulnerability reporting** (Settings → Code security)  
3. Branch protection on `main`: require PR + status check **`ci-success`**  
4. (Optional) Add topics: `golang`, `llm`, `mcp`, `cli`, `coding-agent`  
5. Cut `v0.1.0` when ready ([RELEASING.md](../RELEASING.md))  
6. Do **not** publish private stage endpoints or production mesh URLs in issues/docs  

## Out of scope for open-source binary

- Multi-tenant remote agent hosting  
- OS-level sandbox (Seatbelt/bubblewrap) — recommended for untrusted workloads, not bundled  
- Guarantees about third-party LLM/MCP availability  
