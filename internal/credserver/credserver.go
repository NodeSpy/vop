// Package credserver implements an ECS-compatible container credential
// endpoint. AWS SDKs natively support this protocol via the
// AWS_CONTAINER_CREDENTIALS_FULL_URI environment variable.
//
// The server supports two modes of credential loading:
//   - On-demand: GET /creds/<profile> fetches from 1Password if no cache exists
//   - Push: POST /creds/<profile> accepts credentials from vop commands
//
// Push is the primary mode when the server runs as a daemon. Commands like
// vop shell, exec, auth push credentials after authenticating interactively.
package credserver

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/NodeSpy/vop/internal/config"
	"github.com/NodeSpy/vop/internal/creds"
	"github.com/NodeSpy/vop/internal/op"
	"github.com/NodeSpy/vop/internal/ui"
)

// ecsCredentialResponse matches the JSON contract expected by AWS SDKs
// when using the container credential provider.
type ecsCredentialResponse struct {
	AccessKeyID     string `json:"AccessKeyId"`
	SecretAccessKey string `json:"SecretAccessKey"`
	Token           string `json:"Token"`
	Expiration      string `json:"Expiration"`
}

// ServerInfo is persisted to disk so other vop commands can find a running server.
type ServerInfo struct {
	PID   int    `json:"pid"`
	Port  int    `json:"port"`
	Token string `json:"token"`
}

// ServerInfoPath returns the path to the server info file.
func ServerInfoPath() string {
	return filepath.Join(config.DefaultConfigDir(), "serve.json")
}

// LoadServerInfo reads the server info file. Returns nil if not found.
func LoadServerInfo() *ServerInfo {
	data, err := os.ReadFile(ServerInfoPath())
	if err != nil {
		return nil
	}
	var info ServerInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil
	}
	// Check if the process is still running.
	if !processRunning(info.PID) {
		_ = os.Remove(ServerInfoPath())
		return nil
	}
	return &info
}

// SaveServerInfo writes the server info file.
func SaveServerInfo(info *ServerInfo) error {
	dir := filepath.Dir(ServerInfoPath())
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return os.WriteFile(ServerInfoPath(), data, 0600)
}

// RemoveServerInfo deletes the server info file.
func RemoveServerInfo() {
	_ = os.Remove(ServerInfoPath())
}

// processRunning checks if a PID is alive.
func processRunning(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 tests if the process exists without actually signaling it.
	return p.Signal(syscall.Signal(0)) == nil
}

// profileCache holds cached credentials for a profile.
type profileCache struct {
	mu         sync.Mutex
	creds      *creds.AWSCredentials
	expiration time.Time
}

// Server is the ECS-compatible credential endpoint.
type Server struct {
	listener  net.Listener
	authToken string
	config    *config.Config
	opClient  op.Client
	cache     sync.Map // map[profileName]*profileCache
	server    http.Server
	Port      int
}

// NewServer creates a credential server on the given port.
// If authToken is empty, a random token is generated.
func NewServer(cfg *config.Config, opClient op.Client, authToken string, port int) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, err
	}

	if authToken == "" {
		authToken = generateToken()
	}

	s := &Server{
		listener:  listener,
		authToken: authToken,
		config:    cfg,
		opClient:  opClient,
		Port:      port,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRoot)
	mux.HandleFunc("/creds/", s.handleCreds)
	mux.HandleFunc("/status", s.handleStatus)
	s.server.Handler = s.withAuth(withLogging(mux))

	return s, nil
}

// Addr returns the address the server is listening on.
func (s *Server) Addr() string {
	return s.listener.Addr().String()
}

// AuthToken returns the authorization token for this server.
func (s *Server) AuthToken() string {
	return s.authToken
}

// Serve starts accepting connections. Blocks until the server is closed.
func (s *Server) Serve() error {
	return s.server.Serve(s.listener)
}

// Close shuts down the server.
func (s *Server) Close() error {
	return s.server.Close()
}

// PutCreds stores credentials in the cache directly (used by push).
func (s *Server) PutCreds(profileName string, awsCreds *creds.AWSCredentials) {
	v, _ := s.cache.LoadOrStore(profileName, &profileCache{})
	pc := v.(*profileCache)

	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.creds = awsCreds
	pc.expiration = time.Time{}
	if awsCreds.Expiration != "" {
		if exp, err := time.Parse(time.RFC3339, awsCreds.Expiration); err == nil {
			pc.expiration = exp
		} else if exp, err := time.Parse("2006-01-02T15:04:05Z", awsCreds.Expiration); err == nil {
			pc.expiration = exp
		}
	}
}

