import { create } from 'zustand'

// Types
export type Vpc = {
  id: string
  projectId: string
  name: string
  cidr: string
  state: 'available' | 'creating'
}
export type RouteRow = { id: string; projectId: string; destination: string; target: string }
export type LB = {
  id: string
  projectId: string
  name: string
  type: 'L4' | 'L7'
  status: 'active' | 'provisioning'
}
export type SGRule = {
  id: string
  projectId: string
  direction: 'ingress' | 'egress'
  protocol: string
  ports: string
  cidr: string
}
export type ASN = { id: string; projectId: string; number: number; description?: string }

export type Cluster = {
  id: string
  projectId: string
  name: string
  version: string
  status: 'running' | 'provisioning'
}
export type Image = { id: string; name: string; sizeGiB: number; status: 'available' | 'uploading' }
export type Flavor = { id: string; name: string; vcpu: number; memoryGiB: number }
export type Snapshot = {
  id: string
  projectId: string
  sourceId: string
  kind: 'vm' | 'volume'
  status: 'ready' | 'creating'
}
export type Hypervisor = {
  id: string
  name: string
  virtType: 'KVM' | 'QEMU' | 'Xen' | 'VMware'
  status: 'online' | 'offline'
}
export type SSHKey = { id: string; projectId: string; name: string; publicKey: string }
export type Volume = {
  id: string
  projectId: string
  name: string
  sizeGiB: number
  status: 'available' | 'in-use'
}
export type Backup = {
  id: string
  projectId: string
  sourceId: string
  status: 'ready' | 'creating'
}
export type StorageBackend = {
  id: string
  name: string
  category: 'Primary' | 'Secondary'
  type: 'S3' | 'CephRBD'
  endpoint: string
  status: 'connected' | 'disconnected'
}
export type Instance = {
  id: string
  projectId: string
  name: string
  ip: string
  state: 'running' | 'stopped'
  rootImage?: string
  networks?: Array<{ uuid: string; ip: string }>
}
export type PublicIP = {
  id: string
  projectId: string
  address: string
  status: 'allocated' | 'free'
}
export type Role = {
  id: string
  name: string
  description?: string
  roleType: 'system' | 'custom'
  policyIds?: string[]
}
export type Policy = {
  id: string
  name: string
  description?: string
  type: 'system' | 'custom'
  document: string
}
export type Account = {
  id: string
  name: string
  status: 'active' | 'disabled'
  role: string
  roleType: 'system' | 'custom'
  source: 'SSO' | 'Local'
  policyIds?: string[]
}
export type ISO = { id: string; name: string; sizeGiB: number }
export type Notice = {
  id: string
  time: string
  resource: string
  type: string
  status: 'unread' | 'read'
}
export type Project = { id: string; name: string }
export type UtilPoint = { t: number; vcpu: number; memGiB: number; storageGiB: number }
export type UtilSeries = { projectId: string; points: UtilPoint[] }
// CMDB tree
export type CmdbNodeBase = { id: string; name: string; kind: 'group' | 'host' }
export type CmdbHost = CmdbNodeBase & {
  kind: 'host'
  address: string
  defaultUser?: string
  tags?: string[]
}
export type CmdbGroup = CmdbNodeBase & { kind: 'group'; children: CmdbNode[] }
export type CmdbNode = CmdbGroup | CmdbHost

type State = {
  projects: Project[]
  capacity: { vcpu: number; memGiB: number; storageGiB: number }
  utilization: UtilSeries[]
  // Network
  vpcs: Vpc[]
  routes: RouteRow[]
  lbs: LB[]
  sgRules: SGRule[]
  asns: ASN[]
  addVpc: (v: Omit<Vpc, 'id' | 'state'> & Partial<Pick<Vpc, 'state'>>) => void
  removeVpc: (id: string) => void
  addAsn: (a: Omit<ASN, 'id'>) => void
  removeAsn: (id: string) => void

  // Compute
  clusters: Cluster[]
  images: Image[]
  flavors: Flavor[]
  setFlavors: (f: Flavor[]) => void
  snapshots: Snapshot[]
  setSnapshots: (s: Snapshot[]) => void
  hypervisors: Hypervisor[]
  sshKeys: SSHKey[]
  addImage: (img: Omit<Image, 'id' | 'status'>) => void
  addCluster: (c: Omit<Cluster, 'id' | 'status'> & Partial<Pick<Cluster, 'status'>>) => void
  addFlavor: (f: Omit<Flavor, 'id'>) => void
  addSnapshot: (s: Omit<Snapshot, 'id' | 'status'>) => void
  addSSHKey: (k: Omit<SSHKey, 'id'>) => void
  removeSSHKey: (id: string) => void
  instances: Instance[]
  setInstances: (list: Instance[]) => void
  addInstance: (i: Omit<Instance, 'id'>) => void
  publicIPs: PublicIP[]
  addPublicIP: (p: Omit<PublicIP, 'id'>) => void

  // Storage
  volumes: Volume[]
  addVolume: (v: Omit<Volume, 'id' | 'status'> & Partial<Pick<Volume, 'status'>>) => void
  backups: Backup[]
  addBackup: (b: Omit<Backup, 'id' | 'status'>) => void
  backends: StorageBackend[]
  addBackend: (
    b: Omit<StorageBackend, 'id' | 'status'> & Partial<Pick<StorageBackend, 'status'>>
  ) => void
  removeBackend: (id: string) => void

  // Notifications
  notices: Notice[]
  markNotice: (id: string, status: 'read' | 'unread') => void

  // CMDB
  cmdb: CmdbNode[]
  // Templates / ISO
  isos: ISO[]
  k8sIsos: ISO[]
  // IAM / Accounts
  roles: Role[]
  addRole: (r: Omit<Role, 'id'>) => void
  updateRole: (id: string, r: Partial<Role>) => void
  removeRole: (id: string) => void
  policies: Policy[]
  addPolicy: (p: Omit<Policy, 'id'>) => void
  removePolicy: (id: string) => void
  accounts: Account[]
  updateAccount: (id: string, a: Partial<Account>) => void
}

