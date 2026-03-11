package hpc

import (
	"time"

	"go.uber.org/zap"
)

// seedHPCRoles creates HPC-specific custom roles if they do not exist.
// Unlike the 4 system-wide roles (admin, operator, member, viewer),
// these are purpose-built for HPC workload management:
//   - hpc_admin: manage HPC clusters + partitions + user sync + jobs
//   - hpc_user:  submit/view jobs + view clusters and GPU resources
func (s *Service) seedHPCRoles() {
	type roleSpec struct {
		Name        string
		Description string
		Permissions []string // resource:action pairs
	}

	roles := []roleSpec{
		{
			Name:        "hpc_admin",
			Description: "HPC platform administrator — manage clusters, partitions, GPU pools, and user sync",
			Permissions: []string{
				"hpc_cluster:list", "hpc_cluster:get", "hpc_cluster:create", "hpc_cluster:update", "hpc_cluster:delete", "hpc_cluster:scale",
				"hpc_gpu:list", "hpc_gpu:get", "hpc_gpu:create", "hpc_gpu:delete",
				"slurm_cluster:list", "slurm_cluster:get", "slurm_cluster:create", "slurm_cluster:update", "slurm_cluster:delete",
				"slurm_partition:list", "slurm_partition:create", "slurm_partition:update", "slurm_partition:delete",
				"hpc_job:list", "hpc_job:get", "hpc_job:create", "hpc_job:delete",
				"slurm_user:list", "slurm_user:create", "slurm_user:delete",
				"hpc_monitoring:list",
			},
		},
		{
			Name:        "hpc_user",
			Description: "HPC user — submit and manage own jobs, view cluster and GPU resources",
			Permissions: []string{
				"hpc_job:list", "hpc_job:get", "hpc_job:create", "hpc_job:delete",
				"hpc_cluster:list", "hpc_cluster:get",
				"hpc_gpu:list", "hpc_gpu:get",
				"slurm_cluster:list", "slurm_cluster:get",
				"slurm_partition:list",
				"hpc_monitoring:list",
			},
		},
	}

	for _, rs := range roles {
		// Check if role already exists.
		type role struct {
			ID   uint
			Name string
		}
		var existing role
		if err := s.db.Table("roles").Where("name = ?", rs.Name).First(&existing).Error; err == nil {
			// Role already exists, skip.
			continue
		}

		// Create the role.
		now := time.Now()
		result := s.db.Exec(
			"INSERT INTO roles (name, description, created_at, updated_at) VALUES (?, ?, ?, ?)",
			rs.Name, rs.Description, now, now,
		)
		if result.Error != nil {
			s.logger.Warn("Failed to seed HPC role", zap.String("role", rs.Name), zap.Error(result.Error))
			continue
		}

		// Retrieve the created role ID.
		var newRole role
		if err := s.db.Table("roles").Where("name = ?", rs.Name).First(&newRole).Error; err != nil {
			s.logger.Warn("Failed to find seeded HPC role", zap.String("role", rs.Name))
			continue
		}

		// Find matching permissions and attach them.
		for _, permName := range rs.Permissions {
			type perm struct {
				ID uint
			}
			var p perm
			if err := s.db.Table("permissions").Where("name = ?", permName).First(&p).Error; err != nil {
				// Permission may not exist yet (will be created by identity seed).
				continue
			}
			_ = s.db.Exec(
				"INSERT INTO role_permissions (role_id, permission_id, created_at) VALUES (?, ?, ?) ON CONFLICT DO NOTHING",
				newRole.ID, p.ID, now,
			)
		}
		s.logger.Info("Seeded HPC role", zap.String("role", rs.Name))
	}
}
