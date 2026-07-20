## Context

Users are created only via the `create-user` CLI, which needs shell access. The app already has all the primitives an invite flow needs: bcrypt hashing (`golang.org/x/crypto/bcrypt`), `db.CreateUser`, session issuance, an idempotent schema applied on `db.Open`, and a public/protected route split in `router.go`. Sessions are stored as plaintext UUIDs, and there is no user-role concept — every authenticated user can already create projects and change global settings.

This change adds a thin invite layer: a new table, a handful of DB methods, one protected create handler, and a public two-route accept flow, plus minimal templates.

## Goals / Non-Goals

**Goals:**
- Let an existing user generate a one-time, expiring link that onboards a new user by email.
- The invited person sets their own password; the account is created only on acceptance.
- Keep it simple and consistent with existing patterns; no new dependencies, no config.

**Non-Goals:**
- A real admin/role system or per-invite permissions (the model stays flat).
- Sending email. The inviter copies and shares the link out-of-band.
- Editing/deleting existing users, or inviting to reset an existing user's password.

## Decisions

**Storage: a new `invites` table in the schema constant.**
Columns: `token TEXT PRIMARY KEY`, `email TEXT NOT NULL`, `expires_at TEXT NOT NULL`, `used_at TEXT` (nullable), `created_at TEXT NOT NULL DEFAULT (datetime('now'))`. Added to the existing `schema` const so it migrates on startup like everything else — no migration framework needed. Timestamps use the same `2006-01-02 15:04:05` UTC format as sessions.

**Token: crypto/rand, ~32 bytes, base64url, stored as-is.**
`crypto/rand` → 32 random bytes → `base64.RawURLEncoding` gives a URL-safe, unguessable token used directly in the path (`/invite/{token}`). Stored in plaintext, consistent with how session UUIDs are stored today. Alternative — storing only a hash of the token — is more defensive if the DB leaks, but adds lookup complexity and diverges from the existing session model; deferred and noted as a risk. UUID was rejected in favor of a longer random token since the token travels in a shareable URL.

**Expiry: a code constant (default 7 days), single successful use.**
`inviteTTL = 7 * 24 * time.Hour`, mirroring how `sessionTTL` is a constant in `auth.go`. Validity = token exists AND `used_at IS NULL` AND `expires_at > now`. Acceptance sets `used_at` in the same step as user creation so the link is single-use.

**Supersede prior pending invites for the same email.**
Creating an invite deletes any existing unused invite rows for that email first, so only one link is ever active per address ("one-time link" stays unambiguous). Simple `DELETE ... WHERE email=? AND used_at IS NULL` before insert.

**Placement: create from Settings; accept on a public page.**
The invite-create form and pending list live on the existing `/settings` page (new `POST /settings/invite`), avoiding a new nav entry and matching "keep it simple." The accept flow is public — `GET/POST /invite/{token}` registered outside the `protect(...)` wrapper in `router.go`, next to `/login` — since the invitee has no session yet. The accept page reuses the login-style layout (renders without nav because `base.html` only shows nav when `.User` is set).

**After acceptance: create the user, then redirect to `/login` with a success flash.**
Reuses `db.CreateUser` + the existing bcrypt path and avoids duplicating cookie/session issuance. Auto-login (issuing a session immediately) is a reasonable UX alternative but is deferred to keep the flow minimal and the session logic in one place.

**Validation reuse.** The set-password handler enforces the same rules as `ChangePassword` (min 8 chars, confirmation match). A duplicate-email race at acceptance time (someone created that user meanwhile) is handled by surfacing `db.ErrDuplicateEmail` as an error.

## Risks / Trade-offs

- [Plaintext tokens in the DB — a database leak exposes usable invite links] → Tokens are single-use and short-lived (7 days), and this matches the existing plaintext-session model; hashing tokens at rest can be added later without changing the URL format.
- [Invite reveals nothing until accepted, but a valid token grants account creation for its email] → Tokens are 256-bit random and unguessable; expiry plus single-use bound the window. Invalid/expired/used tokens return a generic error and never echo the email.
- [No email delivery — links shared manually could be forwarded] → Acceptable for a self-hosted, small-team tool; documented as a non-goal. Expiry limits exposure.
- [No rate limiting on invite creation] → Only authenticated users can create invites and the model is already trust-on-login; out of scope.

## Migration Plan

Additive only. The `invites` table is created on next startup via the idempotent schema. No backfill, no config changes; rollback is dropping the new routes/table (no existing data depends on it).

## Open Questions

- Should acceptance auto-login the new user instead of redirecting to `/login`? Defaulting to redirect-for-login for simplicity; easy to switch later.
- Do we want a configurable invite TTL (env var) rather than a constant? Starting with a constant; can promote to config if operators ask.
