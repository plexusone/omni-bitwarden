// Package register provides automatic registration of the Bitwarden provider
// with omnivault's provider registry.
//
// Import this package for side effects to enable Bitwarden support in
// omnivault.VaultFromURI:
//
//	import _ "github.com/plexusone/omni-bitwarden/omnivault/register"
//
//	// Now you can use bw:// URIs
//	v, err := omnivault.VaultFromURI("bw://org-id/my-secret")
//
// Prerequisites:
//   - Set BW_ACCESS_TOKEN environment variable
//   - Set BW_ORGANIZATION_ID environment variable (or include org ID in path)
package register

import (
	"strings"

	bitwarden "github.com/plexusone/omni-bitwarden/omnivault"
	"github.com/plexusone/omnivault"
	"github.com/plexusone/omnivault/vault"
)

func init() {
	omnivault.RegisterProvider("bw", factory)
}

func factory(uri string) (vault.Vault, error) {
	// Parse: bw://orgID/secretKey or bw://secretKey
	// Extract default organization from URI if provided
	path := strings.TrimPrefix(uri, "bw://")
	config := bitwarden.Config{}

	if path != "" {
		// First path component might be the organization ID
		parts := strings.SplitN(path, "/", 2)
		if len(parts) >= 1 && looksLikeUUID(parts[0]) {
			config.OrganizationID = parts[0]
		}
	}

	return bitwarden.New(config)
}

// looksLikeUUID returns true if the string appears to be a UUID.
func looksLikeUUID(s string) bool {
	// Simple check: UUIDs are typically 36 chars with hyphens or 32 without
	if len(s) == 36 && strings.Count(s, "-") == 4 {
		return true
	}
	if len(s) == 32 && strings.Count(s, "-") == 0 {
		return true
	}
	return false
}
