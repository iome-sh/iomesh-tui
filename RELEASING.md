# Releasing

Pre-1.0: ship from `main` via PR; tags are optional until the first public release.

## Checklist before a public tag

1. [ ] `make ci` green locally  
2. [ ] GitHub Actions **ci-success** green on the release commit  
3. [ ] [CHANGELOG.md](CHANGELOG.md) updated (move Unreleased → version section)  
4. [ ] No secrets in tree (`git grep` for keys; review `configs/`, examples)  
5. [ ] [SECURITY.md](SECURITY.md) / [docs/security.md](docs/security.md) current  
6. [ ] Version string in `cmd/iomesh` / `-ldflags` as used by `make build`  

## Tag and publish (maintainers)

```bash
git checkout main
git pull origin main
# edit CHANGELOG.md
git commit -am "chore: release v0.1.0"
git tag -a v0.1.0 -m "v0.1.0"
git push origin main --tags
```

Create a GitHub Release from the tag; attach notes from CHANGELOG.

## Versioning policy

- **0.x** — breaking changes allowed without major bump; document in CHANGELOG  
- **1.0+** — SemVer; breaking CLI/API changes require major bump  

## Artifacts

CI builds and tests; official release binaries may be added later (GitHub Actions release workflow). Until then:

```bash
make build   # → bin/iomesh
```
