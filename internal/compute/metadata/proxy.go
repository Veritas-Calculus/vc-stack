// Package metadata provides an EC2/OpenStack-compatible metadata proxy for
// compute nodes. It resolves the requesting VM by source IP (via the
// net_ports table) and returns instance metadata from the database.
//
// This enables cloud-init inside VMs to retrieve instance-id, hostname,
// SSH keys, and user-data without requiring a dedicated metadata service.
package metadata

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ProxyConfig configures the metadata proxy.
type ProxyConfig struct {
	DB     *gorm.DB
	Logger *zap.Logger
	Port   string // Listen port, default "8082"
}

// Proxy serves EC2/OpenStack-compatible metadata to VMs.
type Proxy struct {
	db     *gorm.DB
	logger *zap.Logger
	port   string
	mux    *http.ServeMux
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

// portRow is used for querying the net_ports table.
type portRow struct {
	DeviceID   string         `gorm:"column:device_id"`
	FixedIPs   sql.NullString `gorm:"column:fixed_ips"`
	MACAddress string         `gorm:"column:mac_address"`
}

// instanceRow is used for querying the instances table.
type instanceRow struct {
	UUID      string         `gorm:"column:uuid"`
	Name      string         `gorm:"column:name"`
	SSHKey    string         `gorm:"column:ssh_key"`
	UserData  string         `gorm:"column:user_data"`
	IPAddress string         `gorm:"column:ip_address"`
	Metadata  sql.NullString `gorm:"column:metadata"`
}

// flavorRow is used for querying flavors.
type flavorRow struct {
	Name string `gorm:"column:name"`
}

// imageRow is used for querying images.
type imageRow struct {
	UUID string `gorm:"column:uuid"`
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
		db:     cfg.DB,
		logger: cfg.Logger,
		port:   cfg.Port,
		mux:    http.NewServeMux(),
	}
	p.registerRoutes()
	return p, nil
}

// ListenAndServe starts the metadata proxy HTTP server.
// This blocks, so call it in a goroutine.
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
	srcIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		srcIP = r.RemoteAddr
	}

	// Also check X-Forwarded-For (for DNAT scenarios).
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		srcIP = strings.TrimSpace(strings.Split(xff, ",")[0])
	}

	// Query net_ports: find port where fixed_ips JSON contains this IP.
	var port portRow
	err = p.db.Table("net_ports").
		Select("device_id, fixed_ips, mac_address").
		Where("fixed_ips::text LIKE ?", "%"+srcIP+"%").
		First(&port).Error
	if err != nil {
		return nil, fmt.Errorf("no port found for IP %s: %w", srcIP, err)
	}

	if port.DeviceID == "" {
		return nil, fmt.Errorf("port for IP %s has no device_id", srcIP)
	}

	// Query instances table.
	var inst instanceRow
	err = p.db.Table("instances").
		Select("uuid, name, ssh_key, user_data, ip_address, metadata").
		Where("uuid = ? AND deleted_at IS NULL", port.DeviceID).
		First(&inst).Error
	if err != nil {
		return nil, fmt.Errorf("instance %s not found: %w", port.DeviceID, err)
	}

	info := &instanceInfo{
		UUID:      inst.UUID,
		Name:      inst.Name,
		SSHKey:    inst.SSHKey,
		UserData:  inst.UserData,
		IPAddress: inst.IPAddress,
	}

	// Parse metadata JSON.
	if inst.Metadata.Valid && inst.Metadata.String != "" {
		_ = json.Unmarshal([]byte(inst.Metadata.String), &info.Metadata)
	}

	// Get flavor name.
	var flavor flavorRow
	if err := p.db.Table("instances").
		Select("flavors.name").
		Joins("JOIN flavors ON flavors.id = instances.flavor_id").
		Where("instances.uuid = ?", inst.UUID).
		First(&flavor).Error; err == nil {
		info.FlavorName = flavor.Name
	}

	// Get image UUID.
	var image imageRow
	if err := p.db.Table("instances").
		Select("images.uuid").
		Joins("JOIN images ON images.id = instances.image_id").
		Where("instances.uuid = ?", inst.UUID).
		First(&image).Error; err == nil {
		info.ImageUUID = image.UUID
	}

	return info, nil
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

	// Extract the key from the path (after /meta-data/).
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
		// Directory listing.
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
		// Return empty 200 (not 404) — cloud-init expects this.
		w.WriteHeader(http.StatusOK)
	}
}

// openStackMeta is the JSON format for OpenStack metadata.
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

// handleOpenStackMetaData returns OpenStack-format metadata JSON.
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
