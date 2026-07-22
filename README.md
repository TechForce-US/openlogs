# OpenLogs

A self-hosted, lightweight log management server. Applications send structured
JSON log entries over HTTP; OpenLogs stores them in SQLite (with full-text
search), and shows them in a real-time web UI. It ships as a single Go binary
with no runtime dependencies.

- **Framework-agnostic ingest** — any app that can POST JSON can send logs. The
  wire format matches Monolog's `JsonFormatter`, so PHP/Laravel works out of the box.
- **Multi-project** — each project has its own API key.
- **Live tail** — new logs stream to the browser over Server-Sent Events.
- **Full-text search** — powered by SQLite FTS5 over message and context.
- **Two themes** — `modern` and `terminal`, switchable at runtime.
- **Automatic retention** — logs older than 30 days (configurable) are pruned daily.
- **Simple deploy** — Docker Compose + Caddy for automatic HTTPS, or run the
  binary behind any reverse proxy.

---

## Quickstart (Docker Compose + Caddy)

Caddy handles TLS automatically for a public domain.

1. Point your domain's DNS at the server.
2. Create a `.env` file with your domain and a session secret:

   ```sh
   echo "OPENLOGS_URL=logs.example.com" > .env
   echo "OPENLOGS_SECRET_KEY=$(openssl rand -hex 32)" >> .env
   ```

   Replace `logs.example.com` with your actual domain. No need to edit `Caddyfile`.

4. Start the stack:

   ```sh
   docker compose up -d
   ```

