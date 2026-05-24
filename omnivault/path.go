package bitwarden

import (
	"fmt"
	"strings"
)

// ParsedPath represents a parsed Bitwarden secret path.
//
// Path formats supported:
//   - "organizationID/secretKey"
//   - "organizationID/secretKey/field"
//   - "secretKey" (uses default organization)
//   - "secretKey/field" (uses default organization)
//   - "bw://organizationID/secretKey"
//   - "bw://organizationID/secretKey/field"
type ParsedPath struct {
	// OrganizationID is the Bitwarden organization ID.
	OrganizationID string

	// SecretKey is the secret name/key.
	SecretKey string

	// Field is the optional field to retrieve (value, key, note).
	// Empty means return the whole secret (default: value).
	Field string
}

// ParsePath parses a Bitwarden secret path.
func ParsePath(path string, defaultOrgID string) (*ParsedPath, error) {
	// Remove bw:// prefix if present
	path = strings.TrimPrefix(path, "bw://")
	path = strings.TrimPrefix(path, "/")

	if path == "" {
		return nil, fmt.Errorf("path is required")
	}

	parts := strings.Split(path, "/")

	result := &ParsedPath{}

	switch len(parts) {
	case 1:
		// "secretKey" - uses default organization
		if defaultOrgID == "" {
			return nil, fmt.Errorf("organization ID is required: provide in path or set default")
		}
		result.OrganizationID = defaultOrgID
		result.SecretKey = parts[0]

	case 2:
		// Could be "orgID/secretKey" or "secretKey/field"
		// If the first part looks like a UUID, treat it as orgID
		// Otherwise, treat it as secretKey/field
		if looksLikeUUID(parts[0]) {
			result.OrganizationID = parts[0]
			result.SecretKey = parts[1]
		} else if defaultOrgID != "" {
			result.OrganizationID = defaultOrgID
			result.SecretKey = parts[0]
			result.Field = parts[1]
		} else {
			// Assume it's orgID/secretKey
			result.OrganizationID = parts[0]
			result.SecretKey = parts[1]
		}

	case 3:
		// "orgID/secretKey/field"
		result.OrganizationID = parts[0]
		result.SecretKey = parts[1]
		result.Field = parts[2]

	default:
		return nil, fmt.Errorf("invalid path format: %s", path)
	}

	if result.SecretKey == "" {
		return nil, fmt.Errorf("secret key is required")
	}

	return result, nil
}

// String returns the path as a string.
func (p *ParsedPath) String() string {
	if p.Field != "" {
		return fmt.Sprintf("%s/%s/%s", p.OrganizationID, p.SecretKey, p.Field)
	}
	return fmt.Sprintf("%s/%s", p.OrganizationID, p.SecretKey)
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
