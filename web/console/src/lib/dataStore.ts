import { create } from 'zustand'

// Types
export type Vpc = { id: string; projectId: string; name: string; cidr: string; state: 'available' | 'creating' }
export type RouteRow = { id: string; projectId: string; destination: string; target: string }
export type LB = { id: string; projectId: string; name: string; type: 'L4' | 'L7'; status: 'active' | 'provisioning' }
export type SGRule = { id: string; projectId: string; direction: 'ingress' | 'egress'; protocol: string; ports: string; cidr: string }
export type ASN = { id: string; projectId: string; number: number; description?: string }

export type Cluster = { id: string; projectId: string; name: string; version: string; status: 'running' | 'provisioning' }
export type Image = { id: string; name: string; sizeGiB: number; status: 'available' | 'uploading' }
export type Flavor = { id: string; name: string; vcpu: number; memoryGiB: number }
export type Snapshot = { id: string; projectId: string; sourceId: string; kind: 'vm' | 'volume'; status: 'ready' | 'creating' }
export type Hypervisor = { id: string; name: string; virtType: 'KVM' | 'QEMU' | 'Xen' | 'VMware'; status: 'online' | 'offline' }
export type SSHKey = { id: string; projectId: string; name: string; publicKey: string }
export type Volume = { id: string; projectId: string; name: string; sizeGiB: number; status: 'available' | 'in-use' }
export type Backup = { id: string; projectId: string; sourceId: string; status: 'ready' | 'creating' }
export type StorageBackend = { id: string; name: string; category: 'Primary' | 'Secondary'; type: 'S3' | 'CephRBD'; endpoint: string; status: 'connected' | 'disconnected' }
export type Instance = { id: string; projectId: string; name: string; ip: string; state: 'running' | 'stopped' }
export type PublicIP = { id: string; projectId: string; address: string; status: 'allocated' | 'free' }
export type Role = { id: string; name: string; description?: string; roleType: 'system' | 'custom' }
export type Account = { id: string; name: string; status: 'active' | 'disabled'; role: string; roleType: 'system' | 'custom'; source: 'SSO' | 'Local' }
export type ISO = { id: string; name: string; sizeGiB: number }
export type Notice = { id: string; time: string; resource: string; type: string; status: 'unread' | 'read' }
export type Project = { id: string; name: string }
export type UtilPoint = { t: number; vcpu: number; memGiB: number; storageGiB: number }
export type UtilSeries = { projectId: string; points: UtilPoint[] }
// CMDB tree
export type CmdbNodeBase = { id: string; name: string; kind: 'group' | 'host' }
export type CmdbHost = CmdbNodeBase & { kind: 'host'; address: string; defaultUser?: string; tags?: string[] }
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
  addBackend: (b: Omit<StorageBackend, 'id' | 'status'> & Partial<Pick<StorageBackend, 'status'>>) => void
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
  removeRole: (id: string) => void
  accounts: Account[]
}

function uid() {
  return Math.random().toString(36).slice(2, 9)
}

const now = () => new Date().toISOString()

