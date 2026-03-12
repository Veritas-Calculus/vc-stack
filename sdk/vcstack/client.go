// Package vcstack provides a Go client SDK for the VC Stack IaaS platform.
//
// It supports two authentication modes:
//   - JWT Bearer token (obtained via Login)
//   - HMAC-SHA256 API key (for service accounts / automation)
//
// Quick start:
//
//	client := vcstack.NewClient("https://vc.example.com/api")
//	client.SetAPIKey("VC-AKIA-0123456789abcdef", "your-secret-key")
//	instances, err := client.Instances.List(ctx)
package vcstack

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// Client is the top-level VC Stack API client.
type Client struct {
	// BaseURL is the API base (e.g. "https://vc.example.com/api").
	BaseURL    string
	HTTPClient *http.Client

	// Auth
	bearerToken string
	accessKeyID string
	secretKey   string

	// Resource clients
	Instances       *InstanceClient
	Flavors         *FlavorClient
	Images          *ImageClient
	Volumes         *VolumeClient
	Networks        *NetworkClient
	Subnets         *SubnetClient
	SecurityGroups  *SecurityGroupClient
	FloatingIPs     *FloatingIPClient
	SSHKeys         *SSHKeyClient
	ServiceAccounts *ServiceAccountClient
}

// NewClient creates a new VC Stack API client.
func NewClient(baseURL string) *Client {
	c := &Client{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
	c.Instances = &InstanceClient{c: c}
	c.Flavors = &FlavorClient{c: c}
	c.Images = &ImageClient{c: c}
	c.Volumes = &VolumeClient{c: c}
	c.Networks = &NetworkClient{c: c}
	c.Subnets = &SubnetClient{c: c}
	c.SecurityGroups = &SecurityGroupClient{c: c}
	c.FloatingIPs = &FloatingIPClient{c: c}
	c.SSHKeys = &SSHKeyClient{c: c}
	c.ServiceAccounts = &ServiceAccountClient{c: c}
	return c
}

// SetToken sets a JWT Bearer token for authentication.
func (c *Client) SetToken(token string) {
	c.bearerToken = token
}

// SetAPIKey sets HMAC-SHA256 API key credentials (service account).
func (c *Client) SetAPIKey(accessKeyID, secretKey string) {
	c.accessKeyID = accessKeyID
	c.secretKey = secretKey
}

// Login authenticates with username/password and stores the JWT token.
func (c *Client) Login(ctx context.Context, username, password string) (*LoginResponse, error) {
	body := map[string]string{"username": username, "password": password}
	var resp LoginResponse
	if err := c.do(ctx, http.MethodPost, "/v1/auth/login", body, &resp); err != nil {
		return nil, err
	}
	c.bearerToken = resp.AccessToken
	return &resp, nil
}

// LoginResponse is returned from a successful login.
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// ──────────────────────────────────────────────────────────────────────
// HTTP Transport
// ──────────────────────────────────────────────────────────────────────

// APIError represents an error response from the API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("vcstack: API error %d: %s", e.StatusCode, e.Message)
}

// do performs an authenticated HTTP request.
func (c *Client) do(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := c.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Sign the request.
	c.signRequest(req, method, path)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		// Try to extract error message.
		var errResp struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(respBody, &errResp)
		msg := errResp.Error
		if msg == "" {
			msg = string(respBody)
		}
		return &APIError{StatusCode: resp.StatusCode, Message: msg}
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

// signRequest adds authentication headers.
func (c *Client) signRequest(req *http.Request, method, path string) {
	if c.accessKeyID != "" && c.secretKey != "" {
		// HMAC-SHA256 API key auth.
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		signingString := c.accessKeyID + "\n" + ts + "\n" + method + "\n" + path
		mac := hmac.New(sha256.New, []byte(c.accessKeyID+":"+c.secretKey))
		mac.Write([]byte(signingString))
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("Authorization", fmt.Sprintf(
			"VC-HMAC-SHA256 AccessKeyId=%s, Timestamp=%s, Signature=%s",
			c.accessKeyID, ts, sig,
		))
	} else if c.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearerToken)
	}
}

// ──────────────────────────────────────────────────────────────────────
// Pagination
// ──────────────────────────────────────────────────────────────────────

// ListOptions specifies optional pagination parameters.
type ListOptions struct {
	Page    int
	PerPage int
}
