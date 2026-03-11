// Package iam provides centralized IAM types, constants, and utilities.
// This is the single source of truth for all VC Stack permission actions.
package iam

// ──────────────────────────────────────────────────────────────────────
// Service Names
// ──────────────────────────────────────────────────────────────────────

const (
	ServiceCompute       = "compute"
	ServiceNetwork       = "network"
	ServiceStorage       = "storage"
	ServiceIAM           = "iam"
	ServiceImage         = "image"
	ServiceDNS           = "dns"
	ServiceObjectStorage = "objectstorage"
	ServiceOrchestration = "orchestration"
	ServiceCaaS          = "caas"
	ServiceHPC           = "hpc"
	ServiceInfra         = "infra"
	ServiceBareMetal     = "baremetal"
	ServiceKMS           = "kms"
	ServiceEncryption    = "encryption"
	ServiceAudit         = "audit"
	ServiceMonitoring    = "monitoring"
	ServiceVPN           = "vpn"
	ServiceBackup        = "backup"
	ServiceCatalog       = "catalog"
	ServiceHA            = "ha"
	ServiceDR            = "dr"
	ServiceSelfHeal      = "selfheal"
	ServiceAutoScale     = "autoscale"
	ServiceNotification  = "notification"
	ServiceQuota         = "quota"
	ServiceTask          = "task"
	ServiceTag           = "tag"
	ServiceConfig        = "config"
)

// ──────────────────────────────────────────────────────────────────────
// Compute Actions
// ──────────────────────────────────────────────────────────────────────

const (
	ComputeListInstances  = "vc:compute:ListInstances"
	ComputeGetInstance    = "vc:compute:GetInstance"
	ComputeCreateInstance = "vc:compute:CreateInstance"
	ComputeUpdateInstance = "vc:compute:UpdateInstance"
	ComputeDeleteInstance = "vc:compute:DeleteInstance"
	ComputeStartInstance  = "vc:compute:StartInstance"
	ComputeStopInstance   = "vc:compute:StopInstance"
	ComputeRebootInstance = "vc:compute:RebootInstance"
	ComputeConsole        = "vc:compute:GetConsole"

	ComputeListFlavors  = "vc:compute:ListFlavors"
	ComputeCreateFlavor = "vc:compute:CreateFlavor"
	ComputeDeleteFlavor = "vc:compute:DeleteFlavor"

	ComputeListSSHKeys  = "vc:compute:ListSSHKeys"
	ComputeCreateSSHKey = "vc:compute:CreateSSHKey"
	ComputeDeleteSSHKey = "vc:compute:DeleteSSHKey"
)

// ──────────────────────────────────────────────────────────────────────
// Storage / Volume Actions
// ──────────────────────────────────────────────────────────────────────

const (
	StorageListVolumes  = "vc:storage:ListVolumes"
	StorageGetVolume    = "vc:storage:GetVolume"
	StorageCreateVolume = "vc:storage:CreateVolume"
	StorageUpdateVolume = "vc:storage:UpdateVolume"
	StorageDeleteVolume = "vc:storage:DeleteVolume"
	StorageAttachVolume = "vc:storage:AttachVolume"
	StorageDetachVolume = "vc:storage:DetachVolume"

	StorageListSnapshots  = "vc:storage:ListSnapshots"
	StorageGetSnapshot    = "vc:storage:GetSnapshot"
	StorageCreateSnapshot = "vc:storage:CreateSnapshot"
	StorageUpdateSnapshot = "vc:storage:UpdateSnapshot"
	StorageDeleteSnapshot = "vc:storage:DeleteSnapshot"
)

// ──────────────────────────────────────────────────────────────────────
// Network Actions
// ──────────────────────────────────────────────────────────────────────