5. Create your first user (interactive password prompt):

   ```sh
   docker compose exec openlogs openlogs create-user you@example.com
   ```

   Alternatively, set `OPENLOGS_ADMIN_EMAIL` and `OPENLOGS_ADMIN_PASSWORD` (see
   [Configuration](#configuration)) to have the admin created automatically on
   startup, skipping this step.

6. Open `https://your-domain` and log in.

The SQLite database lives in `./data/openlogs.db` — back up that directory. To
keep logs beyond the retention window, copy/rotate the `.db` file before pruning.

---

## Run locally (Docker Compose, no TLS)

For local development there's a dedicated `docker-compose.local.yml` that runs
OpenLogs on its own — no Caddy, no TLS — exposed directly on
`http://localhost:8080`. A default session secret is baked in so it works with
zero configuration.

1. Build and start the container:

   ```sh
   docker compose -f docker-compose.local.yml up --build -d
   ```

2. Create your first user (interactive password prompt):

   ```sh
   docker compose -f docker-compose.local.yml exec openlogs openlogs create-user you@example.com
   ```

   Or set `OPENLOGS_ADMIN_EMAIL` / `OPENLOGS_ADMIN_PASSWORD` to auto-create it on
   startup (see [Configuration](#configuration)).

3. Open `http://localhost:8080` and log in.

Stop the stack when you're done:

```sh
docker compose -f docker-compose.local.yml down
```

Notes:

- The SQLite database persists in `./data`, and custom CSS overrides can be dropped
  into `./custom` (see [Custom theming](#custom-theming)).
- Override any setting by exporting the matching variable (or adding it to a `.env`
  file) before running compose, e.g. `OPENLOGS_THEME=terminal`.
- The default `OPENLOGS_SECRET_KEY` is for local use only — never use it in
  production. For a public deployment use the main `docker-compose.yml` (Caddy + TLS).

---

## Manual setup (any reverse proxy)

OpenLogs is a single binary that listens on `:8080` (configurable). Put any
reverse proxy (nginx, Apache, Traefik, …) in front of it for TLS.

```sh
go build -o openlogs ./cmd/openlogs

export OPENLOGS_SECRET_KEY=$(openssl rand -hex 32)
export OPENLOGS_DB_PATH=./openlogs.db
export OPENLOGS_THEME=modern

./openlogs create-user you@example.com   # one-time
./openlogs serve
```

Example nginx location block:

```nginx
location / {
    proxy_pass http://127.0.0.1:8080;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-Proto $scheme;   # marks session cookies Secure

    # Required for the live-tail SSE endpoint:
    proxy_set_header Connection '';
    proxy_buffering off;
    proxy_read_timeout 1h;
}
```

---

## Configuration

All configuration is via environment variables:

| Variable | Default | Description |
|---|---|---|
| `OPENLOGS_DB_PATH` | `openlogs.db` | Path to the SQLite database file |
| `OPENLOGS_SECRET_KEY` | *(required)* | Secret for session signing; server exits if unset |
| `OPENLOGS_PORT` | `8080` | HTTP listen port |
| `OPENLOGS_THEME` | `modern` | UI theme: `modern` or `terminal` |
| `OPENLOGS_RETENTION_DAYS` | `30` | Delete logs older than this many days |
| `OPENLOGS_CUSTOM_CSS_DIR` | `web/static/custom` | Directory of custom `.css` overrides (see below) |
| `OPENLOGS_ADMIN_EMAIL` | *(unset)* | With `OPENLOGS_ADMIN_PASSWORD`, auto-creates this admin user on startup (see below) |
| `OPENLOGS_ADMIN_PASSWORD` | *(unset)* | Password for the bootstrap admin user (min 8 chars) |

### Automatic admin user

If both `OPENLOGS_ADMIN_EMAIL` and `OPENLOGS_ADMIN_PASSWORD` are set, the server
creates that user on startup — handy for Docker deployments where you'd otherwise
run `create-user` manually. It is **idempotent**: if the user already exists it is
left untouched (the password is not reset), so it's safe across restarts. Leave
both unset to manage users only via `openlogs create-user`.

---

## Custom theming

Beyond the two built-in themes, you can override any style by dropping `.css`
files into a directory — no rebuild required. Every `.css` file in the directory
is linked into each page **after** the built-in theme, so your rules win by the
normal CSS cascade (no `!important` needed). Files load in alphabetical order, so
prefix them to control ordering (e.g. `00-vars.css`, `10-layout.css`).

The directory is read at request time, so newly added files appear on the next
page load. If the directory is missing or empty, nothing changes.

**Binary / `go run`:** files live in `web/static/custom/` by default (that
directory is kept in the repo but its `.css` contents are gitignored). Point
`OPENLOGS_CUSTOM_CSS_DIR` elsewhere if you prefer.

```sh
echo ':root { --accent: #e11d48; }' > web/static/custom/overrides.css
```

**Docker:** the compose files mount `./custom` into the container and set
`OPENLOGS_CUSTOM_CSS_DIR=/custom`. Drop your `.css` files into `./custom` next to
the compose file and refresh:

```sh
mkdir -p custom
echo ':root { --accent: #e11d48; }' > custom/overrides.css
```

> Custom CSS has full power over the UI — if a change breaks the layout, just
> remove the file to restore the built-in look.

---

## Sending logs

`POST /api/ingest` with your project's API key in the `X-API-Key` header and a
JSON body:

```sh
curl -X POST https://your-domain/api/ingest \
  -H "X-API-Key: <project-api-key>" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Payment failed",
    "level_name": "ERROR",
    "level": 400,
    "channel": "billing",
    "datetime": "2026-07-18T14:23:00+00:00",
    "context": {"user_id": 42, "amount": 99.0},
    "extra": {}
  }'
```

### Payload fields

| Field | Type | Required | Notes |
|---|---|---|---|
| `message` | string | yes | Log message |
| `level_name` | string | yes | `DEBUG`,`INFO`,`NOTICE`,`WARNING`,`ERROR`,`CRITICAL`,`ALERT`,`EMERGENCY` |
| `level` | integer | no | Monolog numeric level (100–600); used for ordering |
| `channel` | string | no | Logical source name |
| `datetime` | string | yes | ISO 8601 timestamp |
| `context` | object | no | Structured data (searchable) |
| `extra` | object | no | Processor metadata |

Get an API key by creating a project in the UI (**Projects → Create project**);
the key is shown on the project's settings page.

---

## Laravel / Monolog adapter (reference)

OpenLogs' ingest format is exactly Monolog's `JsonFormatter` output, so a thin
custom handler is all you need. (A dedicated Composer package may come later.)

Create the handler:

```php
<?php

namespace App\Logging;

use Monolog\Handler\AbstractProcessingHandler;
use Monolog\Formatter\JsonFormatter;
use Monolog\LogRecord;

class OpenLogsHandler extends AbstractProcessingHandler
{
    public function __construct(
        private string $endpoint,
        private string $apiKey,
        $level = \Monolog\Level::Debug,
        bool $bubble = true,
    ) {
        parent::__construct($level, $bubble);
    }

    protected function getDefaultFormatter(): \Monolog\Formatter\FormatterInterface
    {
        return new JsonFormatter();
    }

    protected function write(LogRecord $record): void
    {
        $ch = curl_init($this->endpoint);
        curl_setopt_array($ch, [
            CURLOPT_POST => true,
            CURLOPT_POSTFIELDS => $record->formatted,
            CURLOPT_RETURNTRANSFER => true,
            CURLOPT_TIMEOUT => 3,
            CURLOPT_HTTPHEADER => [
                'Content-Type: application/json',
                'X-API-Key: ' . $this->apiKey,
            ],
        ]);
        curl_exec($ch);
        curl_close($ch);
    }
}
```

Wire it up as a custom channel in `config/logging.php`:

```php
'channels' => [
    'openlogs' => [
        'driver' => 'monolog',
        'handler' => App\Logging\OpenLogsHandler::class,
        'handler_with' => [
            'endpoint' => env('OPENLOGS_ENDPOINT', 'https://your-domain/api/ingest'),
            'apiKey'   => env('OPENLOGS_API_KEY'),
        ],
    ],

    // Optionally send to OpenLogs and the local file at once:
    'stack' => [
        'driver'   => 'stack',
        'channels' => ['single', 'openlogs'],
    ],
],
```

Then set `LOG_CHANNEL=openlogs` (or `stack`) and add `OPENLOGS_ENDPOINT` /
`OPENLOGS_API_KEY` to your `.env`.

> For production, consider queueing or buffering log delivery so a slow OpenLogs
> instance never blocks a request. Monolog's `BufferHandler` around
> `OpenLogsHandler` is an easy start.

---

## Development

```sh
go build ./...      # compile
go vet ./...        # static checks
go test ./...       # run all tests
go run ./cmd/openlogs create-user you@example.com
OPENLOGS_SECRET_KEY=dev go run ./cmd/openlogs serve
```

### Seeding with real log data (`importlogs`)

`importlogs` parses a Laravel log file (the default `LineFormatter` output, e.g.
`storage/logs/laravel.log`) and inserts every entry directly into the SQLite
database — no running server required. It is useful for populating a local
development database with realistic data.

```sh
go run ./cmd/importlogs [flags]
```

| Flag | Default | Description |
|---|---|---|
| `-file` | `laravel.log` | Path to the Laravel log file to import |
| `-db` | `$OPENLOGS_DB_PATH` or `data/openlogs.db` | Path to the OpenLogs SQLite database |
| `-project` | `laravel` | Project name or ID to import into; created automatically if absent |
| `-recent` | `false` | Shift all timestamps forward so the newest entry lands at *now*, preserving relative spacing — use this to keep entries inside the 30-day retention window |

**Example — import and keep entries visible:**

```sh
OPENLOGS_SECRET_KEY=dev go run ./cmd/openlogs serve &
go run ./cmd/importlogs -file storage/logs/laravel.log -recent
```

Without `-recent`, entries whose timestamps predate the 30-day retention window
will be pruned on the next server start. The tool prints a reminder if this is
the case.

## Contributing

OpenLogs is built with **spec-driven development** using
[OpenSpec](https://github.com/Fission-AI/OpenSpec). Every contribution ships as an
OpenSpec change (proposal, design, specs, tasks) and includes unit tests. See
[CONTRIBUTING.md](CONTRIBUTING.md) for the full workflow and PR checklist.

## License

MIT
