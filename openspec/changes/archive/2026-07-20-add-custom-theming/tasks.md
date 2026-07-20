## 1. Configuration

- [x] 1.1 Add `CustomCSSDir` to `config.Config` in `internal/config/config.go`, read from `OPENLOGS_CUSTOM_CSS_DIR`, defaulting to `web/static/custom`

## 2. Custom CSS loading

- [x] 2.1 Add a helper (e.g. `internal/handler/app.go`) that lists `*.css` files in the configured custom dir, sorted lexicographically, returning an empty slice when the dir is missing/empty (no error)
- [x] 2.2 Include the custom CSS filenames in the base-layout render data (extend `pageData`) so every page render reflects the current directory contents

## 3. Serve & link custom CSS

- [x] 3.1 In `internal/handler/router.go`, add a filesystem route `GET /static/custom/` using `http.FileServer(http.Dir(customDir))`, ensuring it takes precedence over the embedded `/static/` handler
- [x] 3.2 Update `web/templates/base.html` to loop over the custom CSS filenames and emit `<link rel="stylesheet" href="/static/custom/{file}">` after the built-in `/static/{theme}.css` link

## 4. Repository & directory setup

- [x] 4.1 Create `web/static/custom/.gitkeep` so the directory exists on a fresh clone
- [x] 4.2 Update `.gitignore` to ignore `web/static/custom/*.css` while keeping `.gitkeep` tracked (also ignores the Docker `/custom/` mount)
- [x] 4.3 Verify the `//go:embed` in `web/embed.go` still builds with the new directory present (builds clean; `.gitkeep` embeds harmlessly, custom `.css` is served from disk not embed)

## 5. Deployment & docs

- [x] 5.1 Add an optional custom-dir volume mount + `OPENLOGS_CUSTOM_CSS_DIR` example to `docker-compose.yml` and `docker-compose.local.yml` (mounts `./custom` → `/custom`)
- [x] 5.2 Document the custom theming mechanism in `README.md`: where to drop files, load order, the env var, and the Docker mount

## 6. Verification

- [x] 6.1 With no custom files: confirm pages render only the built-in theme link and the app builds/runs (unit `TestCustomCSSAbsent`; full suite green)
- [x] 6.2 With a custom file present: confirm its `<link>` appears after the theme link, the file is served at `/static/custom/<file>` (200), and it visually overrides a built-in style (unit `TestCustomCSSInjectedAfterTheme`; live Docker stack served override 200)
- [x] 6.3 With multiple files: confirm sorted load order; with a non-`.css` file: confirm it is ignored (unit `TestCustomCSSInjectedAfterTheme`)
- [x] 6.4 Confirm `web/static/custom/*.css` is gitignored while `.gitkeep` remains tracked (rules present in `.gitignore`; `.gitkeep` in place — note: project is not a git repo, so verified by inspection)
