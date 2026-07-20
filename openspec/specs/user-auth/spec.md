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
The system SHALL provide a `create-user` CLI subcommand (`openlogs create-user <email>`) that prompts for a password interactively, hashes it with bcrypt, and inserts a user record into the database. This is the only supported mechanism for creating users in v1.

#### Scenario: User created via CLI
- **WHEN** `openlogs create-user admin@example.com` is run and a valid password is entered
- **THEN** a new user record with bcrypt-hashed password is written to the database

#### Scenario: Duplicate email rejected
- **WHEN** `openlogs create-user` is run with an email that already exists in the database
- **THEN** an error is printed and no record is created

#### Scenario: Weak password warning
- **WHEN** a password shorter than 8 characters is entered during user creation
- **THEN** the CLI rejects it with an error message

