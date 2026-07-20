## 1. Schema & database layer

- [x] 1.1 Add an `invites` table (`token` PK, `email`, `expires_at`, `used_at` nullable, `created_at` default now) to the `schema` constant in `internal/db/schema.go`.
- [x] 1.2 Create `internal/db/invites.go` with an `Invite` struct and methods: `CreateInvite(email, token string, ttl time.Duration)` (deletes prior unused invites for the email, then inserts), `GetValidInvite(token string)` (returns the invite only if unused and unexpired, else `(nil, nil)`), `MarkInviteUsed(token string)`, `DeleteInvite(token string)`, and `ListPendingInvites()` for the settings view.

## 2. Token generation & invite creation handler

- [x] 2.1 Add a `newInviteToken()` helper (in handler package) using `crypto/rand` + `base64.RawURLEncoding` to produce an unguessable token.
- [x] 2.2 Add `POST /settings/invite` handler: normalize/validate email, reject if a user already exists for it (`GetUserByEmail`), generate a token, call `CreateInvite`, and re-render settings with the resulting one-time link (absolute path `/invite/{token}`) shown for copying.
- [x] 2.3 (Optional) Add `POST /settings/invite/revoke` handler that calls `DeleteInvite(token)` and re-renders settings.

## 3. Public accept-invite flow

- [x] 3.1 Add `GET /invite/{token}` handler: look up via `GetValidInvite`; on hit render a set-password page, on miss render a generic "invalid or expired invite" error page (never echo the email on miss).
- [x] 3.2 Add `POST /invite/{token}` handler: re-validate the token, enforce password rules (min 8 chars, confirm match), `CreateUser` with a bcrypt hash, `MarkInviteUsed`, and redirect to `/login` with a success flash. Handle `ErrDuplicateEmail` as an error.

## 4. Routing & templates

- [x] 4.1 Register routes in `internal/handler/router.go`: public `GET /invite/{token}` and `POST /invite/{token}` (outside `protect`), and protected `POST /settings/invite` (+ optional revoke route).
- [x] 4.2 Create `web/templates/invite.html`: a login-style public page with a set-password form (`new_password`, `confirm_password`) posting to `/invite/{token}`, plus the error state.
- [x] 4.3 Update `web/templates/settings.html`: add an "Invite a user" section (email input → `POST /settings/invite`), display the generated link after creation, and list pending invites (email, expiry, copyable link, optional revoke).
- [x] 4.4 Extend the settings view data (`settingsView`) to carry the newly generated link and the pending-invite list.

## 5. Tests & verification

- [x] 5.1 `internal/db` tests: create supersedes prior pending invite; `GetValidInvite` rejects expired and used tokens; `MarkInviteUsed` makes a token invalid.
- [x] 5.2 Handler tests: create-invite rejects an existing-user email; accept flow with a valid token creates the user and consumes the invite; a used/expired/invalid token shows the error page and creates no user; weak/mismatched passwords rejected.
- [x] 5.3 Run `go test ./...` and `go vet ./...`; manually verify the end-to-end flow (invite from settings → open link in a fresh session → set password → log in) in the running app.
