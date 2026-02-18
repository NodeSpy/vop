package credserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/NodeSpy/vop/internal/creds"
)

// Client talks to a running vop credential server.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// NewClient creates a client for the running server, or returns nil
// if no server is running.
func NewClient() *Client {
	info := LoadServerInfo()
	if info == nil {
		return nil
	}
	return &Client{
		baseURL: fmt.Sprintf("http://127.0.0.1:%d", info.Port),
		token:   info.Token,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// NewClientFrom creates a client from explicit connection info.
func NewClientFrom(port int, token string) *Client {
	return &Client{
		baseURL: fmt.Sprintf("http://127.0.0.1:%d", port),
		token:   token,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// Ping checks if the server is reachable.
func (c *Client) Ping() bool {
	req, _ := http.NewRequest("GET", c.baseURL+"/status", nil)
	req.Header.Set("Authorization", c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// PushCreds sends credentials to the server for caching.
func (c *Client) PushCreds(profileName string, awsCreds *creds.AWSCredentials) error {
	body, err := json.Marshal(awsCreds)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.baseURL+"/creds/"+profileName, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach credential server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, msg)
	}
	return nil
}

// FetchCreds retrieves credentials from the server.
func (c *Client) FetchCreds(profileName string) (*creds.AWSCredentials, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/creds/"+profileName, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to reach credential server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, msg)
	}

	// The server returns ECS format; map back to our type.
	var ecs ecsCredentialResponse
	if err := json.NewDecoder(resp.Body).Decode(&ecs); err != nil {
		return nil, err
	}

	return &creds.AWSCredentials{
		AccessKeyID:     ecs.AccessKeyID,
		SecretAccessKey: ecs.SecretAccessKey,
		SessionToken:    ecs.Token,
		Expiration:      ecs.Expiration,
	}, nil
}
