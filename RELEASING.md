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
7. [ ] Annotated tag `vX.Y.Z` pushed (GoReleaser **release** workflow green; assets on GitHub Release; optional `gh release edit` for notes)

## Tag and publish (maintainers)

```bash
git checkout main
git pull origin main
# edit CHANGELOG.md + main.version + README/SECURITY if needed
git commit -am "chore: release vX.Y.Z"
git push origin main

# Annotated tag — triggers .github/workflows/release.yml (GoReleaser)
git tag -a vX.Y.Z -m "vX.Y.Z — short release title"
git push origin vX.Y.Z

# Optional: seed release notes from CHANGELOG if GoReleaser footer is not enough.
# gh release edit vX.Y.Z --notes-file <(…)
```

`make build` embeds `git describe` (or `VERSION=`) into the binary via `-X main.version=…`.  
GoReleaser ldflags set `main.version` to the tag version on published assets.

### GoReleaser (binaries)

| Piece | Path |
|-------|------|
| Config | [`.goreleaser.yaml`](.goreleaser.yaml) |
| Workflow | [`.github/workflows/release.yml`](.github/workflows/release.yml) (on `v*` tags) |
| Local dry-run | `make release-snapshot` (needs `goreleaser` + `syft` on PATH for SBOM) |

Cross-builds: linux/darwin/windows × amd64/arm64 (windows/arm64 ignored). Archives include LICENSE + README + CHANGELOG; `checksums.txt` + per-archive **SPDX SBOM** (`*.sbom.spdx.json`) attach to the GitHub Release.

**Signing (keyless cosign):** tag releases sign `checksums.txt` with **cosign sign-blob** via GitHub OIDC / Fulcio (`id-token: write`). No long-lived `COSIGN_*` secrets. Snapshot/workflow_dispatch runs use `--skip=sign`.

Verify:

```bash
cosign verify-blob \
  --certificate checksums.txt.pem \
  --signature checksums.txt.sig \
  --certificate-identity-regexp 'https://github.com/iome-sh/iomesh-tui/.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  checksums.txt
```

## Versioning policy

- **0.x** — breaking changes allowed without major bump; document in CHANGELOG  
- **1.0+** — SemVer; breaking CLI/API changes require major bump  
- Prefer **minor** bumps when shipping a completed backlog wave (memory plane, dogfood, agent path, release packaging), even if flags default off  

## Artifacts

| Path | Notes |
|------|--------|
| GitHub Release assets | GoReleaser on each `v*` tag (primary) |
| `go install …@vX.Y.Z` | Always works with a Go toolchain |
| CI `build` job | linux/amd64 artifact on main pushes (smoke) |

```bash
make build                 # → bin/iomesh
make release-snapshot      # → dist/ (local multi-arch, no publish)
go install github.com/iome-sh/iomesh-tui/cmd/iomesh@vX.Y.Z
```
