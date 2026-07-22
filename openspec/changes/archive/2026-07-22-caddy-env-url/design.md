## Context

The `Caddyfile` currently contains a hardcoded `logs.example.com` hostname. Operators must edit this tracked file to configure their own domain, which is inconsistent with how every other OpenLogs deployment value is handled (environment variables / `.env`). Caddy supports `{env.VARIABLE}` substitution natively — no plugins or wrapper scripts required.

## Goals / Non-Goals

**Goals:**
- Replace the hardcoded hostname with `{env.OPENLOGS_URL}` so the Caddyfile never needs to be edited.
- Surface `OPENLOGS_URL` in `docker-compose.yml` (caddy service `environment` block) and `.env.example`.
- Update README setup instructions to point users at `.env` instead of the Caddyfile.

**Non-Goals:**
- Supporting multiple virtual hosts or complex Caddy routing.
- Changing how the `openlogs` binary is configured (no new app env vars).
- Altering the `docker-compose.local.yml` path (it has no Caddy service).

## Decisions

**Use `OPENLOGS_URL` as the variable name.**
Consistent with the existing `OPENLOGS_*` namespace. `APP_URL` is also plausible (user mentioned it), but `OPENLOGS_URL` matches the project's existing convention and is unambiguous in a multi-service `.env`.

**Caddy native env substitution (`{env.OPENLOGS_URL}`).**
Caddy resolves `{env.VAR}` at startup. No wrapper script, no sed, no templating layer. If the variable is unset Caddy logs a clear error and refuses to start — a safe-fail behaviour.

**Pass via `docker-compose.yml` `environment` block, not a Caddy-specific env file.**
Keeps the single `.env` file as the sole operator-facing configuration surface.

**Default value in compose: `${OPENLOGS_URL:-logs.example.com}`.**
The `:- fallback` means the stack still starts in demos/CI without requiring the variable, while clearly communicating what needs to be set for a real deployment.

## Risks / Trade-offs

- [Caddy exits if `OPENLOGS_URL` is unset and no fallback is provided in compose] → Mitigated by the `:-logs.example.com` default in compose and a clear comment in `.env.example`.
- [Operators who previously edited Caddyfile directly must migrate] → The migration is one line in `.env`; the old Caddyfile edit can be dropped. Documented in README.
