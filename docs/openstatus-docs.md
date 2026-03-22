# OpenStatus CLI

## CLI interface - openstatus

OpenStatus is a command line interface for managing your monitors and triggering your synthetics tests.   Please report any issues at https://github.com/openstatusHQ/cli/issues/new.

This is OpenStatus Command Line Interface, the OpenStatus.dev CLI.

Usage:

```bash
$ openstatus [COMMAND] [COMMAND FLAGS] [ARGUMENTS...]
```

### `monitors` command

Manage your monitors.

Usage:

```bash
$ openstatus [GLOBAL FLAGS] monitors [ARGUMENTS...]
```

### `monitors apply` subcommand

Create or update monitors.

> openstatus monitors apply [options]

Creates or updates monitors according to the OpenStatus configuration file.

Usage:

```bash
$ openstatus [GLOBAL FLAGS] monitors apply [COMMAND FLAGS] [ARGUMENTS...]
```

The following flags are supported:

| Name                        | Description                                           |   Default value   |  Environment variables |
|-----------------------------|-------------------------------------------------------|:-----------------:|:----------------------:|
| `--config="…"` (`-c`)       | The configuration file containing monitor information | `openstatus.yaml` |         *none*         |
| `--access-token="…"` (`-t`) | OpenStatus API Access Token                           |                   | `OPENSTATUS_API_TOKEN` |
| `--auto-accept` (`-y`)      | Automatically accept the prompt                       |      `false`      |         *none*         |

### `monitors create` subcommand

Create monitors (beta).

> openstatus monitors create [options]

Create the monitors defined in the openstatus.yaml file.

Usage:

```bash
$ openstatus [GLOBAL FLAGS] monitors create [COMMAND FLAGS] [ARGUMENTS...]
```

The following flags are supported:

| Name                        | Description                                           |   Default value   |  Environment variables |
|-----------------------------|-------------------------------------------------------|:-----------------:|:----------------------:|
| `--config="…"` (`-c`)       | The configuration file containing monitor information | `openstatus.yaml` |         *none*         |
| `--access-token="…"` (`-t`) | OpenStatus API Access Token                           |                   | `OPENSTATUS_API_TOKEN` |
| `--auto-accept` (`-y`)      | Automatically accept the prompt                       |      `false`      |         *none*         |

### `monitors delete` subcommand

Delete a monitor.

> openstatus monitors delete [MonitorID] [options]

Usage:

```bash
$ openstatus [GLOBAL FLAGS] monitors delete [COMMAND FLAGS] [ARGUMENTS...]
```

The following flags are supported:

| Name                        | Description                     | Default value |  Environment variables |
|-----------------------------|---------------------------------|:-------------:|:----------------------:|
| `--access-token="…"` (`-t`) | OpenStatus API Access Token     |               | `OPENSTATUS_API_TOKEN` |
| `--auto-accept` (`-y`)      | Automatically accept the prompt |    `false`    |         *none*         |

### `monitors import` subcommand

Import all your monitors.

> openstatus monitors import [options]

Import all your monitors from your workspace to a YAML file; it will also create a lock file to manage your monitors with 'apply'.

Usage:

```bash
$ openstatus [GLOBAL FLAGS] monitors import [COMMAND FLAGS] [ARGUMENTS...]
```

The following flags are supported:

| Name                        | Description                 |   Default value   |  Environment variables |
|-----------------------------|-----------------------------|:-----------------:|:----------------------:|
| `--access-token="…"` (`-t`) | OpenStatus API Access Token |                   | `OPENSTATUS_API_TOKEN` |
| `--output="…"` (`-o`)       | The output file name        | `openstatus.yaml` |         *none*         |

### `monitors info` subcommand

Get a monitor information.

> openstatus monitors info [MonitorID]

Fetch the monitor information. The monitor information includes details such as name, description, endpoint, method, frequency, locations, active status, public status, timeout, degraded after, and body. The body is truncated to 40 characters.

Usage:

```bash
$ openstatus [GLOBAL FLAGS] monitors info [COMMAND FLAGS] [ARGUMENTS...]
```

