package bitwarden

import "log/slog"

const (
	// ProviderName is the name returned by Provider.Name().
	ProviderName = "bitwarden"

	// EnvAccessToken is the environment variable for the access token.
	EnvAccessToken = "BW_ACCESS_TOKEN" //nolint:gosec // G101: this is an env var name, not a credential

	// EnvOrganizationID is the environment variable for the default organization ID.
	EnvOrganizationID = "BW_ORGANIZATION_ID"

	// EnvAPIURL is the environment variable for the API URL.
	EnvAPIURL = "BW_API_URL"

	// EnvIdentityURL is the environment variable for the identity URL.
	EnvIdentityURL = "BW_IDENTITY_URL"
)

// Config holds configuration for the Bitwarden provider.
type Config struct {
	// AccessToken is the Bitwarden access token for authentication.
	// Required. Can also be set via BW_ACCESS_TOKEN environment variable.
	AccessToken string

	// OrganizationID is the default organization ID for operations.
	// Required when path doesn't specify an organization.
	// Can be set via BW_ORGANIZATION_ID environment variable.
	OrganizationID string

	// APIURL is the Bitwarden API URL.
	// Optional. Uses default Bitwarden cloud if not set.
	// Can be set via BW_API_URL environment variable.
	APIURL string

	// IdentityURL is the Bitwarden Identity URL.
	// Optional. Uses default Bitwarden cloud if not set.
	// Can be set via BW_IDENTITY_URL environment variable.
	IdentityURL string

	// StateFile is an optional file path to persist authentication state.
	// This can speed up subsequent connections.
	StateFile string

	// Logger for debug output. Optional.
	Logger *slog.Logger
}

// withDefaults returns a copy of the config with default values applied.
func (c Config) withDefaults() Config {
	// No defaults to apply currently; all configuration comes from
	// environment variables or explicit config
	return c
}
