# syntax=docker/dockerfile:1

# ---- build stage ----
FROM golang:1.25 AS build
WORKDIR /src

# Cache dependencies first.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Pure-Go build (no CGO) produces a static binary that runs on Alpine.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /openlogs ./cmd/openlogs

# ---- runtime stage ----
FROM alpine:3.20
RUN apk add --no-cache su-exec \
    && adduser -D -u 10001 openlogs \
    && mkdir -p /data \
    && chown openlogs:openlogs /data
COPY --from=build /openlogs /usr/local/bin/openlogs
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh
# Run from /data so the default relative DB path resolves to a writable, persisted
# location even when run without an explicit OPENLOGS_DB_PATH.
WORKDIR /data
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["serve"]
