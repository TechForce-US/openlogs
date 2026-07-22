.PHONY: test build serve

test:
	go test ./...

build:
	go build ./cmd/openlogs/...

serve:
	OPENLOGS_SECRET_KEY=$${OPENLOGS_SECRET_KEY:-dev} go run ./cmd/openlogs serve
