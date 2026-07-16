# Contributing

Thanks for helping improve **iomesh-tui**. Please treat quality, security, and tests as first-class.

## Development setup

```bash
# Go version: see go.mod (CI uses that exact toolchain via GOTOOLCHAIN=auto)
git clone https://github.com/iome-sh/iomesh-tui.git
cd iomesh-tui
make test
make vet
make build
```

Optional:

```bash
make test-race
make cover
make vuln
make ci    # full local gate (fmt + vet + test + race + cover + vuln + build)
```

## Coding standards

- **Pure Go** in hot paths (`internal/router`, tools, workspace) — avoid heavy SDKs unless justified  
- **Fail closed** on path/URL/shell policy; **fail open** only for optional I/O Mesh context / MCP optional surfaces  
- **Never log secrets** — use `security.Redact` for error strings that may contain credentials  
- Prefer small, focused PRs with tests for new behavior  
- Run `gofmt` (or `make fmt`) before commit  

## Tests

| Package | Focus |
|---------|--------|
| `internal/security` | redaction, env scrub, shell policy, URL validation |
| `internal/workspace` | path jail, symlink escape, size limits |
| `internal/router` | cascade, fallback, streaming, URL scheme rejection |
| `internal/agent` | tools, approval, subagent/MCP registration |
| `internal/subagent` | spawn, parallel, worktree |
| `internal/mcp` | stdio/HTTP, resources/prompts, OAuth |
| `internal/iomesh` | fail-open, dogfood |
| `internal/tui` | REPL, fullscreen, themes |
| `internal/config` | TOML merge, env overrides |

New features should include unit tests. Network calls in tests must use `httptest` (no live provider keys in CI).

## Security-sensitive changes

If you touch filesystem, shell, HTTP clients, auth, or config secret handling:

1. Add/adjust tests under `internal/security` or the affected package  
2. Update [SECURITY.md](SECURITY.md) / [docs/security.md](docs/security.md) if the threat model changes  
3. Prefer explicit deny lists + allowlists with clear comments  

Report vulnerabilities privately — see [SECURITY.md](SECURITY.md). **Do not open public issues for exploits.**

## Issues & discussions

- Bugs / features: use [issue templates](https://github.com/iome-sh/iomesh-tui/issues/new/choose)  
- Support channels: [SUPPORT.md](SUPPORT.md)  

## Pull requests

- Clear description of *what* and *why*  
- Link related issues  
- Ensure CI is green  
- Do not commit API keys, `.env`, or real workspace secrets  
- Update [CHANGELOG.md](CHANGELOG.md) **Unreleased** for user-visible changes  

### CI on PR and merge

GitHub Actions workflow [`.github/workflows/ci.yml`](.github/workflows/ci.yml) runs on:

| Event | When |
|-------|------|
| `pull_request` | opened / synchronize / reopened / ready_for_review → `main` |
| `push` | commits to `main` (after merge) |
| `merge_group` | GitHub merge queue (if enabled) |
| `workflow_dispatch` | manual re-run |

Jobs: **lint** · **test** (race + coverage artifact) · **build** · **govulncheck** · **ci-success** (aggregate gate).

Recommended branch protection on `main`:

1. Require a pull request before merging  
2. Require status checks to pass: **`ci-success`**  
3. Require branches to be up to date before merging  

Local parity:

```bash
make ci
```

## License

By contributing, you agree that your contributions are licensed under the MIT License (see [LICENSE](LICENSE)).
