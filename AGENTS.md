# AGENTS.md - repo rules for agents

Testbox builds a Docker image from a repo
and runs it locally or in cloud. Config
lives in `make.yaml` at the repo root.

## Layout

- `cmd/core-service` - CLI entry.
  Commands: build/up/down/status/logs/run.
- `cmd/api-server` - HTTP API entry.
  Same Engine behind REST routes.
- `cmd/mcp-server` - MCP server entry.
  Tools from OpenAPI + custom file.
- `internal/engine` - orchestrator.
  Thin glue over parser/docker/deployer.
  Keeps CLI and API in sync. Stateless.
- `internal/parser` - reads make.yaml.
  Parse plus fail-fast validation.
- `internal/docker` - runs docker build.
  Pure arg logic stays testable.
- `internal/deployer` - compose files.
  Generate, up, down, status, logs.
- `internal/mcp` - MCP server logic.
  OpenAPI parse, tools merge, server.
- `internal/api` - HTTP handlers.
  Thin. No logic beyond Engine calls.
- `pkg/models` - shared structs.
  Source of truth for make.yaml.
- `scripts/` - agent-run checks.
- `.github/workflows/ci.yaml` - CI.
  Jobs: build-test and live-docker.

## Key files

- `pkg/models/make_yaml_types.go`
  All make.yaml shapes live here.
- `internal/engine/engine.go`
  All flows start here. Read first.
- `.testbox/` - generated workspace.
  Compose file plus mcp build dir.
  Never commit. Already gitignored.

## Commands

- `make lint` - fmt check, vet, comments.
- `make test` - full unit suite.
- `make build` - builds all packages.
- `make install` - installs both CLIs.

## Comment rule (enforced)

- Max 2 lines per comment block.
- Max 40 chars of text per line.
- Text means content after `//`.
- Check: `sh scripts/check-comments.sh`.
- Keep comments short and plain.
- No chain-of-thought in comments.

## Lint rules (enforced by make lint)

- `gofmt` must report no files.
- `go vet ./...` must pass.
- Error strings start lowercase.
- Wrap errors with `%w` to keep cause.

## Code rules

- Fail fast. Return first error.
- No global state. Engine is stateless.
- Pure logic split from exec calls.
  Example: arg builders have own funcs.
- Absolute paths for generated files.
  Compose resolves relative to its dir.
- New deps need user approval first.

## Test rules

- Every feature ships with UTs.
- UTs must not need Docker daemon.
- Docker paths are proven by CI job
  `live-docker`, not by unit tests.
- Run `make lint` and `make test`
  before opening a PR.

## PR rules

- CI must pass before merge.
- Always squash and merge.
- Server enforces squash only.
- Owner merges with --admin flag.
- Others need owner approval first.
- One feature per PR with proof:
  what ran, what output, what curl.

## Host gotcha

- Colima plus stale `credsStore`
  breaks image pulls. Symptom:
  `docker-credential-desktop missing`.
  Fix is one line in docker config,
  or a stripped DOCKER_CONFIG copy.
