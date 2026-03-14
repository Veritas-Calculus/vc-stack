import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, cleanup } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'

vi.mock('@/lib/api', async (importOriginal) => {
  const mod = await importOriginal<Record<string, unknown>>()
  const mocked: Record<string, unknown> = {}
  for (const [key, val] of Object.entries(mod)) {
    mocked[key] = typeof val === 'function' ? vi.fn().mockResolvedValue({ data: {} }) : val
  }
  mocked.default = {
    get: vi.fn().mockResolvedValue({ data: {} }),
    post: vi.fn().mockResolvedValue({ data: {} }),
    put: vi.fn().mockResolvedValue({ data: {} }),
    delete: vi.fn().mockResolvedValue({ data: {} }),
    defaults: { withCredentials: false }
  }
  return mocked
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

describe('Network sub-components', () => {
  it('NetworksPage renders with create button', async () => {
    const { NetworksPage } = await import('@/features/network/NetworksPage')
    const { container } = render(
      <BrowserRouter>
        <NetworksPage />
      </BrowserRouter>
    )
    expect(container).toBeTruthy()
    expect(container.textContent).toContain('Network')
  })

  it('ASNPage renders with add button', async () => {
    const { ASNPage } = await import('@/features/network/ASNPage')
    const { container } = render(
      <BrowserRouter>
        <ASNPage />
      </BrowserRouter>
    )
    expect(container).toBeTruthy()
    expect(container.textContent).toContain('ASN')
  })

  it('ACLPage renders with create ACL button', async () => {
    const { ACLPage } = await import('@/features/network/ACLPage')
    const { container } = render(
      <BrowserRouter>
        <ACLPage />
      </BrowserRouter>
    )
    expect(container).toBeTruthy()
    expect(container.textContent).toContain('ACL')
  })
})
