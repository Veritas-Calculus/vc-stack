package compute

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

type AgentClient struct {
	httpClient    *http.Client
	internalToken string
}

func NewAgentClient(token string) *AgentClient {
	return &AgentClient{
		httpClient:    &http.Client{},
		internalToken: token,
	}
}

func (c *AgentClient) StartVM(ctx context.Context, hostAddr string, inst *models.Instance) error {
	url := fmt.Sprintf("%s/api/v1/agent/vms", hostAddr)
	return c.do(ctx, http.MethodPost, url, inst)
}

func (c *AgentClient) StopVM(ctx context.Context, hostAddr string, uuid string) error {
	url := fmt.Sprintf("%s/api/v1/agent/vms/%s", hostAddr, uuid)
	return c.do(ctx, http.MethodDelete, url, nil)
}

func (c *AgentClient) ConfigureNetwork(ctx context.Context, hostAddr string, bridgeMappings string) error {
	url := fmt.Sprintf("%s/api/v1/agent/network/setup", hostAddr)
	payload := map[string]string{
		"bridge_mappings": bridgeMappings,
	}
	return c.do(ctx, http.MethodPost, url, payload)
}

func (c *AgentClient) GetVNCConsole(ctx context.Context, hostAddr string, uuid string) (string, int, error) {
	url := fmt.Sprintf("%s/api/v1/agent/vms/%s/vnc", hostAddr, uuid)

	resp, err := c.doWithResponse(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	var result struct {
		VNCAddr string `json:"vnc_address"`
		VNCPort int    `json:"vnc_port"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", 0, err
	}
	return result.VNCAddr, result.VNCPort, nil
}

func (c *AgentClient) do(ctx context.Context, method, url string, body interface{}) error {
	resp, err := c.doWithResponse(ctx, method, url, body)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	return nil
}

func (c *AgentClient) doWithResponse(ctx context.Context, method, url string, body interface{}) (*http.Response, error) {
	var reader bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&reader).Encode(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, &reader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Token", c.internalToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("agent error: %s", resp.Status)
	}
	return resp, nil
}
