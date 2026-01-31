# OpenStatus CLI

OpenStatus CLI is a command line interface for OpenStatus.

## Installation

```bash
brew install openstatusHQ/cli/openstatus --cask
```

#### Windows
```powershell
iwr https://raw.githubusercontent.com/openstatusHQ/cli/refs/heads/main/install.ps1 | iex
```

#### macOS
```bash
curl -fsSL https://raw.githubusercontent.com/openstatusHQ/cli/refs/heads/main/install.sh| bash
```

#### Linux
```bash
curl -fsSL https://raw.githubusercontent.com/openstatusHQ/cli/refs/heads/main/install.sh | bash
```

## Development

### Generate Documentation

Run this command to generate the documentation:

```bash
 go run cmd/docs/docs.go
 cd docs
 pandoc  -s -t man openstatus-docs.md -o openstatus.1
 ```