The following flags are supported:

| Name                        | Description                 | Default value |  Environment variables |
|-----------------------------|-----------------------------|:-------------:|:----------------------:|
| `--access-token="…"` (`-t`) | OpenStatus API Access Token |               | `OPENSTATUS_API_TOKEN` |

### `monitors list` subcommand

List all monitors.

> openstatus monitors list [options]

List all monitors. The list shows all your monitors attached to your workspace. It displays the ID, name, and URL of each monitor.

Usage:

```bash
$ openstatus [GLOBAL FLAGS] monitors list [COMMAND FLAGS] [ARGUMENTS...]
```

The following flags are supported:

| Name                        | Description                               | Default value |  Environment variables |
|-----------------------------|-------------------------------------------|:-------------:|:----------------------:|
| `--all`                     | List all monitors including inactive ones |    `false`    |         *none*         |
| `--access-token="…"` (`-t`) | OpenStatus API Access Token               |               | `OPENSTATUS_API_TOKEN` |

### `monitors trigger` subcommand

Trigger a monitor execution.

> openstatus monitors trigger [MonitorId] [options]

Trigger a monitor execution on demand. This command allows you to launch your tests on demand.

Usage:

```bash
$ openstatus [GLOBAL FLAGS] monitors trigger [COMMAND FLAGS] [ARGUMENTS...]
```

The following flags are supported:

| Name                        | Description                 | Default value |  Environment variables |
|-----------------------------|-----------------------------|:-------------:|:----------------------:|
| `--access-token="…"` (`-t`) | OpenStatus API Access Token |               | `OPENSTATUS_API_TOKEN` |

### `status-report` command (aliases: `sr`)

Manage status reports.

Usage:

```bash
$ openstatus [GLOBAL FLAGS] status-report [ARGUMENTS...]
```

### `status-report list` subcommand

List all status reports.

> openstatus status-report list [options]

Usage:

```bash
$ openstatus [GLOBAL FLAGS] status-report list [COMMAND FLAGS] [ARGUMENTS...]
```

The following flags are supported:

| Name                        | Description                                                        | Default value |  Environment variables |
|-----------------------------|--------------------------------------------------------------------|:-------------:|:----------------------:|
| `--access-token="…"` (`-t`) | OpenStatus API Access Token                                        |               | `OPENSTATUS_API_TOKEN` |
| `--status="…"`              | Filter by status (investigating, identified, monitoring, resolved) |               |         *none*         |
| `--limit="…"`               | Maximum number of reports to return (1-100)                        |      `0`      |         *none*         |

### `status-report info` subcommand

Get status report details.

> openstatus status-report info <ReportID>

Usage:

```bash
$ openstatus [GLOBAL FLAGS] status-report info [COMMAND FLAGS] [ARGUMENTS...]
```

The following flags are supported:

| Name                        | Description                 | Default value |  Environment variables |
|-----------------------------|-----------------------------|:-------------:|:----------------------:|
| `--access-token="…"` (`-t`) | OpenStatus API Access Token |               | `OPENSTATUS_API_TOKEN` |

### `status-report create` subcommand

Create a status report.

> openstatus status-report create --title "API Degradation" --status investigating --message "Investigating increased latency" --page-id 123

Usage:

```bash
$ openstatus [GLOBAL FLAGS] status-report create [COMMAND FLAGS] [ARGUMENTS...]
```

The following flags are supported:

| Name                        | Description                                                      | Default value |  Environment variables |
|-----------------------------|------------------------------------------------------------------|:-------------:|:----------------------:|
| `--access-token="…"` (`-t`) | OpenStatus API Access Token                                      |               | `OPENSTATUS_API_TOKEN` |
| `--title="…"`               | Title of the status report                                       |               |         *none*         |
| `--status="…"`              | Initial status (investigating, identified, monitoring, resolved) |               |         *none*         |
| `--message="…"`             | Initial message describing the incident                          |               |         *none*         |
| `--page-id="…"`             | Status page ID to associate with this report                     |               |         *none*         |
| `--component-ids="…"`       | Comma-separated page component IDs                               |               |         *none*         |
| `--notify`                  | Notify subscribers about this status report                      |    `false`    |         *none*         |
| `--date="…"`                | Date when the event occurred (RFC 3339 format, defaults to now)  |               |         *none*         |

