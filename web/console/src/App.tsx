import { Navigate, Route, Routes, useLocation, useParams } from 'react-router-dom'
import { useEffect, useState, type ReactNode } from 'react'
import { Layout } from '@/components/Layout'
import { SettingsIndex } from '@/features/settings/SettingsIndex'
import { Network } from '@/features/network/Network'
import { Compute } from '@/features/compute/Compute'
import { Storage } from '@/features/storage/Storage'
import { Projects } from '@/features/projects/Projects'
import { Notifications } from '@/features/notifications/Notifications'
import { Login } from '@/features/auth/Login'
import { OidcCallback } from '@/features/auth/OidcCallback'
import { Docs } from '@/features/docs/Docs'
import { Images } from '@/features/images/Images'
import { Utilization } from '@/features/utilization/Utilization'
import { WebShell } from '@/features/webshell/WebShell'
import { SessionList } from '@/features/webshell/SessionList'
import { SessionReplay } from '@/features/webshell/SessionReplay'
import { useAuthStore } from '@/lib/auth'
import { useAppStore } from '@/lib/appStore'
import { Project } from '@/features/project/Project'
import { Templates } from '@/features/templates/Templates'
import { Isos, K8sIsos } from '@/features/templates/Isos'
import { Roles } from '@/features/iam/Roles'
import { Policies } from '@/features/iam/Policies'
import { ServiceAccounts } from '@/features/iam/ServiceAccounts'
import { AlertRules } from '@/features/monitoring/AlertRules'
import { LogViewer } from '@/features/monitoring/LogViewer'
import { MetricsDashboard } from '@/features/monitoring/MetricsDashboard'
import { LoadBalancersL7 } from '@/features/network/LoadBalancersL7'
import { FlowLogs } from '@/features/network/FlowLogs'
import { VPCPeering } from '@/features/network/VPCPeering'
import { DBaaS } from '@/features/compute/DBaaS'
import { AutoScale as AutoScaleV2 } from '@/features/compute/AutoScale'
import { ContainerRegistry } from '@/features/compute/ContainerRegistry'
import { Organizations } from '@/features/iam/Organizations'
import { SecretsManager } from '@/features/iam/SecretsManager'
import { BudgetAlerts } from '@/features/billing/BudgetAlerts'
import { PlacementGroups } from '@/features/compute/PlacementGroups'
import { FileShares } from '@/features/storage/FileShares'
import { StorageQoS } from '@/features/storage/StorageQoS'
import { PreemptibleInstances } from '@/features/compute/PreemptibleInstances'
import { ManagedRedis } from '@/features/compute/ManagedRedis'
import { ManagedTiDB } from '@/features/compute/ManagedTiDB'
import { ManagedElasticsearch } from '@/features/compute/ManagedElasticsearch'
import { NATGateways } from '@/features/network/NATGateways'
import { ABACPolicies } from '@/features/iam/ABACPolicies'
import { Invoices } from '@/features/billing/Invoices'
import { StackDrift } from '@/features/orchestration/StackDrift'
import { GPUResources as GPUSchedulerPage } from '@/features/compute/GPUResources'
import { Accounts } from '@/features/accounts/Accounts'
import { Infrastructure } from '@/features/infrastructure/Infrastructure'
import { Dashboard } from '@/features/dashboard/Dashboard'
import { Events } from '@/features/events/Events'
import { SecurityGroups } from '@/features/network/SecurityGroups'
import { GlobalSettings } from '@/features/settings/GlobalSettings'
import { Offerings } from '@/features/offerings/Offerings'
import { Domains } from '@/features/domains/Domains'
import { SnapshotSchedules } from '@/features/storage/SnapshotSchedules'
import { AffinityGroups } from '@/features/compute/AffinityGroups'
import { UsageBilling } from '@/features/billing/UsageBilling'
import { Webhooks } from '@/features/tools/Webhooks'
import { VPNManagement } from '@/features/vpn/VPNManagement'
import { Backups } from '@/features/backup/Backups'
import { AutoScale } from '@/features/autoscale/AutoScale'
import { DNSManagement } from '@/features/dns/DNSManagement'
import { ObjectStorage } from '@/features/objectstorage/ObjectStorage'
import { Orchestration } from '@/features/orchestration/Orchestration'
import { RBAC } from '@/features/rbac/RBAC'
import { Federation } from '@/features/federation/Federation'
import { HighAvailability } from '@/features/ha/HighAvailability'
import { KeyManagement } from '@/features/kms/KeyManagement'
import { RateLimiting } from '@/features/ratelimit/RateLimiting'
import { DataEncryption } from '@/features/encryption/DataEncryption'
import { Kubernetes } from '@/features/kubernetes/Kubernetes'
import { ComplianceAudit } from '@/features/audit/ComplianceAudit'
import { DisasterRecovery } from '@/features/dr/DisasterRecovery'
import { BareMetal } from '@/features/baremetal/BareMetal'
import { ServiceCatalog } from '@/features/catalog/ServiceCatalog'
import { SelfHealing } from '@/features/selfheal/SelfHealing'
import { PlatformSettings } from '@/features/platform/PlatformSettings'
import { HPC } from '@/features/hpc/HPC'
import { HPCJobs } from '@/features/hpc/HPCJobs'
import { HPCClusters } from '@/features/hpc/HPCClusters'
import { GPUResources } from '@/features/hpc/GPUResources'

