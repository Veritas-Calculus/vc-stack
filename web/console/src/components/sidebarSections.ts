/**
 * Sidebar navigation sections — extracted from Layout.tsx for clarity.
 *
 * Groups follow a CloudStack-inspired hierarchy:
 *   Dashboard → Compute → Storage → Network → Images →
 *   Infrastructure → Security → Service Offerings → Administration
 */

export type LinkItem = { type: 'link'; to: string; label: string }
export type GroupItem = {
  type: 'group'
  label: string
  base: string
  children: Array<{ to: string; label: string }>
}
export type SidebarSection = LinkItem | GroupItem

/** Pre-project sidebar (no project selected yet). */
export function getGlobalSections(): SidebarSection[] {
  return [
    { type: 'link', to: '/dashboard', label: 'Dashboard' },

    // ── Compute ──────────────────────────────────────────────
    {
      type: 'group',
      label: 'Compute',
      base: '/_compute',
      children: [
        { to: '/kubernetes', label: 'Kubernetes' },
        { to: '/bare-metal', label: 'Bare Metal' },
        { to: '/autoscale', label: 'Auto Scale' },
        { to: '/affinity-groups', label: 'Affinity Groups' }
      ]
    },

    // ── Storage ──────────────────────────────────────────────
    {
      type: 'group',
      label: 'Storage',
      base: '/_storage',
      children: [
        { to: '/object-storage', label: 'Object Storage' },
        { to: '/backups', label: 'Backups' }
      ]
    },

    // ── Network ──────────────────────────────────────────────
    {
      type: 'group',
      label: 'Network',
      base: '/_network',
      children: [
        { to: '/vpn', label: 'VPN' },
        { to: '/dns', label: 'DNS' }
      ]
    },

    // ── Images ───────────────────────────────────────────────
    { type: 'link', to: '/images', label: 'Images' },

    // ── Infrastructure ───────────────────────────────────────
    {
      type: 'group',
      label: 'Infrastructure',
      base: '/infrastructure',
      children: [
        { to: '/infrastructure/overview', label: 'Overview' },
        { to: '/infrastructure/zones', label: 'Zones' },
        { to: '/infrastructure/clusters', label: 'Clusters' },
        { to: '/infrastructure/hosts', label: 'Hosts' },
        { to: '/platform-settings', label: 'Platform Services' },
        { to: '/self-healing', label: 'Self-Healing' },
        { to: '/ha', label: 'High Availability' },
        { to: '/disaster-recovery', label: 'Disaster Recovery' }
      ]
    },

    // ── Security ─────────────────────────────────────────────
    {
      type: 'group',
      label: 'Security',
      base: '/_security',
      children: [
        { to: '/rbac', label: 'Access Control' },
        { to: '/kms', label: 'Key Management' },
        { to: '/encryption', label: 'Data Encryption' },
        { to: '/federation', label: 'Federation' },
        { to: '/compliance-audit', label: 'Compliance' }
      ]
    },

    // ── Service Offerings ────────────────────────────────────
    {
      type: 'group',
      label: 'Service Offerings',
      base: '/_offerings',
      children: [
        { to: '/offerings', label: 'Offerings' },
        { to: '/service-catalog', label: 'Service Catalog' },
        { to: '/orchestration', label: 'Orchestration' }
      ]
    },

    // ── Administration ───────────────────────────────────────
    {
      type: 'group',
      label: 'Administration',
      base: '/_admin',
      children: [
        { to: '/projects', label: 'Projects' },
        { to: '/accounts', label: 'Accounts' },
        { to: '/domains', label: 'Domains' },
        { to: '/events', label: 'Events' },
        { to: '/webhooks', label: 'Webhooks' },
        { to: '/usage', label: 'Usage & Billing' },
        { to: '/rate-limits', label: 'Rate Limiting' },
        { to: '/settings/global', label: 'Global Settings' },
        { to: '/utilization', label: 'Utilization' },
        { to: '/docs', label: 'Docs' }
      ]
    }
  ]
}

