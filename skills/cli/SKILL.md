---
name: openstatus-cli
description: |
  OpenStatus CLI for managing uptime monitors, incident reports, status pages, maintenance windows, and synthetic tests. Use this skill whenever the user wants to monitor a website or API, set up uptime checks, create or manage monitors, report an incident, update a status page, schedule maintenance, run synthetic tests, check latency or availability, define monitors as code, generate Terraform configuration, export to Terraform, or use the openstatus command. Also trigger when the user says "is my site up", "check my endpoint", "create a status report", "monitor this URL", "run uptime tests", "set up monitoring", "our API is down", "schedule maintenance", "maintenance window", "planned downtime", "terraform", "generate terraform", "export to terraform", "infrastructure as code", or mentions openstatus in any context. This skill knows the full CLI — commands, flags, config format, and workflows — so Claude can act without guessing.
allowed-tools:
  - Bash(openstatus *)
---

# OpenStatus CLI

Manage uptime monitors, incident reports, status pages, and maintenance windows from the terminal. The CLI supports monitors-as-code via YAML config files.

Run `openstatus --help` or `openstatus <command> --help` for full option details.

## Prerequisites

Must be authenticated. Verify with:

```bash
openstatus whoami
```

This shows your workspace name, slug, and plan. If not authenticated, run `openstatus login` and paste your API token from the OpenStatus dashboard.

Token resolution order:
1. `--access-token` / `-t` flag
2. `OPENSTATUS_API_TOKEN` environment variable
3. Saved token at `~/.config/openstatus/token`

## Command Overview

| Task | Command | When to use |
|------|---------|-------------|
| Sync monitors from config | `monitors apply` | You have an `openstatus.yaml` and want to create/update/delete monitors |
| List all monitors | `monitors list` | See what monitors exist in the workspace |
| Get monitor details + metrics | `monitors info <ID>` | Check latency, status, and config for a specific monitor |
| Trigger a monitor now | `monitors trigger <ID>` | Run an on-demand check across all regions |
| Delete a monitor | `monitors delete <ID>` | Remove a monitor |
| Export monitors to YAML | `monitors import` | Pull existing monitors into an `openstatus.yaml` + lock file |
| Create incident report | `status-report create` | Something is broken, notify users |
| Add update to incident | `status-report add-update <ID>` | Post a progress update on an ongoing incident |
| List incidents | `status-report list` | See active/recent incidents |
| Get incident details | `status-report info <ID>` | View full incident timeline |
| Update incident metadata | `status-report update <ID>` | Change title or components |
| Delete incident | `status-report delete <ID>` | Remove a status report |
| List status pages | `status-page list` | See all your status pages |
| Get status page details | `status-page info <ID>` | View page config, components, theme |
| Create a maintenance window | `maintenance create` | Plan a maintenance window for a status page |
| List maintenance windows | `maintenance list` | See scheduled/active/completed maintenance |
| Get maintenance details | `maintenance info <ID>` | View full details of a maintenance window |
| Update a maintenance window | `maintenance update <ID>` | Change title, message, or time window |
| Delete a maintenance window | `maintenance delete <ID>` | Remove a maintenance window |
| Run synthetic tests | `run` | Execute on-demand tests for specific monitors |
| Generate Terraform config | `terraform generate` | Export workspace resources to Terraform HCL files |
| Check workspace | `whoami` | Verify auth and workspace info |

Command aliases: `monitors` = `m`, `status-report` = `sr`, `status-page` = `sp`, `maintenance` = `mt`, `terraform` = `tf`, `run` = `r`, `whoami` = `w`.

## Workflows

### Setting up monitors (monitors-as-code)

This is the primary way to manage monitors. Write a YAML config, then let the CLI sync it.

**Starting from scratch:**
1. Create an `openstatus.yaml` file — see [references/monitor-config.md](references/monitor-config.md) for the full schema
2. Preview changes: `openstatus monitors apply --dry-run`
3. Apply: `openstatus monitors apply`
4. The CLI creates a `openstatus.lock` file to track state — commit this alongside your config

**Starting from existing monitors:**
1. Export: `openstatus monitors import` (creates `openstatus.yaml` + `openstatus.lock`)
2. Edit the YAML as needed
3. Apply changes: `openstatus monitors apply`

