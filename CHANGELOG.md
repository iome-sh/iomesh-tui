# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/) once tagged releases begin.

## [Unreleased]

### Added

- Open-source launch hygiene: issue templates, Dependabot, CHANGELOG, SUPPORT, RELEASING, docs index
- Expanded security residual-risk documentation for public operators

### Foundation (pre-1.0, main)

Features landed on `main` before the first tagged release include:

- DeepSeek-first LLM cascade + pure-Go OpenAI-compatible router
- Agent loop, workspace tools, path jail, shell policy, secret scrubbing
- Subagents (parallel, worktree isolation, apply/merge)
- Session persistence, interactive permissions, full-screen Bubble Tea TUI
- ACP stdio + WebSocket, skills loader, MCP (stdio/HTTP, tools/resources/prompts, OAuth helpers)
- Stage mesh dogfood probe, CI (lint/test/race/govulncheck/build)

## [0.1.0] — TBD

First public tagged release after open-source launch checklist (see [RELEASING.md](RELEASING.md)).
