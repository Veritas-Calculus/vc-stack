import { Navigate, Route, Routes, useLocation, useParams } from 'react-router-dom'
import { lazy, Suspense, useEffect, useState, type ReactNode } from 'react'
import { Layout } from '@/components/Layout'
import { Login } from '@/features/auth/Login'
import { OidcCallback } from '@/features/auth/OidcCallback'
import { useAuthStore } from '@/lib/auth'
import { useAppStore } from '@/lib/appStore'

// ── Lazy-loaded feature pages ──────────────────────────────────────
// Vite will code-split each into a separate chunk automatically.
const SettingsIndex = lazy(() =>
  import('@/features/settings/SettingsIndex').then((m) => ({ default: m.SettingsIndex }))
)
const Network = lazy(() =>
  import('@/features/network/Network').then((m) => ({ default: m.Network }))
)
const Compute = lazy(() =>
  import('@/features/compute/Compute').then((m) => ({ default: m.Compute }))
)
const Storage = lazy(() =>
  import('@/features/storage/Storage').then((m) => ({ default: m.Storage }))
)
const Projects = lazy(() =>
  import('@/features/projects/Projects').then((m) => ({ default: m.Projects }))
)
const Notifications = lazy(() =>
  import('@/features/notifications/Notifications').then((m) => ({ default: m.Notifications }))
)
const Docs = lazy(() => import('@/features/docs/Docs').then((m) => ({ default: m.Docs })))
const Images = lazy(() => import('@/features/images/Images').then((m) => ({ default: m.Images })))
const Utilization = lazy(() =>
  import('@/features/utilization/Utilization').then((m) => ({ default: m.Utilization }))
)
const WebShell = lazy(() =>
  import('@/features/webshell/WebShell').then((m) => ({ default: m.WebShell }))
)
const SessionList = lazy(() =>
  import('@/features/webshell/SessionList').then((m) => ({ default: m.SessionList }))
)
const SessionReplay = lazy(() =>
  import('@/features/webshell/SessionReplay').then((m) => ({ default: m.SessionReplay }))
)
const Project = lazy(() =>
  import('@/features/project/Project').then((m) => ({ default: m.Project }))
)
const Templates = lazy(() =>
  import('@/features/templates/Templates').then((m) => ({ default: m.Templates }))
)
const Isos = lazy(() => import('@/features/templates/Isos').then((m) => ({ default: m.Isos })))
const K8sIsos = lazy(() =>
  import('@/features/templates/Isos').then((m) => ({ default: m.K8sIsos }))
)
const Roles = lazy(() => import('@/features/iam/Roles').then((m) => ({ default: m.Roles })))
const Policies = lazy(() =>
  import('@/features/iam/Policies').then((m) => ({ default: m.Policies }))
)
const ServiceAccounts = lazy(() =>
  import('@/features/iam/ServiceAccounts').then((m) => ({ default: m.ServiceAccounts }))
)
const AlertRules = lazy(() =>
  import('@/features/monitoring/AlertRules').then((m) => ({ default: m.AlertRules }))
)
const LogViewer = lazy(() =>
  import('@/features/monitoring/LogViewer').then((m) => ({ default: m.LogViewer }))
)
const MetricsDashboard = lazy(() =>
  import('@/features/monitoring/MetricsDashboard').then((m) => ({ default: m.MetricsDashboard }))
)
const LoadBalancersL7 = lazy(() =>
  import('@/features/network/LoadBalancersL7').then((m) => ({ default: m.LoadBalancersL7 }))
)
const FlowLogs = lazy(() =>
  import('@/features/network/FlowLogs').then((m) => ({ default: m.FlowLogs }))
)
const VPCPeering = lazy(() =>
  import('@/features/network/VPCPeering').then((m) => ({ default: m.VPCPeering }))
)
const DBaaS = lazy(() => import('@/features/compute/DBaaS').then((m) => ({ default: m.DBaaS })))
const AutoScaleV2 = lazy(() =>
  import('@/features/compute/AutoScale').then((m) => ({ default: m.AutoScale }))
)
const ContainerRegistry = lazy(() =>
  import('@/features/compute/ContainerRegistry').then((m) => ({ default: m.ContainerRegistry }))
)
const Organizations = lazy(() =>
  import('@/features/iam/Organizations').then((m) => ({ default: m.Organizations }))
)
const SecretsManager = lazy(() =>
  import('@/features/iam/SecretsManager').then((m) => ({ default: m.SecretsManager }))
)
const BudgetAlerts = lazy(() =>
  import('@/features/billing/BudgetAlerts').then((m) => ({ default: m.BudgetAlerts }))
)
const PlacementGroups = lazy(() =>
  import('@/features/compute/PlacementGroups').then((m) => ({ default: m.PlacementGroups }))
)
const FileShares = lazy(() =>
  import('@/features/storage/FileShares').then((m) => ({ default: m.FileShares }))
)
const StorageQoS = lazy(() =>
  import('@/features/storage/StorageQoS').then((m) => ({ default: m.StorageQoS }))
)
const PreemptibleInstances = lazy(() =>
  import('@/features/compute/PreemptibleInstances').then((m) => ({
    default: m.PreemptibleInstances
  }))
)
const ManagedRedis = lazy(() =>
  import('@/features/compute/ManagedRedis').then((m) => ({ default: m.ManagedRedis }))
)
const ManagedTiDB = lazy(() =>
  import('@/features/compute/ManagedTiDB').then((m) => ({ default: m.ManagedTiDB }))
)
const ManagedElasticsearch = lazy(() =>
  import('@/features/compute/ManagedElasticsearch').then((m) => ({
    default: m.ManagedElasticsearch
  }))
)
const NATGateways = lazy(() =>
  import('@/features/network/NATGateways').then((m) => ({ default: m.NATGateways }))
)
const ABACPolicies = lazy(() =>
  import('@/features/iam/ABACPolicies').then((m) => ({ default: m.ABACPolicies }))
)
const Invoices = lazy(() =>
  import('@/features/billing/Invoices').then((m) => ({ default: m.Invoices }))
)
const StackDrift = lazy(() =>
  import('@/features/orchestration/StackDrift').then((m) => ({ default: m.StackDrift }))
)
const GPUSchedulerPage = lazy(() =>
  import('@/features/compute/GPUResources').then((m) => ({ default: m.GPUResources }))
)
const Accounts = lazy(() =>
  import('@/features/accounts/Accounts').then((m) => ({ default: m.Accounts }))
)
const Infrastructure = lazy(() =>
  import('@/features/infrastructure/Infrastructure').then((m) => ({ default: m.Infrastructure }))
)
const Dashboard = lazy(() =>
  import('@/features/dashboard/Dashboard').then((m) => ({ default: m.Dashboard }))
)
const Events = lazy(() => import('@/features/events/Events').then((m) => ({ default: m.Events })))
const SecurityGroups = lazy(() =>
  import('@/features/network/SecurityGroups').then((m) => ({ default: m.SecurityGroups }))
)
const GlobalSettings = lazy(() =>
  import('@/features/settings/GlobalSettings').then((m) => ({ default: m.GlobalSettings }))
)
const Offerings = lazy(() =>
  import('@/features/offerings/Offerings').then((m) => ({ default: m.Offerings }))
)
const Domains = lazy(() =>
  import('@/features/domains/Domains').then((m) => ({ default: m.Domains }))
)
const SnapshotSchedules = lazy(() =>
  import('@/features/storage/SnapshotSchedules').then((m) => ({ default: m.SnapshotSchedules }))
)
const AffinityGroups = lazy(() =>
  import('@/features/compute/AffinityGroups').then((m) => ({ default: m.AffinityGroups }))
)
const UsageBilling = lazy(() =>
  import('@/features/billing/UsageBilling').then((m) => ({ default: m.UsageBilling }))
)
const Webhooks = lazy(() =>
  import('@/features/tools/Webhooks').then((m) => ({ default: m.Webhooks }))
)
const VPNManagement = lazy(() =>
  import('@/features/vpn/VPNManagement').then((m) => ({ default: m.VPNManagement }))
)
const Backups = lazy(() =>
  import('@/features/backup/Backups').then((m) => ({ default: m.Backups }))
)
const AutoScale = lazy(() =>
  import('@/features/autoscale/AutoScale').then((m) => ({ default: m.AutoScale }))
)
const DNSManagement = lazy(() =>
  import('@/features/dns/DNSManagement').then((m) => ({ default: m.DNSManagement }))
)
const ObjectStorage = lazy(() =>
  import('@/features/objectstorage/ObjectStorage').then((m) => ({ default: m.ObjectStorage }))
)
const Orchestration = lazy(() =>
  import('@/features/orchestration/Orchestration').then((m) => ({ default: m.Orchestration }))
)
const RBAC = lazy(() => import('@/features/rbac/RBAC').then((m) => ({ default: m.RBAC })))
const Federation = lazy(() =>
  import('@/features/federation/Federation').then((m) => ({ default: m.Federation }))
)
const HighAvailability = lazy(() =>
  import('@/features/ha/HighAvailability').then((m) => ({ default: m.HighAvailability }))
)
const KeyManagement = lazy(() =>
  import('@/features/kms/KeyManagement').then((m) => ({ default: m.KeyManagement }))
)
const RateLimiting = lazy(() =>
  import('@/features/ratelimit/RateLimiting').then((m) => ({ default: m.RateLimiting }))
)
const DataEncryption = lazy(() =>
  import('@/features/encryption/DataEncryption').then((m) => ({ default: m.DataEncryption }))
)
const Kubernetes = lazy(() =>
  import('@/features/kubernetes/Kubernetes').then((m) => ({ default: m.Kubernetes }))
)
const ComplianceAudit = lazy(() =>
  import('@/features/audit/ComplianceAudit').then((m) => ({ default: m.ComplianceAudit }))
)
const DisasterRecovery = lazy(() =>
  import('@/features/dr/DisasterRecovery').then((m) => ({ default: m.DisasterRecovery }))
)
const BareMetal = lazy(() =>
  import('@/features/baremetal/BareMetal').then((m) => ({ default: m.BareMetal }))
)
const ServiceCatalog = lazy(() =>
  import('@/features/catalog/ServiceCatalog').then((m) => ({ default: m.ServiceCatalog }))
)
const SelfHealing = lazy(() =>
  import('@/features/selfheal/SelfHealing').then((m) => ({ default: m.SelfHealing }))
)
const PlatformSettings = lazy(() =>
  import('@/features/platform/PlatformSettings').then((m) => ({ default: m.PlatformSettings }))
)
const HPC = lazy(() => import('@/features/hpc/HPC').then((m) => ({ default: m.HPC })))
const HPCJobs = lazy(() => import('@/features/hpc/HPCJobs').then((m) => ({ default: m.HPCJobs })))
const HPCClusters = lazy(() =>
  import('@/features/hpc/HPCClusters').then((m) => ({ default: m.HPCClusters }))
)
const GPUResources = lazy(() =>
  import('@/features/hpc/GPUResources').then((m) => ({ default: m.GPUResources }))
)

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
              <Suspense
                fallback={
                  <div
                    style={{
                      display: 'flex',
                      justifyContent: 'center',
                      alignItems: 'center',
                      height: '60vh'
                    }}
                  >
                    <div className="loading-spinner" />
                  </div>
                }
              >
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
              </Suspense>
            </Layout>
          </RequireAuth>
        }
      />
    </Routes>
  )
}
