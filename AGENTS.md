# Repository Guidelines

## Project Structure & Module Organization

Sponge is a Go code generator and Gin/gRPC framework. `cmd/sponge/` contains the CLI and generation UI; `cmd/protoc-gen-*/` contains Protobuf plugins. Reusable components live in `pkg/`. Service templates span `internal/` (handlers, services, DAO, models, configuration), `cmd/serverNameExample_*/`, and `api/` (Protobuf definitions and generated code).

Tests sit beside Go sources as `*_test.go`. `test/auto-test/` holds generation checks, `test/server/` provides dependency services, and `test/sql/` contains fixtures. Configuration examples live in `configs/`, deployment files in `deployments/`, automation in `scripts/`, documentation in `docs/`, and images in `assets/`. Bundled UI assets live in `cmd/sponge/server/static/`.

## Build, Test, and Development Commands

Run commands from the repository root. `go.mod` and CI use Go 1.27.1.

- `go build -o /tmp/sponge ./cmd/sponge`: build the CLI for your machine.
- `go run ./cmd/sponge run`: start the generation UI at `http://localhost:24631`.
- `make build-sponge`: cross-compile the CLI for Linux/amd64.
- `make build`: cross-compile the mixed HTTP/gRPC service template.
- `make test`: run uncached short tests, excluding `api/`, `cmd/`, and vendor packages.
- `make cover`: produce `cover.out` and open HTML coverage.
- `make ci-lint`: format Go files in place, then run GolangCI-Lint. Use v2.13.2 to match CI.

## Coding Style & Naming Conventions

Use Go formatting with tabs (`gofmt -s`) and `goimports`, grouping local imports under `github.com/go-dev-frame/sponge`. Follow `.golangci.yml`; its line-length limit is 200 characters. Use lowercase package names, exported `PascalCase` identifiers, and unexported `camelCase` identifiers.

Preserve generator placeholders such as `serverNameExample` and `userExample`, template markers, and related `.tpl`, `.exp`, and `.mgo` variants. Regenerate Protobuf output with `make proto FILES=api/path/file.proto` after installing the required plugins.

## Testing Guidelines

Use Go's `testing` package and existing Testify assertions; database/cache tests use SQLMock and miniredis helpers. Name tests `TestXxx` in adjacent `*_test.go` files. Use `t.TempDir()` for temporary artifacts. Run focused tests with `go test -count=1 ./pkg/cache`. Test changed CLI packages explicitly because `make test` excludes them.

Codecov targets 75% project coverage and 60% patch coverage; the patch target is informational. For generator changes, follow `test/auto-test/README.md` to validate generated services.

## Commit & Pull Request Guidelines

History commonly uses `feat:`, `fix:`, `docs:`, `build:`, `style:`, and `release:` prefixes, alongside imperative summaries. Prefer a concise, scoped subject such as `fix: handle empty query results`.

Describe the problem, resulting behavior, related issues, and validation commands in PRs. Include screenshots for visible UI changes and generated-output examples for template changes. Keep related templates and documentation consistent.
