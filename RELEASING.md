# Releasing

Pre-1.0: ship from `main` via PR; cut annotated tags for public consumers.

## When to bump and tag

**Do not leave feature waves only under `[Unreleased]`.** After merging a coherent minor/major capability set, cut a release in the same delivery loop (or immediately after):

| Trigger | Bump | Examples |
|---------|------|----------|
| New operator-facing capability, config/env, or dogfood surface | **minor** (`0.x → 0.(x+1).0`) | Memory Phase 3 sync retrieve + sidecar, new mesh plane |
| Breaking CLI/API/config (pre-1.0 still document; prefer minor) | **minor** (0.x) or **major** (1.0+) | Rename flags, remove defaults users rely on |
| Docs-only / serial renumber / fixups | usually **no** tag | SRED renumber docs, typo |
| Security fix on latest minor | **patch** if needed (`0.x.y`) | CVE follow-up |

Dogfood waves that ship real code (agent, mesh client, CLI) count as minor features — **bump + tag** when the wave closes, not only at arbitrary milestones.

Checklist items that must move with the tag:

- [CHANGELOG.md](CHANGELOG.md) `[Unreleased]` → `## [X.Y.Z]`
- `cmd/iomesh` default `version = "X.Y.Z"`
- [README.md](README.md) status line (`vX.Y.x`)
- [SECURITY.md](SECURITY.md) supported-versions table

## Checklist before a public tag

1. [ ] `make ci` green locally  
2. [ ] GitHub Actions **ci-success** green on the release commit  
3. [ ] [CHANGELOG.md](CHANGELOG.md) updated (move Unreleased → version section)  
4. [ ] No secrets in tree (`git grep` for keys; review `configs/`, examples)  
5. [ ] [SECURITY.md](SECURITY.md) / [docs/security.md](docs/security.md) current  
6. [ ] Default `main.version` string in `cmd/iomesh` matches the tag (ldflags override via `make build`)  
7. [ ] Annotated tag `vX.Y.Z` + `gh release create` published  

## Tag and publish (maintainers)

```bash
git checkout main
git pull origin main
# edit CHANGELOG.md + main.version + README/SECURITY if needed
git commit -am "chore: release vX.Y.Z"
git push origin main
git tag -a vX.Y.Z -m "vX.Y.Z — short release title"
git push origin vX.Y.Z
gh release create vX.Y.Z --title "vX.Y.Z" --notes-file <(sed -n '/## \[X.Y.Z\]/,/## \[/p' CHANGELOG.md | head -n -1)
```

`make build` embeds `git describe` (or `VERSION=`) into the binary via `-X main.version=…`.

## Versioning policy

- **0.x** — breaking changes allowed without major bump; document in CHANGELOG  
- **1.0+** — SemVer; breaking CLI/API changes require major bump  
- Prefer **minor** bumps when shipping a completed backlog wave (memory plane, dogfood, agent path), even if flags default off  

## Artifacts

CI builds and tests; official release binaries may be added later (GitHub Actions release workflow / goreleaser). Until then:

```bash
make build   # → bin/iomesh
go install github.com/iome-sh/iomesh-tui/cmd/iomesh@vX.Y.Z
```
