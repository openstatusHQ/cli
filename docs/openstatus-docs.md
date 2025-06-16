# OpenStatus CLI

# NAME

openstatus - This is OpenStatus Command Line Interface, the OpenStatus.dev CLI

# SYNOPSIS

openstatus

# DESCRIPTION

OpenStatus is a command line interface for managing your monitors and triggering your synthetics tests.

Please report any issues at https://github.com/openstatusHQ/cli/issues/new

**Usage**:

```
openstatus [GLOBAL OPTIONS] [command [COMMAND OPTIONS]] [ARGUMENTS...]
```

# COMMANDS

## monitors

Manage your monitors

### create

Create monitors (beta)

>openstatus monitors create [options]

**--access-token, -t**="": OpenStatus API Access Token

**--auto-accept, -y**: Automatically accept the prompt

**--config**="": The configuration file containing monitor information (default: openstatus.yaml)

### delete

Delete a monitor

>openstatus monitors delete [MonitorID] [options]

**--access-token, -t**="": OpenStatus API Access Token

**--auto-accept, -y**: Automatically accept the prompt

### export

Export all your monitors

>openstatus monitor export [options]

**--access-token, -t**="": OpenStatus API Access Token

**--output, -o**="": The output file name  (default: openstatus.yaml)

### info

Get a monitor information

>openstatus monitor info [MonitorID]

**--access-token, -t**="": OpenStatus API Access Token

### list

List all monitors

>openstatus monitors list [options]

**--access-token, -t**="": OpenStatus API Access Token

**--all**: List all monitors including inactive ones

### trigger

Trigger a monitor execution

>openstatus monitors trigger [MonitorId] [options]

**--access-token, -t**="": OpenStatus API Access Token

## run, r

Run your synthetics tests

>openstatus run [options]

**--access-token, -t**="": OpenStatus API Access Token

**--config**="": The configuration file (default: config.openstatus.yaml)

## whoami, w

Get your workspace information

>openstatus whoami [options]

**--access-token, -t**="": OpenStatus API Access Token
