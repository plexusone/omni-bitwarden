// Package bitwarden provides an OmniVault provider for Bitwarden Secrets Manager.
//
// This package implements the vault.Vault interface using the official
// Bitwarden Go SDK, allowing applications to access secrets stored in
// Bitwarden Secrets Manager through the unified OmniVault interface.
//
// Authentication requires a Bitwarden access token. Create one at:
// https://bitwarden.com/help/access-tokens/
//
// Basic usage:
//
//	provider, err := bitwarden.New(bitwarden.Config{
//	    AccessToken:    os.Getenv("BW_ACCESS_TOKEN"),
//	    OrganizationID: os.Getenv("BW_ORGANIZATION_ID"),
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer provider.Close()
//
//	secret, err := provider.Get(ctx, "my-api-key")
//
// With OmniVault resolver:
//
//	resolver := omnivault.NewResolver()
//	resolver.Register("bw", provider)
//	value, err := resolver.Resolve(ctx, "bw://org-id/my-api-key")
package bitwarden

import (
	"context"
	"fmt"
	"os"
	"sync"

	sdk "github.com/bitwarden/sdk-go/v2"
	"github.com/plexusone/omnivault/vault"
)

// Provider implements vault.Vault for Bitwarden.
type Provider struct {
	client sdk.BitwardenClientInterface
	config Config

	// secretCache caches secret key -> ID mappings per organization
	secretCache map[string]map[string]string // orgID -> (key -> secretID)
	cacheMu     sync.RWMutex

	mu     sync.RWMutex
	closed bool
}

// New creates a new Bitwarden provider with the given configuration.
func New(config Config) (*Provider, error) {
	config = config.withDefaults()

	// Get values from environment if not provided
	accessToken := config.AccessToken
	if accessToken == "" {
		accessToken = os.Getenv(EnvAccessToken)
	}
	if accessToken == "" {
		return nil, fmt.Errorf("access token is required: set Config.AccessToken or %s environment variable", EnvAccessToken)
	}

	if config.OrganizationID == "" {
		config.OrganizationID = os.Getenv(EnvOrganizationID)
	}

	apiURL := config.APIURL
	if apiURL == "" {
		apiURL = os.Getenv(EnvAPIURL)
	}

	identityURL := config.IdentityURL
	if identityURL == "" {
		identityURL = os.Getenv(EnvIdentityURL)
	}

	// Create Bitwarden client
	var apiURLPtr, identityURLPtr *string
	if apiURL != "" {
		apiURLPtr = &apiURL
	}
	if identityURL != "" {
		identityURLPtr = &identityURL
	}

	client, err := sdk.NewBitwardenClient(apiURLPtr, identityURLPtr)
	if err != nil {
		return nil, fmt.Errorf("failed to create Bitwarden client: %w", err)
	}

	// Authenticate with access token
	var stateFilePtr *string
	if config.StateFile != "" {
		stateFilePtr = &config.StateFile
	}

	if err := client.AccessTokenLogin(accessToken, stateFilePtr); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}

	return &Provider{
		client:      client,
		config:      config,
		secretCache: make(map[string]map[string]string),
	}, nil
}

// NewFromEnv creates a new provider using environment variables.
func NewFromEnv() (*Provider, error) {
	return New(Config{})
}

// Get retrieves a secret from Bitwarden.
//
// Path formats supported:
//   - "secretKey" - returns the secret value (uses default organization)
//   - "secretKey/field" - returns specific field (value, key, note)
//   - "orgID/secretKey" - returns secret from specific organization
//   - "orgID/secretKey/field" - returns specific field from specific org
//   - "bw://orgID/secretKey" - native Bitwarden secret reference
func (p *Provider) Get(ctx context.Context, path string) (*vault.Secret, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return nil, vault.NewVaultError("Get", path, ProviderName, vault.ErrClosed)
	}

	parsed, err := ParsePath(path, p.config.OrganizationID)
	if err != nil {
		return nil, vault.NewVaultError("Get", path, ProviderName, err)
	}

	// Resolve secret key to ID
	secretID, err := p.resolveSecretID(ctx, parsed.OrganizationID, parsed.SecretKey)
	if err != nil {
		return nil, mapError("Get", path, err)
	}

	// Get the secret
	secretResp, err := p.client.Secrets().Get(secretID)
	if err != nil {
		return nil, mapError("Get", path, err)
	}

	return p.secretResponseToVaultSecret(secretResp, parsed), nil
}