/** Project-scoped sidebar. */
export function getProjectSections(projectId: string): SidebarSection[] {
  const prefix = `/project/${encodeURIComponent(projectId)}`
  return [
    { type: 'link', to: `${prefix}/dashboard`, label: 'Dashboard' },

    // ── Compute ──────────────────────────────────────────────
    {
      type: 'group',
      label: 'Compute',
      base: `${prefix}/compute`,
      children: [
        { to: `${prefix}/compute/instances`, label: 'Instances' },
        { to: `${prefix}/compute/firecracker`, label: 'Firecracker' },
        { to: `${prefix}/compute/flavors`, label: 'Flavors' },
        { to: `${prefix}/compute/vm-snapshots`, label: 'VM Snapshots' },
        { to: `${prefix}/compute/migrations`, label: 'Migrations' },
        { to: `${prefix}/compute/k8s`, label: 'Kubernetes' },
        { to: `${prefix}/compute/kms`, label: 'SSH Keypairs' },
        { to: '/bare-metal', label: 'Bare Metal' },
        { to: '/autoscale', label: 'Auto Scale' },
        { to: '/affinity-groups', label: 'Affinity Groups' }
      ]
    },

    // ── Storage ──────────────────────────────────────────────
    {
      type: 'group',
      label: 'Storage',
      base: `${prefix}/storage`,
      children: [
        { to: `${prefix}/storage/dashboard`, label: 'Dashboard' },
        { to: `${prefix}/storage/volumes`, label: 'Volumes' },
        { to: `${prefix}/storage/snapshots`, label: 'Snapshots' },
        { to: `${prefix}/storage/backups`, label: 'Backups' },
        { to: `${prefix}/storage/storage-classes`, label: 'Storage Classes' },
        { to: '/snapshot-schedules', label: 'Schedules' },
        { to: '/object-storage', label: 'Object Storage' }
      ]
    },

    // ── Network ──────────────────────────────────────────────
    {
      type: 'group',
      label: 'Network',
      base: `${prefix}/network`,
      children: [
        { to: `${prefix}/network/dashboard`, label: 'Dashboard' },
        { to: `${prefix}/network/vpc`, label: 'Networks' },
        { to: `${prefix}/network/topology`, label: 'Topology' },
        { to: `${prefix}/network/routers`, label: 'Routers' },
        { to: `${prefix}/network/load-balancers`, label: 'Load Balancers' },
        { to: `${prefix}/network/sg`, label: 'Security Groups' },
        { to: `${prefix}/network/firewall`, label: 'Firewalls' },
        { to: `${prefix}/network/public-ips`, label: 'Public IPs' },
        { to: `${prefix}/network/ports`, label: 'Ports' },
        { to: `${prefix}/network/port-forwarding`, label: 'Port Forwarding' },
        { to: `${prefix}/network/qos`, label: 'QoS Policies' },
        { to: `${prefix}/network/vpn`, label: 'VPN' },
        { to: '/dns', label: 'DNS' },
        { to: `${prefix}/network/acl`, label: 'Network ACL' },
        { to: `${prefix}/network/asns`, label: 'ASNs' },
        { to: `${prefix}/network/bgp`, label: 'BGP / Dynamic Routing' }
      ]
    },

    // ── Images ───────────────────────────────────────────────
    {
      type: 'group',
      label: 'Images',
      base: `${prefix}/images`,
      children: [
        { to: `${prefix}/images/templates`, label: 'Templates' },
        { to: `${prefix}/images/iso`, label: 'ISOs' },
        { to: `${prefix}/images/k8s-iso`, label: 'Kubernetes ISO' }
      ]
    },

    // ── Infrastructure ───────────────────────────────────────
    {
      type: 'group',
      label: 'Infrastructure',
      base: `${prefix}/infrastructure`,
      children: [
        { to: `${prefix}/infrastructure/overview`, label: 'Overview' },
        { to: `${prefix}/infrastructure/zones`, label: 'Zones' },
        { to: `${prefix}/infrastructure/clusters`, label: 'Clusters' },
        { to: `${prefix}/infrastructure/hosts`, label: 'Hosts' },
        { to: `${prefix}/infrastructure/primary-storage`, label: 'Primary Storage' },
        { to: `${prefix}/infrastructure/secondary-storage`, label: 'Secondary Storage' },
        { to: `${prefix}/infrastructure/db-usage`, label: 'DB / Usage' },
        { to: `${prefix}/infrastructure/alarms`, label: 'Alarms' },
        { to: '/platform-settings', label: 'Platform Services' },
        { to: '/self-healing', label: 'Self-Healing' },
        { to: '/ha', label: 'High Availability' },
        { to: '/disaster-recovery', label: 'Disaster Recovery' }
      ]
    },

    // ── Security ─────────────────────────────────────────────
    {
      type: 'group',
      label: 'Security',
      base: '/_security',
      children: [
        { to: '/rbac', label: 'Access Control' },
        { to: '/kms', label: 'Key Management' },
        { to: '/encryption', label: 'Data Encryption' },
        { to: '/federation', label: 'Federation' },
        { to: '/compliance-audit', label: 'Compliance' }
      ]
    },

    // ── Service Offerings ────────────────────────────────────
    {
      type: 'group',
      label: 'Service Offerings',
      base: '/_offerings',
      children: [
        { to: '/offerings', label: 'Offerings' },
        { to: '/service-catalog', label: 'Service Catalog' },
        { to: '/orchestration', label: 'Orchestration' }
      ]
    },

    // ── Administration ───────────────────────────────────────
    {
      type: 'group',
      label: 'Administration',
      base: '/_admin',
      children: [
        { to: '/accounts', label: 'Accounts' },
        { to: '/domains', label: 'Domains' },
        { to: '/events', label: 'Events' },
        { to: '/notifications', label: 'Notifications' },
        { to: '/webhooks', label: 'Webhooks' },
        { to: '/usage', label: 'Usage & Billing' },
        { to: '/rate-limits', label: 'Rate Limiting' },
        { to: '/settings/global', label: 'Global Settings' },
        { to: `${prefix}/utilization`, label: 'Utilization' },
        { to: '/docs', label: 'Docs' }
      ]
    }
  ]
}

/**
 * Check whether a group should be auto-expanded for the current path.
 * Works for both real base paths (e.g. /infrastructure) and synthetic
 * ones (e.g. /_security) by also checking child paths.
 */
export function shouldExpandGroup(group: GroupItem, pathname: string): boolean {
  return (
    pathname.startsWith(group.base) ||
    group.children.some(
      (c) => pathname === c.to || pathname.startsWith(c.to + '/')
    )
  )
}
