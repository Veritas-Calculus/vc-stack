package orchestration

import "time"

// --- Constants ---

const (
	// Stack statuses.
	StackStatusCreateInProgress   = "CREATE_IN_PROGRESS"
	StackStatusCreateComplete     = "CREATE_COMPLETE"
	StackStatusCreateFailed       = "CREATE_FAILED"
	StackStatusUpdateInProgress   = "UPDATE_IN_PROGRESS"
	StackStatusUpdateComplete     = "UPDATE_COMPLETE"
	StackStatusUpdateFailed       = "UPDATE_FAILED"
	StackStatusDeleteInProgress   = "DELETE_IN_PROGRESS"
	StackStatusDeleteComplete     = "DELETE_COMPLETE"
	StackStatusDeleteFailed       = "DELETE_FAILED"
	StackStatusRollbackInProgress = "ROLLBACK_IN_PROGRESS"
	StackStatusRollbackComplete   = "ROLLBACK_COMPLETE"

	// Resource statuses.
	ResourceStatusCreateInProgress = "CREATE_IN_PROGRESS"
	ResourceStatusCreateComplete   = "CREATE_COMPLETE"
	ResourceStatusCreateFailed     = "CREATE_FAILED"
	ResourceStatusDeleteInProgress = "DELETE_IN_PROGRESS"
	ResourceStatusDeleteComplete   = "DELETE_COMPLETE"

	// Event types.
	EventCreate = "CREATE"
	EventUpdate = "UPDATE"
	EventDelete = "DELETE"

	// Supported resource types.
	ResourceTypeInstance      = "VC::Compute::Instance"
	ResourceTypeVolume        = "VC::Storage::Volume"
	ResourceTypeNetwork       = "VC::Network::Network"
	ResourceTypeSubnet        = "VC::Network::Subnet"
	ResourceTypeSecurityGroup = "VC::Network::SecurityGroup"
	ResourceTypeFloatingIP    = "VC::Network::FloatingIP"
	ResourceTypeDNSZone       = "VC::DNS::Zone"
	ResourceTypeDNSRecord     = "VC::DNS::RecordSet"
	ResourceTypeBucket        = "VC::ObjectStorage::Bucket"
	ResourceTypeRouter        = "VC::Network::Router"
	ResourceTypeKeypair       = "VC::Compute::KeyPair"

	// Maximum template size.
	MaxTemplateSize = 512 * 1024 // 512 KB
)

// --- Models ---