// secretResponseToVaultSecret converts a Bitwarden SecretResponse to a vault.Secret.
func (p *Provider) secretResponseToVaultSecret(resp *sdk.SecretResponse, parsed *ParsedPath) *vault.Secret {
	secret := &vault.Secret{
		Value: resp.Value,
		Fields: map[string]string{
			"key":  resp.Key,
			"note": resp.Note,
		},
		Metadata: vault.Metadata{
			Provider:   ProviderName,
			Path:       parsed.String(),
			CreatedAt:  vault.NewTimestamp(resp.CreationDate),
			ModifiedAt: vault.NewTimestamp(resp.RevisionDate),
			Extra: map[string]any{
				"id":             resp.ID,
				"organizationId": resp.OrganizationID,
			},
		},
	}

	if resp.ProjectID != nil {
		secret.Metadata.Extra["projectId"] = *resp.ProjectID
	}

	// If a specific field was requested, return just that field's value
	if parsed.Field != "" {
		switch parsed.Field {
		case "value":
			secret.Value = resp.Value
		case "key":
			secret.Value = resp.Key
		case "note":
			secret.Value = resp.Note
		}
	}

	return secret
}

// Set stores a secret in Bitwarden.
func (p *Provider) Set(ctx context.Context, path string, secret *vault.Secret) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return vault.NewVaultError("Set", path, ProviderName, vault.ErrClosed)
	}

	parsed, err := ParsePath(path, p.config.OrganizationID)
	if err != nil {
		return vault.NewVaultError("Set", path, ProviderName, err)
	}

	// Check if secret exists
	secretID, err := p.resolveSecretID(ctx, parsed.OrganizationID, parsed.SecretKey)
	if err == nil {
		// Update existing secret
		return p.updateSecret(ctx, secretID, parsed, secret)
	}

	// Create new secret
	return p.createSecret(ctx, parsed, secret)
}

// createSecret creates a new secret in Bitwarden.
func (p *Provider) createSecret(ctx context.Context, parsed *ParsedPath, secret *vault.Secret) error {
	key := parsed.SecretKey
	value := secret.Value
	note := ""

	// Get note from fields if provided
	if secret.Fields != nil {
		if n, ok := secret.Fields["note"]; ok {
			note = n
		}
	}

	// Get project IDs from metadata if provided
	var projectIDs []string
	if secret.Metadata.Extra != nil {
		if pids, ok := secret.Metadata.Extra["projectIds"].([]string); ok {
			projectIDs = pids
		}
	}

	_, err := p.client.Secrets().Create(key, value, note, parsed.OrganizationID, projectIDs)
	if err != nil {
		return mapError("Set", parsed.String(), err)
	}

	// Invalidate cache for this org
	p.invalidateOrgCache(parsed.OrganizationID)

	return nil
}

// updateSecret updates an existing secret in Bitwarden.
func (p *Provider) updateSecret(ctx context.Context, secretID string, parsed *ParsedPath, secret *vault.Secret) error {
	key := parsed.SecretKey
	value := secret.Value
	note := ""

	// Get note from fields if provided
	if secret.Fields != nil {
		if n, ok := secret.Fields["note"]; ok {
			note = n
		}
	}

	// Get project IDs from metadata if provided
	var projectIDs []string
	if secret.Metadata.Extra != nil {
		if pids, ok := secret.Metadata.Extra["projectIds"].([]string); ok {
			projectIDs = pids
		}
	}

	_, err := p.client.Secrets().Update(secretID, key, value, note, parsed.OrganizationID, projectIDs)
	if err != nil {
		return mapError("Set", parsed.String(), err)
	}

	return nil
}

// Delete removes a secret from Bitwarden.
func (p *Provider) Delete(ctx context.Context, path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return vault.NewVaultError("Delete", path, ProviderName, vault.ErrClosed)
	}

	parsed, err := ParsePath(path, p.config.OrganizationID)
	if err != nil {
		return vault.NewVaultError("Delete", path, ProviderName, err)
	}

	// Resolve secret key to ID
	secretID, err := p.resolveSecretID(ctx, parsed.OrganizationID, parsed.SecretKey)
	if err != nil {
		// Not found = nothing to delete
		if isNotFoundError(err) {
			return nil
		}
		return mapError("Delete", path, err)
	}

	_, err = p.client.Secrets().Delete([]string{secretID})
	if err != nil {
		if isNotFoundError(err) {
			return nil
		}
		return mapError("Delete", path, err)
	}

	// Invalidate cache for this org
	p.invalidateOrgCache(parsed.OrganizationID)

	return nil
}