export const useDataStore = create<State>((set) => ({
  projects: [
    { id: '1', name: 'admin' },
    { id: '2', name: 'demo' }
  ],
  capacity: { vcpu: 200, memGiB: 1024, storageGiB: 5000 },
  utilization: [
    { projectId: '1', points: Array.from({ length: 24 }).map((_, i) => ({ t: Date.now() - (24 - i) * 3600_000, vcpu: 40 + i % 5, memGiB: 300 + (i % 7) * 5, storageGiB: 1200 + i * 10 })) },
    { projectId: '2', points: Array.from({ length: 24 }).map((_, i) => ({ t: Date.now() - (24 - i) * 3600_000, vcpu: 15 + i % 3, memGiB: 120 + (i % 6) * 3, storageGiB: 600 + i * 7 })) }
  ],
  // Seed data
  vpcs: [
    { id: 'vpc1', projectId: '1', name: 'prod-vpc', cidr: '10.0.0.0/16', state: 'available' },
    { id: 'vpc2', projectId: '2', name: 'dev-vpc', cidr: '10.1.0.0/16', state: 'available' }
  ],
  routes: [
    { id: 'r1', projectId: '1', destination: '0.0.0.0/0', target: 'igw-123' }
  ],
  lbs: [
    { id: 'lb1', projectId: '1', name: 'web-lb', type: 'L7', status: 'active' }
  ],
  sgRules: [
    { id: 'sg1', projectId: '1', direction: 'ingress', protocol: 'tcp', ports: '80,443', cidr: '0.0.0.0/0' }
  ],
  asns: [
    { id: 'asn1', projectId: '1', number: 65001, description: 'Primary ASN' }
  ],
  addVpc: (v) =>
    set((s) => ({ vpcs: [...s.vpcs, { id: uid(), name: v.name, projectId: v.projectId, cidr: v.cidr, state: v.state ?? 'available' }] })),
  removeVpc: (id) => set((s) => ({ vpcs: s.vpcs.filter((x) => x.id !== id) })),
  addAsn: (a) => set((s) => ({ asns: [...s.asns, { id: uid(), ...a }] })),
  removeAsn: (id) => set((s) => ({ asns: s.asns.filter((x) => x.id !== id) })),

  clusters: [
    { id: 'c1', projectId: '1', name: 'prod-k8s', version: '1.29', status: 'running' }
  ],
  images: [],
  flavors: [],
  setFlavors: (f) => set(() => ({ flavors: f })),
  snapshots: [
    { id: 'snp1', projectId: '1', sourceId: 'vm-1', kind: 'vm', status: 'ready' },
    { id: 'snp2', projectId: '1', sourceId: 'vol1', kind: 'volume', status: 'ready' }
  ],
  setSnapshots: (s) => set(() => ({ snapshots: s })),
  hypervisors: [
    { id: 'hv1', name: 'hv-a', virtType: 'KVM', status: 'online' },
    { id: 'hv2', name: 'hv-b', virtType: 'QEMU', status: 'online' }
  ],
  sshKeys: [],
  addCluster: (c) => set((s) => ({ clusters: [...s.clusters, { id: uid(), status: c.status ?? 'provisioning', ...c }] })),
  addImage: (img) => set((s) => ({ images: [...s.images, { id: uid(), ...img, status: 'available' }] })),
  addFlavor: (f) => set((s) => ({ flavors: [...s.flavors, { id: uid(), ...f }] })),
  addSnapshot: (sn) => set((s) => ({ snapshots: [...s.snapshots, { id: uid(), ...sn, status: 'creating' }] })),
  addSSHKey: (k) => set((s) => ({ sshKeys: [...s.sshKeys, { id: uid(), ...k }] })),
  removeSSHKey: (id) => set((s) => ({ sshKeys: s.sshKeys.filter((x) => x.id !== id) })),
  instances: [
    { id: 'vm-1', projectId: '1', name: 'web-app', ip: '10.0.0.21', state: 'running' },
    { id: 'vm-2', projectId: '2', name: 'dev-vm', ip: '10.1.0.30', state: 'stopped' }
  ],
  setInstances: (list) => set(() => ({ instances: list })),
  addInstance: (i) => set((s) => ({ instances: [...s.instances, { id: uid(), ...i }] })),
  publicIPs: [
    { id: 'pip1', projectId: '1', address: '203.0.113.10', status: 'allocated' }
  ],
  addPublicIP: (p) => set((s) => ({ publicIPs: [...s.publicIPs, { id: uid(), ...p }] })),

  volumes: [
    { id: 'vol1', projectId: '1', name: 'data-1', sizeGiB: 100, status: 'available' }
  ],
  addVolume: (v) => set((s) => ({ volumes: [...s.volumes, { id: uid(), status: v.status ?? 'available', ...v }] })),
  backups: [
    { id: 'b1', projectId: '1', sourceId: 'vol1', status: 'ready' }
  ],
  addBackup: (b) => set((s) => ({ backups: [...s.backups, { id: uid(), ...b, status: 'creating' }] })),

  backends: [
    { id: 'bk1', name: 's3-primary', category: 'Primary', type: 'S3', endpoint: 's3.example.com', status: 'connected' },
    { id: 'bk2', name: 'ceph-rbd', category: 'Secondary', type: 'CephRBD', endpoint: 'ceph-mon01:6789', status: 'connected' }
  ],
  addBackend: (b) => set((s) => ({ backends: [...s.backends, { id: uid(), status: b.status ?? 'connected', ...b }] })),
  removeBackend: (id) => set((s) => ({ backends: s.backends.filter((x) => x.id !== id) })),

  notices: [
    { id: 'n1', time: now(), resource: 'prod-vpc', type: 'created', status: 'unread' }
  ],
  markNotice: (id, status) => set((s) => ({ notices: s.notices.map((n) => (n.id === id ? { ...n, status } : n)) }))
  ,
  cmdb: [
    {
      id: 'grp-root',
      name: 'Datacenter',
      kind: 'group',
      children: [
        {
          id: 'grp-prod',
          name: 'Production',
          kind: 'group',
          children: [
            { id: 'h1', name: 'web-01', kind: 'host', address: '10.0.0.11', defaultUser: 'ubuntu', tags: ['vm'] },
            { id: 'h2', name: 'web-02', kind: 'host', address: '10.0.0.12', defaultUser: 'ubuntu', tags: ['vm'] },
            { id: 'h3', name: 'db-01', kind: 'host', address: '10.0.1.10', defaultUser: 'postgres', tags: ['bm'] }
          ]
        },
        {
          id: 'grp-dev',
          name: 'Development',
          kind: 'group',
          children: [
            { id: 'h4', name: 'dev-app', kind: 'host', address: '10.1.0.20', defaultUser: 'dev', tags: ['vm'] }
          ]
        }
      ]
    }
  ],
  isos: [],
  k8sIsos: [],
  roles: [
    { id: 'r1', name: 'Administrator', roleType: 'system' },
    { id: 'r2', name: 'Viewer', roleType: 'system' }
  ],
  addRole: (r) => set((s) => ({ roles: [...s.roles, { id: uid(), ...r }] })),
  removeRole: (id) => set((s) => ({ roles: s.roles.filter((x) => x.id !== id) })),
  accounts: [
    { id: 'u1', name: 'admin', status: 'active', role: 'Administrator', roleType: 'system', source: 'Local' }
  ]
}))
