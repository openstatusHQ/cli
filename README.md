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
# Authenticate with your API token
openstatus login

# List your monitors
openstatus monitors list

# Get detailed info on a monitor (live status + latency percentiles)
openstatus monitors info 123

# Trigger an on-demand check
openstatus monitors trigger 123

# Report an incident
openstatus status-report create --title "API degradation" --status investigating --page-id 1

# Run synthetic tests from config
openstatus run
```

## Commands

| Command | Alias | Description |
|---------|-------|-------------|
| `login` / `logout` | | Authenticate with the OpenStatus API |
| `whoami` | `w` | Show current workspace info |
| `monitors` | `m` | List, inspect, trigger, import, and apply monitors |
| `status-report` | `sr` | Create and manage incident reports |
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

## Monitors as Code

Define monitors declaratively in YAML and sync them with `apply`:

```bash
# Export existing monitors to openstatus.yaml + openstatus.lock
openstatus monitors import

# Preview changes
openstatus monitors apply --dry-run

# Apply changes
openstatus monitors apply
```

## Terraform Export

Generate Terraform HCL for your entire workspace:

```bash
openstatus terraform generate --output-dir ./terraform
```

This creates `provider.tf`, `monitors.tf`, `notifications.tf`, `status_pages.tf`, and `imports.tf` ready for `terraform plan`.

## Authentication

The CLI resolves your API token in this order:

1. `--access-token` / `-t` flag
2. `OPENSTATUS_API_TOKEN` environment variable
3. Saved token at `~/.config/openstatus/token` (written by `openstatus login`)

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
