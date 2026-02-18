// Package op provides a unified interface for 1Password operations,
// backed by either the op CLI or the 1Password SDK (service accounts).
package op

// Client defines the operations vop needs from 1Password.
// CLIClient and SDKClient both implement this interface.
type Client interface {
	// EnsureSignedIn verifies authentication, prompting if necessary.
	// For SDK clients this is a no-op (token-based auth).
	EnsureSignedIn(account string) error

	// ReadField reads a specific field from a 1Password item.
	ReadField(account, item, field string) (string, error)

	// GetTOTP retrieves the current TOTP code from a 1Password item.
	GetTOTP(account, item string) (string, error)

	// EditItem updates fields on an existing 1Password item.
	EditItem(account, item string, assignments ...string) error

	// ListAccounts returns all configured 1Password accounts.
	ListAccounts() ([]OPAccount, error)

	// ListVaults returns all vaults for a given account.
	ListVaults(account string) ([]OPVault, error)

	// CreateItem creates a new 1Password item.
	CreateItem(account, vault, category, title, tags string, assignments ...string) error

	// ListItems returns items in a vault for a given account.
	ListItems(account, vault string) ([]OPItem, error)

	// ListFields returns the field labels on a 1Password item.
	ListFields(account, item string) ([]OPField, error)

	// IsInstalled reports whether the backend is available.
	IsInstalled() bool
}

// OPAccount represents a 1Password account.
type OPAccount struct {
	URL   string `json:"url"`
	Email string `json:"email"`
}

// OPVault represents a 1Password vault.
type OPVault struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// OPItem represents a 1Password item (summary).
type OPItem struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Category string `json:"category"`
}

// OPField represents a field on a 1Password item.
type OPField struct {
	Label string `json:"label"`
	Type  string `json:"type"` // e.g. "STRING", "CONCEALED"
}
