// Auto-generated domain module — do not edit manually.
// This file was extracted from index.ts for better code organization.
import api from './client'

export type LoginResponse = {
  access_token: string
  refresh_token: string
  expires_in: number
  token_type: string
}

export async function loginApi(username: string, password: string): Promise<LoginResponse> {
  const res = await api.post<LoginResponse>('/v1/auth/login', { username, password })
  return res.data
}

// Identity: Projects and Users (for name mapping)
export type UIProject = { id: string; name: string; description?: string; user_id?: string }

export async function fetchProjects(): Promise<UIProject[]> {
  const res = await api.get<{
    projects: Array<{ id: number; name: string; description?: string; user_id?: number }>
  }>('/v1/projects')
  return (res.data.projects ?? []).map((p) => ({
    id: String(p.id),
    name: p.name,
    description: p.description,
    user_id: p.user_id ? String(p.user_id) : undefined
  }))
}

export async function createProject(body: {
  name: string
  description?: string
}): Promise<UIProject> {
  const res = await api.post<{ project: { id: number; name: string; description?: string } }>(
    '/v1/projects',
    body
  )
  const p = res.data.project
  return { id: String(p.id), name: p.name, description: p.description }
}

export async function deleteProject(id: string): Promise<void> {
  await api.delete(`/v1/projects/${id}`)
}

export type UIProjectMember = {
  id: string
  project_id: string
  user_id: string
  role: string // admin, member, viewer
  user?: { id: number; username: string; email: string; first_name?: string; last_name?: string }
  created_at?: string
}

export async function fetchProjectMembers(projectId: string): Promise<UIProjectMember[]> {
  const res = await api.get<{
    members: Array<{
      id: number
      project_id: number
      user_id: number
      role: string
      created_at?: string
      user?: {
        id: number
        username: string
        email: string
        first_name?: string
        last_name?: string
      }
    }>
  }>(`/v1/projects/${projectId}/members`)
  return (res.data.members ?? []).map((m) => ({
    id: String(m.id),
    project_id: String(m.project_id),
    user_id: String(m.user_id),
    role: m.role,
    user: m.user,
    created_at: m.created_at
  }))
}

export async function addProjectMember(
  projectId: string,
  userId: number,
  role = 'member'
): Promise<UIProjectMember> {
  const res = await api.post<{
    member: { id: number; project_id: number; user_id: number; role: string }
  }>(`/v1/projects/${projectId}/members`, { user_id: userId, role })
  const m = res.data.member
  return {
    id: String(m.id),
    project_id: String(m.project_id),
    user_id: String(m.user_id),
    role: m.role
  }
}

export async function removeProjectMember(projectId: string, memberId: string): Promise<void> {
  await api.delete(`/v1/projects/${projectId}/members/${memberId}`)
}

export async function updateProjectMemberRole(
  projectId: string,
  memberId: string,
  role: string
): Promise<void> {
  await api.put(`/v1/projects/${projectId}/members/${memberId}`, { role })
}

export type UIUser = {
  id: string
  username?: string
  email?: string
  first_name?: string
  last_name?: string
}

export async function fetchUsers(): Promise<UIUser[]> {
  const res = await api.get<{
    users: Array<{
      id: number
      username?: string
      email?: string
      first_name?: string
      last_name?: string
    }>
  }>('/v1/users')
  return (res.data.users ?? []).map((u) => ({
    id: String(u.id),
    username: u.username,
    email: u.email,
    first_name: u.first_name,
    last_name: u.last_name
  }))
}

export async function createUser(body: {
  username: string
  email: string
  password: string
  first_name?: string
  last_name?: string
}): Promise<UIUser> {
  const res = await api.post<{ user: { id: number; username: string; email: string } }>(
    '/v1/users',
    body
  )
  const u = res.data.user
  return { id: String(u.id), username: u.username, email: u.email }
}

export async function deleteUser(id: string): Promise<void> {
  await api.delete(`/v1/users/${id}`)
}

export async function updateUserRole(userId: string, role: string): Promise<void> {
  await api.put(`/v1/users/${userId}/role`, { role })
}

export type UIRole = {
  id: string
  name: string
  description?: string
  permissions?: string[]
}

export async function fetchRoles(): Promise<UIRole[]> {
  const res = await api.get<{
    roles: Array<{ id: number; name: string; description?: string; permissions?: string[] }>
  }>('/v1/roles')
  return (res.data.roles ?? []).map((r) => ({ ...r, id: String(r.id) }))
}

export async function createRole(body: {
  name: string
  description?: string
  permissions?: string[]
}): Promise<UIRole> {
  const res = await api.post<{ role: { id: number; name: string } }>('/v1/roles', body)
  return { id: String(res.data.role.id), name: res.data.role.name }
}

