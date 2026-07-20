## Context

OpenLogs ships as a single binary with all web assets embedded via `//go:embed` (`web/embed.go`). The base layout links exactly one stylesheet — `/static/{theme}.css` — chosen by `OPENLOGS_THEME`. Static files are served from the embedded FS at `/static/`.

The custom-theming feature must let operators add CSS **after the binary is built** (drop a file, restart or refresh). That is fundamentally incompatible with `//go:embed`, which captures files at compile time. So custom CSS has to be a runtime, filesystem-backed concern, deliberately separate from the embedded built-in assets.

## Goals / Non-Goals

**Goals:**
- Operators can override any built-in style by dropping `.css` files into a directory, with no rebuild.
- Custom rules load after built-in CSS so they win by cascade order.
- Works for `go run` (repo layout), the compiled binary, and Docker (via volume mount).
- Absent/empty directory behaves exactly like today (zero custom links).
- Custom files stay out of version control.

**Non-Goals:**
- A per-user or per-project theme upload UI (files are placed by the operator on disk).
- Live hot-reload / websocket push of CSS changes (a browser refresh picks up changes).
- Validation, sanitising, or bundling/minifying of custom CSS.
- SCSS/LESS or any preprocessing.
- Custom JavaScript or template overrides (CSS only).

## Decisions

### 1. Runtime filesystem directory, not embedded

Custom CSS is read from a real directory at request/render time, served by a filesystem `http.FileServer` rooted at that directory — separate from the embedded `/static/` tree. This is the only approach that satisfies "add files without rebuilding."

**Alternative considered**: embedding a `custom/` dir — rejected because embed is compile-time; a gitignored dir would embed as empty and could never receive runtime files.

### 2. Configurable directory path (`OPENLOGS_CUSTOM_CSS_DIR`)

Default `web/static/custom` (resolves correctly under `go run` from the repo root). Operators of the compiled binary or Docker set it to a mounted path. Making it configurable avoids assumptions about the working directory and enables clean Docker volume mounts.

**Alternative considered**: hard-coding `web/static/custom` — rejected; the compiled binary's CWD rarely contains that path, and Docker needs an explicit mount target.

### 3. Serve custom files at `/static/custom/<file>` from disk

A dedicated route (`GET /static/custom/`) uses `http.FileServer(http.Dir(customDir))`, distinct from the embedded `/static/` handler. Route specificity in Go 1.22+ `ServeMux` means `/static/custom/` takes precedence over `/static/`. Only `.css` is linked by the layout, though the file server itself just serves the directory.

### 4. Enumerate on each layout render, tolerate absence

The handler lists `*.css` files in the custom dir (sorted lexicographically for deterministic order) and passes the filenames to the base template, which emits one `<link>` per file after the theme stylesheet. A directory read per page render is negligible at OpenLogs' scale and means newly added files appear on the next navigation without a restart. Missing directory → empty list → no links, no error.

**Alternative considered**: caching the listing at startup — rejected; it would require a restart to pick up new files, undermining the drop-in goal. (If profiling ever shows the stat/readdir as hot, a short TTL cache can be added later.)

### 5. Load order = built-in theme, then custom (sorted)

The cascade resolves later rules over earlier ones at equal specificity, so appending custom links after the theme link lets operators override without `!important`. Deterministic alphabetical ordering gives predictable results across multiple custom files (e.g. `00-vars.css`, `10-layout.css`).

### 6. Keep the directory in the repo, ignore its contents

Add `web/static/custom/.gitkeep` so the directory exists on a fresh clone; add `web/static/custom/*.css` (and `!.gitkeep`) to `.gitignore` so operator overrides are never committed.

## Risks / Trade-offs

- **Path confusion between `go run` and compiled binary** → default suits `go run`; README and Docker files document setting `OPENLOGS_CUSTOM_CSS_DIR` and mounting a volume for other deployments. → Mitigation: clear docs + sensible default + no-op when absent.
- **Operator CSS can break the UI** → by design; custom CSS has full power. → Mitigation: documented as an advanced, at-your-own-risk feature; removing the files restores the built-in look.
- **Directory read per render** → trivial cost at this scale; acceptable for the drop-in UX. Path is fixed per process, so no traversal risk from user input.
- **Serving arbitrary files from the custom dir** → the file server only exposes the operator-controlled directory; operators should place only CSS there. Not user-writable via the app.

## Open Questions

- None for v1. A future enhancement could add a settings toggle to enable/disable custom CSS without moving files.
