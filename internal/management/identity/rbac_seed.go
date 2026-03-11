package identity

import (
	"go.uber.org/zap"

	"github.com/Veritas-Calculus/vc-stack/pkg/iam"
)

// defaultPermissions is now built from the centralized iam.Registry() plus
// HPC-specific permissions that are not yet in the registry.
// The old "resource:action" format remains the source of truth until full migration.
var defaultPermissions []struct {
	Resource    string
	Action      string
	Description string
}

// hpcPermissions are HPC-specific and not in the generic iam.Registry().
var hpcPermissions = []struct {
	Resource    string
	Action      string
	Description string
}{
	// HPC Kubernetes Clusters
	{"hpc_cluster", "list", "List HPC Kubernetes clusters"},
	{"hpc_cluster", "get", "View HPC cluster details"},
	{"hpc_cluster", "create", "Create HPC Kubernetes clusters"},
	{"hpc_cluster", "update", "Update HPC cluster configuration"},
	{"hpc_cluster", "delete", "Delete HPC Kubernetes clusters"},
	{"hpc_cluster", "scale", "Scale HPC cluster nodes"},
	// HPC GPU Resources
	{"hpc_gpu", "list", "List GPU resources and pools"},
	{"hpc_gpu", "get", "View GPU resource details"},
	{"hpc_gpu", "create", "Create GPU pools"},
	{"hpc_gpu", "delete", "Delete GPU pools"},
	// Slurm Clusters
	{"slurm_cluster", "list", "List Slurm clusters"},
	{"slurm_cluster", "get", "View Slurm cluster details"},
	{"slurm_cluster", "create", "Create Slurm clusters"},
	{"slurm_cluster", "update", "Update Slurm configuration"},
	{"slurm_cluster", "delete", "Delete Slurm clusters"},
	// Slurm Partitions
	{"slurm_partition", "list", "List Slurm partitions"},
	{"slurm_partition", "create", "Create Slurm partitions"},
	{"slurm_partition", "update", "Update Slurm partition settings"},
	{"slurm_partition", "delete", "Delete Slurm partitions"},
	// HPC Jobs (unified)
	{"hpc_job", "list", "List HPC jobs"},
	{"hpc_job", "get", "View HPC job details and logs"},
	{"hpc_job", "create", "Submit HPC jobs"},
	{"hpc_job", "delete", "Cancel or delete HPC jobs"},
	// Slurm Users (admin-only sync)
	{"slurm_user", "list", "List Slurm user mappings"},
	{"slurm_user", "create", "Sync IAM users to Slurm"},
	{"slurm_user", "delete", "Remove Slurm user mappings"},
	// HPC Monitoring
	{"hpc_monitoring", "list", "View HPC metrics and dashboards"},
	// CaaS (Kubernetes as a Service)
	{"caas_cluster", "list", "List CaaS clusters"},
	{"caas_cluster", "get", "View CaaS cluster details"},
	{"caas_cluster", "create", "Create CaaS clusters"},
	{"caas_cluster", "update", "Update CaaS clusters"},
	{"caas_cluster", "delete", "Delete CaaS clusters"},
	{"caas_cluster", "scale", "Scale CaaS cluster nodes"},
	// Object Storage (legacy)
	{"bucket", "list", "List buckets"},
	{"bucket", "create", "Create buckets"},
	{"bucket", "update", "Update bucket settings"},
	{"bucket", "delete", "Delete buckets"},
	{"s3_credential", "list", "List S3 credentials"},
	{"s3_credential", "create", "Create S3 credentials"},
	{"s3_credential", "delete", "Revoke S3 credentials"},
	// Orchestration (legacy names)
	{"stack", "list", "List stacks"},
	{"stack", "get", "View stack details"},
	{"stack", "create", "Create stacks"},
	{"stack", "update", "Update stacks"},
	{"stack", "delete", "Delete stacks"},
	{"template", "list", "List templates"},
	{"template", "create", "Create templates"},
	{"template", "delete", "Delete templates"},
}

func init() {
	// Build defaultPermissions from the centralized iam.Registry.
	type perm = struct {
		Resource    string
		Action      string
		Description string
	}

	// Deduplicate using a set.
	seen := map[string]bool{}
	var perms []perm

	// 1. Add all permissions from the centralized registry.
	for _, m := range iam.Registry() {
		key := m.Resource + ":" + m.Action
		if seen[key] {
			continue
		}
		seen[key] = true
		perms = append(perms, perm{m.Resource, m.Action, m.Description})
	}

	// 2. Add HPC and other non-registry permissions.
	for _, hp := range hpcPermissions {
		key := hp.Resource + ":" + hp.Action
		if seen[key] {
			continue
		}
		seen[key] = true
		perms = append(perms, perm{hp.Resource, hp.Action, hp.Description})
	}

	defaultPermissions = perms
}

