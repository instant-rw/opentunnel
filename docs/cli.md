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
opentunnel up --domain my-app 3000
```

`opentunnel up` remembers the selected domain and port. Later runs can use
`opentunnel up` without arguments. `OPENTUNNEL_API_URL` or the global
`--api-url` option selects a non-production control plane.

## Server integration

The CLI connects to the server's authenticated binary WebSocket at:

```text
GET {control-plane-origin}/tunnel?domainId={uuid}
Authorization: Bearer {cli-token}
```

For the default base this resolves to
`wss://opts.ink/tunnel?domainId={uuid}`. Frames are protobuf
`opentunnel.tunnel.v1.Envelope` messages with protocol version `1`, as defined
in `protocol/tunnel.proto`.

Tokens are stored in macOS Keychain, Secret Service on Linux, or Windows
Credential Locker. If the native facility is unavailable, the CLI uses a
user-only `0600` credential file in its configuration directory.
