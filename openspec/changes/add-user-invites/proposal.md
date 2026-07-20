## Why

Today the only way to create a web UI user is the `create-user` CLI subcommand, which requires shell access to the server. There is no way to onboard a teammate from the UI. We want an existing user to invite someone by email and hand them a one-time link where they set their own password — no shared secrets, no CLI access.

## What Changes

- Add an **invite** concept: an existing authenticated user enters an email address and generates a **one-time, expiring link**.
- The invited person opens the link (a public route), sets a password, and their user account is created — after which the link is consumed and cannot be reused.
- Add a small **"Invite a user"** section to the account settings page: enter an email, get back a copyable one-time link, and see/revoke any still-pending invites.
- Add a public **accept-invite** page that validates the token and shows a set-password form.
- Invites are stored in a new `invites` table (added to the idempotent startup schema — no separate migration step).

Notes on scope / permissions:
- The app has **no role system** — every authenticated user already has full instance powers (create projects, change the global theme). Invites follow the same model: any logged-in user can create them. A dedicated admin role is out of scope for this change.
- One-time link means single successful use: an invite is consumed when the password is set, and also becomes invalid once it expires.

## Capabilities

### New Capabilities

<!-- none — this extends existing authentication -->

### Modified Capabilities

- `user-auth`: Adds invite-based account creation (generate a one-time link, accept it to set a password and create the user) as a second onboarding mechanism, and updates the existing "Initial user creation via CLI" requirement which currently claims the CLI is the only supported mechanism.

## Impact

- **Schema**: new `invites` table (`token`, `email`, `expires_at`, `used_at`, `created_at`) added to the schema constant; applied on startup like the rest.
- **Code**:
  - `internal/db`: new `invites.go` (create / lookup-by-token / mark-used / delete / list-pending), reusing the existing user-creation path.
  - `internal/handler`: invite-create handler (from settings), public accept-invite GET/POST handlers; a crypto-random token generator.
  - `internal/handler/router.go`: public `GET/POST /invite/{token}` routes and protected `POST /settings/invite` (+ optional revoke route).
  - `web/templates`: new `invite.html` (public set-password page) and an "Invite a user" section + pending list on `settings.html`.
- **Behaviour**: additive. The CLI `create-user` path is unchanged. No API or config changes required (invite lifetime is a code constant).
