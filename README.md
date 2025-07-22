# OpenStatus CLI

OpenStatus CLI is a command line interface for OpenStatus.

## Installation

```bash
brew install openstatusHQ/cli/openstatus --cask
```

#### Windows
```powershell
iwr instl.sh/openstatushq/cli/windows | iex
```

#### macOS
```bash
curl -sSL instl.sh/openstatushq/cli/macos | bash
```

#### Linux
```bash
curl -sSL instl.sh/openstatushq/cli/linux | bash
```

## Development

### Generate Documentation

Run this command to generate the documentation:

```bash
 go run cmd/docs/docs.go
 cd docs
 pandoc  -s -t man openstatus-docs.md -o openstatus.1
 ```
