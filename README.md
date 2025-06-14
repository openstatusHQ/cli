# OpenStatus CLI

OpenStatus CLI is a command line interface for OpenStatus.

## Installation

```bash
brew tap openstatusHQ/cli
brew install openstatus
```


## Development

### Generate Documentation

Run this command to generate the documentation:

```bash
 go run cmd/docs/docs.go
 cd docs
 pandoc  -s -t man openstatus-docs.md -o openstatus.1
 ```