// handleRoot returns a list of available profiles and their cache status.
func (s *Server) handleRoot(w http.ResponseWriter, _ *http.Request) {
	names := s.config.ProfileNames()
	profiles := make(map[string]string, len(names))
	for _, name := range names {
		if v, ok := s.cache.Load(name); ok {
			pc := v.(*profileCache)
			pc.mu.Lock()
			if pc.creds != nil {
				if pc.expiration.IsZero() {
					profiles[name] = "loaded"
				} else if time.Until(pc.expiration) > 0 {
					profiles[name] = "loaded (expires: " + pc.expiration.Format(time.RFC3339) + ")"
				} else {
					profiles[name] = "expired"
				}
			} else {
				profiles[name] = "not loaded"
			}
			pc.mu.Unlock()
		} else {
			profiles[name] = "not loaded"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"profiles": profiles,
	})
}

// handleStatus returns basic server health info.
func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleCreds handles both GET (fetch) and POST (push) for credentials.
func (s *Server) handleCreds(w http.ResponseWriter, r *http.Request) {
	profileName := strings.TrimPrefix(r.URL.Path, "/creds/")
	profileName = strings.TrimSuffix(profileName, "/")

	if profileName == "" {
		writeError(w, "profile name required: use /creds/<profile>", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleCredsGet(w, profileName)
	case http.MethodPost, http.MethodPut:
		s.handleCredsPush(w, r, profileName)
	default:
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleCredsGet returns cached credentials or fetches fresh ones.
func (s *Server) handleCredsGet(w http.ResponseWriter, profileName string) {
	profile, ok := s.config.Profiles[profileName]
	if !ok {
		writeError(w, fmt.Sprintf("unknown profile: %s", profileName), http.StatusNotFound)
		return
	}

	awsCreds, err := s.getCreds(profileName, profile)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := ecsCredentialResponse{
		AccessKeyID:     awsCreds.AccessKeyID,
		SecretAccessKey: awsCreds.SecretAccessKey,
		Token:           awsCreds.SessionToken,
		Expiration:      awsCreds.Expiration,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("credserver: failed to encode response: %v", err)
	}
}

// handleCredsPush accepts credentials pushed from vop commands.
func (s *Server) handleCredsPush(w http.ResponseWriter, r *http.Request, profileName string) {
	var awsCreds creds.AWSCredentials
	if err := json.NewDecoder(r.Body).Decode(&awsCreds); err != nil {
		writeError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	s.PutCreds(profileName, &awsCreds)
	log.Printf("credserver: cached credentials for '%s'", profileName)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "profile": profileName})
}

// getCreds returns cached credentials or fetches fresh ones.
func (s *Server) getCreds(profileName string, profile *config.Profile) (*creds.AWSCredentials, error) {
	v, _ := s.cache.LoadOrStore(profileName, &profileCache{})
	pc := v.(*profileCache)

	pc.mu.Lock()
	defer pc.mu.Unlock()

	// Return cached credentials if still valid (with 60s buffer).
	if pc.creds != nil {
		if pc.expiration.IsZero() || time.Until(pc.expiration) > 60*time.Second {
			return pc.creds, nil
		}
	}

	// No op client available (daemon mode) — can't fetch ourselves.
	if s.opClient == nil {
		return nil, fmt.Errorf("no credentials loaded for '%s': run 'vop auth %s' to authenticate", profileName, profileName)
	}

	// Fetch fresh credentials.
	ui.Quiet = true
	awsCreds, err := creds.Fetch(profile, profileName, s.opClient)
	ui.Quiet = false
	if err != nil {
		return nil, fmt.Errorf("failed to fetch credentials for '%s': %w", profileName, err)
	}

	pc.creds = awsCreds
	if awsCreds.Expiration != "" {
		if exp, parseErr := time.Parse(time.RFC3339, awsCreds.Expiration); parseErr == nil {
			pc.expiration = exp
		} else if exp, parseErr := time.Parse("2006-01-02T15:04:05Z", awsCreds.Expiration); parseErr == nil {
			pc.expiration = exp
		}
	}

	return awsCreds, nil
}

// withAuth wraps a handler to require the Authorization header.
func (s *Server) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != s.authToken {
			writeError(w, "invalid authorization token", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("credserver: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

func writeError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"Message": msg})
}

func generateToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
