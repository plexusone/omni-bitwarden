# Omni-Bitwarden

[![Go CI][go-ci-svg]][go-ci-url]
[![Go Lint][go-lint-svg]][go-lint-url]
[![Go Report Card][goreport-svg]][goreport-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![License][license-svg]][license-url]

 [go-ci-svg]: https://github.com/plexusone/omni-bitwarden/actions/workflows/go-ci.yaml/badge.svg?branch=main
 [go-ci-url]: https://github.com/plexusone/omni-bitwarden/actions/workflows/go-ci.yaml
 [go-lint-svg]: https://github.com/plexusone/omni-bitwarden/actions/workflows/go-lint.yaml/badge.svg?branch=main
 [go-lint-url]: https://github.com/plexusone/omni-bitwarden/actions/workflows/go-lint.yaml
 [goreport-svg]: https://goreportcard.com/badge/github.com/plexusone/omni-bitwarden
 [goreport-url]: https://goreportcard.com/report/github.com/plexusone/omni-bitwarden
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/plexusone/omni-bitwarden
 [docs-godoc-url]: https://pkg.go.dev/github.com/plexusone/omni-bitwarden
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/plexusone/omni-bitwarden/blob/main/LICENSE

Bitwarden Secrets Manager provider for [omnivault](https://github.com/plexusone/omnivault).

## Overview

This package implements the `vault.Vault` interface using the official [Bitwarden Go SDK](https://github.com/bitwarden/sdk-go), allowing applications to access secrets stored in Bitwarden Secrets Manager through the unified OmniVault interface.

## Installation

```bash
go get github.com/plexusone/omni-bitwarden
```

## Authentication

Bitwarden Secrets Manager requires:

| Variable | Required | Description |
|----------|----------|-------------|
| `BW_ACCESS_TOKEN` | Yes | Access token for authentication |
| `BW_ORGANIZATION_ID` | Yes* | Default organization ID (*can be specified in path) |

Optional for self-hosted instances:

| Variable | Description |
|----------|-------------|
| `BW_API_URL` | Custom API URL |
| `BW_IDENTITY_URL` | Custom Identity URL |

To create an access token, see [Bitwarden Access Tokens documentation](https://bitwarden.com/help/access-tokens/).

## Quick Start

### Direct Usage

```go
import (
    bitwarden "github.com/plexusone/omni-bitwarden/omnivault"
)

// Create provider (uses BW_ACCESS_TOKEN and BW_ORGANIZATION_ID env vars)
provider, err := bitwarden.NewFromEnv()
if err != nil {
    log.Fatal(err)
}
defer provider.Close()

// Get a secret
secret, err := provider.Get(ctx, "my-api-key")
fmt.Println("Value:", secret.Value)

// Get a specific field
secret, err := provider.Get(ctx, "my-api-key/note")
fmt.Println("Note:", secret.Value)
```

### With OmniVault Resolver

```go
import (
    "github.com/plexusone/omnivault"
    bitwarden "github.com/plexusone/omni-bitwarden/omnivault"
)

provider, _ := bitwarden.NewFromEnv()
resolver := omnivault.NewResolver()
resolver.Register("bw", provider)

// Resolve using URI
value, err := resolver.Resolve(ctx, "bw://org-id/my-api-key")
```

### Auto-Registration

Import the register package to automatically register the `bw://` scheme:

```go
import (
    "github.com/plexusone/omnivault"
    _ "github.com/plexusone/omni-bitwarden/omnivault/register"
)

// Now bw:// URIs work automatically
vault, err := omnivault.VaultFromURI("bw://org-id")
secret, err := vault.Get(ctx, "my-api-key")
```

## Path Formats

| Format | Example | Description |
|--------|---------|-------------|
| Secret key | `my-api-key` | Uses default organization ID |
| Key with field | `my-api-key/note` | Returns specific field |
| Org + key | `org-id/my-api-key` | Specific organization |
| Org + key + field | `org-id/my-api-key/value` | Specific field from org |
| Native URI | `bw://org-id/my-api-key` | Bitwarden secret reference |

## Supported Fields

| Field | Description |
|-------|-------------|
| `value` | Secret value (default) |
| `key` | Secret key/name |
| `note` | Secret note |

## Configuration

```go
provider, err := bitwarden.New(bitwarden.Config{
    AccessToken:    "access-token",   // Required
    OrganizationID: "org-id",         // Default org for operations
    APIURL:         "",               // For self-hosted (optional)
    IdentityURL:    "",               // For self-hosted (optional)
    StateFile:      "",               // Persist auth state (optional)
})
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `BW_ACCESS_TOKEN` | Access token for authentication |
| `BW_ORGANIZATION_ID` | Default organization ID |
| `BW_API_URL` | Custom API URL (self-hosted) |
| `BW_IDENTITY_URL` | Custom Identity URL (self-hosted) |

## Capabilities

| Capability | Supported |
|------------|-----------|
| Read | Yes |
| Write | Yes |
| Delete | Yes |
| List | Yes |
| Multi-Field | Yes (key, value, note) |
| Batch | Yes |
| Binary | No |

## Usage with omnivault-desktop

For applications using multiple vault providers (1Password, Bitwarden, Keeper), import [omnivault-desktop](https://github.com/plexusone/omnivault-desktop):

```go
import (
    "github.com/plexusone/omnivault"
    _ "github.com/plexusone/omnivault-desktop" // Registers all desktop vault providers
)

func main() {
    // Bitwarden provider is automatically registered
    vault, err := omnivault.VaultFromURI("bw://org-id")
    // ...
}
```

## License

MIT
