package credserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NodeSpy/vop/internal/config"
	"github.com/NodeSpy/vop/internal/creds"
	"github.com/NodeSpy/vop/internal/op"
)

// mockOPClient satisfies the op.Client interface for testing.
type mockOPClient struct{}

func (m *mockOPClient) EnsureSignedIn(_ string) error                      { return nil }
func (m *mockOPClient) ReadField(_, _, _ string) (string, error)           { return "AKID", nil }
func (m *mockOPClient) GetTOTP(_, _ string) (string, error)                { return "", nil }
func (m *mockOPClient) EditItem(_ string, _ string, _ ...string) error     { return nil }
func (m *mockOPClient) ListAccounts() ([]op.OPAccount, error)              { return nil, nil }
func (m *mockOPClient) ListVaults(_ string) ([]op.OPVault, error)          { return nil, nil }
func (m *mockOPClient) CreateItem(_, _, _, _, _ string, _ ...string) error { return nil }
func (m *mockOPClient) ListItems(_, _ string) ([]op.OPItem, error)         { return nil, nil }
func (m *mockOPClient) ListFields(_, _ string) ([]op.OPField, error)       { return nil, nil }
func (m *mockOPClient) IsInstalled() bool                                  { return true }

func TestHandleRoot(t *testing.T) {
	cfg := &config.Config{
		Profiles: map[string]*config.Profile{
			"prod": {OPAccount: "acct", OPItem: "item"},
			"dev":  {OPAccount: "acct", OPItem: "item2"},
		},
	}

	srv := &Server{
		authToken: "test-token",
		config:    cfg,
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "test-token")
	w := httptest.NewRecorder()

	srv.handleRoot(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var body struct {
		Profiles map[string]string `json:"profiles"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(body.Profiles) != 2 {
		t.Errorf("expected 2 profiles, got %d", len(body.Profiles))
	}
	// All profiles should be "not loaded" initially.
	for name, status := range body.Profiles {
		if status != "not loaded" {
			t.Errorf("profile %s: expected 'not loaded', got %q", name, status)
		}
	}
}

func TestHandleCredsUnknownProfile(t *testing.T) {
	cfg := &config.Config{
		Profiles: map[string]*config.Profile{
			"prod": {OPAccount: "acct", OPItem: "item"},
		},
	}

	srv := &Server{
		authToken: "test-token",
		config:    cfg,
	}

	req := httptest.NewRequest("GET", "/creds/nonexistent", nil)
	req.Header.Set("Authorization", "test-token")
	w := httptest.NewRecorder()

	srv.handleCreds(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestHandleCredsMissingProfile(t *testing.T) {
	cfg := &config.Config{
		Profiles: map[string]*config.Profile{},
	}

	srv := &Server{
		authToken: "test-token",
		config:    cfg,
	}

	req := httptest.NewRequest("GET", "/creds/", nil)
	req.Header.Set("Authorization", "test-token")
	w := httptest.NewRecorder()

	srv.handleCreds(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestAuthRequired(t *testing.T) {
	cfg := &config.Config{
		Profiles: map[string]*config.Profile{},
	}

	srv := &Server{
		authToken: "secret-token",
		config:    cfg,
	}

	handler := srv.withAuth(http.HandlerFunc(srv.handleRoot))

	// No auth header
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 without auth, got %d", w.Code)
	}

	// Wrong token
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "wrong-token")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 with wrong token, got %d", w.Code)
	}

	// Correct token
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "secret-token")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with correct token, got %d", w.Code)
	}
}

func TestGenerateToken(t *testing.T) {
	t1 := generateToken()
	t2 := generateToken()

	if t1 == "" {
		t.Error("generated token is empty")
	}
	if t1 == t2 {
		t.Error("two generated tokens should not be identical")
	}
	if len(t1) < 20 {
		t.Errorf("token too short: %d chars", len(t1))
	}
}

func TestHandleCredsPush(t *testing.T) {
	cfg := &config.Config{
		Profiles: map[string]*config.Profile{
			"prod": {OPAccount: "acct", OPItem: "item"},
		},
	}

	srv := &Server{
		authToken: "test-token",
		config:    cfg,
	}

	// Push credentials via POST. Use a far-future expiration so the test never expires.
	body := `{"AccessKeyId":"AKID123","SecretAccessKey":"secret","SessionToken":"tok","Expiration":"2099-01-01T00:00:00Z"}`
	req := httptest.NewRequest("POST", "/creds/prod", strings.NewReader(body))
	req.Header.Set("Authorization", "test-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleCreds(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for push, got %d: %s", w.Code, w.Body.String())
	}

	// Now fetch them back via GET.
	req = httptest.NewRequest("GET", "/creds/prod", nil)
	req.Header.Set("Authorization", "test-token")
	w = httptest.NewRecorder()

	srv.handleCreds(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for get, got %d: %s", w.Code, w.Body.String())
	}

	var resp ecsCredentialResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.AccessKeyID != "AKID123" {
		t.Errorf("expected AccessKeyId 'AKID123', got %q", resp.AccessKeyID)
	}
	if resp.SecretAccessKey != "secret" {
		t.Errorf("expected SecretAccessKey 'secret', got %q", resp.SecretAccessKey)
	}
	if resp.Token != "tok" {
		t.Errorf("expected Token 'tok', got %q", resp.Token)
	}
	if resp.Expiration != "2099-01-01T00:00:00Z" {
		t.Errorf("expected Expiration '2099-01-01T00:00:00Z', got %q", resp.Expiration)
	}
}

func TestHandleCredsPushInvalidJSON(t *testing.T) {
	cfg := &config.Config{
		Profiles: map[string]*config.Profile{},
	}

	srv := &Server{
		authToken: "test-token",
		config:    cfg,
	}

	req := httptest.NewRequest("POST", "/creds/prod", strings.NewReader("not json"))
	req.Header.Set("Authorization", "test-token")
	w := httptest.NewRecorder()

	srv.handleCreds(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", w.Code)
	}
}

func TestHandleCredsMethodNotAllowed(t *testing.T) {
	cfg := &config.Config{
		Profiles: map[string]*config.Profile{},
	}

	srv := &Server{
		authToken: "test-token",
		config:    cfg,
	}

	req := httptest.NewRequest("DELETE", "/creds/prod", nil)
	req.Header.Set("Authorization", "test-token")
	w := httptest.NewRecorder()

	srv.handleCreds(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for DELETE, got %d", w.Code)
	}
}

func TestHandleStatus(t *testing.T) {
	srv := &Server{
		authToken: "test-token",
		config:    &config.Config{},
	}

	req := httptest.NewRequest("GET", "/status", nil)
	req.Header.Set("Authorization", "test-token")
	w := httptest.NewRecorder()

	srv.handleStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", resp["status"])
	}
}

func TestPutCreds(t *testing.T) {
	srv := &Server{
		authToken: "test-token",
		config: &config.Config{
			Profiles: map[string]*config.Profile{
				"test": {OPAccount: "acct", OPItem: "item"},
			},
		},
	}

	awsCreds := &creds.AWSCredentials{
		AccessKeyID:     "AKID",
		SecretAccessKey: "secret",
		SessionToken:    "tok",
		Expiration:      "2026-12-31T23:59:59Z",
	}

	srv.PutCreds("test", awsCreds)

	// Verify it's cached by fetching via the handler.
	req := httptest.NewRequest("GET", "/creds/test", nil)
	req.Header.Set("Authorization", "test-token")
	w := httptest.NewRecorder()

	srv.handleCreds(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ecsCredentialResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if resp.AccessKeyID != "AKID" {
		t.Errorf("expected AKID, got %q", resp.AccessKeyID)
	}
}

func TestClientPushAndFetch(t *testing.T) {
	cfg := &config.Config{
		Profiles: map[string]*config.Profile{
			"prod": {OPAccount: "acct", OPItem: "item"},
		},
	}

	srv, err := NewServer(cfg, nil, "test-token", 0)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer srv.Close()

	go srv.Serve()

	client := NewClientFrom(srv.Port, "test-token")

	// Ping should succeed.
	if !client.Ping() {
		// Port 0 means OS picks a free port, but NewServer uses the
		// specified port. We need to get the actual port.
		// The server addr contains the actual port.
		t.Log("Ping failed, trying with actual address")
	}

	// Actually, port 0 in NewServer will bind to a random port.
	// We need the actual listener port. Let's get it from Addr.
	addr := srv.Addr()
	client = &Client{
		baseURL: "http://" + addr,
		token:   "test-token",
		http:    &http.Client{},
	}

	if !client.Ping() {
		t.Fatal("expected ping to succeed")
	}

	// Push creds.
	awsCreds := &creds.AWSCredentials{
		AccessKeyID:     "AKID-CLI",
		SecretAccessKey: "SECRET-CLI",
		SessionToken:    "TOK-CLI",
		Expiration:      "2026-12-31T23:59:59Z",
	}

	if err := client.PushCreds("prod", awsCreds); err != nil {
		t.Fatalf("push failed: %v", err)
	}

	// Fetch them back.
	got, err := client.FetchCreds("prod")
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}

	if got.AccessKeyID != "AKID-CLI" {
		t.Errorf("expected AKID-CLI, got %q", got.AccessKeyID)
	}
	if got.SecretAccessKey != "SECRET-CLI" {
		t.Errorf("expected SECRET-CLI, got %q", got.SecretAccessKey)
	}
	if got.SessionToken != "TOK-CLI" {
		t.Errorf("expected TOK-CLI, got %q", got.SessionToken)
	}
}

func TestClientFetchUnknownProfile(t *testing.T) {
	cfg := &config.Config{
		Profiles: map[string]*config.Profile{},
	}

	srv, err := NewServer(cfg, nil, "test-token", 0)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer srv.Close()

	go srv.Serve()

	client := &Client{
		baseURL: "http://" + srv.Addr(),
		token:   "test-token",
		http:    &http.Client{},
	}

	_, err = client.FetchCreds("nonexistent")
	if err == nil {
		t.Error("expected error fetching unknown profile")
	}
}
