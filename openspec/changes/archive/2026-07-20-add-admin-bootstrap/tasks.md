# Note: this change was authored retroactively to document an already-implemented
# and verified feature; tasks are marked complete to reflect the current codebase.

## 1. Configuration

- [x] 1.1 Add optional `AdminEmail` and `AdminPassword` fields to `config.Config`, read from `OPENLOGS_ADMIN_EMAIL` / `OPENLOGS_ADMIN_PASSWORD` in `config.Load()` (no required-value validation)

## 2. Startup bootstrap

- [x] 2.1 Implement `bootstrapAdmin(db, email, password)` in `cmd/openlogs/main.go`: no-op when either value is empty; skip (log) if the user exists; enforce password length ≥ 8; bcrypt-hash and create the user otherwise
- [x] 2.2 Call `bootstrapAdmin` in `runServe()` after opening the database and before serving; abort startup on error

## 3. Deployment & docs

- [x] 3.1 Add the two variables to `.env.example` (commented, opt-in) with guidance
- [x] 3.2 Add the variables to `docker-compose.yml` (passthrough from env) and `docker-compose.local.yml` (commented examples)
- [x] 3.3 Document in `README.md`: configuration table rows + "Automatic admin user" note + pointers at the `create-user` quickstart steps

## 4. Tests & verification

- [x] 4.1 Unit tests in `cmd/openlogs/main_test.go`: no-op when unset/partial; creates user; idempotent (no password reset on second run); rejects password < 8 chars
- [x] 4.2 Confirm `go build ./...`, `go vet ./...`, `go test ./...` pass
- [x] 4.3 Live check: server logs `admin bootstrap: created user …` on first boot, login succeeds with the bootstrapped credentials, and a restart logs `already exists, skipping`
