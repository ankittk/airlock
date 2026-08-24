# Changelog

All notable changes to Airlock are documented here.

Format inspired by [Keep a Changelog](https://keepachangelog.com/).
Versions follow [SemVer](https://semver.org/) with prerelease tags (`beta`, `rc`).

## [Unreleased]

### Added
- Agent-driven supply chain: APM package dependencies tracked as `manifest.Dependency`; a new dependency landing alongside an AI-artifact change (prompt/skill/MCP/agent) raises `NEEDS_APPROVAL` in `airlock diff` / `airlock ci --fail-on-approval`. Dependency-only PRs are left to SCA.

### Changed
### Fixed
- `airlock ci --comment` no longer bypasses `--fail-on-approval` / `--fail-on-eval` / `fail_on_ai_change` — it now only changes the stdout format and still writes `ci-comment.md`; the fail-closed gate always runs.

## [0.1.0-beta.2] – 2026-08-21

Re-cut of the first public beta with working GitHub Release assets (beta.1 publish raced and left an empty release).

### Changed
- README quick start: console walkthrough (prompt change → diff → CI comment; MCP → `NEEDS_APPROVAL`)

### Fixed
- Publish a complete multi-arch release for `install.sh` (linux/darwin × amd64/arm64)

Install from this tag (not beta.1). One pin lives in [README — Install](README.md#install).

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

[Unreleased]: https://github.com/ankittk/airlock/compare/v0.1.0-beta.2...HEAD
[0.1.0-beta.2]: https://github.com/ankittk/airlock/releases/tag/v0.1.0-beta.2
[0.1.0-beta.1]: https://github.com/ankittk/airlock/releases/tag/v0.1.0-beta.1
