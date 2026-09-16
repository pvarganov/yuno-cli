# yuno-cli

Command-line interface for the full [Yuno](https://y.uno) REST API: all 172 operations of the
public API (payments, routing, connections, organizations, subscriptions, payouts, recipients,
webhooks, reports, sellers, PCI proxy, banking connectivity and the rest) in one binary, with
pipe-friendly JSON output.

## Installation

### Homebrew

```
brew install --cask pvarganov/tap/yuno-cli
```

### Binary releases

Download pre-built binaries for Linux and macOS from
[GitHub Releases](https://github.com/pvarganov/yuno-cli/releases).

### From source

Requirements: Go 1.25+

```
git clone https://github.com/pvarganov/yuno-cli
cd yuno-cli
make build
```

The binary is placed at `./yuno-cli`. Move it somewhere in your `PATH`:

```
mv yuno-cli /usr/local/bin/
```

## Authentication

yuno-cli needs a Yuno API key pair: a **public API key** and a **private secret key**, issued in the
Yuno dashboard under Developers → API keys. Keys are environment-specific — a sandbox key does not
work against production.

```
yuno-cli auth --environment sandbox
```

The command prompts for both keys (input is not echoed on a TTY) and writes them to
`~/.config/yuno-cli/config.yaml` with `0600` permissions inside a `0700` directory. Leave a prompt
empty to keep the key already stored.

Optional profile defaults:

```
yuno-cli auth --environment prod-us \
  --account-code <code> \
  --organization-code <code> \
  --account-id <uuid>
```

`--account-code` and `--organization-code` become the `X-Account-Code` and `X-Organization-Code`
headers; `--account-id` is the default `account_id` used by account-scoped commands such as
`yuno-cli routing list`. Note that `account_id` is a UUID and is *not* the same value as the account
code.

Check what is in effect — the credentials are printed masked:

```
yuno-cli auth status
yuno-cli auth status --json
```

## Profiles

A profile is a named set of credentials plus the environment they belong to. The three supported
environments map to the servers declared in the OpenAPI spec:

| Environment | Base URL |
|---|---|
| `sandbox` | `https://api-sandbox.y.uno/v1` |
| `prod-us` | `https://api.y.uno/v1` |
| `prod-eu` | `https://api.eu.y.uno/v1` |

```
yuno-cli auth --profile sandbox --environment sandbox
yuno-cli auth --profile prod    --environment prod-us

yuno-cli profile list
yuno-cli profile use prod          # make it the default
yuno-cli profile delete sandbox
```

Pick a profile per command with `--profile`:

```
yuno-cli payment get <payment_id> --profile sandbox
```

Resolution order for the active profile: `--profile` flag → `YUNO_PROFILE` → `default_profile` in
the config file → the profile literally named `default`.

### Configuration file

`~/.config/yuno-cli/config.yaml`:

```yaml
default_profile: sandbox
profiles:
  sandbox:
    environment: sandbox
    public_api_key: pk_xxx
    private_secret_key: sk_xxx
  prod:
    environment: prod-us
    public_api_key: pk_yyy
    private_secret_key: sk_yyy
    account_code: ACC123
    account_id: 00000000-0000-0000-0000-000000000000
```

### Environment variables

Environment variables override the stored profile, which makes the CLI usable in CI without a config
file at all:

| Variable | Effect |
|---|---|
| `YUNO_PROFILE` | profile to use |
| `YUNO_PUBLIC_API_KEY` | overrides `public_api_key` |
| `YUNO_PRIVATE_SECRET_KEY` | overrides `private_secret_key` |
| `YUNO_ACCOUNT_CODE` | overrides `account_code` (`X-Account-Code`) |
| `YUNO_ORGANIZATION_CODE` | overrides `organization_code` (`X-Organization-Code`) |
| `YUNO_ACCOUNT_ID` | overrides the default `account_id` |
| `YUNO_API_ENDPOINT` | overrides the base URL of the environment |
| `YUNO_CONFIG_DIR` | overrides `~/.config/yuno-cli` |
| `YUNO_ASSUME_YES` | truthy value behaves like `--yes` |

## Usage

```
yuno-cli <resource> <action> [id] [flags]
```

```
yuno-cli payment get <payment_id>
yuno-cli payment list --merchant-order-id order-42
yuno-cli routing list --account-id <uuid> --payment-method CARD
yuno-cli customer update <customer_id> --email new@example.com
yuno-cli payment create --file payment.json --idempotency-key <uuid>
```

`yuno-cli --help` lists every resource group; `yuno-cli <resource> --help` lists its actions. The
full command-to-operation map is in [`docs/yuno-api.md`](docs/yuno-api.md).

### Persistent flags

| Flag | Effect |
|---|---|
| `--json` | print raw JSON instead of a table |
| `--profile <name>` | profile to use for this command |
| `--verbose` | dump the HTTP request and response to stderr (masked) |
| `--yes` | skip the confirmation prompt of write operations |
| `--unmask` | print card data and credentials unmasked |
| `--timeout <duration>` | per-request timeout (default `30s`), e.g. `--timeout 2m` |

### Request bodies

Large payloads come from a file or from stdin; the named flags of a command are overlaid on top of
the payload and only when explicitly set:

```
yuno-cli payment create --file payment.json
echo '{"email":"a@b.c"}' | yuno-cli customer create --data @-
yuno-cli customer create --data '{"email":"a@b.c","country":"CO"}'
```

Because update bodies are built only from the flags you actually passed, `--email ""` clears the
field while omitting `--email` leaves it untouched.

### Pagination

List commands fetch every page by default. `--limit N` stops after `N` items (`--limit 0`, the
default, fetches everything); `--page-size N` sets how many items are requested per HTTP call. Each
command exposes whichever paging parameters its own Yuno operation uses (`page`/`page_size`,
`limit`/`offset`, or `size`), normalised to these same two flags:

```
yuno-cli payment list --merchant-order-id order-42 --limit 20
yuno-cli org account list --page-size 50
```

### Idempotency

`X-Idempotency-Key` is generated for every `POST`/`PATCH`. Pin it with `--idempotency-key <uuid>` so
a retried command cannot create a second payment.

## Confirmation gate

Every non-`GET` operation is confirmed before it is sent:

```
$ yuno-cli payment create --file payment.json
POST /payments
Proceed? [y/N]
```

- Answering anything but `y`/`yes` aborts with exit code `2`.
- `--yes` (or `YUNO_ASSUME_YES=1`) skips the prompt — required for scripts and CI.
- When stdin is not a terminal and `--yes` was not given, the command fails instead of hanging.

## JSON output and piping

`--json` prints the API response verbatim (after masking), which makes `jq` the natural companion:

```
yuno-cli payment get <payment_id> --json | jq '.transactions[].provider_data'
yuno-cli payment list --merchant-order-id order-42 --json | jq -r '.[] | [.id,.status] | @tsv'
yuno-cli routing list --account-id <uuid> --json | jq '.[] | select(.payment_method == "CARD")'
yuno-cli org account list --json | jq -r '.[].id' | while read -r id; do
  yuno-cli routing list --account-id "$id" --json
done
```

Without `--json`, results are rendered as a table of the fields that matter most for that resource.

## Masking

Card numbers, security codes, tokens, cryptograms and API keys are masked everywhere the CLI writes
output — tables, `--json`, and the `--verbose` HTTP dump alike. Pass `--unmask` only when you truly
need the raw value, and never in a shared terminal or a CI log.

`yuno-cli report download <report_id>` prints the pre-signed download link by default. Pass
`--output <path>` (or `--output -` for stdout) to stream the file itself instead; the download
request never sends your Yuno API keys to the storage host, and a file written to disk gets `0600`
permissions since a report carries payment data.

## The `raw` escape hatch

`raw` sends an arbitrary request with the credentials of the selected profile — useful for an
endpoint released by Yuno after this version of the CLI, or for a parameter no flag exposes yet:

```
yuno-cli raw GET /routing --query account_id=<uuid>
yuno-cli raw POST /customers --file body.json
echo '{"email":"a@b.c"}' | yuno-cli raw POST /customers --data @- --yes
yuno-cli raw PATCH /customers/<id> --data '{"email":"a@b.c"}' --idempotency-key <uuid>
```

`--query key=value` and `--header 'Name: value'` are repeatable. The confirmation gate applies to
`raw` exactly as it does to the typed commands.

## Shell completion

```
yuno-cli completion zsh  > "${fpath[1]}/_yuno-cli"
yuno-cli completion bash > /etc/bash_completion.d/yuno-cli
yuno-cli completion fish > ~/.config/fish/completions/yuno-cli.fish
yuno-cli completion powershell | Out-String | Invoke-Expression
```

## Exit codes

| Code | Meaning |
|---|---|
| `0` | success |
| `1` | the API answered with an error |
| `2` | confirmation declined, or stdin is not a terminal |
| `3` | configuration or input error |

## Development

```
make test    # go test -race ./...
make lint    # golangci-lint run
make build   # runs lint first
```

`docs/yuno-openapi.json` is the source of truth for the API; `docs/yuno-operations.txt` is the
coverage checklist enforced by `internal/cmd/coverage_test.go`, which fails if any operation of the
spec has no command mapped to it.