function RequireAuth({ children }: { children: React.ReactNode }) {
  const token = useAuthStore((s) => s.token)
  const location = useLocation()
  const [isReady, setIsReady] = useState(false)

  // Helper to log to both console and localStorage for debugging
  const debugLog = (msg: string) => {
    // eslint-disable-next-line no-console
    console.log(msg)
    try {
      const logs = JSON.parse(localStorage.getItem('debug_logs') || '[]')
      logs.push({ time: new Date().toISOString(), msg })
      // Keep only last 50 logs
      if (logs.length > 50) logs.shift()
      localStorage.setItem('debug_logs', JSON.stringify(logs))
    } catch {
      // ignore
    }
  }

  // Wait for Zustand persist to hydrate from localStorage
  // This prevents the flash of redirect before token is loaded
  useEffect(() => {
    // Check if we have a token in localStorage
    const checkAuth = async () => {
      try {
        const authData = localStorage.getItem('auth')
        debugLog(`[RequireAuth] Checking localStorage auth: ${authData ? 'Found' : 'Not found'}`)
        if (authData) {
          const parsed = JSON.parse(authData)
          if (parsed?.state?.token) {
            debugLog('[RequireAuth] Token found in localStorage, waiting for Zustand hydration...')
            // Token exists in localStorage, wait a bit for Zustand to catch up
            await new Promise((resolve) => setTimeout(resolve, 100))
          }
        }
      } catch {
        // Ignore parse errors
      }
      setIsReady(true)
    }
    checkAuth()
  }, [])

  // Don't render anything until we've checked localStorage
  if (!isReady) {
    return null
  }

  // Now check the Zustand store
  debugLog(
    `[RequireAuth] Ready. Token in Zustand store: ${token ? 'Found' : 'Not found'} at ${location.pathname}`
  )

  if (!token) {
    debugLog(`[RequireAuth] No token, redirecting to /login from: ${location.pathname}`)
    // Save the location they were trying to access
    return <Navigate to="/login" state={{ from: location }} replace />
  }

  debugLog(`[RequireAuth] Authenticated, rendering children for: ${location.pathname}`)
  return <>{children}</>
}

/**
 * Auto-set activeProjectId when navigating to /project/:projectId/* URLs.
 * This ensures project context is always correctly set from the URL,
 * whether the user arrives from login, a bookmark, or a direct link.
 */
