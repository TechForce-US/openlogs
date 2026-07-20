# project-management Specification

## Purpose
TBD - created by archiving change build-openlogs. Update Purpose after archive.
## Requirements
### Requirement: Create a project
An authenticated user SHALL be able to create a new project by providing a name. The system SHALL generate a cryptographically random API key for the project on creation.

#### Scenario: Project created successfully
- **WHEN** an authenticated user submits a valid project name
- **THEN** a new project is created, a unique API key is generated, and the user is shown the API key

#### Scenario: Duplicate project name
- **WHEN** an authenticated user submits a project name that already exists
- **THEN** an error is displayed and no project is created

#### Scenario: Empty project name rejected
- **WHEN** an authenticated user submits an empty project name
- **THEN** an error is displayed and no project is created

### Requirement: View project list
An authenticated user SHALL be able to view a list of all projects, including each project's name and creation date.

#### Scenario: Project list displayed
- **WHEN** an authenticated user navigates to `/projects`
- **THEN** all projects are listed with name and creation date

### Requirement: View and manage project settings
An authenticated user SHALL be able to view and edit a project's settings at `/projects/{id}/settings`, including renaming the project and regenerating the API key.

#### Scenario: Project renamed
- **WHEN** an authenticated user submits a new name for a project
- **THEN** the project name is updated

#### Scenario: API key regenerated
- **WHEN** an authenticated user requests a new API key for a project
- **THEN** a new cryptographically random API key is generated, the old key is invalidated immediately, and the new key is displayed

### Requirement: Delete a project
An authenticated user SHALL be able to delete a project. Deleting a project SHALL also permanently delete all log entries associated with that project.

#### Scenario: Project deleted with logs
- **WHEN** an authenticated user confirms deletion of a project
- **THEN** the project record and all associated log entries are deleted from the database

#### Scenario: Deletion requires confirmation
- **WHEN** an authenticated user initiates project deletion
- **THEN** a confirmation step is required before deletion proceeds

### Requirement: API key security
Project API keys SHALL be generated using a cryptographically secure random source and SHALL be at minimum 32 bytes of entropy, encoded as a hex or base64 string. API keys SHALL be stored in plaintext in the database (they are not passwords; they are not reversible secrets). They SHALL be displayed in the UI only on the project settings page.

#### Scenario: API key entropy
- **WHEN** a new API key is generated
- **THEN** it is at least 32 bytes of cryptographically random data

