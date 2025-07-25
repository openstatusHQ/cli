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

### `run` command (aliases: `r`)

Run your synthetics tests.

> openstatus run [options]

Run the synthetic tests defined in the config.openstatus.yaml.

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