const (
	NetworkListNetworks  = "vc:network:ListNetworks"
	NetworkGetNetwork    = "vc:network:GetNetwork"
	NetworkCreateNetwork = "vc:network:CreateNetwork"
	NetworkUpdateNetwork = "vc:network:UpdateNetwork"
	NetworkDeleteNetwork = "vc:network:DeleteNetwork"

	NetworkListSecurityGroups  = "vc:network:ListSecurityGroups"
	NetworkCreateSecurityGroup = "vc:network:CreateSecurityGroup"
	NetworkUpdateSecurityGroup = "vc:network:UpdateSecurityGroup"
	NetworkDeleteSecurityGroup = "vc:network:DeleteSecurityGroup"

	NetworkListFloatingIPs  = "vc:network:ListFloatingIPs"
	NetworkCreateFloatingIP = "vc:network:CreateFloatingIP"
	NetworkDeleteFloatingIP = "vc:network:DeleteFloatingIP"

	NetworkListRouters  = "vc:network:ListRouters"
	NetworkCreateRouter = "vc:network:CreateRouter"
	NetworkUpdateRouter = "vc:network:UpdateRouter"
	NetworkDeleteRouter = "vc:network:DeleteRouter"
)

// ──────────────────────────────────────────────────────────────────────
// Image Actions
// ──────────────────────────────────────────────────────────────────────

const (
	ImageListImages  = "vc:image:ListImages"
	ImageGetImage    = "vc:image:GetImage"
	ImageCreateImage = "vc:image:CreateImage"
	ImageUpdateImage = "vc:image:UpdateImage"
	ImageDeleteImage = "vc:image:DeleteImage"
)

// ──────────────────────────────────────────────────────────────────────
// DNS Actions
// ──────────────────────────────────────────────────────────────────────

const (
	DNSListZones    = "vc:dns:ListZones"
	DNSGetZone      = "vc:dns:GetZone"
	DNSCreateZone   = "vc:dns:CreateZone"
	DNSUpdateZone   = "vc:dns:UpdateZone"
	DNSDeleteZone   = "vc:dns:DeleteZone"
	DNSListRecords  = "vc:dns:ListRecords"
	DNSCreateRecord = "vc:dns:CreateRecord"
	DNSUpdateRecord = "vc:dns:UpdateRecord"
	DNSDeleteRecord = "vc:dns:DeleteRecord"
)

// ──────────────────────────────────────────────────────────────────────
// IAM Actions
// ──────────────────────────────────────────────────────────────────────

const (
	IAMListUsers     = "vc:iam:ListUsers"
	IAMGetUser       = "vc:iam:GetUser"
	IAMCreateUser    = "vc:iam:CreateUser"
	IAMUpdateUser    = "vc:iam:UpdateUser"
	IAMDeleteUser    = "vc:iam:DeleteUser"
	IAMListRoles     = "vc:iam:ListRoles"
	IAMCreateRole    = "vc:iam:CreateRole"
	IAMUpdateRole    = "vc:iam:UpdateRole"
	IAMDeleteRole    = "vc:iam:DeleteRole"
	IAMListPolicies  = "vc:iam:ListPolicies"
	IAMCreatePolicy  = "vc:iam:CreatePolicy"
	IAMUpdatePolicy  = "vc:iam:UpdatePolicy"
	IAMDeletePolicy  = "vc:iam:DeletePolicy"
	IAMAttachPolicy  = "vc:iam:AttachPolicy"
	IAMDetachPolicy  = "vc:iam:DetachPolicy"
	IAMListProjects  = "vc:iam:ListProjects"
	IAMCreateProject = "vc:iam:CreateProject"
	IAMDeleteProject = "vc:iam:DeleteProject"

	// Service Account Actions (P3)
	IAMListServiceAccounts  = "vc:iam:ListServiceAccounts"
	IAMGetServiceAccount    = "vc:iam:GetServiceAccount"
	IAMCreateServiceAccount = "vc:iam:CreateServiceAccount"
	IAMDeleteServiceAccount = "vc:iam:DeleteServiceAccount"
	IAMRotateServiceAccount = "vc:iam:RotateServiceAccountKey"

	// Group Actions (P5)
	IAMListGroups  = "vc:iam:ListGroups"
	IAMGetGroup    = "vc:iam:GetGroup"
	IAMCreateGroup = "vc:iam:CreateGroup"
	IAMUpdateGroup = "vc:iam:UpdateGroup"
	IAMDeleteGroup = "vc:iam:DeleteGroup"

	// Permission Boundary Actions (P5)
	IAMSetPermissionBoundary    = "vc:iam:SetPermissionBoundary"
	IAMGetPermissionBoundary    = "vc:iam:GetPermissionBoundary"
	IAMDeletePermissionBoundary = "vc:iam:DeletePermissionBoundary"
)