// Stack represents a deployed set of resources from a template.
type Stack struct {
	ID              string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name            string    `json:"name" gorm:"not null;index"`
	Description     string    `json:"description"`
	ProjectID       string    `json:"project_id" gorm:"type:varchar(36);index"`
	Status          string    `json:"status" gorm:"default:'CREATE_IN_PROGRESS';index"`
	StatusReason    string    `json:"status_reason"`
	Template        string    `json:"-" gorm:"type:text"` // Raw YAML template (hidden from list)
	TemplateDesc    string    `json:"template_description,omitempty"`
	Parameters      string    `json:"parameters,omitempty" gorm:"type:text"` // JSON key-value
	Outputs         string    `json:"outputs,omitempty" gorm:"type:text"`    // JSON outputs
	Tags            string    `json:"tags,omitempty"`
	Timeout         int       `json:"timeout_mins" gorm:"column:timeout_mins;default:60"` // Deployment timeout in minutes
	DisableRollback bool      `json:"disable_rollback" gorm:"default:false"`
	ParentID        string    `json:"parent_id,omitempty" gorm:"type:varchar(36)"` // For nested stacks
	ResourceCount   int       `json:"resource_count" gorm:"default:0"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (Stack) TableName() string { return "orchestration_stacks" }

// StackResource represents a single resource within a stack.
type StackResource struct {
	ID           string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	StackID      string    `json:"stack_id" gorm:"type:varchar(36);index"`
	LogicalID    string    `json:"logical_id"`            // Name in template (e.g., "web_server")
	PhysicalID   string    `json:"physical_id,omitempty"` // Actual resource ID after creation
	Type         string    `json:"type"`                  // e.g., "VC::Compute::Instance"
	Status       string    `json:"status" gorm:"default:'CREATE_IN_PROGRESS'"`
	StatusReason string    `json:"status_reason"`
	Properties   string    `json:"properties,omitempty" gorm:"type:text"` // JSON properties
	DependsOn    string    `json:"depends_on,omitempty"`                  // Comma-separated logical IDs
	RequiredBy   string    `json:"required_by,omitempty"`                 // Comma-separated logical IDs
	Metadata     string    `json:"metadata,omitempty" gorm:"type:text"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (StackResource) TableName() string { return "orchestration_resources" }

// StackEvent records lifecycle events for a stack or resource.
type StackEvent struct {
	ID           uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	StackID      string    `json:"stack_id" gorm:"type:varchar(36);index"`
	ResourceID   string    `json:"resource_id,omitempty" gorm:"type:varchar(36)"`
	LogicalID    string    `json:"logical_id,omitempty"`
	ResourceType string    `json:"resource_type,omitempty"`
	EventType    string    `json:"event_type"` // CREATE, UPDATE, DELETE
	Status       string    `json:"status"`
	StatusReason string    `json:"status_reason"`
	PhysicalID   string    `json:"physical_id,omitempty"`
	Timestamp    time.Time `json:"timestamp" gorm:"autoCreateTime"`
}

func (StackEvent) TableName() string { return "orchestration_events" }

// StackTemplate represents a reusable, versioned template.
type StackTemplate struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name        string    `json:"name" gorm:"not null;index"`
	Description string    `json:"description"`
	ProjectID   string    `json:"project_id" gorm:"type:varchar(36);index"`
	Version     string    `json:"version" gorm:"default:'1.0'"`
	Template    string    `json:"template" gorm:"type:text"` // YAML content
	IsPublic    bool      `json:"is_public" gorm:"default:false"`
	Category    string    `json:"category,omitempty"` // web, database, network, etc.
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (StackTemplate) TableName() string { return "orchestration_templates" }

// --- Template Schema ---

// TemplateResource defines a resource in a parsed template.
type TemplateResource struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	DependsOn  []string               `json:"depends_on,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ParsedTemplate represents a parsed stack template.
type ParsedTemplate struct {
	Description string                      `json:"description"`
	Parameters  map[string]TemplateParam    `json:"parameters,omitempty"`
	Resources   map[string]TemplateResource `json:"resources"`
	Outputs     map[string]TemplateOutput   `json:"outputs,omitempty"`
}

// TemplateParam defines a template parameter.
type TemplateParam struct {
	Type        string      `json:"type"`
	Default     interface{} `json:"default,omitempty"`
	Description string      `json:"description,omitempty"`
	Constraints []string    `json:"constraints,omitempty"`
}

// TemplateOutput defines a template output value.
type TemplateOutput struct {
	Description string      `json:"description,omitempty"`
	Value       interface{} `json:"value"`
}

// --- Request/Response Types ---

type CreateStackRequest struct {
	Name            string            `json:"name" binding:"required"`
	Description     string            `json:"description"`
	Template        string            `json:"template" binding:"required"` // JSON template (simplified)
	Parameters      map[string]string `json:"parameters"`
	Tags            string            `json:"tags"`
	TimeoutMins     int               `json:"timeout_mins"`
	DisableRollback bool              `json:"disable_rollback"`
}

type UpdateStackRequest struct {
	Description string            `json:"description"`
	Template    string            `json:"template"`
	Parameters  map[string]string `json:"parameters"`
	Tags        string            `json:"tags"`
	TimeoutMins int               `json:"timeout_mins"`
}

type CreateTemplateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Template    string `json:"template" binding:"required"`
	Version     string `json:"version"`
	IsPublic    bool   `json:"is_public"`
	Category    string `json:"category"`
}

type PreviewResponse struct {
	Resources []PreviewResource `json:"resources"`
	Warnings  []string          `json:"warnings,omitempty"`
}

type PreviewResource struct {
	LogicalID string `json:"logical_id"`
	Type      string `json:"type"`
	Action    string `json:"action"`
	DependsOn string `json:"depends_on,omitempty"`
}
