## Summary

<!-- What and why (1–3 bullets). -->

-

## Type of change

- [ ] Feature
- [ ] Bug fix
- [ ] Security / hardening
- [ ] Docs / CI
- [ ] Refactor (no behavior change)

## Test plan

- [ ] `make check` (or CI green: lint, test, build, govulncheck)
- [ ] New/changed behavior covered by unit tests
- [ ] No secrets in config/examples/logs

## Security checklist (if touching tools, FS, shell, HTTP, auth)

- [ ] Path jail / shell policy / secret scrub still hold
- [ ] Errors/logs use redaction where credentials may appear
- [ ] Docs updated if threat model changes (`SECURITY.md`, `docs/security.md`)