// ──────────────────────────────────────────────────────────────────────
// Infrastructure Actions
// ──────────────────────────────────────────────────────────────────────

const (
	InfraListHosts       = "vc:infra:ListHosts"
	InfraCreateHost      = "vc:infra:CreateHost"
	InfraUpdateHost      = "vc:infra:UpdateHost"
	InfraDeleteHost      = "vc:infra:DeleteHost"
	InfraMaintenanceHost = "vc:infra:SetHostMaintenance"
	InfraListClusters    = "vc:infra:ListClusters"
	InfraCreateCluster   = "vc:infra:CreateCluster"
	InfraUpdateCluster   = "vc:infra:UpdateCluster"
	InfraDeleteCluster   = "vc:infra:DeleteCluster"
)

// ──────────────────────────────────────────────────────────────────────
// KMS / Encryption Actions
// ──────────────────────────────────────────────────────────────────────

const (
	KMSListKeys  = "vc:kms:ListKeys"
	KMSGetKey    = "vc:kms:GetKey"
	KMSCreateKey = "vc:kms:CreateKey"
	KMSUpdateKey = "vc:kms:UpdateKey"
	KMSDeleteKey = "vc:kms:DeleteKey"
	KMSEncrypt   = "vc:kms:Encrypt"
	KMSDecrypt   = "vc:kms:Decrypt"
	KMSRotateKey = "vc:kms:RotateKey"

	EncryptionList   = "vc:encryption:ListConfigs"
	EncryptionGet    = "vc:encryption:GetConfig"
	EncryptionCreate = "vc:encryption:CreateConfig"
	EncryptionUpdate = "vc:encryption:UpdateConfig"
	EncryptionDelete = "vc:encryption:DeleteConfig"
)

// ──────────────────────────────────────────────────────────────────────
// Bare Metal Actions
// ──────────────────────────────────────────────────────────────────────

const (
	BareMetalListNodes     = "vc:baremetal:ListNodes"
	BareMetalGetNode       = "vc:baremetal:GetNode"
	BareMetalCreateNode    = "vc:baremetal:CreateNode"
	BareMetalUpdateNode    = "vc:baremetal:UpdateNode"
	BareMetalDeleteNode    = "vc:baremetal:DeleteNode"
	BareMetalProvisionNode = "vc:baremetal:ProvisionNode"
)

// ──────────────────────────────────────────────────────────────────────
// HA / DR / Backup Actions
// ──────────────────────────────────────────────────────────────────────

const (
	HAListPolicies = "vc:ha:ListPolicies"
	HACreatePolicy = "vc:ha:CreatePolicy"
	HAUpdatePolicy = "vc:ha:UpdatePolicy"
	HADeletePolicy = "vc:ha:DeletePolicy"

	DRListPlans    = "vc:dr:ListPlans"
	DRCreatePlan   = "vc:dr:CreatePlan"
	DRUpdatePlan   = "vc:dr:UpdatePlan"
	DRDeletePlan   = "vc:dr:DeletePlan"
	DRExecuteDrill = "vc:dr:ExecuteDrill"

	BackupListBackups  = "vc:backup:ListBackups"
	BackupCreateBackup = "vc:backup:CreateBackup"
	BackupDeleteBackup = "vc:backup:DeleteBackup"
	BackupRestore      = "vc:backup:Restore"
)

// ──────────────────────────────────────────────────────────────────────
// VPN / Monitoring / Misc Actions
// ──────────────────────────────────────────────────────────────────────

const (
	VPNListConnections  = "vc:vpn:ListConnections"
	VPNCreateConnection = "vc:vpn:CreateConnection"
	VPNUpdateConnection = "vc:vpn:UpdateConnection"
	VPNDeleteConnection = "vc:vpn:DeleteConnection"

	MonitoringGetMetrics = "vc:monitoring:GetMetrics"
	MonitoringListAlerts = "vc:monitoring:ListAlerts"

	CatalogListItems  = "vc:catalog:ListItems"
	CatalogCreateItem = "vc:catalog:CreateItem"
	CatalogDeleteItem = "vc:catalog:DeleteItem"

	NotificationListSubscriptions  = "vc:notification:ListSubscriptions"
	NotificationCreateSubscription = "vc:notification:CreateSubscription"
	NotificationDeleteSubscription = "vc:notification:DeleteSubscription"

	AutoScaleListPolicies = "vc:autoscale:ListPolicies"
	AutoScaleCreatePolicy = "vc:autoscale:CreatePolicy"
	AutoScaleUpdatePolicy = "vc:autoscale:UpdatePolicy"
	AutoScaleDeletePolicy = "vc:autoscale:DeletePolicy"
)

