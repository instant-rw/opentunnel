# OpenTunnel CLI

Install the latest release on macOS or Linux:

```sh
curl -fsSL https://opts.ink/install.sh | sh
```

On Windows:

```powershell
irm https://opts.ink/install.ps1 | iex
```

Authenticate and start forwarding:

```sh
opentunnel login
opentunnel domains create my-app
opentunnel up 3000 --domain my-app
```

`opentunnel up` remembers the selected domain and port. Later runs can use
`opentunnel up` without arguments. `OPENTUNNEL_API_URL` or the global
`--api-url` option selects a non-production control plane.

Update an installed release binary:

```sh
opentunnel update
```

`opentunnel update` downloads the matching GitHub release archive, verifies
`checksums.txt`, and replaces the running binary. Override the repository with
`OPENTUNNEL_REPOSITORY` (default `optunnel/opentunnel`).

## Server integration

The CLI connects to the server's authenticated binary WebSocket at:

```text
GET {control-plane-origin}/tunnel?domainId={uuid}
Authorization: Bearer {cli-token}
```

For the default base this resolves to
`wss://opts.ink/tunnel?domainId={uuid}`. Frames are protobuf
`opentunnel.tunnel.v1.Envelope` messages with protocol version `1`, as defined
in `shared/protocol/tunnel.proto`.

Tokens are stored in macOS Keychain, Secret Service on Linux, or Windows
Credential Locker. If the native facility is unavailable, the CLI uses a
user-only `0600` credential file in its configuration directory.
