package bitwarden

import (
	"strings"

	"github.com/plexusone/omnivault/vault"
)

// mapError maps Bitwarden SDK errors to vault errors.
func mapError(op, path string, err error) error {
	if err == nil {
		return nil
	}

	// Check for common error patterns in the error message
	errStr := err.Error()

	if isNotFoundError(err) {
		return vault.NewVaultError(op, path, ProviderName, vault.ErrSecretNotFound)
	}

	if strings.Contains(errStr, "unauthorized") || strings.Contains(errStr, "authentication") {
		return vault.NewVaultError(op, path, ProviderName, vault.ErrAuthenticationFailed)
	}

	return vault.NewVaultError(op, path, ProviderName, err)
}

// isNotFoundError checks if an error indicates the secret was not found.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "not found") ||
		strings.Contains(errStr, "does not exist") ||
		strings.Contains(errStr, "no secret")
}
