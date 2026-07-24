# syntax=docker/dockerfile:1.7

FROM golang:1.25-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates
COPY shared/go.mod shared/go.sum ./shared/
COPY backend/go.mod backend/go.sum ./backend/
RUN cd shared && go mod download && cd ../backend && go mod download
COPY shared ./shared
COPY backend ./backend
WORKDIR /src/backend
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/opentunnel-server ./cmd/server

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S opentunnel && adduser -S -G opentunnel opentunnel
COPY --from=build /out/opentunnel-server /usr/local/bin/opentunnel-server
USER opentunnel
EXPOSE 8080
ENV OPENTUNNEL_HTTP_ADDR=:8080
ENTRYPOINT ["/usr/local/bin/opentunnel-server"]
