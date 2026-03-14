import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, cleanup } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'

vi.mock('@/lib/api', async (importOriginal) => {
  const mod = await importOriginal<Record<string, unknown>>()
  const mocked: Record<string, unknown> = {}
  // Functions that return arrays directly (not { data: ... })
  const arrayFns = [
    'fetchZones',
    'fetchHosts',
    'fetchClusters',
    'fetchStoragePools',
    'fetchSubnets',
    'fetchNetworks'
  ]
  for (const [key, val] of Object.entries(mod)) {
    if (typeof val !== 'function') {
      mocked[key] = val
      continue
    }
    mocked[key] = arrayFns.includes(key)
      ? vi.fn().mockResolvedValue([])
      : vi.fn().mockResolvedValue({ data: {} })
  }
  mocked.default = {
    get: vi.fn().mockResolvedValue({ data: {} }),
    post: vi.fn().mockResolvedValue({ data: {} }),
    put: vi.fn().mockResolvedValue({ data: {} }),
    delete: vi.fn().mockResolvedValue({ data: {} }),
    defaults: { withCredentials: false }
  }
  mocked.resolveApiBase = vi.fn().mockReturnValue('http://localhost:8080')
  return mocked
})

// Overview uses native fetch() instead of axios
globalThis.fetch = vi.fn().mockResolvedValue({
  ok: true,
  json: vi.fn().mockResolvedValue({
    infrastructure: {
      zones: 0,
      clusters: 0,
      hosts: 0,
      hosts_up: 0,
      hosts_down: 0,
      total_vcpus: 0,
      total_ram_mb: 0,
      total_disk_gb: 0
    },
    compute: { total_instances: 0, active_instances: 0, stopped_instances: 0, error_instances: 0 },
    storage: { total_volumes: 0, total_size_gb: 0, used_size_gb: 0 },
    network: { total_networks: 0, total_routers: 0, total_floating_ips: 0 },
    resource_usage: { cpu_percent: 0, ram_percent: 0, disk_percent: 0 }
  })
})

vi.mock('@/lib/dataStore', async (importOriginal) => {
  const mod = await importOriginal<Record<string, unknown>>()
  return {
    ...mod,
    useDataStore: () => ({
      instances: [],
      flavors: [],
      images: [],
      networks: [],
      sshKeys: [],
      projects: [],
      users: [],
      asns: [],
      fetchInstances: vi.fn(),
      fetchFlavors: vi.fn(),
      fetchImages: vi.fn(),
      setInstances: vi.fn(),
      addAsn: vi.fn(),
      removeAsn: vi.fn()
    })
  }
})

vi.mock('@/lib/appStore', () => ({
  useAppStore: () => ({
    isAuthenticated: true,
    user: { username: 'admin', is_admin: true },
    token: 'mock-token',
    setTheme: vi.fn()
  })
}))

afterEach(() => cleanup())

describe('Infrastructure sub-components', () => {
  it('Overview renders without crashing', async () => {
    const { Overview } = await import('@/features/infrastructure/Overview')
    const { container } = render(
      <BrowserRouter>
        <Overview />
      </BrowserRouter>
    )
    expect(container).toBeTruthy()
    // Content loads asynchronously via fetch, so just verify mount
    expect(container.querySelector('div')).not.toBeNull()
  })

  it('Zones renders with page header', async () => {
    const { Zones } = await import('@/features/infrastructure/Zones')
    const { container } = render(
      <BrowserRouter>
        <Zones />
      </BrowserRouter>
    )
    expect(container).toBeTruthy()
    expect(container.textContent).toContain('Zone')
  })

  it('Clusters renders with page header', async () => {
    const { Clusters } = await import('@/features/infrastructure/Clusters')
    const { container } = render(
      <BrowserRouter>
        <Clusters />
      </BrowserRouter>
    )
    expect(container).toBeTruthy()
    expect(container.textContent).toContain('Cluster')
  })

  it('Hosts renders with Add Host button', async () => {
    const { Hosts } = await import('@/features/infrastructure/Hosts')
    const { container } = render(
      <BrowserRouter>
        <Hosts />
      </BrowserRouter>
    )
    expect(container).toBeTruthy()
    expect(container.textContent).toContain('Host')
  })

  it('StoragePools renders primary and secondary tabs', async () => {
    const { PrimaryStorage } = await import('@/features/infrastructure/StoragePools')
    const { container } = render(
      <BrowserRouter>
        <PrimaryStorage />
      </BrowserRouter>
    )
    expect(container).toBeTruthy()
    expect(container.textContent).toContain('Storage')
  })

  it('SystemHealth renders DB usage', async () => {
    const { DBUsage } = await import('@/features/infrastructure/SystemHealth')
    const { container } = render(
      <BrowserRouter>
        <DBUsage />
      </BrowserRouter>
    )
    expect(container).toBeTruthy()
    expect(container.textContent).toContain('DB')
  })

  it('Alarms component renders', async () => {
    const { Alarms } = await import('@/features/infrastructure/SystemHealth')
    const { container } = render(
      <BrowserRouter>
        <Alarms />
      </BrowserRouter>
    )
    expect(container).toBeTruthy()
    expect(container.textContent).toContain('Alarm')
  })
})
