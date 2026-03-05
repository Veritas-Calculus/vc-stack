import { Navigate, Route, Routes, useLocation } from 'react-router-dom'
import { useEffect, useState } from 'react'
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
import { Project } from '@/features/project/Project'
import { Templates } from '@/features/templates/Templates'
import { Isos, K8sIsos } from '@/features/templates/Isos'
import { Roles } from '@/features/iam/Roles'
import { Policies } from '@/features/iam/Policies'
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
                <Route path="/" element={<Navigate to="/projects" replace />} />
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
                {/* Global Infrastructure */}
                <Route path="/infrastructure/*" element={<Infrastructure />} />
                {/* Project-scoped Security Groups */}
                <Route path="/project/:projectId/network/sg" element={<SecurityGroups />} />
                {/* Project-scoped Infrastructure */}
                <Route path="/project/:projectId/infrastructure/*" element={<Infrastructure />} />
                <Route path="/projects/*" element={<Projects />} />
                {/* Project-scoped */}
                <Route path="/project/:projectId" element={<Project />} />
                <Route path="/project/:projectId/images" element={<Images />} />
                <Route path="/project/:projectId/utilization" element={<Utilization />} />
                <Route path="/project/:projectId/images/templates" element={<Templates />} />
                <Route path="/project/:projectId/images/iso" element={<Isos />} />
                <Route path="/project/:projectId/images/k8s-iso" element={<K8sIsos />} />
                <Route path="/project/:projectId/compute/*" element={<Compute />} />
                <Route path="/project/:projectId/network/*" element={<Network />} />
                <Route path="/project/:projectId/storage/*" element={<Storage />} />
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
