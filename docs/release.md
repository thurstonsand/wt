# Release

`wt` publishes all distribution channels from one `vX.Y.Z` tag:

- Go module source through the public Go proxy
- signed and notarized binaries through GitHub Releases
- the `wt` cask in `thurstonsand/homebrew-tap`
- the `@thurstonsand/pi-wt` package on npm

The release workflow runs the Go and Pi extension gates before publishing. GoReleaser cross-compiles the CLI, injects the tag-derived version, signs and notarizes Darwin binaries, creates archives and checksums, publishes the GitHub Release, and updates the Homebrew tap. The npm job publishes only after the binary release succeeds.

GoReleaser writes the Homebrew cask through a repository-scoped deploy key stored as `HOMEBREW_TAP_PRIVATE_KEY`. The matching public key has write access only to `thurstonsand/homebrew-tap`.

## Repository secrets

| Secret                                    | Purpose                                    |
| ----------------------------------------- | ------------------------------------------ |
| `HOMEBREW_TAP_PRIVATE_KEY`                | Update `thurstonsand/homebrew-tap`         |
| `APPLE_DEVELOPER_ID_CERTIFICATE_BASE64`   | Sign Darwin binaries                       |
| `APPLE_DEVELOPER_ID_CERTIFICATE_PASSWORD` | Unlock the Developer ID certificate        |
| `APPLE_NOTARY_KEY`                        | Authenticate with Apple's notarization API |
| `APPLE_NOTARY_KEY_ID`                     | Identify the App Store Connect API key     |
| `APPLE_NOTARY_ISSUER_ID`                  | Identify the App Store Connect API issuer  |

The Apple credentials are shared with GhosttyKit. GoReleaser uses Quill to sign and notarize the bare Darwin binaries from the Linux GitHub runner.

## Local verification

Validate the release definition and build unpublished artifacts:

```bash
mise run release-check
```

Snapshot binaries report a prerelease version such as `v0.0.1-next`. Signing, publishing, and tap updates are disabled in snapshot mode.
