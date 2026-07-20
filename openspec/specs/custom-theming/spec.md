# custom-theming Specification

## Purpose
TBD - created by archiving change add-custom-theming. Update Purpose after archive.
## Requirements
### Requirement: Custom CSS directory
The system SHALL load operator-provided CSS from a runtime filesystem directory (not from embedded assets), so files can be added or changed without rebuilding the binary. The directory path SHALL be configurable via the `OPENLOGS_CUSTOM_CSS_DIR` environment variable, defaulting to `web/static/custom`.

#### Scenario: Default directory used
- **WHEN** `OPENLOGS_CUSTOM_CSS_DIR` is not set
- **THEN** the system reads custom CSS from `web/static/custom`

#### Scenario: Configured directory used
- **WHEN** `OPENLOGS_CUSTOM_CSS_DIR=/data/custom` is set
- **THEN** the system reads custom CSS from `/data/custom`

### Requirement: Inject custom stylesheets after built-in theme
The base layout SHALL enumerate all files ending in `.css` in the custom directory, sorted lexicographically by filename, and emit one `<link rel="stylesheet">` per file in `<head>` **after** the built-in theme stylesheet. This ordering SHALL allow custom rules to override built-in styles via the CSS cascade.

#### Scenario: Custom file overrides built-in style
- **WHEN** a `.css` file exists in the custom directory
- **THEN** its `<link>` appears in `<head>` after the `/static/{theme}.css` link on every rendered page

#### Scenario: Multiple custom files load in sorted order
- **WHEN** the custom directory contains `10-layout.css` and `00-vars.css`
- **THEN** the layout links `00-vars.css` before `10-layout.css`

#### Scenario: Non-CSS files are ignored
- **WHEN** the custom directory contains a file that does not end in `.css` (e.g. `notes.txt`)
- **THEN** no `<link>` is emitted for that file

### Requirement: Serve custom CSS over HTTP from the filesystem
The system SHALL serve files from the custom directory at the `/static/custom/<file>` path from the filesystem, distinct from the embedded `/static/` assets.

#### Scenario: Custom file is served
- **WHEN** a browser requests `/static/custom/overrides.css` and that file exists in the custom directory
- **THEN** the server responds with the file contents and HTTP 200

#### Scenario: Missing custom file
- **WHEN** a browser requests a custom CSS path that does not exist
- **THEN** the server responds with HTTP 404

### Requirement: Graceful behaviour when directory is absent or empty
When the custom directory does not exist or contains no `.css` files, the system SHALL render pages exactly as it does without the feature: no custom `<link>` tags and no errors.

#### Scenario: Directory missing
- **WHEN** the custom directory does not exist
- **THEN** pages render with only the built-in theme stylesheet and no error is raised

#### Scenario: Directory empty
- **WHEN** the custom directory exists but contains no `.css` files
- **THEN** pages render with only the built-in theme stylesheet

### Requirement: Custom files excluded from version control
The repository SHALL keep the custom directory present (via a tracked placeholder such as `.gitkeep`) while ignoring operator-provided `.css` files so they are never committed.

#### Scenario: Custom CSS is gitignored
- **WHEN** an operator adds `overrides.css` to the custom directory
- **THEN** the file is ignored by git and the directory itself remains tracked

