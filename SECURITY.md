# Security Policy

## Supported versions

**First public beta:** `v0.1.0-beta.1` (and later `0.1.x` prereleases as tagged).

Security fixes land on `main` and ship in the next tagged release. Only tagged builds are supported for production-adjacent use; `go install @latest` / untagged `main` may differ.

## Reporting a vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Preferred:

1. Use GitHub **Security Advisories** (Private vulnerability reporting) on this repository once enabled.
2. Or email the maintainer listed on the GitHub profile that owns the repo, subject `Airlock security`.

Include:

- Affected command / package path
- Reproduction steps (PoC preferred)
- Impact (data leak, trust bypass, CI gate bypass, etc.)

You should get an acknowledgement within a few days. Please give a reasonable window before public disclosure.

## Security model (OSS CLI)

- **Local-first:** the OSS binary does not upload traces or eval data.
- **Redaction:** `ingest` / `baseline` run local regex redaction before writing under `.airlock/`.
- **Trust boundary:** treat `.airlock/` contents and eval fixtures as sensitive if they came from production.
- **Approvals:** `NEEDS_APPROVAL` + the local approval ledger are advisory unless CI uses `--fail-on-approval` (sample app workflow defaults that flag on). Skill and MCP expansions both raise approval.

Known non-goals for the OSS CLI: multi-tenant auth, remote policy sync, guaranteed GDPR tooling.
