package compute

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
	"go.uber.org/zap"
)

// ControllerClient handles communication with the vc-management internal APIs.
type ControllerClient struct {
	baseURL       string
	internalToken string
	nodeID        string
	httpClient    *http.Client
	logger        *zap.Logger
}

// NewControllerClient creates a new client for vc-management.
func NewControllerClient(baseURL, internalToken string, logger *zap.Logger) *ControllerClient {
	return &ControllerClient{
		baseURL:       baseURL,
		internalToken: internalToken,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

// do sends an authenticated request to the management plane.
func (c *ControllerClient) do(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := fmt.Sprintf("%s/api/v1/internal%s", c.baseURL, path)
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Token", c.internalToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}

	return resp, nil
}

// SetNodeID sets the assigned node UUID for heartbeats.
func (c *ControllerClient) SetNodeID(id string) {
	c.nodeID = id
}

// RegisterNode registers this compute node with the management plane.
func (c *ControllerClient) RegisterNode(ctx context.Context, info interface{}, port int, zoneID, clusterID string) (string, error) {
	payload := map[string]interface{}{
		"info":       info,
		"port":       port,
		"zone_id":    zoneID,
		"cluster_id": clusterID,
	}

	resp, err := c.do(ctx, http.MethodPost, "/hosts/register", payload)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("registration failed (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		UUID string `json:"uuid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode registration response: %w", err)
	}

	c.nodeID = result.UUID
	return result.UUID, nil
}

// ReportHeartbeat sends host resource usage and health status.
func (c *ControllerClient) ReportHeartbeat(ctx context.Context, req interface{}) error {
	resp, err := c.do(ctx, http.MethodPost, "/hosts/heartbeat", req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("heartbeat failed with status %d", resp.StatusCode)
	}
	return nil
}

// GetMyInstances retrieves the list of instances assigned to this host.
func (c *ControllerClient) GetMyInstances(ctx context.Context, hostID string) ([]models.Instance, error) {
	path := fmt.Sprintf("/hosts/%s/instances", hostID)
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get instances failed with status %d", resp.StatusCode)
	}

	var instances []models.Instance
	if err := json.NewDecoder(resp.Body).Decode(&instances); err != nil {
		return nil, fmt.Errorf("decode instances: %w", err)
	}
	return instances, nil
}

// GetInstance retrieves detailed information for a single instance.
func (c *ControllerClient) GetInstance(ctx context.Context, uuid string) (*models.Instance, error) {
	path := fmt.Sprintf("/instances/%s", uuid)
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get instance %s failed with status %d", uuid, resp.StatusCode)
	}

	var instance models.Instance
	if err := json.NewDecoder(resp.Body).Decode(&instance); err != nil {
		return nil, fmt.Errorf("decode instance: %w", err)
	}
	return &instance, nil
}

// UpdateInstanceStatus reports state changes of a virtual machine.
func (c *ControllerClient) UpdateInstanceStatus(ctx context.Context, uuid string, updates map[string]interface{}) error {
	path := fmt.Sprintf("/instances/%s/status", uuid)
	resp, err := c.do(ctx, http.MethodPatch, path, updates)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update status failed (%d): %s", resp.StatusCode, string(body))
	}
	return nil
}

// PushMetrics sends node performance metrics to the management plane.
func (c *ControllerClient) PushMetrics(ctx context.Context, hostID string, metrics interface{}) error {
	path := fmt.Sprintf("/monitoring/nodes/%s/metrics", hostID)
	resp, err := c.do(ctx, http.MethodPost, path, metrics)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("push metrics failed with status %d", resp.StatusCode)
	}
	return nil
}

// GetMetadataByIP resolves instance metadata by source IP.
func (c *ControllerClient) GetMetadataByIP(ctx context.Context, ip string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/metadata?ip=%s", ip)
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata resolution failed with status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode metadata: %w", err)
	}
	return result, nil
}
