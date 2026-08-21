# Changelog

All notable changes to Airlock are documented here.

Format inspired by [Keep a Changelog](https://keepachangelog.com/).
Versions follow [SemVer](https://semver.org/) with prerelease tags (`beta`, `rc`).

## [Unreleased]

### Added
### Changed
### Fixed

## [0.1.0-beta.1] – 2026-08-21

**First public beta** of the local-first Airlock CLI: AI release control for git repos (manifest → snapshot → diff → statistical eval → CI gate).

### Highlights

- Treat models, prompts, tools, **skills**, MCP servers, judges, and evals as one releasable unit
- Content-addressed snapshots, blast-radius diff, policy verdicts (`PASS` / `FAIL` / `INCONCLUSIVE` / `NEEDS_APPROVAL`)
- PR-oriented `airlock ci` with optional fail-closed flags and sample GitHub Actions workflow
- Local-first store under `.airlock/` — no telemetry

### Added

#### Discovery & manifest

- Import [Microsoft APM](https://github.com/microsoft/apm) lockfiles (`apm.lock.yaml` / `apm.yml`)
- Scan prompts, MCP configs, env trees, model-string heuristics
- First-class **`skill`** artifacts (APM skills are skills, not tools)
- Discover Agent Skills: `SKILL.md` under `.claude/skills`, `.agents/skills`, `.gemini/skills`
- Discover Cursor rules (`.cursor/rules/*.mdc`, `*.md`) as prompts (`source: cursor-rules`)

#### Release loop

- `init` / `snapshot` / `diff` / `history` (optional local `--serve` UI)
- Statistical eval runner with Wilson/bootstrap CIs, budgets, cassettes (replay) and live modes
- `import promptfoo`, OTel ingest + redaction, baselines, `drift`
- Judge registry: calibrate / attribution
- `approve` / `rollback` + gateway routing hints

#### CI & security gates

- Adversarial / injection suites; MCP **or skill** changes auto-prefer adversarial cases
- Permission expansion → `NEEDS_APPROVAL`: MCP permission growth, write/unknown tools, **skill add/change**
- `data_boundary.fail_on_pii` on model I/O
- Sample app workflow [`.github/workflows/airlock.yml`](.github/workflows/airlock.yml): defaults `AIRLOCK_FAIL_ON_APPROVAL=true`
- Multi-arch GitHub Release assets for `install.sh` (linux/darwin × amd64/arm64)

#### Docs & fixtures

- [README](README.md), [GUIDE](docs/GUIDE.md) (incl. MCP approval demo + Security in CI), [RELEASING](docs/RELEASING.md)
- Toy agent under `testdata/toy-agent` (prompts, skill, Cursor rule, evals, APM lock)

### Known limits (honest)

- No hosted control plane / SSO / K8s admission (Phases 5–6)
- No `airlock sentinel` or deep SDK/LangGraph AST scan yet (Phase 4)
- Approvals are advisory unless CI passes `--fail-on-approval`
- Windows install not supported yet

### Install

```bash
AIRLOCK_VERSION=v0.1.0-beta.1 curl -sSL https://raw.githubusercontent.com/ankittk/airlock/main/install.sh | sh
# or: go install github.com/ankittk/airlock/cmd/airlock@v0.1.0-beta.1
```

Pin the tag: GitHub “latest” skips pre-releases.

[Unreleased]: https://github.com/ankittk/airlock/compare/v0.1.0-beta.1...HEAD
[0.1.0-beta.1]: https://github.com/ankittk/airlock/releases/tag/v0.1.0-beta.1
