# OpenStatus CLI - Research

## What It Is

The OpenStatus CLI (`openstatus`) is the official command-line interface for [OpenStatus](https://www.openstatus.dev), a monitoring and status-page SaaS platform. It lets teams manage their monitoring infrastructure from the terminal — monitors, incidents, status pages, maintenance windows, notifications — and export everything to Terraform.

**Go module**: `github.com/openstatusHQ/cli`
**Current version**: `v1.0.3` (hardcoded in `internal/cmd/app.go`)

---

## Project Structure

```
cmd/
  openstatus/main.go          # Binary entry point
  docs/docs.go                # Man page generation tool
internal/
  cmd/app.go                  # Root command assembly — all subcommands registered here
  api/client.go               # HTTP client, base URLs, auth interceptor
  auth/auth.go                # Token resolution, save, remove
  config/
    config.go                 # Run-test config (config.openstatus.yaml)
    lock.go                   # Lock file model (openstatus.lock)
    monitor.go                # Monitor data model + all enums (frequencies, regions, kinds)
    openstatus.go             # Parser for openstatus.yaml
    xdg.go                    # XDG config directory resolution
  cli/
    output.go                 # JSON/quiet/debug mode, PrintJSON, FormatTimestamp
    errors.go                 # ConnectRPC error formatter
    spinner.go                # Terminal spinner wrapper
    confirmation.go           # y/N interactive prompt
    pager.go                  # $PAGER support
  monitors/                   # monitors command + all subcommands
  statusreport/               # status-report command + subcommands
  maintenance/                # maintenance command + subcommands
  statuspage/                 # status-page command (read-only)
  notification/               # notification command (read-only)
  run/                        # run command (synthetic tests)
  login/                      # login + logout commands
  whoami/                     # whoami command
  terraform/                  # terraform generate command
  wizard/                     # Shared interactive form helpers
```

All packages under `internal/` — nothing is importable externally.

---

## Commands

### Global Flags
| Flag | Effect |
|------|--------|
| `--json` | Machine-readable JSON output |
| `--no-color` | Disable colored output |
| `--quiet` / `-q` | Suppress non-error output |
| `--debug` | Log HTTP method + duration to stderr |
| `--access-token` / `-t` | API token override |

### Auth
- **`login`** — Prompts for API token (masked in TTY), verifies via `/v1/whoami`, saves to `~/.config/openstatus/token` (0600, atomic write via temp+rename)
- **`logout`** — Removes saved token file
- **`whoami`** (alias `w`) — Shows workspace name, slug, and plan

### Monitors (alias `m`)
| Subcommand | What it does |
|------------|-------------|
| `list` | List monitors (active by default, `--all` includes inactive) |
| `info <ID>` | Detailed view: config + live per-region status (parallel goroutines) + summary metrics (P50/P75/P90/P95/P99, 1d/7d/14d) |
| `trigger <ID>` | Fire an on-demand check |
| `import` | Export all workspace monitors to `openstatus.yaml` + `openstatus.lock` |
| `apply` | Declarative sync: diff YAML vs lock, create/update/delete. Supports `--dry-run` and `-y` |
| `create` | (Hidden) Create from config without lock diffing |
| `delete <ID>` | (Hidden) Delete a monitor by ID |

### Status Reports (alias `sr`)
| Subcommand | What it does |
|------------|-------------|
| `list` | List reports, filterable by `--status`, `--limit` |
| `info <ID>` | Metadata + full update timeline |
| `create` | Create incident report (interactive wizard in TTY when flags missing) |
| `add-update <ID>` | Append status update to open report |
| `update <ID>` | Update metadata (title or component IDs) |
| `delete <ID>` | Delete report (confirms unless `-y`) |

Statuses: `investigating`, `identified`, `monitoring`, `resolved`

### Maintenance (alias `mt`)
| Subcommand | What it does |
|------------|-------------|
| `list` | List windows, filterable by `--page-id`, `--limit`. Status computed from timestamps |
| `info <ID>` | Full details |
| `create` | Create maintenance window (wizard in TTY) |
| `update <ID>` | Patch title/message/from/to/component-ids |
| `delete <ID>` | Delete with confirmation |

### Status Pages (alias `sp`) — Read-only
| Subcommand | What it does |
|------------|-------------|
| `list` | List all pages (ID, title, URL) |
| `info <ID>` | Full details: access type, theme, locale, grouped components |

### Notifications (alias `n`) — Read-only
| Subcommand | What it does |
|------------|-------------|
| `list` | List channels (ID, name, provider, monitor count) |
| `info <ID>` | Provider-specific config and linked monitor IDs |

12 providers: discord, email, google_chat, grafana_oncall, ntfy, pagerduty, opsgenie, slack, sms, telegram, webhook, whatsapp

### Run (alias `r`)
Reads `config.openstatus.yaml` (list of monitor IDs), runs each in parallel via `POST /v1/monitor/{id}/run`, shows per-region latency + pass/fail. Exits code 1 if any region fails. 2-minute timeout per request.

### Terraform (alias `tf`)
| Subcommand | What it does |
|------------|-------------|
| `generate` (alias `gen`) | Fetches all workspace resources, writes HCL to `--output-dir` |

Generated files:
- `provider.tf` — provider block (`openstatusHQ/openstatus ~> 0.1.0`)
- `monitors.tf` — `openstatus_http_monitor`, `openstatus_tcp_monitor`, `openstatus_dns_monitor`
- `notifications.tf` — `openstatus_notification` with provider-specific nested blocks
- `status_pages.tf` — page + component groups + components
- `imports.tf` — Terraform 1.5+ `import` blocks for all resources

Sensitive values emitted as `"REPLACE_ME"` with `# TODO:` comments.

---

## Architecture

### Framework
Built on **`github.com/urfave/cli/v3`** (v3.0.0-alpha9.2). Root command assembled in `internal/cmd/app.go`, each feature area is a separate package returning a `*cli.Command` tree.

### Execution Flow
```
main() → cmd.NewApp() → individual XxxCmd() funcs
  ↓
Before hook: sets global output flags (JSON/quiet/debug/color)
  ↓
Each command action:
  1. Resolve API token (auth.ResolveAccessToken)
  2. Start spinner (if interactive TTY)
  3. Create ConnectRPC client (or plain HTTP client for REST)
  4. Call API
  5. Stop spinner
  6. Print result (JSON or human-readable table)
```

### Key Design Patterns

**Dual client pattern**: Every service package exposes `NewXxxClient(apiKey)` and `NewXxxClientWithHTTPClient(httpClient, apiKey)` — the latter for test injection.

**Wizard pattern**: When required flags are missing in a TTY, `create` commands fall back to interactive `charmbracelet/huh` forms. Non-TTY or `--json` mode errors with a list of missing flags. Shared helpers in `internal/wizard/`.

**Atomic output flags**: `IsJSONOutput()`, `IsQuiet()`, `IsDebug()` use `sync/atomic.Bool` for goroutine safety.

**Spinner lifecycle**: `StartSpinner()` returns nil if not interactive; all consumers nil-check, so same code path works in scripted contexts.

**Signal handling**: `RunApp()` sets up `signal.NotifyContext` for SIGINT/SIGTERM. Second signal forces `os.Exit(130)`.

---

## API Integration

### Two Transport Layers

**ConnectRPC (Protobuf/JSON over HTTP)** — all structured resource operations:
- Base URL: `https://api.openstatus.dev/rpc`
- Generated from `buf.build/gen/go/openstatus/api` protobuf definitions
- Auth via unary interceptor adding `x-openstatus-key: <token>`
- Uses `connect.WithProtoJSON()` (JSON wire format, not binary protobuf)
- Services: `MonitorService`, `StatusReportService`, `MaintenanceService`, `StatusPageService`, `NotificationService`

**Plain REST (JSON over HTTP)** — two endpoints:
- `GET https://api.openstatus.dev/v1/whoami`
- `POST https://api.openstatus.dev/v1/monitor/{id}/run` (2-min timeout)

### Auth Resolution Order
1. `--access-token` / `-t` flag
2. `OPENSTATUS_API_TOKEN` env var
3. Token file at `$XDG_CONFIG_HOME/openstatus/token` (default `~/.config/openstatus/token`)

---

## Data Models

### Monitor (config representation — `internal/config/monitor.go`)
```
Monitor { Name, Description, Frequency, Regions[], Active, Kind (http|tcp),
          Retry, Public, Request{URL, Method, Headers, Body, FollowRedirects, Host, Port},
          DegradedAfter, Timeout, Assertions[], OpenTelemetry{Endpoint, Headers} }
```
- Frequencies: 30s, 1m, 5m, 10m, 30m, 1h
- Kinds: http, tcp
- 35+ Fly.io regions, 6 Koyeb, 4 Railway

### Lock (`internal/config/lock.go`)
```
Lock { ID int, Monitor Monitor }
MonitorsLock map[string]Lock   // key = logical name in YAML
```

### Status Report
States: investigating → identified → monitoring → resolved. Timestamped update timeline.

### Maintenance
Title, message, from/to (RFC 3339), page ID, component IDs. Status computed from timestamps: scheduled, in_progress, completed.

### Notification
12 provider types, provider-specific data blob, linked monitor IDs.

### Status Page
Slug, custom domain, access type (public/password/authenticated), theme (system/light/dark), locale, components + component groups.

---

## Configuration Files

| File | Purpose |
|------|---------|
| `~/.config/openstatus/token` | Saved API token (0600) |
| `openstatus.yaml` | Declarative monitor definitions |
| `openstatus.lock` | Lock file mapping logical names to API IDs |
| `config.openstatus.yaml` | Run-test config (list of monitor IDs) |
| `.env` | Auto-loaded via `godotenv` on startup |

Env vars: `OPENSTATUS_API_TOKEN`, `NO_COLOR`, `TERM=dumb`, `XDG_CONFIG_HOME`, `PAGER`

---

## Monitors-as-Code (`apply` Deep Dive)

The `monitors apply` command in `internal/monitors/monitor_apply.go` implements full declarative sync:

1. Read `openstatus.yaml` and `openstatus.lock`
2. `countChanges()` diffs without API calls (using `google/go-cmp`)
3. Print plan summary (N to create, N to update, N to delete)
4. If `--dry-run`, stop
5. Otherwise prompt for confirmation (unless `-y`)
6. `ApplyChanges()` creates/updates/deletes as needed
7. Write updated lock file (O_TRUNC + fsync)

Diff key = logical name in YAML map. Renaming a monitor in YAML = create new + delete old.

---

## Testing

Standard `go test`, no external framework. External test packages (`package xxx_test`) for black-box testing.

Every feature package has thorough test coverage. Tests use the dual-client pattern — injecting mock HTTP clients via `NewXxxClientWithHTTPClient`. Terraform tests construct protobuf messages directly and assert on HCL string output. CI runs with `-race -count=1`.

---

## Build & Release

### GoReleaser (`.goreleaser.yaml`)
- Targets: linux/amd64, darwin/amd64+arm64, windows/amd64
- CGO disabled (static binaries)
- Archives: `.tar.gz` (`.zip` for Windows), includes man page
- Homebrew cask auto-published to `openstatusHQ/homebrew-cli`

### CI
- **Test**: `.github/workflows/test.yml` — on push to master + all PRs, `go test -timeout 30s -race -count=1 ./...`
- **Release**: `.github/workflows/release.yml` — on tag push, `goreleaser release --clean`

### Distribution
- Homebrew: `brew install openstatusHQ/cli/openstatus --cask`
- Shell installer: `curl -fsSL .../install.sh | bash`
- PowerShell: `iwr .../install.ps1 | iex`
- GitHub Releases: direct downloads

### Docs Generation
`cmd/docs/docs.go` uses `urfave/cli-docs/v3` to generate markdown, then `pandoc -s -t man` for roff.

---

## Key Dependencies

| Dependency | Purpose |
|------------|---------|
| `urfave/cli/v3` | CLI framework |
| `connectrpc.com/connect` | ConnectRPC client |
| `buf.build/gen/go/openstatus/api` | Generated protobuf types |
| `charmbracelet/huh` | Interactive forms (wizards) |
| `charmbracelet/lipgloss` | Styled terminal output |
| `hashicorp/hcl/v2/hclwrite` | Programmatic HCL generation |
| `knadh/koanf` | YAML config parsing |
| `google/go-cmp` | Deep comparison for diffs |
| `joho/godotenv` | .env file loading |
