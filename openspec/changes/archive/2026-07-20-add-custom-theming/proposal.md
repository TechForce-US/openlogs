## Why

Self-hosted users want to tweak OpenLogs' appearance (colours, spacing, logo, small layout changes) without forking the project or editing the built-in themes. Today the only styling knobs are the two built-in themes selected via `OPENLOGS_THEME`. A drop-in custom CSS mechanism lets operators override any built-in style at deploy time with no rebuild.

## What Changes

- Add a `web/static/custom/` directory where operators drop `.css` files.
- The directory's `.css` files are **gitignored** so local overrides are never committed (a `.gitkeep` keeps the directory present in the repo).
- Custom CSS is read from the **filesystem at runtime** (not embedded), so files can be added/changed without rebuilding the binary. A configurable path (`OPENLOGS_CUSTOM_CSS_DIR`) allows Docker volume mounts.
- The base layout enumerates all `.css` files in the custom directory (sorted) and injects a `<link>` for each into `<head>` **after** the built-in theme stylesheet, so custom rules win via the cascade.
- Custom files are served over HTTP from the filesystem at `/static/custom/<file>`.
- Missing or empty custom directory is a no-op (no links, no errors).
- Document the mechanism (including the Docker volume mount) in the README.

## Capabilities

### New Capabilities

- `custom-theming`: Load operator-provided CSS files from a runtime directory and apply them after the built-in theme so any built-in style can be overridden without rebuilding.

### Modified Capabilities

## Impact

- **Code**: base template (`web/templates/base.html`), handler layout data + a filesystem static route (`internal/handler`), config (`internal/config` — new `OPENLOGS_CUSTOM_CSS_DIR`).
- **Assets**: new `web/static/custom/` directory with `.gitkeep`; `.gitignore` updated to ignore `web/static/custom/*.css`.
- **Deployment**: `docker-compose.yml` / `docker-compose.local.yml` gain an optional volume mount for the custom directory; README documents usage.
- **No breaking changes**: with no custom files present, behaviour is identical to today.
