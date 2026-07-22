# user-auth Specification

## Purpose
TBD - created by archiving change build-openlogs. Update Purpose after archive.
## Requirements
### Requirement: Password-based login
The system SHALL provide a login page at `GET /login` and accept credentials via `POST /login`. Passwords SHALL be hashed with bcrypt before storage. On successful login, a session is created and a session cookie is set.

#### Scenario: Successful login
- **WHEN** a user submits a valid email and password
- **THEN** a session is created, a session cookie is set, and the user is redirected to `/`

#### Scenario: Invalid credentials rejected
- **WHEN** a user submits an email or password that does not match any user record
- **THEN** a generic error message is displayed ("Invalid email or password") without indicating which field was wrong

#### Scenario: Empty credentials rejected
- **WHEN** a user submits a login form with empty email or password
- **THEN** an error is displayed and no session is created

### Requirement: Session management
Sessions SHALL be stored in the SQLite `sessions` table with a UUID primary key and an expiry timestamp. Sessions SHALL expire after 24 hours. The session ID SHALL be stored in an `httpOnly`, `Secure`, `SameSite=Strict` cookie. Expired sessions SHALL be rejected and the user redirected to `/login`.

#### Scenario: Session cookie set on login
- **WHEN** a user successfully logs in
- **THEN** an `httpOnly`, `Secure`, `SameSite=Strict` cookie containing the session UUID is set

#### Scenario: Expired session rejected
- **WHEN** a user presents a session cookie whose `expires_at` is in the past
- **THEN** the session is treated as invalid and the user is redirected to `/login`

#### Scenario: Logout clears session
- **WHEN** a user POSTs to `/logout`
- **THEN** the session record is deleted from the database and the cookie is cleared

### Requirement: Authenticated route protection
All web routes except `GET /login` and `POST /login` SHALL require a valid session. Unauthenticated requests to protected routes SHALL be redirected to `/login`.

#### Scenario: Unauthenticated access redirected
- **WHEN** an unauthenticated request is made to any protected route
- **THEN** the server responds with a redirect to `/login`

### Requirement: Initial user creation via CLI
The system SHALL provide a `create-user` CLI subcommand (`openlogs create-user <email>`) that prompts for a password interactively, hashes it with bcrypt, and inserts a user record into the database. This is one of the supported mechanisms for creating users; users may also be onboarded via one-time invite links generated from the web UI.

#### Scenario: User created via CLI
- **WHEN** `openlogs create-user admin@example.com` is run and a valid password is entered
- **THEN** a new user record with bcrypt-hashed password is written to the database

#### Scenario: Duplicate email rejected
- **WHEN** `openlogs create-user` is run with an email that already exists in the database
- **THEN** an error is printed and no record is created

#### Scenario: Weak password warning
- **WHEN** a password shorter than 8 characters is entered during user creation
- **THEN** the CLI rejects it with an error message

### Requirement: Generate a user invite
Any authenticated user SHALL be able to generate an invite for a given email address via `POST /settings/invite`. Generating an invite SHALL create an `invites` record containing a cryptographically random, unguessable token, the target email (normalized to lowercase), an expiry timestamp, and a null `used_at`. The system SHALL return a one-time acceptance link containing the token so the inviter can share it. If a user account already exists for that email, no invite SHALL be created and an error SHALL be shown. Generating a new invite for an email that already has a pending (unused, unexpired) invite SHALL supersede the prior one so only one link is active per email.

#### Scenario: Invite created for a new email
- **WHEN** an authenticated user submits a valid, unused email address
- **THEN** an invite record with a random token and future expiry is created and a one-time link containing the token is displayed

#### Scenario: Invite rejected for an existing user
- **WHEN** an authenticated user submits an email that already belongs to a user account
- **THEN** no invite is created and an error message is shown

#### Scenario: Regenerating supersedes a pending invite
- **WHEN** an authenticated user generates an invite for an email that already has a pending invite
- **THEN** the previous pending invite is invalidated and only the newly generated link is valid

### Requirement: Accept an invite to set a password
The system SHALL provide a public page at `GET /invite/{token}` that, for a valid (existing, unused, unexpired) token, displays a set-password form, and SHALL accept the submission at `POST /invite/{token}`. On a valid submission the system SHALL create a user for the invite's email with the chosen password hashed via bcrypt, mark the invite as used, and direct the person to log in. The set-password form SHALL enforce the same password rules as the rest of the app (minimum 8 characters, confirmation must match). An invalid, expired, or already-used token SHALL NOT reveal the target email and SHALL show an error rather than a password form.

#### Scenario: Valid token shows the set-password form
- **WHEN** a person opens `GET /invite/{token}` with a valid, unused, unexpired token
- **THEN** a set-password form is displayed

#### Scenario: Password set creates the user and consumes the invite
- **WHEN** a person submits a valid matching password to `POST /invite/{token}` for a valid token
- **THEN** a user is created for the invite's email with a bcrypt-hashed password, the invite is marked used, and the person is directed to log in

#### Scenario: Weak or mismatched password rejected
- **WHEN** a person submits a password shorter than 8 characters or two passwords that do not match
- **THEN** an error is shown and no user is created

#### Scenario: Invalid, expired, or used token rejected
- **WHEN** a person opens or submits `/invite/{token}` with a token that does not exist, has expired, or has already been used
- **THEN** an error is shown, no set-password form is offered, and the target email is not revealed

#### Scenario: Token is single-use
- **WHEN** an invite has already been used to create a user
- **THEN** re-opening or re-submitting the same link is rejected
