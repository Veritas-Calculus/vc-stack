// Package metadata provides an EC2/OpenStack-compatible metadata proxy for
// compute nodes. It resolves the requesting VM by source IP and retrieves
// metadata from the management plane via internal API.
package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// ProxyConfig configures the metadata proxy.
type ProxyConfig struct {
	Logger        *zap.Logger
	Port          string // Listen port, default "8082"
	ControllerURL string
	InternalToken string
}

// ControllerClient defines the subset of methods needed from the compute package
// to avoid circular dependencies (Proxy -> Compute -> ControllerClient -> Proxy).
type ControllerClient interface {
	GetMetadataByIP(ctx context.Context, ip string) (map[string]interface{}, error)
}

// Proxy serves EC2/OpenStack-compatible metadata to VMs.
type Proxy struct {
	controller    ControllerClient
	logger        *zap.Logger
	port          string
	mux           *http.ServeMux
}

// instanceInfo holds the data returned to the VM.
type instanceInfo struct {
	UUID       string
	Name       string
	FlavorName string
	ImageUUID  string
	SSHKey     string
	UserData   string
	IPAddress  string
	Metadata   map[string]interface{}
}

// NewProxy creates a new metadata proxy.
func NewProxy(cfg ProxyConfig) (*Proxy, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	if cfg.Port == "" {
		cfg.Port = "8082"
	}

	p := &Proxy{
		logger: cfg.Logger,
		port:   cfg.Port,
		mux:    http.NewServeMux(),
	}

	// In production, the controller client will be injected.
	// For bootstrapping via NewProxy, we can't easily inject the 'compute.ControllerClient'
	// due to circular imports. The caller (NewNode) is responsible for ensuring
	// compatibility or we can define a local implementation.
	
	p.registerRoutes()
	return p, nil
}

// SetController injects the controller client after initialization to avoid
// circular import issues if needed.
func (p *Proxy) SetController(c ControllerClient) {
	p.controller = c
}

// ListenAndServe starts the metadata proxy HTTP server.
func (p *Proxy) ListenAndServe() error {
	addr := "0.0.0.0:" + p.port
	srv := &http.Server{
		Addr:              addr,
		Handler:           p.mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
	}
	p.logger.Info("metadata proxy starting", zap.String("addr", addr))
	return srv.ListenAndServe()
}

func (p *Proxy) registerRoutes() {
	// EC2-compatible metadata (multiple API versions).
	for _, prefix := range []string{"/latest", "/2009-04-04", "/2007-01-19"} {
		pfx := prefix // capture
		p.mux.HandleFunc(pfx+"/meta-data", p.handleMetaDataDir)
		p.mux.HandleFunc(pfx+"/meta-data/", p.handleMetaData)
		p.mux.HandleFunc(pfx+"/user-data", p.handleUserData)
	}

	// OpenStack-compatible metadata.
	p.mux.HandleFunc("/openstack/latest/meta_data.json", p.handleOpenStackMetaData)
	p.mux.HandleFunc("/openstack/latest/user_data", p.handleUserData)

	// Root index.
	p.mux.HandleFunc("/", p.handleRoot)
}

// resolveInstance finds the instance by the request's source IP address.
func (p *Proxy) resolveInstance(r *http.Request) (*instanceInfo, error) {
	if p.controller == nil {
		return nil, fmt.Errorf("controller client not initialized")
	}

	srcIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		srcIP = r.RemoteAddr
	}

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		srcIP = strings.TrimSpace(strings.Split(xff, ",")[0])
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	data, err := p.controller.GetMetadataByIP(ctx, srcIP)
	if err != nil {
		return nil, fmt.Errorf("lookup failed for IP %s: %w", srcIP, err)
	}

	info := &instanceInfo{
		UUID:       getString(data, "uuid"),
		Name:       getString(data, "name"),
		FlavorName: getString(data, "flavor_name"),
		ImageUUID:  getString(data, "image_uuid"),
		SSHKey:     getString(data, "ssh_key"),
		UserData:   getString(data, "user_data"),
		IPAddress:  getString(data, "ip_address"),
	}

	if m, ok := data["metadata"].(map[string]interface{}); ok {
		info.Metadata = m
	}

	return info, nil
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// handleRoot returns the list of API versions.
func (p *Proxy) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, "1.0\n2007-01-19\n2009-04-04\nlatest\n")
}

