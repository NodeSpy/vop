package op

import (
	"context"
	"fmt"
	"strings"

	onepassword "github.com/1password/onepassword-sdk-go"
)

// SDKClient implements Client using the 1Password Go SDK with a service
// account token. It requires no op CLI binary.
type SDKClient struct {
	token string
	vault string // default vault name for this profile
	inner *onepassword.Client
}

// NewSDK creates a new SDKClient. The vault parameter is the default vault
// name used for operations that need it.
func NewSDK(token, vault string) (*SDKClient, error) {
	client, err := onepassword.NewClient(
		context.Background(),
		onepassword.WithServiceAccountToken(token),
		onepassword.WithIntegrationInfo("vop", "1.0.0"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize 1Password SDK: %w", err)
	}
	return &SDKClient{
		token: token,
		vault: vault,
		inner: client,
	}, nil
}

func (s *SDKClient) IsInstalled() bool {
	return true // SDK is compiled in, always available
}

func (s *SDKClient) EnsureSignedIn(_ string) error {
	return nil // service account tokens are stateless — no sign-in needed
}

func (s *SDKClient) ReadField(_, item, field string) (string, error) {
	ref := fmt.Sprintf("op://%s/%s/%s", s.vault, item, field)
	val, err := s.inner.Secrets().Resolve(context.Background(), ref)
	if err != nil {
		return "", fmt.Errorf("failed to read field %q from item %q: %w", field, item, err)
	}
	val = strings.TrimSpace(val)
	if val == "" {
		return "", fmt.Errorf("empty value for field %q in item %q", field, item)
	}
	return val, nil
}

func (s *SDKClient) GetTOTP(_, item string) (string, error) {
	ref := fmt.Sprintf("op://%s/%s/one-time password?attribute=otp", s.vault, item)
	code, err := s.inner.Secrets().Resolve(context.Background(), ref)
	if err != nil {
		return "", fmt.Errorf("failed to get TOTP from item %q: %w", item, err)
	}
	return strings.TrimSpace(code), nil
}

func (s *SDKClient) EditItem(_, item string, assignments ...string) error {
	// Find the item first
	vaultID, err := s.resolveVaultID()
	if err != nil {
		return err
	}

	items, err := s.inner.Items().List(context.Background(), vaultID)
	if err != nil {
		return fmt.Errorf("failed to list items: %w", err)
	}

	var itemID string
	for _, it := range items {
		if it.Title == item {
			itemID = it.ID
			break
		}
	}
	if itemID == "" {
		return fmt.Errorf("item %q not found in vault %q", item, s.vault)
	}

	fullItem, err := s.inner.Items().Get(context.Background(), vaultID, itemID)
	if err != nil {
		return fmt.Errorf("failed to get item %q: %w", item, err)
	}

	// Apply assignments (format: "field label=value" or "field label[type]=value")
	for _, assignment := range assignments {
		eqIdx := strings.Index(assignment, "=")
		if eqIdx < 0 {
			continue
		}
		label := assignment[:eqIdx]
		value := assignment[eqIdx+1:]
		// Strip type annotation like [text] or [password]
		if bracketIdx := strings.Index(label, "["); bracketIdx >= 0 {
			label = label[:bracketIdx]
		}

		found := false
		for i := range fullItem.Fields {
			if strings.EqualFold(fullItem.Fields[i].Title, label) {
				fullItem.Fields[i].Value = value
				found = true
				break
			}
		}
		if !found {
			// Add as a new text field
			fullItem.Fields = append(fullItem.Fields, onepassword.ItemField{
				Title:     label,
				Value:     value,
				FieldType: onepassword.ItemFieldTypeText,
			})
		}
	}

	_, err = s.inner.Items().Put(context.Background(), fullItem)
	return err
}

func (s *SDKClient) ListAccounts() ([]OPAccount, error) {
	// Service accounts are single-account; return a placeholder.
	return []OPAccount{{URL: "(service account)", Email: "(service account)"}}, nil
}

func (s *SDKClient) ListVaults(_ string) ([]OPVault, error) {
	vaults, err := s.inner.Vaults().List(context.Background())
	if err != nil {
		return nil, err
	}
	result := make([]OPVault, len(vaults))
	for i, v := range vaults {
		result[i] = OPVault{ID: v.ID, Name: v.Title}
	}
	return result, nil
}

func (s *SDKClient) ListItems(_, vault string) ([]OPItem, error) {
	if vault == "" {
		vault = s.vault
	}
	vaultID, err := s.resolveVaultIDByName(vault)
	if err != nil {
		return nil, err
	}
	items, err := s.inner.Items().List(context.Background(), vaultID)
	if err != nil {
		return nil, err
	}
	result := make([]OPItem, len(items))
	for i, it := range items {
		result[i] = OPItem{ID: it.ID, Title: it.Title, Category: string(it.Category)}
	}
	return result, nil
}

func (s *SDKClient) ListFields(_, item string) ([]OPField, error) {
	vaultID, err := s.resolveVaultID()
	if err != nil {
		return nil, err
	}

	items, err := s.inner.Items().List(context.Background(), vaultID)
	if err != nil {
		return nil, fmt.Errorf("failed to list items: %w", err)
	}

	var itemID string
	for _, it := range items {
		if it.Title == item {
			itemID = it.ID
			break
		}
	}
	if itemID == "" {
		return nil, fmt.Errorf("item %q not found in vault %q", item, s.vault)
	}

	fullItem, err := s.inner.Items().Get(context.Background(), vaultID, itemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get item %q: %w", item, err)
	}

	var fields []OPField
	for _, f := range fullItem.Fields {
		if f.Title == "" {
			continue
		}
		fields = append(fields, OPField{
			Label: f.Title,
			Type:  string(f.FieldType),
		})
	}
	return fields, nil
}

func (s *SDKClient) CreateItem(_, vault, category, title, tags string, assignments ...string) error {
	if vault == "" {
		vault = s.vault
	}

	vaultID, err := s.resolveVaultIDByName(vault)
	if err != nil {
		return err
	}

	cat := mapCategory(category)

	var fields []onepassword.ItemField
	for _, assignment := range assignments {
		eqIdx := strings.Index(assignment, "=")
		if eqIdx < 0 {
			continue
		}
		label := assignment[:eqIdx]
		value := assignment[eqIdx+1:]

		fieldType := onepassword.ItemFieldTypeText
		if bracketIdx := strings.Index(label, "["); bracketIdx >= 0 {
			typeStr := label[bracketIdx+1 : len(label)-1]
			label = label[:bracketIdx]
			if typeStr == "password" {
				fieldType = onepassword.ItemFieldTypeConcealed
			}
		}

		fields = append(fields, onepassword.ItemField{
			Title:     label,
			Value:     value,
			FieldType: fieldType,
		})
	}

	var tagList []string
	if tags != "" {
		tagList = strings.Split(tags, ",")
	}

	params := onepassword.ItemCreateParams{
		VaultID:  vaultID,
		Title:    title,
		Category: cat,
		Fields:   fields,
		Tags:     tagList,
	}

	_, err = s.inner.Items().Create(context.Background(), params)
	return err
}

// resolveVaultID returns the vault ID for the client's default vault.
func (s *SDKClient) resolveVaultID() (string, error) {
	return s.resolveVaultIDByName(s.vault)
}

// resolveVaultIDByName looks up a vault ID by name.
func (s *SDKClient) resolveVaultIDByName(name string) (string, error) {
	vaults, err := s.inner.Vaults().List(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to list vaults: %w", err)
	}
	for _, v := range vaults {
		if strings.EqualFold(v.Title, name) {
			return v.ID, nil
		}
	}
	return "", fmt.Errorf("vault %q not found", name)
}

func mapCategory(cat string) onepassword.ItemCategory {
	switch strings.ToLower(cat) {
	case "login":
		return onepassword.ItemCategoryLogin
	case "password":
		return onepassword.ItemCategoryPassword
	case "secure note", "securenote":
		return onepassword.ItemCategorySecureNote
	case "api credential", "apicredential":
		return onepassword.ItemCategoryAPICredentials
	default:
		return onepassword.ItemCategoryLogin
	}
}