### `status-report update` subcommand

Update status report metadata.

> openstatus status-report update <ReportID> [--title "New title"] [--component-ids id1,id2]

Usage:

```bash
$ openstatus [GLOBAL FLAGS] status-report update [COMMAND FLAGS] [ARGUMENTS...]
```

The following flags are supported:

| Name                        | Description                                                 | Default value |  Environment variables |
|-----------------------------|-------------------------------------------------------------|:-------------:|:----------------------:|
| `--access-token="…"` (`-t`) | OpenStatus API Access Token                                 |               | `OPENSTATUS_API_TOKEN` |
| `--title="…"`               | New title for the report                                    |               |         *none*         |
| `--component-ids="…"`       | Comma-separated page component IDs (replaces existing list) |               |         *none*         |

### `status-report delete` subcommand

Delete a status report.

> openstatus status-report delete <ReportID>

Usage:

```bash
$ openstatus [GLOBAL FLAGS] status-report delete [COMMAND FLAGS] [ARGUMENTS...]
```

The following flags are supported:

| Name                        | Description                     | Default value |  Environment variables |
|-----------------------------|---------------------------------|:-------------:|:----------------------:|
| `--access-token="…"` (`-t`) | OpenStatus API Access Token     |               | `OPENSTATUS_API_TOKEN` |
| `--auto-accept` (`-y`)      | Automatically accept the prompt |    `false`    |         *none*         |

### `status-report add-update` subcommand

Add an update to a status report.

> openstatus status-report add-update <ReportID> --status resolved --message "Issue has been resolved"

Usage:

```bash
$ openstatus [GLOBAL FLAGS] status-report add-update [COMMAND FLAGS] [ARGUMENTS...]
```

The following flags are supported:

| Name                        | Description                                                  | Default value |  Environment variables |
|-----------------------------|--------------------------------------------------------------|:-------------:|:----------------------:|
| `--access-token="…"` (`-t`) | OpenStatus API Access Token                                  |               | `OPENSTATUS_API_TOKEN` |
| `--status="…"`              | New status (investigating, identified, monitoring, resolved) |               |         *none*         |
| `--message="…"`             | Message describing what changed                              |               |         *none*         |
| `--date="…"`                | Date for the update (RFC 3339 format, defaults to now)       |               |         *none*         |
| `--notify`                  | Notify subscribers about this update                         |    `false`    |         *none*         |

### `run` command (aliases: `r`)

Run your synthetics tests.

> openstatus run [options]

Run the synthetic tests defined in the config.openstatus.yaml. The config file should be in the following format:  tests:   ids:      - monitor-id-1      - monitor-id-2.

Usage:

```bash
$ openstatus [GLOBAL FLAGS] run [COMMAND FLAGS] [ARGUMENTS...]
```

The following flags are supported:

| Name                        | Description                 |      Default value       |  Environment variables |
|-----------------------------|-----------------------------|:------------------------:|:----------------------:|
| `--config="…"`              | The configuration file      | `config.openstatus.yaml` |         *none*         |
| `--access-token="…"` (`-t`) | OpenStatus API Access Token |                          | `OPENSTATUS_API_TOKEN` |

### `whoami` command (aliases: `w`)

Get your workspace information.

> openstatus whoami [options]

Get your current workspace information, display the workspace name, slug, and plan.

Usage:

```bash
$ openstatus [GLOBAL FLAGS] whoami [COMMAND FLAGS] [ARGUMENTS...]
```

The following flags are supported:

| Name                        | Description                 | Default value |  Environment variables |
|-----------------------------|-----------------------------|:-------------:|:----------------------:|
| `--access-token="…"` (`-t`) | OpenStatus API Access Token |               | `OPENSTATUS_API_TOKEN` |