// handleMetaDataDir returns the directory listing of metadata keys.
func (p *Proxy) handleMetaDataDir(w http.ResponseWriter, r *http.Request) {
	info, err := p.resolveInstance(r)
	if err != nil {
		p.logger.Debug("metadata: failed to resolve instance",
			zap.String("remote", r.RemoteAddr), zap.Error(err))
		http.Error(w, "instance not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	lines := []string{
		"ami-id",
		"hostname",
		"instance-id",
		"instance-type",
		"local-hostname",
		"local-ipv4",
	}
	if info.SSHKey != "" {
		lines = append(lines, "public-keys/")
	}
	fmt.Fprint(w, strings.Join(lines, "\n")+"\n")
}

// handleMetaData returns individual metadata keys.
func (p *Proxy) handleMetaData(w http.ResponseWriter, r *http.Request) {
	info, err := p.resolveInstance(r)
	if err != nil {
		p.logger.Debug("metadata: failed to resolve instance",
			zap.String("remote", r.RemoteAddr), zap.Error(err))
		http.Error(w, "instance not found", http.StatusNotFound)
		return
	}

	path := r.URL.Path
	var key string
	for _, prefix := range []string{"/latest/meta-data/", "/2009-04-04/meta-data/", "/2007-01-19/meta-data/"} {
		if strings.HasPrefix(path, prefix) {
			key = strings.TrimPrefix(path, prefix)
			break
		}
	}
	key = strings.TrimSuffix(key, "/")

	w.Header().Set("Content-Type", "text/plain")

	switch key {
	case "instance-id":
		fmt.Fprint(w, info.UUID)
	case "local-hostname", "hostname":
		fmt.Fprint(w, info.Name)
	case "ami-id":
		if info.ImageUUID != "" {
			fmt.Fprint(w, info.ImageUUID)
		} else {
			fmt.Fprint(w, info.UUID)
		}
	case "instance-type":
		if info.FlavorName != "" {
			fmt.Fprint(w, info.FlavorName)
		} else {
			fmt.Fprint(w, "unknown")
		}
	case "local-ipv4":
		fmt.Fprint(w, info.IPAddress)
	case "public-keys":
		if info.SSHKey != "" {
			fmt.Fprint(w, "0=default\n")
		} else {
			http.NotFound(w, r)
		}
	case "public-keys/0/openssh-key":
		if info.SSHKey != "" {
			fmt.Fprint(w, info.SSHKey)
		} else {
			http.NotFound(w, r)
		}
	case "":
		p.handleMetaDataDir(w, r)
	default:
		http.NotFound(w, r)
	}
}

// handleUserData returns the user-data for the requesting instance.
func (p *Proxy) handleUserData(w http.ResponseWriter, r *http.Request) {
	info, err := p.resolveInstance(r)
	if err != nil {
		p.logger.Debug("metadata: failed to resolve instance for user-data",
			zap.String("remote", r.RemoteAddr), zap.Error(err))
		http.Error(w, "instance not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	if info.UserData != "" {
		fmt.Fprint(w, info.UserData)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

type openStackMeta struct {
	UUID             string                 `json:"uuid"`
	Hostname         string                 `json:"hostname"`
	Name             string                 `json:"name"`
	LaunchIndex      int                    `json:"launch_index"`
	AvailabilityZone string                 `json:"availability_zone"`
	Meta             map[string]interface{} `json:"meta"`
	PublicKeys       map[string]string      `json:"public_keys,omitempty"`
	Keys             []sshKeyEntry          `json:"keys,omitempty"`
}

type sshKeyEntry struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Data string `json:"data"`
}

func (p *Proxy) handleOpenStackMetaData(w http.ResponseWriter, r *http.Request) {
	info, err := p.resolveInstance(r)
	if err != nil {
		p.logger.Debug("metadata: failed to resolve instance for openstack",
			zap.String("remote", r.RemoteAddr), zap.Error(err))
		http.Error(w, "instance not found", http.StatusNotFound)
		return
	}

	meta := openStackMeta{
		UUID:             info.UUID,
		Hostname:         info.Name,
		Name:             info.Name,
		LaunchIndex:      0,
		AvailabilityZone: "default",
		Meta:             info.Metadata,
	}

	if info.SSHKey != "" {
		meta.PublicKeys = map[string]string{"default": info.SSHKey}
		meta.Keys = []sshKeyEntry{
			{Name: "default", Type: "ssh", Data: info.SSHKey},
		}
	}

	if meta.Meta == nil {
		meta.Meta = map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(meta)
}
