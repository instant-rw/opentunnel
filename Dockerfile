# syntax=docker/dockerfile:1.7

FROM node:22-alpine AS web
WORKDIR /src
COPY package.json package-lock.json ./
COPY web/package.json web/package.json
RUN npm ci
COPY api api
COPY web web
RUN npm run build --workspace web

FROM golang:1.24-alpine AS server
WORKDIR /src
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/out /tmp/web-out
RUN rm -rf internal/webassets/out && \
    cp -R /tmp/web-out internal/webassets/out && \
    test -d internal/webassets/out/_next/static/chunks && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/opentunnel-server ./cmd/server

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S opentunnel && adduser -S -G opentunnel opentunnel
COPY --from=server /out/opentunnel-server /usr/local/bin/opentunnel-server
USER opentunnel
EXPOSE 8080
ENV OPENTUNNEL_HTTP_ADDR=:8080
ENTRYPOINT ["/usr/local/bin/opentunnel-server"]
