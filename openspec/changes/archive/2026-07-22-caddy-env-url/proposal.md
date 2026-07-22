## Why

The current `Caddyfile` hardcodes `logs.example.com`, forcing every operator to edit a version-controlled file to set their domain — a change that is easy to lose across merges and inconsistent with how every other runtime value is configured. The domain is deployment-specific configuration, and should live in `.env` like everything else.

## What Changes

- Replace the hardcoded hostname in `Caddyfile` with a Caddy environment variable placeholder (`{env.OPENLOGS_URL}`).
- Pass `OPENLOGS_URL` from the host environment into the `caddy` service via `docker-compose.yml`.
- Add `OPENLOGS_URL` to `.env.example` with a sensible comment.
- Update the README so that "set your domain" instructions point to `.env` instead of editing the Caddyfile.

## Capabilities

### New Capabilities

<!-- none -->

### Modified Capabilities

- `deployment`: The "Caddyfile template" requirement changes — instead of a hardcoded placeholder domain that users must edit in a tracked file, the domain is read from the `OPENLOGS_URL` environment variable at Caddy startup.

## Impact

- **Files**: `Caddyfile`, `docker-compose.yml`, `.env.example`, `README.md`
- **Behaviour**: Caddy's environment variable substitution (`{env.OPENLOGS_URL}`) is a built-in Caddy feature; no new dependencies. Operators who previously edited `Caddyfile` directly must migrate their domain value into `.env`.
