.PHONY: build check dev format generate install

install:
	npm install

generate:
	npm run generate

format:
	npx prettier --write .
	gofmt -w cmd internal

check:
	npm run format:check
	npm run check
	go test ./cmd/... ./internal/...

build:
	npm run build
	go build ./cmd/server ./cmd/opentunnel

dev:
	go run ./cmd/server