function uid() {
  return Math.random().toString(36).slice(2, 9)
}

export const useDataStore = create<State>((set) => ({
  projects: [],
  capacity: { vcpu: 0, memGiB: 0, storageGiB: 0 },
  utilization: [],
  vpcs: [],
  routes: [],
  lbs: [],
  sgRules: [],
  asns: [],
  addVpc: (v) =>
    set((s) => ({
      vpcs: [
        ...s.vpcs,
        {
          id: uid(),
          name: v.name,
          projectId: v.projectId,
          cidr: v.cidr,
          state: v.state ?? 'available'
        }
      ]
    })),
  removeVpc: (id) => set((s) => ({ vpcs: s.vpcs.filter((x) => x.id !== id) })),
  addAsn: (a) => set((s) => ({ asns: [...s.asns, { id: uid(), ...a }] })),
  removeAsn: (id) => set((s) => ({ asns: s.asns.filter((x) => x.id !== id) })),

  clusters: [],
  images: [],
  flavors: [],
  setFlavors: (f) => set(() => ({ flavors: f })),
  snapshots: [],
  setSnapshots: (s) => set(() => ({ snapshots: s })),
  hypervisors: [],
  sshKeys: [],
  addCluster: (c) =>
    set((s) => ({
      clusters: [...s.clusters, { id: uid(), status: c.status ?? 'provisioning', ...c }]
    })),
  addImage: (img) =>
    set((s) => ({ images: [...s.images, { id: uid(), ...img, status: 'available' }] })),
  addFlavor: (f) => set((s) => ({ flavors: [...s.flavors, { id: uid(), ...f }] })),
  addSnapshot: (sn) =>
    set((s) => ({ snapshots: [...s.snapshots, { id: uid(), ...sn, status: 'creating' }] })),
  addSSHKey: (k) => set((s) => ({ sshKeys: [...s.sshKeys, { id: uid(), ...k }] })),
  removeSSHKey: (id) => set((s) => ({ sshKeys: s.sshKeys.filter((x) => x.id !== id) })),
  instances: [],
  setInstances: (list) => set(() => ({ instances: list })),
  addInstance: (i) => set((s) => ({ instances: [...s.instances, { id: uid(), ...i }] })),
  publicIPs: [],
  addPublicIP: (p) => set((s) => ({ publicIPs: [...s.publicIPs, { id: uid(), ...p }] })),

  volumes: [],
  addVolume: (v) =>
    set((s) => ({ volumes: [...s.volumes, { id: uid(), status: v.status ?? 'available', ...v }] })),
  backups: [],
  addBackup: (b) =>
    set((s) => ({ backups: [...s.backups, { id: uid(), ...b, status: 'creating' }] })),

  backends: [],
  addBackend: (b) =>
    set((s) => ({
      backends: [...s.backends, { id: uid(), status: b.status ?? 'connected', ...b }]
    })),
  removeBackend: (id) => set((s) => ({ backends: s.backends.filter((x) => x.id !== id) })),

  notices: [],
  markNotice: (id, status) =>
    set((s) => ({ notices: s.notices.map((n) => (n.id === id ? { ...n, status } : n)) })),
  cmdb: [],
  isos: [],
  k8sIsos: [],
  roles: [
    { id: 'r1', name: 'Administrator', roleType: 'system' },
    { id: 'r2', name: 'Viewer', roleType: 'system' }
  ],
  addRole: (r) => set((s) => ({ roles: [...s.roles, { id: uid(), ...r }] })),
  updateRole: (id, r) =>
    set((s) => ({ roles: s.roles.map((x) => (x.id === id ? { ...x, ...r } : x)) })),
  removeRole: (id) => set((s) => ({ roles: s.roles.filter((x) => x.id !== id) })),
  policies: [
    {
      id: 'p1',
      name: 'AdministratorAccess',
      type: 'system',
      document:
        '{"Version": "2012-10-17", "Statement": [{"Effect": "Allow", "Action": "*", "Resource": "*"}]}'
    },
    {
      id: 'p2',
      name: 'ReadOnlyAccess',
      type: 'system',
      document:
        '{"Version": "2012-10-17", "Statement": [{"Effect": "Allow", "Action": ["read", "list"], "Resource": "*"}]}'
    }
  ],
  addPolicy: (p) => set((s) => ({ policies: [...s.policies, { id: uid(), ...p }] })),
  removePolicy: (id) => set((s) => ({ policies: s.policies.filter((x) => x.id !== id) })),
  accounts: [
    {
      id: 'u1',
      name: 'admin',
      status: 'active',
      role: 'Administrator',
      roleType: 'system',
      source: 'Local'
    }
  ],
  updateAccount: (id, a) =>
    set((s) => ({ accounts: s.accounts.map((x) => (x.id === id ? { ...x, ...a } : x)) }))
}))
