# DEV.md

## Setup

```sh
mise trust && mise bootstrap
```

## Commands

If `mise` is unavailable, inform the user. Always prefer these tasks over running anything directly.

```bash
mise run build          # Build binary to bin/wt
mise run test           # Run unit tests
mise run test-int       # Run unit+integration tests (//go:build integration)
mise run test-coverage  # Generate coverage.html
mise run lint           # version consistency, go mod tidy -diff, golangci-lint, actionlint, markdownlint-cli2
mise run lint-fix       # Auto-fix lint issues
mise run wt -- [ARGS]   # Run wt with arguments: mise run wt -- --version
mise run install        # Build and copy to $(go env GOPATH)/bin
mise run update-deps    # Upgrade Go modules + pinned tools, refresh lockfiles
mise run release-check  # Build snapshot release artifacts and dry-run npm packaging
mise run clean          # Remove build and coverage artifacts
```

Run a single test:

```bash
go test -v -run TestRootCommand ./internal/cmd/...
```

## Data locations

- Global config: `~/.wt/config.yaml`
- Worktree directories: `~/.wt/worktrees/<repo-name>/<wt-name>/`
- Branch metadata: `git config branch.<name>.wt-parent`

## Code style

- Prefer deterministic int comparisons (e.g., exit codes) over string parsing where possible
- Use functional options (`WithX()`) for dependency injection (e.g., `NewManager` options)
- Use plain structs for "data bag" parameters (e.g., `ForkOptions`)

## Testing

Integration tests (`//go:build integration`) should use helpers from `internal/testutil`.
