package op

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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
	fullItem, err := s.getItem(item)
	if err != nil {
		return "", fmt.Errorf("failed to read field %q from item %q: %w", field, item, err)
	}

	// Build section title→ID map to support "section.field" notation used by the CLI.
	sectionIDs := make(map[string]string, len(fullItem.Sections))
	for _, sec := range fullItem.Sections {
		sectionIDs[strings.ToLower(sec.Title)] = sec.ID
	}

	// If the field name contains a dot, treat the prefix as a section name.
	var wantSectionID string
	fieldTitle := field
	if dotIdx := strings.Index(field, "."); dotIdx >= 0 {
		sectionName := field[:dotIdx]
		fieldTitle = field[dotIdx+1:]
		wantSectionID = sectionIDs[strings.ToLower(sectionName)]
	}

	for _, f := range fullItem.Fields {
		if !strings.EqualFold(f.Title, fieldTitle) {
			continue
		}
		if wantSectionID != "" && (f.SectionID == nil || !strings.EqualFold(*f.SectionID, wantSectionID)) {
			continue
		}
		val := strings.TrimSpace(f.Value)
		if val == "" {
			return "", fmt.Errorf("empty value for field %q in item %q", field, item)
		}
		return val, nil
	}
	return "", fmt.Errorf("field %q not found in item %q", field, item)
}

func (s *SDKClient) GetTOTP(_, item string) (string, error) {
	fullItem, err := s.getItem(item)
	if err != nil {
		return "", fmt.Errorf("failed to get TOTP from item %q: %w", item, err)
	}
	for _, f := range fullItem.Fields {
		if f.Details != nil {
			if otp := f.Details.OTP(); otp != nil {
				if otp.ErrorMessage != nil && *otp.ErrorMessage != "" {
					return "", fmt.Errorf("TOTP error for item %q: %s", item, *otp.ErrorMessage)
				}
				if otp.Code != nil && *otp.Code != "" {
					return strings.TrimSpace(*otp.Code), nil
				}
			}
		}
	}
	return "", fmt.Errorf("no TOTP field found in item %q", item)
}

func (s *SDKClient) EditItem(_, item string, assignments ...string) error {
	fullItem, err := s.getItem(item)
	if err != nil {
		return err
	}

	// Build section title→ID map to support "section.field" notation used by the CLI.
	sectionIDs := make(map[string]string, len(fullItem.Sections))
	for _, sec := range fullItem.Sections {
		sectionIDs[strings.ToLower(sec.Title)] = sec.ID
	}

	// Apply assignments (format: "field label=value" or "section.field label[type]=value")
	for _, assignment := range assignments {
		eqIdx := strings.Index(assignment, "=")
		if eqIdx < 0 {
			continue
		}
		label := assignment[:eqIdx]
		value := assignment[eqIdx+1:]
		// Strip type annotation like [text] or [password]
		fieldType := onepassword.ItemFieldTypeText
		if bracketIdx := strings.Index(label, "["); bracketIdx >= 0 {
			typeStr := label[bracketIdx+1 : len(label)-1]
			label = label[:bracketIdx]
			if typeStr == "password" {
				fieldType = onepassword.ItemFieldTypeConcealed
			}
		}

		// If label contains a dot, treat prefix as section name.
		var wantSectionID string
		fieldTitle := label
		if dotIdx := strings.Index(label, "."); dotIdx >= 0 {
			sectionName := label[:dotIdx]
			fieldTitle = label[dotIdx+1:]
			wantSectionID = sectionIDs[strings.ToLower(sectionName)]
		}

		found := false
		for i := range fullItem.Fields {
			if !strings.EqualFold(fullItem.Fields[i].Title, fieldTitle) {
				continue
			}
			if wantSectionID != "" && (fullItem.Fields[i].SectionID == nil || !strings.EqualFold(*fullItem.Fields[i].SectionID, wantSectionID)) {
				continue
			}
			fullItem.Fields[i].Value = value
			found = true
			break
		}
		if !found {
			newField := onepassword.ItemField{
				Title:     fieldTitle,
				Value:     value,
				FieldType: fieldType,
			}
			if wantSectionID != "" {
				newField.SectionID = &wantSectionID
			}
			fullItem.Fields = append(fullItem.Fields, newField)
		}
	}

	// Strip fields of type Unsupported so the SDK Put has a chance.
	kept := fullItem.Fields[:0]
	for _, f := range fullItem.Fields {
		if f.FieldType == onepassword.ItemFieldTypeUnsupported {
			continue
		}
		kept = append(kept, f)
	}
	fullItem.Fields = kept

	_, err = s.inner.Items().Put(context.Background(), fullItem)
	if err == nil {
		return nil
	}

	// SDK Put rejects items with legacy/orphan fields it cannot round-trip
	// (empty-titled TOTPs, "add more" sections from old templates, etc.) with
	// "Editing is not supported for unsupported fields". Fall back to the op
	// CLI, which does partial updates and doesn't need to round-trip the item.
	if cliErr := s.editViaCLI(item, assignments...); cliErr != nil {
		return fmt.Errorf("SDK update failed (%v); CLI fallback also failed: %w", err, cliErr)
	}
	return nil
}

// editViaCLI shells out to the op CLI authenticated with the same service
// account token. Used as a fallback when the SDK Put rejects an item.
func (s *SDKClient) editViaCLI(item string, assignments ...string) error {
	if _, err := exec.LookPath("op"); err != nil {
		return fmt.Errorf("op CLI not found in PATH (needed as fallback for items the SDK cannot edit). Install: https://developer.1password.com/docs/cli/get-started/")
	}

	args := []string{"item", "edit", item}
	if s.vault != "" {
		args = append(args, "--vault", s.vault)
	}
	args = append(args, assignments...)

	cmd := exec.Command("op", args...)
	cmd.Env = append(os.Environ(), "OP_SERVICE_ACCOUNT_TOKEN="+s.token)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
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
	fullItem, err := s.getItem(item)
	if err != nil {
		return nil, err
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

// getItem looks up a 1Password item by title in the default vault and
// returns the full item with all fields populated.
func (s *SDKClient) getItem(title string) (onepassword.Item, error) {
	vaultID, err := s.resolveVaultID()
	if err != nil {
		return onepassword.Item{}, err
	}

	items, err := s.inner.Items().List(context.Background(), vaultID)
	if err != nil {
		return onepassword.Item{}, fmt.Errorf("failed to list items: %w", err)
	}

	var itemID string
	for _, it := range items {
		if it.Title == title {
			itemID = it.ID
			break
		}
	}
	if itemID == "" {
		return onepassword.Item{}, fmt.Errorf("item %q not found in vault %q", title, s.vault)
	}

	fullItem, err := s.inner.Items().Get(context.Background(), vaultID, itemID)
	if err != nil {
		return onepassword.Item{}, fmt.Errorf("failed to get item %q: %w", title, err)
	}
	return fullItem, nil
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
