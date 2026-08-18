# Support

## How to get help

| Need | Where |
|------|--------|
| Usage questions / bugs | [GitHub Issues](https://github.com/iome-sh/iomesh-tui/issues) (use the templates) |
| Security vulnerability | Private [Security Advisory](https://github.com/iome-sh/iomesh-tui/security/advisories/new) or **security@iome.sh** — see [SECURITY.md](SECURITY.md) |
| Architecture / threat model | [docs/](docs/) especially [docs/security.md](docs/security.md) and [docs/architecture/overview.md](docs/architecture/overview.md) |
| Contributing | [CONTRIBUTING.md](CONTRIBUTING.md) |

## What we maintain

- **Best effort** on `main` and security fixes on the latest `v1.x` tag
- Security fixes on the default branch  
- No paid support SLA for the open-source binary  

## What we do not provide here

- Hosted I/O Mesh cloud onboarding (see your I/O Mesh operator / [iome.sh](https://iome.sh) if applicable)  
- Guarantees about third-party LLM or MCP server availability  

## Before filing an issue

1. Run `make check` or note CI failures  
2. Redact API keys, tokens, and private paths from logs  
3. Include `iomesh version` / commit SHA and OS  
