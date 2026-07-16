# Releasing

Pre-1.0: ship from `main` via PR; cut annotated tags for public consumers.

## Checklist before a public tag

1. [ ] `make ci` green locally  
2. [ ] GitHub Actions **ci-success** green on the release commit  
3. [ ] [CHANGELOG.md](CHANGELOG.md) updated (move Unreleased → version section)  
4. [ ] No secrets in tree (`git grep` for keys; review `configs/`, examples)  
5. [ ] [SECURITY.md](SECURITY.md) / [docs/security.md](docs/security.md) current  
6. [ ] Default `main.version` string in `cmd/iomesh` matches the tag (ldflags override via `make build`)  

## Tag and publish (maintainers)

```bash
git checkout main
git pull origin main
# edit CHANGELOG.md + main.version default if needed
git commit -am "chore: release vX.Y.Z"
git push origin main
git tag -a vX.Y.Z -m "vX.Y.Z"
git push origin vX.Y.Z
gh release create vX.Y.Z --title "vX.Y.Z" --notes-file <(sed -n '/## \[X.Y.Z\]/,/## \[/p' CHANGELOG.md | head -n -1)
```

`make build` embeds `git describe` (or `VERSION=`) into the binary via `-X main.version=…`.

## Versioning policy

- **0.x** — breaking changes allowed without major bump; document in CHANGELOG  
- **1.0+** — SemVer; breaking CLI/API changes require major bump  

## Artifacts

CI builds and tests; official release binaries may be added later (GitHub Actions release workflow). Until then:

```bash
make build   # → bin/iomesh
```