**The apply workflow** compares your `openstatus.yaml` against the lock file and the API, then creates, updates, or deletes monitors to match. Use `--dry-run` to preview, `-y` to skip the confirmation prompt.

### Incident lifecycle

Use status reports for **unplanned** outages and incidents.

Status reports follow a progression: `investigating` -> `identified` -> `monitoring` -> `resolved`. These are the only valid status values — the CLI rejects anything else.

**1. Find your status page ID and component IDs first:**
```bash
openstatus status-page list
openstatus status-page info <PAGE_ID>   # shows components grouped by section
```

**2. Create the incident:**
```bash
openstatus status-report create \
  --title "API Degradation" \
  --status investigating \
  --message "We are investigating increased error rates on the API" \
  --page-id 123 \
  --component-ids "comp-1,comp-2" \
  --notify
```

On success, the CLI prints the report ID and suggests the next command:
```
Status report created successfully (ID: 456)
To add updates, run: openstatus status-report add-update 456 --status identified --message '...'
```

**`create` flags:**

| Flag | Required | Description |
|------|----------|-------------|
| `--title` | yes | Incident title |
| `--status` | yes | `investigating`, `identified`, `monitoring`, or `resolved` |
| `--message` | yes | Initial message describing the incident |
| `--page-id` | yes | Status page ID (get it from `status-page list`) |
| `--component-ids` | no | Comma-separated component IDs in a single string: `"id1,id2"` |
| `--notify` | no | Send notification to status page subscribers |
| `--date` | no | RFC 3339 timestamp (e.g. `2026-03-25T10:00:00Z`), defaults to now (UTC) |

**3. Post updates as you learn more:**
```bash
openstatus status-report add-update 456 \
  --status identified \
  --message "Root cause identified: database connection pool exhaustion" \
  --notify
```

**`add-update` flags:**

| Flag | Required | Description |
|------|----------|-------------|
| `--status` | yes | New status value |
| `--message` | yes | Update message |
| `--notify` | no | Notify subscribers |
| `--date` | no | RFC 3339 timestamp, defaults to now (UTC) |

**4. Resolve:**
```bash
openstatus status-report add-update 456 \
  --status resolved \
  --message "Connection pool limits increased, monitoring confirms recovery" \
  --notify
```

When status is set to `resolved`, the CLI confirms: `Report resolved.`

**5. Update metadata (title or components) without changing status:**
```bash
openstatus status-report update 456 \
  --title "API Degradation - Resolved" \
  --component-ids "comp-1,comp-3"
```

At least one of `--title` or `--component-ids` must be provided. Note: `--component-ids` **replaces** the entire list — it's not additive.

**6. Track incident progress:**
```bash
openstatus status-report info 456
```

Shows full metadata plus the **update timeline** — each update displayed as `<date> [status] <message>`.

**7. List and filter incidents:**
```bash
openstatus status-report list                          # all reports
openstatus status-report list --status investigating   # only active investigations
openstatus status-report list --limit 10               # last 10 reports
```

**8. Delete an incident:**
```bash
openstatus status-report delete 456        # prompts for confirmation
openstatus status-report delete 456 -y     # skip confirmation
```

### Scheduling maintenance

Use maintenance for **planned** downtime windows.

**1. Find your status page ID and component IDs first:**
```bash
openstatus status-page list
openstatus status-page info <PAGE_ID>   # shows components grouped by section
```

**2. Create a maintenance window:**
```bash
openstatus maintenance create \
  --title "Database Migration" \
  --message "Scheduled database migration to improve performance" \
  --from "2026-04-05T02:00:00Z" \
  --to "2026-04-05T04:00:00Z" \
  --page-id 123 \
  --component-ids "comp-1,comp-2" \
  --notify
```

On success, the CLI prints the maintenance ID and suggests the next command:
```
Maintenance created successfully (ID: 789)
Run 'openstatus maintenance info 789' to see details
```

**`create` flags:**

| Flag | Required | Description |
|------|----------|-------------|
| `--title` | yes | Maintenance title |
| `--message` | yes | Description of the maintenance |
| `--from` | yes | Start time in RFC 3339 format (e.g. `2026-04-05T02:00:00Z`) |
| `--to` | yes | End time in RFC 3339 format |
| `--page-id` | yes | Status page ID (get it from `status-page list`) |
| `--component-ids` | no | Comma-separated component IDs in a single string: `"id1,id2"` |
| `--notify` | no | Notify status page subscribers |

