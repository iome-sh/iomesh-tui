# Open-source launch audit

**Maintainer process residual** — not a product spec and not a user guide. Visibility is already **public MIT**. Operators: [README.md](../README.md) · [SECURITY.md](../SECURITY.md).

Checklist completed for making **iomesh-tui** a public repository. Visibility flip is **complete** (public MIT). Re-run the process bar before each major release. Do **not** re-run a visibility flip.

Honesty locks: dual_write **OFF** · **not Memory GA** · catalog ≠ Connected. Public MIT ≠ Memory GA.

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

## Maintainer actions after going public (visibility already public)

Historical closeout from launch day. Completing this list was **not** a product claim and **does not** require flipping visibility again.

1. GitHub visibility → Public (**done** — flip complete). Do not re-run a visibility flip.  
2. Enable **Private vulnerability reporting** (Settings → Code security)  
3. Branch protection on `main`: require PR + status check **`ci-success`**  
4. (Optional) Add topics: `golang`, `llm`, `mcp`, `cli`, `coding-agent`  
5. First public tag (`v0.1.0`) — **historical**; latest published pin is **v1.3.6** ([RELEASING.md](../RELEASING.md))  
6. Do **not** publish private stage endpoints or production mesh URLs in issues/docs  

## Out of scope for open-source binary

- Multi-tenant remote agent hosting  
- OS-level sandbox (Seatbelt/bubblewrap) — recommended for untrusted workloads, not bundled  
- Guarantees about third-party LLM/MCP availability  
- Re-doing the visibility flip (deliberate maintainer act already done)  
- Inventing Memory GA / dual_write ON / catalog Connected from residual PASS  
