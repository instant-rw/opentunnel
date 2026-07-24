# Security Policy

## Supported versions

OpenTunnel is under active development. Security fixes are applied to the
latest code on the default branch and, when practical, the latest published
release. Older releases are not supported.

## Reporting a vulnerability

Do not open a public issue for a suspected vulnerability. Email
[iranzithierry@opts.ink](mailto:iranzithierry@opts.ink) with:

- A description of the issue and its potential impact
- Affected versions, commits, endpoints, or components
- Reproduction steps or a minimal proof of concept
- Any suggested remediation
- A safe way to contact you

Do not include real credentials, access tokens, personal data, or captured
tunnel traffic. Use synthetic data and redact sensitive values.

You should receive an acknowledgement within seven days. Please allow time to
investigate and prepare a fix before disclosing the issue publicly. This
project does not currently operate a paid bug bounty program.

## Deployment responsibility

Self-hosters are responsible for securely configuring TLS, PostgreSQL,
credentials, backups, retention, network access, and platform permissions. Read
the project's [security model](docs/security.md) and
[operations guide](docs/operations.md) before exposing an instance publicly.
