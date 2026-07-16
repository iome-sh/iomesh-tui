# Contributing

Thanks for helping improve **iomesh-tui**. This project will be open-sourced; please treat quality, security, and tests as first-class.

## Development setup

```bash
# Go 1.22+ (CI uses the version in go.mod)
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
```

## Coding standards

- **Pure Go** in hot paths (`internal/router`, tools, workspace) — avoid heavy SDKs unless justified
- **Fail closed** on path/URL/shell policy; **fail open** only for optional I/O Mesh context
- **Never log secrets** — use `security.Redact` for error strings that may contain credentials
- Prefer small, focused PRs with tests for new behavior
- Run `gofmt` (or `make fmt`) before commit

## Tests

| Package | Focus |
|---------|--------|
| `internal/security` | redaction, env scrub, shell policy, URL validation |
| `internal/workspace` | path jail, symlink escape, size limits |
| `internal/router` | cascade, fallback, streaming, URL scheme rejection |
| `internal/agent` | tool filter, yolo deny, subagent registration |
| `internal/subagent` | spawn, background, depth, resume |
| `internal/iomesh` | fail-open, endpoint validation |
| `internal/config` | TOML merge, env overrides |

New features should include unit tests. Network calls in tests must use `httptest` (no live provider keys in CI).

## Security-sensitive changes

If you touch filesystem, shell, HTTP clients, auth, or config secret handling:

1. Add/adjust tests under `internal/security` or the affected package
2. Update [SECURITY.md](SECURITY.md) / [docs/security.md](docs/security.md) if the threat model changes
3. Prefer explicit deny lists + allowlists with clear comments

Report vulnerabilities privately — see [SECURITY.md](SECURITY.md).

## Pull requests

- Clear description of *what* and *why*
- Link related issues
- Ensure CI is green
- Do not commit API keys, `.env`, or real workspace secrets

### CI on PR and merge

GitHub Actions workflow [`.github/workflows/ci.yml`](.github/workflows/ci.yml) runs on:

| Event | When |
|-------|------|
| `pull_request` | opened / synchronize / reopened / ready_for_review → `main` |
| `push` | commits to `main` (after merge) |
| `merge_group` | GitHub merge queue (if enabled) |
| `workflow_dispatch` | manual re-run |

Jobs: **lint** · **test** (race + coverage artifact) · **build** (`iomesh version` / `models`) · **govulncheck** · **ci-success** (aggregate gate).

Recommended branch protection on `main` (Settings → Branches):

1. Require a pull request before merging  
2. Require status checks to pass: **`ci-success`** (or each of lint/test/build/govulncheck)  
3. Require branches to be up to date before merging  
4. Do not allow bypassing the above for admins (optional but preferred for OSS)

Local parity:

```bash
make ci   # fmt-check + vet + test + race + cover + vuln + build
```

## License

By contributing, you agree that your contributions are licensed under the MIT License (see [LICENSE](LICENSE)).
