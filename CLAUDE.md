# yuno-cli Project Instructions

CLI covering every operation of the public Yuno REST API. Architecture copied from the sibling
projects `linear-cli` and `orx-cli`: cobra, stdlib testing, `httptest`, golangci-lint v2, goreleaser.

## Project Files

- `docs/yuno-openapi.json` - Yuno OpenAPI 3.1.0 spec (authoritative source)
- `docs/yuno-operations.txt` - flat `METHOD /path` listing of all 172 operations (coverage checklist)
- `docs/yuno-api.md` - command-to-operation map and API conventions

### API Reference Priority

`docs/yuno-openapi.json` is the primary source of truth. Verify paths, request/response field names,
enums, nullability and path parameters against the spec when writing code and tests. Use
`docs/yuno-api.md` as a navigation aid, and resolve any discrepancy in favour of the spec.
Re-fetch the spec from <https://docs.y.uno/openapi.json> when it looks stale.

## Yuno-Specific Rules

These are non-negotiable - the CLI handles real money and real cardholder data.

- **Never log card data.** Card numbers, `security_code`/`cvv`, tokens, cryptograms and API keys must
  never reach stdout, stderr or a file unmasked.
- **Always mask.** Every output path goes through `internal/mask`: tables, `--json` and the
  `--verbose` HTTP dump. A new sensitive field name belongs in `mask.sensitiveFields`. `--unmask` is
  the single, explicit escape hatch.
- **Writes are confirmed.** Every non-`GET` operation goes through the `internal/confirm` gate before
  the request is sent. `--yes` (or `YUNO_ASSUME_YES`) is the only bypass; a non-TTY stdin without
  `--yes` fails rather than hangs.
- **Idempotency.** `X-Idempotency-Key` is generated for every `POST`/`PATCH`; `--idempotency-key`
  pins it.
- **No credentials in tests or fixtures.** Tests point at `httptest` servers via `YUNO_API_ENDPOINT`
  and use obviously fake keys.

## Layout

```
cmd/yuno-cli/main.go     # 10 lines: build the root command, map errors to exit codes
internal/cmd             # cobra commands, one file per resource group + its _test.go
internal/api             # HTTP client: auth headers, retry, typed errors, pagination
internal/config          # profiles, environments, env overrides
internal/confirm         # the write confirmation gate
internal/mask            # secret and card masking
internal/model           # request/response types
internal/output          # Formatter: JSON and tables
```

## Command Conventions

- Shape: `yuno-cli <resource> <action> [id] [flags]`. Path parameters are positional; everything
  else is a flag.
- Three-level constructors: `newXCommand` -> `newXListCommand` -> `runXList`.
- Flags are read untyped inside the run function (`flagString(cmd, "account-id")`).
- Every leaf command that talks to the API declares its operations with
  `Annotations: apiOperations("GET /payments/{payment_id}")`. `internal/cmd/coverage_test.go` fails
  if any spec operation has no command, and `internal/cmd/docs_test.go` fails if `docs/yuno-api.md`
  drifts from the command tree - regenerate the table when adding a command.
- Large payloads come from `--file`/`--data` (`registerBodyFlags`); frequently used fields also get
  named flags overlaid on top of the payload.
- Update bodies are `map[string]any` built from `flags.Changed(...)` so "absent" and "empty" stay
  distinguishable. Never use typed structs with `omitempty` for PATCH input.
- Persistent flags (`--json`, `--profile`, `--verbose`, `--yes`, `--unmask`, `--timeout`) are
  declared once on the root command.
- Exit codes: `0` success, `1` API error, `2` confirmation declined, `3` config/input error - see
  `internal/cmd/exit.go`.

## Code Style

### Imports

Group in order, separated by blank lines: standard library, external packages, local packages
(`github.com/pvarganov/yuno-cli/...`).

### Naming

- Package names: short, lowercase, no underscores (`api`, `config`, `mask`)
- Exported: PascalCase; unexported: camelCase
- Acronyms keep a consistent case (`URL`, `HTTP`, `API`)
- Receivers: 1-2 letters (`c` for `*Client`, `g` for `*Gate`)
- Sentinel errors: `Err` prefix, package-level `var`

### Functions

- Early returns for error handling
- Max 80 lines / 50 statements, cyclomatic complexity <= 10, nesting <= 5

### Error Handling

- Wrap with context, lowercase, no trailing dot: `fmt.Errorf("list routings: %w", err)`
- Check every error (enforced by `errcheck`)
- Compare with `errors.Is`/`errors.As`

### Structs

- JSON tags on all exported fields, `omitempty` for optional ones
- Pointer types (`*string`, `*int`) for optional request fields

### Comments

Only for non-obvious logic; English, brief. No comments for self-explanatory code.

## Testing

- stdlib `testing` only - **testify and mock libraries are banned**
- HTTP is mocked with `httptest.NewServer`, wired in via `YUNO_API_ENDPOINT`; config is redirected
  with `YUNO_CONFIG_DIR`
- `internal/api` uses an internal test package (`package api`) so `sleep` can be swapped;
  `internal/cmd` uses `package cmd_test` and drives the real root command with `root.SetArgs(...)`
- Every command test asserts the HTTP method, the path and the request body
- Table-driven tests for multiple scenarios; test error paths (4xx bodies, 5xx retry, cancellation)
- Run with the race detector: `make test` (`go test -race ./...`)

## Building

- Always build with `make build` (runs the linter first). Direct `go build` skips linting - avoid it.
- `make lint` runs `golangci-lint run`; fix formatting with `gofmt -w` / `goimports -w`.

## Language

All code, comments, documentation and user-facing strings are in English.
