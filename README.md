# OpenStatus CLI

Manage status pages, monitors, and incidents from the terminal.

## Installation

### Homebrew

```bash
brew install openstatusHQ/cli/openstatus --cask
```

### macOS / Linux

```bash
curl -fsSL https://raw.githubusercontent.com/openstatusHQ/cli/refs/heads/main/install.sh | bash
```

### Windows

```powershell
iwr https://raw.githubusercontent.com/openstatusHQ/cli/refs/heads/main/install.ps1 | iex
```

## Quick Start

```bash
# Run a one-shot HTTP check against a URL from 28 global regions (no auth)
openstatus check https://openstat.us

# Log in with your browser
openstatus login

# List your monitors
openstatus monitors list

# Get detailed info on a monitor (live status + latency percentiles)
openstatus monitors info 123

# Trigger an on-demand check
openstatus monitors trigger 123

# Declare an incident, then publish a linked public status report
ID=$(openstatus incident declare --title "Checkout API returns 502" --severity major -q)
openstatus status-report create --title "Checkout issues" --status investigating \
  --message "We are investigating failed payments" --page-id 1 --incident-id "$ID"

# Run the incident: notes, status changes, postmortem
openstatus inc note "$ID" --message "Rolled back the deploy"
openstatus inc resolve "$ID" --note "Error rate back to normal"
openstatus inc postmortem edit "$ID" --file postmortem.md
openstatus inc postmortem approve "$ID" --close -y

# Post a public status update
openstatus status-report create --title "API degradation" --status investigating --page-id 1

# Report an incident with per-component impact
openstatus sr create --title "API down" --status investigating --message "Investigating" \
  --page-id 1 --impact comp_api=major_outage --impact comp_db=degraded

# Update an incident and change impact
openstatus sr add-update sr_42 --status monitoring --message "Recovering" \
  --impact comp_api=degraded,comp_db=operational

# Run synthetic tests from config
openstatus run
```

## Global Speed Check

`openstatus check <URL>` runs a one-shot HTTP check from 28 global probe
regions via the public OpenStatus speed checker. No API token required.

```bash
openstatus check https://openstat.us
openstatus check https://openstat.us -X POST -H 'Authorization: Bearer …' -d '{"ping":true}'
openstatus check https://openstat.us -d @payload.json
openstatus check https://openstat.us --timing            # adds DNS/Connection/TLS/TTFB/Transfer columns
openstatus check https://openstat.us --json | jq '.summary'
```

Rate limit: 3 requests per 60 seconds.

## Commands

| Command | Alias | Description |
|---------|-------|-------------|
| `check` | `c` | Run an HTTP check against a URL from 28 global regions (no auth) |
| `login` / `logout` | | Authenticate with the OpenStatus API |
| `whoami` | `w` | Show current workspace info |
| `monitors` | `m` | List, inspect, trigger, and import monitors |
| `incident` | `inc`, `incidents` | Declare and run incidents (lifecycle, notes, status report link, postmortem) |
| `status-report` | `sr` | Post and manage public status reports |
| `maintenance` | `mt` | Schedule and manage maintenance windows |
| `status-page` | `sp` | View status pages and components |
| `notification` | `n` | View notification channels |
| `run` | `r` | Run synthetic tests across global regions |
| `terraform generate` | `tf gen` | Export workspace resources to Terraform HCL |

### Global Flags

| Flag | Description |
|------|-------------|
| `--json` | Machine-readable JSON output |
| `--quiet`, `-q` | Suppress non-error output |
| `--debug` | Print HTTP request details to stderr |
| `--no-color` | Disable colored output |
| `--access-token`, `-t` | Override API token |

### Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | General error, including failed checks in `openstatus run` |
| `2` | Usage error: unknown command or flag, missing argument, invalid value or config |
| `3` | Authentication: missing, invalid or insufficient API token |
| `4` | Resource not found |
| `5` | API unreachable, timed out or rate limited |
| `130` | Interrupted (Ctrl-C) |

With `--json`, errors are written to stderr as JSON:

```json
{
  "error": {
    "code": "not_found",
    "message": "monitor 7 not found. Run 'openstatus monitors list' to see available monitors",
    "exit_code": 4
  }
}
```

`code` is one of `error`, `usage`, `unauthenticated`, `permission_denied`, `not_found`, `invalid_argument`, `failed_precondition`, `rate_limited`, `unavailable`, `timeout`, `cancelled` or `checks_failed`.

## Terraform Export

Generate Terraform HCL for your entire workspace:

```bash
openstatus terraform generate --output-dir ./terraform
```

This creates `provider.tf`, `monitors.tf`, `notifications.tf`, `status_pages.tf`, `private_locations.tf`, and `imports.tf` ready for `terraform plan`.

## Authentication

The CLI resolves your API token in this order:

1. `--access-token` / `-t` flag
2. `OPENSTATUS_API_TOKEN` environment variable
3. Saved login at `~/.config/openstatus/oauth.json` (written by `openstatus login`, refreshed automatically)
4. Saved API token at `~/.config/openstatus/token` (written by `openstatus login --with-token`)

`openstatus login` opens the dashboard in your browser, where you sign in and choose a workspace, then returns you to the CLI. Revoke a login with `openstatus logout` or from **Settings / Integrations** in the dashboard.

Where a browser isn't available, such as CI or an SSH session, use an API token from the dashboard:

```bash
openstatus login --with-token          # prompts for the token
printf "$TOKEN" | openstatus login     # reads it from stdin
```

## Self-Hosted

Point the CLI at your own API server with `OPENSTATUS_API_URL` (default `https://api.openstatus.dev`):

```bash
export OPENSTATUS_API_URL=https://api.openstatus.example.com
openstatus whoami
```

`openstatus terraform generate` then writes the matching `base_url` into `provider.tf`. The `check` command always uses the public speed checker.

## Usage Data

Requests to the API carry a `User-Agent` with the CLI version and platform, plus the command being run (`x-openstatus-cli-command`, e.g. `monitors list`) and a random per-run ID (`x-openstatus-cli-invocation`). We use them to understand which commands are used. No arguments, flag values or file contents are sent.

To stop sending the command and run ID, set either variable (`0`, `false`, `no` and `off` leave it enabled):

```bash
export DO_NOT_TRACK=1
# or
export OPENSTATUS_NO_TELEMETRY=1
```

Opting out does not remove the `User-Agent`: like any HTTP client, the CLI still identifies itself with its version and platform (e.g. `openstatus-cli/v1.6.0 (darwin; arm64)`).

When `OPENSTATUS_API_URL` points at a self-hosted instance, the command and run ID only go to that instance; `check` does not send them to openstatus.dev.

## Development

### Run Tests

```bash
go test -race ./...
```

### Generate Documentation

```bash
go run cmd/docs/docs.go
cd docs
pandoc -s -t man openstatus-docs.md -o openstatus.1
```
