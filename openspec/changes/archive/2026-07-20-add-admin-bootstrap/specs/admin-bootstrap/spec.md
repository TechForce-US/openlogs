## ADDED Requirements

### Requirement: Bootstrap admin from environment variables
On startup of the `serve` command, when both `OPENLOGS_ADMIN_EMAIL` and `OPENLOGS_ADMIN_PASSWORD` are set to non-empty values, the system SHALL create a user with that email and password if no user with that email already exists. The password SHALL be stored bcrypt-hashed, consistent with `create-user`.

#### Scenario: Admin created on first startup
- **WHEN** the server starts with `OPENLOGS_ADMIN_EMAIL` and `OPENLOGS_ADMIN_PASSWORD` set and no matching user exists
- **THEN** a user with that email and a bcrypt-hashed password is created and can log in with those credentials

#### Scenario: Both variables required
- **WHEN** only one of `OPENLOGS_ADMIN_EMAIL` or `OPENLOGS_ADMIN_PASSWORD` is set (the other empty)
- **THEN** no user is created and startup proceeds normally

#### Scenario: Neither variable set
- **WHEN** the server starts with both variables unset
- **THEN** no bootstrap occurs and startup behaviour is unchanged

### Requirement: Idempotent and non-destructive
The bootstrap SHALL be idempotent. When a user with the configured email already exists, the system SHALL leave that user unchanged (it SHALL NOT reset the password) and continue startup.

#### Scenario: Existing user is not modified
- **WHEN** the server starts with the admin variables set and a user with that email already exists
- **THEN** the existing user's password hash is left unchanged and the server starts normally

#### Scenario: Safe across restarts
- **WHEN** the server is restarted with the same admin variables still set
- **THEN** no duplicate user is created and no password is reset

### Requirement: Password policy enforced
The bootstrap password SHALL be at least 8 characters. When a shorter password is supplied, the system SHALL fail startup with a clear error naming the offending variable and SHALL NOT create the user.

#### Scenario: Short password rejected at startup
- **WHEN** the server starts with `OPENLOGS_ADMIN_PASSWORD` shorter than 8 characters and no matching user exists
- **THEN** startup fails with an error identifying `OPENLOGS_ADMIN_PASSWORD` and no user is created
