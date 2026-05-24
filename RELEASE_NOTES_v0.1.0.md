# Release v0.1.0

Initial release of omni-bitwarden, a Bitwarden Secrets Manager provider for [omnivault](https://github.com/plexusone/omnivault).

## Features

- Bitwarden Secrets Manager integration via the official [Bitwarden Go SDK](https://github.com/bitwarden/sdk-go)
- Full CRUD operations (read, write, delete, list)
- Multi-field support (key, value, note)
- Flexible path formats for secret access
- Auto-registration support for `bw://` URI scheme
- Compatible with omnivault-desktop for multi-provider setups

## Installation

```bash
go get github.com/plexusone/omni-bitwarden@v0.1.0
```

## Quick Start

```go
import (
    bitwarden "github.com/plexusone/omni-bitwarden/omnivault"
)

// Create provider using environment variables
provider, err := bitwarden.NewFromEnv()
if err != nil {
    log.Fatal(err)
}
defer provider.Close()

// Get a secret
secret, err := provider.Get(ctx, "my-api-key")
fmt.Println("Value:", secret.Value)
```

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `BW_ACCESS_TOKEN` | Yes | Access token for authentication |
| `BW_ORGANIZATION_ID` | Yes | Default organization ID |
| `BW_API_URL` | No | Custom API URL (self-hosted) |
| `BW_IDENTITY_URL` | No | Custom Identity URL (self-hosted) |

## What's Changed

### New Features

- Bitwarden Secrets Manager provider implementation
- Auto-registration for `bw://` URI scheme

### Documentation

- Comprehensive README with usage examples
- Authentication and configuration guides

### Infrastructure

- GitHub Actions CI/CD workflows
- golangci-lint configuration
- MIT license
