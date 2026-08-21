# Developer guide

Airlock is **CI/CD for AI behavior**: a local-first Go CLI that treats models, prompts, tools, skills, MCP servers, judges, and evals as a releasable unit — then diffs, evaluates, and gates the ship.

**First public beta** — see [CHANGELOG](../CHANGELOG.md) for the current tag. Expect discovery gaps and CLI churn before 1.0.

Normal CI: “Did code build / tests pass?”  
Airlock: “Did this **AI change** stay within policy, with statistical confidence?”

State lives under `.airlock/` in your **application** repo. No cloud upload by default.

---

## Who should use this

**Good fit**

- LLM apps / agents with prompts, tools, or MCP
- Teams that change prompts or models often and want PR gates
- Repos with (or planning) eval cases / Promptfoo
- Stacks that can export OTel GenAI-ish spans for baselines later

**Weak fit**

- Pure CRUD with no model/prompt/tool surface
- Expecting a hosted org dashboard today (control plane is later)

**Examples:** support bot, RAG assistant, tool-using agent, MCP workflow, multi-agent repo with APM lockfiles.

---

## How it fits your stack

Airlock sits **beside** frameworks, gateways, and observability — it does not replace them.

| You already use… | Airlock’s role |
|------------------|----------------|
| LangGraph / LangChain / Vercel AI SDK / custom agent | Release gate in that **git repo** |
| **LangSmith** / Braintrust (traces, datasets, online evals, playground) | Keep them for observe/iterate; Airlock gates **ship/block/approve** on the PR — [ROADMAP — LangSmith](ROADMAP.md#langsmith--braintrust--langfuse--phoenix) |
| Langfuse / Datadog / Phoenix (OTel) | Consume traces via `ingest otel` → baseline / drift |
| LiteLLM / Bifrost / Portkey | Decider: emits routing hints; gateway is the actuator |
| Microsoft APM | Import `apm.lock.yaml` — do not re-implement package resolution |
| Promptfoo | `import promptfoo` → Airlock eval JSONL |

There is no first-party `integrate litellm|langgraph|langsmith` plugin yet. Integration is protocol-level (files, OTel JSONL, routing JSON). Native connectors = Phase 4–5.

---

## Mental model

```text
edit prompt / skill / model / MCP / tools
        ↓
airlock snapshot          # freeze “what the AI system is”
        ↓
airlock diff              # what changed + blast radius
        ↓
airlock test / airlock ci # evals + policy verdict
        ↓
ship / block / human approve
```

---

## Quick path (toy agent)

From this repository (after install or `go build -o airlock ./cmd/airlock`):

```bash
cd testdata/toy-agent
airlock init && airlock snapshot
# edit a file under prompts/
airlock snapshot && airlock diff
airlock test --mode replay
airlock ci --comment
```

---

## Install

Pin a pre-release tag from [Releases](https://github.com/ankittk/airlock/releases) (or copy the command from [README — Install](../README.md#install)). GitHub “latest” skips pre-releases.

```bash
curl -sSL https://raw.githubusercontent.com/ankittk/airlock/main/install.sh | AIRLOCK_VERSION=<tag> bash
# or: go install github.com/ankittk/airlock/cmd/airlock@<tag>   # Go 1.25+
```

Maintainers cutting releases: see [RELEASING.md](RELEASING.md).

---

## Day-to-day commands

Run these in your **agent / app** repo (`--path DIR` optional).

### Inventory

```bash
airlock init          # discover artifacts → .airlock/manifest + policy stub
airlock snapshot      # content-addressed release snapshot
airlock diff          # vs previous snapshot: changes + affected agents
airlock history       # local release history (optional --serve :8787)
```

### Eval & CI

```bash
airlock test --mode replay          # cassette HTTP when possible (cheap)
airlock test --mode live            # real provider calls
airlock test --affected             # only cases for agents in blast radius
airlock test --adversarial          # injection / jailbreak-style suite

airlock import promptfoo promptfoo.yaml

airlock ci --comment                # markdown for PR bodies
airlock ci --fail-on-eval
airlock ci --fail-on-approval       # block until approve on permission / skill expansion
```

PR automation: copy [`.github/workflows/airlock.yml`](../.github/workflows/airlock.yml) into the **application** repo (not the Airlock source repo). Sample defaults **fail-on-approval** (`AIRLOCK_FAIL_ON_APPROVAL`, default `true`).

### Security in CI

Airlock is an **AI change-control** gate, not a replacement for AppSec scanners.

- **Does:** MCP / write-tool / skill `NEEDS_APPROVAL`, adversarial suites on MCP/skill diffs, eval gates, optional PII fail on model I/O.
- **Does not:** CodeQL, dependency CVEs, whole-repo secret scanning — keep those jobs.

Company default: `--fail-on-approval` (and usually `--fail-on-eval`). Approvals are advisory until that flag is set.

### Approvals & rollback

```bash
airlock approve --base <snap> --head <snap>
airlock rollback --to <good-snapshot-id>   # re-pin + routing_decision.json for gateways
```

### Production loop

```bash
airlock ingest otel --file spans.jsonl --redact pii
airlock baseline create --from ingest
airlock drift
```

### Judges

```bash
airlock judge calibrate
airlock judge attribution
```

---

## Policy knobs

Edit `.airlock/policy.yml` after `init`. Useful fields:

- Gates with confidence intervals (`tool_success`, `json_valid`, `task_success`, `adversarial_critical`)
- Budgets (`max_cost_per_pr`, `max_samples_per_case`)
- `fail_on_ai_change`
- `data_boundary.fail_on_pii` — fail if PII/secret patterns appear in model I/O

Gates fire only when a CI **excludes** the threshold (no silent point-estimate fails).

MCP or skill artifact changes in `ci` auto-prefer adversarial / injection cases when available.

---

## MCP approval demo

Show the company wedge: MCP permission expansion → human gate.

From `testdata/toy-agent` (after `airlock init && airlock snapshot`):

1. Widen MCP permissions in `apm.lock.yaml` (e.g. add `write` under `local-fs.permissions`) **or** edit the discovered MCP config so the schema/permissions hash changes toward more power.
2. `airlock snapshot && airlock diff` — expect `NEEDS_APPROVAL` / MCP permission reasons.
3. `airlock ci --comment --fail-on-approval` — non-zero exit until approved.
4. `airlock approve --base <base-snap> --head <head-snap>` then re-run `ci` (or merge after the ledger records approval).

Optional: `airlock test --adversarial` / `ci` with MCP or skill touched auto-prefers injection cases when the suite exists.

Skill adds/edits also raise `NEEDS_APPROVAL` (same fail-closed path).

---

## Discovery coverage (honest)

`airlock init` is **not** “every industry SDK.” What works today vs later:

| Source | Today |
|--------|--------|
| APM (`apm.lock.yaml` / `apm.yml`) | Yes — skills are first-class `skill` (not folded into `tool`) |
| Agent Skills (`SKILL.md` under `.claude/skills`, `.agents/skills`, `.gemini/skills`) | Yes |
| Cursor rules (`.cursor/rules/*.mdc`, `*.md`) | Yes — hashed as `prompt` with source `cursor-rules` |
| MCP configs (`mcp.json`, Cursor/VS Code/Claude Desktop paths) | Yes |
| Prompt files under `prompts/`, `*.prompt.md`, etc. | Yes |
| Model strings in common config / `.env.example` | Heuristic only |
| Promptfoo / eval path globs | Thin — deepen in Phase 4 |
| `env.json` | Yes |
| OpenAI / Anthropic / Google SDK AST scan | Phase 4 |
| Vercel AI SDK, LangGraph, CrewAI, … | Phase 4+ |
| Langfuse / remote prompt registries | Phase 4 |
| Retrieval index / embedding version | Not yet |
| Live MCP schema fetch | Config hash only — Phase 4 |

Anything discoverable but not hashable should show up as an **unpinned risk** in the snapshot when we can detect it — be honest about blind spots.

**Package managers (npm/pip):** Airlock does not re-scan every LLM library lockfile. Agent dependency locking is APM’s job; Airlock imports it.

---

## Solo developer vs company

| | Solo / small team | Company |
|--|-------------------|---------|
| Where | CLI in agent repo | Same + CI workflow in **app** repos |
| Loop | init → snapshot → diff → test | + `ci` on PRs, approvals, policy |
| Prod | optional ingest / drift | baselines from redacted OTel |
| Fail closed | optional | `--fail-on-eval` / `--fail-on-approval` |

---

## What not to expect yet

Deferred work (Sentinel, eval flexibility, CUSTODY/LangSmith/SCA integration maps, publish gate, non-goals) lives in **[ROADMAP.md](ROADMAP.md)**. Do not expect Airlock to replace Promptfoo, LangSmith, Dependabot/Socket, APM, or a managed agent runtime.

**Shipped in the OSS beta:** harness skills/rules discovery, first-class `skill`, skill/MCP approval gates, Security-in-CI docs — see [discovery](#discovery-coverage-honest) and [MCP approval demo](#mcp-approval-demo).

Release notes: [CHANGELOG.md](../CHANGELOG.md).

---

## Next reading

- [README](../README.md) — product overview  
- [ROADMAP.md](ROADMAP.md) — phases, integrations, non-goals  
- [RELEASING.md](RELEASING.md) — cutting GitHub releases  
- [CONTRIBUTING.md](../CONTRIBUTING.md) — developing Airlock itself  
