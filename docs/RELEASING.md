# Releasing Airlock

Cut a tagged GitHub Release. `install.sh` downloads assets named:

`airlock_<version>_<os>_<arch>.tar.gz` (e.g. `airlock_0.1.0-beta.1_linux_amd64.tar.gz`)

First public cut: **`v0.1.0-beta.1`**. See [CHANGELOG.md](../CHANGELOG.md) for the full beta notes users will read on the Release page.

## Checklist

1. CI green on `main` (`.github/workflows/ci.yml`).
2. Update [CHANGELOG.md](../CHANGELOG.md):
   - Move items from **Unreleased** under the new version heading (`Added` / `Changed` / `Fixed`).
   - For prereleases, keep a short **Highlights** + **Known limits** block (same shape as `0.1.0-beta.1`).
   - Paste the changelog section into the GitHub Release body (or link to `CHANGELOG.md`).
3. Align version mentions if needed: README status line, GUIDE install pin, SUPPORT, SECURITY “supported versions”.
4. Tag and push (must start with `v`):

```bash
git checkout main
git pull
git tag -a v0.1.0-beta.1 -m "v0.1.0-beta.1 — first public beta"
git push origin v0.1.0-beta.1
```

5. Watch **Release** workflow (`.github/workflows/release.yml`). It builds linux/darwin × amd64/arm64, attaches tarballs + sha256, creates the GitHub Release (pre-release if the tag contains `beta` / `rc` / `alpha`).
6. Smoke:

```bash
# betas are GitHub pre-releases — pin the tag (latest skips them)
AIRLOCK_VERSION=v0.1.0-beta.1 curl -sSL https://raw.githubusercontent.com/ankittk/airlock/main/install.sh | sh
airlock version   # expect: airlock 0.1.0-beta.1 (or similar)
```

## Manual / dry-run

Actions → **Release** → **Run workflow** → optional `tag` input (e.g. `v0.1.0-beta.1`). Prefer a real tag push for production cuts.

## What not to do

- Do not attach hand-built binaries that skip `-ldflags "-X main.version=…"`.
- Do not use tags without a `v` prefix — `install.sh` and the workflow both expect `v*`.
- Do not ship with an empty or “Unreleased-only” changelog — betas need readable Release notes.
