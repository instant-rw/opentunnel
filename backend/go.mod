module github.com/opentunnel/opentunnel/backend

go 1.25.0

require (
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/jackc/pgx/v5 v5.7.6
	github.com/oapi-codegen/runtime v1.6.0
	github.com/opentunnel/opentunnel/shared v0.0.0
	golang.org/x/crypto v0.53.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/go-chi/chi/v5 v5.3.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.38.0 // indirect
)

replace github.com/opentunnel/opentunnel/shared => ../shared