Status is computed automatically: `scheduled` (before `--from`), `in_progress` (between `--from` and `--to`), `completed` (after `--to`). There is no `--status` flag.

**3. Update a maintenance window:**
```bash
openstatus maintenance update <ID> \
  --title "Extended Maintenance" \
  --to "2026-04-05T06:00:00Z"
```

Only provided flags are updated. At least one of `--title`, `--message`, `--from`, `--to`, or `--component-ids` must be set. `--component-ids` replaces the entire list.

**4. List and filter:**
```bash
openstatus maintenance list                        # all maintenance windows
openstatus maintenance list --page-id 123          # filter by page
openstatus maintenance list --limit 10             # limit results
```

**5. View details:**
```bash
openstatus maintenance info <ID>
```

**6. Delete:**
```bash
openstatus maintenance delete <ID>       # prompts for confirmation
openstatus maintenance delete <ID> -y    # skip confirmation
```

### On-demand testing

Run specific monitors immediately across all their configured regions.

**Trigger a single monitor:**
```bash
openstatus monitors trigger <MONITOR_ID>
```

**Run a batch of monitors via config:**
1. Create `config.openstatus.yaml`:
   ```yaml
   tests:
     ids:
       - monitor-id-1
       - monitor-id-2
   ```
2. Run: `openstatus run`

Both approaches run tests in parallel and show latency + status per region.

### Inspecting state

**Monitor details with metrics:**
```bash
openstatus monitors info <ID>
```
Shows config, live status per region, and summary metrics (P50/P75/P95/P99 latency). Defaults to last 24h.

**List monitors (including inactive):**
```bash
openstatus monitors list --all
```

**Incident timeline:**
```bash
openstatus status-report info <ID>
```

**Maintenance details:**
```bash
openstatus maintenance info <ID>
```

**Status page components and config:**
```bash
openstatus status-page info <ID>
```

### Terraform export

Generate Terraform HCL configuration from all workspace resources. This creates ready-to-use `.tf` files with import blocks for adopting Terraform on an existing OpenStatus setup.

```bash
openstatus terraform generate
```

This creates an `openstatus-terraform/` directory with:
- `provider.tf` — Terraform provider configuration
- `monitors.tf` — all HTTP, TCP, and DNS monitors
- `notifications.tf` — all notification channels with provider-specific blocks
- `status_pages.tf` — status pages, components, and component groups
- `imports.tf` — Terraform 1.5+ import blocks for all resources

**Custom output directory:**
```bash
openstatus terraform generate --output-dir ./infra/openstatus/
```

**After generating:**
```bash
cd openstatus-terraform
terraform init
terraform plan
```

Sensitive values (passwords, API keys) are emitted as `"REPLACE_ME"` with a TODO comment — update them before running `terraform apply`.

## Global Flags

Every command supports these:

| Flag | Effect |
|------|--------|
| `--json` | Machine-readable JSON output |
| `--no-color` | Disable colored output |
| `--quiet` / `-q` | Suppress non-error output |
| `--debug` | Enable debug output |

Use `--json` when you need to parse output programmatically or pipe it to `jq`.

## Best Practices

- **Use `apply`, not `create`** — `monitors apply` is the declarative, idempotent way to manage monitors. `monitors create` exists but `apply` handles creates, updates, and deletes in one command.
- **Always `--dry-run` first** — preview what `apply` will change before committing.
- **Get the page ID before creating reports** — `status-report create` requires `--page-id`. Run `status-page list` first. Then use `status-page info <ID>` to find component IDs if you need `--component-ids`.
- **`--component-ids` is a single comma-separated string** — use `"id1,id2,id3"`, not multiple flags. On `update`, it replaces the full list.
- **Status values are strict** — only `investigating`, `identified`, `monitoring`, `resolved`. The CLI rejects anything else.
- **Use `--notify` deliberately** — it emails all subscribers. Useful for `create` and `resolved`, but you may want to skip it for intermediate updates.
- **Commit your lock file** — `openstatus.lock` tracks the mapping between your YAML and the API. Without it, `apply` can't diff properly.
- **Use `-y` in scripts** — skip interactive confirmations with `--auto-accept` / `-y` for CI/CD pipelines.