// Exists checks if a secret exists in Bitwarden.
func (p *Provider) Exists(ctx context.Context, path string) (bool, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return false, vault.NewVaultError("Exists", path, ProviderName, vault.ErrClosed)
	}

	parsed, err := ParsePath(path, p.config.OrganizationID)
	if err != nil {
		return false, vault.NewVaultError("Exists", path, ProviderName, err)
	}

	_, err = p.resolveSecretID(ctx, parsed.OrganizationID, parsed.SecretKey)
	if err != nil {
		if isNotFoundError(err) {
			return false, nil
		}
		return false, mapError("Exists", path, err)
	}

	return true, nil
}

// List returns all secret paths matching the prefix.
func (p *Provider) List(ctx context.Context, prefix string) ([]string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return nil, vault.NewVaultError("List", prefix, ProviderName, vault.ErrClosed)
	}

	orgID := p.config.OrganizationID
	if orgID == "" {
		return nil, vault.NewVaultError("List", prefix, ProviderName, fmt.Errorf("organization ID is required for List operation"))
	}

	// List all secrets in the organization
	resp, err := p.client.Secrets().List(orgID)
	if err != nil {
		return nil, mapError("List", prefix, err)
	}

	var results []string
	for _, secret := range resp.Data {
		path := fmt.Sprintf("%s/%s", orgID, secret.Key)
		if prefix == "" || len(path) >= len(prefix) && path[:len(prefix)] == prefix {
			results = append(results, path)
		}

		// Cache the ID while we're at it
		p.cacheSecretID(orgID, secret.Key, secret.ID)
	}

	return results, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return ProviderName
}

// Capabilities returns the provider capabilities.
func (p *Provider) Capabilities() vault.Capabilities {
	return vault.Capabilities{
		Read:       true,
		Write:      true,
		Delete:     true,
		List:       true,
		Versioning: false, // SDK doesn't expose version history
		Rotation:   false, // No rotation API in SDK
		Binary:     false, // Secrets are string-based
		MultiField: true,  // Secrets have key, value, note
		Batch:      true,  // GetByIDS for reads
	}
}

// Close releases resources held by the provider.
func (p *Provider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true
	p.client.Close()
	return nil
}

// resolveSecretID resolves a secret key to its ID within an organization.
func (p *Provider) resolveSecretID(ctx context.Context, orgID, secretKey string) (string, error) {
	// Check cache first
	p.cacheMu.RLock()
	if orgCache, ok := p.secretCache[orgID]; ok {
		if id, ok := orgCache[secretKey]; ok {
			p.cacheMu.RUnlock()
			return id, nil
		}
	}
	p.cacheMu.RUnlock()

	// List secrets to find the match
	resp, err := p.client.Secrets().List(orgID)
	if err != nil {
		return "", err
	}

	for _, secret := range resp.Data {
		// Cache all secrets while we're at it
		p.cacheSecretID(orgID, secret.Key, secret.ID)

		// Check for match by key or ID
		if secret.Key == secretKey || secret.ID == secretKey {
			return secret.ID, nil
		}
	}

	return "", fmt.Errorf("secret not found: %s", secretKey)
}

// cacheSecretID caches a secret key -> ID mapping.
func (p *Provider) cacheSecretID(orgID, key, id string) {
	p.cacheMu.Lock()
	defer p.cacheMu.Unlock()

	if p.secretCache[orgID] == nil {
		p.secretCache[orgID] = make(map[string]string)
	}
	p.secretCache[orgID][key] = id
	// Also cache ID -> ID for direct lookups
	p.secretCache[orgID][id] = id
}

// invalidateOrgCache invalidates the cache for an organization.
func (p *Provider) invalidateOrgCache(orgID string) {
	p.cacheMu.Lock()
	defer p.cacheMu.Unlock()
	delete(p.secretCache, orgID)
}

// Ensure Provider implements vault.Vault.
var _ vault.Vault = (*Provider)(nil)
