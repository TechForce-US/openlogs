## Context

OpenLogs users are created only via the `openlogs create-user <email>` CLI subcommand, which prompts for a password using `golang.org/x/term.ReadPassword` — it requires an interactive TTY. On a fresh deployment, nobody can log in until an operator runs that command inside the container. For Docker/compose and automated provisioning this is an extra manual, TTY-dependent step.

We want a way to provision the first admin from configuration alone, without a TTY, that is safe to leave enabled across restarts.

## Goals / Non-Goals

**Goals:**
- Create an initial admin user from environment variables at startup, no TTY required.
- Idempotent and restart-safe: never clobber an existing user or reset its password.
- Reuse the existing password rules (bcrypt hash, min 8 chars).
- Zero behaviour change when the variables are unset.

**Non-Goals:**
- Managing multiple users, roles, or updates via env (only the single bootstrap user).
- Rotating or resetting an existing user's password via env.
- A first-run web setup wizard.
- Doing the work in the Dockerfile at build time (there is no database or runtime then).

## Decisions

### 1. Bootstrap in the `serve` startup path, not the Dockerfile or an entrypoint

The request framed this as "in the Dockerfile", but the Dockerfile runs at build time with no database. An entrypoint shell script could call `create-user`, but that command reads the password from a TTY, so it cannot be scripted cleanly. Instead, `runServe()` calls `bootstrapAdmin(db, email, password)` after opening the database and before serving. This works identically for the bare binary, `docker run`, and compose, and needs no Dockerfile changes.

**Alternatives considered**:
- *Entrypoint script calling `create-user`* — rejected; `create-user`'s TTY password prompt isn't scriptable, and piping would require refactoring it.
- *A new non-interactive `create-user --password-stdin` flag invoked by an entrypoint* — more moving parts than an in-process bootstrap and still needs the compose entrypoint plumbing.

### 2. Require both variables; otherwise no-op

The bootstrap runs only when `OPENLOGS_ADMIN_EMAIL` and `OPENLOGS_ADMIN_PASSWORD` are both non-empty. This makes the feature strictly opt-in and keeps the default experience unchanged.

### 3. Idempotent by existence check, never reset

`bootstrapAdmin` looks up the user by email; if present, it logs and returns without changes. This makes it safe to keep the variables set permanently (the common Docker case) — restarts won't reset a password the operator may have since changed in the UI.

**Alternative considered**: upserting/resetting the password on every boot — rejected; it would silently revert password changes made through the app and turn a leaked env value into a permanent backdoor on each restart.

### 4. Reuse existing password policy and hashing

The password is bcrypt-hashed via the same path as `create-user`, and the same minimum length (8) is enforced. A shorter password fails startup with a clear error rather than creating a weak account.

### 5. Configuration lives in `config.Config`

`AdminEmail` and `AdminPassword` are read in `config.Load()` alongside the other settings, keeping all environment parsing in one place. They are optional and do not participate in the required-value validation that `OPENLOGS_SECRET_KEY` does.

## Risks / Trade-offs

- **Secret in the environment** → the admin password sits in env/compose files. → Mitigation: documented as opt-in; recommend `.env` (gitignored) and changing the password in-app after first login. Idempotency ensures a later in-app password change is not reverted.
- **Startup failure on a weak password** → a `<8` char value aborts `serve`. → Intentional: fail fast and loud rather than provisioning a weak admin; the error message names the variable.
- **Email typo creates an unintended account** → low impact; the extra user can be managed/removed, and only one account is created.

## Open Questions

- None for this change. A future enhancement could add a non-interactive `create-user --password-stdin` for other automation flows, independent of this startup bootstrap.
