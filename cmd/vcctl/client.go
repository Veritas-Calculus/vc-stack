package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

// apiClient provides HTTP helpers for vcctl commands.
type apiClient struct {
	baseURL    string
	httpClient *http.Client
}

func newAPIClient() *apiClient {
	base := apiEndpoint
	if base == "" {
		base = os.Getenv("VCSTACK_ENDPOINT")
	}
	if base == "" {
		base = "http://127.0.0.1"
	}
	base = strings.TrimRight(base, "/")

	return &apiClient{
		baseURL:    base,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *apiClient) get(path string) (map[string]interface{}, error) {
	url := c.baseURL + "/api" + path
	req, err := http.NewRequest("GET", url, http.NoBody)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

func (c *apiClient) post(path string, body interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	url := c.baseURL + "/api" + path
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

//nolint:unused // Available for future CLI commands.
func (c *apiClient) put(path string, body interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	url := c.baseURL + "/api" + path
	req, err := http.NewRequest("PUT", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

func (c *apiClient) delete(path string) error {
	url := c.baseURL + "/api" + path
	req, err := http.NewRequest("DELETE", url, http.NoBody)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req) // #nosec
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (c *apiClient) do(req *http.Request) (map[string]interface{}, error) {
	resp, err := c.httpClient.Do(req) // #nosec
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return result, nil
}

// outputJSON prints data as formatted JSON.
func outputJSON(data interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(data)
}

// newTabWriter creates a tab writer for table output.
func newTabWriter() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
}

// getString safely extracts a string from a map.
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

// getFloat safely extracts a number from a map.
func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok && v != nil {
		switch t := v.(type) {
		case float64:
			return t
		case int:
			return float64(t)
		}
	}
	return 0
}