export async function deleteRole(id: string): Promise<void> {
  await api.delete(`/v1/roles/${id}`)
}
// ── Service Accounts (API Key Management) ───────────────────

export type UIServiceAccount = {
  id: number
  name: string
  description: string
  project_id?: number
  created_by_id: number
  access_key_id: string
  is_active: boolean
  last_used_at?: string
  expires_at?: string
  created_at: string
  updated_at: string
  roles?: Array<{ id: number; name: string }>
  policies?: Array<{ id: number; name: string }>
}

export type CreateServiceAccountResponse = {
  service_account: UIServiceAccount
  access_key_id: string
  secret_key: string
}

export async function fetchServiceAccounts(): Promise<UIServiceAccount[]> {
  const res = await api.get<{ service_accounts: UIServiceAccount[] }>('/v1/service-accounts')
  return res.data.service_accounts ?? []
}

export async function createServiceAccount(body: {
  name: string
  description?: string
  project_id?: number
  expires_in?: string
}): Promise<CreateServiceAccountResponse> {
  const res = await api.post<CreateServiceAccountResponse>('/v1/service-accounts', body)
  return res.data
}

export async function deleteServiceAccount(id: number): Promise<void> {
  await api.delete(`/v1/service-accounts/${id}`)
}

export async function rotateServiceAccountKey(id: number): Promise<CreateServiceAccountResponse> {
  const res = await api.post<CreateServiceAccountResponse>(`/v1/service-accounts/${id}/rotate`)
  return res.data
}

export async function toggleServiceAccountStatus(id: number, active: boolean): Promise<void> {
  await api.patch(`/v1/service-accounts/${id}/status`, { active })
}

// ── Alert Rules ─────────────────────────────────────────────

// ── N5: Organization ────────────────────────────────────────

export type UIOrganization = {
  id: number
  name: string
  display_name: string
  description: string
  status: string
  ous?: Array<{ id: number; name: string; path: string }>
  created_at: string
}

export async function fetchOrganizations(): Promise<UIOrganization[]> {
  const res = await api.get<{ organizations: UIOrganization[] }>('/v1/organizations')
  return res.data.organizations ?? []
}

export async function createOrganization(body: {
  name: string
  display_name?: string
  description?: string
}): Promise<UIOrganization> {
  const res = await api.post<{ organization: UIOrganization }>('/v1/organizations', body)
  return res.data.organization
}

export async function deleteOrganization(id: number): Promise<void> {
  await api.delete(`/v1/organizations/${id}`)
}

// ── N5: Secrets Manager ─────────────────────────────────────

// ── N5: Secrets Manager ─────────────────────────────────────

export type UISecret = {
  id: number
  name: string
  description: string
  version_id: number
  rotate_after_days: number
  last_rotated?: string
  created_at: string
}

export async function fetchSecrets(): Promise<UISecret[]> {
  const res = await api.get<{ secrets: UISecret[] }>('/v1/secrets')
  return res.data.secrets ?? []
}

export async function createSecret(body: {
  name: string
  description?: string
  value: string
  rotate_after_days?: number
}): Promise<UISecret> {
  const res = await api.post<{ secret: UISecret }>('/v1/secrets', body)
  return res.data.secret
}

export async function deleteSecret(id: number): Promise<void> {
  await api.delete(`/v1/secrets/${id}`)
}

export async function rotateSecret(id: number): Promise<void> {
  await api.post(`/v1/secrets/${id}/rotate`)
}

// ── N5: Budget ──────────────────────────────────────────────

// ── N7: ABAC Policies ───────────────────────────────────────

export type UIABACPolicy = {
  id: number
  name: string
  description: string
  effect: string
  resource: string
  actions: string
  conditions: string
  priority: number
  enabled: boolean
  created_at: string
}

export async function fetchABACPolicies(): Promise<UIABACPolicy[]> {
  const res = await api.get<{ policies: UIABACPolicy[] }>('/v1/abac/policies')
  return res.data.policies ?? []
}

export async function createABACPolicy(body: {
  name: string
  effect: string
  resource: string
  actions: string
  conditions?: Array<{ key: string; operator: string; value: string }>
  priority?: number
}): Promise<UIABACPolicy> {
  const res = await api.post<{ policy: UIABACPolicy }>('/v1/abac/policies', body)
  return res.data.policy
}

export async function deleteABACPolicy(id: number): Promise<void> {
  await api.delete(`/v1/abac/policies/${id}`)
}

// ── N8: Managed TiDB ────────────────────────────────────────