// ──────────────────────────────────────────────────────────────────────
// Action Registry — maps old format (resource:action) to new format
// ──────────────────────────────────────────────────────────────────────

// ActionMapping maps a legacy "resource:action" permission to its new
// "vc:service:PascalCaseAction" equivalent. Used during dual-write migration.
type ActionMapping struct {
	Legacy      string // e.g. "compute:create"
	New         string // e.g. "vc:compute:CreateInstance"
	Resource    string // e.g. "compute"
	Action      string // e.g. "create"
	Description string
}

// Registry returns all known permission action mappings.
// The old format remains authoritative during migration.
func Registry() []ActionMapping {
	return []ActionMapping{
		// Compute
		{"compute:list", ComputeListInstances, "compute", "list", "List instances"},
		{"compute:get", ComputeGetInstance, "compute", "get", "View instance details"},
		{"compute:create", ComputeCreateInstance, "compute", "create", "Create instances"},
		{"compute:update", ComputeUpdateInstance, "compute", "update", "Update instances"},
		{"compute:delete", ComputeDeleteInstance, "compute", "delete", "Delete instances"},
		{"compute:start", ComputeStartInstance, "compute", "start", "Start instances"},
		{"compute:stop", ComputeStopInstance, "compute", "stop", "Stop instances"},
		{"compute:reboot", ComputeRebootInstance, "compute", "reboot", "Reboot instances"},
		{"compute:console", ComputeConsole, "compute", "console", "Access instance console"},

		// Flavor
		{"flavor:list", ComputeListFlavors, "flavor", "list", "List flavors"},
		{"flavor:create", ComputeCreateFlavor, "flavor", "create", "Create flavors"},
		{"flavor:delete", ComputeDeleteFlavor, "flavor", "delete", "Delete flavors"},

		// Image
		{"image:list", ImageListImages, "image", "list", "List images"},
		{"image:get", ImageGetImage, "image", "get", "View image details"},
		{"image:create", ImageCreateImage, "image", "create", "Upload images"},
		{"image:update", ImageUpdateImage, "image", "update", "Update images"},
		{"image:delete", ImageDeleteImage, "image", "delete", "Delete images"},

		// Volume
		{"volume:list", StorageListVolumes, "volume", "list", "List volumes"},
		{"volume:get", StorageGetVolume, "volume", "get", "View volume details"},
		{"volume:create", StorageCreateVolume, "volume", "create", "Create volumes"},
		{"volume:update", StorageUpdateVolume, "volume", "update", "Update volumes"},
		{"volume:delete", StorageDeleteVolume, "volume", "delete", "Delete volumes"},
		{"volume:attach", StorageAttachVolume, "volume", "attach", "Attach volumes"},
		{"volume:detach", StorageDetachVolume, "volume", "detach", "Detach volumes"},

		// Snapshot
		{"snapshot:list", StorageListSnapshots, "snapshot", "list", "List snapshots"},
		{"snapshot:get", StorageGetSnapshot, "snapshot", "get", "View snapshot details"},
		{"snapshot:create", StorageCreateSnapshot, "snapshot", "create", "Create snapshots"},
		{"snapshot:update", StorageUpdateSnapshot, "snapshot", "update", "Update snapshots"},
		{"snapshot:delete", StorageDeleteSnapshot, "snapshot", "delete", "Delete snapshots"},

		// Network
		{"network:list", NetworkListNetworks, "network", "list", "List networks"},
		{"network:get", NetworkGetNetwork, "network", "get", "View network details"},
		{"network:create", NetworkCreateNetwork, "network", "create", "Create networks"},
		{"network:update", NetworkUpdateNetwork, "network", "update", "Update networks"},
		{"network:delete", NetworkDeleteNetwork, "network", "delete", "Delete networks"},

		// Security Group
		{"security_group:list", NetworkListSecurityGroups, "security_group", "list", "List security groups"},
		{"security_group:create", NetworkCreateSecurityGroup, "security_group", "create", "Create security groups"},
		{"security_group:update", NetworkUpdateSecurityGroup, "security_group", "update", "Update security group rules"},
		{"security_group:delete", NetworkDeleteSecurityGroup, "security_group", "delete", "Delete security groups"},

		// Floating IP
		{"floating_ip:list", NetworkListFloatingIPs, "floating_ip", "list", "List floating IPs"},
		{"floating_ip:create", NetworkCreateFloatingIP, "floating_ip", "create", "Allocate floating IPs"},
		{"floating_ip:delete", NetworkDeleteFloatingIP, "floating_ip", "delete", "Release floating IPs"},

		// Router
		{"router:list", NetworkListRouters, "router", "list", "List routers"},
		{"router:create", NetworkCreateRouter, "router", "create", "Create routers"},
		{"router:update", NetworkUpdateRouter, "router", "update", "Update routers"},
		{"router:delete", NetworkDeleteRouter, "router", "delete", "Delete routers"},

		// DNS
		{"dns_zone:list", DNSListZones, "dns_zone", "list", "List DNS zones"},
		{"dns_zone:get", DNSGetZone, "dns_zone", "get", "View DNS zone details"},
		{"dns_zone:create", DNSCreateZone, "dns_zone", "create", "Create DNS zones"},
		{"dns_zone:update", DNSUpdateZone, "dns_zone", "update", "Update DNS zones"},
		{"dns_zone:delete", DNSDeleteZone, "dns_zone", "delete", "Delete DNS zones"},
		{"dns_record:list", DNSListRecords, "dns_record", "list", "List DNS records"},
		{"dns_record:create", DNSCreateRecord, "dns_record", "create", "Create DNS records"},
		{"dns_record:update", DNSUpdateRecord, "dns_record", "update", "Update DNS records"},
		{"dns_record:delete", DNSDeleteRecord, "dns_record", "delete", "Delete DNS records"},

		// IAM
		{"user:list", IAMListUsers, "user", "list", "List users"},
		{"user:get", IAMGetUser, "user", "get", "View user details"},
		{"user:create", IAMCreateUser, "user", "create", "Create users"},
		{"user:update", IAMUpdateUser, "user", "update", "Update users"},
		{"user:delete", IAMDeleteUser, "user", "delete", "Delete users"},
		{"role:list", IAMListRoles, "role", "list", "List roles"},
		{"role:create", IAMCreateRole, "role", "create", "Create roles"},
		{"role:update", IAMUpdateRole, "role", "update", "Update roles"},
		{"role:delete", IAMDeleteRole, "role", "delete", "Delete roles"},
		{"policy:list", IAMListPolicies, "policy", "list", "List policies"},
		{"policy:create", IAMCreatePolicy, "policy", "create", "Create policies"},
		{"policy:update", IAMUpdatePolicy, "policy", "update", "Update policies"},
		{"policy:delete", IAMDeletePolicy, "policy", "delete", "Delete policies"},
		{"project:list", IAMListProjects, "project", "list", "List projects"},
		{"project:create", IAMCreateProject, "project", "create", "Create projects"},
		{"project:delete", IAMDeleteProject, "project", "delete", "Delete projects"},

		// Service Accounts (P3)
		{"service_account:list", IAMListServiceAccounts, "service_account", "list", "List service accounts"},
		{"service_account:get", IAMGetServiceAccount, "service_account", "get", "View service account details"},
		{"service_account:create", IAMCreateServiceAccount, "service_account", "create", "Create service accounts"},
		{"service_account:delete", IAMDeleteServiceAccount, "service_account", "delete", "Delete service accounts"},
		{"service_account:rotate", IAMRotateServiceAccount, "service_account", "rotate", "Rotate service account keys"},

		// Groups (P5)
		{"group:list", IAMListGroups, "group", "list", "List groups"},
		{"group:get", IAMGetGroup, "group", "get", "View group details"},
		{"group:create", IAMCreateGroup, "group", "create", "Create groups"},
		{"group:update", IAMUpdateGroup, "group", "update", "Update groups"},
		{"group:delete", IAMDeleteGroup, "group", "delete", "Delete groups"},

		// Permission Boundaries (P5)
		{"permission_boundary:set", IAMSetPermissionBoundary, "permission_boundary", "set", "Set permission boundary"},
		{"permission_boundary:get", IAMGetPermissionBoundary, "permission_boundary", "get", "View permission boundary"},
		{"permission_boundary:delete", IAMDeletePermissionBoundary, "permission_boundary", "delete", "Delete permission boundary"},

		// Infrastructure
		{"host:list", InfraListHosts, "host", "list", "List hosts"},
		{"host:create", InfraCreateHost, "host", "create", "Register hosts"},
		{"host:update", InfraUpdateHost, "host", "update", "Update host configuration"},
		{"host:delete", InfraDeleteHost, "host", "delete", "Deregister hosts"},
		{"host:maintenance", InfraMaintenanceHost, "host", "maintenance", "Toggle host maintenance mode"},
		{"cluster:list", InfraListClusters, "cluster", "list", "List clusters"},
		{"cluster:create", InfraCreateCluster, "cluster", "create", "Create clusters"},
		{"cluster:update", InfraUpdateCluster, "cluster", "update", "Update clusters"},
		{"cluster:delete", InfraDeleteCluster, "cluster", "delete", "Delete clusters"},

		// KMS
		{"kms:list", KMSListKeys, "kms", "list", "List KMS keys"},
		{"kms:get", KMSGetKey, "kms", "get", "View KMS key details"},
		{"kms:create", KMSCreateKey, "kms", "create", "Create KMS keys"},
		{"kms:update", KMSUpdateKey, "kms", "update", "Update KMS keys"},
		{"kms:delete", KMSDeleteKey, "kms", "delete", "Delete KMS keys"},

		// Encryption
		{"encryption:list", EncryptionList, "encryption", "list", "List encryption configs"},
		{"encryption:get", EncryptionGet, "encryption", "get", "View encryption config"},
		{"encryption:create", EncryptionCreate, "encryption", "create", "Create encryption configs"},
		{"encryption:update", EncryptionUpdate, "encryption", "update", "Update encryption configs"},
		{"encryption:delete", EncryptionDelete, "encryption", "delete", "Delete encryption configs"},

		// Bare Metal
		{"baremetal:list", BareMetalListNodes, "baremetal", "list", "List bare metal nodes"},
		{"baremetal:get", BareMetalGetNode, "baremetal", "get", "View bare metal node details"},
		{"baremetal:create", BareMetalCreateNode, "baremetal", "create", "Register bare metal nodes"},
		{"baremetal:update", BareMetalUpdateNode, "baremetal", "update", "Update bare metal nodes"},
		{"baremetal:delete", BareMetalDeleteNode, "baremetal", "delete", "Deregister bare metal nodes"},

		// HA
		{"ha:list", HAListPolicies, "ha", "list", "List HA policies"},
		{"ha:get", HAListPolicies, "ha", "get", "View HA policy details"},
		{"ha:create", HACreatePolicy, "ha", "create", "Create HA policies"},
		{"ha:update", HAUpdatePolicy, "ha", "update", "Update HA policies"},
		{"ha:delete", HADeletePolicy, "ha", "delete", "Delete HA policies"},

		// DR
		{"dr:list", DRListPlans, "dr", "list", "List DR plans"},
		{"dr:get", DRListPlans, "dr", "get", "View DR plan details"},
		{"dr:create", DRCreatePlan, "dr", "create", "Create DR plans"},
		{"dr:update", DRUpdatePlan, "dr", "update", "Update DR plans"},
		{"dr:delete", DRDeletePlan, "dr", "delete", "Delete DR plans"},

		// Backup
		{"backup:list", BackupListBackups, "backup", "list", "List backups"},
		{"backup:get", BackupListBackups, "backup", "get", "View backup details"},
		{"backup:create", BackupCreateBackup, "backup", "create", "Create backups"},
		{"backup:delete", BackupDeleteBackup, "backup", "delete", "Delete backups"},

		// VPN
		{"vpn:list", VPNListConnections, "vpn", "list", "List VPN connections"},
		{"vpn:get", VPNListConnections, "vpn", "get", "View VPN connection details"},
		{"vpn:create", VPNCreateConnection, "vpn", "create", "Create VPN connections"},
		{"vpn:update", VPNUpdateConnection, "vpn", "update", "Update VPN connections"},
		{"vpn:delete", VPNDeleteConnection, "vpn", "delete", "Delete VPN connections"},

		// Monitoring
		{"monitoring:list", MonitoringListAlerts, "monitoring", "list", "List monitoring alerts"},
		{"monitoring:get", MonitoringGetMetrics, "monitoring", "get", "View monitoring metrics"},
		{"monitoring:create", MonitoringGetMetrics, "monitoring", "create", "Create monitoring configs"},

		// Catalog
		{"catalog:list", CatalogListItems, "catalog", "list", "List catalog items"},
		{"catalog:get", CatalogListItems, "catalog", "get", "View catalog item details"},
		{"catalog:create", CatalogCreateItem, "catalog", "create", "Create catalog items"},
		{"catalog:update", CatalogCreateItem, "catalog", "update", "Update catalog items"},
		{"catalog:delete", CatalogDeleteItem, "catalog", "delete", "Delete catalog items"},

		// Notification
		{"notification:list", NotificationListSubscriptions, "notification", "list", "List notification subscriptions"},
		{"notification:create", NotificationCreateSubscription, "notification", "create", "Create subscriptions"},
		{"notification:update", NotificationCreateSubscription, "notification", "update", "Update subscriptions"},
		{"notification:delete", NotificationDeleteSubscription, "notification", "delete", "Delete subscriptions"},

		// Auto Scale
		{"autoscale:list", AutoScaleListPolicies, "autoscale", "list", "List autoscale policies"},
		{"autoscale:get", AutoScaleListPolicies, "autoscale", "get", "View autoscale policy details"},
		{"autoscale:create", AutoScaleCreatePolicy, "autoscale", "create", "Create autoscale policies"},
		{"autoscale:update", AutoScaleUpdatePolicy, "autoscale", "update", "Update autoscale policies"},
		{"autoscale:delete", AutoScaleDeletePolicy, "autoscale", "delete", "Delete autoscale policies"},

		// SelfHeal
		{"selfheal:list", "vc:selfheal:ListPolicies", "selfheal", "list", "List self-heal policies"},
		{"selfheal:get", "vc:selfheal:GetPolicy", "selfheal", "get", "View self-heal policy details"},
		{"selfheal:create", "vc:selfheal:CreatePolicy", "selfheal", "create", "Create self-heal policies"},
		{"selfheal:update", "vc:selfheal:UpdatePolicy", "selfheal", "update", "Update self-heal policies"},
		{"selfheal:delete", "vc:selfheal:DeletePolicy", "selfheal", "delete", "Delete self-heal policies"},

		// Quota
		{"quota:list", "vc:quota:ListQuotas", "quota", "list", "List quotas"},
		{"quota:get", "vc:quota:GetQuota", "quota", "get", "View quota details"},
		{"quota:create", "vc:quota:SetQuota", "quota", "create", "Set quotas"},
		{"quota:update", "vc:quota:UpdateQuota", "quota", "update", "Update quotas"},
		{"quota:delete", "vc:quota:DeleteQuota", "quota", "delete", "Delete quotas"},

		// Storage service
		{"storage:list", "vc:storage:ListResources", "storage", "list", "List storage resources"},
		{"storage:get", "vc:storage:GetResource", "storage", "get", "View storage resource details"},
		{"storage:create", "vc:storage:CreateResource", "storage", "create", "Create storage resources"},
		{"storage:update", "vc:storage:UpdateResource", "storage", "update", "Update storage resources"},
		{"storage:delete", "vc:storage:DeleteResource", "storage", "delete", "Delete storage resources"},

		// Orchestration
		{"orchestration:list", "vc:orchestration:ListStacks", "orchestration", "list", "List stacks"},
		{"orchestration:get", "vc:orchestration:GetStack", "orchestration", "get", "View stack details"},
		{"orchestration:create", "vc:orchestration:CreateStack", "orchestration", "create", "Create stacks"},
		{"orchestration:update", "vc:orchestration:UpdateStack", "orchestration", "update", "Update stacks"},
		{"orchestration:delete", "vc:orchestration:DeleteStack", "orchestration", "delete", "Delete stacks"},

		// Task
		{"task:list", "vc:task:ListTasks", "task", "list", "List tasks"},
		{"task:get", "vc:task:GetTask", "task", "get", "View task details"},
		{"task:create", "vc:task:CreateTask", "task", "create", "Create tasks"},

		// Tag
		{"tag:list", "vc:tag:ListTags", "tag", "list", "List tags"},
		{"tag:get", "vc:tag:GetTag", "tag", "get", "View tag details"},
		{"tag:create", "vc:tag:CreateTag", "tag", "create", "Create tags"},
		{"tag:update", "vc:tag:UpdateTag", "tag", "update", "Update tags"},
		{"tag:delete", "vc:tag:DeleteTag", "tag", "delete", "Delete tags"},

		// Audit
		{"audit:list", "vc:audit:ListLogs", "audit", "list", "List audit logs"},
		{"audit:get", "vc:audit:GetLog", "audit", "get", "View audit log details"},
		{"audit:create", "vc:audit:CreateConfig", "audit", "create", "Create audit configs"},
		{"audit:update", "vc:audit:UpdateConfig", "audit", "update", "Update audit configs"},
		{"audit:delete", "vc:audit:DeleteConfig", "audit", "delete", "Delete audit configs"},

		// Usage
		{"usage:list", "vc:usage:ListRecords", "usage", "list", "List usage records"},
		{"usage:get", "vc:usage:GetRecord", "usage", "get", "View usage details"},

		// Event
		{"event:list", "vc:event:ListEvents", "event", "list", "List events"},
		{"event:get", "vc:event:GetEvent", "event", "get", "View event details"},

		// Metadata
		{"metadata:list", "vc:metadata:ListConfigs", "metadata", "list", "List metadata configs"},
		{"metadata:get", "vc:metadata:GetConfig", "metadata", "get", "View metadata config"},
		{"metadata:create", "vc:metadata:CreateConfig", "metadata", "create", "Create metadata configs"},
		{"metadata:update", "vc:metadata:UpdateConfig", "metadata", "update", "Update metadata configs"},
		{"metadata:delete", "vc:metadata:DeleteConfig", "metadata", "delete", "Delete metadata configs"},

		// Scheduler
		{"scheduler:list", "vc:scheduler:ListRules", "scheduler", "list", "List scheduler rules"},
		{"scheduler:get", "vc:scheduler:GetRule", "scheduler", "get", "View scheduler rule"},
		{"scheduler:create", "vc:scheduler:CreateRule", "scheduler", "create", "Create scheduler rules"},
		{"scheduler:update", "vc:scheduler:UpdateRule", "scheduler", "update", "Update scheduler rules"},
		{"scheduler:delete", "vc:scheduler:DeleteRule", "scheduler", "delete", "Delete scheduler rules"},

		// Config
		{"config:list", "vc:config:ListConfigs", "config", "list", "List configurations"},
		{"config:get", "vc:config:GetConfig", "config", "get", "View configuration"},
		{"config:create", "vc:config:CreateConfig", "config", "create", "Create configurations"},
		{"config:update", "vc:config:UpdateConfig", "config", "update", "Update configurations"},
		{"config:delete", "vc:config:DeleteConfig", "config", "delete", "Delete configurations"},

		// Rate Limit
		{"ratelimit:list", "vc:ratelimit:ListPolicies", "ratelimit", "list", "List rate limit policies"},
		{"ratelimit:get", "vc:ratelimit:GetPolicy", "ratelimit", "get", "View rate limit policy"},
		{"ratelimit:create", "vc:ratelimit:CreatePolicy", "ratelimit", "create", "Create rate limit policies"},
		{"ratelimit:update", "vc:ratelimit:UpdatePolicy", "ratelimit", "update", "Update rate limit policies"},
		{"ratelimit:delete", "vc:ratelimit:DeletePolicy", "ratelimit", "delete", "Delete rate limit policies"},
	}
}

// LegacyToNew returns a lookup map from legacy permission format to new format.
func LegacyToNew() map[string]string {
	m := make(map[string]string, len(Registry()))
	for _, r := range Registry() {
		m[r.Legacy] = r.New
	}
	return m
}

// NewToLegacy returns a lookup map from new permission format to legacy format.
func NewToLegacy() map[string]string {
	m := make(map[string]string, len(Registry()))
	for _, r := range Registry() {
		m[r.New] = r.Legacy
	}
	return m
}
