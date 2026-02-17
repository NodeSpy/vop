package op

import (
	"encoding/json"
	"fmt"
	"testing"
)

// mockCommander implements Commander for testing.
type mockCommander struct {
	// RunFunc is called for each Run invocation.
	RunFunc func(args ...string) (string, error)
	// RunPassthroughFunc is called for each RunPassthrough invocation.
	RunPassthroughFunc func(args ...string) error
	// Calls records all Run calls for assertion.
	Calls [][]string
}

func (m *mockCommander) Run(args ...string) (string, error) {
	m.Calls = append(m.Calls, args)
	if m.RunFunc != nil {
		return m.RunFunc(args...)
	}
	return "", nil
}

func (m *mockCommander) RunPassthrough(args ...string) error {
	m.Calls = append(m.Calls, args)
	if m.RunPassthroughFunc != nil {
		return m.RunPassthroughFunc(args...)
	}
	return nil
}

func TestNewCLI(t *testing.T) {
	client := NewCLI()
	if client == nil {
		t.Fatal("NewCLI() returned nil")
	}
	if client.Cmd == nil {
		t.Fatal("NewCLI().Cmd is nil")
	}
}

func TestCLIClient_ImplementsInterface(t *testing.T) {
	var _ Client = (*CLIClient)(nil)
}

func TestSDKClient_ImplementsInterface(t *testing.T) {
	var _ Client = (*SDKClient)(nil)
}

func TestEnsureSignedIn_AlreadySignedIn(t *testing.T) {
	mock := &mockCommander{
		RunFunc: func(args ...string) (string, error) {
			return "ok", nil
		},
	}
	client := &CLIClient{Cmd: mock}

	err := client.EnsureSignedIn("my.1password.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.Calls))
	}
	// Should have called account get
	args := mock.Calls[0]
	if args[0] != "account" || args[1] != "get" {
		t.Errorf("expected 'account get', got %v", args[:2])
	}
}

func TestEnsureSignedIn_NeedsSignin(t *testing.T) {
	callCount := 0
	mock := &mockCommander{
		RunFunc: func(args ...string) (string, error) {
			return "", fmt.Errorf("not signed in")
		},
		RunPassthroughFunc: func(args ...string) error {
			callCount++
			return nil
		},
	}
	client := &CLIClient{Cmd: mock}

	err := client.EnsureSignedIn("my.1password.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected RunPassthrough called once, got %d", callCount)
	}
	// Second call should be signin
	args := mock.Calls[1]
	if args[0] != "signin" {
		t.Errorf("expected 'signin', got %v", args[0])
	}
}

func TestReadField_Success(t *testing.T) {
	mock := &mockCommander{
		RunFunc: func(args ...string) (string, error) {
			return "AKIAEXAMPLE", nil
		},
	}
	client := &CLIClient{Cmd: mock}

	val, err := client.ReadField("acct", "AWS - prod", "access key id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "AKIAEXAMPLE" {
		t.Errorf("expected 'AKIAEXAMPLE', got %q", val)
	}

	// Verify correct args
	args := mock.Calls[0]
	if args[0] != "item" || args[1] != "get" || args[2] != "AWS - prod" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestReadField_Empty(t *testing.T) {
	mock := &mockCommander{
		RunFunc: func(args ...string) (string, error) {
			return "", nil
		},
	}
	client := &CLIClient{Cmd: mock}

	_, err := client.ReadField("acct", "item", "field")
	if err == nil {
		t.Fatal("expected error for empty field value, got nil")
	}
}

func TestReadField_Error(t *testing.T) {
	mock := &mockCommander{
		RunFunc: func(args ...string) (string, error) {
			return "", fmt.Errorf("op error")
		},
	}
	client := &CLIClient{Cmd: mock}

	_, err := client.ReadField("acct", "item", "field")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetTOTP(t *testing.T) {
	mock := &mockCommander{
		RunFunc: func(args ...string) (string, error) {
			return "123456", nil
		},
	}
	client := &CLIClient{Cmd: mock}

	code, err := client.GetTOTP("acct", "MFA item")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != "123456" {
		t.Errorf("expected '123456', got %q", code)
	}

	args := mock.Calls[0]
	if args[0] != "item" || args[1] != "get" {
		t.Errorf("unexpected args: %v", args)
	}
	// Should have --otp flag
	foundOTP := false
	for _, a := range args {
		if a == "--otp" {
			foundOTP = true
		}
	}
	if !foundOTP {
		t.Error("expected --otp flag in args")
	}
}

func TestEditItem(t *testing.T) {
	mock := &mockCommander{}
	client := &CLIClient{Cmd: mock}

	err := client.EditItem("acct", "item", "field1=val1", "field2=val2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	args := mock.Calls[0]
	if args[0] != "item" || args[1] != "edit" || args[2] != "item" {
		t.Errorf("unexpected args: %v", args)
	}
	// Check assignments are at the end
	if args[len(args)-2] != "field1=val1" || args[len(args)-1] != "field2=val2" {
		t.Errorf("expected assignments at end, got %v", args)
	}
}

func TestListAccounts(t *testing.T) {
	accounts := []OPAccount{
		{URL: "https://my.1password.com", Email: "user@example.com"},
		{URL: "https://team.1password.com", Email: "team@example.com"},
	}
	accountJSON, _ := json.Marshal(accounts)

	mock := &mockCommander{
		RunFunc: func(args ...string) (string, error) {
			return string(accountJSON), nil
		},
	}
	client := &CLIClient{Cmd: mock}

	result, err := client.ListAccounts()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(result))
	}
	if result[0].Email != "user@example.com" {
		t.Errorf("expected email 'user@example.com', got %q", result[0].Email)
	}
}

func TestListVaults(t *testing.T) {
	vaults := []OPVault{{Name: "Private"}, {Name: "Shared"}}
	vaultJSON, _ := json.Marshal(vaults)

	mock := &mockCommander{
		RunFunc: func(args ...string) (string, error) {
			return string(vaultJSON), nil
		},
	}
	client := &CLIClient{Cmd: mock}

	result, err := client.ListVaults("acct")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 vaults, got %d", len(result))
	}
	if result[0].Name != "Private" {
		t.Errorf("expected 'Private', got %q", result[0].Name)
	}
}

func TestCreateItem(t *testing.T) {
	mock := &mockCommander{}
	client := &CLIClient{Cmd: mock}

	err := client.CreateItem("acct", "Private", "Login", "AWS - test", "aws,vop",
		"access key id[text]=AKIA...", "secret access key[password]=secret...")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	args := mock.Calls[0]
	if args[0] != "item" || args[1] != "create" {
		t.Errorf("expected 'item create', got %v", args[:2])
	}

	// Verify assignments are present
	found := false
	for _, a := range args {
		if a == "access key id[text]=AKIA..." {
			found = true
		}
	}
	if !found {
		t.Error("expected assignment in args")
	}
}

func TestCreateItem_NoTags(t *testing.T) {
	mock := &mockCommander{}
	client := &CLIClient{Cmd: mock}

	err := client.CreateItem("acct", "Private", "Login", "item", "", "field=val")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	args := mock.Calls[0]
	for _, a := range args {
		if a == "--tags" {
			t.Error("did not expect --tags flag when tags is empty")
		}
	}
}
