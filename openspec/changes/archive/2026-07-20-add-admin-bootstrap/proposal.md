## Why

Fresh OpenLogs deployments require a manual `openlogs create-user` step before anyone can log in, which reads a password from an interactive terminal. That is awkward for Docker/compose deployments and automated provisioning, where there may be no TTY and operators want a turnkey first boot. An environment-driven admin bootstrap removes that friction.

## What Changes

- Add two optional environment variables: `OPENLOGS_ADMIN_EMAIL` and `OPENLOGS_ADMIN_PASSWORD`.
- When **both** are set, the server creates that user on startup (during `serve`) if it does not already exist.
- The bootstrap is **idempotent**: if the user already exists it is left untouched (the password is not reset), so it is safe across container restarts.
- The bootstrap password must be at least 8 characters, matching the `create-user` rule; a shorter value fails startup with a clear error.
- If either variable is unset, behaviour is unchanged (users are managed only via `create-user`).
- Wire the variables through `.env.example`, `docker-compose.yml`, and `docker-compose.local.yml`, and document them in the README.

## Capabilities

### New Capabilities

- `admin-bootstrap`: Create an initial admin user from environment variables on server startup, idempotently, so deployments can provision the first user without an interactive `create-user` step.

### Modified Capabilities

## Impact

- **Code**: `internal/config` (two new fields), `cmd/openlogs/main.go` (`bootstrapAdmin` run during `serve`).
- **Config/deploy**: `.env.example`, `docker-compose.yml`, `docker-compose.local.yml`.
- **Docs**: `README.md` configuration table + "Automatic admin user" note.
- **No breaking changes**: with the variables unset, startup behaves exactly as before.