// defaultRoles defines the 4 system roles and their permission sets.
// admin: full access (all permissions).
// operator: create + manage resources, no IAM/infra management.
// member: create + manage own resources (limited).
// viewer: read-only access to all resources.
var defaultRoles = map[string]struct {
	Description string
	Actions     []string // allowed actions; match against Permission.Action
}{
	"admin": {
		Description: "Full system access — all resources and operations",
		Actions:     []string{"*"}, // wild-card: gets every permission
	},
	"operator": {
		Description: "Manage cloud resources — compute, storage, network, DNS, orchestration, HPC",
		Actions:     []string{"list", "get", "create", "update", "delete", "start", "stop", "reboot", "console", "attach", "detach", "scale"},
	},
	"member": {
		Description: "Standard project member — create and manage own resources",
		Actions:     []string{"list", "get", "create", "update", "delete", "start", "stop", "reboot", "console", "attach", "detach", "scale"},
	},
	"viewer": {
		Description: "Read-only access to all resources",
		Actions:     []string{"list", "get"},
	},
}

// memberExcludedResources are resources that 'member' role cannot manage.
// Members can submit HPC jobs but cannot manage clusters, partitions, or user sync.
var memberExcludedResources = map[string]bool{
	"user": true, "role": true, "policy": true,
	"host": true, "cluster": true, "flavor": true,
	// HPC management resources — member can submit jobs but not manage infra.
	"hpc_cluster":     true,
	"slurm_cluster":   true,
	"slurm_partition": true,
	"slurm_user":      true,
}

// seedDefaultPermissions creates the standard permissions if they don't exist.
// It seeds both the legacy "resource:action" format and the new "vc:service:Action" format
// (dual-write strategy for gradual migration).
func (s *Service) seedDefaultPermissions() {
	legacyToNew := iam.LegacyToNew()

	for _, dp := range defaultPermissions {
		// 1. Seed legacy format (resource:action) — always.
		name := dp.Resource + ":" + dp.Action
		var count int64
		s.db.Model(&Permission{}).Where("name = ?", name).Count(&count)
		if count == 0 {
			perm := &Permission{
				Name:        name,
				Resource:    dp.Resource,
				Action:      dp.Action,
				Description: dp.Description,
			}
			if err := s.db.Create(perm).Error; err != nil {
				s.logger.Warn("Failed to seed permission", zap.String("name", name), zap.Error(err))
			}
		}

		// 2. Seed new format (vc:service:Action) — if mapping exists.
		if newName, ok := legacyToNew[name]; ok {
			var newCount int64
			s.db.Model(&Permission{}).Where("name = ?", newName).Count(&newCount)
			if newCount == 0 {
				perm := &Permission{
					Name:        newName,
					Resource:    dp.Resource,
					Action:      dp.Action,
					Description: dp.Description + " [v2]",
				}
				if err := s.db.Create(perm).Error; err != nil {
					s.logger.Warn("Failed to seed v2 permission", zap.String("name", newName), zap.Error(err))
				}
			}
		}
	}
}

// seedDefaultRoles creates the 4 system roles and attaches permissions.
func (s *Service) seedDefaultRoles() {
	// Load all permissions.
	var allPerms []Permission
	s.db.Find(&allPerms)

	permByName := map[string]Permission{}
	for _, p := range allPerms {
		permByName[p.Name] = p
	}

	for roleName, roleDef := range defaultRoles {
		var role Role
		err := s.db.Where("name = ?", roleName).First(&role).Error
		if err != nil {
			// Create the role.
			role = Role{
				Name:        roleName,
				Description: roleDef.Description,
			}
			if err := s.db.Create(&role).Error; err != nil {
				s.logger.Warn("Failed to seed role", zap.String("name", roleName), zap.Error(err))
				continue
			}
			s.logger.Info("Seeded default role", zap.String("name", roleName))
		}

		// Determine which permissions to assign.
		var rolePerms []Permission
		if len(roleDef.Actions) == 1 && roleDef.Actions[0] == "*" {
			// Admin gets everything.
			rolePerms = allPerms
		} else {
			actionSet := map[string]bool{}
			for _, a := range roleDef.Actions {
				actionSet[a] = true
			}
			for _, p := range allPerms {
				if !actionSet[p.Action] {
					continue
				}
				// Members can't manage IAM/infra resources.
				if roleName == "member" && memberExcludedResources[p.Resource] {
					continue
				}
				rolePerms = append(rolePerms, p)
			}
		}

		// Replace association (idempotent).
		if err := s.db.Model(&role).Association("Permissions").Replace(rolePerms); err != nil {
			s.logger.Warn("Failed to assign permissions to role",
				zap.String("role", roleName), zap.Error(err))
		}
	}
}

// SeedRBAC seeds default permissions and roles. Called during service init.
func (s *Service) SeedRBAC() {
	s.seedDefaultPermissions()
	s.seedDefaultRoles()

	// Assign admin role to admin user if not already assigned.
	var adminUser User
	if err := s.db.Where("username = ?", "admin").First(&adminUser).Error; err == nil {
		var adminRole Role
		if err := s.db.Where("name = ?", "admin").First(&adminRole).Error; err == nil {
			var count int64
			s.db.Raw("SELECT COUNT(*) FROM user_roles WHERE user_id = ? AND role_id = ?",
				adminUser.ID, adminRole.ID).Scan(&count)
			if count == 0 {
				_ = s.db.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)",
					adminUser.ID, adminRole.ID).Error
				s.logger.Info("Assigned admin role to admin user")
			}
		}
	}
}