function ProjectContextSetter({ children }: { children: ReactNode }) {
  const { projectId } = useParams<{ projectId: string }>()
  const setActiveProjectId = useAppStore((s) => s.setActiveProjectId)
  const setProjectContext = useAppStore((s) => s.setProjectContext)

  useEffect(() => {
    if (projectId) {
      setActiveProjectId(projectId)
      setProjectContext(true)
    }
  }, [projectId, setActiveProjectId, setProjectContext])

  return <>{children}</>
}

/** Smart redirect: if a project is active, go to its dashboard; otherwise global dashboard. */
function SmartRedirect() {
  const activeProjectId = useAppStore((s) => s.activeProjectId)
  if (activeProjectId) {
    return <Navigate to={`/project/${encodeURIComponent(activeProjectId)}/dashboard`} replace />
  }
  return <Navigate to="/dashboard" replace />
}

export default function App() {
  // Version marker to confirm new code is loaded
  useEffect(() => {
    // eslint-disable-next-line no-console
    console.log('[App] VC Console loaded - Version: 2025-12-08-23:20')
  }, [])

  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/auth/oidc/callback" element={<OidcCallback />} />
      <Route
        path="/*"
        element={
          <RequireAuth>
            <Layout>
              <Routes>
                <Route path="/" element={<SmartRedirect />} />
                {/* Global/top-level */}
                <Route path="/dashboard" element={<Dashboard />} />
                <Route path="/events" element={<Events />} />
                <Route path="/docs" element={<Docs />} />
                <Route path="/images" element={<Images />} />
                <Route path="/utilization" element={<Utilization />} />
                <Route path="/webshell" element={<WebShell />} />
                <Route path="/webshell/sessions" element={<SessionList />} />
                <Route path="/webshell/replay/:sessionId" element={<SessionReplay />} />
                {/* Global modules (kept as fallback) */}
                <Route path="/images/templates" element={<Templates />} />
                <Route path="/images/iso" element={<Isos />} />
                <Route path="/images/k8s-iso" element={<K8sIsos />} />
                <Route path="/iam/roles" element={<Roles />} />
                <Route path="/iam/policies" element={<Policies />} />
                <Route path="/iam/service-accounts" element={<ServiceAccounts />} />
                {/* Observability */}
                <Route path="/monitoring/alerts" element={<AlertRules />} />
                <Route path="/monitoring/logs" element={<LogViewer />} />
                <Route path="/monitoring/metrics" element={<MetricsDashboard />} />
                {/* Network Enhancement */}
                <Route path="/network/alb" element={<LoadBalancersL7 />} />
                <Route path="/network/flow-logs" element={<FlowLogs />} />
                <Route path="/network/peering" element={<VPCPeering />} />
                {/* Managed Services */}
                <Route path="/compute/databases" element={<DBaaS />} />
                <Route path="/compute/autoscale" element={<AutoScaleV2 />} />
                <Route path="/compute/registry" element={<ContainerRegistry />} />
                {/* N5: IAM & Cost Governance */}
                <Route path="/iam/organizations" element={<Organizations />} />
                <Route path="/iam/secrets" element={<SecretsManager />} />
                <Route path="/billing/budgets" element={<BudgetAlerts />} />
                <Route path="/compute/placement-groups" element={<PlacementGroups />} />
                {/* N6: Storage & Advanced Compute */}
                <Route path="/storage/file-shares" element={<FileShares />} />
                <Route path="/storage/qos" element={<StorageQoS />} />
                <Route path="/compute/preemptible" element={<PreemptibleInstances />} />
                {/* N7: Production Critical */}
                <Route path="/compute/redis" element={<ManagedRedis />} />
                <Route path="/network/nat-gateways" element={<NATGateways />} />
                <Route path="/iam/abac" element={<ABACPolicies />} />
                {/* N8: Data Services */}
                <Route path="/compute/tidb" element={<ManagedTiDB />} />
                <Route path="/compute/elasticsearch" element={<ManagedElasticsearch />} />
                <Route path="/billing/invoices" element={<Invoices />} />
                {/* N9: Operations */}
                <Route path="/orchestration/drift" element={<StackDrift />} />
                <Route path="/compute/gpu-scheduler" element={<GPUSchedulerPage />} />
                <Route path="/accounts" element={<Accounts />} />
                {/* Security Groups (global) */}
                <Route path="/network/security-groups" element={<SecurityGroups />} />
                {/* Service Offerings */}
                <Route path="/offerings" element={<Offerings />} />
                {/* Global Settings */}
                <Route path="/settings/global" element={<GlobalSettings />} />
                {/* Domains */}
                <Route path="/domains" element={<Domains />} />
                {/* Snapshot Schedules */}
                <Route path="/snapshot-schedules" element={<SnapshotSchedules />} />
                {/* Affinity Groups */}
                <Route path="/affinity-groups" element={<AffinityGroups />} />
                {/* Usage & Billing */}
                <Route path="/usage" element={<UsageBilling />} />
                {/* Webhooks */}
                <Route path="/webhooks" element={<Webhooks />} />
                {/* VPN */}
                <Route path="/vpn" element={<VPNManagement />} />
                {/* Backups */}
                <Route path="/backups" element={<Backups />} />
                {/* Auto Scale */}
                <Route path="/autoscale" element={<AutoScale />} />
                {/* DNS */}
                <Route path="/dns" element={<DNSManagement />} />
                {/* Object Storage */}
                <Route path="/object-storage" element={<ObjectStorage />} />
                {/* Orchestration */}
                <Route path="/orchestration" element={<Orchestration />} />
                {/* RBAC */}
                <Route path="/rbac" element={<RBAC />} />
                {/* Federation */}
                <Route path="/federation" element={<Federation />} />
                {/* High Availability */}
                <Route path="/ha" element={<HighAvailability />} />
                <Route path="/kms" element={<KeyManagement />} />
                <Route path="/rate-limits" element={<RateLimiting />} />
                <Route path="/encryption" element={<DataEncryption />} />
                <Route path="/kubernetes" element={<Kubernetes />} />
                <Route path="/compliance-audit" element={<ComplianceAudit />} />
                <Route path="/disaster-recovery" element={<DisasterRecovery />} />
                <Route path="/bare-metal" element={<BareMetal />} />
                <Route path="/service-catalog" element={<ServiceCatalog />} />
                <Route path="/self-healing" element={<SelfHealing />} />
                <Route path="/platform-settings" element={<PlatformSettings />} />
                {/* HPC */}
                <Route path="/hpc" element={<HPC />} />
                <Route path="/hpc/clusters" element={<HPCClusters />} />
                <Route path="/hpc/jobs" element={<HPCJobs />} />
                <Route path="/hpc/gpu" element={<GPUResources />} />
                <Route path="/hpc/*" element={<HPC />} />
                {/* Global Infrastructure */}
                <Route path="/infrastructure/*" element={<Infrastructure />} />
                {/* ── Project-scoped routes (all wrapped in ProjectContextSetter) ── */}
                <Route
                  path="/project/:projectId/dashboard"
                  element={
                    <ProjectContextSetter>
                      <Dashboard />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/bare-metal"
                  element={
                    <ProjectContextSetter>
                      <BareMetal />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/autoscale"
                  element={
                    <ProjectContextSetter>
                      <AutoScale />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/affinity-groups"
                  element={
                    <ProjectContextSetter>
                      <AffinityGroups />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/snapshot-schedules"
                  element={
                    <ProjectContextSetter>
                      <SnapshotSchedules />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/object-storage"
                  element={
                    <ProjectContextSetter>
                      <ObjectStorage />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/dns"
                  element={
                    <ProjectContextSetter>
                      <DNSManagement />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/platform-settings"
                  element={
                    <ProjectContextSetter>
                      <PlatformSettings />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/self-healing"
                  element={
                    <ProjectContextSetter>
                      <SelfHealing />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/ha"
                  element={
                    <ProjectContextSetter>
                      <HighAvailability />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/disaster-recovery"
                  element={
                    <ProjectContextSetter>
                      <DisasterRecovery />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/rbac"
                  element={
                    <ProjectContextSetter>
                      <RBAC />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/kms"
                  element={
                    <ProjectContextSetter>
                      <KeyManagement />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/encryption"
                  element={
                    <ProjectContextSetter>
                      <DataEncryption />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/federation"
                  element={
                    <ProjectContextSetter>
                      <Federation />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/compliance-audit"
                  element={
                    <ProjectContextSetter>
                      <ComplianceAudit />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/offerings"
                  element={
                    <ProjectContextSetter>
                      <Offerings />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/service-catalog"
                  element={
                    <ProjectContextSetter>
                      <ServiceCatalog />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/orchestration"
                  element={
                    <ProjectContextSetter>
                      <Orchestration />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/accounts"
                  element={
                    <ProjectContextSetter>
                      <Accounts />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/domains"
                  element={
                    <ProjectContextSetter>
                      <Domains />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/events"
                  element={
                    <ProjectContextSetter>
                      <Events />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/notifications"
                  element={
                    <ProjectContextSetter>
                      <Notifications />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/webhooks"
                  element={
                    <ProjectContextSetter>
                      <Webhooks />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/usage"
                  element={
                    <ProjectContextSetter>
                      <UsageBilling />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/rate-limits"
                  element={
                    <ProjectContextSetter>
                      <RateLimiting />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/settings/global"
                  element={
                    <ProjectContextSetter>
                      <GlobalSettings />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/utilization"
                  element={
                    <ProjectContextSetter>
                      <Utilization />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/docs"
                  element={
                    <ProjectContextSetter>
                      <Docs />
                    </ProjectContextSetter>
                  }
                />
                {/* ── Project-scoped HPC routes ── */}
                <Route
                  path="/project/:projectId/hpc/clusters"
                  element={
                    <ProjectContextSetter>
                      <HPCClusters />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/hpc/jobs"
                  element={
                    <ProjectContextSetter>
                      <HPCJobs />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/hpc/gpu"
                  element={
                    <ProjectContextSetter>
                      <GPUResources />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/hpc"
                  element={
                    <ProjectContextSetter>
                      <HPC />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/network/sg"
                  element={
                    <ProjectContextSetter>
                      <SecurityGroups />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/infrastructure/*"
                  element={
                    <ProjectContextSetter>
                      <Infrastructure />
                    </ProjectContextSetter>
                  }
                />
                <Route path="/projects/*" element={<Projects />} />
                <Route
                  path="/project/:projectId"
                  element={
                    <ProjectContextSetter>
                      <Project />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/images"
                  element={
                    <ProjectContextSetter>
                      <Images />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/utilization"
                  element={
                    <ProjectContextSetter>
                      <Utilization />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/images/templates"
                  element={
                    <ProjectContextSetter>
                      <Templates />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/images/iso"
                  element={
                    <ProjectContextSetter>
                      <Isos />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/images/k8s-iso"
                  element={
                    <ProjectContextSetter>
                      <K8sIsos />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/compute/*"
                  element={
                    <ProjectContextSetter>
                      <Compute />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/network/*"
                  element={
                    <ProjectContextSetter>
                      <Network />
                    </ProjectContextSetter>
                  }
                />
                <Route
                  path="/project/:projectId/storage/*"
                  element={
                    <ProjectContextSetter>
                      <Storage />
                    </ProjectContextSetter>
                  }
                />
                <Route path="/settings/*" element={<SettingsIndex />} />
                <Route path="/notifications" element={<Notifications />} />
              </Routes>
            </Layout>
          </RequireAuth>
        }
      />
    </Routes>
  )
}
