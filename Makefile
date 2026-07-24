.PHONY: backend-dev frontend-dev cli-build generate check format build

backend-dev:
	go run ./backend/cmd/server

frontend-dev:
	cd frontend && bun run dev

cli-build:
	go build -o bin/opentunnel ./cli/cmd/opentunnel

generate:
	cd shared && go generate ./gen/api
	cd shared && go run github.com/bufbuild/buf/cmd/buf@v1.72.0 generate
	cd frontend && bunx openapi-typescript ../shared/api/openapi.yaml -o src/lib/api.generated.ts
	cd frontend && bunx prettier --write src/lib/api.generated.ts

format:
	cd frontend && bun run format
	gofmt -w backend cli shared

check:
	cd frontend && bun run typecheck
	cd frontend && bun run build
	go test ./backend/... ./cli/... ./shared/...
	go build ./backend/cmd/server ./cli/cmd/opentunnel

build: cli-build
	go build -o bin/opentunnel-server ./backend/cmd/server
	cd frontend && bun run build
