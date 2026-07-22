## 1. Caddyfile

- [x] 1.1 Replace the hardcoded `logs.example.com` hostname in `Caddyfile` with `{env.OPENLOGS_URL}` and update the inline comment to explain the env var.

## 2. Docker Compose

- [x] 2.1 Add `OPENLOGS_URL: ${OPENLOGS_URL:-logs.example.com}` to the `caddy` service `environment` block in `docker-compose.yml`.

## 3. Environment template

- [x] 3.1 Add `OPENLOGS_URL=logs.example.com` (with a descriptive comment) to `.env.example`.

## 4. README

- [x] 4.1 Update any README instructions that tell users to edit the `Caddyfile` to instead tell them to set `OPENLOGS_URL` in `.env`.
