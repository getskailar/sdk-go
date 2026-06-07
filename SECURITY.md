# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in this SDK, please report it
privately to **security@skailar.com**. Do not open a public issue for
security-sensitive reports.

Please include:

- A description of the vulnerability and its impact.
- Steps to reproduce, or a proof-of-concept.
- The SDK version (`go.mod` entry) and Go version (`go version`).

We aim to acknowledge reports within 3 business days and to provide a
remediation timeline after triage.

## Handling of Credentials

This SDK reads the API key from the `SKAILAR_API_KEY` environment variable or
from an explicit `WithAPIKey` option. The key is sent only to the configured
base URL (`https://api.skailar.com` by default) over the `Authorization`
header. The SDK never logs the key, and the `Authorization` header cannot be
overridden by `WithDefaultHeader` or per-call headers.

Never commit API keys to source control. Rotate any key that may have been
exposed from the Skailar dashboard at https://skailar.com.

## Supported Versions

This project is pre-1.0. Security fixes are applied to the latest `0.0.x`
release.
